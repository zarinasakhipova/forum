package handlers

import (
	"database/sql"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"01.tomorrow-school.ai/git/zsakhipo/forum/database"
	"unicode/utf8"
)

// Обработчик отправки комментария к посту
func Comments(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Проверка метода запроса: только POST
		if r.Method != http.MethodPost {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		// Проверка сессии пользователя
		cookie, err := r.Cookie("session_id")
		if err != nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		var userID int
		err = db.QueryRow("SELECT user_id FROM sessions WHERE id = ? AND expiry > ? ", cookie.Value, time.Now()).Scan(&userID)
		if err != nil {
			log.Println("Ошибка валидации сессии пользователя для комментария:", err)
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		// Получение ID поста
		postID, err := strconv.Atoi(r.FormValue("post_id"))
		if err != nil {
			log.Println("Неверный ID поста для комментария:", r.FormValue("post_id"), err)
			http.Error(w, "Invalid Post ID", http.StatusBadRequest)
			return
		}

		// Проверка поста
		if !database.PostExists(db, postID) {
			http.Error(w, "Post not found", http.StatusBadRequest)
			return
		}

		// Получение содержимого комментария
		content := r.FormValue("content")
		if strings.TrimSpace(content) == "" {
			// Получаем данные для posts.html
			categoryFilter := r.FormValue("redirect_category")
			// Получаем посты и категории (импортировать нужные структуры)
			// Для простоты: перенаправляем на /posts с параметром ошибки
			http.Redirect(w, r, "/posts?error=empty_comment"+func() string {
				if categoryFilter != "" {
					return "&category=" + categoryFilter
				}
				return ""
			}(), http.StatusSeeOther)
			return
		}
		// Ограничение по количеству слов для комментария
		if utf8.RuneCountInString(content) > 120 {
			categoryFilter := r.FormValue("redirect_category")
			http.Redirect(w, r, "/posts?error=comment_too_long"+func() string {
				if categoryFilter != "" {
					return "&category=" + categoryFilter
				}
				return ""
			}(), http.StatusSeeOther)
			return
		}

		// Вставка комментария в БД
		_, err = db.Exec("INSERT INTO comments (post_id, user_id, content) VALUES (?, ?, ?)", postID, userID, content)
		if err != nil {
			log.Println("Не удалось сохранить комментарий в БД:", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// Перенаправление на страницу постов, сохраняя текущую категорию
		redirectCategory := r.FormValue("redirect_category")
		redirectURL := "/posts"
		if redirectCategory != "" {
			redirectURL += "?category=" + redirectCategory
		}
		http.Redirect(w, r, redirectURL, http.StatusSeeOther)
	}
}

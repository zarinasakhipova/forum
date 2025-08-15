package handlers

import (
	"database/sql"
	"log"
	"net/http"
	"strconv"
	"time"

	"01.tomorrow-school.ai/git/zsakhipo/forum/database"
)

// Like обрабатывает лайки/дизлайки для постов и комментариев
func Like(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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
		if err := db.QueryRow(
			"SELECT user_id FROM sessions WHERE id = ? AND expiry > ?",
			cookie.Value, time.Now(),
		).Scan(&userID); err != nil {
			log.Println("Ошибка валидации сессии пользователя для лайка:", err)
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		isLike := r.FormValue("is_like") == "true"
		var postID, commentID sql.NullInt64

		// Получение ID поста
		if v := r.FormValue("post_id"); v != "" {
			if id, err := strconv.Atoi(v); err == nil {
				postID.Int64, postID.Valid = int64(id), true
			} else {
				log.Println("Неверный ID поста для лайка:", v, err)
				http.Error(w, "Invalid Post ID", http.StatusBadRequest)
				return
			}
		}

		// Получение ID комментария
		if v := r.FormValue("comment_id"); v != "" {
			if id, err := strconv.Atoi(v); err == nil {
				commentID.Int64, commentID.Valid = int64(id), true
			} else {
				log.Println("Неверный ID комментария для лайка:", v, err)
				http.Error(w, "Invalid Comment ID", http.StatusBadRequest)
				return
			}
		}

		// Проверка, что предоставлен либо post_id, либо comment_id, но не оба
		if (postID.Valid && commentID.Valid) || (!postID.Valid && !commentID.Valid) {
			http.Error(w, "Must provide either post_id or comment_id", http.StatusBadRequest)
			return
		}

		// Проверка поста
		if postID.Valid {
			if !database.PostExists(db, int(postID.Int64)) {
				http.Error(w, "Post not found", http.StatusBadRequest)
				return
			}
		}

		// Проверка существования голоса от пользователя для данного поста/комментария
		var existingIsLike bool
		var existingVoteID int
		query := ""
		var args []interface{}

		if postID.Valid {
			query = "SELECT id, is_like FROM likes WHERE user_id = ? AND post_id = ? AND comment_id IS NULL"
			args = []interface{}{userID, postID.Int64}
		} else { // commentID.Valid
			query = "SELECT id, is_like FROM likes WHERE user_id = ? AND comment_id = ? AND post_id IS NULL"
			args = []interface{}{userID, commentID.Int64}
		}

		err = db.QueryRow(query, args...).Scan(&existingVoteID, &existingIsLike)

		if err == nil { // Голос уже существует
			if existingIsLike == isLike { // Если голос совпадает (лайк на лайк, дизлайк на дизлайк)
				// Удаляем голос (отменяем)
				deleteQuery := ""
				deleteArgs := []interface{}{userID}
				if postID.Valid {
					deleteQuery = "DELETE FROM likes WHERE user_id = ? AND post_id = ? AND comment_id IS NULL"
					deleteArgs = append(deleteArgs, postID.Int64)
				} else {
					deleteQuery = "DELETE FROM likes WHERE user_id = ? AND comment_id = ? AND post_id IS NULL"
					deleteArgs = append(deleteArgs, commentID.Int64)
				}
				_, err := db.Exec(deleteQuery, deleteArgs...)
				if err != nil {
					log.Println("Ошибка удаления существующего голоса:", err)
					http.Error(w, "Internal Server Error", http.StatusInternalServerError)
					return
				}
			} else { // Если голос не совпадает (меняем лайк на дизлайк или наоборот)
				updateQuery := ""
				updateArgs := []interface{}{isLike, userID}
				if postID.Valid {
					updateQuery = "UPDATE likes SET is_like = ? WHERE user_id = ? AND post_id = ? AND comment_id IS NULL"
					updateArgs = append(updateArgs, postID.Int64)
				} else {
					updateQuery = "UPDATE likes SET is_like = ? WHERE user_id = ? AND comment_id = ? AND post_id IS NULL"
					updateArgs = append(updateArgs, commentID.Int64)
				}
				_, err := db.Exec(updateQuery, updateArgs...)
				if err != nil {
					log.Println("Ошибка обновления существующего голоса:", err)
					http.Error(w, "Internal Server Error", http.StatusInternalServerError)
					return
				}
			}
		} else if err == sql.ErrNoRows { // Голоса нет, вставляем новый
			insertQuery := ""
			insertArgs := []interface{}{userID}
			if postID.Valid {
				insertQuery = "INSERT INTO likes(user_id, post_id, is_like) VALUES(?, ?, ?)"
				insertArgs = append(insertArgs, postID.Int64, isLike)
			} else {
				insertQuery = "INSERT INTO likes(user_id, comment_id, is_like) VALUES(?, ?, ?)"
				insertArgs = append(insertArgs, commentID.Int64, isLike)
			}
			_, err := db.Exec(insertQuery, insertArgs...)
			if err != nil {
				log.Println("Ошибка вставки нового голоса:", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
		} else { // Другая ошибка БД при проверке голоса
			log.Println("Ошибка запроса существующего голоса:", err)
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

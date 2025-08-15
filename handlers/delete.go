package handlers

import (
	"database/sql"
	"log"
	"net/http"
	"strconv"
	"time"

	"01.tomorrow-school.ai/git/zsakhipo/forum/database"
	"encoding/json"
)

// DeletePost удаляет пост, если текущий пользователь является его автором
func DeletePost(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Проверка метода запроса: только DELETE
		if r.Method != http.MethodDelete {
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
		err = db.QueryRow("SELECT user_id FROM sessions WHERE id = ? AND expiry > ?", cookie.Value, time.Now()).Scan(&userID)
		if err != nil {
			log.Println("Error validating user session for post deletion:", err)
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		// Получение ID поста из формы или JSON
		var postID int
		if r.Method == http.MethodDelete {
			var req struct{ PostID int `json:"post_id"` }
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, "Invalid request", http.StatusBadRequest)
				return
			}
			postID = req.PostID
		} else {
			postID, err = strconv.Atoi(r.FormValue("post_id"))
			if err != nil {
				log.Println("Invalid post ID for deletion:", r.FormValue("post_id"), err)
				http.Error(w, "Invalid Post ID", http.StatusBadRequest)
				return
			}
		}

		// Проверка поста
		if !database.PostExists(db, postID) {
			http.Error(w, "Post not found", http.StatusBadRequest)
			return
		}

		// Проверка прав: является ли пользователь автором поста
		var postAuthorID int
		err = db.QueryRow("SELECT user_id FROM posts WHERE id = ?", postID).Scan(&postAuthorID)
		if err != nil {
			if err == sql.ErrNoRows {
				http.Error(w, "Post not found", http.StatusNotFound)
				return
			}
			log.Println("Error querying post author for deletion:", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// Если пользователь не автор, возвращаем 403 Forbidden
		if postAuthorID != userID {
			http.Error(w, "Forbidden: You are not the author of this post.", http.StatusForbidden)
			return
		}

		// Начало транзакции для атомарного удаления связанных данных
		tx, err := db.Begin()
		if err != nil {
			log.Println("Failed to begin transaction for post deletion:", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		defer tx.Rollback() // Откат транзакции в случае ошибки

		// Удаление связанных лайков/дизлайков
		_, err = tx.Exec("DELETE FROM likes WHERE post_id = ?", postID)
		if err != nil {
			log.Println("Failed to delete likes for post:", postID, err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// Удаление связанных комментариев
		_, err = tx.Exec("DELETE FROM comments WHERE post_id = ?", postID)
		if err != nil {
			log.Println("Failed to delete comments for post:", postID, err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// Удаление связанных категорий
		_, err = tx.Exec("DELETE FROM post_categories WHERE post_id = ?", postID)
		if err != nil {
			log.Println("Failed to delete post categories for post:", postID, err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// Удаление самого поста
		res, err := tx.Exec("DELETE FROM posts WHERE id = ? AND user_id = ?", postID, userID)
		if err != nil {
			log.Println("Failed to delete post from DB:", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		// Проверка успешности удаления поста
		if count, _ := res.RowsAffected(); count == 0 {
			log.Println("Post not found or unauthorized during final delete check (should not happen): postID", postID, "userID", userID)
			http.Error(w, "Post not found or unauthorized", http.StatusForbidden)
			return
		}

		// Фиксация транзакции
		if err := tx.Commit(); err != nil {
			log.Println("Failed to commit transaction for post deletion:", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// Перенаправление на страницу с постами, сохраняя текущую категорию
		redirectCategory := r.FormValue("redirect_category")
		redirectURL := "/posts"
		if redirectCategory != "" {
			redirectURL += "?category=" + redirectCategory
		}
		http.Redirect(w, r, redirectURL, http.StatusSeeOther)
	}
}

// DeleteComment удаляет комментарий, если текущий пользователь является его автором
func DeleteComment(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Проверка метода запроса: только DELETE
		if r.Method != http.MethodDelete {
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
		err = db.QueryRow("SELECT user_id FROM sessions WHERE id = ? AND expiry > ?", cookie.Value, time.Now()).Scan(&userID)
		if err != nil {
			log.Println("Error validating user session for comment deletion:", err)
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		// Получение ID комментария из формы или JSON
		var commentID int
		if r.Method == http.MethodDelete {
			var req struct{ CommentID int `json:"comment_id"` }
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, "Invalid request", http.StatusBadRequest)
				return
			}
			commentID = req.CommentID
		} else {
			commentID, err = strconv.Atoi(r.FormValue("comment_id"))
			if err != nil {
				log.Println("Invalid comment ID for deletion:", r.FormValue("comment_id"), err)
				http.Error(w, "Invalid Comment ID", http.StatusBadRequest)
				return
			}
		}

		// Проверка прав: является ли пользователь автором комментария
		var commentAuthorID int
		err = db.QueryRow("SELECT user_id FROM comments WHERE id = ?", commentID).Scan(&commentAuthorID)
		if err != nil {
			if err == sql.ErrNoRows {
				http.Error(w, "Comment not found", http.StatusNotFound)
				return
			}
			log.Println("Error querying comment author for deletion:", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// Если пользователь не автор, возвращаем 403 Forbidden
		if commentAuthorID != userID {
			http.Error(w, "Forbidden: You are not the author of this comment.", http.StatusForbidden)
			return
		}

		// Начало транзакции для атомарного удаления связанных данных
		tx, err := db.Begin()
		if err != nil {
			log.Println("Failed to begin transaction for comment deletion:", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		defer tx.Rollback() // Откат транзакции в случае ошибки

		// Удаление связанных лайков/дизлайков для комментария
		_, err = tx.Exec("DELETE FROM likes WHERE comment_id = ?", commentID)
		if err != nil {
			log.Println("Failed to delete likes for comment:", commentID, err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// Удаление самого комментария
		res, err := tx.Exec("DELETE FROM comments WHERE id = ? AND user_id = ?", commentID, userID)
		if err != nil {
			log.Println("Failed to delete comment from DB:", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		// Проверка успешности удаления комментария
		if count, _ := res.RowsAffected(); count == 0 {
			log.Println("Comment not found or unauthorized during final delete check (should not happen): commentID", commentID, "userID", userID)
			http.Error(w, "Comment not found or unauthorized", http.StatusForbidden)
			return
		}

		// Фиксация транзакции
		if err := tx.Commit(); err != nil {
			log.Println("Failed to commit transaction for comment deletion:", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// Перенаправление на страницу с постами, сохраняя текущую категорию
		redirectCategory := r.FormValue("redirect_category")
		redirectURL := "/posts"
		if redirectCategory != "" {
			redirectURL += "?category=" + redirectCategory
		}
		http.Redirect(w, r, redirectURL, http.StatusSeeOther)
	}
}

package handlers

import (
	"database/sql"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"01.tomorrow-school.ai/git/zsakhipo/forum/database"
)

// EditPostPageData определяет данные, передаваемые в шаблон edit_post.html
type EditPostPageData struct {
	Categories     []Category // Список всех доступных категорий
	Error          string     // Для отображения ошибок в шаблоне
	Post           Post       // Данные поста для редактирования
	CategoryFilter string     // Для сохранения текущего фильтра категории
}

// Handler for editing a post (for post author only)
func EditPost(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Проверка сессии пользователя
		cookie, err := r.Cookie("session_id")
		if err != nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		var userID int
		var username string
		err = db.QueryRow("SELECT u.id, u.username FROM sessions s JOIN users u ON s.user_id = u.id WHERE s.id = ? AND s.expiry > ?", cookie.Value, time.Now()).Scan(&userID, &username)
		if err != nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		// Получение ID поста из URL
		postIDStr := r.URL.Query().Get("id")
		if postIDStr == "" {
			http.Error(w, "Post ID is required", http.StatusBadRequest)
			return
		}

		postID, err := strconv.Atoi(postIDStr)
		if err != nil {
			http.Error(w, "Invalid post ID", http.StatusBadRequest)
			return
		}

		// Проверка поста
		if !database.PostExists(db, postID) {
			http.Error(w, "Post not found", http.StatusBadRequest)
			return
		}

		// Получение всех категорий для формы выбора
		rows, err := db.Query("SELECT id, name FROM categories")
		if err != nil {
			log.Println("Error fetching categories:", err)
			http.Error(w, "Failed to load categories", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		var allCategories []Category
		for rows.Next() {
			var cat Category
			if err := rows.Scan(&cat.ID, &cat.Name); err != nil {
				log.Println("Error scanning category:", err)
				continue
			}
			allCategories = append(allCategories, cat)
		}

		// Обработка GET-запроса: отображение формы редактирования
		if r.Method == http.MethodGet {
			// Получение данных поста
			var post Post
			var createdAt time.Time
			var imagePath sql.NullString
			err := db.QueryRow(`
				SELECT p.id, p.title, p.content, u.username, p.created_at, p.image_path
				FROM posts p
				JOIN users u ON p.user_id = u.id
				WHERE p.id = ?
			`, postID).Scan(&post.ID, &post.Title, &post.Content, &post.Author, &createdAt, &imagePath)

			if err != nil {
				log.Println("Error fetching post:", err)
				http.Error(w, "Post not found", http.StatusNotFound)
				return
			}

			// Проверка, что пользователь является автором поста
			if post.Author != username {
				http.Error(w, "You can only edit your own posts", http.StatusForbidden)
				return
			}

			if imagePath.Valid {
				post.ImagePath = imagePath.String
			}

			// Получение категорий поста
			categoryRows, err := db.Query(`
				SELECT c.id, c.name
				FROM categories c
				JOIN post_categories pc ON c.id = pc.category_id
				WHERE pc.post_id = ?
			`, postID)
			if err == nil {
				defer categoryRows.Close()
				for categoryRows.Next() {
					var cat Category
					if err := categoryRows.Scan(&cat.ID, &cat.Name); err == nil {
						post.Categories = append(post.Categories, cat)
					}
				}
			}

			currentCategory := r.URL.Query().Get("category")
			data := EditPostPageData{
				Categories:     allCategories,
				Post:           post,
				CategoryFilter: currentCategory,
			}

			tmpl, tmplErr := template.ParseFiles("templates/edit_post.html")
			if tmplErr != nil {
				log.Println("Error parsing edit_post.html template (GET):", tmplErr)
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}
			tmpl.Execute(w, data)
			return
		}

		// Обработка POST-запроса: обновление поста
		title := r.FormValue("title")
		content := r.FormValue("content")
		selectedCategories := r.Form["categories"]
		redirectCategory := r.FormValue("redirect_category")

		// Серверная валидация полей
		if strings.TrimSpace(title) == "" || strings.TrimSpace(content) == "" {
			// Получение данных поста для отображения ошибки
			var post Post
			var imagePath sql.NullString
			db.QueryRow(`
				SELECT p.id, p.title, p.content, u.username, p.image_path
				FROM posts p
				JOIN users u ON p.user_id = u.id
				WHERE p.id = ?
			`, postID).Scan(&post.ID, &post.Title, &post.Content, &post.Author, &imagePath)

			if imagePath.Valid {
				post.ImagePath = imagePath.String
			}

			data := EditPostPageData{
				Categories:     allCategories,
				Error:          "Title and content cannot be empty or only spaces.",
				Post:           post,
				CategoryFilter: redirectCategory,
			}
			w.WriteHeader(http.StatusBadRequest)
			tmpl, tmplErr := template.ParseFiles("templates/edit_post.html")
			if tmplErr != nil {
				log.Println("Error parsing edit_post.html template (POST validation):", tmplErr)
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}
			tmpl.Execute(w, data)
			return
		}
		if utf8.RuneCountInString(title) > 120 {
			var post Post
			var imagePath sql.NullString
			db.QueryRow(`
				SELECT p.id, p.title, p.content, u.username, p.image_path
				FROM posts p
				JOIN users u ON p.user_id = u.id
				WHERE p.id = ?
			`, postID).Scan(&post.ID, &post.Title, &post.Content, &post.Author, &imagePath)
			if imagePath.Valid {
				post.ImagePath = imagePath.String
			}
			data := EditPostPageData{
				Categories:     allCategories,
				Error:          "Title cannot exceed 120 characters (unicode).",
				Post:           post,
				CategoryFilter: redirectCategory,
			}
			w.WriteHeader(http.StatusBadRequest)
			tmpl, tmplErr := template.ParseFiles("templates/edit_post.html")
			if tmplErr != nil {
				log.Println("Error parsing edit_post.html template (title rune limit):", tmplErr)
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}
			tmpl.Execute(w, data)
			return
		}
		if utf8.RuneCountInString(content) > 500 {
			var post Post
			var imagePath sql.NullString
			db.QueryRow(`
				SELECT p.id, p.title, p.content, u.username, p.image_path
				FROM posts p
				JOIN users u ON p.user_id = u.id
				WHERE p.id = ?
			`, postID).Scan(&post.ID, &post.Title, &post.Content, &post.Author, &imagePath)
			if imagePath.Valid {
				post.ImagePath = imagePath.String
			}
			data := EditPostPageData{
				Categories:     allCategories,
				Error:          "Content cannot exceed 500 characters (unicode).",
				Post:           post,
				CategoryFilter: redirectCategory,
			}
			w.WriteHeader(http.StatusBadRequest)
			tmpl, tmplErr := template.ParseFiles("templates/edit_post.html")
			if tmplErr != nil {
				log.Println("Error parsing edit_post.html template (content rune limit):", tmplErr)
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}
			tmpl.Execute(w, data)
			return
		}
		// Проверка на дублирование категорий
		catSet := make(map[string]struct{})
		for _, catIDStr := range selectedCategories {
			if catIDStr == "" {
				continue
			}
			if _, exists := catSet[catIDStr]; exists {
				var post Post
				var imagePath sql.NullString
				db.QueryRow(`
					SELECT p.id, p.title, p.content, u.username, p.image_path
					FROM posts p
					JOIN users u ON p.user_id = u.id
					WHERE p.id = ?
				`, postID).Scan(&post.ID, &post.Title, &post.Content, &post.Author, &imagePath)
				if imagePath.Valid {
					post.ImagePath = imagePath.String
				}
				data := EditPostPageData{
					Categories:     allCategories,
					Error:          "Duplicate categories are not allowed.",
					Post:           post,
					CategoryFilter: redirectCategory,
				}
				w.WriteHeader(http.StatusBadRequest)
				tmpl, tmplErr := template.ParseFiles("templates/edit_post.html")
				if tmplErr != nil {
					log.Println("Error parsing edit_post.html template (duplicate category):", tmplErr)
					http.Error(w, "Internal server error", http.StatusInternalServerError)
					return
				}
				tmpl.Execute(w, data)
				return
			}
			catSet[catIDStr] = struct{}{}
		}

		// Проверка, что пользователь является автором поста
		var author string
		err = db.QueryRow("SELECT u.username FROM posts p JOIN users u ON p.user_id = u.id WHERE p.id = ?", postID).Scan(&author)
		if err != nil || author != username {
			http.Error(w, "You can only edit your own posts", http.StatusForbidden)
			return
		}

		// Обработка загруженного изображения
		var imagePath string
		file, header, err := r.FormFile("image")
		if err == nil && file != nil {
			defer file.Close()

			// Проверяем размер файла (максимум 5MB)
			if header.Size > 5*1024*1024 {
				data := EditPostPageData{
					Categories:     allCategories,
					Error:          "Image file is too large. Maximum size is 5MB.",
					Post:           Post{ID: postID, Title: title, Content: content},
					CategoryFilter: redirectCategory,
				}
				w.WriteHeader(http.StatusBadRequest)
				tmpl, tmplErr := template.ParseFiles("templates/edit_post.html")
				if tmplErr != nil {
					log.Println("Error parsing edit_post.html template (file size validation):", tmplErr)
					http.Error(w, "Internal server error", http.StatusInternalServerError)
					return
				}
				tmpl.Execute(w, data)
				return
			}

			// Проверяем тип файла
			ext := strings.ToLower(filepath.Ext(header.Filename))
			if ext != ".jpg" && ext != ".jpeg" && ext != ".png" && ext != ".gif" {
				data := EditPostPageData{
					Categories:     allCategories,
					Error:          "Invalid file type. Only JPG, PNG, and GIF are allowed.",
					Post:           Post{ID: postID, Title: title, Content: content},
					CategoryFilter: redirectCategory,
				}
				w.WriteHeader(http.StatusBadRequest)
				tmpl, tmplErr := template.ParseFiles("templates/edit_post.html")
				if tmplErr != nil {
					log.Println("Error parsing edit_post.html template (file type validation):", tmplErr)
					http.Error(w, "Internal server error", http.StatusInternalServerError)
					return
				}
				tmpl.Execute(w, data)
				return
			}

			// Создаем уникальное имя файла
			timestamp := time.Now().Unix()
			filename := fmt.Sprintf("%d_%d%s", userID, timestamp, ext)
			filepath := filepath.Join("static", "uploads", filename)

			// Создаем файл
			dst, err := os.Create(filepath)
			if err != nil {
				log.Println("Error creating file:", err)
				http.Error(w, "Failed to save image", http.StatusInternalServerError)
				return
			}
			defer dst.Close()

			// Копируем содержимое файла
			_, err = io.Copy(dst, file)
			if err != nil {
				log.Println("Error copying file:", err)
				http.Error(w, "Failed to save image", http.StatusInternalServerError)
				return
			}

			imagePath = "/static/uploads/" + filename
		}

		// Проверка валидности выбранных категорий
		for _, catIDStr := range selectedCategories {
			if catIDStr == "" {
				continue
			}
			catID, convErr := strconv.Atoi(catIDStr)
			if convErr != nil || !database.IsValidCategory(db, catID) {
				data := EditPostPageData{
					Categories:     allCategories,
					Error:          "Invalid category selected.",
					Post:           Post{ID: postID, Title: title, Content: content},
					CategoryFilter: redirectCategory,
				}
				w.WriteHeader(http.StatusBadRequest)
				tmpl, tmplErr := template.ParseFiles("templates/edit_post.html")
				if tmplErr != nil {
					log.Println("Error parsing edit_post.html template (category validation):", tmplErr)
					http.Error(w, "Internal server error", http.StatusInternalServerError)
					return
				}
				tmpl.Execute(w, data)
				return
			}
		}

		// Начало транзакции
		tx, err := db.Begin()
		if err != nil {
			log.Println("Error beginning transaction:", err)
			http.Error(w, "Failed to update post", http.StatusInternalServerError)
			return
		}
		defer tx.Rollback()

		// Обновление поста
		if imagePath != "" {
			_, err = tx.Exec("UPDATE posts SET title = ?, content = ?, image_path = ? WHERE id = ?", title, content, imagePath, postID)
		} else {
			_, err = tx.Exec("UPDATE posts SET title = ?, content = ? WHERE id = ?", title, content, postID)
		}

		if err != nil {
			tx.Rollback()
			log.Println("Error updating post:", err)
			http.Error(w, "Failed to update post", http.StatusInternalServerError)
			return
		}

		// Удаляем старые связи с категориями
		_, err = tx.Exec("DELETE FROM post_categories WHERE post_id = ?", postID)
		if err != nil {
			tx.Rollback()
			log.Println("Error deleting old post categories:", err)
			http.Error(w, "Failed to update post", http.StatusInternalServerError)
			return
		}

		// Добавляем новые связи с категориями
		if len(selectedCategories) > 0 {
			stmt, err := tx.Prepare("INSERT INTO post_categories (post_id, category_id) VALUES (?, ?)")
			if err != nil {
				tx.Rollback()
				log.Println("Error preparing statement for post categories:", err)
				http.Error(w, "Failed to update post", http.StatusInternalServerError)
				return
			}
			defer stmt.Close()

			for _, catIDStr := range selectedCategories {
				if catIDStr == "" {
					continue
				}

				catID, convErr := strconv.Atoi(catIDStr)
				if convErr != nil {
					tx.Rollback()
					log.Println("Invalid category ID:", convErr)
					http.Error(w, "Invalid category selected", http.StatusBadRequest)
					return
				}
				_, err = stmt.Exec(postID, catID)
				if err != nil {
					tx.Rollback()
					if strings.Contains(err.Error(), "UNIQUE constraint failed: post_categories.post_id, post_categories.category_id") {
						var post Post
						var imagePath sql.NullString
						db.QueryRow(`
							SELECT p.id, p.title, p.content, u.username, p.image_path
							FROM posts p
							JOIN users u ON p.user_id = u.id
							WHERE p.id = ?
						`, postID).Scan(&post.ID, &post.Title, &post.Content, &post.Author, &imagePath)
						if imagePath.Valid {
							post.ImagePath = imagePath.String
						}
						data := EditPostPageData{
							Categories:     allCategories,
							Error:          "Duplicate categories are not allowed.",
							Post:           post,
							CategoryFilter: redirectCategory,
						}
						w.WriteHeader(http.StatusBadRequest)
						tmpl, tmplErr := template.ParseFiles("templates/edit_post.html")
						if tmplErr != nil {
							log.Println("Error parsing edit_post.html template (duplicate category DB):", tmplErr)
							http.Error(w, "Internal server error", http.StatusInternalServerError)
							return
						}
						tmpl.Execute(w, data)
						return
					}
					log.Println("Error inserting post category:", err)
					http.Error(w, "Failed to link categories", http.StatusInternalServerError)
					return
				}
			}
		}

		// Фиксация транзакции
		if err := tx.Commit(); err != nil {
			log.Println("Error committing transaction:", err)
			http.Error(w, "Failed to update post", http.StatusInternalServerError)
			return
		}

		// Перенаправление на страницу постов
		redirectURL := "/posts"
		if redirectCategory != "" {
			redirectURL += "?category=" + redirectCategory
		}
		http.Redirect(w, r, redirectURL, http.StatusSeeOther)
	}
}

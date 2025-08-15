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
)

// CreatePostPageData определяет данные, передаваемые в шаблон create_post.html
type CreatePostPageData struct {
	Categories     []Category // Список всех доступных категорий
	Error          string     // Для отображения ошибок в шаблоне
	Title          string     // Сохраняет введенный заголовок при ошибке
	Content        string     // Сохраняет введенное содержание при ошибке
	CategoryFilter string     // Для сохранения текущего фильтра категории
}

// Handler for creating a new post (for logged-in users only)
func CreatePost(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Проверка сессии пользователя
		cookie, err := r.Cookie("session_id")
		if err != nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		var userID int
		err = db.QueryRow("SELECT user_id FROM sessions WHERE id = ? AND expiry > ?", cookie.Value, time.Now()).Scan(&userID)
		if err != nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		// Получение всех категорий для формы выбора
		rows, err := db.Query("SELECT id, name FROM categories")
		if err != nil {
			log.Println("Error fetching categories:", err)
			tmpl, _ := template.ParseFiles("templates/error.html")
			tmpl.Execute(w, map[string]string{"Message": "Failed to load categories. Please try again later."})
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

		// Обработка GET-запроса: отображение формы
		if r.Method == http.MethodGet {
			currentCategory := r.URL.Query().Get("category") // Получение категории из URL
			data := CreatePostPageData{
				Categories:     allCategories,
				CategoryFilter: currentCategory, // Передача в шаблон
			}
			tmpl, tmplErr := template.ParseFiles("templates/create_post.html")
			if tmplErr != nil {
				log.Println("Error parsing create_post.html template (GET):", tmplErr)
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}
			tmpl.Execute(w, data)
			return
		}

		// Обработка POST-запроса: создание поста
		title := r.FormValue("title")
		content := r.FormValue("content")
		selectedCategories := r.Form["categories"]           // Получение выбранных категорий
		redirectCategory := r.FormValue("redirect_category") // Получение категории для редиректа

		// Автоматический перенос строк для title и content при превышении лимита слов
		splitAndWrap := func(text string, wordLimit int) string {
			words := strings.Fields(text)
			if len(words) <= wordLimit {
				return text
			}
			var sb strings.Builder
			for i, w := range words {
				sb.WriteString(w)
				if (i+1)%wordLimit == 0 {
					sb.WriteString("\n")
				} else {
					sb.WriteString(" ")
				}
			}
			return strings.TrimSpace(sb.String())
		}
		title = splitAndWrap(title, 20)
		content = splitAndWrap(content, 30)

		// Серверная валидация полей
		if strings.TrimSpace(title) == "" || strings.TrimSpace(content) == "" {
			data := CreatePostPageData{
				Categories:     allCategories,
				Error:          "Title and content cannot be empty or only spaces.",
				Title:          title,
				Content:        content,
				CategoryFilter: redirectCategory, // Передача данных обратно в шаблон
			}
			w.WriteHeader(http.StatusBadRequest)
			tmpl, tmplErr := template.ParseFiles("templates/create_post.html")
			if tmplErr != nil {
				log.Println("Error parsing create_post.html template (POST validation):", tmplErr)
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}
			tmpl.Execute(w, data)
			return
		}
		if utf8.RuneCountInString(title) > 120 {
			data := CreatePostPageData{
				Categories:     allCategories,
				Error:          "Title cannot exceed 120 characters (unicode).",
				Title:          title,
				Content:        content,
				CategoryFilter: redirectCategory,
			}
			w.WriteHeader(http.StatusBadRequest)
			tmpl, tmplErr := template.ParseFiles("templates/create_post.html")
			if tmplErr != nil {
				log.Println("Error parsing create_post.html template (title rune limit):", tmplErr)
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}
			tmpl.Execute(w, data)
			return
		}
		if utf8.RuneCountInString(content) > 500 {
			data := CreatePostPageData{
				Categories:     allCategories,
				Error:          "Content cannot exceed 500 characters (unicode).",
				Title:          title,
				Content:        content,
				CategoryFilter: redirectCategory,
			}
			w.WriteHeader(http.StatusBadRequest)
			tmpl, tmplErr := template.ParseFiles("templates/create_post.html")
			if tmplErr != nil {
				log.Println("Error parsing create_post.html template (content rune limit):", tmplErr)
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}
			tmpl.Execute(w, data)
			return
		}

		// Обработка загруженного изображения
		var imagePath string
		file, header, err := r.FormFile("image")
		if err == nil && file != nil {
			defer file.Close()

			// Проверяем размер файла (максимум 5MB)
			if header.Size > 5*1024*1024 {
				data := CreatePostPageData{
					Categories:     allCategories,
					Error:          "Image file is too large. Maximum size is 5MB.",
					Title:          title,
					Content:        content,
					CategoryFilter: redirectCategory,
				}
				w.WriteHeader(http.StatusBadRequest)
				tmpl, tmplErr := template.ParseFiles("templates/create_post.html")
				if tmplErr != nil {
					log.Println("Error parsing create_post.html template (file size validation):", tmplErr)
					http.Error(w, "Internal server error", http.StatusInternalServerError)
					return
				}
				tmpl.Execute(w, data)
				return
			}

			// Проверяем тип файла по расширению
			ext := strings.ToLower(filepath.Ext(header.Filename))
			if ext != ".jpg" && ext != ".jpeg" && ext != ".png" && ext != ".gif" {
				data := CreatePostPageData{
					Categories:     allCategories,
					Error:          "Invalid file type. Only JPG, PNG, and GIF are allowed.",
					Title:          title,
					Content:        content,
					CategoryFilter: redirectCategory,
				}
				w.WriteHeader(http.StatusBadRequest)
				tmpl, tmplErr := template.ParseFiles("templates/create_post.html")
				if tmplErr != nil {
					log.Println("Error parsing create_post.html template (file type validation):", tmplErr)
					http.Error(w, "Internal server error", http.StatusInternalServerError)
					return
				}
				tmpl.Execute(w, data)
				return
			}

			// Проверяем MIME-type
			buf := make([]byte, 512)
			_, err := file.Read(buf)
			if err != nil {
				log.Println("Error reading file for MIME check:", err)
				http.Error(w, "Failed to read image", http.StatusInternalServerError)
				return
			}
			filetype := http.DetectContentType(buf)
			if filetype != "image/jpeg" && filetype != "image/png" && filetype != "image/gif" {
				data := CreatePostPageData{
					Categories:     allCategories,
					Error:          "Invalid image type. Only JPG, PNG, and GIF are allowed.",
					Title:          title,
					Content:        content,
					CategoryFilter: redirectCategory,
				}
				w.WriteHeader(http.StatusBadRequest)
				tmpl, tmplErr := template.ParseFiles("templates/create_post.html")
				if tmplErr != nil {
					log.Println("Error parsing create_post.html template (MIME type validation):", tmplErr)
					http.Error(w, "Internal server error", http.StatusInternalServerError)
					return
				}
				tmpl.Execute(w, data)
				return
			}
			// Сбросить указатель файла для последующего копирования
			file.Seek(0, 0)

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
		catSet := make(map[string]struct{})
		for _, catIDStr := range selectedCategories {
			if catIDStr == "" {
				continue
			}
			if _, exists := catSet[catIDStr]; exists {
				data := CreatePostPageData{
					Categories:     allCategories,
					Error:          "Duplicate categories are not allowed.",
					Title:          title,
					Content:        content,
					CategoryFilter: redirectCategory,
				}
				w.WriteHeader(http.StatusBadRequest)
				tmpl, tmplErr := template.ParseFiles("templates/create_post.html")
				if tmplErr != nil {
					log.Println("Error parsing create_post.html template (duplicate category):", tmplErr)
					http.Error(w, "Internal server error", http.StatusInternalServerError)
					return
				}
				tmpl.Execute(w, data)
				return
			}
			catSet[catIDStr] = struct{}{}
		}

		// Начало транзакции для атомарности операций
		tx, err := db.Begin()
		if err != nil {
			log.Println("Error beginning transaction:", err)
			http.Error(w, "Failed to create post", http.StatusInternalServerError)
			return
		}
		defer tx.Rollback() // Откат транзакции в случае ошибки

		// Вставка нового поста с изображением
		var postResult sql.Result
		if imagePath != "" {
			postResult, err = tx.Exec("INSERT INTO posts (user_id, title, content, image_path) VALUES (?, ?, ?, ?)", userID, title, content, imagePath)
		} else {
			postResult, err = tx.Exec("INSERT INTO posts (user_id, title, content) VALUES (?, ?, ?)", userID, title, content)
		}
		if err != nil {
			tx.Rollback()
			log.Println("Error inserting post:", err)
			http.Error(w, "Failed to create post", http.StatusInternalServerError)
			return
		}

		// Получение ID нового поста
		postID, err := postResult.LastInsertId()
		if err != nil {
			tx.Rollback()
			log.Println("Error getting last insert ID for post:", err)
			http.Error(w, "Failed to create post", http.StatusInternalServerError)
			return
		}

		// Добавляем категории только если они выбраны и не пустые
		if len(selectedCategories) > 0 {
			// Подготовка запроса для вставки связей пост-категория
			stmt, err := tx.Prepare("INSERT INTO post_categories (post_id, category_id) VALUES (?, ?)")
			if err != nil {
				tx.Rollback()
				log.Println("Error preparing statement for post categories:", err)
				http.Error(w, "Failed to create post", http.StatusInternalServerError)
				return
			}
			defer stmt.Close()

			// Вставка связей для каждой выбранной категории
			for _, catIDStr := range selectedCategories {
				// Пропускаем пустые значения (No Category)
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
						data := CreatePostPageData{
							Categories:     allCategories,
							Error:          "Duplicate categories are not allowed.",
							Title:          title,
							Content:        content,
							CategoryFilter: redirectCategory,
						}
						w.WriteHeader(http.StatusBadRequest)
						tmpl, tmplErr := template.ParseFiles("templates/create_post.html")
						if tmplErr != nil {
							log.Println("Error parsing create_post.html template (duplicate category DB):", tmplErr)
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
			http.Error(w, "Failed to create post", http.StatusInternalServerError)
			return
		}

		// Перенаправление на страницу постов, сохраняя текущую категорию
		redirectURL := "/posts"
		if redirectCategory != "" {
			redirectURL += "?category=" + redirectCategory
		}
		http.Redirect(w, r, redirectURL, http.StatusSeeOther)
	}
}

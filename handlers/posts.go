package handlers

import (
	"database/sql"
	"html/template"
	"net/http"
	"strings"
	"time"
)

// Структура для комментария
type Comment struct {
	ID           int
	Author       string
	Content      string
	Likes        int
	Dislikes     int
	UserLiked    bool
	UserDisliked bool
	CreatedAt    string // Дата создания комментария
}

// Структура для категории
type Category struct {
	ID   int
	Name string
}

// Структура для поста
type Post struct {
	ID           int
	Title        string
	Content      string
	Author       string
	CreatedAt    string
	Likes        int
	Dislikes     int
	UserLiked    bool
	UserDisliked bool
	Comments     []Comment
	Categories   []Category
	ImagePath    string // Путь к изображению поста
}

// Структура для данных, передаваемых в шаблон posts.html
type PostsPageData struct {
	IsLoggedIn     bool
	CurrentUser    string
	Posts          []Post
	Categories     []Category
	Filter         string
	CategoryFilter string
	Error          string // Для вывода ошибок (например, пустой комментарий)
}

// nl2br — функция для преобразования переносов строк в HTML <br> для корректного отображения в шаблоне
func nl2br(text string) template.HTML {
	return template.HTML(strings.ReplaceAll(template.HTMLEscapeString(text), "\n", "<br>"))
}

// Функция для автоматического переноса строк по количеству слов
func splitAndWrap(text string, wordLimit int) string {
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

// Posts — обработчик вывода всех постов с фильтрами, категориями и комментариями
func Posts(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Разрешаем только GET-запросы для просмотра постов
		if r.Method != http.MethodGet {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		isLoggedIn := false
		var userID int
		var username string

		// Проверяем сессию пользователя (авторизация)
		cookie, err := r.Cookie("session_id")
		if err == nil {
			err := db.QueryRow(`
				SELECT u.id, u.username
				FROM sessions s
				JOIN users u ON s.user_id = u.id
				WHERE s.id = ? AND s.expiry > ?`, cookie.Value, time.Now(),
			).Scan(&userID, &username)
			if err != nil {
				isLoggedIn = false
			} else {
				isLoggedIn = true
			}
		}

		// Получаем фильтры из URL (поиск по автору, лайкам, категориям)
		filter := r.URL.Query().Get("filter")
		categoryFilter := r.URL.Query().Get("category")

		// Формируем SQL-запрос для выборки постов с учётом фильтров
		var rows *sql.Rows
		var query string
		whereClauses := []string{}
		joinClauses := []string{}
		queryArgs := []interface{}{}

		// Основной SELECT с подсчётом лайков/дизлайков и автором
		selectClause := `
			SELECT p.id, p.title, p.content, u.username, p.created_at,
				(SELECT COUNT(*) FROM likes WHERE post_id = p.id AND is_like = true AND comment_id IS NULL),
				(SELECT COUNT(*) FROM likes WHERE post_id = p.id AND is_like = false AND comment_id IS NULL),
				p.image_path
		`
		fromClause := `
			FROM posts p
			JOIN users u ON p.user_id = u.id
		`

		if filter == "created" {
			if isLoggedIn {
				whereClauses = append(whereClauses, "p.user_id = ?")
				queryArgs = append(queryArgs, userID)
			} else {
				// Если пользователь не авторизован, показываем пустой список
				whereClauses = append(whereClauses, "p.id = -1")
			}
		} else if filter == "liked" {
			if isLoggedIn {
				// Для фильтрации по лайкам нужен JOIN с таблицей likes
				joinClauses = append(joinClauses, "JOIN likes l ON p.id = l.post_id")
				whereClauses = append(whereClauses, "l.user_id = ? AND l.is_like = true")
				queryArgs = append(queryArgs, userID)
			} else {
				// Если пользователь не авторизован, показываем пустой список
				whereClauses = append(whereClauses, "p.id = -1")
			}
		}

		// Добавляем фильтрацию по категории
		if categoryFilter != "" {
			// Для фильтрации по категории нужен JOIN с post_categories и categories
			joinClauses = append(joinClauses, "JOIN post_categories pc ON p.id = pc.post_id")
			joinClauses = append(joinClauses, "JOIN categories cat ON pc.category_id = cat.id")
			whereClauses = append(whereClauses, "cat.name = ?")
			queryArgs = append(queryArgs, categoryFilter)
		}

		// Собираем полный запрос
		query = selectClause + fromClause + strings.Join(joinClauses, " ")
		if len(whereClauses) > 0 {
			query += " WHERE " + strings.Join(whereClauses, " AND ")
		}
		query += " ORDER BY p.created_at DESC"

		var posts []Post
		rows, err = db.Query(query, queryArgs...)
		if err != nil {
			if err == sql.ErrNoRows {
				// Нет постов — это не ошибка, просто показываем пустой список
				posts = []Post{}
			} else {
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
		} else {
			defer rows.Close()
			for rows.Next() {
				var p Post
				var createdAt time.Time
				var imagePath sql.NullString
				if err := rows.Scan(&p.ID, &p.Title, &p.Content, &p.Author, &createdAt, &p.Likes, &p.Dislikes, &imagePath); err != nil {
					continue
				}
				if imagePath.Valid {
					p.ImagePath = imagePath.String
				}
				// Форматируем дату для отображения
				p.CreatedAt = createdAt.Format("Jan 02, 2006 at 15:04")
				p.Comments = []Comment{}
				p.Categories = []Category{}

				// Применяем автоматический перенос строк
				p.Title = splitAndWrap(p.Title, 20)
				p.Content = splitAndWrap(p.Content, 30)

				// Проверяем, лайкнул ли текущий пользователь этот пост
				if isLoggedIn {
					var userVote bool
					err := db.QueryRow(`
						SELECT is_like FROM likes 
						WHERE user_id = ? AND post_id = ? AND comment_id IS NULL
					`, userID, p.ID).Scan(&userVote)
					if err == nil {
						p.UserLiked = userVote
						p.UserDisliked = !userVote
					} else if err != sql.ErrNoRows {
					}
					// Если пользователь не голосовал, оба поля остаются false
				}

				// Получаем комментарии для поста
				commentRows, err := db.Query(`
					SELECT c.id, u.username, c.content,
						(SELECT COUNT(*) FROM likes WHERE comment_id = c.id AND is_like = true AND post_id IS NULL),
						(SELECT COUNT(*) FROM likes WHERE comment_id = c.id AND is_like = false AND post_id IS NULL),
						c.created_at
					FROM comments c
					JOIN users u ON c.user_id = u.id
					WHERE c.post_id = ?
					ORDER BY c.created_at ASC
				`, p.ID)
				if err == nil {
					for commentRows.Next() {
						var c Comment
						var createdAt time.Time
						if err := commentRows.Scan(&c.ID, &c.Author, &c.Content, &c.Likes, &c.Dislikes, &createdAt); err == nil {
							c.CreatedAt = createdAt.Format("Jan 02, 2006 at 15:04")
							// Проверяем, лайкнул ли текущий пользователь этот комментарий
							if isLoggedIn {
								var userVote bool
								err := db.QueryRow(`
									SELECT is_like FROM likes 
									WHERE user_id = ? AND comment_id = ? AND post_id IS NULL
								`, userID, c.ID).Scan(&userVote)
								if err == nil {
									c.UserLiked = userVote
									c.UserDisliked = !userVote
								} else if err != sql.ErrNoRows {
								}
								// Если пользователь не голосовал, оба поля остаются false
							}
							p.Comments = append(p.Comments, c)
						} else {
						}
					}
					commentRows.Close()
				} else {
				}

				// Получаем категории для текущего поста
				categoryRows, err := db.Query(`
					SELECT c.id, c.name
					FROM categories c
					JOIN post_categories pc ON c.id = pc.category_id
					WHERE pc.post_id = ?
					ORDER BY c.name ASC
				`, p.ID)
				if err == nil {
					for categoryRows.Next() {
						var cat Category
						if err := categoryRows.Scan(&cat.ID, &cat.Name); err == nil {
							p.Categories = append(p.Categories, cat)
						} else {
						}
					}
					categoryRows.Close()
				} else {
				}

				posts = append(posts, p)
			}
		}

		// Проверяем ошибки после итерации по rows
		if err = rows.Err(); err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// Получаем все доступные категории для отображения в фильтре
		var allCategories []Category
		rowsCategories, err := db.Query("SELECT id, name FROM categories ORDER BY name ASC")
		if err != nil {
		} else {
			defer rowsCategories.Close()
			for rowsCategories.Next() {
				var cat Category
				if err := rowsCategories.Scan(&cat.ID, &cat.Name); err == nil {
					allCategories = append(allCategories, cat)
				} else {
				}
			}
			if err = rowsCategories.Err(); err != nil {
			}
		}

		data := PostsPageData{
			IsLoggedIn:     isLoggedIn,
			CurrentUser:    username,
			Posts:          posts,
			Categories:     allCategories,
			Filter:         filter,
			CategoryFilter: categoryFilter,
			Error:          "",
		}
		if errMsg := r.URL.Query().Get("error"); errMsg != "" {
			if errMsg == "empty_comment" {
				data.Error = "Comment cannot be empty."
			} else {
				data.Error = "An error occurred."
			}
		}

		tmpl, err := template.New("posts.html").Funcs(template.FuncMap{"nl2br": nl2br}).ParseFiles("templates/posts.html")
		if err != nil {
			http.Error(w, "Internal Server Error: Could not parse template.", http.StatusInternalServerError)
			return
		}

		err = tmpl.Execute(w, data)
		if err != nil {
			http.Error(w, "Internal Server Error: Could not execute template.", http.StatusInternalServerError)
		}
	}
}

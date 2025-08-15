package handlers

import (
	"database/sql"
	"html/template"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"unicode/utf8"
	"unicode"
)

// Обработчик регистрации нового пользователя
func Register(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			// Проверка: если уже авторизован, редирект на /posts
			cookie, err := r.Cookie("session_id")
			if err == nil {
				var userID int
				err = db.QueryRow("SELECT user_id FROM sessions WHERE id = ? AND expiry > ?", cookie.Value, time.Now()).Scan(&userID)
				if err == nil {
					http.Redirect(w, r, "/posts", http.StatusSeeOther)
					return
				}
			}
			tmpl, err := template.ParseFiles("templates/register.html")
			if err != nil {
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			tmpl.Execute(w, nil)
			return
		}

		if r.Method != http.MethodPost {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		username := r.FormValue("username")
		email := r.FormValue("email")
		password := r.FormValue("password")
		if strings.TrimSpace(email) == "" || strings.TrimSpace(username) == "" || strings.TrimSpace(password) == "" {
			tmpl, err := template.ParseFiles("templates/register.html")
			if err != nil {
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusBadRequest)
			tmpl.Execute(w, map[string]string{"Error": "Все поля должны быть заполнены и не могут состоять только из пробелов или переводов строк"})
			return
		}
		if strings.TrimSpace(username) == "" {
			tmpl, err := template.ParseFiles("templates/register.html")
			if err != nil {
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusBadRequest)
			tmpl.Execute(w, map[string]string{"Error": "Username не может быть пустым или состоять только из пробелов"})
			return
		}
		if strings.Contains(username, " ") || strings.Contains(username, "\n") || strings.Contains(username, "\t") {
			tmpl, err := template.ParseFiles("templates/register.html")
			if err != nil {
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusBadRequest)
			tmpl.Execute(w, map[string]string{"Error": "Username не может содержать пробелы или переводы строк"})
			return
		}
		if strings.Contains(email, " ") || strings.Contains(email, "\n") || strings.Contains(email, "\t") {
			tmpl, err := template.ParseFiles("templates/register.html")
			if err != nil {
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusBadRequest)
			tmpl.Execute(w, map[string]string{"Error": "Email не может содержать пробелы или переводы строк"})
			return
		}
		if utf8.RuneCountInString(password) < 8 || utf8.RuneCountInString(password) > 20 {
			tmpl, err := template.ParseFiles("templates/register.html")
			if err != nil {
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusBadRequest)
			tmpl.Execute(w, map[string]string{"Error": "Password must be between 8 and 20 characters (unicode)"})
			return
		}
		if utf8.RuneCountInString(username) < 3 || utf8.RuneCountInString(username) > 20 {
			tmpl, err := template.ParseFiles("templates/register.html")
			if err != nil {
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusBadRequest)
			tmpl.Execute(w, map[string]string{"Error": "Username must be between 3 and 20 characters (unicode)"})
			return
		}
		if utf8.RuneCountInString(email) < 5 || utf8.RuneCountInString(email) > 40 {
			tmpl, err := template.ParseFiles("templates/register.html")
			if err != nil {
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusBadRequest)
			tmpl.Execute(w, map[string]string{"Error": "Email must be between 5 and 40 characters (unicode)"})
			return
		}
		if !strings.Contains(email, "@") || !strings.Contains(email, ".") {
			tmpl, err := template.ParseFiles("templates/register.html")
			if err != nil {
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusBadRequest) // 400 Bad Request
			tmpl.Execute(w, map[string]string{"Error": "Введите корректный email"})
			return
		}

		// Проверка username: только буквы, цифры, подчеркивание, дефис
		for _, r := range username {
			if !(unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '-') {
				tmpl, err := template.ParseFiles("templates/register.html")
				if err != nil {
					http.Error(w, "Internal Server Error", http.StatusInternalServerError)
					return
				}
				w.WriteHeader(http.StatusBadRequest)
				tmpl.Execute(w, map[string]string{"Error": "Username can only contain letters, digits, underscore, and hyphen."})
				return
			}
		}

		if len(strings.Fields(username)) > 15 {
			tmpl, err := template.ParseFiles("templates/register.html")
			if err != nil {
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusBadRequest)
			tmpl.Execute(w, map[string]string{"Error": "Username cannot exceed 15 words."})
			return
		}

		// Проверка уникальности username
		var exists int
		err := db.QueryRow("SELECT COUNT(*) FROM users WHERE username = ?", username).Scan(&exists)
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		if exists > 0 {
			tmpl, tempErr := template.ParseFiles("templates/register.html")
			if tempErr != nil {
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusConflict)
			tmpl.Execute(w, map[string]string{"Error": "Username уже занят"})
			return
		}

		hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		_, err = db.Exec("INSERT INTO users (email, username, password) VALUES (?, ?, ?)", email, username, hashed)
		if err != nil {
			// Проверка на конфликт уникальности (email уже занят)
			if strings.Contains(err.Error(), "UNIQUE constraint failed: users.email") {
				tmpl, tempErr := template.ParseFiles("templates/register.html")
				if tempErr != nil {
					http.Error(w, "Internal Server Error", http.StatusInternalServerError)
					return
				}
				w.WriteHeader(http.StatusConflict) // 409 Conflict
				tmpl.Execute(w, map[string]string{"Error": "Email уже занят"})
				return
			}
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/login", http.StatusSeeOther) // 303 See Other
	}
}

// Обработчик входа пользователя
func Login(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			// Проверка: если уже авторизован, редирект на /posts
			cookie, err := r.Cookie("session_id")
			if err == nil {
				var userID int
				err = db.QueryRow("SELECT user_id FROM sessions WHERE id = ? AND expiry > ?", cookie.Value, time.Now()).Scan(&userID)
				if err == nil {
					http.Redirect(w, r, "/posts", http.StatusSeeOther)
					return
				}
			}
			tmpl, err := template.ParseFiles("templates/login.html")
			if err != nil {
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			tmpl.Execute(w, nil)
			return
		}

		if r.Method != http.MethodPost {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		email := r.FormValue("email")
		password := r.FormValue("password")
		if strings.TrimSpace(email) == "" || strings.TrimSpace(password) == "" {
			tmpl, err := template.ParseFiles("templates/login.html")
			if err != nil {
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusBadRequest)
			tmpl.Execute(w, map[string]string{"Error": "Email и пароль не могут быть пустыми или состоять только из пробелов/переводов строк"})
			return
		}

		if email == "" || password == "" {
			tmpl, err := template.ParseFiles("templates/login.html")
			if err != nil {
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusBadRequest)
			tmpl.Execute(w, map[string]string{"Error": "Email и пароль не могут быть пустыми"})
			return
		}
		if utf8.RuneCountInString(password) > 20 {
			tmpl, err := template.ParseFiles("templates/login.html")
			if err != nil {
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusBadRequest)
			tmpl.Execute(w, map[string]string{"Error": "Password must be between 8 and 20 characters (unicode)"})
			return
		}

		var id int
		var hash string
		err := db.QueryRow("SELECT id, password FROM users WHERE email = ?", email).Scan(&id, &hash)
		if err != nil {
			if err == sql.ErrNoRows {
				tmpl, tempErr := template.ParseFiles("templates/login.html")
				if tempErr != nil {
					http.Error(w, "Internal Server Error", http.StatusInternalServerError)
					return
				}
				w.WriteHeader(http.StatusUnauthorized)
				tmpl.Execute(w, map[string]string{"Error": "Неверный email или пароль"})
				return
			}
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		if bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) != nil {
			tmpl, err := template.ParseFiles("templates/login.html")
			if err != nil {
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusUnauthorized)
			tmpl.Execute(w, map[string]string{"Error": "Неверный email или пароль"})
			return
		}

		sessionID := uuid.New().String()
		expiry := time.Now().Add(24 * time.Hour)
		_, err = db.Exec("INSERT INTO sessions (id, user_id, expiry) VALUES (?, ?, ?)", sessionID, id, expiry)
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		http.SetCookie(w, &http.Cookie{
			Name:    "session_id",
			Value:   sessionID,
			Expires: expiry,
			Path:    "/",
			// Secure: true, // Включить на продакшене для HTTPS
			// HttpOnly: true, // Защита от XSS
			SameSite: http.SameSiteLaxMode, // Защита от CSRF
		})

		http.Redirect(w, r, "/posts", http.StatusSeeOther) // 303 See Other
	}
}

// Logout обрабатывает выход пользователя из системы.
func Logout(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost && r.Method != http.MethodGet {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		cookie, err := r.Cookie("session_id")
		if err == nil { // Кука существует, значит, сессия активна
			_, err = db.Exec("DELETE FROM sessions WHERE id = ?", cookie.Value)
			if err != nil {
				// log.Println("Ошибка удаления сессии из БД (выход):", err)
			}
			// Удаляем куку, устанавливая истекший срок действия
			http.SetCookie(w, &http.Cookie{
				Name:    "session_id",
				Value:   "",
				Expires: time.Now().Add(-time.Hour), // Срок действия в прошлом
				Path:    "/",
			})
		}
		http.Redirect(w, r, "/login", http.StatusSeeOther) // Всегда перенаправляем на страницу входа
	}
}

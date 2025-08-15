package main

import (
	"html/template"
	"log"
	"net/http"
	"strings"

	"01.tomorrow-school.ai/git/zsakhipo/forum/database"
	"01.tomorrow-school.ai/git/zsakhipo/forum/handlers"
)

func nl2br(text string) template.HTML {
	return template.HTML(strings.ReplaceAll(template.HTMLEscapeString(text), "\n", "<br>"))
}

func main() {
	db := database.Init("forum.db") // Инициализация базы данных

	// Обработчики маршрутов
	http.HandleFunc("/register", handlers.Register(db))
	http.HandleFunc("/login", handlers.Login(db))
	http.HandleFunc("/logout", handlers.Logout(db))
	http.HandleFunc("/posts", handlers.Posts(db))
	http.HandleFunc("/post/create", handlers.CreatePost(db))
	http.HandleFunc("/comment", handlers.Comments(db))
	http.HandleFunc("/like", handlers.Like(db))
	http.HandleFunc("/post/delete", handlers.DeletePost(db))
	http.HandleFunc("/edit-post", handlers.EditPost(db))
	http.HandleFunc("/comment/delete", handlers.DeleteComment(db))

	// Отдача статических файлов (CSS, JS и т.д.)
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	// Отдача favicon.ico из папки static
	http.Handle("/favicon.ico", http.FileServer(http.Dir("static")))

	// Перенаправление с корневого URL на /posts
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			http.Redirect(w, r, "/posts", http.StatusSeeOther)
			return
		}
		w.WriteHeader(http.StatusNotFound)
		tmpl, _ := template.ParseFiles("templates/error.html")
		tmpl.Execute(w, map[string]string{"Message": "404 - Page not found"})
	})

	log.Println("Server starting on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

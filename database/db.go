package database

import (
	"database/sql"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

// Init открывает базу данных и создаёт необходимые таблицы.
func Init(path string) *sql.DB {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		log.Fatalf("Ошибка открытия базы данных: %v", err)
	}

	// Определение схемы всех таблиц
	schema := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			email TEXT UNIQUE NOT NULL,
			username TEXT NOT NULL,
			password TEXT NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS sessions (
			id TEXT PRIMARY KEY,
			user_id INTEGER NOT NULL,
			expiry DATETIME NOT NULL,
			FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
		);`,
		`CREATE TABLE IF NOT EXISTS posts (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			title TEXT NOT NULL,
			content TEXT NOT NULL,
			image_path TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
		);`,
		`CREATE TABLE IF NOT EXISTS comments (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			post_id INTEGER NOT NULL,
			content TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE,
			FOREIGN KEY(post_id) REFERENCES posts(id) ON DELETE CASCADE
		);`,
		`CREATE TABLE IF NOT EXISTS likes (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			post_id INTEGER,     -- Может быть NULL, если это лайк комментария
			comment_id INTEGER,  -- Может быть NULL, если это лайк поста
			is_like BOOLEAN NOT NULL,
			UNIQUE(user_id, post_id, comment_id), -- Пользователь может лайкнуть один пост/коммент только один раз
			FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE,
			FOREIGN KEY(post_id) REFERENCES posts(id) ON DELETE CASCADE,
			FOREIGN KEY(comment_id) REFERENCES comments(id) ON DELETE CASCADE
		);`,
		`CREATE TABLE IF NOT EXISTS categories (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT UNIQUE NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS post_categories (
			post_id INTEGER NOT NULL,
			category_id INTEGER NOT NULL,
			PRIMARY KEY (post_id, category_id),
			FOREIGN KEY(post_id) REFERENCES posts(id) ON DELETE CASCADE,
			FOREIGN KEY(category_id) REFERENCES categories(id) ON DELETE CASCADE
		);`,
	}

	// Выполняем запросы для создания всех таблиц
	for _, q := range schema {
		_, err := db.Exec(q)
		if err != nil {
			log.Fatalf("Ошибка создания схемы базы данных: %v, запрос: %s", err, q)
		}
	}

	// Вставляем предопределенные категории, если их еще нет.
	insertCategories(db)

	return db
}

// insertCategories вставляет предопределенные категории в базу данных.
func insertCategories(db *sql.DB) {
	categories := []string{
		"General",
		"Announcements",
		"Discussions",
		"Questions",
		"Suggestions",
		"Off-topic",
	}

	for _, categoryName := range categories {
		// Проверяем, существует ли категория, прежде чем вставлять её
		var count int
		err := db.QueryRow("SELECT COUNT(*) FROM categories WHERE name = ?", categoryName).Scan(&count)
		if err != nil {
			log.Printf("Ошибка при проверке существования категории '%s': %v", categoryName, err)
			continue
		}
		if count == 0 {
			// Если категория не существует, вставляем её
			_, err := db.Exec("INSERT INTO categories (name) VALUES (?)", categoryName)
			if err != nil {
				log.Printf("Ошибка при вставке категории '%s': %v", categoryName, err)
			}
		}
	}
}

// Проверяет, существует ли пост с таким ID
func PostExists(db *sql.DB, postID int) bool {
	var exists bool
	err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM posts WHERE id = ?)", postID).Scan(&exists)
	return err == nil && exists
}

// Проверяет, существует ли категория с таким ID
func IsValidCategory(db *sql.DB, categoryID int) bool {
	var exists bool
	err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM categories WHERE id = ?)", categoryID).Scan(&exists)
	return err == nil && exists
}

# 🗂️ Форум

## 📌 Описание проекта

Этот проект — минималистичный веб-форум на Go. Все данные хранятся в базе SQLite. Пользователи могут регистрироваться, входить, создавать посты и комментарии, ставить лайки/дизлайки, фильтровать контент. Приложение контейнеризовано с помощью Docker.

---

## 🎯 Функционал

| Компонент             | Возможности |
|----------------------|-------------|
| **Общение**          | Зарегистрированные пользователи могут создавать посты и комментарии |
| **Категории**        | Каждый пост может иметь одну или несколько категорий |
| **Лайки / Дизлайки** | Голосование за посты и комментарии (+1 / -1), видно всем |
| **Фильтрация**       | Фильтрация постов по категориям, моим постам и понравившимся |
| **Аутентификация**   | Регистрация и вход с помощью cookie и UUID, хеширование пароля через bcrypt |
| **База данных**      | SQLite с запросами CREATE, INSERT, SELECT |
| **Docker**           | Контейнеризация, простой запуск через Docker |
| **Обработка ошибок** | Корректная обработка HTTP-статусов и ошибок |
| **Тестирование**     | Поддержка unit-тестов (рекомендуется) |

---

## 📡 HTTP-статусы

| Статус | Где в коде                | Когда срабатывает                                      |
|--------|---------------------------|--------------------------------------------------------|
| `200 OK` | По умолчанию Go         | Отдача HTML-страниц (`/posts`, формы, статика)         |
| `303 See Other` | `http.Redirect(..., 303)` | POST/PUT → GET (логин, регистрация, CRUD-формы) |
| `400 Bad Request` | `http.Error(..., 400)` | Некорректные данные формы (пустые поля, плохой ID)    |
| `401 Unauthorized` | `http.Error(..., 401)` | Неверный email/пароль при входе                       |
| `403 Forbidden` | `http.Error(..., 403)` | Попытка удалить чужой пост/комментарий                |
| `404 Not Found` | корень `/` если путь ≠ `/posts` | Запрошен несуществующий URL                  |
| `409 Conflict` | регистрация           | Email/username уже занят                              |
| `500 Internal Server Error` | `http.Error(..., 500)` | Ошибки БД, парсинг шаблонов, неожиданные ошибки |

---

## 🏗️ Структура проекта

```bash
forum/
├── cmd/                    # main.go
├── database/               # Логика работы с SQLite
├── handlers/               # HTTP-обработчики
├── static/                 # стили, картинки и т.д.
├── templates/              # HTML-шаблоны
├── Dockerfile
├── go.mod
├── go.sum
├── README.md
├── README_RUS.md
└── forum.db
```

## 🗄️ Пример структуры базы данных

```sql
CREATE TABLE users (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  email TEXT UNIQUE NOT NULL,
  username TEXT NOT NULL,
  password TEXT NOT NULL
);

CREATE TABLE sessions (
  id TEXT PRIMARY KEY,
  user_id INTEGER NOT NULL,
  expiry DATETIME NOT NULL,
  FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE TABLE categories (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT UNIQUE NOT NULL
);

CREATE TABLE posts (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  user_id INTEGER NOT NULL,
  title TEXT NOT NULL,
  content TEXT NOT NULL,
  image_path TEXT,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE TABLE post_categories (
  post_id INTEGER NOT NULL,
  category_id INTEGER NOT NULL,
  PRIMARY KEY (post_id, category_id),
  FOREIGN KEY(post_id) REFERENCES posts(id) ON DELETE CASCADE,
  FOREIGN KEY(category_id) REFERENCES categories(id) ON DELETE CASCADE
);

CREATE TABLE comments (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  user_id INTEGER NOT NULL,
  post_id INTEGER NOT NULL,
  content TEXT NOT NULL,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE,
  FOREIGN KEY(post_id) REFERENCES posts(id) ON DELETE CASCADE
);

CREATE TABLE likes (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  user_id INTEGER NOT NULL,
  post_id INTEGER,
  comment_id INTEGER,
  is_like BOOLEAN NOT NULL,
  UNIQUE(user_id, post_id, comment_id),
  FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE,
  FOREIGN KEY(post_id) REFERENCES posts(id) ON DELETE CASCADE,
  FOREIGN KEY(comment_id) REFERENCES comments(id) ON DELETE CASCADE
);
```

---

## Аутентификация и использование

- **Регистрация:**
  - Требуются email, username, пароль.
  - Email и username должны быть уникальны.
  - Пароль хешируется через bcrypt.
- **Вход:**
  - Требуются email и пароль.
  - При успехе создаётся сессия (UUID), устанавливается cookie с истечением.
  - Только одна активная сессия на пользователя.
- **Посты и комментарии:**
  - Только зарегистрированные пользователи могут создавать.
  - Для постов можно выбрать категории.
  - Все пользователи (включая гостей) могут просматривать посты и комментарии.
- **Лайки/дизлайки:**
  - Только авторизованные пользователи могут голосовать.
  - Один голос на объект от пользователя (можно менять).
  - Все видят количество лайков/дизлайков.
- **Фильтрация:**
  - По категориям (все)
  - По моим постам (только авторизованные)
  - По понравившимся постам (только авторизованные)

---

## Запуск приложения
```bash
go run ./cmd/main.go
```

Откройте в браузере:
```
http://localhost:8080
```

---

## 🚀 Быстрый старт

```bash
chmod +x build.sh
./build.sh
```
- Собирает Docker-образ, удаляет старый контейнер, запускает новый.
- Форум будет доступен по адресу: http://localhost:8080

### Ручной запуск через Docker
```bash
docker build -t forum .
docker run -d -p 8080:8080 --name forum forum
```

### Остановка и удаление контейнера
```bash
docker stop forum
docker rm forum
```

### Проверка контейнеров и образов
```bash
docker ps -a
docker images
```

---

## ⚙️ Использование .dockerignore
- `.dockerignore` исключает временные, IDE и git-файлы из Docker-образа.
- Чтобы хранить базу вне контейнера, раскомментируйте `forum.db` в `.dockerignore` и используйте volume.

---

### Пример docker-compose.yaml
```yaml
version: "3.9"
services:
  forum:
    build: .
    ports:
      - "8080:8080"
    volumes:
      - .:/app
    environment:
      DB_PATH: /app/data/forum.db
```

---

## 🛠️ Технологии

- Go 1.22+
- SQLite (`github.com/mattn/go-sqlite3`)
- bcrypt (`golang.org/x/crypto/bcrypt`)
- UUID (`github.com/google/uuid`)
- HTML + CSS (без фреймворков)
- Docker

---

Автор:
- Zarina Sakhipova @zsakhipo
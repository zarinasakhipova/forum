# 🗂️ Forum

## 📌 Project Description

This project is a minimalist web forum written in Go. All data is stored in a SQLite database. Users can register, log in, create posts and comments, like/dislike, and filter content. The application is containerized with Docker.

---

## 🎯 Features

| Component             | Functionality |
|----------------------|---------------|
| **Communication**    | Registered users can create posts and comments |
| **Categories**       | Each post can have one or more categories |
| **Likes / Dislikes** | Voting for posts and comments (+1 / -1), visible to all |
| **Filtering**        | Filter posts by categories, my posts, and liked posts |
| **Authentication**   | Registration and login using cookies and UUID, password hashing with bcrypt |
| **Database**         | SQLite with CREATE, INSERT, SELECT queries |
| **Docker**           | Containerized app, easy launch via Docker |
| **Error Handling**   | Proper HTTP status and error handling |
| **Testing**          | Unit test support (recommended) |

---

## 📡 HTTP Status Codes

| Status | Where in code           | When triggered                                         |
|--------|-------------------------|--------------------------------------------------------|
| `200 OK` | Go default           | Serving HTML pages (`/posts`, forms, static)           |
| `303 See Other` | `http.Redirect(..., 303)` | POST/PUT → GET (login, registration, CRUD forms) |
| `400 Bad Request` | `http.Error(..., 400)` | Invalid form data (empty fields, bad ID)          |
| `401 Unauthorized` | `http.Error(..., 401)` | Wrong email/password on login                      |
| `403 Forbidden` | `http.Error(..., 403)` | Attempt to delete someone else's post/comment      |
| `404 Not Found` | root `/` if path ≠ `/posts` | Non-existent URL requested                    |
| `409 Conflict` | registration | Email/username already taken                        |
| `500 Internal Server Error` | `http.Error(..., 500)` | DB errors, template parsing, unexpected errors |

---

## 🏗️ Project Structure

```bash
forum/
├── cmd/                    # main.go
├── database/               # SQLite logic
├── handlers/               # HTTP request handlers
├── static/                 # styles, images, etc.
├── templates/              # HTML templates
├── Dockerfile
├── go.mod
├── go.sum
├── README.md
├── README_RUS.md
└── forum.db
```

---

## 🗄️ Example Database Structure

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

## Authentication & Usage

- **Registration:**
  - Email, username, password required.
  - Email and username must be unique.
  - Password is hashed with bcrypt.
- **Login:**
  - Email and password required.
  - On success, creates a session (UUID), sets cookie with expiration.
  - Only one active session per user.
- **Posts & Comments:**
  - Only registered users can create.
  - Categories can be selected for posts.
  - All users (including guests) can view posts and comments.
- **Likes/Dislikes:**
  - Only authorized users can vote.
  - One vote per object per user (can be changed).
  - All users see the number of likes/dislikes.
- **Filtering:**
  - By categories (all users)
  - By my posts (authorized only)
  - By liked posts (authorized only)

---

## Run the application
```bash
go run ./cmd/main.go
```

Open in browser:
```
http://localhost:8080
```

---

## 🚀 Quick Start

```bash
chmod +x build.sh
./build.sh
```
- Builds Docker image, removes old container, runs a new one.
- Forum available at: http://localhost:8080

### Manual Docker Run
```bash
docker build -t forum .
docker run -d -p 8080:8080 --name forum forum
```

### Stop and Remove Container
```bash
docker stop forum
docker rm forum
```

### Check Containers and Images
```bash
docker ps -a
docker images
```

---

## ⚙️ Using .dockerignore
- `.dockerignore` excludes temp, IDE, and git files from the Docker image.
- To keep the database outside the container, uncomment `forum.db` in `.dockerignore` and use a volume.

---

### Example docker-compose.yaml
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

## 🛠️ Technologies

- Go 1.22+
- SQLite (`github.com/mattn/go-sqlite3`)
- bcrypt (`golang.org/x/crypto/bcrypt`)
- UUID (`github.com/google/uuid`)
- HTML + CSS (no frameworks)
- Docker

---

Author:
- Zarina Sakhipova @zsakhipo
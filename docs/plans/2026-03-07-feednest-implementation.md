# FeedNest Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build a modern, self-hosted RSS feed reader with SvelteKit frontend + Go backend, featuring card/list views, clean article reader, smart prioritization, and multi-user auth.

**Architecture:** SvelteKit (frontend) communicates via REST API with a Go backend that handles feed fetching, RSS parsing, content extraction, and article scoring. SQLite for storage. Docker Compose for deployment.

**Tech Stack:** Go 1.22+, SvelteKit 2, TypeScript, Tailwind CSS 4, SQLite (mattn/go-sqlite3), chi router, gofeed (RSS parsing), go-readability, golang-jwt, bcrypt

---

## Phase 1: Go Backend Foundation

### Task 1: Initialize Go Module and Project Structure

**Files:**
- Create: `backend/cmd/feednest/main.go`
- Create: `backend/go.mod`

**Step 1: Create directory structure**

```bash
mkdir -p backend/cmd/feednest backend/internal/{api/handlers,models,store,fetcher,parser,readability,scorer,scheduler}
```

**Step 2: Initialize Go module**

```bash
cd backend && go mod init github.com/feednest/backend
```

**Step 3: Write minimal main.go**

```go
package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"status":"ok"}`)
	})

	log.Printf("FeedNest backend starting on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}
```

**Step 4: Verify it compiles and runs**

```bash
cd backend && go run ./cmd/feednest/
# In another terminal: curl http://localhost:8080/api/health
# Expected: {"status":"ok"}
```

**Step 5: Commit**

```bash
git add backend/
git commit -m "feat: initialize Go backend with health endpoint"
```

---

### Task 2: Install Dependencies and Set Up Chi Router

**Files:**
- Modify: `backend/cmd/feednest/main.go`
- Create: `backend/internal/api/router.go`
- Create: `backend/internal/api/middleware.go`

**Step 1: Install dependencies**

```bash
cd backend
go get github.com/go-chi/chi/v5
go get github.com/go-chi/chi/v5/middleware
go get github.com/go-chi/cors
```

**Step 2: Create router.go**

```go
package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

func NewRouter() http.Handler {
	r := chi.NewRouter()

	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)
	r.Use(chimw.RealIP)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:5173", "http://localhost:3000"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	r.Get("/api/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})

	return r
}
```

**Step 3: Create middleware.go (placeholder for auth middleware)**

```go
package api

import "net/http"

// AuthMiddleware validates JWT tokens. Implemented in Task 6.
func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// TODO: JWT validation
		next.ServeHTTP(w, r)
	})
}
```

**Step 4: Update main.go to use the router**

```go
package main

import (
	"log"
	"os"

	"net/http"

	"github.com/feednest/backend/internal/api"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	router := api.NewRouter()

	log.Printf("FeedNest backend starting on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, router))
}
```

**Step 5: Verify it runs**

```bash
cd backend && go run ./cmd/feednest/
# curl http://localhost:8080/api/health -> {"status":"ok"}
```

**Step 6: Commit**

```bash
git add backend/
git commit -m "feat: add chi router with CORS and logging middleware"
```

---

### Task 3: SQLite Database Setup and Migrations

**Files:**
- Create: `backend/internal/store/db.go`
- Create: `backend/internal/store/migrations.go`
- Create: `backend/internal/store/db_test.go`

**Step 1: Install SQLite driver**

```bash
cd backend
go get github.com/mattn/go-sqlite3
```

**Step 2: Write the failing test**

```go
package store

import (
	"os"
	"testing"
)

func TestNewDB_CreatesTablesOnInit(t *testing.T) {
	dbPath := t.TempDir() + "/test.db"
	db, err := NewDB(dbPath)
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	defer db.Close()
	defer os.Remove(dbPath)

	// Verify all expected tables exist
	tables := []string{"users", "feeds", "categories", "articles", "tags", "article_tags", "reading_events", "settings"}
	for _, table := range tables {
		var name string
		err := db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name=?", table).Scan(&name)
		if err != nil {
			t.Errorf("table %q not found: %v", table, err)
		}
	}
}
```

**Step 3: Run test to verify it fails**

```bash
cd backend && go test ./internal/store/ -v -run TestNewDB
# Expected: FAIL - NewDB not defined
```

**Step 4: Write db.go**

```go
package store

import (
	"database/sql"

	_ "github.com/mattn/go-sqlite3"
)

func NewDB(dbPath string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_foreign_keys=on")
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	if err := runMigrations(db); err != nil {
		return nil, err
	}

	return db, nil
}
```

**Step 5: Write migrations.go**

```go
package store

import "database/sql"

func runMigrations(db *sql.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		username TEXT UNIQUE NOT NULL,
		email TEXT UNIQUE NOT NULL,
		password_hash TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS categories (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		name TEXT NOT NULL,
		position INTEGER DEFAULT 0,
		UNIQUE(user_id, name)
	);

	CREATE TABLE IF NOT EXISTS feeds (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		url TEXT NOT NULL,
		title TEXT,
		site_url TEXT,
		icon_url TEXT,
		category_id INTEGER REFERENCES categories(id) ON DELETE SET NULL,
		fetch_interval INTEGER DEFAULT 900,
		last_fetched DATETIME,
		engagement_score REAL DEFAULT 0.0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(user_id, url)
	);

	CREATE TABLE IF NOT EXISTS articles (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		feed_id INTEGER NOT NULL REFERENCES feeds(id) ON DELETE CASCADE,
		guid TEXT NOT NULL,
		title TEXT,
		url TEXT,
		author TEXT,
		content_raw TEXT,
		content_clean TEXT,
		thumbnail_url TEXT,
		published_at DATETIME,
		fetched_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		word_count INTEGER DEFAULT 0,
		reading_time INTEGER DEFAULT 0,
		is_read BOOLEAN DEFAULT 0,
		is_starred BOOLEAN DEFAULT 0,
		read_at DATETIME,
		score REAL DEFAULT 0.0,
		UNIQUE(feed_id, guid)
	);

	CREATE TABLE IF NOT EXISTS tags (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		name TEXT NOT NULL,
		UNIQUE(user_id, name)
	);

	CREATE TABLE IF NOT EXISTS article_tags (
		article_id INTEGER NOT NULL REFERENCES articles(id) ON DELETE CASCADE,
		tag_id INTEGER NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
		PRIMARY KEY (article_id, tag_id)
	);

	CREATE TABLE IF NOT EXISTS reading_events (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		article_id INTEGER NOT NULL REFERENCES articles(id) ON DELETE CASCADE,
		event_type TEXT NOT NULL,
		duration_seconds INTEGER DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS settings (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		key TEXT NOT NULL,
		value TEXT,
		UNIQUE(user_id, key)
	);

	CREATE INDEX IF NOT EXISTS idx_articles_feed_id ON articles(feed_id);
	CREATE INDEX IF NOT EXISTS idx_articles_published_at ON articles(published_at);
	CREATE INDEX IF NOT EXISTS idx_articles_score ON articles(score);
	CREATE INDEX IF NOT EXISTS idx_articles_is_read ON articles(is_read);
	CREATE INDEX IF NOT EXISTS idx_feeds_user_id ON feeds(user_id);
	CREATE INDEX IF NOT EXISTS idx_reading_events_article_id ON reading_events(article_id);
	`

	_, err := db.Exec(schema)
	return err
}
```

**Step 6: Run test to verify it passes**

```bash
cd backend && go test ./internal/store/ -v -run TestNewDB
# Expected: PASS
```

**Step 7: Commit**

```bash
git add backend/
git commit -m "feat: add SQLite database with schema migrations"
```

---

### Task 4: Models Package

**Files:**
- Create: `backend/internal/models/user.go`
- Create: `backend/internal/models/feed.go`
- Create: `backend/internal/models/category.go`
- Create: `backend/internal/models/article.go`
- Create: `backend/internal/models/tag.go`
- Create: `backend/internal/models/event.go`

**Step 1: Create all model files**

`backend/internal/models/user.go`:
```go
package models

import "time"

type User struct {
	ID           int64     `json:"id"`
	Username     string    `json:"username"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type RegisterRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type AuthResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	User         User   `json:"user"`
}
```

`backend/internal/models/category.go`:
```go
package models

type Category struct {
	ID       int64  `json:"id"`
	UserID   int64  `json:"user_id"`
	Name     string `json:"name"`
	Position int    `json:"position"`
}

type CreateCategoryRequest struct {
	Name     string `json:"name"`
	Position int    `json:"position"`
}

type UpdateCategoryRequest struct {
	Name     *string `json:"name,omitempty"`
	Position *int    `json:"position,omitempty"`
}
```

`backend/internal/models/feed.go`:
```go
package models

import "time"

type Feed struct {
	ID              int64      `json:"id"`
	UserID          int64      `json:"user_id"`
	URL             string     `json:"url"`
	Title           string     `json:"title"`
	SiteURL         string     `json:"site_url"`
	IconURL         string     `json:"icon_url"`
	CategoryID      *int64     `json:"category_id"`
	FetchInterval   int        `json:"fetch_interval"`
	LastFetched     *time.Time `json:"last_fetched"`
	EngagementScore float64    `json:"engagement_score"`
	CreatedAt       time.Time  `json:"created_at"`
	UnreadCount     int        `json:"unread_count,omitempty"`
}

type CreateFeedRequest struct {
	URL           string `json:"url"`
	CategoryID    *int64 `json:"category_id,omitempty"`
	FetchInterval int    `json:"fetch_interval,omitempty"`
}

type UpdateFeedRequest struct {
	Title         *string `json:"title,omitempty"`
	CategoryID    *int64  `json:"category_id,omitempty"`
	FetchInterval *int    `json:"fetch_interval,omitempty"`
}
```

`backend/internal/models/article.go`:
```go
package models

import "time"

type Article struct {
	ID           int64      `json:"id"`
	FeedID       int64      `json:"feed_id"`
	GUID         string     `json:"guid"`
	Title        string     `json:"title"`
	URL          string     `json:"url"`
	Author       string     `json:"author"`
	ContentRaw   string     `json:"content_raw,omitempty"`
	ContentClean string     `json:"content_clean,omitempty"`
	ThumbnailURL string     `json:"thumbnail_url"`
	PublishedAt  *time.Time `json:"published_at"`
	FetchedAt    time.Time  `json:"fetched_at"`
	WordCount    int        `json:"word_count"`
	ReadingTime  int        `json:"reading_time"`
	IsRead       bool       `json:"is_read"`
	IsStarred    bool       `json:"is_starred"`
	ReadAt       *time.Time `json:"read_at"`
	Score        float64    `json:"score"`
	FeedTitle    string     `json:"feed_title,omitempty"`
	FeedIconURL  string     `json:"feed_icon_url,omitempty"`
	Tags         []string   `json:"tags,omitempty"`
}

type ArticleListParams struct {
	CategoryID *int64  `json:"category_id"`
	FeedID     *int64  `json:"feed_id"`
	Status     string  `json:"status"` // all, unread, starred, read
	Sort       string  `json:"sort"`   // smart, newest, oldest
	Tag        string  `json:"tag"`
	Page       int     `json:"page"`
	Limit      int     `json:"limit"`
}

type UpdateArticleRequest struct {
	IsRead    *bool `json:"is_read,omitempty"`
	IsStarred *bool `json:"is_starred,omitempty"`
}

type BulkArticleRequest struct {
	ArticleIDs []int64 `json:"article_ids"`
	Action     string  `json:"action"` // mark_read, mark_unread, star, unstar
}
```

`backend/internal/models/tag.go`:
```go
package models

type Tag struct {
	ID     int64  `json:"id"`
	UserID int64  `json:"user_id"`
	Name   string `json:"name"`
}

type AddTagRequest struct {
	Name string `json:"name"`
}
```

`backend/internal/models/event.go`:
```go
package models

import "time"

type ReadingEvent struct {
	ID              int64     `json:"id"`
	ArticleID       int64     `json:"article_id"`
	EventType       string    `json:"event_type"` // click, read, star, dismiss
	DurationSeconds int       `json:"duration_seconds"`
	CreatedAt       time.Time `json:"created_at"`
}

type CreateEventRequest struct {
	ArticleID       int64  `json:"article_id"`
	EventType       string `json:"event_type"`
	DurationSeconds int    `json:"duration_seconds"`
}
```

**Step 2: Verify it compiles**

```bash
cd backend && go build ./internal/models/
# Expected: no errors
```

**Step 3: Commit**

```bash
git add backend/internal/models/
git commit -m "feat: add data models for all entities"
```

---

### Task 5: User Store (Data Access Layer)

**Files:**
- Create: `backend/internal/store/user_store.go`
- Create: `backend/internal/store/user_store_test.go`

**Step 1: Write the failing test**

```go
package store

import (
	"testing"

	"github.com/feednest/backend/internal/models"
)

func setupTestDB(t *testing.T) *Queries {
	t.Helper()
	dbPath := t.TempDir() + "/test.db"
	db, err := NewDB(dbPath)
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return New(db)
}

func TestUserStore_CreateAndGet(t *testing.T) {
	q := setupTestDB(t)

	user, err := q.CreateUser("testuser", "test@example.com", "hashedpassword")
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}
	if user.ID == 0 {
		t.Error("expected non-zero ID")
	}
	if user.Username != "testuser" {
		t.Errorf("expected username 'testuser', got %q", user.Username)
	}

	got, err := q.GetUserByUsername("testuser")
	if err != nil {
		t.Fatalf("failed to get user: %v", err)
	}
	if got.ID != user.ID {
		t.Errorf("expected ID %d, got %d", user.ID, got.ID)
	}
}

func TestUserStore_GetUserByUsername_NotFound(t *testing.T) {
	q := setupTestDB(t)

	_, err := q.GetUserByUsername("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent user")
	}
}

func TestUserStore_UserCount(t *testing.T) {
	q := setupTestDB(t)

	count, err := q.UserCount()
	if err != nil {
		t.Fatalf("failed to count users: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 users, got %d", count)
	}

	q.CreateUser("user1", "u1@test.com", "hash")
	count, _ = q.UserCount()
	if count != 1 {
		t.Errorf("expected 1 user, got %d", count)
	}
}
```

**Step 2: Run test to verify it fails**

```bash
cd backend && go test ./internal/store/ -v -run TestUserStore
# Expected: FAIL - Queries type not defined
```

**Step 3: Write user_store.go and the Queries wrapper**

```go
package store

import (
	"database/sql"

	"github.com/feednest/backend/internal/models"
)

type Queries struct {
	db *sql.DB
}

func New(db *sql.DB) *Queries {
	return &Queries{db: db}
}

func (q *Queries) CreateUser(username, email, passwordHash string) (*models.User, error) {
	result, err := q.db.Exec(
		"INSERT INTO users (username, email, password_hash) VALUES (?, ?, ?)",
		username, email, passwordHash,
	)
	if err != nil {
		return nil, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}

	return q.GetUserByID(id)
}

func (q *Queries) GetUserByID(id int64) (*models.User, error) {
	user := &models.User{}
	err := q.db.QueryRow(
		"SELECT id, username, email, password_hash, created_at, updated_at FROM users WHERE id = ?", id,
	).Scan(&user.ID, &user.Username, &user.Email, &user.PasswordHash, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (q *Queries) GetUserByUsername(username string) (*models.User, error) {
	user := &models.User{}
	err := q.db.QueryRow(
		"SELECT id, username, email, password_hash, created_at, updated_at FROM users WHERE username = ?", username,
	).Scan(&user.ID, &user.Username, &user.Email, &user.PasswordHash, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (q *Queries) UserCount() (int, error) {
	var count int
	err := q.db.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
	return count, err
}
```

**Step 4: Run test to verify it passes**

```bash
cd backend && go test ./internal/store/ -v -run TestUserStore
# Expected: PASS
```

**Step 5: Commit**

```bash
git add backend/internal/store/
git commit -m "feat: add user store with create, get, and count operations"
```

---

### Task 6: Auth System (JWT + bcrypt)

**Files:**
- Create: `backend/internal/api/auth.go`
- Create: `backend/internal/api/auth_test.go`
- Modify: `backend/internal/api/router.go`
- Modify: `backend/internal/api/middleware.go`

**Step 1: Install JWT and bcrypt dependencies**

```bash
cd backend
go get github.com/golang-jwt/jwt/v5
go get golang.org/x/crypto/bcrypt
```

**Step 2: Write auth_test.go**

```go
package api

import (
	"testing"
	"time"
)

func TestGenerateAndValidateToken(t *testing.T) {
	secret := "test-secret-key-for-testing-only"
	userID := int64(42)

	token, err := generateAccessToken(userID, secret, 15*time.Minute)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	claims, err := validateToken(token, secret)
	if err != nil {
		t.Fatalf("failed to validate token: %v", err)
	}

	if claims.UserID != userID {
		t.Errorf("expected userID %d, got %d", userID, claims.UserID)
	}
}

func TestValidateToken_Expired(t *testing.T) {
	secret := "test-secret"
	token, _ := generateAccessToken(1, secret, -1*time.Minute)

	_, err := validateToken(token, secret)
	if err == nil {
		t.Error("expected error for expired token")
	}
}

func TestHashAndCheckPassword(t *testing.T) {
	password := "mysecretpassword"
	hash, err := hashPassword(password)
	if err != nil {
		t.Fatalf("failed to hash: %v", err)
	}

	if !checkPassword(password, hash) {
		t.Error("password should match hash")
	}
	if checkPassword("wrongpassword", hash) {
		t.Error("wrong password should not match hash")
	}
}
```

**Step 3: Run test to verify it fails**

```bash
cd backend && go test ./internal/api/ -v -run "TestGenerate|TestHash"
# Expected: FAIL
```

**Step 4: Write auth.go**

```go
package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/feednest/backend/internal/models"
	"github.com/feednest/backend/internal/store"
)

type Claims struct {
	UserID int64 `json:"user_id"`
	jwt.RegisteredClaims
}

type AuthHandler struct {
	store     *store.Queries
	jwtSecret string
}

func NewAuthHandler(store *store.Queries, jwtSecret string) *AuthHandler {
	return &AuthHandler{store: store, jwtSecret: jwtSecret}
}

func generateAccessToken(userID int64, secret string, duration time.Duration) (string, error) {
	claims := Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(duration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

func validateToken(tokenStr string, secret string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
	}
	return claims, nil
}

func hashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

func checkPassword(password, hash string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req models.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	if req.Username == "" || req.Email == "" || req.Password == "" {
		http.Error(w, `{"error":"username, email, and password are required"}`, http.StatusBadRequest)
		return
	}

	hash, err := hashPassword(req.Password)
	if err != nil {
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}

	user, err := h.store.CreateUser(req.Username, req.Email, hash)
	if err != nil {
		http.Error(w, `{"error":"username or email already taken"}`, http.StatusConflict)
		return
	}

	accessToken, err := generateAccessToken(user.ID, h.jwtSecret, 24*time.Hour)
	if err != nil {
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}

	refreshToken, err := generateAccessToken(user.ID, h.jwtSecret, 7*24*time.Hour)
	if err != nil {
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}

	resp := models.AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User:         *user,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req models.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	user, err := h.store.GetUserByUsername(req.Username)
	if err != nil {
		http.Error(w, `{"error":"invalid credentials"}`, http.StatusUnauthorized)
		return
	}

	if !checkPassword(req.Password, user.PasswordHash) {
		http.Error(w, `{"error":"invalid credentials"}`, http.StatusUnauthorized)
		return
	}

	accessToken, _ := generateAccessToken(user.ID, h.jwtSecret, 24*time.Hour)
	refreshToken, _ := generateAccessToken(user.ID, h.jwtSecret, 7*24*time.Hour)

	resp := models.AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User:         *user,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	var body struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	claims, err := validateToken(body.RefreshToken, h.jwtSecret)
	if err != nil {
		http.Error(w, `{"error":"invalid refresh token"}`, http.StatusUnauthorized)
		return
	}

	accessToken, _ := generateAccessToken(claims.UserID, h.jwtSecret, 24*time.Hour)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"access_token": accessToken})
}

func (h *AuthHandler) UserCount(w http.ResponseWriter, r *http.Request) {
	count, err := h.store.UserCount()
	if err != nil {
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]int{"count": count})
}

// ExtractUserID gets the user ID from the request context (set by auth middleware)
func ExtractUserID(r *http.Request) int64 {
	if claims, ok := r.Context().Value(contextKeyUserID).(*Claims); ok {
		return claims.UserID
	}
	return 0
}

type contextKey string

const contextKeyUserID contextKey = "user_claims"
```

**Step 5: Update middleware.go with real JWT validation**

```go
package api

import (
	"context"
	"net/http"
	"strings"
)

func AuthMiddleware(jwtSecret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, `{"error":"authorization header required"}`, http.StatusUnauthorized)
				return
			}

			tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
			if tokenStr == authHeader {
				http.Error(w, `{"error":"invalid authorization format"}`, http.StatusUnauthorized)
				return
			}

			claims, err := validateToken(tokenStr, jwtSecret)
			if err != nil {
				http.Error(w, `{"error":"invalid or expired token"}`, http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), contextKeyUserID, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
```

**Step 6: Update router.go to wire auth routes**

```go
package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"github.com/feednest/backend/internal/store"
)

func NewRouter(queries *store.Queries, jwtSecret string) http.Handler {
	r := chi.NewRouter()

	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)
	r.Use(chimw.RealIP)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:5173", "http://localhost:3000"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	r.Get("/api/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})

	auth := NewAuthHandler(queries, jwtSecret)

	// Public routes
	r.Post("/api/auth/register", auth.Register)
	r.Post("/api/auth/login", auth.Login)
	r.Post("/api/auth/refresh", auth.Refresh)
	r.Get("/api/auth/user-count", auth.UserCount)

	// Protected routes
	r.Group(func(r chi.Router) {
		r.Use(AuthMiddleware(jwtSecret))
		// Handlers will be added in subsequent tasks
	})

	return r
}
```

**Step 7: Update main.go to wire database and router together**

```go
package main

import (
	"log"
	"os"

	"net/http"

	"github.com/feednest/backend/internal/api"
	"github.com/feednest/backend/internal/store"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "./feednest.db"
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "change-me-in-production"
		log.Println("WARNING: using default JWT secret. Set JWT_SECRET env var in production.")
	}

	db, err := store.NewDB(dbPath)
	if err != nil {
		log.Fatalf("failed to initialize database: %v", err)
	}
	defer db.Close()

	queries := store.New(db)
	router := api.NewRouter(queries, jwtSecret)

	log.Printf("FeedNest backend starting on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, router))
}
```

**Step 8: Run all tests**

```bash
cd backend && go test ./... -v
# Expected: all PASS
```

**Step 9: Commit**

```bash
git add backend/
git commit -m "feat: add JWT auth system with register, login, and refresh"
```

---

## Phase 2: Core Backend (Categories, Feeds, Articles)

### Task 7: Category Store and Handler

**Files:**
- Create: `backend/internal/store/category_store.go`
- Create: `backend/internal/store/category_store_test.go`
- Create: `backend/internal/api/handlers/categories.go`

**Step 1: Write category_store_test.go**

```go
package store

import "testing"

func createTestUser(t *testing.T, q *Queries) int64 {
	t.Helper()
	user, err := q.CreateUser("testuser", "test@example.com", "hash")
	if err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}
	return user.ID
}

func TestCategoryStore_CRUD(t *testing.T) {
	q := setupTestDB(t)
	userID := createTestUser(t, q)

	// Create
	cat, err := q.CreateCategory(userID, "Tech", 0)
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}
	if cat.Name != "Tech" {
		t.Errorf("expected 'Tech', got %q", cat.Name)
	}

	// List
	cats, err := q.ListCategories(userID)
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if len(cats) != 1 {
		t.Fatalf("expected 1 category, got %d", len(cats))
	}

	// Update
	newName := "Technology"
	err = q.UpdateCategory(cat.ID, userID, &newName, nil)
	if err != nil {
		t.Fatalf("update failed: %v", err)
	}
	updated, _ := q.ListCategories(userID)
	if updated[0].Name != "Technology" {
		t.Errorf("expected 'Technology', got %q", updated[0].Name)
	}

	// Delete
	err = q.DeleteCategory(cat.ID, userID)
	if err != nil {
		t.Fatalf("delete failed: %v", err)
	}
	remaining, _ := q.ListCategories(userID)
	if len(remaining) != 0 {
		t.Errorf("expected 0 categories after delete, got %d", len(remaining))
	}
}
```

**Step 2: Run test to verify it fails**

```bash
cd backend && go test ./internal/store/ -v -run TestCategoryStore
# Expected: FAIL
```

**Step 3: Write category_store.go**

```go
package store

import "github.com/feednest/backend/internal/models"

func (q *Queries) CreateCategory(userID int64, name string, position int) (*models.Category, error) {
	result, err := q.db.Exec(
		"INSERT INTO categories (user_id, name, position) VALUES (?, ?, ?)",
		userID, name, position,
	)
	if err != nil {
		return nil, err
	}
	id, _ := result.LastInsertId()
	return &models.Category{ID: id, UserID: userID, Name: name, Position: position}, nil
}

func (q *Queries) ListCategories(userID int64) ([]models.Category, error) {
	rows, err := q.db.Query(
		"SELECT id, user_id, name, position FROM categories WHERE user_id = ? ORDER BY position, name",
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var categories []models.Category
	for rows.Next() {
		var c models.Category
		if err := rows.Scan(&c.ID, &c.UserID, &c.Name, &c.Position); err != nil {
			return nil, err
		}
		categories = append(categories, c)
	}
	return categories, nil
}

func (q *Queries) UpdateCategory(id, userID int64, name *string, position *int) error {
	if name != nil {
		if _, err := q.db.Exec("UPDATE categories SET name = ? WHERE id = ? AND user_id = ?", *name, id, userID); err != nil {
			return err
		}
	}
	if position != nil {
		if _, err := q.db.Exec("UPDATE categories SET position = ? WHERE id = ? AND user_id = ?", *position, id, userID); err != nil {
			return err
		}
	}
	return nil
}

func (q *Queries) DeleteCategory(id, userID int64) error {
	_, err := q.db.Exec("DELETE FROM categories WHERE id = ? AND user_id = ?", id, userID)
	return err
}
```

**Step 4: Write categories handler**

```go
package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/feednest/backend/internal/api"
	"github.com/feednest/backend/internal/models"
	"github.com/feednest/backend/internal/store"
)

type CategoryHandler struct {
	store *store.Queries
}

func NewCategoryHandler(store *store.Queries) *CategoryHandler {
	return &CategoryHandler{store: store}
}

func (h *CategoryHandler) List(w http.ResponseWriter, r *http.Request) {
	userID := api.ExtractUserID(r)
	cats, err := h.store.ListCategories(userID)
	if err != nil {
		http.Error(w, `{"error":"failed to list categories"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(cats)
}

func (h *CategoryHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID := api.ExtractUserID(r)
	var req models.CreateCategoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}
	if req.Name == "" {
		http.Error(w, `{"error":"name is required"}`, http.StatusBadRequest)
		return
	}

	cat, err := h.store.CreateCategory(userID, req.Name, req.Position)
	if err != nil {
		http.Error(w, `{"error":"failed to create category"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(cat)
}

func (h *CategoryHandler) Update(w http.ResponseWriter, r *http.Request) {
	userID := api.ExtractUserID(r)
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, `{"error":"invalid id"}`, http.StatusBadRequest)
		return
	}

	var req models.UpdateCategoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	if err := h.store.UpdateCategory(id, userID, req.Name, req.Position); err != nil {
		http.Error(w, `{"error":"failed to update category"}`, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *CategoryHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID := api.ExtractUserID(r)
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, `{"error":"invalid id"}`, http.StatusBadRequest)
		return
	}

	if err := h.store.DeleteCategory(id, userID); err != nil {
		http.Error(w, `{"error":"failed to delete category"}`, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
```

**Step 5: Run tests**

```bash
cd backend && go test ./... -v
# Expected: all PASS
```

**Step 6: Commit**

```bash
git add backend/
git commit -m "feat: add category CRUD store and API handler"
```

---

### Task 8: Feed Store and Handler

**Files:**
- Create: `backend/internal/store/feed_store.go`
- Create: `backend/internal/store/feed_store_test.go`
- Create: `backend/internal/api/handlers/feeds.go`

**Step 1: Write feed_store_test.go**

```go
package store

import "testing"

func TestFeedStore_CRUD(t *testing.T) {
	q := setupTestDB(t)
	userID := createTestUser(t, q)

	// Create
	feed, err := q.CreateFeed(userID, "https://example.com/rss", "Example Feed", "https://example.com", "", nil)
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}
	if feed.Title != "Example Feed" {
		t.Errorf("expected 'Example Feed', got %q", feed.Title)
	}

	// List
	feeds, err := q.ListFeeds(userID)
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if len(feeds) != 1 {
		t.Fatalf("expected 1 feed, got %d", len(feeds))
	}

	// Delete
	err = q.DeleteFeed(feed.ID, userID)
	if err != nil {
		t.Fatalf("delete failed: %v", err)
	}
	remaining, _ := q.ListFeeds(userID)
	if len(remaining) != 0 {
		t.Errorf("expected 0 feeds, got %d", len(remaining))
	}
}

func TestFeedStore_DuplicateURL(t *testing.T) {
	q := setupTestDB(t)
	userID := createTestUser(t, q)

	q.CreateFeed(userID, "https://example.com/rss", "Feed 1", "", "", nil)
	_, err := q.CreateFeed(userID, "https://example.com/rss", "Feed 2", "", "", nil)
	if err == nil {
		t.Error("expected error for duplicate URL")
	}
}
```

**Step 2: Run test to verify it fails**

```bash
cd backend && go test ./internal/store/ -v -run TestFeedStore
# Expected: FAIL
```

**Step 3: Write feed_store.go**

```go
package store

import (
	"time"

	"github.com/feednest/backend/internal/models"
)

func (q *Queries) CreateFeed(userID int64, url, title, siteURL, iconURL string, categoryID *int64) (*models.Feed, error) {
	result, err := q.db.Exec(
		"INSERT INTO feeds (user_id, url, title, site_url, icon_url, category_id) VALUES (?, ?, ?, ?, ?, ?)",
		userID, url, title, siteURL, iconURL, categoryID,
	)
	if err != nil {
		return nil, err
	}
	id, _ := result.LastInsertId()
	return &models.Feed{
		ID:        id,
		UserID:    userID,
		URL:       url,
		Title:     title,
		SiteURL:   siteURL,
		IconURL:   iconURL,
		CategoryID: categoryID,
		FetchInterval: 900,
		CreatedAt: time.Now(),
	}, nil
}

func (q *Queries) ListFeeds(userID int64) ([]models.Feed, error) {
	rows, err := q.db.Query(`
		SELECT f.id, f.user_id, f.url, f.title, f.site_url, f.icon_url,
			f.category_id, f.fetch_interval, f.last_fetched, f.engagement_score, f.created_at,
			COALESCE((SELECT COUNT(*) FROM articles a WHERE a.feed_id = f.id AND a.is_read = 0), 0) as unread_count
		FROM feeds f WHERE f.user_id = ? ORDER BY f.title`, userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var feeds []models.Feed
	for rows.Next() {
		var f models.Feed
		if err := rows.Scan(&f.ID, &f.UserID, &f.URL, &f.Title, &f.SiteURL, &f.IconURL,
			&f.CategoryID, &f.FetchInterval, &f.LastFetched, &f.EngagementScore, &f.CreatedAt, &f.UnreadCount); err != nil {
			return nil, err
		}
		feeds = append(feeds, f)
	}
	return feeds, nil
}

func (q *Queries) GetFeed(id, userID int64) (*models.Feed, error) {
	var f models.Feed
	err := q.db.QueryRow(`
		SELECT id, user_id, url, title, site_url, icon_url, category_id, fetch_interval, last_fetched, engagement_score, created_at
		FROM feeds WHERE id = ? AND user_id = ?`, id, userID,
	).Scan(&f.ID, &f.UserID, &f.URL, &f.Title, &f.SiteURL, &f.IconURL,
		&f.CategoryID, &f.FetchInterval, &f.LastFetched, &f.EngagementScore, &f.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &f, nil
}

func (q *Queries) UpdateFeed(id, userID int64, req *models.UpdateFeedRequest) error {
	if req.Title != nil {
		q.db.Exec("UPDATE feeds SET title = ? WHERE id = ? AND user_id = ?", *req.Title, id, userID)
	}
	if req.CategoryID != nil {
		q.db.Exec("UPDATE feeds SET category_id = ? WHERE id = ? AND user_id = ?", *req.CategoryID, id, userID)
	}
	if req.FetchInterval != nil {
		q.db.Exec("UPDATE feeds SET fetch_interval = ? WHERE id = ? AND user_id = ?", *req.FetchInterval, id, userID)
	}
	return nil
}

func (q *Queries) UpdateFeedLastFetched(id int64) error {
	_, err := q.db.Exec("UPDATE feeds SET last_fetched = CURRENT_TIMESTAMP WHERE id = ?", id)
	return err
}

func (q *Queries) DeleteFeed(id, userID int64) error {
	_, err := q.db.Exec("DELETE FROM feeds WHERE id = ? AND user_id = ?", id, userID)
	return err
}

func (q *Queries) GetFeedsDueForFetch() ([]models.Feed, error) {
	rows, err := q.db.Query(`
		SELECT id, user_id, url, title, site_url, icon_url, category_id, fetch_interval, last_fetched, engagement_score, created_at
		FROM feeds
		WHERE last_fetched IS NULL
		   OR (strftime('%s','now') - strftime('%s', last_fetched)) >= fetch_interval
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var feeds []models.Feed
	for rows.Next() {
		var f models.Feed
		if err := rows.Scan(&f.ID, &f.UserID, &f.URL, &f.Title, &f.SiteURL, &f.IconURL,
			&f.CategoryID, &f.FetchInterval, &f.LastFetched, &f.EngagementScore, &f.CreatedAt); err != nil {
			return nil, err
		}
		feeds = append(feeds, f)
	}
	return feeds, nil
}
```

**Step 4: Write feeds handler** (`backend/internal/api/handlers/feeds.go`)

```go
package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/feednest/backend/internal/api"
	"github.com/feednest/backend/internal/models"
	"github.com/feednest/backend/internal/store"
)

type FeedHandler struct {
	store *store.Queries
}

func NewFeedHandler(store *store.Queries) *FeedHandler {
	return &FeedHandler{store: store}
}

func (h *FeedHandler) List(w http.ResponseWriter, r *http.Request) {
	userID := api.ExtractUserID(r)
	feeds, err := h.store.ListFeeds(userID)
	if err != nil {
		http.Error(w, `{"error":"failed to list feeds"}`, http.StatusInternalServerError)
		return
	}
	if feeds == nil {
		feeds = []models.Feed{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(feeds)
}

func (h *FeedHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID := api.ExtractUserID(r)
	var req models.CreateFeedRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}
	if req.URL == "" {
		http.Error(w, `{"error":"url is required"}`, http.StatusBadRequest)
		return
	}

	// TODO: Auto-discover RSS URL from page URL (Task 11)
	feed, err := h.store.CreateFeed(userID, req.URL, "", "", "", req.CategoryID)
	if err != nil {
		http.Error(w, `{"error":"failed to create feed or URL already exists"}`, http.StatusConflict)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(feed)
}

func (h *FeedHandler) Update(w http.ResponseWriter, r *http.Request) {
	userID := api.ExtractUserID(r)
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, `{"error":"invalid id"}`, http.StatusBadRequest)
		return
	}

	var req models.UpdateFeedRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	if err := h.store.UpdateFeed(id, userID, &req); err != nil {
		http.Error(w, `{"error":"failed to update feed"}`, http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *FeedHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID := api.ExtractUserID(r)
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, `{"error":"invalid id"}`, http.StatusBadRequest)
		return
	}

	if err := h.store.DeleteFeed(id, userID); err != nil {
		http.Error(w, `{"error":"failed to delete feed"}`, http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
```

**Step 5: Run tests**

```bash
cd backend && go test ./... -v
# Expected: all PASS
```

**Step 6: Commit**

```bash
git add backend/
git commit -m "feat: add feed CRUD store and API handler"
```

---

### Task 9: Article Store and Handler

**Files:**
- Create: `backend/internal/store/article_store.go`
- Create: `backend/internal/store/article_store_test.go`
- Create: `backend/internal/api/handlers/articles.go`

**Step 1: Write article_store_test.go**

```go
package store

import (
	"testing"
	"time"
)

func TestArticleStore_CreateAndList(t *testing.T) {
	q := setupTestDB(t)
	userID := createTestUser(t, q)
	feed, _ := q.CreateFeed(userID, "https://example.com/rss", "Test Feed", "", "", nil)

	now := time.Now()
	err := q.CreateArticle(feed.ID, "guid-1", "Article 1", "https://example.com/1", "Author", "<p>raw</p>", "<p>clean</p>", "https://img.com/1.jpg", &now, 200, 1)
	if err != nil {
		t.Fatalf("create article failed: %v", err)
	}

	// Duplicate guid should be ignored
	err = q.CreateArticle(feed.ID, "guid-1", "Article 1 Dup", "", "", "", "", "", nil, 0, 0)
	if err != nil {
		t.Fatalf("expected upsert to not fail: %v", err)
	}

	articles, total, err := q.ListArticles(userID, &ArticleFilter{Limit: 30, Page: 1, Sort: "newest"})
	if err != nil {
		t.Fatalf("list articles failed: %v", err)
	}
	if total != 1 {
		t.Errorf("expected 1 article, got %d", total)
	}
	if len(articles) != 1 {
		t.Fatalf("expected 1 article in slice, got %d", len(articles))
	}
	if articles[0].Title != "Article 1" {
		t.Errorf("expected 'Article 1', got %q", articles[0].Title)
	}
}

func TestArticleStore_MarkReadAndStar(t *testing.T) {
	q := setupTestDB(t)
	userID := createTestUser(t, q)
	feed, _ := q.CreateFeed(userID, "https://example.com/rss", "Feed", "", "", nil)

	now := time.Now()
	q.CreateArticle(feed.ID, "guid-1", "Article", "", "", "", "", "", &now, 100, 1)

	articles, _, _ := q.ListArticles(userID, &ArticleFilter{Limit: 30, Page: 1, Sort: "newest"})
	articleID := articles[0].ID

	isRead := true
	err := q.UpdateArticle(articleID, userID, &isRead, nil)
	if err != nil {
		t.Fatalf("mark read failed: %v", err)
	}

	article, err := q.GetArticle(articleID, userID)
	if err != nil {
		t.Fatalf("get article failed: %v", err)
	}
	if !article.IsRead {
		t.Error("expected article to be read")
	}
}
```

**Step 2: Run test to verify it fails**

```bash
cd backend && go test ./internal/store/ -v -run TestArticleStore
# Expected: FAIL
```

**Step 3: Write article_store.go**

```go
package store

import (
	"fmt"
	"strings"
	"time"

	"github.com/feednest/backend/internal/models"
)

type ArticleFilter struct {
	CategoryID *int64
	FeedID     *int64
	Status     string // all, unread, starred, read
	Sort       string // smart, newest, oldest
	Tag        string
	Page       int
	Limit      int
}

func (q *Queries) CreateArticle(feedID int64, guid, title, url, author, contentRaw, contentClean, thumbnailURL string, publishedAt *time.Time, wordCount, readingTime int) error {
	_, err := q.db.Exec(`
		INSERT OR IGNORE INTO articles (feed_id, guid, title, url, author, content_raw, content_clean, thumbnail_url, published_at, word_count, reading_time)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		feedID, guid, title, url, author, contentRaw, contentClean, thumbnailURL, publishedAt, wordCount, readingTime,
	)
	return err
}

func (q *Queries) GetArticle(id, userID int64) (*models.Article, error) {
	var a models.Article
	err := q.db.QueryRow(`
		SELECT a.id, a.feed_id, a.guid, a.title, a.url, a.author, a.content_raw, a.content_clean,
			a.thumbnail_url, a.published_at, a.fetched_at, a.word_count, a.reading_time,
			a.is_read, a.is_starred, a.read_at, a.score,
			COALESCE(f.title, '') as feed_title, COALESCE(f.icon_url, '') as feed_icon_url
		FROM articles a
		JOIN feeds f ON a.feed_id = f.id
		WHERE a.id = ? AND f.user_id = ?`, id, userID,
	).Scan(&a.ID, &a.FeedID, &a.GUID, &a.Title, &a.URL, &a.Author, &a.ContentRaw, &a.ContentClean,
		&a.ThumbnailURL, &a.PublishedAt, &a.FetchedAt, &a.WordCount, &a.ReadingTime,
		&a.IsRead, &a.IsStarred, &a.ReadAt, &a.Score, &a.FeedTitle, &a.FeedIconURL)
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func (q *Queries) ListArticles(userID int64, filter *ArticleFilter) ([]models.Article, int, error) {
	var conditions []string
	var args []interface{}

	conditions = append(conditions, "f.user_id = ?")
	args = append(args, userID)

	if filter.FeedID != nil {
		conditions = append(conditions, "a.feed_id = ?")
		args = append(args, *filter.FeedID)
	}
	if filter.CategoryID != nil {
		conditions = append(conditions, "f.category_id = ?")
		args = append(args, *filter.CategoryID)
	}
	switch filter.Status {
	case "unread":
		conditions = append(conditions, "a.is_read = 0")
	case "starred":
		conditions = append(conditions, "a.is_starred = 1")
	case "read":
		conditions = append(conditions, "a.is_read = 1")
	}
	if filter.Tag != "" {
		conditions = append(conditions, "EXISTS (SELECT 1 FROM article_tags at JOIN tags t ON at.tag_id = t.id WHERE at.article_id = a.id AND t.name = ?)")
		args = append(args, filter.Tag)
	}

	where := strings.Join(conditions, " AND ")

	// Count total
	var total int
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM articles a JOIN feeds f ON a.feed_id = f.id WHERE %s", where)
	if err := q.db.QueryRow(countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// Order
	var orderBy string
	switch filter.Sort {
	case "oldest":
		orderBy = "COALESCE(a.published_at, a.fetched_at) ASC"
	case "smart":
		orderBy = "a.score DESC, COALESCE(a.published_at, a.fetched_at) DESC"
	default: // newest
		orderBy = "COALESCE(a.published_at, a.fetched_at) DESC"
	}

	offset := (filter.Page - 1) * filter.Limit
	query := fmt.Sprintf(`
		SELECT a.id, a.feed_id, a.guid, a.title, a.url, a.author, '', '',
			a.thumbnail_url, a.published_at, a.fetched_at, a.word_count, a.reading_time,
			a.is_read, a.is_starred, a.read_at, a.score,
			COALESCE(f.title, '') as feed_title, COALESCE(f.icon_url, '') as feed_icon_url
		FROM articles a
		JOIN feeds f ON a.feed_id = f.id
		WHERE %s
		ORDER BY %s
		LIMIT ? OFFSET ?`, where, orderBy)

	queryArgs := append(args, filter.Limit, offset)
	rows, err := q.db.Query(query, queryArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var articles []models.Article
	for rows.Next() {
		var a models.Article
		if err := rows.Scan(&a.ID, &a.FeedID, &a.GUID, &a.Title, &a.URL, &a.Author, &a.ContentRaw, &a.ContentClean,
			&a.ThumbnailURL, &a.PublishedAt, &a.FetchedAt, &a.WordCount, &a.ReadingTime,
			&a.IsRead, &a.IsStarred, &a.ReadAt, &a.Score, &a.FeedTitle, &a.FeedIconURL); err != nil {
			return nil, 0, err
		}
		articles = append(articles, a)
	}
	return articles, total, nil
}

func (q *Queries) UpdateArticle(id, userID int64, isRead *bool, isStarred *bool) error {
	if isRead != nil {
		var readAt interface{}
		if *isRead {
			readAt = time.Now()
		}
		q.db.Exec(`
			UPDATE articles SET is_read = ?, read_at = ?
			WHERE id = ? AND feed_id IN (SELECT id FROM feeds WHERE user_id = ?)`,
			*isRead, readAt, id, userID)
	}
	if isStarred != nil {
		q.db.Exec(`
			UPDATE articles SET is_starred = ?
			WHERE id = ? AND feed_id IN (SELECT id FROM feeds WHERE user_id = ?)`,
			*isStarred, id, userID)
	}
	return nil
}

func (q *Queries) BulkUpdateArticles(userID int64, articleIDs []int64, action string) error {
	if len(articleIDs) == 0 {
		return nil
	}

	placeholders := strings.Repeat("?,", len(articleIDs))
	placeholders = placeholders[:len(placeholders)-1]

	var query string
	args := make([]interface{}, 0, len(articleIDs)+1)

	switch action {
	case "mark_read":
		query = fmt.Sprintf(`UPDATE articles SET is_read = 1, read_at = CURRENT_TIMESTAMP
			WHERE id IN (%s) AND feed_id IN (SELECT id FROM feeds WHERE user_id = ?)`, placeholders)
	case "mark_unread":
		query = fmt.Sprintf(`UPDATE articles SET is_read = 0, read_at = NULL
			WHERE id IN (%s) AND feed_id IN (SELECT id FROM feeds WHERE user_id = ?)`, placeholders)
	case "star":
		query = fmt.Sprintf(`UPDATE articles SET is_starred = 1
			WHERE id IN (%s) AND feed_id IN (SELECT id FROM feeds WHERE user_id = ?)`, placeholders)
	case "unstar":
		query = fmt.Sprintf(`UPDATE articles SET is_starred = 0
			WHERE id IN (%s) AND feed_id IN (SELECT id FROM feeds WHERE user_id = ?)`, placeholders)
	default:
		return fmt.Errorf("unknown action: %s", action)
	}

	for _, id := range articleIDs {
		args = append(args, id)
	}
	args = append(args, userID)

	_, err := q.db.Exec(query, args...)
	return err
}
```

**Step 4: Write articles handler** (`backend/internal/api/handlers/articles.go`)

```go
package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/feednest/backend/internal/api"
	"github.com/feednest/backend/internal/models"
	"github.com/feednest/backend/internal/store"
)

type ArticleHandler struct {
	store *store.Queries
}

func NewArticleHandler(store *store.Queries) *ArticleHandler {
	return &ArticleHandler{store: store}
}

func (h *ArticleHandler) List(w http.ResponseWriter, r *http.Request) {
	userID := api.ExtractUserID(r)

	filter := &store.ArticleFilter{
		Status: r.URL.Query().Get("status"),
		Sort:   r.URL.Query().Get("sort"),
		Tag:    r.URL.Query().Get("tag"),
		Page:   1,
		Limit:  30,
	}

	if filter.Sort == "" {
		filter.Sort = "smart"
	}

	if p := r.URL.Query().Get("page"); p != "" {
		filter.Page, _ = strconv.Atoi(p)
	}
	if l := r.URL.Query().Get("limit"); l != "" {
		filter.Limit, _ = strconv.Atoi(l)
	}
	if feedID := r.URL.Query().Get("feed"); feedID != "" {
		id, _ := strconv.ParseInt(feedID, 10, 64)
		filter.FeedID = &id
	}
	if catID := r.URL.Query().Get("category"); catID != "" {
		id, _ := strconv.ParseInt(catID, 10, 64)
		filter.CategoryID = &id
	}

	articles, total, err := h.store.ListArticles(userID, filter)
	if err != nil {
		http.Error(w, `{"error":"failed to list articles"}`, http.StatusInternalServerError)
		return
	}
	if articles == nil {
		articles = []models.Article{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"articles": articles,
		"total":    total,
		"page":     filter.Page,
		"limit":    filter.Limit,
	})
}

func (h *ArticleHandler) Get(w http.ResponseWriter, r *http.Request) {
	userID := api.ExtractUserID(r)
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, `{"error":"invalid id"}`, http.StatusBadRequest)
		return
	}

	article, err := h.store.GetArticle(id, userID)
	if err != nil {
		http.Error(w, `{"error":"article not found"}`, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(article)
}

func (h *ArticleHandler) Update(w http.ResponseWriter, r *http.Request) {
	userID := api.ExtractUserID(r)
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, `{"error":"invalid id"}`, http.StatusBadRequest)
		return
	}

	var req models.UpdateArticleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	if err := h.store.UpdateArticle(id, userID, req.IsRead, req.IsStarred); err != nil {
		http.Error(w, `{"error":"failed to update article"}`, http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *ArticleHandler) Dismiss(w http.ResponseWriter, r *http.Request) {
	userID := api.ExtractUserID(r)
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, `{"error":"invalid id"}`, http.StatusBadRequest)
		return
	}

	isRead := true
	h.store.UpdateArticle(id, userID, &isRead, nil)

	// Log dismiss event for scoring
	h.store.CreateReadingEvent(id, "dismiss", 0)

	w.WriteHeader(http.StatusNoContent)
}

func (h *ArticleHandler) Bulk(w http.ResponseWriter, r *http.Request) {
	userID := api.ExtractUserID(r)
	var req models.BulkArticleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	if err := h.store.BulkUpdateArticles(userID, req.ArticleIDs, req.Action); err != nil {
		http.Error(w, `{"error":"failed to perform bulk action"}`, http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
```

**Step 5: Run tests**

```bash
cd backend && go test ./... -v
# Expected: all PASS
```

**Step 6: Commit**

```bash
git add backend/
git commit -m "feat: add article store and API handler with filtering and bulk ops"
```

---

### Task 10: Tags, Events, and Settings Stores + Handlers

**Files:**
- Create: `backend/internal/store/tag_store.go`
- Create: `backend/internal/store/event_store.go`
- Create: `backend/internal/store/settings_store.go`
- Create: `backend/internal/api/handlers/tags.go`
- Create: `backend/internal/api/handlers/events.go`
- Create: `backend/internal/api/handlers/settings.go`

**Step 1: Write tag_store.go**

```go
package store

import "github.com/feednest/backend/internal/models"

func (q *Queries) ListTags(userID int64) ([]models.Tag, error) {
	rows, err := q.db.Query("SELECT id, user_id, name FROM tags WHERE user_id = ? ORDER BY name", userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tags []models.Tag
	for rows.Next() {
		var t models.Tag
		if err := rows.Scan(&t.ID, &t.UserID, &t.Name); err != nil {
			return nil, err
		}
		tags = append(tags, t)
	}
	return tags, nil
}

func (q *Queries) AddTagToArticle(userID int64, articleID int64, tagName string) error {
	// Create tag if it doesn't exist
	_, err := q.db.Exec("INSERT OR IGNORE INTO tags (user_id, name) VALUES (?, ?)", userID, tagName)
	if err != nil {
		return err
	}

	// Get tag ID
	var tagID int64
	err = q.db.QueryRow("SELECT id FROM tags WHERE user_id = ? AND name = ?", userID, tagName).Scan(&tagID)
	if err != nil {
		return err
	}

	// Link article to tag
	_, err = q.db.Exec("INSERT OR IGNORE INTO article_tags (article_id, tag_id) VALUES (?, ?)", articleID, tagID)
	return err
}

func (q *Queries) RemoveTagFromArticle(articleID int64, tagName string, userID int64) error {
	_, err := q.db.Exec(`
		DELETE FROM article_tags
		WHERE article_id = ? AND tag_id = (SELECT id FROM tags WHERE name = ? AND user_id = ?)`,
		articleID, tagName, userID)
	return err
}

func (q *Queries) GetArticleTags(articleID int64) ([]string, error) {
	rows, err := q.db.Query(`
		SELECT t.name FROM tags t
		JOIN article_tags at ON t.id = at.tag_id
		WHERE at.article_id = ? ORDER BY t.name`, articleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tags []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		tags = append(tags, name)
	}
	return tags, nil
}
```

**Step 2: Write event_store.go**

```go
package store

func (q *Queries) CreateReadingEvent(articleID int64, eventType string, durationSeconds int) error {
	_, err := q.db.Exec(
		"INSERT INTO reading_events (article_id, event_type, duration_seconds) VALUES (?, ?, ?)",
		articleID, eventType, durationSeconds,
	)
	return err
}
```

**Step 3: Write settings_store.go**

```go
package store

func (q *Queries) GetSettings(userID int64) (map[string]string, error) {
	rows, err := q.db.Query("SELECT key, value FROM settings WHERE user_id = ?", userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	settings := make(map[string]string)
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return nil, err
		}
		settings[key] = value
	}
	return settings, nil
}

func (q *Queries) SetSetting(userID int64, key, value string) error {
	_, err := q.db.Exec(
		"INSERT INTO settings (user_id, key, value) VALUES (?, ?, ?) ON CONFLICT(user_id, key) DO UPDATE SET value = ?",
		userID, key, value, value,
	)
	return err
}

func (q *Queries) UpdateSettings(userID int64, settings map[string]string) error {
	for key, value := range settings {
		if err := q.SetSetting(userID, key, value); err != nil {
			return err
		}
	}
	return nil
}
```

**Step 4: Write handlers for tags, events, settings**

`backend/internal/api/handlers/tags.go`:
```go
package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/feednest/backend/internal/api"
	"github.com/feednest/backend/internal/models"
	"github.com/feednest/backend/internal/store"
)

type TagHandler struct {
	store *store.Queries
}

func NewTagHandler(store *store.Queries) *TagHandler {
	return &TagHandler{store: store}
}

func (h *TagHandler) List(w http.ResponseWriter, r *http.Request) {
	userID := api.ExtractUserID(r)
	tags, err := h.store.ListTags(userID)
	if err != nil {
		http.Error(w, `{"error":"failed to list tags"}`, http.StatusInternalServerError)
		return
	}
	if tags == nil {
		tags = []models.Tag{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tags)
}

func (h *TagHandler) AddToArticle(w http.ResponseWriter, r *http.Request) {
	userID := api.ExtractUserID(r)
	articleID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, `{"error":"invalid id"}`, http.StatusBadRequest)
		return
	}

	var req models.AddTagRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" {
		http.Error(w, `{"error":"name is required"}`, http.StatusBadRequest)
		return
	}

	if err := h.store.AddTagToArticle(userID, articleID, req.Name); err != nil {
		http.Error(w, `{"error":"failed to add tag"}`, http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func (h *TagHandler) RemoveFromArticle(w http.ResponseWriter, r *http.Request) {
	userID := api.ExtractUserID(r)
	articleID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, `{"error":"invalid id"}`, http.StatusBadRequest)
		return
	}
	tagName := chi.URLParam(r, "tag")

	if err := h.store.RemoveTagFromArticle(articleID, tagName, userID); err != nil {
		http.Error(w, `{"error":"failed to remove tag"}`, http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
```

`backend/internal/api/handlers/events.go`:
```go
package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/feednest/backend/internal/models"
	"github.com/feednest/backend/internal/store"
)

type EventHandler struct {
	store *store.Queries
}

func NewEventHandler(store *store.Queries) *EventHandler {
	return &EventHandler{store: store}
}

func (h *EventHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req models.CreateEventRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	if err := h.store.CreateReadingEvent(req.ArticleID, req.EventType, req.DurationSeconds); err != nil {
		http.Error(w, `{"error":"failed to create event"}`, http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
}
```

`backend/internal/api/handlers/settings.go`:
```go
package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/feednest/backend/internal/api"
	"github.com/feednest/backend/internal/store"
)

type SettingsHandler struct {
	store *store.Queries
}

func NewSettingsHandler(store *store.Queries) *SettingsHandler {
	return &SettingsHandler{store: store}
}

func (h *SettingsHandler) Get(w http.ResponseWriter, r *http.Request) {
	userID := api.ExtractUserID(r)
	settings, err := h.store.GetSettings(userID)
	if err != nil {
		http.Error(w, `{"error":"failed to get settings"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(settings)
}

func (h *SettingsHandler) Update(w http.ResponseWriter, r *http.Request) {
	userID := api.ExtractUserID(r)
	var settings map[string]string
	if err := json.NewDecoder(r.Body).Decode(&settings); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	if err := h.store.UpdateSettings(userID, settings); err != nil {
		http.Error(w, `{"error":"failed to update settings"}`, http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
```

**Step 5: Wire all handlers into router.go** -- update the protected routes group:

```go
// Inside NewRouter, in the protected r.Group:
r.Group(func(r chi.Router) {
    r.Use(AuthMiddleware(jwtSecret))

    categories := handlers.NewCategoryHandler(queries)
    r.Get("/api/categories", categories.List)
    r.Post("/api/categories", categories.Create)
    r.Put("/api/categories/{id}", categories.Update)
    r.Delete("/api/categories/{id}", categories.Delete)

    feeds := handlers.NewFeedHandler(queries)
    r.Get("/api/feeds", feeds.List)
    r.Post("/api/feeds", feeds.Create)
    r.Put("/api/feeds/{id}", feeds.Update)
    r.Delete("/api/feeds/{id}", feeds.Delete)

    articles := handlers.NewArticleHandler(queries)
    r.Get("/api/articles", articles.List)
    r.Get("/api/articles/{id}", articles.Get)
    r.Put("/api/articles/{id}", articles.Update)
    r.Post("/api/articles/{id}/dismiss", articles.Dismiss)
    r.Post("/api/articles/bulk", articles.Bulk)

    tags := handlers.NewTagHandler(queries)
    r.Get("/api/tags", tags.List)
    r.Post("/api/articles/{id}/tags", tags.AddToArticle)
    r.Delete("/api/articles/{id}/tags/{tag}", tags.RemoveFromArticle)

    events := handlers.NewEventHandler(queries)
    r.Post("/api/events", events.Create)

    settingsH := handlers.NewSettingsHandler(queries)
    r.Get("/api/settings", settingsH.Get)
    r.Put("/api/settings", settingsH.Update)
})
```

**Step 6: Run tests**

```bash
cd backend && go test ./... -v
# Expected: all PASS
```

**Step 7: Commit**

```bash
git add backend/
git commit -m "feat: add tags, events, settings stores and handlers, wire all routes"
```

---

## Phase 3: Feed Fetching and Content Extraction

### Task 11: RSS/Atom Feed Fetcher and Parser

**Files:**
- Create: `backend/internal/fetcher/fetcher.go`
- Create: `backend/internal/fetcher/fetcher_test.go`

**Step 1: Install gofeed**

```bash
cd backend && go get github.com/mmcdole/gofeed
```

**Step 2: Write fetcher_test.go**

```go
package fetcher

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFetchFeed(t *testing.T) {
	rssXML := `<?xml version="1.0"?>
<rss version="2.0">
  <channel>
    <title>Test Feed</title>
    <link>https://example.com</link>
    <item>
      <title>Article 1</title>
      <link>https://example.com/1</link>
      <guid>guid-1</guid>
      <description>Content 1</description>
    </item>
    <item>
      <title>Article 2</title>
      <link>https://example.com/2</link>
      <guid>guid-2</guid>
      <description>Content 2</description>
    </item>
  </channel>
</rss>`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.Write([]byte(rssXML))
	}))
	defer server.Close()

	result, err := FetchFeed(server.URL)
	if err != nil {
		t.Fatalf("fetch failed: %v", err)
	}
	if result.Title != "Test Feed" {
		t.Errorf("expected 'Test Feed', got %q", result.Title)
	}
	if len(result.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(result.Items))
	}
	if result.Items[0].Title != "Article 1" {
		t.Errorf("expected 'Article 1', got %q", result.Items[0].Title)
	}
}
```

**Step 3: Run test to verify it fails**

```bash
cd backend && go test ./internal/fetcher/ -v -run TestFetchFeed
# Expected: FAIL
```

**Step 4: Write fetcher.go**

```go
package fetcher

import (
	"fmt"
	"math"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/mmcdole/gofeed"
)

type FeedResult struct {
	Title   string
	SiteURL string
	IconURL string
	Items   []FeedItem
}

type FeedItem struct {
	GUID         string
	Title        string
	URL          string
	Author       string
	ContentRaw   string
	ThumbnailURL string
	PublishedAt  *time.Time
	WordCount    int
	ReadingTime  int
}

func FetchFeed(url string) (*FeedResult, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d for %s", resp.StatusCode, url)
	}

	fp := gofeed.NewParser()
	feed, err := fp.Parse(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse feed %s: %w", url, err)
	}

	result := &FeedResult{
		Title:   feed.Title,
		SiteURL: feed.Link,
	}

	if feed.Image != nil {
		result.IconURL = feed.Image.URL
	}

	for _, item := range feed.Items {
		fi := FeedItem{
			GUID:  item.GUID,
			Title: item.Title,
			URL:   item.Link,
		}

		if fi.GUID == "" {
			fi.GUID = item.Link
		}

		if item.Author != nil {
			fi.Author = item.Author.Name
		}

		content := item.Content
		if content == "" {
			content = item.Description
		}
		fi.ContentRaw = content

		// Extract thumbnail from media or enclosure
		if item.Image != nil {
			fi.ThumbnailURL = item.Image.URL
		}
		if fi.ThumbnailURL == "" && len(item.Enclosures) > 0 {
			for _, enc := range item.Enclosures {
				if strings.HasPrefix(enc.Type, "image/") {
					fi.ThumbnailURL = enc.URL
					break
				}
			}
		}

		if item.PublishedParsed != nil {
			fi.PublishedAt = item.PublishedParsed
		} else if item.UpdatedParsed != nil {
			fi.PublishedAt = item.UpdatedParsed
		}

		fi.WordCount = countWords(content)
		fi.ReadingTime = int(math.Ceil(float64(fi.WordCount) / 200.0))

		result.Items = append(result.Items, fi)
	}

	return result, nil
}

func countWords(s string) int {
	// Strip HTML tags crudely for word count
	inTag := false
	var text strings.Builder
	for _, r := range s {
		if r == '<' {
			inTag = true
			continue
		}
		if r == '>' {
			inTag = false
			text.WriteRune(' ')
			continue
		}
		if !inTag {
			text.WriteRune(r)
		}
	}

	words := strings.Fields(text.String())
	count := 0
	for _, w := range words {
		if utf8.RuneCountInString(w) > 0 {
			count++
		}
	}
	return count
}
```

**Step 5: Run tests**

```bash
cd backend && go test ./internal/fetcher/ -v
# Expected: PASS
```

**Step 6: Commit**

```bash
git add backend/
git commit -m "feat: add RSS/Atom feed fetcher with gofeed parser"
```

---

### Task 12: Content Extraction (Readability)

**Files:**
- Create: `backend/internal/readability/readability.go`
- Create: `backend/internal/readability/readability_test.go`

**Step 1: Install go-readability**

```bash
cd backend && go get github.com/go-shiori/go-readability
```

**Step 2: Write readability_test.go**

```go
package readability

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestExtractContent(t *testing.T) {
	html := `<html><head><title>Test</title></head><body>
		<nav>Navigation</nav>
		<article><h1>Hello World</h1><p>This is the main content of the article. It has enough text to be extracted properly by the readability algorithm.</p></article>
		<footer>Footer</footer>
	</body></html>`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(html))
	}))
	defer server.Close()

	content, err := ExtractContent(server.URL)
	if err != nil {
		t.Fatalf("extract failed: %v", err)
	}
	if content == "" {
		t.Error("expected non-empty content")
	}
}

func TestExtractThumbnail(t *testing.T) {
	html := `<html><head>
		<meta property="og:image" content="https://example.com/image.jpg">
	</head><body><article><p>Content</p></article></body></html>`

	thumb := ExtractThumbnailFromHTML(html)
	if thumb != "https://example.com/image.jpg" {
		t.Errorf("expected og:image URL, got %q", thumb)
	}
}
```

**Step 3: Run test to verify it fails**

```bash
cd backend && go test ./internal/readability/ -v
# Expected: FAIL
```

**Step 4: Write readability.go**

```go
package readability

import (
	"net/url"
	"regexp"
	"strings"
	"time"

	goreadability "github.com/go-shiori/go-readability"
)

func ExtractContent(articleURL string) (string, error) {
	u, err := url.Parse(articleURL)
	if err != nil {
		return "", err
	}

	article, err := goreadability.FromURL(articleURL, 30*time.Second)
	if err != nil {
		return "", err
	}

	_ = u // used for resolving relative URLs if needed
	return article.Content, nil
}

var ogImageRe = regexp.MustCompile(`<meta[^>]+property=["']og:image["'][^>]+content=["']([^"']+)["']`)
var ogImageRe2 = regexp.MustCompile(`<meta[^>]+content=["']([^"']+)["'][^>]+property=["']og:image["']`)

func ExtractThumbnailFromHTML(html string) string {
	matches := ogImageRe.FindStringSubmatch(html)
	if len(matches) > 1 {
		return strings.TrimSpace(matches[1])
	}
	matches = ogImageRe2.FindStringSubmatch(html)
	if len(matches) > 1 {
		return strings.TrimSpace(matches[1])
	}
	return ""
}
```

**Step 5: Run tests**

```bash
cd backend && go test ./internal/readability/ -v
# Expected: PASS
```

**Step 6: Commit**

```bash
git add backend/
git commit -m "feat: add readability content extraction and OG image parsing"
```

---

### Task 13: Background Feed Scheduler

**Files:**
- Create: `backend/internal/scheduler/scheduler.go`
- Modify: `backend/cmd/feednest/main.go`

**Step 1: Write scheduler.go**

```go
package scheduler

import (
	"log"
	"sync"
	"time"

	"github.com/feednest/backend/internal/fetcher"
	"github.com/feednest/backend/internal/readability"
	"github.com/feednest/backend/internal/store"
)

type Scheduler struct {
	store    *store.Queries
	interval time.Duration
	stop     chan struct{}
}

func New(store *store.Queries, interval time.Duration) *Scheduler {
	return &Scheduler{
		store:    store,
		interval: interval,
		stop:     make(chan struct{}),
	}
}

func (s *Scheduler) Start() {
	go func() {
		// Run immediately on start
		s.fetchAll()

		ticker := time.NewTicker(s.interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				s.fetchAll()
			case <-s.stop:
				return
			}
		}
	}()
	log.Printf("Feed scheduler started (interval: %v)", s.interval)
}

func (s *Scheduler) Stop() {
	close(s.stop)
}

func (s *Scheduler) fetchAll() {
	feeds, err := s.store.GetFeedsDueForFetch()
	if err != nil {
		log.Printf("scheduler: failed to get feeds: %v", err)
		return
	}

	if len(feeds) == 0 {
		return
	}

	log.Printf("scheduler: fetching %d feeds", len(feeds))

	var wg sync.WaitGroup
	sem := make(chan struct{}, 5) // limit concurrency to 5

	for _, feed := range feeds {
		wg.Add(1)
		sem <- struct{}{}

		go func(feedID int64, feedURL, feedTitle string) {
			defer wg.Done()
			defer func() { <-sem }()

			result, err := fetcher.FetchFeed(feedURL)
			if err != nil {
				log.Printf("scheduler: failed to fetch %s: %v", feedURL, err)
				return
			}

			// Update feed metadata if it was empty
			if feedTitle == "" && result.Title != "" {
				title := result.Title
				s.store.UpdateFeed(feedID, 0, &store.FeedUpdate{Title: &title, SiteURL: &result.SiteURL, IconURL: &result.IconURL})
			}

			for _, item := range result.Items {
				thumbnailURL := item.ThumbnailURL

				// Try to extract clean content and thumbnail
				var contentClean string
				if item.URL != "" {
					if clean, err := readability.ExtractContent(item.URL); err == nil {
						contentClean = clean
					}
					if thumbnailURL == "" {
						thumbnailURL = readability.ExtractThumbnailFromHTML(item.ContentRaw)
					}
				}

				s.store.CreateArticle(
					feedID, item.GUID, item.Title, item.URL, item.Author,
					item.ContentRaw, contentClean, thumbnailURL,
					item.PublishedAt, item.WordCount, item.ReadingTime,
				)
			}

			s.store.UpdateFeedLastFetched(feedID)
			log.Printf("scheduler: fetched %s (%d items)", feedURL, len(result.Items))
		}(feed.ID, feed.URL, feed.Title)
	}

	wg.Wait()
}
```

Note: The `UpdateFeed` call in the scheduler uses a different signature than the handler's version. We need a simpler internal update method. Add to `feed_store.go`:

```go
type FeedUpdate struct {
	Title   *string
	SiteURL *string
	IconURL *string
}

func (q *Queries) UpdateFeedMetadata(id int64, update *FeedUpdate) error {
	if update.Title != nil {
		q.db.Exec("UPDATE feeds SET title = ? WHERE id = ?", *update.Title, id)
	}
	if update.SiteURL != nil {
		q.db.Exec("UPDATE feeds SET site_url = ? WHERE id = ?", *update.SiteURL, id)
	}
	if update.IconURL != nil {
		q.db.Exec("UPDATE feeds SET icon_url = ? WHERE id = ?", *update.IconURL, id)
	}
	return nil
}
```

Update the scheduler to use `UpdateFeedMetadata` instead of `UpdateFeed`.

**Step 2: Update main.go to start the scheduler**

Add after router creation:

```go
import "github.com/feednest/backend/internal/scheduler"

// In main(), after router creation:
sched := scheduler.New(queries, 5*time.Minute)
sched.Start()
defer sched.Stop()
```

**Step 3: Verify it compiles**

```bash
cd backend && go build ./cmd/feednest/
# Expected: no errors
```

**Step 4: Commit**

```bash
git add backend/
git commit -m "feat: add background feed scheduler with concurrent fetching"
```

---

### Task 14: Article Scoring Engine

**Files:**
- Create: `backend/internal/scorer/scorer.go`
- Create: `backend/internal/scorer/scorer_test.go`

**Step 1: Write scorer_test.go**

```go
package scorer

import (
	"testing"
	"time"
)

func TestCalculateScore(t *testing.T) {
	now := time.Now()

	// Recent article from high-engagement source
	score1 := CalculateScore(now.Add(-1*time.Hour), 0.8)
	// Old article from low-engagement source
	score2 := CalculateScore(now.Add(-48*time.Hour), 0.1)

	if score1 <= score2 {
		t.Errorf("recent high-engagement article (%.2f) should score higher than old low-engagement (%.2f)", score1, score2)
	}
}

func TestRecencyScore(t *testing.T) {
	now := time.Now()
	score1h := recencyScore(now.Add(-1 * time.Hour))
	score24h := recencyScore(now.Add(-24 * time.Hour))
	score72h := recencyScore(now.Add(-72 * time.Hour))

	if score1h <= score24h || score24h <= score72h {
		t.Errorf("recency should decrease over time: 1h=%.2f, 24h=%.2f, 72h=%.2f", score1h, score24h, score72h)
	}
}
```

**Step 2: Run test to verify it fails**

```bash
cd backend && go test ./internal/scorer/ -v
# Expected: FAIL
```

**Step 3: Write scorer.go**

```go
package scorer

import (
	"math"
	"time"
)

const (
	recencyWeight    = 0.6
	engagementWeight = 0.4
	decayHalfLife    = 24.0 // hours
)

func CalculateScore(publishedAt time.Time, feedEngagement float64) float64 {
	recency := recencyScore(publishedAt)
	return (recencyWeight * recency) + (engagementWeight * feedEngagement)
}

func recencyScore(publishedAt time.Time) float64 {
	hoursAgo := time.Since(publishedAt).Hours()
	if hoursAgo < 0 {
		hoursAgo = 0
	}
	return math.Exp(-0.693 * hoursAgo / decayHalfLife) // 0.693 = ln(2)
}
```

**Step 4: Run tests**

```bash
cd backend && go test ./internal/scorer/ -v
# Expected: PASS
```

**Step 5: Commit**

```bash
git add backend/internal/scorer/
git commit -m "feat: add article scoring engine with recency decay and engagement"
```

---

### Task 15: OPML Import/Export

**Files:**
- Create: `backend/internal/api/handlers/opml.go`
- Create: `backend/internal/api/handlers/opml_test.go`

**Step 1: Write opml_test.go**

```go
package handlers

import (
	"strings"
	"testing"
)

func TestParseOPML(t *testing.T) {
	opml := `<?xml version="1.0" encoding="UTF-8"?>
<opml version="2.0">
  <body>
    <outline text="Tech" title="Tech">
      <outline type="rss" text="Ars Technica" xmlUrl="https://feeds.arstechnica.com/arstechnica/features" htmlUrl="https://arstechnica.com"/>
      <outline type="rss" text="Hacker News" xmlUrl="https://news.ycombinator.com/rss"/>
    </outline>
    <outline type="rss" text="Uncategorized Feed" xmlUrl="https://example.com/rss"/>
  </body>
</opml>`

	feeds, err := parseOPML(strings.NewReader(opml))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(feeds) != 3 {
		t.Fatalf("expected 3 feeds, got %d", len(feeds))
	}
	if feeds[0].Category != "Tech" {
		t.Errorf("expected category 'Tech', got %q", feeds[0].Category)
	}
	if feeds[2].Category != "" {
		t.Errorf("expected empty category for uncategorized feed, got %q", feeds[2].Category)
	}
}

func TestGenerateOPML(t *testing.T) {
	feeds := []opmlFeed{
		{Title: "Ars", XMLURL: "https://feeds.ars.com/rss", HTMLURL: "https://ars.com", Category: "Tech"},
		{Title: "BBC", XMLURL: "https://bbc.com/rss", HTMLURL: "https://bbc.com", Category: "News"},
	}

	output, err := generateOPML(feeds)
	if err != nil {
		t.Fatalf("generate failed: %v", err)
	}
	if !strings.Contains(output, "Ars") {
		t.Error("output should contain feed title")
	}
	if !strings.Contains(output, "<outline text=\"Tech\"") {
		t.Error("output should contain category outline")
	}
}
```

**Step 2: Run test to verify it fails**

```bash
cd backend && go test ./internal/api/handlers/ -v -run TestOPML
# Expected: FAIL
```

**Step 3: Write opml.go**

```go
package handlers

import (
	"encoding/xml"
	"fmt"
	"io"
	"net/http"

	"github.com/feednest/backend/internal/api"
	"github.com/feednest/backend/internal/store"
)

type OPMLHandler struct {
	store *store.Queries
}

func NewOPMLHandler(store *store.Queries) *OPMLHandler {
	return &OPMLHandler{store: store}
}

type opmlFeed struct {
	Title    string
	XMLURL   string
	HTMLURL  string
	Category string
}

type opmlDocument struct {
	XMLName xml.Name    `xml:"opml"`
	Version string      `xml:"version,attr"`
	Body    opmlBody    `xml:"body"`
}

type opmlBody struct {
	Outlines []opmlOutline `xml:"outline"`
}

type opmlOutline struct {
	Text     string         `xml:"text,attr"`
	Title    string         `xml:"title,attr,omitempty"`
	Type     string         `xml:"type,attr,omitempty"`
	XMLURL   string         `xml:"xmlUrl,attr,omitempty"`
	HTMLURL  string         `xml:"htmlUrl,attr,omitempty"`
	Outlines []opmlOutline  `xml:"outline,omitempty"`
}

func parseOPML(r io.Reader) ([]opmlFeed, error) {
	var doc opmlDocument
	if err := xml.NewDecoder(r).Decode(&doc); err != nil {
		return nil, err
	}

	var feeds []opmlFeed
	for _, outline := range doc.Body.Outlines {
		if outline.XMLURL != "" {
			feeds = append(feeds, opmlFeed{
				Title:  outline.Text,
				XMLURL: outline.XMLURL,
				HTMLURL: outline.HTMLURL,
			})
		} else {
			// Category with child feeds
			for _, child := range outline.Outlines {
				if child.XMLURL != "" {
					feeds = append(feeds, opmlFeed{
						Title:    child.Text,
						XMLURL:   child.XMLURL,
						HTMLURL:  child.HTMLURL,
						Category: outline.Text,
					})
				}
			}
		}
	}
	return feeds, nil
}

func generateOPML(feeds []opmlFeed) (string, error) {
	categories := make(map[string][]opmlOutline)
	var uncategorized []opmlOutline

	for _, f := range feeds {
		outline := opmlOutline{
			Text:   f.Title,
			Title:  f.Title,
			Type:   "rss",
			XMLURL: f.XMLURL,
			HTMLURL: f.HTMLURL,
		}
		if f.Category != "" {
			categories[f.Category] = append(categories[f.Category], outline)
		} else {
			uncategorized = append(uncategorized, outline)
		}
	}

	var body []opmlOutline
	for cat, outlines := range categories {
		body = append(body, opmlOutline{
			Text:     cat,
			Title:    cat,
			Outlines: outlines,
		})
	}
	body = append(body, uncategorized...)

	doc := opmlDocument{
		Version: "2.0",
		Body:    opmlBody{Outlines: body},
	}

	data, err := xml.MarshalIndent(doc, "", "  ")
	if err != nil {
		return "", err
	}
	return xml.Header + string(data), nil
}

func (h *OPMLHandler) Import(w http.ResponseWriter, r *http.Request) {
	userID := api.ExtractUserID(r)

	file, _, err := r.FormFile("file")
	if err != nil {
		http.Error(w, `{"error":"file upload required"}`, http.StatusBadRequest)
		return
	}
	defer file.Close()

	feeds, err := parseOPML(file)
	if err != nil {
		http.Error(w, `{"error":"invalid OPML file"}`, http.StatusBadRequest)
		return
	}

	imported := 0
	for _, f := range feeds {
		var categoryID *int64
		if f.Category != "" {
			cat, err := h.store.CreateCategory(userID, f.Category, 0)
			if err != nil {
				// Category might already exist, try to find it
				cats, _ := h.store.ListCategories(userID)
				for _, c := range cats {
					if c.Name == f.Category {
						categoryID = &c.ID
						break
					}
				}
			} else {
				categoryID = &cat.ID
			}
		}

		_, err := h.store.CreateFeed(userID, f.XMLURL, f.Title, f.HTMLURL, "", categoryID)
		if err == nil {
			imported++
		}
	}

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"imported":%d,"total":%d}`, imported, len(feeds))
}

func (h *OPMLHandler) Export(w http.ResponseWriter, r *http.Request) {
	userID := api.ExtractUserID(r)

	feedList, err := h.store.ListFeeds(userID)
	if err != nil {
		http.Error(w, `{"error":"failed to list feeds"}`, http.StatusInternalServerError)
		return
	}

	cats, _ := h.store.ListCategories(userID)
	catMap := make(map[int64]string)
	for _, c := range cats {
		catMap[c.ID] = c.Name
	}

	var opmlFeeds []opmlFeed
	for _, f := range feedList {
		of := opmlFeed{
			Title:  f.Title,
			XMLURL: f.URL,
			HTMLURL: f.SiteURL,
		}
		if f.CategoryID != nil {
			of.Category = catMap[*f.CategoryID]
		}
		opmlFeeds = append(opmlFeeds, of)
	}

	output, err := generateOPML(opmlFeeds)
	if err != nil {
		http.Error(w, `{"error":"failed to generate OPML"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/xml")
	w.Header().Set("Content-Disposition", "attachment; filename=feednest-export.opml")
	w.Write([]byte(output))
}
```

**Step 4: Wire OPML routes in router.go** (inside protected group):

```go
opmlH := handlers.NewOPMLHandler(queries)
r.Post("/api/opml/import", opmlH.Import)
r.Get("/api/opml/export", opmlH.Export)
```

**Step 5: Run tests**

```bash
cd backend && go test ./... -v
# Expected: all PASS
```

**Step 6: Commit**

```bash
git add backend/
git commit -m "feat: add OPML import/export with category support"
```

---

## Phase 4: SvelteKit Frontend

### Task 16: Initialize SvelteKit Project

**Step 1: Create SvelteKit project**

```bash
cd /mnt/d/git/feednest
npx sv create frontend --template minimal --types ts
cd frontend
npm install
```

**Step 2: Install Tailwind CSS**

```bash
cd /mnt/d/git/feednest/frontend
npx sv add tailwindcss
npm install
```

**Step 3: Install additional dependencies**

```bash
npm install clsx
```

**Step 4: Verify dev server starts**

```bash
cd /mnt/d/git/feednest/frontend && npm run dev
# Expected: Server running on http://localhost:5173
```

**Step 5: Commit**

```bash
git add frontend/
git commit -m "feat: initialize SvelteKit frontend with Tailwind CSS"
```

---

### Task 17: API Client and Auth Store

**Files:**
- Create: `frontend/src/lib/api/client.ts`
- Create: `frontend/src/lib/stores/auth.ts`

**Step 1: Write API client**

```typescript
// frontend/src/lib/api/client.ts

const API_BASE = import.meta.env.VITE_API_URL || 'http://localhost:8080';

let accessToken: string | null = null;

export function setAccessToken(token: string | null) {
	accessToken = token;
}

export function getAccessToken(): string | null {
	return accessToken;
}

async function request<T>(method: string, path: string, body?: unknown): Promise<T> {
	const headers: Record<string, string> = {
		'Content-Type': 'application/json',
	};

	if (accessToken) {
		headers['Authorization'] = `Bearer ${accessToken}`;
	}

	const res = await fetch(`${API_BASE}${path}`, {
		method,
		headers,
		body: body ? JSON.stringify(body) : undefined,
	});

	if (res.status === 401 && accessToken) {
		// Try refresh
		const refreshed = await refreshToken();
		if (refreshed) {
			headers['Authorization'] = `Bearer ${accessToken}`;
			const retry = await fetch(`${API_BASE}${path}`, {
				method,
				headers,
				body: body ? JSON.stringify(body) : undefined,
			});
			if (!retry.ok) throw new Error(`API error: ${retry.status}`);
			if (retry.status === 204) return undefined as T;
			return retry.json();
		}
	}

	if (!res.ok) {
		const err = await res.json().catch(() => ({ error: `HTTP ${res.status}` }));
		throw new Error(err.error || `API error: ${res.status}`);
	}

	if (res.status === 204) return undefined as T;
	return res.json();
}

async function refreshToken(): Promise<boolean> {
	const refreshTok = localStorage.getItem('feednest_refresh_token');
	if (!refreshTok) return false;

	try {
		const res = await fetch(`${API_BASE}/api/auth/refresh`, {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify({ refresh_token: refreshTok }),
		});
		if (!res.ok) return false;
		const data = await res.json();
		accessToken = data.access_token;
		return true;
	} catch {
		return false;
	}
}

export const api = {
	get: <T>(path: string) => request<T>('GET', path),
	post: <T>(path: string, body?: unknown) => request<T>('POST', path, body),
	put: <T>(path: string, body?: unknown) => request<T>('PUT', path, body),
	del: <T>(path: string) => request<T>('DELETE', path),
};
```

**Step 2: Write auth store**

```typescript
// frontend/src/lib/stores/auth.ts

import { writable } from 'svelte/store';
import { api, setAccessToken } from '$lib/api/client';

interface User {
	id: number;
	username: string;
	email: string;
}

interface AuthState {
	user: User | null;
	isAuthenticated: boolean;
	loading: boolean;
}

function createAuthStore() {
	const { subscribe, set, update } = writable<AuthState>({
		user: null,
		isAuthenticated: false,
		loading: true,
	});

	return {
		subscribe,

		async login(username: string, password: string) {
			const data = await api.post<{ access_token: string; refresh_token: string; user: User }>(
				'/api/auth/login',
				{ username, password }
			);
			setAccessToken(data.access_token);
			localStorage.setItem('feednest_refresh_token', data.refresh_token);
			set({ user: data.user, isAuthenticated: true, loading: false });
		},

		async register(username: string, email: string, password: string) {
			const data = await api.post<{ access_token: string; refresh_token: string; user: User }>(
				'/api/auth/register',
				{ username, email, password }
			);
			setAccessToken(data.access_token);
			localStorage.setItem('feednest_refresh_token', data.refresh_token);
			set({ user: data.user, isAuthenticated: true, loading: false });
		},

		logout() {
			setAccessToken(null);
			localStorage.removeItem('feednest_refresh_token');
			set({ user: null, isAuthenticated: false, loading: false });
		},

		async checkAuth() {
			const refreshTok = localStorage.getItem('feednest_refresh_token');
			if (!refreshTok) {
				set({ user: null, isAuthenticated: false, loading: false });
				return;
			}

			try {
				const data = await api.post<{ access_token: string }>('/api/auth/refresh', {
					refresh_token: refreshTok,
				});
				setAccessToken(data.access_token);
				// We don't have a /me endpoint yet, so mark as authenticated
				set({ user: null, isAuthenticated: true, loading: false });
			} catch {
				set({ user: null, isAuthenticated: false, loading: false });
			}
		},

		async getUserCount(): Promise<number> {
			const data = await api.get<{ count: number }>('/api/auth/user-count');
			return data.count;
		},
	};
}

export const auth = createAuthStore();
```

**Step 3: Commit**

```bash
git add frontend/src/lib/
git commit -m "feat: add API client with JWT refresh and auth store"
```

---

### Task 18: Auth Pages (Login + Register)

**Files:**
- Create: `frontend/src/routes/auth/login/+page.svelte`
- Create: `frontend/src/routes/auth/register/+page.svelte`
- Modify: `frontend/src/routes/+layout.svelte`

**Step 1: Write login page**

```svelte
<!-- frontend/src/routes/auth/login/+page.svelte -->
<script lang="ts">
	import { goto } from '$app/navigation';
	import { auth } from '$lib/stores/auth';

	let username = '';
	let password = '';
	let error = '';
	let loading = false;

	async function handleLogin() {
		error = '';
		loading = true;
		try {
			await auth.login(username, password);
			goto('/');
		} catch (e) {
			error = e instanceof Error ? e.message : 'Login failed';
		} finally {
			loading = false;
		}
	}
</script>

<div class="min-h-screen flex items-center justify-center bg-gray-50 dark:bg-gray-900">
	<div class="w-full max-w-md p-8 bg-white dark:bg-gray-800 rounded-xl shadow-lg">
		<h1 class="text-2xl font-bold text-center text-gray-900 dark:text-white mb-8">
			Sign in to FeedNest
		</h1>

		{#if error}
			<div class="mb-4 p-3 bg-red-50 dark:bg-red-900/30 text-red-600 dark:text-red-400 rounded-lg text-sm">
				{error}
			</div>
		{/if}

		<form on:submit|preventDefault={handleLogin} class="space-y-4">
			<div>
				<label for="username" class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Username</label>
				<input
					id="username"
					type="text"
					bind:value={username}
					required
					class="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-2 focus:ring-blue-500 focus:border-transparent"
				/>
			</div>

			<div>
				<label for="password" class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Password</label>
				<input
					id="password"
					type="password"
					bind:value={password}
					required
					class="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-2 focus:ring-blue-500 focus:border-transparent"
				/>
			</div>

			<button
				type="submit"
				disabled={loading}
				class="w-full py-2 px-4 bg-blue-600 hover:bg-blue-700 text-white font-medium rounded-lg disabled:opacity-50 transition-colors"
			>
				{loading ? 'Signing in...' : 'Sign In'}
			</button>
		</form>

		<p class="mt-6 text-center text-sm text-gray-500 dark:text-gray-400">
			Don't have an account? <a href="/auth/register" class="text-blue-600 hover:underline">Register</a>
		</p>
	</div>
</div>
```

**Step 2: Write register page**

```svelte
<!-- frontend/src/routes/auth/register/+page.svelte -->
<script lang="ts">
	import { goto } from '$app/navigation';
	import { auth } from '$lib/stores/auth';

	let username = '';
	let email = '';
	let password = '';
	let confirmPassword = '';
	let error = '';
	let loading = false;

	async function handleRegister() {
		error = '';
		if (password !== confirmPassword) {
			error = 'Passwords do not match';
			return;
		}
		if (password.length < 8) {
			error = 'Password must be at least 8 characters';
			return;
		}

		loading = true;
		try {
			await auth.register(username, email, password);
			goto('/');
		} catch (e) {
			error = e instanceof Error ? e.message : 'Registration failed';
		} finally {
			loading = false;
		}
	}
</script>

<div class="min-h-screen flex items-center justify-center bg-gray-50 dark:bg-gray-900">
	<div class="w-full max-w-md p-8 bg-white dark:bg-gray-800 rounded-xl shadow-lg">
		<h1 class="text-2xl font-bold text-center text-gray-900 dark:text-white mb-8">
			Create your FeedNest account
		</h1>

		{#if error}
			<div class="mb-4 p-3 bg-red-50 dark:bg-red-900/30 text-red-600 dark:text-red-400 rounded-lg text-sm">
				{error}
			</div>
		{/if}

		<form on:submit|preventDefault={handleRegister} class="space-y-4">
			<div>
				<label for="username" class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Username</label>
				<input id="username" type="text" bind:value={username} required
					class="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-2 focus:ring-blue-500 focus:border-transparent" />
			</div>

			<div>
				<label for="email" class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Email</label>
				<input id="email" type="email" bind:value={email} required
					class="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-2 focus:ring-blue-500 focus:border-transparent" />
			</div>

			<div>
				<label for="password" class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Password</label>
				<input id="password" type="password" bind:value={password} required minlength="8"
					class="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-2 focus:ring-blue-500 focus:border-transparent" />
			</div>

			<div>
				<label for="confirm" class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Confirm Password</label>
				<input id="confirm" type="password" bind:value={confirmPassword} required
					class="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-2 focus:ring-blue-500 focus:border-transparent" />
			</div>

			<button type="submit" disabled={loading}
				class="w-full py-2 px-4 bg-blue-600 hover:bg-blue-700 text-white font-medium rounded-lg disabled:opacity-50 transition-colors">
				{loading ? 'Creating account...' : 'Create Account'}
			</button>
		</form>

		<p class="mt-6 text-center text-sm text-gray-500 dark:text-gray-400">
			Already have an account? <a href="/auth/login" class="text-blue-600 hover:underline">Sign in</a>
		</p>
	</div>
</div>
```

**Step 3: Write root layout with auth guard**

```svelte
<!-- frontend/src/routes/+layout.svelte -->
<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import { auth } from '$lib/stores/auth';
	import '../app.css';

	onMount(async () => {
		await auth.checkAuth();
	});

	$: {
		if (!$auth.loading) {
			const isAuthPage = $page.url.pathname.startsWith('/auth');
			if (!$auth.isAuthenticated && !isAuthPage) {
				goto('/auth/login');
			}
		}
	}
</script>

{#if $auth.loading}
	<div class="min-h-screen flex items-center justify-center bg-gray-50 dark:bg-gray-900">
		<div class="text-gray-500 dark:text-gray-400">Loading...</div>
	</div>
{:else}
	<slot />
{/if}
```

**Step 4: Commit**

```bash
git add frontend/src/
git commit -m "feat: add login and register pages with auth guard"
```

---

### Task 19: Stores for Feeds, Articles, Settings

**Files:**
- Create: `frontend/src/lib/stores/feeds.ts`
- Create: `frontend/src/lib/stores/articles.ts`
- Create: `frontend/src/lib/stores/settings.ts`
- Create: `frontend/src/lib/utils/time.ts`

**Step 1: Write all store files**

```typescript
// frontend/src/lib/stores/feeds.ts
import { writable } from 'svelte/store';
import { api } from '$lib/api/client';

export interface Feed {
	id: number;
	url: string;
	title: string;
	site_url: string;
	icon_url: string;
	category_id: number | null;
	unread_count: number;
}

export interface Category {
	id: number;
	name: string;
	position: number;
}

function createFeedsStore() {
	const { subscribe, set } = writable<Feed[]>([]);

	return {
		subscribe,
		async load() {
			const feeds = await api.get<Feed[]>('/api/feeds');
			set(feeds || []);
		},
		async add(url: string, categoryId?: number) {
			await api.post('/api/feeds', { url, category_id: categoryId });
			await this.load();
		},
		async remove(id: number) {
			await api.del(`/api/feeds/${id}`);
			await this.load();
		},
	};
}

function createCategoriesStore() {
	const { subscribe, set } = writable<Category[]>([]);

	return {
		subscribe,
		async load() {
			const cats = await api.get<Category[]>('/api/categories');
			set(cats || []);
		},
		async add(name: string) {
			await api.post('/api/categories', { name });
			await this.load();
		},
		async remove(id: number) {
			await api.del(`/api/categories/${id}`);
			await this.load();
		},
	};
}

export const feeds = createFeedsStore();
export const categories = createCategoriesStore();
```

```typescript
// frontend/src/lib/stores/articles.ts
import { writable } from 'svelte/store';
import { api } from '$lib/api/client';

export interface Article {
	id: number;
	feed_id: number;
	title: string;
	url: string;
	author: string;
	content_clean: string;
	thumbnail_url: string;
	published_at: string | null;
	word_count: number;
	reading_time: number;
	is_read: boolean;
	is_starred: boolean;
	score: number;
	feed_title: string;
	feed_icon_url: string;
	tags: string[];
}

interface ArticlesResponse {
	articles: Article[];
	total: number;
	page: number;
	limit: number;
}

export interface ArticleFilters {
	status?: string;
	sort?: string;
	feed?: number;
	category?: number;
	tag?: string;
	page?: number;
}

function createArticlesStore() {
	const { subscribe, set, update } = writable<{
		articles: Article[];
		total: number;
		loading: boolean;
	}>({ articles: [], total: 0, loading: false });

	return {
		subscribe,

		async load(filters: ArticleFilters = {}) {
			update((s) => ({ ...s, loading: true }));
			const params = new URLSearchParams();
			if (filters.status) params.set('status', filters.status);
			if (filters.sort) params.set('sort', filters.sort || 'smart');
			if (filters.feed) params.set('feed', String(filters.feed));
			if (filters.category) params.set('category', String(filters.category));
			if (filters.tag) params.set('tag', filters.tag);
			if (filters.page) params.set('page', String(filters.page));

			const data = await api.get<ArticlesResponse>(`/api/articles?${params}`);
			set({ articles: data.articles || [], total: data.total, loading: false });
		},

		async toggleRead(id: number, isRead: boolean) {
			await api.put(`/api/articles/${id}`, { is_read: isRead });
			update((s) => ({
				...s,
				articles: s.articles.map((a) => (a.id === id ? { ...a, is_read: isRead } : a)),
			}));
		},

		async toggleStar(id: number, isStarred: boolean) {
			await api.put(`/api/articles/${id}`, { is_starred: isStarred });
			update((s) => ({
				...s,
				articles: s.articles.map((a) => (a.id === id ? { ...a, is_starred: isStarred } : a)),
			}));
		},

		async dismiss(id: number) {
			await api.post(`/api/articles/${id}/dismiss`);
			update((s) => ({
				...s,
				articles: s.articles.filter((a) => a.id !== id),
				total: s.total - 1,
			}));
		},

		async getArticle(id: number): Promise<Article> {
			return api.get<Article>(`/api/articles/${id}`);
		},
	};
}

export const articles = createArticlesStore();
```

```typescript
// frontend/src/lib/stores/settings.ts
import { writable } from 'svelte/store';
import { api } from '$lib/api/client';

export type ViewMode = 'cards' | 'list';
export type Theme = 'light' | 'dark' | 'system';

interface AppSettings {
	viewMode: ViewMode;
	theme: Theme;
}

function createSettingsStore() {
	const { subscribe, set, update } = writable<AppSettings>({
		viewMode: 'cards',
		theme: 'system',
	});

	return {
		subscribe,

		async load() {
			try {
				const data = await api.get<Record<string, string>>('/api/settings');
				set({
					viewMode: (data.view_mode as ViewMode) || 'cards',
					theme: (data.theme as Theme) || 'system',
				});
			} catch {
				// Use defaults
			}
		},

		async setViewMode(mode: ViewMode) {
			update((s) => ({ ...s, viewMode: mode }));
			await api.put('/api/settings', { view_mode: mode });
		},

		async setTheme(theme: Theme) {
			update((s) => ({ ...s, theme }));
			await api.put('/api/settings', { theme });
			applyTheme(theme);
		},
	};
}

export function applyTheme(theme: Theme) {
	if (theme === 'dark' || (theme === 'system' && window.matchMedia('(prefers-color-scheme: dark)').matches)) {
		document.documentElement.classList.add('dark');
	} else {
		document.documentElement.classList.remove('dark');
	}
}

export const settings = createSettingsStore();
```

```typescript
// frontend/src/lib/utils/time.ts
export function timeAgo(dateStr: string | null): string {
	if (!dateStr) return '';
	const date = new Date(dateStr);
	const now = new Date();
	const seconds = Math.floor((now.getTime() - date.getTime()) / 1000);

	if (seconds < 60) return 'just now';
	if (seconds < 3600) return `${Math.floor(seconds / 60)}m ago`;
	if (seconds < 86400) return `${Math.floor(seconds / 3600)}h ago`;
	if (seconds < 604800) return `${Math.floor(seconds / 86400)}d ago`;

	return date.toLocaleDateString();
}
```

**Step 2: Commit**

```bash
git add frontend/src/lib/
git commit -m "feat: add stores for feeds, articles, and settings with time utils"
```

---

### Task 20: Sidebar Component

**Files:**
- Create: `frontend/src/lib/components/Sidebar.svelte`

**Step 1: Write Sidebar.svelte**

```svelte
<script lang="ts">
	import { feeds, categories, type Feed, type Category } from '$lib/stores/feeds';
	import { auth } from '$lib/stores/auth';
	import { createEventDispatcher } from 'svelte';

	const dispatch = createEventDispatcher<{
		selectAll: void;
		selectStarred: void;
		selectFeed: { id: number };
		selectCategory: { id: number };
		addFeed: void;
	}>();

	export let collapsed = false;
	export let activeFeed: number | null = null;
	export let activeCategory: number | null = null;
	export let activeView: 'all' | 'starred' | 'feed' | 'category' = 'all';

	$: feedsByCategory = groupByCategory($feeds, $categories);

	function groupByCategory(feedList: Feed[], catList: Category[]) {
		const uncategorized: Feed[] = [];
		const grouped: { category: Category; feeds: Feed[] }[] = [];

		const catMap = new Map<number, Feed[]>();
		for (const cat of catList) {
			catMap.set(cat.id, []);
		}

		for (const feed of feedList) {
			if (feed.category_id && catMap.has(feed.category_id)) {
				catMap.get(feed.category_id)!.push(feed);
			} else {
				uncategorized.push(feed);
			}
		}

		for (const cat of catList) {
			grouped.push({ category: cat, feeds: catMap.get(cat.id) || [] });
		}

		return { grouped, uncategorized };
	}

	function totalUnread(feedList: Feed[]): number {
		return feedList.reduce((sum, f) => sum + f.unread_count, 0);
	}
</script>

<aside
	class="h-full bg-white dark:bg-gray-800 border-r border-gray-200 dark:border-gray-700 flex flex-col transition-all"
	class:w-64={!collapsed}
	class:w-0={collapsed}
	class:overflow-hidden={collapsed}
>
	<div class="p-4 border-b border-gray-200 dark:border-gray-700">
		<h1 class="text-xl font-bold text-gray-900 dark:text-white">FeedNest</h1>
	</div>

	<nav class="flex-1 overflow-y-auto p-2 space-y-1">
		<button
			on:click={() => dispatch('selectAll')}
			class="w-full flex items-center justify-between px-3 py-2 rounded-lg text-sm transition-colors"
			class:bg-blue-50={activeView === 'all'}
			class:dark:bg-blue-900/30={activeView === 'all'}
			class:text-blue-700={activeView === 'all'}
			class:dark:text-blue-300={activeView === 'all'}
			class:text-gray-700={activeView !== 'all'}
			class:dark:text-gray-300={activeView !== 'all'}
			class:hover:bg-gray-100={activeView !== 'all'}
			class:dark:hover:bg-gray-700={activeView !== 'all'}
		>
			<span>All Articles</span>
			{#if totalUnread($feeds) > 0}
				<span class="text-xs bg-blue-100 dark:bg-blue-900 text-blue-700 dark:text-blue-300 px-2 py-0.5 rounded-full">
					{totalUnread($feeds)}
				</span>
			{/if}
		</button>

		<button
			on:click={() => dispatch('selectStarred')}
			class="w-full flex items-center px-3 py-2 rounded-lg text-sm transition-colors"
			class:bg-yellow-50={activeView === 'starred'}
			class:dark:bg-yellow-900/30={activeView === 'starred'}
			class:text-yellow-700={activeView === 'starred'}
			class:text-gray-700={activeView !== 'starred'}
			class:dark:text-gray-300={activeView !== 'starred'}
			class:hover:bg-gray-100={activeView !== 'starred'}
			class:dark:hover:bg-gray-700={activeView !== 'starred'}
		>
			Starred
		</button>

		<div class="pt-2">
			{#each feedsByCategory.grouped as { category, feeds: catFeeds }}
				<div class="mb-1">
					<button
						on:click={() => dispatch('selectCategory', { id: category.id })}
						class="w-full flex items-center justify-between px-3 py-1.5 text-xs font-semibold uppercase tracking-wider rounded"
						class:text-blue-700={activeView === 'category' && activeCategory === category.id}
						class:text-gray-500={!(activeView === 'category' && activeCategory === category.id)}
						class:dark:text-gray-400={!(activeView === 'category' && activeCategory === category.id)}
						class:hover:bg-gray-100={true}
						class:dark:hover:bg-gray-700={true}
					>
						{category.name}
						{#if totalUnread(catFeeds) > 0}
							<span class="text-xs font-normal">{totalUnread(catFeeds)}</span>
						{/if}
					</button>

					{#each catFeeds as feed}
						<button
							on:click={() => dispatch('selectFeed', { id: feed.id })}
							class="w-full flex items-center justify-between pl-6 pr-3 py-1.5 text-sm rounded-lg transition-colors"
							class:bg-blue-50={activeView === 'feed' && activeFeed === feed.id}
							class:dark:bg-blue-900/30={activeView === 'feed' && activeFeed === feed.id}
							class:text-gray-600={!(activeView === 'feed' && activeFeed === feed.id)}
							class:dark:text-gray-400={!(activeView === 'feed' && activeFeed === feed.id)}
							class:hover:bg-gray-100={true}
							class:dark:hover:bg-gray-700={true}
						>
							<span class="truncate">{feed.title || feed.url}</span>
							{#if feed.unread_count > 0}
								<span class="text-xs text-gray-400">{feed.unread_count}</span>
							{/if}
						</button>
					{/each}
				</div>
			{/each}

			{#each feedsByCategory.uncategorized as feed}
				<button
					on:click={() => dispatch('selectFeed', { id: feed.id })}
					class="w-full flex items-center justify-between px-3 py-1.5 text-sm rounded-lg transition-colors"
					class:bg-blue-50={activeView === 'feed' && activeFeed === feed.id}
					class:text-gray-600={!(activeView === 'feed' && activeFeed === feed.id)}
					class:dark:text-gray-400={!(activeView === 'feed' && activeFeed === feed.id)}
					class:hover:bg-gray-100={true}
					class:dark:hover:bg-gray-700={true}
				>
					<span class="truncate">{feed.title || feed.url}</span>
					{#if feed.unread_count > 0}
						<span class="text-xs text-gray-400">{feed.unread_count}</span>
					{/if}
				</button>
			{/each}
		</div>
	</nav>

	<div class="p-3 border-t border-gray-200 dark:border-gray-700 space-y-2">
		<button
			on:click={() => dispatch('addFeed')}
			class="w-full flex items-center justify-center gap-2 px-3 py-2 text-sm text-blue-600 dark:text-blue-400 hover:bg-blue-50 dark:hover:bg-blue-900/30 rounded-lg transition-colors"
		>
			+ Add Feed
		</button>
		<button
			on:click={() => auth.logout()}
			class="w-full text-xs text-gray-400 hover:text-gray-600 dark:hover:text-gray-300"
		>
			Sign Out
		</button>
	</div>
</aside>
```

**Step 2: Commit**

```bash
git add frontend/src/lib/components/
git commit -m "feat: add sidebar component with feed/category navigation"
```

---

### Task 21: Article Card and List Components

**Files:**
- Create: `frontend/src/lib/components/ArticleCard.svelte`
- Create: `frontend/src/lib/components/ArticleList.svelte`

**Step 1: Write ArticleCard.svelte**

```svelte
<script lang="ts">
	import type { Article } from '$lib/stores/articles';
	import { articles } from '$lib/stores/articles';
	import { timeAgo } from '$lib/utils/time';

	export let article: Article;

	function handleStar(e: Event) {
		e.stopPropagation();
		articles.toggleStar(article.id, !article.is_starred);
	}
</script>

<a
	href="/article/{article.id}"
	class="block bg-white dark:bg-gray-800 rounded-xl border border-gray-200 dark:border-gray-700 overflow-hidden hover:shadow-lg transition-shadow group"
	class:opacity-60={article.is_read}
>
	{#if article.thumbnail_url}
		<div class="aspect-video bg-gray-100 dark:bg-gray-700 overflow-hidden">
			<img
				src={article.thumbnail_url}
				alt=""
				class="w-full h-full object-cover group-hover:scale-105 transition-transform duration-300"
				loading="lazy"
			/>
		</div>
	{/if}

	<div class="p-4">
		<h3 class="font-semibold text-gray-900 dark:text-white leading-snug line-clamp-2 mb-2">
			{article.title}
		</h3>

		<div class="flex items-center justify-between text-xs text-gray-500 dark:text-gray-400">
			<div class="flex items-center gap-1.5 min-w-0">
				<span class="truncate">{article.feed_title}</span>
				<span>·</span>
				<span class="whitespace-nowrap">{timeAgo(article.published_at)}</span>
				{#if article.reading_time > 0}
					<span>·</span>
					<span class="whitespace-nowrap">{article.reading_time} min</span>
				{/if}
			</div>

			<button
				on:click={handleStar}
				class="ml-2 p-1 hover:text-yellow-500 transition-colors flex-shrink-0"
				class:text-yellow-500={article.is_starred}
			>
				{article.is_starred ? '★' : '☆'}
			</button>
		</div>
	</div>
</a>
```

**Step 2: Write ArticleList.svelte**

```svelte
<script lang="ts">
	import type { Article } from '$lib/stores/articles';
	import { articles } from '$lib/stores/articles';
	import { timeAgo } from '$lib/utils/time';

	export let article: Article;

	function handleStar(e: Event) {
		e.stopPropagation();
		articles.toggleStar(article.id, !article.is_starred);
	}

	function handleMarkRead(e: Event) {
		e.stopPropagation();
		articles.toggleRead(article.id, !article.is_read);
	}
</script>

<a
	href="/article/{article.id}"
	class="flex items-center gap-3 px-4 py-3 border-b border-gray-100 dark:border-gray-700 hover:bg-gray-50 dark:hover:bg-gray-800/50 transition-colors"
	class:opacity-60={article.is_read}
>
	<div class="w-2 h-2 rounded-full flex-shrink-0" class:bg-blue-500={!article.is_read} class:bg-transparent={article.is_read}></div>

	<div class="flex-1 min-w-0">
		<h3 class="text-sm font-medium text-gray-900 dark:text-white truncate" class:font-normal={article.is_read}>
			{article.title}
		</h3>
	</div>

	<span class="text-xs text-gray-500 dark:text-gray-400 whitespace-nowrap flex-shrink-0">
		{article.feed_title}
	</span>

	<span class="text-xs text-gray-400 dark:text-gray-500 whitespace-nowrap flex-shrink-0 w-16 text-right">
		{timeAgo(article.published_at)}
	</span>

	<button
		on:click={handleStar}
		class="p-1 text-gray-400 hover:text-yellow-500 transition-colors flex-shrink-0"
		class:text-yellow-500={article.is_starred}
	>
		{article.is_starred ? '★' : '☆'}
	</button>
</a>
```

**Step 3: Commit**

```bash
git add frontend/src/lib/components/
git commit -m "feat: add ArticleCard and ArticleList components"
```

---

### Task 22: Dashboard Page (Home)

**Files:**
- Modify: `frontend/src/routes/+page.svelte`
- Modify: `frontend/src/routes/+layout.svelte`

**Step 1: Write the dashboard page**

```svelte
<!-- frontend/src/routes/+page.svelte -->
<script lang="ts">
	import { onMount } from 'svelte';
	import { articles, type ArticleFilters } from '$lib/stores/articles';
	import { feeds, categories } from '$lib/stores/feeds';
	import { settings } from '$lib/stores/settings';
	import Sidebar from '$lib/components/Sidebar.svelte';
	import ArticleCard from '$lib/components/ArticleCard.svelte';
	import ArticleListItem from '$lib/components/ArticleList.svelte';

	let sidebarCollapsed = false;
	let activeFeed: number | null = null;
	let activeCategory: number | null = null;
	let activeView: 'all' | 'starred' | 'feed' | 'category' = 'all';
	let sortMode: 'smart' | 'newest' | 'oldest' = 'smart';
	let statusFilter: string = 'unread';
	let showAddFeed = false;
	let newFeedUrl = '';

	onMount(async () => {
		await Promise.all([feeds.load(), categories.load(), settings.load()]);
		await loadArticles();
	});

	async function loadArticles() {
		const filters: ArticleFilters = {
			sort: sortMode,
			status: statusFilter === 'all' ? undefined : statusFilter,
		};
		if (activeView === 'feed' && activeFeed) filters.feed = activeFeed;
		if (activeView === 'category' && activeCategory) filters.category = activeCategory;
		if (activeView === 'starred') filters.status = 'starred';

		await articles.load(filters);
	}

	function selectAll() {
		activeView = 'all';
		activeFeed = null;
		activeCategory = null;
		loadArticles();
	}

	function selectStarred() {
		activeView = 'starred';
		activeFeed = null;
		activeCategory = null;
		loadArticles();
	}

	function selectFeed(e: CustomEvent<{ id: number }>) {
		activeView = 'feed';
		activeFeed = e.detail.id;
		activeCategory = null;
		loadArticles();
	}

	function selectCategory(e: CustomEvent<{ id: number }>) {
		activeView = 'category';
		activeCategory = e.detail.id;
		activeFeed = null;
		loadArticles();
	}

	async function addFeed() {
		if (!newFeedUrl.trim()) return;
		await feeds.add(newFeedUrl.trim());
		newFeedUrl = '';
		showAddFeed = false;
		await loadArticles();
	}

	function toggleView() {
		const mode = $settings.viewMode === 'cards' ? 'list' : 'cards';
		settings.setViewMode(mode);
	}

	$: sortMode, statusFilter, loadArticles();
</script>

<div class="flex h-screen bg-gray-50 dark:bg-gray-900">
	<Sidebar
		bind:collapsed={sidebarCollapsed}
		{activeFeed}
		{activeCategory}
		{activeView}
		on:selectAll={selectAll}
		on:selectStarred={selectStarred}
		on:selectFeed={selectFeed}
		on:selectCategory={selectCategory}
		on:addFeed={() => (showAddFeed = true)}
	/>

	<main class="flex-1 overflow-y-auto">
		<!-- Toolbar -->
		<div class="sticky top-0 z-10 bg-gray-50/95 dark:bg-gray-900/95 backdrop-blur border-b border-gray-200 dark:border-gray-700 px-6 py-3">
			<div class="flex items-center justify-between">
				<div class="flex items-center gap-2">
					<button on:click={() => (sidebarCollapsed = !sidebarCollapsed)} class="p-2 text-gray-500 hover:text-gray-700 dark:hover:text-gray-300 lg:hidden">
						&#9776;
					</button>

					<div class="flex gap-1 bg-gray-200 dark:bg-gray-700 rounded-lg p-0.5">
						{#each ['all', 'unread', 'starred'] as filter}
							<button
								on:click={() => (statusFilter = filter)}
								class="px-3 py-1 text-sm rounded-md capitalize transition-colors"
								class:bg-white={statusFilter === filter}
								class:dark:bg-gray-600={statusFilter === filter}
								class:shadow-sm={statusFilter === filter}
								class:text-gray-500={statusFilter !== filter}
								class:dark:text-gray-400={statusFilter !== filter}
							>
								{filter}
							</button>
						{/each}
					</div>
				</div>

				<div class="flex items-center gap-2">
					<select
						bind:value={sortMode}
						class="text-sm bg-transparent border border-gray-300 dark:border-gray-600 rounded-lg px-2 py-1 text-gray-700 dark:text-gray-300"
					>
						<option value="smart">Smart</option>
						<option value="newest">Newest</option>
						<option value="oldest">Oldest</option>
					</select>

					<button
						on:click={toggleView}
						class="p-2 text-gray-500 hover:text-gray-700 dark:hover:text-gray-300"
						title="Toggle view"
					>
						{$settings.viewMode === 'cards' ? '☰' : '▦'}
					</button>
				</div>
			</div>
		</div>

		<!-- Content -->
		<div class="p-6">
			{#if $articles.loading}
				<div class="text-center text-gray-500 dark:text-gray-400 py-12">Loading articles...</div>
			{:else if $articles.articles.length === 0}
				<div class="text-center py-12">
					<p class="text-gray-500 dark:text-gray-400 mb-4">No articles yet</p>
					<button
						on:click={() => (showAddFeed = true)}
						class="text-blue-600 hover:underline"
					>
						Add your first feed
					</button>
				</div>
			{:else if $settings.viewMode === 'cards'}
				<div class="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-4">
					{#each $articles.articles as article (article.id)}
						<ArticleCard {article} />
					{/each}
				</div>
			{:else}
				<div class="bg-white dark:bg-gray-800 rounded-xl border border-gray-200 dark:border-gray-700 overflow-hidden">
					{#each $articles.articles as article (article.id)}
						<ArticleListItem {article} />
					{/each}
				</div>
			{/if}
		</div>
	</main>
</div>

<!-- Add Feed Modal -->
{#if showAddFeed}
	<div class="fixed inset-0 bg-black/50 flex items-center justify-center z-50" on:click={() => (showAddFeed = false)} on:keydown={() => {}}>
		<div class="bg-white dark:bg-gray-800 rounded-xl p-6 w-full max-w-md shadow-xl" on:click|stopPropagation on:keydown={() => {}}>
			<h2 class="text-lg font-semibold text-gray-900 dark:text-white mb-4">Add Feed</h2>
			<form on:submit|preventDefault={addFeed}>
				<input
					type="url"
					bind:value={newFeedUrl}
					placeholder="https://example.com/rss"
					required
					class="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white mb-4 focus:ring-2 focus:ring-blue-500"
				/>
				<div class="flex justify-end gap-2">
					<button type="button" on:click={() => (showAddFeed = false)} class="px-4 py-2 text-gray-500 hover:text-gray-700">Cancel</button>
					<button type="submit" class="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700">Add</button>
				</div>
			</form>
		</div>
	</div>
{/if}
```

**Step 2: Commit**

```bash
git add frontend/src/routes/
git commit -m "feat: add dashboard page with card/list views, filters, and add feed modal"
```

---

### Task 23: Article Reader Page

**Files:**
- Create: `frontend/src/routes/article/[id]/+page.svelte`

**Step 1: Write the article reader page**

```svelte
<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import { articles, type Article } from '$lib/stores/articles';
	import { timeAgo } from '$lib/utils/time';
	import { api } from '$lib/api/client';

	let article: Article | null = null;
	let loading = true;
	let error = '';

	onMount(async () => {
		const id = Number($page.params.id);
		try {
			article = await articles.getArticle(id);
			// Mark as read
			if (article && !article.is_read) {
				await articles.toggleRead(article.id, true);
				article.is_read = true;
			}
			// Track reading event
			api.post('/api/events', { article_id: id, event_type: 'click', duration_seconds: 0 });
		} catch (e) {
			error = 'Article not found';
		} finally {
			loading = false;
		}
	});

	// Track time spent reading
	let startTime = Date.now();
	function trackReadTime() {
		if (article) {
			const duration = Math.floor((Date.now() - startTime) / 1000);
			if (duration > 5) {
				api.post('/api/events', {
					article_id: article.id,
					event_type: 'read',
					duration_seconds: duration,
				});
			}
		}
	}

	function handleStar() {
		if (article) {
			article.is_starred = !article.is_starred;
			articles.toggleStar(article.id, article.is_starred);
		}
	}
</script>

<svelte:window on:beforeunload={trackReadTime} />

<div class="min-h-screen bg-white dark:bg-gray-900">
	<!-- Header -->
	<header class="sticky top-0 z-10 bg-white/95 dark:bg-gray-900/95 backdrop-blur border-b border-gray-200 dark:border-gray-700">
		<div class="max-w-4xl mx-auto px-6 py-3 flex items-center justify-between">
			<button
				on:click={() => { trackReadTime(); goto('/'); }}
				class="flex items-center gap-2 text-gray-500 hover:text-gray-700 dark:hover:text-gray-300 transition-colors"
			>
				&larr; Back
			</button>

			{#if article}
				<div class="flex items-center gap-3">
					<button
						on:click={handleStar}
						class="p-2 hover:text-yellow-500 transition-colors"
						class:text-yellow-500={article.is_starred}
						class:text-gray-400={!article.is_starred}
					>
						{article.is_starred ? '★' : '☆'} Star
					</button>

					{#if article.url}
						<a
							href={article.url}
							target="_blank"
							rel="noopener noreferrer"
							class="p-2 text-gray-500 hover:text-gray-700 dark:hover:text-gray-300 transition-colors"
						>
							Open Original ↗
						</a>
					{/if}
				</div>
			{/if}
		</div>
	</header>

	<!-- Content -->
	<main class="max-w-[680px] mx-auto px-6 py-12">
		{#if loading}
			<div class="text-center text-gray-500 dark:text-gray-400">Loading...</div>
		{:else if error}
			<div class="text-center text-red-500">{error}</div>
		{:else if article}
			<article>
				<h1 class="text-3xl md:text-4xl font-bold text-gray-900 dark:text-white leading-tight mb-4">
					{article.title}
				</h1>

				<div class="flex items-center gap-2 text-sm text-gray-500 dark:text-gray-400 mb-8">
					<span class="font-medium">{article.feed_title}</span>
					{#if article.author}
						<span>·</span>
						<span>{article.author}</span>
					{/if}
					{#if article.published_at}
						<span>·</span>
						<span>{new Date(article.published_at).toLocaleDateString('en-US', { year: 'numeric', month: 'long', day: 'numeric' })}</span>
					{/if}
					{#if article.reading_time > 0}
						<span>·</span>
						<span>{article.reading_time} min read</span>
					{/if}
				</div>

				<div
					class="prose prose-lg dark:prose-invert max-w-none
						prose-headings:text-gray-900 dark:prose-headings:text-white
						prose-p:text-gray-700 dark:prose-p:text-gray-300
						prose-p:leading-[1.7]
						prose-a:text-blue-600 dark:prose-a:text-blue-400
						prose-img:rounded-xl prose-img:shadow-md
						prose-code:bg-gray-100 dark:prose-code:bg-gray-800
						prose-pre:bg-gray-900 dark:prose-pre:bg-gray-950"
				>
					{@html article.content_clean || article.content_raw || '<p class="text-gray-400">No content available. <a href="' + article.url + '" target="_blank">Read on original site →</a></p>'}
				</div>
			</article>
		{/if}
	</main>
</div>
```

**Step 2: Install Tailwind Typography plugin**

```bash
cd /mnt/d/git/feednest/frontend && npm install @tailwindcss/typography
```

Add to `tailwind.config.ts`:
```typescript
plugins: [require('@tailwindcss/typography')],
```

**Step 3: Commit**

```bash
git add frontend/
git commit -m "feat: add article reader page with typography, reading time tracking"
```

---

### Task 24: Keyboard Shortcuts

**Files:**
- Create: `frontend/src/lib/utils/keyboard.ts`
- Modify: `frontend/src/routes/+page.svelte` (add keyboard handler)

**Step 1: Write keyboard.ts**

```typescript
export interface KeyboardShortcuts {
	[key: string]: () => void;
}

export function setupKeyboardShortcuts(shortcuts: KeyboardShortcuts) {
	function handler(e: KeyboardEvent) {
		// Don't trigger if user is typing in an input
		const target = e.target as HTMLElement;
		if (target.tagName === 'INPUT' || target.tagName === 'TEXTAREA' || target.isContentEditable) {
			return;
		}

		const key = e.key.toLowerCase();
		if (shortcuts[key]) {
			e.preventDefault();
			shortcuts[key]();
		}
	}

	window.addEventListener('keydown', handler);
	return () => window.removeEventListener('keydown', handler);
}
```

**Step 2: Add keyboard handling to dashboard page** -- add to `+page.svelte`:

In the script section, add:
```typescript
import { onDestroy } from 'svelte';
import { setupKeyboardShortcuts } from '$lib/utils/keyboard';

let selectedIndex = -1;
let cleanup: (() => void) | null = null;

onMount(async () => {
	// ... existing code ...

	cleanup = setupKeyboardShortcuts({
		j: () => {
			if (selectedIndex < $articles.articles.length - 1) selectedIndex++;
		},
		k: () => {
			if (selectedIndex > 0) selectedIndex--;
		},
		enter: () => {
			const article = $articles.articles[selectedIndex];
			if (article) goto(`/article/${article.id}`);
		},
		s: () => {
			const article = $articles.articles[selectedIndex];
			if (article) articles.toggleStar(article.id, !article.is_starred);
		},
		m: () => {
			const article = $articles.articles[selectedIndex];
			if (article) articles.toggleRead(article.id, !article.is_read);
		},
		d: () => {
			const article = $articles.articles[selectedIndex];
			if (article) articles.dismiss(article.id);
		},
		v: toggleView,
	});
});

onDestroy(() => cleanup?.());
```

**Step 3: Commit**

```bash
git add frontend/src/
git commit -m "feat: add keyboard shortcuts (j/k/enter/s/m/d/v)"
```

---

## Phase 5: Docker & Polish

### Task 25: Backend Dockerfile

**Files:**
- Create: `backend/Dockerfile`

**Step 1: Write Dockerfile**

```dockerfile
FROM golang:1.22-alpine AS builder

RUN apk add --no-cache gcc musl-dev

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=1 go build -o feednest ./cmd/feednest/

FROM alpine:3.19
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=builder /app/feednest .

EXPOSE 8080
ENV DB_PATH=/data/feednest.db

CMD ["./feednest"]
```

**Step 2: Commit**

```bash
git add backend/Dockerfile
git commit -m "feat: add backend Dockerfile with multi-stage build"
```

---

### Task 26: Frontend Dockerfile

**Files:**
- Create: `frontend/Dockerfile`

**Step 1: Write Dockerfile**

```dockerfile
FROM node:20-alpine AS builder

WORKDIR /app
COPY package*.json ./
RUN npm ci
COPY . .
RUN npm run build

FROM node:20-alpine
WORKDIR /app
COPY --from=builder /app/build ./build
COPY --from=builder /app/package.json .
COPY --from=builder /app/node_modules ./node_modules

EXPOSE 3000
ENV PORT=3000

CMD ["node", "build"]
```

Note: Requires SvelteKit adapter-node. Install it:

```bash
cd /mnt/d/git/feednest/frontend
npm install @sveltejs/adapter-node
```

Update `svelte.config.js` to use adapter-node:
```javascript
import adapter from '@sveltejs/adapter-node';

export default {
	kit: {
		adapter: adapter()
	}
};
```

**Step 2: Commit**

```bash
git add frontend/
git commit -m "feat: add frontend Dockerfile with adapter-node"
```

---

### Task 27: Docker Compose

**Files:**
- Create: `docker-compose.yml`

**Step 1: Write docker-compose.yml**

```yaml
services:
  backend:
    build: ./backend
    ports:
      - "8080:8080"
    volumes:
      - feednest-data:/data
    environment:
      - DB_PATH=/data/feednest.db
      - JWT_SECRET=${JWT_SECRET:-change-me-in-production}
      - PORT=8080
    restart: unless-stopped

  frontend:
    build: ./frontend
    ports:
      - "3000:3000"
    environment:
      - VITE_API_URL=http://backend:8080
      - PORT=3000
    depends_on:
      - backend
    restart: unless-stopped

volumes:
  feednest-data:
```

**Step 2: Create .env.example**

```
JWT_SECRET=your-secret-key-here
```

**Step 3: Create .gitignore** (root level, if not exists)

```
.env
*.db
node_modules/
```

**Step 4: Commit**

```bash
git add docker-compose.yml .env.example .gitignore
git commit -m "feat: add Docker Compose config for full stack deployment"
```

---

### Task 28: Dark/Light Theme Toggle

**Files:**
- Create: `frontend/src/lib/components/ThemeToggle.svelte`
- Modify: dashboard to include theme toggle in toolbar

**Step 1: Write ThemeToggle.svelte**

```svelte
<script lang="ts">
	import { settings, type Theme } from '$lib/stores/settings';

	const themes: { value: Theme; label: string }[] = [
		{ value: 'light', label: '☀' },
		{ value: 'dark', label: '☾' },
		{ value: 'system', label: '⚙' },
	];
</script>

<div class="flex items-center gap-1 bg-gray-200 dark:bg-gray-700 rounded-lg p-0.5">
	{#each themes as theme}
		<button
			on:click={() => settings.setTheme(theme.value)}
			class="px-2 py-1 text-sm rounded-md transition-colors"
			class:bg-white={$settings.theme === theme.value}
			class:dark:bg-gray-600={$settings.theme === theme.value}
			class:shadow-sm={$settings.theme === theme.value}
			title={theme.value}
		>
			{theme.label}
		</button>
	{/each}
</div>
```

**Step 2: Add `darkMode: 'class'` to tailwind.config.ts** (if not already set)

**Step 3: Commit**

```bash
git add frontend/src/lib/components/ThemeToggle.svelte
git commit -m "feat: add dark/light/system theme toggle component"
```

---

### Task 29: Update README

**Files:**
- Modify: `README.md`

**Step 1: Write README**

```markdown
# FeedNest

A modern, self-hosted RSS feed reader. Beautiful card-based UI, smart prioritization, and clean article reading experience.

## Features

- Card & list views for fast headline scanning
- Clean, distraction-free article reader with great typography
- Smart article prioritization based on your reading behavior
- Categories and tags for organization
- Multi-user support with JWT auth
- OPML import/export for easy migration
- Keyboard shortcuts for power users
- Dark/light theme with system preference detection
- Fully self-hosted via Docker

## Quick Start

```bash
git clone https://github.com/yourusername/feednest.git
cd feednest
cp .env.example .env
# Edit .env and set JWT_SECRET
docker compose up -d
```

Open http://localhost:3000 and create your account.

## Development

### Backend (Go)

```bash
cd backend
go run ./cmd/feednest/
```

### Frontend (SvelteKit)

```bash
cd frontend
npm install
npm run dev
```

## Tech Stack

- **Frontend**: SvelteKit, TypeScript, Tailwind CSS
- **Backend**: Go, Chi router, SQLite
- **Deployment**: Docker Compose

## License

MIT
```

**Step 2: Commit**

```bash
git add README.md
git commit -m "docs: update README with features, quick start, and dev setup"
```

---

## Summary

| Phase | Tasks | Description |
|-------|-------|-------------|
| 1 | 1-6 | Go backend foundation: project setup, router, SQLite, models, user store, JWT auth |
| 2 | 7-10 | Core backend: categories, feeds, articles, tags, events, settings CRUD |
| 3 | 11-15 | Feed engine: RSS fetcher, readability, scheduler, scoring, OPML |
| 4 | 16-24 | SvelteKit frontend: scaffolding, auth, stores, components, pages, keyboard shortcuts |
| 5 | 25-29 | Docker & polish: Dockerfiles, compose, theme toggle, README |

Total: **29 tasks** across **5 phases**

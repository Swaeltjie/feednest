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

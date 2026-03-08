package store

import (
	"strings"

	"github.com/feednest/backend/internal/models"
)

func (q *Queries) CreateCategory(userID int64, name string, position int) (*models.Category, error) {
	result, err := q.db.Exec(
		"INSERT INTO categories (user_id, name, position) VALUES (?, ?, ?)",
		userID, name, position,
	)
	if err != nil {
		return nil, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}
	return &models.Category{ID: id, UserID: userID, Name: name, Position: position}, nil
}

func (q *Queries) GetCategory(id, userID int64) (*models.Category, error) {
	var c models.Category
	err := q.db.QueryRow(
		"SELECT id, user_id, name, position FROM categories WHERE id = ? AND user_id = ?",
		id, userID,
	).Scan(&c.ID, &c.UserID, &c.Name, &c.Position)
	if err != nil {
		return nil, err
	}
	return &c, nil
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
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return categories, nil
}

func (q *Queries) UpdateCategory(id, userID int64, name *string, position *int) error {
	var setClauses []string
	var args []interface{}

	if name != nil {
		setClauses = append(setClauses, "name = ?")
		args = append(args, *name)
	}
	if position != nil {
		setClauses = append(setClauses, "position = ?")
		args = append(args, *position)
	}

	if len(setClauses) == 0 {
		return nil
	}

	query := "UPDATE categories SET " + strings.Join(setClauses, ", ") + " WHERE id = ? AND user_id = ?"
	args = append(args, id, userID)
	_, err := q.db.Exec(query, args...)
	return err
}

func (q *Queries) DeleteCategory(id, userID int64) error {
	_, err := q.db.Exec("DELETE FROM categories WHERE id = ? AND user_id = ?", id, userID)
	return err
}

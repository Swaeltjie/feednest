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
	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}
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
	if err := rows.Err(); err != nil {
		return nil, err
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

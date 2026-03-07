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
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return tags, nil
}

func (q *Queries) AddTagToArticle(userID int64, articleID int64, tagName string) error {
	_, err := q.db.Exec("INSERT OR IGNORE INTO tags (user_id, name) VALUES (?, ?)", userID, tagName)
	if err != nil {
		return err
	}

	var tagID int64
	err = q.db.QueryRow("SELECT id FROM tags WHERE user_id = ? AND name = ?", userID, tagName).Scan(&tagID)
	if err != nil {
		return err
	}

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
		JOIN article_tags at2 ON t.id = at2.tag_id
		WHERE at2.article_id = ? ORDER BY t.name`, articleID)
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
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return tags, nil
}

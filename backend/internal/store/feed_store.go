package store

import (
	"strings"
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
	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}
	return &models.Feed{
		ID:            id,
		UserID:        userID,
		URL:           url,
		Title:         title,
		SiteURL:       siteURL,
		IconURL:       iconURL,
		CategoryID:    categoryID,
		FetchInterval: 900,
		CreatedAt:     time.Now(),
	}, nil
}

func (q *Queries) ListFeeds(userID int64) ([]models.Feed, error) {
	rows, err := q.db.Query(`
		SELECT f.id, f.user_id, f.url, f.title, f.site_url, f.icon_url,
			f.category_id, f.fetch_interval, f.last_fetched, f.engagement_score, f.created_at, f.last_error,
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
			&f.CategoryID, &f.FetchInterval, &f.LastFetched, &f.EngagementScore, &f.CreatedAt, &f.LastError, &f.UnreadCount); err != nil {
			return nil, err
		}
		feeds = append(feeds, f)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return feeds, nil
}

func (q *Queries) GetFeed(id, userID int64) (*models.Feed, error) {
	var f models.Feed
	err := q.db.QueryRow(`
		SELECT id, user_id, url, title, site_url, icon_url, category_id, fetch_interval, last_fetched, engagement_score, created_at, last_error
		FROM feeds WHERE id = ? AND user_id = ?`, id, userID,
	).Scan(&f.ID, &f.UserID, &f.URL, &f.Title, &f.SiteURL, &f.IconURL,
		&f.CategoryID, &f.FetchInterval, &f.LastFetched, &f.EngagementScore, &f.CreatedAt, &f.LastError)
	if err != nil {
		return nil, err
	}
	return &f, nil
}

func (q *Queries) UpdateFeed(id, userID int64, req *models.UpdateFeedRequest) error {
	var setClauses []string
	var args []interface{}

	if req.Title != nil {
		setClauses = append(setClauses, "title = ?")
		args = append(args, *req.Title)
	}
	if req.CategoryID != nil {
		setClauses = append(setClauses, "category_id = ?")
		args = append(args, *req.CategoryID)
	}
	if req.FetchInterval != nil {
		setClauses = append(setClauses, "fetch_interval = ?")
		args = append(args, *req.FetchInterval)
	}

	if len(setClauses) == 0 {
		return nil
	}

	query := "UPDATE feeds SET " + strings.Join(setClauses, ", ") + " WHERE id = ? AND user_id = ?"
	args = append(args, id, userID)
	_, err := q.db.Exec(query, args...)
	return err
}

func (q *Queries) UpdateFeedLastFetched(id int64) error {
	_, err := q.db.Exec("UPDATE feeds SET last_fetched = CURRENT_TIMESTAMP WHERE id = ?", id)
	return err
}

func (q *Queries) ClearFeedCategory(id, userID int64) error {
	_, err := q.db.Exec("UPDATE feeds SET category_id = NULL WHERE id = ? AND user_id = ?", id, userID)
	return err
}

func (q *Queries) DeleteFeed(id, userID int64) error {
	_, err := q.db.Exec("DELETE FROM feeds WHERE id = ? AND user_id = ?", id, userID)
	return err
}

func (q *Queries) GetFeedsDueForFetch() ([]models.Feed, error) {
	rows, err := q.db.Query(`
		SELECT id, user_id, url, title, site_url, icon_url, category_id, fetch_interval, last_fetched, engagement_score, created_at, last_error
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
			&f.CategoryID, &f.FetchInterval, &f.LastFetched, &f.EngagementScore, &f.CreatedAt, &f.LastError); err != nil {
			return nil, err
		}
		feeds = append(feeds, f)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return feeds, nil
}

type FeedMetadataUpdate struct {
	Title   *string
	SiteURL *string
	IconURL *string
}

func (q *Queries) SetFeedError(id int64, errMsg string) error {
	_, err := q.db.Exec("UPDATE feeds SET last_error = ? WHERE id = ?", errMsg, id)
	return err
}

func (q *Queries) ClearFeedError(id int64) error {
	_, err := q.db.Exec("UPDATE feeds SET last_error = NULL WHERE id = ?", id)
	return err
}

func (q *Queries) UpdateFeedMetadata(id int64, update *FeedMetadataUpdate) error {
	if update == nil {
		return nil
	}

	var setClauses []string
	var args []interface{}

	if update.Title != nil {
		setClauses = append(setClauses, "title = ?")
		args = append(args, *update.Title)
	}
	if update.SiteURL != nil {
		setClauses = append(setClauses, "site_url = ?")
		args = append(args, *update.SiteURL)
	}
	if update.IconURL != nil {
		setClauses = append(setClauses, "icon_url = ?")
		args = append(args, *update.IconURL)
	}

	if len(setClauses) == 0 {
		return nil
	}

	query := "UPDATE feeds SET " + strings.Join(setClauses, ", ") + " WHERE id = ?"
	args = append(args, id)
	_, err := q.db.Exec(query, args...)
	return err
}

package store

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/feednest/backend/internal/models"
)

var htmlTagRe = regexp.MustCompile(`<[^>]*>`)

func makeSnippet(html string, maxLen int) string {
	text := htmlTagRe.ReplaceAllString(html, "")
	text = strings.Join(strings.Fields(text), " ")
	if len(text) > maxLen {
		text = text[:maxLen]
		if i := strings.LastIndex(text, " "); i > maxLen-40 {
			text = text[:i]
		}
		text += "\u2026"
	}
	return text
}

type ArticleFilter struct {
	CategoryID *int64
	FeedID     *int64
	Status     string
	Sort       string
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
		conditions = append(conditions, "EXISTS (SELECT 1 FROM article_tags at2 JOIN tags t ON at2.tag_id = t.id WHERE at2.article_id = a.id AND t.name = ?)")
		args = append(args, filter.Tag)
	}

	where := strings.Join(conditions, " AND ")

	var total int
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM articles a JOIN feeds f ON a.feed_id = f.id WHERE %s", where)
	if err := q.db.QueryRow(countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	var orderBy string
	switch filter.Sort {
	case "oldest":
		orderBy = "COALESCE(a.published_at, a.fetched_at) ASC"
	case "smart":
		orderBy = "a.score DESC, COALESCE(a.published_at, a.fetched_at) DESC"
	default:
		orderBy = "COALESCE(a.published_at, a.fetched_at) DESC"
	}

	offset := (filter.Page - 1) * filter.Limit
	query := fmt.Sprintf(`
		SELECT a.id, a.feed_id, a.guid, a.title, a.url, a.author, '', a.content_clean,
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
		a.Snippet = makeSnippet(a.ContentClean, 160)
		a.ContentClean = ""
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

func (q *Queries) CreateReadingEvent(articleID int64, eventType string, durationSeconds int) error {
	_, err := q.db.Exec(
		"INSERT INTO reading_events (article_id, event_type, duration_seconds) VALUES (?, ?, ?)",
		articleID, eventType, durationSeconds,
	)
	return err
}

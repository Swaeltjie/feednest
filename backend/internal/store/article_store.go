package store

import (
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/feednest/backend/internal/models"
)

var htmlTagRe = regexp.MustCompile(`<[^>]*>`)

func makeSnippet(html string, maxLen int) string {
	text := htmlTagRe.ReplaceAllString(html, "")
	text = strings.Join(strings.Fields(text), " ")
	// Filter out blocked/bot-protection content from snippets
	if isSnippetBlocked(text) {
		return ""
	}
	runes := []rune(text)
	if utf8.RuneCountInString(text) > maxLen {
		runes = runes[:maxLen]
		truncated := string(runes)
		if i := strings.LastIndex(truncated, " "); i > maxLen-40 {
			truncated = truncated[:i]
		}
		return truncated + "\u2026"
	}
	return text
}

func isSnippetBlocked(text string) bool {
	lower := strings.ToLower(text)
	markers := []string{
		"please enable cookies",
		"you have been blocked",
		"cloudflare ray id",
		"please enable js and disable any ad blocker",
		"403 forbidden",
		"access denied",
		"robot sensors",
		"security service to protect itself",
	}
	for _, m := range markers {
		if strings.Contains(lower, m) {
			return true
		}
	}
	return false
}

type ArticleFilter struct {
	CategoryID *int64
	FeedID     *int64
	Status     string
	Sort       string
	Tag        string
	Search     string
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
	// Clear blocked content so reader doesn't display it
	if isSnippetBlocked(a.ContentClean) {
		a.ContentClean = ""
	}
	if isSnippetBlocked(a.ContentRaw) {
		a.ContentRaw = ""
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
	if filter.Search != "" {
		escaped := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(filter.Search)
		searchTerm := "%" + escaped + "%"
		conditions = append(conditions, "(a.title LIKE ? ESCAPE '\\' OR a.content_clean LIKE ? ESCAPE '\\' OR a.content_raw LIKE ? ESCAPE '\\')")
		args = append(args, searchTerm, searchTerm, searchTerm)
	}

	// Cross-feed deduplication: keep only the article with the lowest ID for each URL
	conditions = append(conditions, `a.id = (SELECT MIN(a2.id) FROM articles a2 JOIN feeds f2 ON a2.feed_id = f2.id WHERE a2.url = a.url AND f2.user_id = ? AND a2.url != '')`)
	args = append(args, userID)

	// Filter sponsored/ad content
	conditions = append(conditions, `a.title NOT LIKE '%[Sponsored]%' AND a.title NOT LIKE '%[Ad]%' AND a.title NOT LIKE '%Sponsored Post%' AND a.title NOT LIKE '%Advertisement%'`)

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

	if filter.Page < 1 {
		filter.Page = 1
	}
	offset := (filter.Page - 1) * filter.Limit
	query := fmt.Sprintf(`
		SELECT a.id, a.feed_id, a.guid, a.title, a.url, a.author, a.content_raw, a.content_clean,
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
		snippet := makeSnippet(a.ContentClean, 160)
		if snippet == "" {
			snippet = makeSnippet(a.ContentRaw, 160)
		}
		a.Snippet = snippet
		a.ContentClean = ""
		a.ContentRaw = ""
		articles = append(articles, a)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return articles, total, nil
}

func (q *Queries) UpdateArticle(id, userID int64, isRead *bool, isStarred *bool) error {
	if isRead != nil {
		var readAt interface{}
		if *isRead {
			readAt = time.Now()
		}
		if _, err := q.db.Exec(`
			UPDATE articles SET is_read = ?, read_at = ?
			WHERE id = ? AND feed_id IN (SELECT id FROM feeds WHERE user_id = ?)`,
			*isRead, readAt, id, userID); err != nil {
			return err
		}
	}
	if isStarred != nil {
		if _, err := q.db.Exec(`
			UPDATE articles SET is_starred = ?
			WHERE id = ? AND feed_id IN (SELECT id FROM feeds WHERE user_id = ?)`,
			*isStarred, id, userID); err != nil {
			return err
		}
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

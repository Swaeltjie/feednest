package store

import (
	"database/sql"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/feednest/backend/internal/models"
)

// ReadingStats holds weekly reading statistics for a user.
type ReadingStats struct {
	ArticlesRead int `json:"articles_read"`
	TotalMinutes int `json:"total_minutes"`
	FeedsRead    int `json:"feeds_read"`
}

// WPM cache: stores per-user words-per-minute with a 5-minute TTL.
var (
	wpmCache   = make(map[int64]cachedWPM)
	wpmCacheMu sync.RWMutex
)

type cachedWPM struct {
	value     float64
	expiresAt time.Time
}

// GetUserWPM calculates the user's average reading speed in words per minute
// based on their reading_events history. Returns 200.0 (default) if there are
// fewer than 5 qualifying data points. The result is clamped to [100, 600] WPM
// and cached for 5 minutes.
func (q *Queries) GetUserWPM(userID int64) float64 {
	const defaultWPM = 200.0

	// Check cache first
	wpmCacheMu.RLock()
	if cached, ok := wpmCache[userID]; ok && time.Now().Before(cached.expiresAt) {
		wpmCacheMu.RUnlock()
		return cached.value
	}
	wpmCacheMu.RUnlock()

	var avgWPM sql.NullFloat64
	var cnt int
	err := q.db.QueryRow(`
		SELECT AVG(a.word_count * 60.0 / re.duration_seconds) as avg_wpm, COUNT(*) as cnt
		FROM reading_events re
		JOIN articles a ON re.article_id = a.id
		JOIN feeds f ON a.feed_id = f.id
		WHERE f.user_id = ?
		  AND re.event_type = 'read'
		  AND re.duration_seconds >= 15
		  AND re.duration_seconds <= 1800
		  AND a.word_count >= 50`, userID).Scan(&avgWPM, &cnt)
	if err != nil || cnt < 5 || !avgWPM.Valid {
		q.cacheWPM(userID, defaultWPM)
		return defaultWPM
	}

	wpm := avgWPM.Float64
	// Clamp to reasonable human range
	if wpm < 100 {
		wpm = 100
	} else if wpm > 600 {
		wpm = 600
	}

	q.cacheWPM(userID, wpm)
	return wpm
}

func (q *Queries) cacheWPM(userID int64, wpm float64) {
	wpmCacheMu.Lock()
	wpmCache[userID] = cachedWPM{value: wpm, expiresAt: time.Now().Add(5 * time.Minute)}
	wpmCacheMu.Unlock()
}

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
	CategoryID     *int64
	FeedID         *int64
	Status         string
	Sort           string
	Tag            string
	Search         string
	PublishedAfter string
	MinReadingTime int
	MaxReadingTime int
	Page           int
	Limit          int
}

func (q *Queries) CreateArticle(feedID int64, guid, title, url, author, contentRaw, contentClean, thumbnailURL string, publishedAt *time.Time, wordCount, readingTime int) error {
	_, err := q.db.Exec(`
		INSERT OR IGNORE INTO articles (feed_id, guid, title, url, author, content_raw, content_clean, thumbnail_url, published_at, word_count, reading_time)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		feedID, guid, title, url, author, contentRaw, contentClean, thumbnailURL, publishedAt, wordCount, readingTime,
	)
	return err
}

// CreateArticleAndApplyRules creates an article and applies auto_read/auto_star rules.
// Returns true if the article was newly inserted.
func (q *Queries) CreateArticleAndApplyRules(userID, feedID int64, guid, title, url, author, contentRaw, contentClean, thumbnailURL string, publishedAt *time.Time, wordCount, readingTime int) (bool, error) {
	result, err := q.db.Exec(`
		INSERT OR IGNORE INTO articles (feed_id, guid, title, url, author, content_raw, content_clean, thumbnail_url, published_at, word_count, reading_time)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		feedID, guid, title, url, author, contentRaw, contentClean, thumbnailURL, publishedAt, wordCount, readingTime,
	)
	if err != nil {
		return false, err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	if rowsAffected == 0 {
		return false, nil // article already existed
	}

	articleID, err := result.LastInsertId()
	if err != nil {
		return true, err
	}

	// Apply auto rules for the newly created article
	content := contentClean
	if content == "" {
		content = contentRaw
	}
	if err := q.ApplyAutoRules(userID, articleID, feedID, title, author, content); err != nil {
		return true, err
	}

	return true, nil
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

	// Override reading_time with personalized WPM
	userWPM := q.GetUserWPM(userID)
	if a.WordCount > 0 {
		a.ReadingTime = int(math.Ceil(float64(a.WordCount) / userWPM))
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
	if filter.PublishedAfter != "" {
		conditions = append(conditions, "COALESCE(a.published_at, a.fetched_at) >= ?")
		args = append(args, filter.PublishedAfter)
	}
	if filter.MinReadingTime > 0 {
		conditions = append(conditions, "a.reading_time >= ?")
		args = append(args, filter.MinReadingTime)
	}
	if filter.MaxReadingTime > 0 {
		conditions = append(conditions, "a.reading_time <= ?")
		args = append(args, filter.MaxReadingTime)
	}

	// Cross-feed deduplication: keep only the article with the lowest ID for each URL
	conditions = append(conditions, `a.id = (SELECT MIN(a2.id) FROM articles a2 JOIN feeds f2 ON a2.feed_id = f2.id WHERE a2.url = a.url AND f2.user_id = ? AND a2.url != '')`)
	args = append(args, userID)

	// Filter sponsored/ad content
	conditions = append(conditions, `a.title NOT LIKE '%[Sponsored]%' AND a.title NOT LIKE '%[Ad]%' AND a.title NOT LIKE '%Sponsored Post%' AND a.title NOT LIKE '%Advertisement%'`)

	// Apply hide rules (contains/not_contains in SQL; regex rules post-filtered)
	hideRules, _ := q.GetHideRules(userID)
	var regexHideRules []models.FilterRule
	for _, rule := range hideRules {
		col := "a." + rule.Field
		escaped := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(rule.Value)

		if rule.FeedID != nil {
			switch rule.Operator {
			case "contains":
				conditions = append(conditions, fmt.Sprintf("NOT (a.feed_id = ? AND %s LIKE ? ESCAPE '\\')", col))
				args = append(args, *rule.FeedID, "%"+escaped+"%")
			case "not_contains":
				conditions = append(conditions, fmt.Sprintf("NOT (a.feed_id = ? AND %s NOT LIKE ? ESCAPE '\\')", col))
				args = append(args, *rule.FeedID, "%"+escaped+"%")
			case "regex":
				regexHideRules = append(regexHideRules, rule)
			}
		} else {
			switch rule.Operator {
			case "contains":
				conditions = append(conditions, fmt.Sprintf("%s NOT LIKE ? ESCAPE '\\'", col))
				args = append(args, "%"+escaped+"%")
			case "not_contains":
				conditions = append(conditions, fmt.Sprintf("%s LIKE ? ESCAPE '\\'", col))
				args = append(args, "%"+escaped+"%")
			case "regex":
				regexHideRules = append(regexHideRules, rule)
			}
		}
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

	// Override reading_time with personalized WPM
	userWPM := q.GetUserWPM(userID)
	for i := range articles {
		if articles[i].WordCount > 0 {
			articles[i].ReadingTime = int(math.Ceil(float64(articles[i].WordCount) / userWPM))
		}
	}

	// Post-filter regex hide rules
	if len(regexHideRules) > 0 {
		filtered := articles[:0]
		for _, a := range articles {
			hidden := false
			for _, rule := range regexHideRules {
				if rule.FeedID != nil && a.FeedID != *rule.FeedID {
					continue
				}
				var fieldValue string
				switch rule.Field {
				case "title":
					fieldValue = a.Title
				case "author":
					fieldValue = a.Author
				case "content":
					fieldValue = a.Snippet
				}
				if re, err := compileRegexCached(rule.Value); err == nil && re.MatchString(fieldValue) {
					hidden = true
					break
				}
			}
			if !hidden {
				filtered = append(filtered, a)
			}
		}
		articles = filtered
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

func (q *Queries) MarkAllRead(userID int64, feedID *int64, categoryID *int64) (int64, error) {
	var query string
	var args []interface{}

	if feedID != nil {
		query = `UPDATE articles SET is_read = 1, read_at = CURRENT_TIMESTAMP
			WHERE is_read = 0 AND feed_id = ? AND feed_id IN (SELECT id FROM feeds WHERE user_id = ?)`
		args = []interface{}{*feedID, userID}
	} else if categoryID != nil {
		query = `UPDATE articles SET is_read = 1, read_at = CURRENT_TIMESTAMP
			WHERE is_read = 0 AND feed_id IN (SELECT id FROM feeds WHERE user_id = ? AND category_id = ?)`
		args = []interface{}{userID, *categoryID}
	} else {
		query = `UPDATE articles SET is_read = 1, read_at = CURRENT_TIMESTAMP
			WHERE is_read = 0 AND feed_id IN (SELECT id FROM feeds WHERE user_id = ?)`
		args = []interface{}{userID}
	}

	result, err := q.db.Exec(query, args...)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (q *Queries) CatchUp(userID int64, strategy string, value string, count int, feedID *int64, categoryID *int64) (int64, error) {
	switch strategy {
	case "older_than":
		var duration time.Duration
		if len(value) < 2 {
			return 0, fmt.Errorf("invalid duration value")
		}
		numStr := value[:len(value)-1]
		unit := value[len(value)-1:]
		num, err := strconv.Atoi(numStr)
		if err != nil || num <= 0 {
			return 0, fmt.Errorf("invalid duration number")
		}
		switch unit {
		case "d":
			duration = time.Duration(num) * 24 * time.Hour
		case "w":
			duration = time.Duration(num) * 7 * 24 * time.Hour
		case "h":
			duration = time.Duration(num) * time.Hour
		default:
			return 0, fmt.Errorf("invalid duration unit, use h/d/w")
		}

		cutoff := time.Now().Add(-duration)
		var conditions []string
		var args []interface{}

		conditions = append(conditions, "a.is_read = 0")
		conditions = append(conditions, "COALESCE(a.published_at, a.fetched_at) < ?")
		args = append(args, cutoff)

		if feedID != nil {
			conditions = append(conditions, "a.feed_id = ?")
			args = append(args, *feedID)
			conditions = append(conditions, "a.feed_id IN (SELECT id FROM feeds WHERE user_id = ?)")
			args = append(args, userID)
		} else if categoryID != nil {
			conditions = append(conditions, "a.feed_id IN (SELECT id FROM feeds WHERE user_id = ? AND category_id = ?)")
			args = append(args, userID, *categoryID)
		} else {
			conditions = append(conditions, "a.feed_id IN (SELECT id FROM feeds WHERE user_id = ?)")
			args = append(args, userID)
		}

		query := fmt.Sprintf(`UPDATE articles a SET is_read = 1, read_at = CURRENT_TIMESTAMP WHERE %s`,
			strings.Join(conditions, " AND "))
		result, err := q.db.Exec(query, args...)
		if err != nil {
			return 0, err
		}
		return result.RowsAffected()

	case "keep_newest":
		if count <= 0 {
			return 0, fmt.Errorf("count must be positive")
		}

		var feedCondition string
		var args []interface{}

		if feedID != nil {
			feedCondition = "f.user_id = ? AND f.id = ?"
			args = []interface{}{userID, *feedID}
		} else if categoryID != nil {
			feedCondition = "f.user_id = ? AND f.category_id = ?"
			args = []interface{}{userID, *categoryID}
		} else {
			feedCondition = "f.user_id = ?"
			args = []interface{}{userID}
		}

		// Mark as read all unread articles except the newest N per feed
		query := fmt.Sprintf(`
			UPDATE articles SET is_read = 1, read_at = CURRENT_TIMESTAMP
			WHERE is_read = 0
			AND feed_id IN (SELECT id FROM feeds f WHERE %s)
			AND id NOT IN (
				SELECT a.id FROM articles a
				JOIN feeds f ON a.feed_id = f.id
				WHERE %s
				AND a.is_read = 0
				ORDER BY COALESCE(a.published_at, a.fetched_at) DESC
				LIMIT ?
			)`, feedCondition, feedCondition)

		// Double the args for the two subqueries, plus count
		allArgs := append(args, args...)
		allArgs = append(allArgs, count)

		result, err := q.db.Exec(query, allArgs...)
		if err != nil {
			return 0, err
		}
		return result.RowsAffected()

	default:
		return 0, fmt.Errorf("unknown strategy: %s", strategy)
	}
}

func (q *Queries) GetWeeklyReadingStats(userID int64) (*ReadingStats, error) {
	row := q.db.QueryRow(`
		SELECT
			COUNT(*) as articles_read,
			COALESCE(SUM(a.reading_time), 0) as total_minutes,
			COUNT(DISTINCT a.feed_id) as feeds_read
		FROM articles a
		JOIN feeds f ON f.id = a.feed_id AND f.user_id = ?
		WHERE a.is_read = 1
		AND a.read_at >= datetime('now', '-7 days')
	`, userID)

	stats := &ReadingStats{}
	err := row.Scan(&stats.ArticlesRead, &stats.TotalMinutes, &stats.FeedsRead)
	if err != nil {
		return nil, err
	}
	return stats, nil
}

func (q *Queries) CreateReadingEvent(articleID int64, eventType string, durationSeconds int) error {
	_, err := q.db.Exec(
		"INSERT INTO reading_events (article_id, event_type, duration_seconds) VALUES (?, ?, ?)",
		articleID, eventType, durationSeconds,
	)
	return err
}

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
	wpmCacheMu sync.Mutex
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

	// Single Lock for the entire check-and-set to avoid race condition.
	// Less concurrent but safe for SQLite's single-connection model.
	wpmCacheMu.Lock()
	defer wpmCacheMu.Unlock()

	if cached, ok := wpmCache[userID]; ok && time.Now().Before(cached.expiresAt) {
		return cached.value
	}

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
		wpmCache[userID] = cachedWPM{value: defaultWPM, expiresAt: time.Now().Add(5 * time.Minute)}
		return defaultWPM
	}

	wpm := avgWPM.Float64
	// Clamp to reasonable human range
	if wpm < 100 {
		wpm = 100
	} else if wpm > 600 {
		wpm = 600
	}

	wpmCache[userID] = cachedWPM{value: wpm, expiresAt: time.Now().Add(5 * time.Minute)}
	return wpm
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

// ArticleExistsByGUID checks if an article with the given GUID already exists for a feed.
func (q *Queries) ArticleExistsByGUID(feedID int64, guid string) bool {
	var exists int
	err := q.db.QueryRow("SELECT 1 FROM articles WHERE feed_id = ? AND guid = ? LIMIT 1", feedID, guid).Scan(&exists)
	return err == nil
}

func (q *Queries) CreateArticle(feedID int64, guid, title, url, author, contentRaw, contentClean, thumbnailURL string, publishedAt *time.Time, wordCount, readingTime int) error {
	_, err := q.db.Exec(`
		INSERT OR IGNORE INTO articles (feed_id, guid, title, url, author, content_raw, content_clean, thumbnail_url, published_at, word_count, reading_time)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		feedID, guid, title, url, author, contentRaw, contentClean, thumbnailURL, publishedAt, wordCount, readingTime,
	)
	return err
}

// CreateArticleAndApplyRules creates an article and applies auto_read/auto_star
// rules atomically. Returns true if the article was newly inserted.
//
// The insert and the rule-derived flag updates run in a single transaction so a
// crash can't leave a persisted article with its auto_read/auto_star rules
// un-applied (the scheduler then skips it forever via ArticleExistsByGUID).
// Rules are read BEFORE Begin: with SetMaxOpenConns(1) a query issued while the
// transaction holds the sole connection would deadlock.
func (q *Queries) CreateArticleAndApplyRules(userID, feedID int64, guid, title, url, author, contentRaw, contentClean, thumbnailURL string, publishedAt *time.Time, wordCount, readingTime int) (bool, error) {
	content := contentClean
	if content == "" {
		content = contentRaw
	}
	rules, err := q.GetRulesForFeed(userID, &feedID)
	if err != nil {
		return false, err
	}
	autoRead, autoStar := matchedAutoActions(rules, title, author, content)

	tx, err := q.db.Begin()
	if err != nil {
		return false, err
	}
	defer tx.Rollback()

	result, err := tx.Exec(`
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
		return false, tx.Commit() // article already existed — nothing to do
	}

	articleID, err := result.LastInsertId()
	if err != nil {
		return true, err
	}

	if autoRead {
		if _, err := tx.Exec(`UPDATE articles SET is_read = 1, read_at = CURRENT_TIMESTAMP WHERE id = ?`, articleID); err != nil {
			return true, err
		}
	}
	if autoStar {
		if _, err := tx.Exec(`UPDATE articles SET is_starred = 1 WHERE id = ?`, articleID); err != nil {
			return true, err
		}
	}

	return true, tx.Commit()
}

func (q *Queries) GetArticle(id, userID int64) (*models.Article, error) {
	var a models.Article
	err := q.db.QueryRow(`
		SELECT a.id, a.feed_id, a.guid, a.title, a.url, a.author, a.content_raw, a.content_clean,
			a.thumbnail_url, a.published_at, a.fetched_at, a.word_count, a.reading_time,
			a.is_read, a.is_starred, a.read_at, a.score, COALESCE(a.summary, '') as summary,
			COALESCE(f.title, '') as feed_title, COALESCE(f.icon_url, '') as feed_icon_url
		FROM articles a
		JOIN feeds f ON a.feed_id = f.id
		WHERE a.id = ? AND f.user_id = ?`, id, userID,
	).Scan(&a.ID, &a.FeedID, &a.GUID, &a.Title, &a.URL, &a.Author, &a.ContentRaw, &a.ContentClean,
		&a.ThumbnailURL, &a.PublishedAt, &a.FetchedAt, &a.WordCount, &a.ReadingTime,
		&a.IsRead, &a.IsStarred, &a.ReadAt, &a.Score, &a.Summary, &a.FeedTitle, &a.FeedIconURL)
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
		// Prefer FTS5 (stemming, token/phrase matching, relevance) when the
		// SQLite build supports it; otherwise fall back to LIKE substring match.
		if ftsMatch := buildFTSMatch(filter.Search); FTS5Enabled && ftsMatch != "" {
			conditions = append(conditions, "a.id IN (SELECT rowid FROM articles_fts WHERE articles_fts MATCH ?)")
			args = append(args, ftsMatch)
		} else {
			escaped := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(filter.Search)
			searchTerm := "%" + escaped + "%"
			conditions = append(conditions, "(a.title LIKE ? ESCAPE '\\' OR a.content_clean LIKE ? ESCAPE '\\' OR a.content_raw LIKE ? ESCAPE '\\')")
			args = append(args, searchTerm, searchTerm, searchTerm)
		}
	}
	if filter.PublishedAfter != "" {
		// Normalize both sides with SQLite's datetime(): stored values are
		// space-separated with a numeric offset (e.g. "2026-06-09 12:00:00+00:00")
		// and may carry non-UTC offsets, while the threshold is RFC3339
		// ("T"-separated, "Z"). A raw lexicographic compare drops every
		// same-day row; datetime() parses and compares them correctly.
		conditions = append(conditions, "datetime(COALESCE(a.published_at, a.fetched_at)) >= datetime(?)")
		args = append(args, filter.PublishedAfter)
	}
	// Filter by reading time using word_count + user's personalized WPM,
	// so the filter matches what the user actually sees displayed.
	if filter.MinReadingTime > 0 || filter.MaxReadingTime > 0 {
		userWPM := q.GetUserWPM(userID)
		if filter.MinReadingTime > 0 {
			minWords := int(float64(filter.MinReadingTime) * userWPM)
			conditions = append(conditions, "a.word_count >= ?")
			args = append(args, minWords)
		}
		if filter.MaxReadingTime > 0 {
			maxWords := int(math.Ceil(float64(filter.MaxReadingTime) * userWPM))
			conditions = append(conditions, "a.word_count <= ?")
			args = append(args, maxWords)
		}
	}

	// Cross-feed deduplication: keep only the article with the lowest ID for each URL.
	// Articles with empty URLs are exempt (no URL to deduplicate on).
	conditions = append(conditions, `(a.url = '' OR a.id = (SELECT MIN(a2.id) FROM articles a2 JOIN feeds f2 ON a2.feed_id = f2.id WHERE a2.url = a.url AND f2.user_id = ? AND a2.url != ''))`)
	args = append(args, userID)

	// Filter sponsored/ad content
	conditions = append(conditions, `a.title NOT LIKE '%[Sponsored]%' AND a.title NOT LIKE '%[Ad]%' AND a.title NOT LIKE '%Sponsored Post%' AND a.title NOT LIKE '%Advertisement%'`)

	// Apply hide rules (contains/not_contains in SQL; regex rules post-filtered)
	hideRules, err := q.GetHideRules(userID)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get hide rules: %w", err)
	}
	allowedFields := map[string]string{
		"title":   "a.title",
		"author":  "a.author",
		"content": "a.content_raw",
	}
	var regexHideRules []models.FilterRule
	for _, rule := range hideRules {
		col, ok := allowedFields[rule.Field]
		if !ok {
			continue // skip invalid field
		}
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

	// Regex hide rules cannot be expressed in SQL, so they are post-filtered in
	// Go. When any are active we must filter BEFORE pagination, otherwise the
	// SQL COUNT(*) over-reports the total and rows hidden on a truncated page
	// are silently dropped without being backfilled from the next page. In that
	// case we fetch all candidate rows (up to a safety cap), apply the regex
	// filter, recompute the total, then slice the requested page in Go.
	if len(regexHideRules) > 0 {
		// Safety cap to bound memory: hide rules are expected to remove only a
		// small fraction of articles, so over-fetching everything is acceptable.
		const regexCandidateCap = 50000
		query := fmt.Sprintf(`
			SELECT a.id, a.feed_id, a.guid, a.title, a.url, a.author, a.content_raw, a.content_clean,
				a.thumbnail_url, a.published_at, a.fetched_at, a.word_count, a.reading_time,
				a.is_read, a.is_starred, a.read_at, a.score,
				COALESCE(f.title, '') as feed_title, COALESCE(f.icon_url, '') as feed_icon_url
			FROM articles a
			JOIN feeds f ON a.feed_id = f.id
			WHERE %s
			ORDER BY %s
			LIMIT ?`, where, orderBy)

		queryArgs := append(args, regexCandidateCap)
		rows, err := q.db.Query(query, queryArgs...)
		if err != nil {
			return nil, 0, err
		}
		defer rows.Close()

		// Collect candidates and apply the regex hide filter before truncating
		// content, so regex matches against the complete article text.
		var filtered []models.Article
		for rows.Next() {
			var a models.Article
			if err := rows.Scan(&a.ID, &a.FeedID, &a.GUID, &a.Title, &a.URL, &a.Author, &a.ContentRaw, &a.ContentClean,
				&a.ThumbnailURL, &a.PublishedAt, &a.FetchedAt, &a.WordCount, &a.ReadingTime,
				&a.IsRead, &a.IsStarred, &a.ReadAt, &a.Score, &a.FeedTitle, &a.FeedIconURL); err != nil {
				return nil, 0, err
			}
			if articleHiddenByRegex(&a, regexHideRules) {
				continue
			}
			snippet := makeSnippet(a.ContentClean, 160)
			if snippet == "" {
				snippet = makeSnippet(a.ContentRaw, 160)
			}
			a.Snippet = snippet
			filtered = append(filtered, a)
		}
		if err := rows.Err(); err != nil {
			return nil, 0, err
		}

		// Recompute total from the filtered set so pagination and counts stay
		// consistent with what the user can actually reach.
		total := len(filtered)

		// Slice the requested page with bounds-checked offset/limit.
		start := (filter.Page - 1) * filter.Limit
		if start < 0 || start > len(filtered) {
			start = len(filtered)
		}
		end := start + filter.Limit
		if filter.Limit <= 0 || end > len(filtered) {
			end = len(filtered)
		}
		articles := filtered[start:end]

		// Override reading_time with personalized WPM and clear full content
		// (only snippets are needed) on the final sliced page.
		userWPM := q.GetUserWPM(userID)
		for i := range articles {
			if articles[i].WordCount > 0 {
				articles[i].ReadingTime = int(math.Ceil(float64(articles[i].WordCount) / userWPM))
			}
			articles[i].ContentClean = ""
			articles[i].ContentRaw = ""
		}

		return articles, total, nil
	}

	// Common fast path: no regex hide rules, so SQL COUNT + LIMIT/OFFSET are
	// accurate and we let SQLite do the pagination.
	var total int
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM articles a JOIN feeds f ON a.feed_id = f.id WHERE %s", where)
	if err := q.db.QueryRow(countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
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

	// Clear full content from list responses (only snippets needed)
	for i := range articles {
		articles[i].ContentClean = ""
		articles[i].ContentRaw = ""
	}

	return articles, total, nil
}

// articleHiddenByRegex reports whether the article matches any of the given
// regex hide rules. Feed-scoped rules only apply to their own feed. Invalid
// patterns are ignored (they never hide an article).
func articleHiddenByRegex(a *models.Article, regexHideRules []models.FilterRule) bool {
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
			if a.ContentClean != "" {
				fieldValue = a.ContentClean
			} else {
				fieldValue = a.ContentRaw
			}
		}
		if re, err := compileRegexCached(rule.Value); err == nil && re.MatchString(fieldValue) {
			return true
		}
	}
	return false
}

func (q *Queries) UpdateArticleContent(id int64, contentClean string, wordCount, readingTime int) error {
	_, err := q.db.Exec(
		`UPDATE articles SET content_clean = ?, word_count = ?, reading_time = ? WHERE id = ?`,
		contentClean, wordCount, readingTime, id,
	)
	return err
}

// UpdateArticleSummary stores a cached AI-generated TL;DR for an article,
// scoped to the owning user so it can never write across tenants even if a
// future caller skips the user-scoped GetArticle precheck.
func (q *Queries) UpdateArticleSummary(id, userID int64, summary string) error {
	_, err := q.db.Exec(
		`UPDATE articles SET summary = ? WHERE id = ? AND feed_id IN (SELECT id FROM feeds WHERE user_id = ?)`,
		summary, id, userID)
	return err
}

func (q *Queries) UpdateArticleThumbnail(id int64, thumbnailURL string) error {
	_, err := q.db.Exec(
		`UPDATE articles SET thumbnail_url = ? WHERE id = ?`,
		thumbnailURL, id,
	)
	return err
}

// GetArticlesMissingThumbnails returns articles that have a URL but no thumbnail.
func (q *Queries) GetArticlesMissingThumbnails(limit int) ([]models.Article, error) {
	rows, err := q.db.Query(
		`SELECT id, url FROM articles WHERE url != '' AND (thumbnail_url IS NULL OR thumbnail_url = '') ORDER BY id DESC LIMIT ?`,
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var articles []models.Article
	for rows.Next() {
		var a models.Article
		if err := rows.Scan(&a.ID, &a.URL); err != nil {
			return nil, err
		}
		articles = append(articles, a)
	}
	return articles, rows.Err()
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
		if num > 36500 {
			return 0, fmt.Errorf("duration value too large, max 36500")
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

		conditions = append(conditions, "is_read = 0")
		conditions = append(conditions, "COALESCE(published_at, fetched_at) < ?")
		args = append(args, cutoff)

		if feedID != nil {
			conditions = append(conditions, "feed_id = ?")
			args = append(args, *feedID)
			conditions = append(conditions, "feed_id IN (SELECT id FROM feeds WHERE user_id = ?)")
			args = append(args, userID)
		} else if categoryID != nil {
			conditions = append(conditions, "feed_id IN (SELECT id FROM feeds WHERE user_id = ? AND category_id = ?)")
			args = append(args, userID, *categoryID)
		} else {
			conditions = append(conditions, "feed_id IN (SELECT id FROM feeds WHERE user_id = ?)")
			args = append(args, userID)
		}

		query := fmt.Sprintf(`UPDATE articles SET is_read = 1, read_at = CURRENT_TIMESTAMP WHERE %s`,
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
				SELECT id FROM (
					SELECT a.id,
						ROW_NUMBER() OVER (PARTITION BY a.feed_id
							ORDER BY COALESCE(a.published_at, a.fetched_at) DESC) AS rn
					FROM articles a
					JOIN feeds f ON a.feed_id = f.id
					WHERE %s
					AND a.is_read = 0
				) WHERE rn <= ?
			)`, feedCondition, feedCondition)

		// Double the args for the two subqueries, plus count
		allArgs := make([]interface{}, 0, len(args)*2+1)
		allArgs = append(allArgs, args...)
		allArgs = append(allArgs, args...)
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
	// Use reading_events (actual reads with duration > 5s) instead of
	// read_at timestamps which include bulk mark-all-as-read operations.
	row := q.db.QueryRow(`
		SELECT
			COUNT(DISTINCT re.article_id) as articles_read,
			COALESCE(SUM(re.duration_seconds) / 60, 0) as total_minutes,
			COUNT(DISTINCT a.feed_id) as feeds_read
		FROM reading_events re
		JOIN articles a ON a.id = re.article_id
		JOIN feeds f ON f.id = a.feed_id AND f.user_id = ?
		WHERE re.event_type = 'read'
		AND re.duration_seconds > 5
		AND re.created_at >= datetime('now', '-7 days')
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

// ArticleBelongsToUser checks article ownership without fetching full content.
func (q *Queries) ArticleBelongsToUser(articleID, userID int64) (bool, error) {
	var exists int
	err := q.db.QueryRow(
		"SELECT 1 FROM articles a JOIN feeds f ON a.feed_id = f.id WHERE a.id = ? AND f.user_id = ? LIMIT 1",
		articleID, userID,
	).Scan(&exists)
	if err != nil {
		return false, err
	}
	return true, nil
}

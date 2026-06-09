package store

import "strings"

// Passage is a single retrieved article excerpt used to ground "Ask Your
// Feeds" answers. It is deliberately decoupled from the claude package: the
// API handler maps store.Passage to claude.Passage so neither package depends
// on the other.
type Passage struct {
	ArticleID int64
	Title     string
	URL       string
	FeedTitle string
	Excerpt   string
}

// SearchPassages retrieves up to limit relevant article passages from the
// given user's own feeds for a free-text query, used for conversational RAG.
//
// When FTS5 is available it ranks by relevance and returns a query-centered
// snippet over the indexed content. Otherwise it falls back to a LIKE
// substring match ordered newest-first, building the excerpt from the article
// body via makeSnippet. A non-matching query yields an empty (non-nil-or-nil)
// slice rather than an error.
func (q *Queries) SearchPassages(userID int64, query string, limit int) ([]Passage, error) {
	if limit <= 0 {
		limit = 8
	}

	if ftsMatch := buildFTSMatchOR(query); FTS5Enabled && ftsMatch != "" {
		rows, err := q.db.Query(`
			SELECT a.id, COALESCE(a.title,''), COALESCE(a.url,''), COALESCE(f.title,''),
			       snippet(articles_fts, 1, '', '', ' … ', 64)
			FROM articles_fts
			JOIN articles a ON a.id = articles_fts.rowid
			JOIN feeds f ON a.feed_id = f.id
			WHERE articles_fts MATCH ? AND f.user_id = ?
			ORDER BY rank
			LIMIT ?`, ftsMatch, userID, limit)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		var passages []Passage
		for rows.Next() {
			var p Passage
			if err := rows.Scan(&p.ArticleID, &p.Title, &p.URL, &p.FeedTitle, &p.Excerpt); err != nil {
				return nil, err
			}
			p.Excerpt = strings.TrimSpace(p.Excerpt)
			passages = append(passages, p)
		}
		return passages, rows.Err()
	}

	// LIKE fallback: no FTS5 (or query had nothing searchable). Match articles
	// containing ANY topical term (stopwords dropped via topicalTerms, exactly
	// like the FTS OR-recall path) across title and both content columns,
	// newest-first. Using the raw query as a single substring would require the
	// whole natural-language question to appear verbatim and so retrieve
	// nothing — the topical-term split is what lets "what have my feeds said
	// about Rust lately?" still surface the Rust article.
	terms := topicalTerms(query)
	if len(terms) == 0 {
		return nil, nil
	}
	escaper := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	const likeClause = `(a.title LIKE ? ESCAPE '\' OR a.content_clean LIKE ? ESCAPE '\' OR a.content_raw LIKE ? ESCAPE '\')`
	clauses := make([]string, len(terms))
	args := make([]any, 0, len(terms)*3+2)
	args = append(args, userID)
	for i, t := range terms {
		clauses[i] = likeClause
		like := "%" + escaper.Replace(t) + "%"
		args = append(args, like, like, like)
	}
	args = append(args, limit)
	rows, err := q.db.Query(`
		SELECT a.id, COALESCE(a.title,''), COALESCE(a.url,''), COALESCE(f.title,''),
		       COALESCE(a.content_clean, a.content_raw, '')
		FROM articles a
		JOIN feeds f ON a.feed_id = f.id
		WHERE f.user_id = ?
		  AND (`+strings.Join(clauses, " OR ")+`)
		ORDER BY COALESCE(a.published_at, a.fetched_at) DESC
		LIMIT ?`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var passages []Passage
	for rows.Next() {
		var p Passage
		var content string
		if err := rows.Scan(&p.ArticleID, &p.Title, &p.URL, &p.FeedTitle, &content); err != nil {
			return nil, err
		}
		p.Excerpt = makeSnippet(content, 480)
		passages = append(passages, p)
	}
	return passages, rows.Err()
}

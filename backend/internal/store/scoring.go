package store

import (
	"database/sql"
	"time"

	"github.com/feednest/backend/internal/scorer"
)

// sqliteTimestampFormats mirrors the layouts go-sqlite3 uses to (de)serialize
// time.Time values. The driver only auto-parses columns whose declared type is
// DATETIME; a COALESCE(...) expression has no declared type and comes back as a
// raw string, so we parse it with the same layouts the driver would.
var sqliteTimestampFormats = []string{
	"2006-01-02 15:04:05.999999999-07:00",
	"2006-01-02T15:04:05.999999999-07:00",
	"2006-01-02 15:04:05.999999999",
	"2006-01-02T15:04:05.999999999",
	"2006-01-02 15:04:05",
	"2006-01-02T15:04:05",
	"2006-01-02 15:04",
	"2006-01-02T15:04",
	"2006-01-02",
}

// parseSQLiteTime parses a SQLite datetime string (as returned by a COALESCE
// expression) into a time.Time, trying the driver's known layouts in order.
func parseSQLiteTime(s string) (time.Time, error) {
	var firstErr error
	for _, layout := range sqliteTimestampFormats {
		// CURRENT_TIMESTAMP values are UTC and carry no zone, so parse the
		// zone-less layouts in UTC to match the driver.
		t, err := time.ParseInLocation(layout, s, time.UTC)
		if err == nil {
			return t, nil
		}
		if firstErr == nil {
			firstErr = err
		}
	}
	return time.Time{}, firstErr
}

// computeFeedEngagement derives a per-feed engagement value in [0,1] from
// aggregated reading behaviour:
//
//	readRate   = reads / (reads + dismisses + 1)
//	depth      = min(avgReadSeconds / 180, 1)
//	clickBonus = min(clicks / (reads + 1), 1) * 0.2
//	engagement = min(0.6*readRate + 0.2*depth + clickBonus, 1)
//
// A feed with no activity yields 0. The result is clamped to [0,1].
func computeFeedEngagement(reads, dismisses, clicks int, avgReadSeconds float64) float64 {
	readRate := float64(reads) / float64(reads+dismisses+1)

	depth := avgReadSeconds / 180.0
	if depth > 1.0 {
		depth = 1.0
	}

	clickBonus := float64(clicks) / float64(reads+1)
	if clickBonus > 1.0 {
		clickBonus = 1.0
	}
	clickBonus *= 0.2

	engagement := 0.6*readRate + 0.2*depth + clickBonus
	if engagement > 1.0 {
		engagement = 1.0
	}
	if engagement < 0.0 {
		engagement = 0.0
	}
	return engagement
}

// feedEngagementAgg holds the per-feed reading-behaviour aggregates used to
// compute engagement.
type feedEngagementAgg struct {
	feedID         int64
	reads          int
	dismisses      int
	clicks         int
	avgReadSeconds float64
}

// UpdateFeedEngagementScores recomputes feeds.engagement_score for every feed
// from the last 90 days of reading_events. Event counts are aggregated per feed
// in a single GROUP BY query; engagement is computed in Go and written back in
// one transaction. Feeds with no recent events keep their existing score
// (default 0.0).
func (q *Queries) UpdateFeedEngagementScores() error {
	// Aggregate reads/dismisses/clicks and the average healthy read duration
	// per feed in a single pass. 'read' durations are only meaningful between
	// 15 and 1800 seconds; NULL durations are treated as 0 via COALESCE.
	rows, err := q.db.Query(`
		SELECT a.feed_id,
			SUM(CASE WHEN re.event_type = 'read' THEN 1 ELSE 0 END) AS reads,
			SUM(CASE WHEN re.event_type = 'dismiss' THEN 1 ELSE 0 END) AS dismisses,
			SUM(CASE WHEN re.event_type = 'click' THEN 1 ELSE 0 END) AS clicks,
			COALESCE(AVG(CASE
				WHEN re.event_type = 'read' AND re.duration_seconds BETWEEN 15 AND 1800
				THEN re.duration_seconds END), 0) AS avg_read_seconds
		FROM reading_events re
		JOIN articles a ON a.id = re.article_id
		WHERE re.created_at >= datetime('now', '-90 days')
		GROUP BY a.feed_id`)
	if err != nil {
		return err
	}
	defer rows.Close()

	var aggs []feedEngagementAgg
	for rows.Next() {
		var agg feedEngagementAgg
		if err := rows.Scan(&agg.feedID, &agg.reads, &agg.dismisses, &agg.clicks, &agg.avgReadSeconds); err != nil {
			return err
		}
		aggs = append(aggs, agg)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if len(aggs) == 0 {
		return nil
	}

	tx, err := q.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`UPDATE feeds SET engagement_score = ? WHERE id = ?`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, agg := range aggs {
		engagement := computeFeedEngagement(agg.reads, agg.dismisses, agg.clicks, agg.avgReadSeconds)
		if _, err := stmt.Exec(engagement, agg.feedID); err != nil {
			return err
		}
	}

	return tx.Commit()
}

// recentArticleScore carries the data needed to score one article.
type recentArticleScore struct {
	id          int64
	publishedAt time.Time
	engagement  float64
}

// ScoreRecentArticles recomputes articles.score for articles published (or, if
// unpublished, fetched) within the last 30 days, using the owning feed's
// engagement_score and the recency-weighted scorer. All updates run in a single
// transaction. Returns the number of articles updated.
//
// Note: UpdateFeedEngagementScores must run before this so article scores read
// fresh feeds.engagement_score values.
func (q *Queries) ScoreRecentArticles() (int, error) {
	rows, err := q.db.Query(`
		SELECT a.id, COALESCE(a.published_at, a.fetched_at) AS ts, f.engagement_score
		FROM articles a
		JOIN feeds f ON a.feed_id = f.id
		WHERE COALESCE(a.published_at, a.fetched_at) >= datetime('now', '-30 days')`)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	var toScore []recentArticleScore
	for rows.Next() {
		var rec recentArticleScore
		var ts string
		var engagement sql.NullFloat64
		// COALESCE(published_at, fetched_at) has no declared column type, so the
		// driver returns it as a raw datetime string; parse it ourselves.
		if err := rows.Scan(&rec.id, &ts, &engagement); err != nil {
			return 0, err
		}
		publishedAt, err := parseSQLiteTime(ts)
		if err != nil {
			return 0, err
		}
		rec.publishedAt = publishedAt
		if engagement.Valid {
			rec.engagement = engagement.Float64
		}
		toScore = append(toScore, rec)
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}
	if len(toScore) == 0 {
		return 0, nil
	}

	tx, err := q.db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`UPDATE articles SET score = ? WHERE id = ?`)
	if err != nil {
		return 0, err
	}
	defer stmt.Close()

	for _, rec := range toScore {
		score := scorer.CalculateScore(rec.publishedAt, rec.engagement)
		if _, err := stmt.Exec(score, rec.id); err != nil {
			return 0, err
		}
	}

	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return len(toScore), nil
}

package store

import (
	"testing"
	"time"
)

func TestComputeFeedEngagement_NoActivity(t *testing.T) {
	got := computeFeedEngagement(0, 0, 0, 0)
	if got != 0 {
		t.Errorf("no activity: expected 0, got %v", got)
	}
}

func TestComputeFeedEngagement_ReadsOnlyPositive(t *testing.T) {
	got := computeFeedEngagement(10, 0, 0, 60)
	if got <= 0 {
		t.Errorf("reads only: expected positive, got %v", got)
	}
}

func TestComputeFeedEngagement_DismissesLowerScore(t *testing.T) {
	withoutDismiss := computeFeedEngagement(10, 0, 0, 60)
	withDismiss := computeFeedEngagement(10, 10, 0, 60)
	if withDismiss >= withoutDismiss {
		t.Errorf("adding dismisses should lower engagement: without=%v with=%v", withoutDismiss, withDismiss)
	}
}

func TestComputeFeedEngagement_ClicksAddBoundedBonus(t *testing.T) {
	withoutClicks := computeFeedEngagement(10, 0, 0, 60)
	withClicks := computeFeedEngagement(10, 0, 5, 60)
	if withClicks <= withoutClicks {
		t.Errorf("clicks should raise engagement: without=%v with=%v", withoutClicks, withClicks)
	}
	// clickBonus is min(clicks/(reads+1),1)*0.2 — capped at 0.2 above the base.
	diff := withClicks - withoutClicks
	if diff > 0.2+1e-9 {
		t.Errorf("click bonus should be bounded by 0.2, got delta %v", diff)
	}
	// Even with a huge click count, the bonus stays bounded.
	maxClicks := computeFeedEngagement(10, 0, 1000000, 60)
	if maxClicks-withoutClicks > 0.2+1e-9 {
		t.Errorf("click bonus should remain bounded by 0.2 even for many clicks, got delta %v", maxClicks-withoutClicks)
	}
}

func TestComputeFeedEngagement_MonotonicInReads(t *testing.T) {
	prev := computeFeedEngagement(0, 5, 0, 60)
	for reads := 1; reads <= 50; reads++ {
		cur := computeFeedEngagement(reads, 5, 0, 60)
		if cur < prev-1e-9 {
			t.Errorf("engagement should be non-decreasing in reads: reads=%d prev=%v cur=%v", reads, prev, cur)
		}
		prev = cur
	}
}

func TestComputeFeedEngagement_AlwaysWithinUnitRange(t *testing.T) {
	cases := []struct {
		reads, dismisses, clicks int
		avgRead                  float64
	}{
		{0, 0, 0, 0},
		{1000, 0, 1000, 1800},
		{1, 1000, 0, 0},
		{10, 10, 10, 180},
		{1, 0, 1000000, 100000},
		{0, 0, 5, 0},
	}
	for _, c := range cases {
		got := computeFeedEngagement(c.reads, c.dismisses, c.clicks, c.avgRead)
		if got < 0 || got > 1 {
			t.Errorf("engagement out of [0,1] for %+v: got %v", c, got)
		}
	}
}

// TestScoringIntegration exercises the full DB path: feed A accumulates 'read'
// events, feed B accumulates 'dismiss' events; after updating engagement and
// scoring recent articles, feed A should out-rank feed B and article scores
// should have moved above the default 0.0.
func TestScoringIntegration(t *testing.T) {
	q := setupTestDB(t)
	userID := createTestUser(t, q)

	feedA, err := q.CreateFeed(userID, "https://a.example.com/rss", "Feed A", "", "", nil)
	if err != nil {
		t.Fatalf("create feed A failed: %v", err)
	}
	feedB, err := q.CreateFeed(userID, "https://b.example.com/rss", "Feed B", "", "", nil)
	if err != nil {
		t.Fatalf("create feed B failed: %v", err)
	}

	now := time.Now()
	// Recent articles (within the 30-day scoring window).
	if err := q.CreateArticle(feedA.ID, "a-1", "A One", "https://a.example.com/1", "", "", "", "", &now, 500, 3); err != nil {
		t.Fatalf("create article A1 failed: %v", err)
	}
	if err := q.CreateArticle(feedA.ID, "a-2", "A Two", "https://a.example.com/2", "", "", "", "", &now, 500, 3); err != nil {
		t.Fatalf("create article A2 failed: %v", err)
	}
	if err := q.CreateArticle(feedB.ID, "b-1", "B One", "https://b.example.com/1", "", "", "", "", &now, 500, 3); err != nil {
		t.Fatalf("create article B1 failed: %v", err)
	}

	articleAID := articleIDByGUID(t, q, feedA.ID, "a-1")
	articleBID := articleIDByGUID(t, q, feedB.ID, "b-1")

	// Feed A: positive engagement — many reads with healthy durations + clicks.
	for i := 0; i < 10; i++ {
		if err := q.CreateReadingEvent(articleAID, "read", 120); err != nil {
			t.Fatalf("create read event failed: %v", err)
		}
	}
	for i := 0; i < 3; i++ {
		if err := q.CreateReadingEvent(articleAID, "click", 0); err != nil {
			t.Fatalf("create click event failed: %v", err)
		}
	}

	// Feed B: negative engagement — only dismisses.
	for i := 0; i < 10; i++ {
		if err := q.CreateReadingEvent(articleBID, "dismiss", 0); err != nil {
			t.Fatalf("create dismiss event failed: %v", err)
		}
	}

	if err := q.UpdateFeedEngagementScores(); err != nil {
		t.Fatalf("UpdateFeedEngagementScores failed: %v", err)
	}

	gotA, err := q.GetFeed(feedA.ID, userID)
	if err != nil {
		t.Fatalf("get feed A failed: %v", err)
	}
	gotB, err := q.GetFeed(feedB.ID, userID)
	if err != nil {
		t.Fatalf("get feed B failed: %v", err)
	}
	if gotA.EngagementScore <= gotB.EngagementScore {
		t.Errorf("expected feed A engagement (%v) > feed B engagement (%v)", gotA.EngagementScore, gotB.EngagementScore)
	}
	if gotA.EngagementScore <= 0 {
		t.Errorf("expected feed A engagement > 0, got %v", gotA.EngagementScore)
	}

	updated, err := q.ScoreRecentArticles()
	if err != nil {
		t.Fatalf("ScoreRecentArticles failed: %v", err)
	}
	if updated < 3 {
		t.Errorf("expected at least 3 articles scored, got %d", updated)
	}

	// Recent articles should now have a non-zero score (recency component alone
	// guarantees this for just-published articles).
	if score := articleScore(t, q, articleAID); score <= 0 {
		t.Errorf("expected article A score > 0 after scoring, got %v", score)
	}
}

func articleIDByGUID(t *testing.T, q *Queries, feedID int64, guid string) int64 {
	t.Helper()
	var id int64
	if err := q.db.QueryRow("SELECT id FROM articles WHERE feed_id = ? AND guid = ?", feedID, guid).Scan(&id); err != nil {
		t.Fatalf("lookup article id for guid %q failed: %v", guid, err)
	}
	return id
}

func articleScore(t *testing.T, q *Queries, articleID int64) float64 {
	t.Helper()
	var score float64
	if err := q.db.QueryRow("SELECT score FROM articles WHERE id = ?", articleID).Scan(&score); err != nil {
		t.Fatalf("lookup score for article %d failed: %v", articleID, err)
	}
	return score
}

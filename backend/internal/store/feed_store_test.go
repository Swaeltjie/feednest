package store

import "testing"

func TestFeedStore_CRUD(t *testing.T) {
	q := setupTestDB(t)
	userID := createTestUser(t, q)

	feed, err := q.CreateFeed(userID, "https://example.com/rss", "Example Feed", "https://example.com", "", nil)
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}
	if feed.Title != "Example Feed" {
		t.Errorf("expected 'Example Feed', got %q", feed.Title)
	}

	feeds, err := q.ListFeeds(userID)
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if len(feeds) != 1 {
		t.Fatalf("expected 1 feed, got %d", len(feeds))
	}

	err = q.DeleteFeed(feed.ID, userID)
	if err != nil {
		t.Fatalf("delete failed: %v", err)
	}
	remaining, _ := q.ListFeeds(userID)
	if len(remaining) != 0 {
		t.Errorf("expected 0 feeds, got %d", len(remaining))
	}
}

func TestFeedStore_DuplicateURL(t *testing.T) {
	q := setupTestDB(t)
	userID := createTestUser(t, q)

	q.CreateFeed(userID, "https://example.com/rss", "Feed 1", "", "", nil)
	_, err := q.CreateFeed(userID, "https://example.com/rss", "Feed 2", "", "", nil)
	if err == nil {
		t.Error("expected error for duplicate URL")
	}
}

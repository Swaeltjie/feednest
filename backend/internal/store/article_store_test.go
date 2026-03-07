package store

import (
	"testing"
	"time"
)

func TestArticleStore_CreateAndList(t *testing.T) {
	q := setupTestDB(t)
	userID := createTestUser(t, q)
	feed, _ := q.CreateFeed(userID, "https://example.com/rss", "Test Feed", "", "", nil)

	now := time.Now()
	err := q.CreateArticle(feed.ID, "guid-1", "Article 1", "https://example.com/1", "Author", "<p>raw</p>", "<p>clean</p>", "https://img.com/1.jpg", &now, 200, 1)
	if err != nil {
		t.Fatalf("create article failed: %v", err)
	}

	err = q.CreateArticle(feed.ID, "guid-1", "Article 1 Dup", "", "", "", "", "", nil, 0, 0)
	if err != nil {
		t.Fatalf("expected upsert to not fail: %v", err)
	}

	articles, total, err := q.ListArticles(userID, &ArticleFilter{Limit: 30, Page: 1, Sort: "newest"})
	if err != nil {
		t.Fatalf("list articles failed: %v", err)
	}
	if total != 1 {
		t.Errorf("expected 1 article, got %d", total)
	}
	if len(articles) != 1 {
		t.Fatalf("expected 1 article in slice, got %d", len(articles))
	}
	if articles[0].Title != "Article 1" {
		t.Errorf("expected 'Article 1', got %q", articles[0].Title)
	}
}

func TestMakeSnippet(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		maxLen int
	}{
		{"strips HTML", "<p>Hello <b>world</b></p>", 100},
		{"empty input", "", 100},
		{"collapses whitespace", "hello   \n  world", 100},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := makeSnippet(tt.input, tt.maxLen)
			if len(got) > tt.maxLen+5 {
				t.Errorf("snippet too long: got %d chars, max %d", len(got), tt.maxLen)
			}
		})
	}
}

func TestArticleStore_MarkReadAndStar(t *testing.T) {
	q := setupTestDB(t)
	userID := createTestUser(t, q)
	feed, _ := q.CreateFeed(userID, "https://example.com/rss", "Feed", "", "", nil)

	now := time.Now()
	q.CreateArticle(feed.ID, "guid-1", "Article", "https://example.com/1", "", "", "", "", &now, 100, 1)

	articles, _, err := q.ListArticles(userID, &ArticleFilter{Limit: 30, Page: 1, Sort: "newest"})
	if err != nil {
		t.Fatalf("list articles failed: %v", err)
	}
	if len(articles) == 0 {
		t.Fatal("expected at least 1 article")
	}
	articleID := articles[0].ID

	isRead := true
	err = q.UpdateArticle(articleID, userID, &isRead, nil)
	if err != nil {
		t.Fatalf("mark read failed: %v", err)
	}

	article, err := q.GetArticle(articleID, userID)
	if err != nil {
		t.Fatalf("get article failed: %v", err)
	}
	if !article.IsRead {
		t.Error("expected article to be read")
	}
}

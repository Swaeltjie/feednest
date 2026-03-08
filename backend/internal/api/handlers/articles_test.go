package handlers_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/feednest/backend/internal/api/handlers"
	"github.com/feednest/backend/internal/models"
	"github.com/feednest/backend/internal/store"
)

func TestArticleHandler_List(t *testing.T) {
	q := setupTestDB(t)
	userID := createTestUser(t, q)
	feed, _ := q.CreateFeed(userID, "https://example.com/rss", "Test", "", "", nil)
	now := time.Now()
	q.CreateArticle(feed.ID, "g1", "Article 1", "https://example.com/1", "Author", "<p>raw</p>", "<p>clean</p>", "", &now, 200, 1)

	h := handlers.NewArticleHandler(q)
	req := authenticatedRequest("GET", "/api/articles?status=unread&sort=newest", "", userID)
	rr := httptest.NewRecorder()
	h.List(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp struct {
		Articles []models.Article `json:"articles"`
		Total    int              `json:"total"`
	}
	json.NewDecoder(rr.Body).Decode(&resp)
	if resp.Total != 1 {
		t.Errorf("expected total=1, got %d", resp.Total)
	}
	if len(resp.Articles) != 1 {
		t.Fatalf("expected 1 article, got %d", len(resp.Articles))
	}
	if resp.Articles[0].Title != "Article 1" {
		t.Errorf("expected 'Article 1', got %q", resp.Articles[0].Title)
	}
}

func TestArticleHandler_List_Pagination(t *testing.T) {
	q := setupTestDB(t)
	userID := createTestUser(t, q)
	feed, _ := q.CreateFeed(userID, "https://example.com/rss", "Test", "", "", nil)
	now := time.Now()
	for i := 0; i < 5; i++ {
		guid := "g" + formatID(int64(i))
		url := "https://example.com/" + formatID(int64(i))
		q.CreateArticle(feed.ID, guid, "Article", url, "", "", "", "", &now, 100, 1)
	}

	h := handlers.NewArticleHandler(q)
	req := authenticatedRequest("GET", "/api/articles?page=1&limit=2&sort=newest", "", userID)
	rr := httptest.NewRecorder()
	h.List(rr, req)

	var resp struct {
		Articles []models.Article `json:"articles"`
		Total    int              `json:"total"`
		Page     int              `json:"page"`
		Limit    int              `json:"limit"`
	}
	json.NewDecoder(rr.Body).Decode(&resp)
	if resp.Total != 5 {
		t.Errorf("expected total=5, got %d", resp.Total)
	}
	if len(resp.Articles) != 2 {
		t.Errorf("expected 2 articles on page 1, got %d", len(resp.Articles))
	}
}

func TestArticleHandler_Update(t *testing.T) {
	q := setupTestDB(t)
	userID := createTestUser(t, q)
	feed, _ := q.CreateFeed(userID, "https://example.com/rss", "Test", "", "", nil)
	now := time.Now()
	q.CreateArticle(feed.ID, "g1", "Article 1", "https://example.com/1", "", "", "", "", &now, 200, 1)

	filter := &store.ArticleFilter{
		Status: "unread",
		Sort:   "newest",
		Page:   1,
		Limit:  30,
	}
	articles, _, _ := q.ListArticles(userID, filter)
	articleID := articles[0].ID

	h := handlers.NewArticleHandler(q)
	body := `{"is_read": true}`
	req := authenticatedRequest("PUT", "/api/articles/"+formatID(articleID), body, userID)
	req = withChiURLParam(req, "id", formatID(articleID))
	rr := httptest.NewRecorder()
	h.Update(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestArticleHandler_Bulk(t *testing.T) {
	q := setupTestDB(t)
	userID := createTestUser(t, q)
	feed, _ := q.CreateFeed(userID, "https://example.com/rss", "Test", "", "", nil)
	now := time.Now()
	q.CreateArticle(feed.ID, "g1", "A1", "https://example.com/1", "", "", "", "", &now, 200, 1)
	q.CreateArticle(feed.ID, "g2", "A2", "https://example.com/2", "", "", "", "", &now, 200, 1)

	filter := &store.ArticleFilter{
		Status: "unread",
		Sort:   "newest",
		Page:   1,
		Limit:  30,
	}
	articles, _, _ := q.ListArticles(userID, filter)
	ids := make([]int64, len(articles))
	for i, a := range articles {
		ids[i] = a.ID
	}

	h := handlers.NewArticleHandler(q)
	bodyBytes, _ := json.Marshal(map[string]interface{}{
		"article_ids": ids,
		"action":      "mark_read",
	})
	req := authenticatedRequest("POST", "/api/articles/bulk", string(bodyBytes), userID)
	rr := httptest.NewRecorder()
	h.Bulk(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestArticleHandler_Bulk_InvalidAction(t *testing.T) {
	q := setupTestDB(t)
	userID := createTestUser(t, q)

	h := handlers.NewArticleHandler(q)
	body := `{"article_ids": [1], "action": "invalid"}`
	req := authenticatedRequest("POST", "/api/articles/bulk", body, userID)
	rr := httptest.NewRecorder()
	h.Bulk(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

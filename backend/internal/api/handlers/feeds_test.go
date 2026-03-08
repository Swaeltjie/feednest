package handlers_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/feednest/backend/internal/api/handlers"
	"github.com/feednest/backend/internal/models"
)

func TestFeedHandler_List_Empty(t *testing.T) {
	q := setupTestDB(t)
	userID := createTestUser(t, q)

	h := handlers.NewFeedHandler(q, nil)
	req := authenticatedRequest("GET", "/api/feeds", "", userID)
	rr := httptest.NewRecorder()
	h.List(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	var feeds []models.Feed
	json.NewDecoder(rr.Body).Decode(&feeds)
	if len(feeds) != 0 {
		t.Errorf("expected empty feeds, got %d", len(feeds))
	}
}

func TestFeedHandler_Create(t *testing.T) {
	q := setupTestDB(t)
	userID := createTestUser(t, q)

	h := handlers.NewFeedHandler(q, nil)
	body := `{"url": "https://example.com/rss"}`
	req := authenticatedRequest("POST", "/api/feeds", body, userID)
	rr := httptest.NewRecorder()
	h.Create(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}

	var feed models.Feed
	json.NewDecoder(rr.Body).Decode(&feed)
	if feed.URL != "https://example.com/rss" {
		t.Errorf("expected URL 'https://example.com/rss', got %q", feed.URL)
	}
}

func TestFeedHandler_Create_MissingURL(t *testing.T) {
	q := setupTestDB(t)
	userID := createTestUser(t, q)

	h := handlers.NewFeedHandler(q, nil)
	body := `{}`
	req := authenticatedRequest("POST", "/api/feeds", body, userID)
	rr := httptest.NewRecorder()
	h.Create(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestFeedHandler_Delete(t *testing.T) {
	q := setupTestDB(t)
	userID := createTestUser(t, q)
	feed, _ := q.CreateFeed(userID, "https://example.com/rss", "Test", "", "", nil)

	h := handlers.NewFeedHandler(q, nil)
	req := authenticatedRequest("DELETE", "/api/feeds/"+formatID(feed.ID), "", userID)
	req = withChiURLParam(req, "id", formatID(feed.ID))
	rr := httptest.NewRecorder()
	h.Delete(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", rr.Code, rr.Body.String())
	}

	feeds, _ := q.ListFeeds(userID)
	if len(feeds) != 0 {
		t.Errorf("expected 0 feeds after delete, got %d", len(feeds))
	}
}

func TestFeedHandler_Create_DuplicateURL(t *testing.T) {
	q := setupTestDB(t)
	userID := createTestUser(t, q)

	h := handlers.NewFeedHandler(q, nil)

	body := `{"url": "https://example.com/rss"}`
	req := authenticatedRequest("POST", "/api/feeds", body, userID)
	rr := httptest.NewRecorder()
	h.Create(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("first create: expected 201, got %d", rr.Code)
	}

	req = authenticatedRequest("POST", "/api/feeds", body, userID)
	rr = httptest.NewRecorder()
	h.Create(rr, req)
	if rr.Code != http.StatusConflict {
		t.Fatalf("duplicate create: expected 409, got %d", rr.Code)
	}
}

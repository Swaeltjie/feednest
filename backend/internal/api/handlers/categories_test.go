package handlers_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/feednest/backend/internal/api/handlers"
	"github.com/feednest/backend/internal/models"
)

func TestCategoryHandler_List_Empty(t *testing.T) {
	q := setupTestDB(t)
	userID := createTestUser(t, q)

	h := handlers.NewCategoryHandler(q)
	req := authenticatedRequest("GET", "/api/categories", "", userID)
	rr := httptest.NewRecorder()
	h.List(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	var cats []models.Category
	json.NewDecoder(rr.Body).Decode(&cats)
	if len(cats) != 0 {
		t.Errorf("expected empty categories, got %d", len(cats))
	}
}

func TestCategoryHandler_CreateAndList(t *testing.T) {
	q := setupTestDB(t)
	userID := createTestUser(t, q)

	h := handlers.NewCategoryHandler(q)

	body := `{"name": "Tech"}`
	req := authenticatedRequest("POST", "/api/categories", body, userID)
	rr := httptest.NewRecorder()
	h.Create(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}

	req = authenticatedRequest("GET", "/api/categories", "", userID)
	rr = httptest.NewRecorder()
	h.List(rr, req)

	var cats []models.Category
	json.NewDecoder(rr.Body).Decode(&cats)
	if len(cats) != 1 {
		t.Fatalf("expected 1 category, got %d", len(cats))
	}
	if cats[0].Name != "Tech" {
		t.Errorf("expected 'Tech', got %q", cats[0].Name)
	}
}

func TestCategoryHandler_Delete(t *testing.T) {
	q := setupTestDB(t)
	userID := createTestUser(t, q)
	cat, _ := q.CreateCategory(userID, "ToDelete", 0)

	h := handlers.NewCategoryHandler(q)
	req := authenticatedRequest("DELETE", "/api/categories/"+formatID(cat.ID), "", userID)
	req = withChiURLParam(req, "id", formatID(cat.ID))
	rr := httptest.NewRecorder()
	h.Delete(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", rr.Code, rr.Body.String())
	}
}

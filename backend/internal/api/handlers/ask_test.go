package handlers_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/feednest/backend/internal/api/handlers"
	"github.com/feednest/backend/internal/claude"
)

func TestAsk_RejectsEmptyQuestion(t *testing.T) {
	q := setupTestDB(t)
	userID := createTestUser(t, q)
	h := handlers.NewAskHandler(q, claude.New())

	req := authenticatedRequest(http.MethodPost, "/api/ask", `{"question":"   "}`, userID)
	rr := httptest.NewRecorder()
	h.Ask(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for empty question, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestAsk_RejectsTooLongQuestion(t *testing.T) {
	q := setupTestDB(t)
	userID := createTestUser(t, q)
	h := handlers.NewAskHandler(q, claude.New())

	long := strings.Repeat("a", 1001)
	req := authenticatedRequest(http.MethodPost, "/api/ask", `{"question":"`+long+`"}`, userID)
	rr := httptest.NewRecorder()
	h.Ask(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for too-long question, got %d", rr.Code)
	}
}

func TestAsk_RejectsInvalidBody(t *testing.T) {
	q := setupTestDB(t)
	userID := createTestUser(t, q)
	h := handlers.NewAskHandler(q, claude.New())

	req := authenticatedRequest(http.MethodPost, "/api/ask", `not json`, userID)
	rr := httptest.NewRecorder()
	h.Ask(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid body, got %d", rr.Code)
	}
}

func TestAsk_ServiceUnavailableWhenDisabled(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "")
	t.Setenv("CLAUDE_AUTH_MODE", "")
	t.Setenv("CLAUDE_CREDENTIALS_PATH", "/nonexistent/creds.json")

	q := setupTestDB(t)
	userID := createTestUser(t, q)
	c := claude.New()
	if c.Enabled() {
		t.Skip("claude client unexpectedly enabled in this environment")
	}
	h := handlers.NewAskHandler(q, c)

	req := authenticatedRequest(http.MethodPost, "/api/ask", `{"question":"what's new in rust?"}`, userID)
	rr := httptest.NewRecorder()
	h.Ask(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 when claude disabled, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestAsk_NoPassagesReturnsFallback(t *testing.T) {
	// An enabled client (via API key) with no matching articles must return the
	// fallback answer WITHOUT making any network call, because the handler
	// short-circuits on zero passages before invoking Answer.
	t.Setenv("ANTHROPIC_API_KEY", "test-key")
	t.Setenv("CLAUDE_AUTH_MODE", "apikey")

	q := setupTestDB(t)
	userID := createTestUser(t, q)
	c := claude.New()
	if !c.Enabled() {
		t.Skip("claude client not enabled despite API key")
	}
	h := handlers.NewAskHandler(q, c)

	req := authenticatedRequest(http.MethodPost, "/api/ask", `{"question":"zzzznonexistentterm"}`, userID)
	rr := httptest.NewRecorder()
	h.Ask(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 for no-passage fallback, got %d: %s", rr.Code, rr.Body.String())
	}
	var resp struct {
		Answer  string `json:"answer"`
		Sources []struct {
			ID int64 `json:"id"`
		} `json:"sources"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Answer == "" {
		t.Errorf("expected non-empty fallback answer")
	}
	if len(resp.Sources) != 0 {
		t.Errorf("expected empty sources, got %d", len(resp.Sources))
	}
}

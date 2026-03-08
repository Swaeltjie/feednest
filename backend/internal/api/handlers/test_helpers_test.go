package handlers_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/feednest/backend/internal/apiutil"
	"github.com/feednest/backend/internal/store"
	"github.com/feednest/backend/internal/urlutil"
)

func setupTestDB(t *testing.T) *store.Queries {
	t.Helper()
	urlutil.AllowPrivate = true
	t.Cleanup(func() { urlutil.AllowPrivate = false })

	dir := t.TempDir()
	db, err := store.NewDB(dir + "/test.db")
	if err != nil {
		t.Fatalf("failed to init db: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return store.New(db)
}

func createTestUser(t *testing.T, q *store.Queries) int64 {
	t.Helper()
	user, err := q.CreateUser("testuser", "test@example.com", "hashedpassword")
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}
	return user.ID
}

func authenticatedRequest(method, path string, body string, userID int64) *http.Request {
	var req *http.Request
	if body != "" {
		req = httptest.NewRequest(method, path, strings.NewReader(body))
	} else {
		req = httptest.NewRequest(method, path, nil)
	}
	req.Header.Set("Content-Type", "application/json")
	ctx := apiutil.WithUserID(req.Context(), userID)
	return req.WithContext(ctx)
}

func withChiURLParam(r *http.Request, key, value string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add(key, value)
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

func formatID(id int64) string {
	return strconv.FormatInt(id, 10)
}

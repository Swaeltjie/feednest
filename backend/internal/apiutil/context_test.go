package apiutil

import (
	"context"
	"net/http/httptest"
	"testing"
)

func TestWithUserID_And_ExtractUserID(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	ctx := WithUserID(req.Context(), 42)
	req = req.WithContext(ctx)

	got := ExtractUserID(req)
	if got != 42 {
		t.Errorf("expected 42, got %d", got)
	}
}

func TestExtractUserID_Missing(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	got := ExtractUserID(req)
	if got != 0 {
		t.Errorf("expected 0 for missing user ID, got %d", got)
	}
}

func TestExtractUserID_WrongType(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	r := req.WithContext(context.WithValue(req.Context(), ContextKeyUserID, "not-an-int"))
	got := ExtractUserID(r)
	if got != 0 {
		t.Errorf("expected 0 for wrong type, got %d", got)
	}
}

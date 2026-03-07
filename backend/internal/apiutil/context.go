package apiutil

import (
	"context"
	"net/http"
)

type contextKey string

const ContextKeyUserID contextKey = "user_id"

func WithUserID(ctx context.Context, userID int64) context.Context {
	return context.WithValue(ctx, ContextKeyUserID, userID)
}

func ExtractUserID(r *http.Request) int64 {
	if id, ok := r.Context().Value(ContextKeyUserID).(int64); ok {
		return id
	}
	return 0
}

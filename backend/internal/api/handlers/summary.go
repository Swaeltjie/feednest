package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/feednest/backend/internal/apiutil"
	"github.com/feednest/backend/internal/claude"
	"github.com/feednest/backend/internal/store"
)

// SummaryHandler generates and caches short AI TL;DR summaries for articles.
type SummaryHandler struct {
	store  *store.Queries
	claude *claude.Client
}

func NewSummaryHandler(store *store.Queries, c *claude.Client) *SummaryHandler {
	return &SummaryHandler{store: store, claude: c}
}

// Config reports whether AI summarization is available so the frontend can
// show or hide the Summarize button. GET /api/summary/config
func (h *SummaryHandler) Config(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"enabled": h.claude.Enabled(),
		"mode":    h.claude.Mode(),
	})
}

// Summarize returns a cached summary or generates one. POST /api/articles/{id}/summary
func (h *SummaryHandler) Summarize(w http.ResponseWriter, r *http.Request) {
	userID := apiutil.ExtractUserID(r)
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, `{"error":"invalid id"}`, http.StatusBadRequest)
		return
	}

	article, err := h.store.GetArticle(id, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, `{"error":"article not found"}`, http.StatusNotFound)
		} else {
			http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		}
		return
	}

	// Serve cached summary if present.
	if article.Summary != "" {
		writeSummary(w, article.Summary, true)
		return
	}

	if !h.claude.Enabled() {
		http.Error(w, `{"error":"AI summaries are not configured"}`, http.StatusServiceUnavailable)
		return
	}

	content := article.ContentClean
	if content == "" {
		content = article.ContentRaw
	}

	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()

	summary, err := h.claude.Summarize(ctx, article.Title, content)
	if err != nil {
		switch {
		case errors.Is(err, claude.ErrDisabled):
			http.Error(w, `{"error":"AI summaries are not configured"}`, http.StatusServiceUnavailable)
		case errors.Is(err, context.DeadlineExceeded):
			http.Error(w, `{"error":"summary generation timed out"}`, http.StatusGatewayTimeout)
		case r.Context().Err() != nil:
			// Client disconnected — don't bother writing a body.
		default:
			http.Error(w, `{"error":"failed to generate summary"}`, http.StatusBadGateway)
		}
		return
	}

	if err := h.store.UpdateArticleSummary(id, summary); err != nil {
		// Non-fatal: still return the freshly generated summary.
		writeSummary(w, summary, false)
		return
	}
	writeSummary(w, summary, false)
}

func writeSummary(w http.ResponseWriter, summary string, cached bool) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"summary": summary,
		"cached":  cached,
	})
}

package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"net/http"
	"strings"
	"strconv"
	"time"
	"unicode/utf8"

	"github.com/go-chi/chi/v5"

	"github.com/feednest/backend/internal/apiutil"
	"github.com/feednest/backend/internal/models"
	"github.com/feednest/backend/internal/readability"
	"github.com/feednest/backend/internal/store"
)

type ArticleHandler struct {
	store *store.Queries
}

func NewArticleHandler(store *store.Queries) *ArticleHandler {
	return &ArticleHandler{store: store}
}

func countWordsFromHTML(s string) int {
	inTag := false
	var buf []rune
	for _, r := range s {
		if r == '<' {
			inTag = true
			buf = append(buf, ' ')
			continue
		}
		if r == '>' {
			inTag = false
			continue
		}
		if !inTag {
			buf = append(buf, r)
		}
	}
	return len(strings.Fields(string(buf)))
}

func (h *ArticleHandler) List(w http.ResponseWriter, r *http.Request) {
	userID := apiutil.ExtractUserID(r)

	filter := &store.ArticleFilter{
		Status: r.URL.Query().Get("status"),
		Sort:   r.URL.Query().Get("sort"),
		Tag:    r.URL.Query().Get("tag"),
		Page:   1,
		Limit:  30,
	}

	if filter.Sort == "" {
		filter.Sort = "smart"
	}

	if p := r.URL.Query().Get("page"); p != "" {
		if v, err := strconv.Atoi(p); err == nil && v > 0 {
			if v > 10000 {
				v = 10000
			}
			filter.Page = v
		}
	}
	if l := r.URL.Query().Get("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 && v <= 100 {
			filter.Limit = v
		}
	}
	if feedID := r.URL.Query().Get("feed"); feedID != "" {
		id, err := strconv.ParseInt(feedID, 10, 64)
		if err != nil {
			http.Error(w, `{"error":"invalid feed id"}`, http.StatusBadRequest)
			return
		}
		filter.FeedID = &id
	}
	if catID := r.URL.Query().Get("category"); catID != "" {
		id, err := strconv.ParseInt(catID, 10, 64)
		if err != nil {
			http.Error(w, `{"error":"invalid category id"}`, http.StatusBadRequest)
			return
		}
		filter.CategoryID = &id
	}
	if search := r.URL.Query().Get("search"); search != "" {
		if utf8.RuneCountInString(search) > 200 {
			search = string([]rune(search)[:200])
		}
		filter.Search = search
	}
	if pa := r.URL.Query().Get("published_after"); pa != "" {
		// Support relative durations like "24h", "7d", "1w" or RFC3339 strings
		if parsed, err := parseRelativeOrAbsoluteTime(pa); err == nil {
			filter.PublishedAfter = parsed.Format(time.RFC3339)
		} else {
			filter.PublishedAfter = pa
		}
	}
	if mrt := r.URL.Query().Get("min_reading_time"); mrt != "" {
		if v, err := strconv.Atoi(mrt); err == nil && v > 0 {
			filter.MinReadingTime = v
		}
	}
	if mrt := r.URL.Query().Get("max_reading_time"); mrt != "" {
		if v, err := strconv.Atoi(mrt); err == nil && v > 0 {
			filter.MaxReadingTime = v
		}
	}

	articles, total, err := h.store.ListArticles(userID, filter)
	if err != nil {
		http.Error(w, `{"error":"failed to list articles"}`, http.StatusInternalServerError)
		return
	}
	if articles == nil {
		articles = []models.Article{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"articles": articles,
		"total":    total,
		"page":     filter.Page,
		"limit":    filter.Limit,
	})
}

func (h *ArticleHandler) Get(w http.ResponseWriter, r *http.Request) {
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

	// Lazy content extraction: if content_clean is empty and we have a URL,
	// try extracting full article content on demand (RSS feeds often only
	// provide a summary/teaser, and initial extraction may have failed).
	if article.ContentClean == "" && article.URL != "" {
		if clean, err := readability.ExtractContent(article.URL); err == nil && clean != "" {
			article.ContentClean = clean
			wordCount := countWordsFromHTML(clean)
			readingTime := int(math.Ceil(float64(wordCount) / 200.0))
			article.WordCount = wordCount
			article.ReadingTime = readingTime
			// Persist so we don't re-extract next time
			go func() {
				if err := h.store.UpdateArticleContent(id, clean, wordCount, readingTime); err != nil {
					log.Printf("lazy-extract: failed to persist content for article %d: %v", id, err)
				}
			}()
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(article)
}

func (h *ArticleHandler) Update(w http.ResponseWriter, r *http.Request) {
	userID := apiutil.ExtractUserID(r)
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, `{"error":"invalid id"}`, http.StatusBadRequest)
		return
	}

	var req models.UpdateArticleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	if err := h.store.UpdateArticle(id, userID, req.IsRead, req.IsStarred); err != nil {
		http.Error(w, `{"error":"failed to update article"}`, http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *ArticleHandler) Dismiss(w http.ResponseWriter, r *http.Request) {
	userID := apiutil.ExtractUserID(r)
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, `{"error":"invalid id"}`, http.StatusBadRequest)
		return
	}

	isRead := true
	if err := h.store.UpdateArticle(id, userID, &isRead, nil); err != nil {
		http.Error(w, `{"error":"failed to dismiss article"}`, http.StatusInternalServerError)
		return
	}
	if err := h.store.CreateReadingEvent(id, "dismiss", 0); err != nil {
		http.Error(w, `{"error":"failed to create dismiss event"}`, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *ArticleHandler) MarkAllRead(w http.ResponseWriter, r *http.Request) {
	userID := apiutil.ExtractUserID(r)

	var req struct {
		FeedID     *int64 `json:"feed_id"`
		CategoryID *int64 `json:"category_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	affected, err := h.store.MarkAllRead(userID, req.FeedID, req.CategoryID)
	if err != nil {
		http.Error(w, `{"error":"failed to mark as read"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]int64{"affected": affected})
}

func (h *ArticleHandler) Bulk(w http.ResponseWriter, r *http.Request) {
	userID := apiutil.ExtractUserID(r)
	var req models.BulkArticleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	if len(req.ArticleIDs) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if len(req.ArticleIDs) > 500 {
		http.Error(w, `{"error":"too many article IDs, max 500"}`, http.StatusBadRequest)
		return
	}

	validActions := map[string]bool{"mark_read": true, "mark_unread": true, "star": true, "unstar": true}
	if !validActions[req.Action] {
		http.Error(w, `{"error":"invalid action"}`, http.StatusBadRequest)
		return
	}

	if err := h.store.BulkUpdateArticles(userID, req.ArticleIDs, req.Action); err != nil {
		http.Error(w, `{"error":"failed to perform bulk action"}`, http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *ArticleHandler) CatchUp(w http.ResponseWriter, r *http.Request) {
	userID := apiutil.ExtractUserID(r)

	var req struct {
		Strategy   string `json:"strategy"`
		Value      string `json:"value"`
		Count      int    `json:"count"`
		FeedID     *int64 `json:"feed_id"`
		CategoryID *int64 `json:"category_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	if req.Strategy != "older_than" && req.Strategy != "keep_newest" {
		http.Error(w, `{"error":"strategy must be older_than or keep_newest"}`, http.StatusBadRequest)
		return
	}
	if req.Strategy == "older_than" && req.Value == "" {
		http.Error(w, `{"error":"value is required for older_than strategy"}`, http.StatusBadRequest)
		return
	}
	if req.Strategy == "keep_newest" && req.Count <= 0 {
		http.Error(w, `{"error":"count must be positive for keep_newest strategy"}`, http.StatusBadRequest)
		return
	}

	affected, err := h.store.CatchUp(userID, req.Strategy, req.Value, req.Count, req.FeedID, req.CategoryID)
	if err != nil {
		log.Printf("catch-up failed for user %d: %v", userID, err)
		http.Error(w, "catch-up failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]int64{"affected": affected})
}

// parseRelativeOrAbsoluteTime parses a time string that is either a relative
// duration (e.g., "24h", "3d", "1w") or an RFC3339 timestamp.
func parseRelativeOrAbsoluteTime(s string) (time.Time, error) {
	// Try RFC3339 first
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}

	// Try relative duration: number + unit (h/d/w)
	if len(s) < 2 {
		return time.Time{}, fmt.Errorf("invalid time value")
	}
	numStr := s[:len(s)-1]
	unit := s[len(s)-1:]
	num, err := strconv.Atoi(numStr)
	if err != nil || num <= 0 {
		return time.Time{}, fmt.Errorf("invalid duration number")
	}
	if num > 36500 {
		return time.Time{}, fmt.Errorf("duration value too large, max 36500")
	}
	var duration time.Duration
	switch unit {
	case "h":
		duration = time.Duration(num) * time.Hour
	case "d":
		duration = time.Duration(num) * 24 * time.Hour
	case "w":
		duration = time.Duration(num) * 7 * 24 * time.Hour
	default:
		return time.Time{}, fmt.Errorf("invalid duration unit")
	}
	return time.Now().Add(-duration), nil
}

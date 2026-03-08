package handlers

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/feednest/backend/internal/apiutil"
	"github.com/feednest/backend/internal/models"
	"github.com/feednest/backend/internal/scheduler"
	"github.com/feednest/backend/internal/store"
)

type FeedHandler struct {
	store     *store.Queries
	scheduler *scheduler.Scheduler
}

func NewFeedHandler(store *store.Queries, sched *scheduler.Scheduler) *FeedHandler {
	return &FeedHandler{store: store, scheduler: sched}
}

func (h *FeedHandler) List(w http.ResponseWriter, r *http.Request) {
	userID := apiutil.ExtractUserID(r)
	feeds, err := h.store.ListFeeds(userID)
	if err != nil {
		http.Error(w, `{"error":"failed to list feeds"}`, http.StatusInternalServerError)
		return
	}
	if feeds == nil {
		feeds = []models.Feed{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(feeds)
}

func (h *FeedHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID := apiutil.ExtractUserID(r)
	var req models.CreateFeedRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}
	if req.URL == "" {
		http.Error(w, `{"error":"url is required"}`, http.StatusBadRequest)
		return
	}

	// If new_category is provided, create it first
	if req.NewCategory != "" {
		cat, err := h.store.CreateCategory(userID, req.NewCategory, 0)
		if err == nil {
			req.CategoryID = &cat.ID
		}
	}

	// Validate category ownership if category_id was provided
	if req.CategoryID != nil {
		if _, err := h.store.GetCategory(*req.CategoryID, userID); err != nil {
			http.Error(w, `{"error":"category not found"}`, http.StatusBadRequest)
			return
		}
	}

	feed, err := h.store.CreateFeed(userID, req.URL, "", "", "", req.CategoryID)
	if err != nil {
		http.Error(w, `{"error":"failed to create feed or URL already exists"}`, http.StatusConflict)
		return
	}

	if h.scheduler != nil {
		h.scheduler.FetchFeedNow(feed.ID, feed.URL)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(feed)
}

func (h *FeedHandler) Update(w http.ResponseWriter, r *http.Request) {
	userID := apiutil.ExtractUserID(r)
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, `{"error":"invalid id"}`, http.StatusBadRequest)
		return
	}

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	var req models.UpdateFeedRequest
	if err := json.Unmarshal(bodyBytes, &req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	if req.FetchInterval != nil {
		if *req.FetchInterval < 60 {
			http.Error(w, `{"error":"fetch_interval must be at least 60 seconds"}`, http.StatusBadRequest)
			return
		}
		if *req.FetchInterval > 86400 {
			http.Error(w, `{"error":"fetch_interval must not exceed 86400 seconds"}`, http.StatusBadRequest)
			return
		}
	}

	// Check if category_id was explicitly set to null (uncategorize)
	var raw map[string]json.RawMessage
	json.Unmarshal(bodyBytes, &raw)
	clearCategory := false
	if catRaw, exists := raw["category_id"]; exists && string(catRaw) == "null" {
		clearCategory = true
	}

	if req.CategoryID != nil {
		if _, err := h.store.GetCategory(*req.CategoryID, userID); err != nil {
			http.Error(w, `{"error":"category not found"}`, http.StatusBadRequest)
			return
		}
	}

	if err := h.store.UpdateFeed(id, userID, &req); err != nil {
		http.Error(w, `{"error":"failed to update feed"}`, http.StatusInternalServerError)
		return
	}

	if clearCategory {
		if err := h.store.ClearFeedCategory(id, userID); err != nil {
			http.Error(w, `{"error":"failed to clear category"}`, http.StatusInternalServerError)
			return
		}
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *FeedHandler) Retry(w http.ResponseWriter, r *http.Request) {
	userID := apiutil.ExtractUserID(r)
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, `{"error":"invalid id"}`, http.StatusBadRequest)
		return
	}

	feed, err := h.store.GetFeed(id, userID)
	if err != nil {
		http.Error(w, `{"error":"feed not found"}`, http.StatusNotFound)
		return
	}

	if h.scheduler != nil {
		h.scheduler.FetchFeedNow(feed.ID, feed.URL)
	}

	w.WriteHeader(http.StatusAccepted)
}

func (h *FeedHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID := apiutil.ExtractUserID(r)
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, `{"error":"invalid id"}`, http.StatusBadRequest)
		return
	}

	if err := h.store.DeleteFeed(id, userID); err != nil {
		http.Error(w, `{"error":"failed to delete feed"}`, http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

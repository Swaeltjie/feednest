package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/feednest/backend/internal/api"
	"github.com/feednest/backend/internal/models"
	"github.com/feednest/backend/internal/store"
)

type FeedHandler struct {
	store *store.Queries
}

func NewFeedHandler(store *store.Queries) *FeedHandler {
	return &FeedHandler{store: store}
}

func (h *FeedHandler) List(w http.ResponseWriter, r *http.Request) {
	userID := api.ExtractUserID(r)
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
	userID := api.ExtractUserID(r)
	var req models.CreateFeedRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}
	if req.URL == "" {
		http.Error(w, `{"error":"url is required"}`, http.StatusBadRequest)
		return
	}

	feed, err := h.store.CreateFeed(userID, req.URL, "", "", "", req.CategoryID)
	if err != nil {
		http.Error(w, `{"error":"failed to create feed or URL already exists"}`, http.StatusConflict)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(feed)
}

func (h *FeedHandler) Update(w http.ResponseWriter, r *http.Request) {
	userID := api.ExtractUserID(r)
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, `{"error":"invalid id"}`, http.StatusBadRequest)
		return
	}

	var req models.UpdateFeedRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	if err := h.store.UpdateFeed(id, userID, &req); err != nil {
		http.Error(w, `{"error":"failed to update feed"}`, http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *FeedHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID := api.ExtractUserID(r)
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

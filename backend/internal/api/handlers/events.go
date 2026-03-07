package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/feednest/backend/internal/apiutil"
	"github.com/feednest/backend/internal/models"
	"github.com/feednest/backend/internal/store"
)

type EventHandler struct {
	store *store.Queries
}

func NewEventHandler(store *store.Queries) *EventHandler {
	return &EventHandler{store: store}
}

func (h *EventHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID := apiutil.ExtractUserID(r)

	var req models.CreateEventRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	// Verify the article belongs to this user
	if _, err := h.store.GetArticle(req.ArticleID, userID); err != nil {
		http.Error(w, `{"error":"article not found"}`, http.StatusNotFound)
		return
	}

	if err := h.store.CreateReadingEvent(req.ArticleID, req.EventType, req.DurationSeconds); err != nil {
		http.Error(w, `{"error":"failed to create event"}`, http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

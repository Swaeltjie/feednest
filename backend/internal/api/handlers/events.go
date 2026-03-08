package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
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

var validEventTypes = map[string]bool{
	"open": true, "read": true, "dismiss": true, "scroll": true, "click": true,
}

func (h *EventHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID := apiutil.ExtractUserID(r)

	var req models.CreateEventRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	if !validEventTypes[req.EventType] {
		http.Error(w, `{"error":"invalid event_type"}`, http.StatusBadRequest)
		return
	}
	if req.DurationSeconds < 0 || req.DurationSeconds > 86400 {
		http.Error(w, `{"error":"duration_seconds must be 0-86400"}`, http.StatusBadRequest)
		return
	}

	// Verify the article belongs to this user
	if _, err := h.store.GetArticle(req.ArticleID, userID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, `{"error":"article not found"}`, http.StatusNotFound)
		} else {
			http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		}
		return
	}

	if err := h.store.CreateReadingEvent(req.ArticleID, req.EventType, req.DurationSeconds); err != nil {
		http.Error(w, `{"error":"failed to create event"}`, http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

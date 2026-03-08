package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/feednest/backend/internal/apiutil"
	"github.com/feednest/backend/internal/models"
	"github.com/feednest/backend/internal/store"
)

type TagHandler struct {
	store *store.Queries
}

func NewTagHandler(store *store.Queries) *TagHandler {
	return &TagHandler{store: store}
}

func (h *TagHandler) List(w http.ResponseWriter, r *http.Request) {
	userID := apiutil.ExtractUserID(r)
	tags, err := h.store.ListTags(userID)
	if err != nil {
		http.Error(w, `{"error":"failed to list tags"}`, http.StatusInternalServerError)
		return
	}
	if tags == nil {
		tags = []models.Tag{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tags)
}

func (h *TagHandler) AddToArticle(w http.ResponseWriter, r *http.Request) {
	userID := apiutil.ExtractUserID(r)
	articleID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, `{"error":"invalid id"}`, http.StatusBadRequest)
		return
	}

	// Verify the article belongs to this user
	if _, err := h.store.GetArticle(articleID, userID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, `{"error":"article not found"}`, http.StatusNotFound)
		} else {
			http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		}
		return
	}

	var req models.AddTagRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		http.Error(w, `{"error":"name is required"}`, http.StatusBadRequest)
		return
	}
	if len(req.Name) > 255 {
		http.Error(w, `{"error":"tag name must not exceed 255 characters"}`, http.StatusBadRequest)
		return
	}

	if err := h.store.AddTagToArticle(userID, articleID, req.Name); err != nil {
		http.Error(w, `{"error":"failed to add tag"}`, http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func (h *TagHandler) RemoveFromArticle(w http.ResponseWriter, r *http.Request) {
	userID := apiutil.ExtractUserID(r)
	articleID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, `{"error":"invalid id"}`, http.StatusBadRequest)
		return
	}
	// Verify the article belongs to this user
	if _, err := h.store.GetArticle(articleID, userID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, `{"error":"article not found"}`, http.StatusNotFound)
		} else {
			http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		}
		return
	}

	tagName := strings.TrimSpace(chi.URLParam(r, "tag"))
	if tagName == "" {
		http.Error(w, `{"error":"tag name is required"}`, http.StatusBadRequest)
		return
	}

	if err := h.store.RemoveTagFromArticle(articleID, tagName, userID); err != nil {
		http.Error(w, `{"error":"failed to remove tag"}`, http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

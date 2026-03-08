package handlers

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strconv"
	"unicode/utf8"

	"github.com/go-chi/chi/v5"

	"github.com/feednest/backend/internal/apiutil"
	"github.com/feednest/backend/internal/models"
	"github.com/feednest/backend/internal/store"
)

type RulesHandler struct {
	store *store.Queries
}

func NewRulesHandler(store *store.Queries) *RulesHandler {
	return &RulesHandler{store: store}
}

var validFields = map[string]bool{"title": true, "author": true, "content": true}
var validOperators = map[string]bool{"contains": true, "not_contains": true, "regex": true}
var validActions = map[string]bool{"hide": true, "auto_read": true, "auto_star": true}

func (h *RulesHandler) List(w http.ResponseWriter, r *http.Request) {
	userID := apiutil.ExtractUserID(r)
	rules, err := h.store.ListRules(userID)
	if err != nil {
		http.Error(w, `{"error":"failed to list rules"}`, http.StatusInternalServerError)
		return
	}
	if rules == nil {
		rules = []models.FilterRule{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(rules)
}

func (h *RulesHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID := apiutil.ExtractUserID(r)
	var req models.CreateRuleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	if req.Name == "" {
		http.Error(w, `{"error":"name is required"}`, http.StatusBadRequest)
		return
	}
	if utf8.RuneCountInString(req.Name) > 255 {
		http.Error(w, `{"error":"name must not exceed 255 characters"}`, http.StatusBadRequest)
		return
	}
	if !validFields[req.Field] {
		http.Error(w, `{"error":"field must be one of: title, author, content"}`, http.StatusBadRequest)
		return
	}
	if !validOperators[req.Operator] {
		http.Error(w, `{"error":"operator must be one of: contains, not_contains, regex"}`, http.StatusBadRequest)
		return
	}
	if req.Value == "" {
		http.Error(w, `{"error":"value is required"}`, http.StatusBadRequest)
		return
	}
	if utf8.RuneCountInString(req.Value) > 500 {
		http.Error(w, `{"error":"value must not exceed 500 characters"}`, http.StatusBadRequest)
		return
	}
	if !validActions[req.Action] {
		http.Error(w, `{"error":"action must be one of: hide, auto_read, auto_star"}`, http.StatusBadRequest)
		return
	}
	if req.Operator == "regex" {
		if _, err := regexp.Compile(req.Value); err != nil {
			http.Error(w, `{"error":"invalid regex pattern"}`, http.StatusBadRequest)
			return
		}
	}

	rule, err := h.store.CreateRule(userID, req.Name, req.Field, req.Operator, req.Value, req.Action, req.FeedID)
	if err != nil {
		http.Error(w, `{"error":"failed to create rule"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(rule)
}

func (h *RulesHandler) Update(w http.ResponseWriter, r *http.Request) {
	userID := apiutil.ExtractUserID(r)
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, `{"error":"invalid id"}`, http.StatusBadRequest)
		return
	}

	var req models.UpdateRuleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	if req.Name != nil && utf8.RuneCountInString(*req.Name) > 255 {
		http.Error(w, `{"error":"name must not exceed 255 characters"}`, http.StatusBadRequest)
		return
	}
	if req.Field != nil && !validFields[*req.Field] {
		http.Error(w, `{"error":"field must be one of: title, author, content"}`, http.StatusBadRequest)
		return
	}
	if req.Operator != nil && !validOperators[*req.Operator] {
		http.Error(w, `{"error":"operator must be one of: contains, not_contains, regex"}`, http.StatusBadRequest)
		return
	}
	if req.Value != nil && utf8.RuneCountInString(*req.Value) > 500 {
		http.Error(w, `{"error":"value must not exceed 500 characters"}`, http.StatusBadRequest)
		return
	}
	if req.Action != nil && !validActions[*req.Action] {
		http.Error(w, `{"error":"action must be one of: hide, auto_read, auto_star"}`, http.StatusBadRequest)
		return
	}
	if req.Operator != nil && *req.Operator == "regex" && req.Value != nil {
		if _, err := regexp.Compile(*req.Value); err != nil {
			http.Error(w, `{"error":"invalid regex pattern"}`, http.StatusBadRequest)
			return
		}
	}
	// If only value is being updated, check if existing rule uses regex operator
	if req.Value != nil && req.Operator == nil {
		existing, err := h.store.GetRule(id, userID)
		if err == nil && existing.Operator == "regex" {
			if _, err := regexp.Compile(*req.Value); err != nil {
				http.Error(w, `{"error":"invalid regex pattern"}`, http.StatusBadRequest)
				return
			}
		}
	}

	if err := h.store.UpdateRule(id, userID, req); err != nil {
		http.Error(w, `{"error":"failed to update rule"}`, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *RulesHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID := apiutil.ExtractUserID(r)
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, `{"error":"invalid id"}`, http.StatusBadRequest)
		return
	}

	if err := h.store.DeleteRule(id, userID); err != nil {
		http.Error(w, `{"error":"failed to delete rule"}`, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

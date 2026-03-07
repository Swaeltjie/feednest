package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/feednest/backend/internal/apiutil"
	"github.com/feednest/backend/internal/store"
)

type SettingsHandler struct {
	store *store.Queries
}

func NewSettingsHandler(store *store.Queries) *SettingsHandler {
	return &SettingsHandler{store: store}
}

func (h *SettingsHandler) Get(w http.ResponseWriter, r *http.Request) {
	userID := apiutil.ExtractUserID(r)
	settings, err := h.store.GetSettings(userID)
	if err != nil {
		http.Error(w, `{"error":"failed to get settings"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(settings)
}

func (h *SettingsHandler) Update(w http.ResponseWriter, r *http.Request) {
	userID := apiutil.ExtractUserID(r)
	var settings map[string]string
	if err := json.NewDecoder(r.Body).Decode(&settings); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	if err := h.store.UpdateSettings(userID, settings); err != nil {
		http.Error(w, `{"error":"failed to update settings"}`, http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

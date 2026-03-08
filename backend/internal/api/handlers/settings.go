package handlers

import (
	"encoding/json"
	"math"
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

var allowedSettingKeys = map[string]bool{
	"theme": true, "view_mode": true, "default_sort": true,
	"articles_per_page": true, "auto_mark_read": true,
	"refresh_interval": true, "language": true,
	"font_size": true, "compact_mode": true,
	"reader_font_size": true, "reader_font_family": true,
	"reader_line_height": true, "reader_content_width": true,
	"calm_mode": true, "auto_mark_read_scroll": true,
	"infinite_scroll": true,
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

func (h *SettingsHandler) GetWPM(w http.ResponseWriter, r *http.Request) {
	userID := apiutil.ExtractUserID(r)
	wpm := h.store.GetUserWPM(userID)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"wpm":         math.Round(wpm),
		"default_wpm": 200,
	})
}

func (h *SettingsHandler) Update(w http.ResponseWriter, r *http.Request) {
	userID := apiutil.ExtractUserID(r)
	var settings map[string]string
	if err := json.NewDecoder(r.Body).Decode(&settings); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	for key, value := range settings {
		if !allowedSettingKeys[key] {
			http.Error(w, `{"error":"unknown setting key"}`, http.StatusBadRequest)
			return
		}
		if len(value) > 1000 {
			http.Error(w, `{"error":"setting value too long"}`, http.StatusBadRequest)
			return
		}
	}

	if err := h.store.UpdateSettings(userID, settings); err != nil {
		http.Error(w, `{"error":"failed to update settings"}`, http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

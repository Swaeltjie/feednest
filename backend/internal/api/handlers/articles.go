package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"unicode/utf8"

	"github.com/go-chi/chi/v5"

	"github.com/feednest/backend/internal/apiutil"
	"github.com/feednest/backend/internal/models"
	"github.com/feednest/backend/internal/store"
)

type ArticleHandler struct {
	store *store.Queries
}

func NewArticleHandler(store *store.Queries) *ArticleHandler {
	return &ArticleHandler{store: store}
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
		id, _ := strconv.ParseInt(feedID, 10, 64)
		filter.FeedID = &id
	}
	if catID := r.URL.Query().Get("category"); catID != "" {
		id, _ := strconv.ParseInt(catID, 10, 64)
		filter.CategoryID = &id
	}
	if search := r.URL.Query().Get("search"); search != "" {
		if utf8.RuneCountInString(search) > 200 {
			search = string([]rune(search)[:200])
		}
		filter.Search = search
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
		http.Error(w, `{"error":"article not found"}`, http.StatusNotFound)
		return
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

func (h *ArticleHandler) Bulk(w http.ResponseWriter, r *http.Request) {
	userID := apiutil.ExtractUserID(r)
	var req models.BulkArticleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	if len(req.ArticleIDs) > 500 {
		http.Error(w, `{"error":"too many article IDs, max 500"}`, http.StatusBadRequest)
		return
	}

	if err := h.store.BulkUpdateArticles(userID, req.ArticleIDs, req.Action); err != nil {
		http.Error(w, `{"error":"failed to perform bulk action"}`, http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

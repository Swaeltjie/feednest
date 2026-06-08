package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/feednest/backend/internal/apiutil"
	"github.com/feednest/backend/internal/claude"
	"github.com/feednest/backend/internal/store"
)

const (
	// askMaxQuestionLen bounds the user's question length.
	askMaxQuestionLen = 1000
	// askPassageLimit is how many article passages we retrieve to ground an answer.
	askPassageLimit = 8
)

// AskHandler answers free-text questions by retrieving relevant passages from
// the user's own article archive (FTS5) and feeding them to Claude with a
// grounding/citation prompt — "Ask Your Feeds" conversational RAG.
type AskHandler struct {
	store  *store.Queries
	claude *claude.Client
}

func NewAskHandler(store *store.Queries, c *claude.Client) *AskHandler {
	return &AskHandler{store: store, claude: c}
}

type askRequest struct {
	Question string `json:"question"`
}

type askSource struct {
	ID        int64  `json:"id"`
	Title     string `json:"title"`
	URL       string `json:"url"`
	FeedTitle string `json:"feed_title"`
}

type askResponse struct {
	Answer  string      `json:"answer"`
	Sources []askSource `json:"sources"`
}

// Ask handles POST /api/ask.
func (h *AskHandler) Ask(w http.ResponseWriter, r *http.Request) {
	userID := apiutil.ExtractUserID(r)

	var req askRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}
	question := strings.TrimSpace(req.Question)
	if question == "" || len([]rune(question)) > askMaxQuestionLen {
		http.Error(w, `{"error":"question must be between 1 and 1000 characters"}`, http.StatusBadRequest)
		return
	}

	if !h.claude.Enabled() {
		http.Error(w, `{"error":"AI features are not configured"}`, http.StatusServiceUnavailable)
		return
	}

	passages, err := h.store.SearchPassages(userID, question, askPassageLimit)
	if err != nil {
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}

	if len(passages) == 0 {
		writeAsk(w, askResponse{Answer: "Your feeds haven't covered this yet.", Sources: []askSource{}})
		return
	}

	cps := make([]claude.Passage, len(passages))
	sources := make([]askSource, len(passages))
	for i, p := range passages {
		cps[i] = claude.Passage{
			ID:        p.ArticleID,
			Title:     p.Title,
			URL:       p.URL,
			FeedTitle: p.FeedTitle,
			Excerpt:   p.Excerpt,
		}
		sources[i] = askSource{
			ID:        p.ArticleID,
			Title:     p.Title,
			URL:       p.URL,
			FeedTitle: p.FeedTitle,
		}
	}

	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()

	answer, err := h.claude.Answer(ctx, question, cps)
	if err != nil {
		if errors.Is(err, claude.ErrDisabled) {
			http.Error(w, `{"error":"AI features are not configured"}`, http.StatusServiceUnavailable)
			return
		}
		http.Error(w, `{"error":"failed to generate answer"}`, http.StatusBadGateway)
		return
	}

	writeAsk(w, askResponse{Answer: answer, Sources: sources})
}

func writeAsk(w http.ResponseWriter, resp askResponse) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

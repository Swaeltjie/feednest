package handlers

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/feednest/backend/internal/urlutil"
	"github.com/mmcdole/gofeed"
)

type DiscoverHandler struct{}

func NewDiscoverHandler() *DiscoverHandler {
	return &DiscoverHandler{}
}

type DiscoveredFeed struct {
	URL   string `json:"url"`
	Title string `json:"title"`
	Type  string `json:"type"`
}

type DiscoverRequest struct {
	URL string `json:"url"`
}

var linkRe = regexp.MustCompile(`(?i)<link[^>]+>`)
var hrefRe = regexp.MustCompile(`(?i)href\s*=\s*["']([^"']+)["']`)
var typeRe = regexp.MustCompile(`(?i)type\s*=\s*["']([^"']+)["']`)
var titleRe = regexp.MustCompile(`(?i)title\s*=\s*["']([^"']+)["']`)
var relRe = regexp.MustCompile(`(?i)rel\s*=\s*["']([^"']+)["']`)

func (h *DiscoverHandler) Discover(w http.ResponseWriter, r *http.Request) {
	var req DiscoverRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.URL == "" {
		http.Error(w, `{"error":"url is required"}`, http.StatusBadRequest)
		return
	}

	if err := urlutil.IsSafeURL(req.URL); err != nil {
		http.Error(w, `{"error":"unsafe URL"}`, http.StatusBadRequest)
		return
	}

	client := urlutil.SafeHTTPClient(15 * time.Second)
	httpReq, err := http.NewRequest("GET", req.URL, nil)
	if err != nil {
		http.Error(w, `{"error":"invalid URL"}`, http.StatusBadRequest)
		return
	}
	httpReq.Header.Set("User-Agent", "FeedNest/1.0")
	httpReq.Header.Set("Accept", "application/rss+xml, application/atom+xml, application/xml, text/xml, text/html, */*")

	resp, err := client.Do(httpReq)
	if err != nil {
		http.Error(w, `{"error":"failed to fetch URL"}`, http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 2*1024*1024))
	if err != nil {
		http.Error(w, `{"error":"failed to read response"}`, http.StatusBadGateway)
		return
	}

	// Try parsing as feed
	fp := gofeed.NewParser()
	if feed, err := fp.ParseString(string(body)); err == nil && feed != nil {
		feedType := "rss"
		if feed.FeedType == "atom" {
			feedType = "atom"
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"feeds": []DiscoveredFeed{{URL: req.URL, Title: feed.Title, Type: feedType}},
		})
		return
	}

	// Not a feed — parse HTML for <link rel="alternate"> tags
	html := string(body)
	var feeds []DiscoveredFeed

	links := linkRe.FindAllString(html, -1)
	for _, link := range links {
		relMatch := relRe.FindStringSubmatch(link)
		if len(relMatch) < 2 || !strings.Contains(strings.ToLower(relMatch[1]), "alternate") {
			continue
		}

		typeMatch := typeRe.FindStringSubmatch(link)
		if len(typeMatch) < 2 {
			continue
		}
		feedType := strings.ToLower(typeMatch[1])
		if !strings.Contains(feedType, "rss") && !strings.Contains(feedType, "atom") && !strings.Contains(feedType, "xml") {
			continue
		}

		hrefMatch := hrefRe.FindStringSubmatch(link)
		if len(hrefMatch) < 2 {
			continue
		}
		feedURL := hrefMatch[1]

		// Resolve relative URLs
		if !strings.HasPrefix(feedURL, "http") {
			base, err := url.Parse(req.URL)
			if err != nil {
				continue
			}
			ref, err := url.Parse(feedURL)
			if err != nil {
				continue
			}
			feedURL = base.ResolveReference(ref).String()
		}

		title := ""
		titleMatch := titleRe.FindStringSubmatch(link)
		if len(titleMatch) >= 2 {
			title = titleMatch[1]
		}

		shortType := "rss"
		if strings.Contains(feedType, "atom") {
			shortType = "atom"
		}

		feeds = append(feeds, DiscoveredFeed{URL: feedURL, Title: title, Type: shortType})
	}

	if len(feeds) == 0 {
		http.Error(w, `{"error":"no feeds found at this URL"}`, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"feeds": feeds})
}

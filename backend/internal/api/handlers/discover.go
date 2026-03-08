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
var anchorRe = regexp.MustCompile(`(?i)<a\s[^>]*href\s*=\s*["']([^"']+)["'][^>]*>`)
var hrefRe = regexp.MustCompile(`(?i)href\s*=\s*["']([^"']+)["']`)
var typeRe = regexp.MustCompile(`(?i)type\s*=\s*["']([^"']+)["']`)
var titleRe = regexp.MustCompile(`(?i)title\s*=\s*["']([^"']+)["']`)
var relRe = regexp.MustCompile(`(?i)rel\s*=\s*["']([^"']+)["']`)

// feedKeywords are substrings that suggest a URL points to a feed.
var feedKeywords = []string{"rss", "atom", "feed", "xml"}

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
		// Stage 3: Scan <a> tags in page body for feed-like URLs (à la FreshRSS/SimplePie)
		feeds = scanBodyLinks(html, req.URL, client)
	}

	if len(feeds) == 0 {
		// Stage 4: Probe common feed URL paths as last resort
		feeds = probeFeedPaths(req.URL, client)
	}

	if len(feeds) == 0 {
		http.Error(w, `{"error":"no feeds found at this URL"}`, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"feeds": feeds})
}

// scanBodyLinks scans all <a> tags in the HTML body for hrefs containing feed-related
// keywords (rss, atom, feed, xml). Checks same-host first, then subdomains, then any domain.
// This mirrors FreshRSS/SimplePie's ultra-liberal feed locator (steps 2-4).
func scanBodyLinks(html, rawURL string, client *http.Client) []DiscoveredFeed {
	base, err := url.Parse(rawURL)
	if err != nil {
		return nil
	}

	matches := anchorRe.FindAllStringSubmatch(html, -1)
	if len(matches) == 0 {
		return nil
	}

	// Collect candidate URLs, categorized by host affinity
	type candidate struct {
		url      string
		priority int // 0=same host, 1=subdomain, 2=external
	}
	var candidates []candidate
	seen := make(map[string]bool)

	for _, m := range matches {
		href := m[1]
		if href == "" || strings.HasPrefix(href, "#") || strings.HasPrefix(href, "javascript:") || strings.HasPrefix(href, "mailto:") {
			continue
		}

		// Check if href contains any feed keyword
		lowerHref := strings.ToLower(href)
		hasFeedKeyword := false
		for _, kw := range feedKeywords {
			if strings.Contains(lowerHref, kw) {
				hasFeedKeyword = true
				break
			}
		}
		if !hasFeedKeyword {
			continue
		}

		// Resolve relative URLs
		resolved := href
		if !strings.HasPrefix(href, "http") {
			ref, err := url.Parse(href)
			if err != nil {
				continue
			}
			resolved = base.ResolveReference(ref).String()
		}

		if seen[resolved] {
			continue
		}
		seen[resolved] = true

		parsed, err := url.Parse(resolved)
		if err != nil {
			continue
		}

		// Categorize by host relationship
		pri := 2 // external
		if parsed.Host == base.Host {
			pri = 0 // same host
		} else if strings.HasSuffix(parsed.Host, "."+base.Host) || strings.HasSuffix(base.Host, "."+parsed.Host) {
			pri = 1 // subdomain
		}

		candidates = append(candidates, candidate{url: resolved, priority: pri})
	}

	// Sort: same host first, then subdomains, then external
	for i := 0; i < len(candidates); i++ {
		for j := i + 1; j < len(candidates); j++ {
			if candidates[j].priority < candidates[i].priority {
				candidates[i], candidates[j] = candidates[j], candidates[i]
			}
		}
	}

	// Try each candidate (limit to 10 to avoid excessive requests)
	fp := gofeed.NewParser()
	var feeds []DiscoveredFeed
	tried := 0
	for _, c := range candidates {
		if tried >= 10 {
			break
		}

		if err := urlutil.IsSafeURL(c.url); err != nil {
			continue
		}

		req, err := http.NewRequest("GET", c.url, nil)
		if err != nil {
			continue
		}
		req.Header.Set("User-Agent", "FeedNest/1.0")
		req.Header.Set("Accept", "application/rss+xml, application/atom+xml, application/xml, text/xml, */*")

		tried++
		resp, err := client.Do(req)
		if err != nil {
			continue
		}
		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			continue
		}

		body, err := io.ReadAll(io.LimitReader(resp.Body, 512*1024))
		resp.Body.Close()
		if err != nil {
			continue
		}

		feed, err := fp.ParseString(string(body))
		if err != nil || feed == nil {
			continue
		}

		feedType := "rss"
		if feed.FeedType == "atom" {
			feedType = "atom"
		}

		feeds = append(feeds, DiscoveredFeed{
			URL:   c.url,
			Title: feed.Title,
			Type:  feedType,
		})
	}

	return feeds
}

// commonFeedPaths lists well-known feed URL paths to probe when HTML discovery fails.
var commonFeedPaths = []string{
	"/feed",
	"/feed/",
	"/rss",
	"/rss/",
	"/rss.xml",
	"/atom.xml",
	"/feed.xml",
	"/feed.rss",
	"/feed.atom",
	"/index.xml",
	"/index.rss",
	"/feeds/posts/default", // Blogger
	"/blog/feed",
	"/blog/rss",
	"/?feed=rss2", // WordPress
	"/feed/rss2",  // WordPress
}

// probeFeedPaths tries common feed URL paths against the site's base URL.
func probeFeedPaths(rawURL string, client *http.Client) []DiscoveredFeed {
	base, err := url.Parse(rawURL)
	if err != nil {
		return nil
	}
	// Use scheme + host as base (strip path)
	origin := base.Scheme + "://" + base.Host

	fp := gofeed.NewParser()
	var feeds []DiscoveredFeed
	seen := make(map[string]bool)

	for _, path := range commonFeedPaths {
		candidate := origin + path
		if seen[candidate] {
			continue
		}
		seen[candidate] = true

		if err := urlutil.IsSafeURL(candidate); err != nil {
			continue
		}

		req, err := http.NewRequest("GET", candidate, nil)
		if err != nil {
			continue
		}
		req.Header.Set("User-Agent", "FeedNest/1.0")
		req.Header.Set("Accept", "application/rss+xml, application/atom+xml, application/xml, text/xml, */*")

		resp, err := client.Do(req)
		if err != nil {
			continue
		}

		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			continue
		}

		body, err := io.ReadAll(io.LimitReader(resp.Body, 512*1024))
		resp.Body.Close()
		if err != nil {
			continue
		}

		feed, err := fp.ParseString(string(body))
		if err != nil || feed == nil {
			continue
		}

		feedType := "rss"
		if feed.FeedType == "atom" {
			feedType = "atom"
		}

		feeds = append(feeds, DiscoveredFeed{
			URL:   candidate,
			Title: feed.Title,
			Type:  feedType,
		})
	}

	return feeds
}

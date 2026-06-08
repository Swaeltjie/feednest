package readability

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/feednest/backend/internal/urlutil"
	goreadability "codeberg.org/readeck/go-readability/v2"
)

const maxArticleResponseSize = 5 * 1024 * 1024 // 5MB

// ExtractionResult holds extracted content and metadata from an article page.
type ExtractionResult struct {
	Content      string
	ThumbnailURL string
}

// extractionStrategy defines headers/settings for a single extraction attempt.
type extractionStrategy struct {
	name    string
	headers map[string]string
}

// strategies is tried in order until one succeeds.
var strategies = []extractionStrategy{
	{
		name: "browser",
		headers: map[string]string{
			"User-Agent":                "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36",
			"Accept":                    "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8",
			"Accept-Language":           "en-US,en;q=0.9",
			"Sec-Fetch-Dest":            "document",
			"Sec-Fetch-Mode":            "navigate",
			"Sec-Fetch-Site":            "none",
			"Sec-Fetch-User":            "?1",
			"Sec-Ch-Ua":                 `"Google Chrome";v="131", "Chromium";v="131", "Not_A Brand";v="24"`,
			"Sec-Ch-Ua-Mobile":          "?0",
			"Sec-Ch-Ua-Platform":        `"Windows"`,
			"Upgrade-Insecure-Requests": "1",
			"Cache-Control":             "no-cache",
		},
	},
	{
		name: "google-referrer",
		headers: map[string]string{
			"User-Agent":      "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36",
			"Accept":          "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8",
			"Accept-Language":  "en-US,en;q=0.9",
			"Referer":         "https://www.google.com/",
		},
	},
	{
		name: "curl-minimal",
		headers: map[string]string{
			"User-Agent": "Mozilla/5.0 (compatible; FeedNest/1.0)",
			"Accept":     "text/html",
		},
	},
}

// ExtractContent fetches an article URL and returns the readable content.
func ExtractContent(articleURL string) (string, error) {
	result, err := Extract(articleURL)
	if err != nil {
		return "", err
	}
	return result.Content, nil
}

// Extract fetches an article URL and returns content + thumbnail.
func Extract(articleURL string) (*ExtractionResult, error) {
	if err := urlutil.IsSafeURL(articleURL); err != nil {
		return nil, fmt.Errorf("unsafe article URL %s: %w", articleURL, err)
	}

	parsedURL, err := url.Parse(articleURL)
	if err != nil {
		return nil, fmt.Errorf("invalid URL %s: %w", articleURL, err)
	}

	var lastErr error
	var thumbOnly *ExtractionResult
	for _, strategy := range strategies {
		result, err := tryExtract(articleURL, parsedURL, strategy)
		if err != nil {
			lastErr = err
			continue
		}
		if result != nil && result.Content != "" {
			return result, nil
		}
		// Content extraction failed for this strategy, but the page may still
		// have yielded a thumbnail (e.g. og:image on a JS-rendered page that
		// readability can't parse). Keep it as a fallback so a missing article
		// body doesn't also cost us the thumbnail.
		if result != nil && result.ThumbnailURL != "" && thumbOnly == nil {
			thumbOnly = result
		}
	}

	if thumbOnly != nil {
		return thumbOnly, nil
	}
	if lastErr != nil {
		return nil, lastErr
	}
	return nil, fmt.Errorf("all extraction strategies failed for %s", articleURL)
}

func tryExtract(articleURL string, parsedURL *url.URL, strategy extractionStrategy) (*ExtractionResult, error) {
	req, err := http.NewRequest("GET", articleURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	for k, v := range strategy.headers {
		req.Header.Set(k, v)
	}

	client := urlutil.SafeHTTPClient(30 * time.Second)
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("strategy %s: failed to fetch %s: %w", strategy.name, articleURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("strategy %s: status %d for %s", strategy.name, resp.StatusCode, articleURL)
	}

	// Read body into buffer so we can extract og:image AND pass to readability
	rawBody, err := io.ReadAll(io.LimitReader(resp.Body, maxArticleResponseSize))
	if err != nil {
		return nil, fmt.Errorf("strategy %s: failed to read body: %w", strategy.name, err)
	}

	rawHTML := string(rawBody)

	// The page-level thumbnail (og:image / twitter:image) is parsed straight from
	// the raw HTML and does NOT depend on readability successfully extracting the
	// article body — so a JS-rendered page that readability can't parse still
	// yields a thumbnail.
	pageThumbnail := ExtractThumbnailFromHTML(rawHTML)

	var content string
	var readabilityThumb string
	if article, err := goreadability.FromReader(bytes.NewReader(rawBody), parsedURL); err == nil {
		var contentBuf bytes.Buffer
		if article.Node != nil {
			if rerr := article.RenderHTML(&contentBuf); rerr == nil {
				content = contentBuf.String()
			}
		}
		readabilityThumb = article.ImageURL()
	}
	// Drop bot-protection bodies, but keep any thumbnail we found.
	if IsBlockedContent(content) {
		content = ""
	}

	// Thumbnail preference: readability's main image > og:image/twitter:image > first <img> in content.
	thumbnail := readabilityThumb
	if thumbnail == "" {
		thumbnail = pageThumbnail
	}
	if thumbnail == "" {
		thumbnail = extractFirstImg(content)
	}

	return &ExtractionResult{
		Content:      content,
		ThumbnailURL: thumbnail,
	}, nil
}

// IsBlockedContent detects Cloudflare, cookie walls, and other bot-protection pages.
func IsBlockedContent(content string) bool {
	lower := strings.ToLower(content)
	markers := []string{
		"please enable cookies",
		"you have been blocked",
		"enable javascript and cookies to continue",
		"checking your browser",
		"attention required",
		"cloudflare ray id",
		"security service to protect itself",
		"please enable js and disable any ad blocker",
		"403 forbidden",
		"access denied",
		"captcha-delivery",
		"datadome",
		"just a moment",
	}
	for _, m := range markers {
		if strings.Contains(lower, m) {
			return true
		}
	}
	return false
}

var ogImageRe = regexp.MustCompile(`<meta[^>]+property=["']og:image["'][^>]+content=["']([^"']+)["']`)
var ogImageRe2 = regexp.MustCompile(`<meta[^>]+content=["']([^"']+)["'][^>]+property=["']og:image["']`)
var twitterImageRe = regexp.MustCompile(`<meta[^>]+name=["']twitter:image(?::src)?["'][^>]+content=["']([^"']+)["']`)
var twitterImageRe2 = regexp.MustCompile(`<meta[^>]+content=["']([^"']+)["'][^>]+name=["']twitter:image(?::src)?["']`)
var imgSrcRe = regexp.MustCompile(`(?i)<img[^>]+src=["']([^"']+)["']`)

// ExtractThumbnailFromHTML pulls a representative image URL from a page's social
// meta tags: og:image (either attribute order), then twitter:image as a fallback.
func ExtractThumbnailFromHTML(html string) string {
	for _, re := range []*regexp.Regexp{ogImageRe, ogImageRe2, twitterImageRe, twitterImageRe2} {
		if m := re.FindStringSubmatch(html); len(m) > 1 {
			if v := strings.TrimSpace(m[1]); v != "" {
				return v
			}
		}
	}
	return ""
}

// extractFirstImg pulls the first <img src="..."> from extracted article content.
func extractFirstImg(html string) string {
	matches := imgSrcRe.FindStringSubmatch(html)
	if len(matches) > 1 {
		src := strings.TrimSpace(matches[1])
		// Skip tiny tracking pixels and data URIs
		if strings.HasPrefix(src, "data:") || strings.Contains(src, "1x1") || strings.Contains(src, "pixel") {
			return ""
		}
		return src
	}
	return ""
}

package readability

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/feednest/backend/internal/urlutil"
	goreadability "github.com/go-shiori/go-readability"
)

const maxArticleResponseSize = 5 * 1024 * 1024 // 5MB

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
			"User-Agent":      "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36",
			"Accept":          "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8",
			"Accept-Language":  "en-US,en;q=0.9",
			"Sec-Fetch-Dest":   "document",
			"Sec-Fetch-Mode":   "navigate",
			"Sec-Fetch-Site":   "none",
			"Sec-Fetch-User":   "?1",
			"Sec-Ch-Ua":        `"Google Chrome";v="131", "Chromium";v="131", "Not_A Brand";v="24"`,
			"Sec-Ch-Ua-Mobile": "?0",
			"Sec-Ch-Ua-Platform": `"Windows"`,
			"Upgrade-Insecure-Requests": "1",
			"Cache-Control":    "no-cache",
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

func ExtractContent(articleURL string) (string, error) {
	if err := urlutil.IsSafeURL(articleURL); err != nil {
		return "", fmt.Errorf("unsafe article URL %s: %w", articleURL, err)
	}

	parsedURL, err := url.Parse(articleURL)
	if err != nil {
		return "", fmt.Errorf("invalid URL %s: %w", articleURL, err)
	}

	var lastErr error
	for _, strategy := range strategies {
		content, err := tryExtract(articleURL, parsedURL, strategy)
		if err != nil {
			lastErr = err
			continue
		}
		if content != "" {
			return content, nil
		}
	}

	if lastErr != nil {
		return "", lastErr
	}
	return "", fmt.Errorf("all extraction strategies failed for %s", articleURL)
}

func tryExtract(articleURL string, parsedURL *url.URL, strategy extractionStrategy) (string, error) {
	req, err := http.NewRequest("GET", articleURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	for k, v := range strategy.headers {
		req.Header.Set(k, v)
	}

	client := urlutil.SafeHTTPClient(30 * time.Second)
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("strategy %s: failed to fetch %s: %w", strategy.name, articleURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("strategy %s: status %d for %s", strategy.name, resp.StatusCode, articleURL)
	}

	limitedBody := io.LimitReader(resp.Body, maxArticleResponseSize)
	article, err := goreadability.FromReader(limitedBody, parsedURL)
	if err != nil {
		return "", fmt.Errorf("strategy %s: readability error: %w", strategy.name, err)
	}

	content := article.Content
	if IsBlockedContent(content) {
		return "", fmt.Errorf("strategy %s: content is bot-protection page for %s", strategy.name, articleURL)
	}

	return content, nil
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

func ExtractThumbnailFromHTML(html string) string {
	matches := ogImageRe.FindStringSubmatch(html)
	if len(matches) > 1 {
		return strings.TrimSpace(matches[1])
	}
	matches = ogImageRe2.FindStringSubmatch(html)
	if len(matches) > 1 {
		return strings.TrimSpace(matches[1])
	}
	return ""
}

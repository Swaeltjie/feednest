package readability

import (
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/feednest/backend/internal/urlutil"
	goreadability "github.com/go-shiori/go-readability"
)

const userAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"

func ExtractContent(articleURL string) (string, error) {
	if err := urlutil.IsSafeURL(articleURL); err != nil {
		return "", fmt.Errorf("unsafe article URL %s: %w", articleURL, err)
	}

	parsedURL, err := url.Parse(articleURL)
	if err != nil {
		return "", fmt.Errorf("invalid URL %s: %w", articleURL, err)
	}

	req, err := http.NewRequest("GET", articleURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch %s: %w", articleURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status %d for %s", resp.StatusCode, articleURL)
	}

	article, err := goreadability.FromReader(resp.Body, parsedURL)
	if err != nil {
		return "", err
	}

	content := article.Content
	if IsBlockedContent(content) {
		return "", fmt.Errorf("content appears to be a bot-protection page for %s", articleURL)
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

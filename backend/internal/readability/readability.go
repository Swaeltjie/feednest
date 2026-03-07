package readability

import (
	"regexp"
	"strings"
	"time"

	goreadability "github.com/go-shiori/go-readability"
)

func ExtractContent(articleURL string) (string, error) {
	article, err := goreadability.FromURL(articleURL, 30*time.Second)
	if err != nil {
		return "", err
	}

	return article.Content, nil
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

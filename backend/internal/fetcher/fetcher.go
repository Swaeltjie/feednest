package fetcher

import (
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/feednest/backend/internal/urlutil"
	"github.com/mmcdole/gofeed"
)

const maxFeedResponseSize = 10 * 1024 * 1024 // 10MB

type FeedResult struct {
	Title   string
	SiteURL string
	IconURL string
	Items   []FeedItem
}

type FeedItem struct {
	GUID         string
	Title        string
	URL          string
	Author       string
	ContentRaw   string
	ThumbnailURL string
	PublishedAt  *time.Time
	WordCount    int
	ReadingTime  int
}

const userAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"

func FetchFeed(feedURL string) (*FeedResult, error) {
	if err := urlutil.IsSafeURL(feedURL); err != nil {
		return nil, fmt.Errorf("unsafe feed URL %s: %w", feedURL, err)
	}

	req, err := http.NewRequest("GET", feedURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request for %s: %w", feedURL, err)
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "application/rss+xml, application/atom+xml, application/xml, text/xml, */*")

	client := urlutil.SafeHTTPClient(30 * time.Second)
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch %s: %w", feedURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d for %s", resp.StatusCode, feedURL)
	}

	limitedBody := io.LimitReader(resp.Body, maxFeedResponseSize)
	fp := gofeed.NewParser()
	feed, err := fp.Parse(limitedBody)
	if err != nil {
		return nil, fmt.Errorf("failed to parse feed %s: %w", feedURL, err)
	}

	result := &FeedResult{
		Title:   feed.Title,
		SiteURL: feed.Link,
	}

	if feed.Image != nil {
		result.IconURL = feed.Image.URL
	}

	// If no icon from feed, try to derive favicon from site URL
	if result.IconURL == "" && result.SiteURL != "" {
		result.IconURL = deriveFaviconURL(result.SiteURL)
	}
	if result.IconURL == "" {
		// Try deriving from feed URL itself
		result.IconURL = deriveFaviconURL(feedURL)
	}

	for _, item := range feed.Items {
		fi := FeedItem{
			GUID:  item.GUID,
			Title: item.Title,
			URL:   item.Link,
		}

		if fi.GUID == "" {
			fi.GUID = item.Link
		}

		if item.Author != nil {
			fi.Author = item.Author.Name
		}

		content := item.Content
		if content == "" {
			content = item.Description
		}
		fi.ContentRaw = content

		if item.Image != nil {
			fi.ThumbnailURL = item.Image.URL
		}
		if fi.ThumbnailURL == "" && len(item.Enclosures) > 0 {
			for _, enc := range item.Enclosures {
				if strings.HasPrefix(enc.Type, "image/") {
					fi.ThumbnailURL = enc.URL
					break
				}
			}
		}

		if item.PublishedParsed != nil {
			fi.PublishedAt = item.PublishedParsed
		} else if item.UpdatedParsed != nil {
			fi.PublishedAt = item.UpdatedParsed
		}

		fi.WordCount = countWords(content)
		fi.ReadingTime = int(math.Ceil(float64(fi.WordCount) / 200.0))

		// Skip sponsored/ad content
		titleLower := strings.ToLower(fi.Title)
		if strings.Contains(titleLower, "[sponsored]") ||
			strings.Contains(titleLower, "[ad]") ||
			strings.Contains(titleLower, "sponsored post") ||
			strings.Contains(titleLower, "advertisement") ||
			strings.Contains(titleLower, "| sponsored") ||
			strings.Contains(titleLower, "- sponsored") {
			continue
		}

		result.Items = append(result.Items, fi)
	}

	return result, nil
}

func deriveFaviconURL(siteURL string) string {
	u, err := url.Parse(siteURL)
	if err != nil || u.Host == "" {
		return ""
	}
	return fmt.Sprintf("https://www.google.com/s2/favicons?domain=%s&sz=32", u.Host)
}

func countWords(s string) int {
	inTag := false
	var text strings.Builder
	for _, r := range s {
		if r == '<' {
			inTag = true
			continue
		}
		if r == '>' {
			inTag = false
			text.WriteRune(' ')
			continue
		}
		if !inTag {
			text.WriteRune(r)
		}
	}

	words := strings.Fields(text.String())
	count := 0
	for _, w := range words {
		if utf8.RuneCountInString(w) > 0 {
			count++
		}
	}
	return count
}

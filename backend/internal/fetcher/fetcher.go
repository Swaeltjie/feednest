package fetcher

import (
	"fmt"
	"math"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/mmcdole/gofeed"
)

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

func FetchFeed(url string) (*FeedResult, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d for %s", resp.StatusCode, url)
	}

	fp := gofeed.NewParser()
	feed, err := fp.Parse(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse feed %s: %w", url, err)
	}

	result := &FeedResult{
		Title:   feed.Title,
		SiteURL: feed.Link,
	}

	if feed.Image != nil {
		result.IconURL = feed.Image.URL
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

		result.Items = append(result.Items, fi)
	}

	return result, nil
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

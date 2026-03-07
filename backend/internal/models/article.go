package models

import "time"

type Article struct {
	ID           int64      `json:"id"`
	FeedID       int64      `json:"feed_id"`
	GUID         string     `json:"guid"`
	Title        string     `json:"title"`
	URL          string     `json:"url"`
	Author       string     `json:"author"`
	ContentRaw   string     `json:"content_raw,omitempty"`
	ContentClean string     `json:"content_clean,omitempty"`
	ThumbnailURL string     `json:"thumbnail_url"`
	PublishedAt  *time.Time `json:"published_at"`
	FetchedAt    time.Time  `json:"fetched_at"`
	WordCount    int        `json:"word_count"`
	ReadingTime  int        `json:"reading_time"`
	IsRead       bool       `json:"is_read"`
	IsStarred    bool       `json:"is_starred"`
	ReadAt       *time.Time `json:"read_at"`
	Score        float64    `json:"score"`
	FeedTitle    string     `json:"feed_title,omitempty"`
	FeedIconURL  string     `json:"feed_icon_url,omitempty"`
	Tags         []string   `json:"tags,omitempty"`
}

type ArticleListParams struct {
	CategoryID *int64
	FeedID     *int64
	Status     string
	Sort       string
	Tag        string
	Page       int
	Limit      int
}

type UpdateArticleRequest struct {
	IsRead    *bool `json:"is_read,omitempty"`
	IsStarred *bool `json:"is_starred,omitempty"`
}

type BulkArticleRequest struct {
	ArticleIDs []int64 `json:"article_ids"`
	Action     string  `json:"action"`
}

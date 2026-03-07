package models

import "time"

type Feed struct {
	ID              int64      `json:"id"`
	UserID          int64      `json:"user_id"`
	URL             string     `json:"url"`
	Title           string     `json:"title"`
	SiteURL         string     `json:"site_url"`
	IconURL         string     `json:"icon_url"`
	CategoryID      *int64     `json:"category_id"`
	FetchInterval   int        `json:"fetch_interval"`
	LastFetched     *time.Time `json:"last_fetched"`
	EngagementScore float64    `json:"engagement_score"`
	CreatedAt       time.Time  `json:"created_at"`
	UnreadCount     int        `json:"unread_count,omitempty"`
}

type CreateFeedRequest struct {
	URL           string `json:"url"`
	CategoryID    *int64 `json:"category_id,omitempty"`
	FetchInterval int    `json:"fetch_interval,omitempty"`
}

type UpdateFeedRequest struct {
	Title         *string `json:"title,omitempty"`
	CategoryID    *int64  `json:"category_id,omitempty"`
	FetchInterval *int    `json:"fetch_interval,omitempty"`
}

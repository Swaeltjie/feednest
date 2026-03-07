package models

import "time"

type ReadingEvent struct {
	ID              int64     `json:"id"`
	ArticleID       int64     `json:"article_id"`
	EventType       string    `json:"event_type"`
	DurationSeconds int       `json:"duration_seconds"`
	CreatedAt       time.Time `json:"created_at"`
}

type CreateEventRequest struct {
	ArticleID       int64  `json:"article_id"`
	EventType       string `json:"event_type"`
	DurationSeconds int    `json:"duration_seconds"`
}

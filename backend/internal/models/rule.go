package models

import "time"

type FilterRule struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"user_id"`
	Name      string    `json:"name"`
	FeedID    *int64    `json:"feed_id"`
	Field     string    `json:"field"`
	Operator  string    `json:"operator"`
	Value     string    `json:"value"`
	Action    string    `json:"action"`
	Enabled   bool      `json:"enabled"`
	CreatedAt time.Time `json:"created_at"`
}

type CreateRuleRequest struct {
	Name     string `json:"name"`
	FeedID   *int64 `json:"feed_id,omitempty"`
	Field    string `json:"field"`
	Operator string `json:"operator"`
	Value    string `json:"value"`
	Action   string `json:"action"`
}

type UpdateRuleRequest struct {
	Name     *string `json:"name,omitempty"`
	FeedID   *int64  `json:"feed_id,omitempty"`
	Field    *string `json:"field,omitempty"`
	Operator *string `json:"operator,omitempty"`
	Value    *string `json:"value,omitempty"`
	Action   *string `json:"action,omitempty"`
	Enabled  *bool   `json:"enabled,omitempty"`
}

package models

type Category struct {
	ID       int64  `json:"id"`
	UserID   int64  `json:"user_id"`
	Name     string `json:"name"`
	Position int    `json:"position"`
}

type CreateCategoryRequest struct {
	Name     string `json:"name"`
	Position int    `json:"position"`
}

type UpdateCategoryRequest struct {
	Name     *string `json:"name,omitempty"`
	Position *int    `json:"position,omitempty"`
}

package models

import "time"

type Snippet struct {
	ID          int64     `json:"id"`
	UserID      int64     `json:"user_id"`
	Title       string    `json:"title"`
	Content     string    `json:"content"`
	Tags        *[]string `json:"tags,omitempty"` // could be empty
	Language    string    `json:"language"`
	IsFavorite  bool      `json:"is_favorite"`
	Description *string   `json:"description,omitempty"` // could be empty
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	FolderID    *int64    `json:"folder_id,omitempty"`
}

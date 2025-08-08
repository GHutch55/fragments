package models

import "time"

type Snippet struct {
	ID          int64     `json:"id"`
	UserID      int64     `json:"user_id"`
	Title       string    `json:"title"`
	Content     string    `json:"content"`
	Tags        *[]string  `json:"tags,omitempty"` // could be empty
	Language    string    `json:"language"`
	Description string    `json:"description,omitempty"` // could be empty
	CreatedAt   time.Time `json:"created_at"`
}

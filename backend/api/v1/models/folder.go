package models

import "time"

type Folder struct {
	ID          int64     `json:"id"`
	UserID      int64     `json:"user_id"`
	Name        string    `json:"name"`
	Description *string   `json:"description,omitempty"` // could be empty
	ParentID    *int64    `json:"parent_id,omitempty"`   // could be empty, this is for nested folder
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

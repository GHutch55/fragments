package models

import "time"

// User represents a user without sensitive data
type User struct {
	ID          int64     `json:"id"`
	Username    string    `json:"username"`
	CreatedAt   time.Time `json:"created_at,omitempty"`
	UpdatedAt   time.Time `json:"updated_at,omitempty"`
}

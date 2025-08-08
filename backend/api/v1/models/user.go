package models

import "time"

type User struct {
	ID          int64     `json:"id"`
	Username    string    `json:"username"`
	DisplayName *string    `json:"display_name"`
	CreatedAt   time.Time `json:"created_at,omitempty"`
	UpdatedAt   time.Time `json:"updated_at,omitempty"`
}

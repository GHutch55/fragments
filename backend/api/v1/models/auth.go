package models

import "time"

// LoginRequest represents a user login request
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// ChangePasswordRequest represents a password change request
type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

// UserResponse represents a user in API responses (without sensitive data)
type UserResponse struct {
	ID          int64     `json:"id"`
	Username    string    `json:"username"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// AuthResponse represents the response after successful authentication
type AuthResponse struct {
	Token string       `json:"token"`
	User  UserResponse `json:"user"`
}

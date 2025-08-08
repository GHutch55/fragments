package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/GHutch55/fragments/backend/api/v1/database"
	"github.com/GHutch55/fragments/backend/api/v1/models"
	"github.com/go-chi/chi/v5"
)

// UserHandler holds the database connection
type UserHandler struct {
	DB *sql.DB
}

// ErrorResponse represents a JSON error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}

// CreateUser handles user creation requests
func (h *UserHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	// Set response content type
	w.Header().Set("Content-Type", "application/json")

	var newUser models.User
	err := json.NewDecoder(r.Body).Decode(&newUser)
	if err != nil {
		h.sendError(w, "Invalid JSON format", http.StatusBadRequest)
		return
	}

	// Validate the user data
	if err := h.validateUser(&newUser); err != nil {
		h.sendError(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Create the user in the database
	err = database.CreateUser(h.DB, &newUser)
	if err != nil {
		if errors.Is(err, database.ErrUsernameExists) {
			h.sendError(w, "Username already exists", http.StatusConflict)
			return
		}
		if errors.Is(err, database.ErrDatabaseError) {
			h.sendError(w, "Unable to process request at this time", http.StatusInternalServerError)
			return
		}
		// Fallback for unexpected errors
		h.sendError(w, "An unexpected error occurred", http.StatusInternalServerError)
		return
	}

	// Success - return the created user
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(newUser)
}

// validateUser validates the user data before database operations
func (h *UserHandler) validateUser(user *models.User) error {
	// Validate username
	if strings.TrimSpace(user.Username) == "" {
		return errors.New("username is required")
	}

	// Clean up username (remove extra whitespace)
	user.Username = strings.TrimSpace(user.Username)

	// Username validation rules
	if utf8.RuneCountInString(user.Username) < 3 {
		return errors.New("username must be at least 3 characters long")
	}

	if utf8.RuneCountInString(user.Username) > 50 {
		return errors.New("username must be less than 50 characters")
	}

	// Basic username character validation (alphanumeric + underscore/hyphen)
	for _, r := range user.Username {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') || r == '_' || r == '-') {
			return errors.New("username can only contain letters, numbers, underscores, and hyphens")
		}
	}

	// Validate display_name (optional but has limits if provided)
	if user.DisplayName != nil {
		*user.DisplayName = strings.TrimSpace(*user.DisplayName)
		if utf8.RuneCountInString(*user.DisplayName) > 100 {
			return errors.New("display name must be less than 100 characters")
		}
		// If display_name is empty after trimming, set it to nil
		if *user.DisplayName == "" {
			user.DisplayName = nil
		}
	}

	return nil
}

func (h *UserHandler) GetUser(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	userIDStr := chi.URLParam(r, "id")
	if userIDStr == "" {
		h.sendError(w, "User ID is required", http.StatusBadRequest)
		return
	}

	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		h.sendError(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	var gotUser models.User
	err = database.GetUser(h.DB, userID, &gotUser)
	if err != nil {
		// Check if it's a "user not found" error
		if strings.Contains(err.Error(), "not found") {
			h.sendError(w, "User not found", http.StatusNotFound)
			return
		}
		// Check if it's a database error
		if errors.Is(err, database.ErrDatabaseError) {
			h.sendError(w, "Unable to process request at this time", http.StatusInternalServerError)
			return
		}
		// Fallback for unexpected errors
		h.sendError(w, "An unexpected error occurred", http.StatusInternalServerError)
		return
	}

	// Success
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(gotUser)
}

func (h *UserHandler) GetUsers(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Extract query parameters
	query := r.URL.Query()

	// Parse page parameter
	page := 1
	if pageStr := query.Get("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	// Parse limit parameter
	limit := 20 // default
	if limitStr := query.Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	// Get search parameter
	search := query.Get("search")

	// Call your database function using h.DB
	users, total, err := database.GetUsers(h.DB, page, limit, search)
	if err != nil {
		// Check if it's a database error
		if errors.Is(err, database.ErrDatabaseError) {
			h.sendError(w, "Unable to process request at this time", http.StatusInternalServerError)
			return
		}
		// Fallback for unexpected errors
		h.sendError(w, "An unexpected error occurred", http.StatusInternalServerError)
		return
	}

	// Calculate pagination metadata
	totalPages := (total + limit - 1) / limit // ceiling division
	hasNext := page < totalPages
	hasPrev := page > 1

	// Create response structure
	response := map[string]interface{}{
		"data": users,
		"pagination": map[string]interface{}{
			"page":        page,
			"limit":       limit,
			"total":       total,
			"total_pages": totalPages,
			"has_next":    hasNext,
			"has_prev":    hasPrev,
		},
	}

	// Success
	w.WriteHeader(http.StatusOK)

	// Encode and send response
	if err := json.NewEncoder(w).Encode(response); err != nil {
		fmt.Printf("Error encoding response: %v\n", err)
		h.sendError(w, "Unable to process request at this time", http.StatusInternalServerError)
		return
	}
}

// sendError sends a JSON error response
func (h *UserHandler) sendError(w http.ResponseWriter, message string, statusCode int) {
	w.WriteHeader(statusCode)
	response := ErrorResponse{
		Error:   http.StatusText(statusCode),
		Message: message,
	}
	json.NewEncoder(w).Encode(response)
}

package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"html"
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/GHutch55/fragments/backend/api/v1/database"
	"github.com/GHutch55/fragments/backend/api/v1/middleware"
	"github.com/GHutch55/fragments/backend/api/v1/models"
	"github.com/go-chi/chi/v5"
	"golang.org/x/crypto/bcrypt"
)

// UserHandler holds the database connection
type UserHandler struct {
	DB *sql.DB
}

// GetCurrentUser gets the authenticated user's information
func (h *UserHandler) GetCurrentUser(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Use consistent context access
	user, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		SendError(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(user)
}

// UpdateCurrentUser updates the authenticated user's information
func (h *UserHandler) UpdateCurrentUser(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Use consistent context access
	user, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		SendError(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	var updateUser models.User
	if err := json.NewDecoder(r.Body).Decode(&updateUser); err != nil {
		SendError(w, "Invalid JSON format", http.StatusBadRequest)
		return
	}

	if err := h.validateUser(&updateUser); err != nil {
		SendError(w, err.Error(), http.StatusBadRequest)
		return
	}

	err := database.UpdateUser(h.DB, user.ID, &updateUser)
	if err != nil {
		if database.IsUserNotFoundError(err) {
			SendError(w, "User not found", http.StatusNotFound)
			return
		}
		if database.IsUsernameExistsError(err) {
			SendError(w, "Username already exists", http.StatusConflict)
			return
		}
		if errors.Is(err, database.ErrDatabaseError) {
			SendError(w, "Unable to process request at this time", http.StatusInternalServerError)
			return
		}
		SendError(w, "An unexpected error occurred", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(updateUser)
}

// DeleteCurrentUser deletes the authenticated user's account
func (h *UserHandler) DeleteCurrentUser(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Use consistent context access
	user, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		SendError(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	err := database.DeleteUser(h.DB, user.ID)
	if err != nil {
		if database.IsUserNotFoundError(err) {
			SendError(w, "User not found", http.StatusNotFound)
			return
		}
		if errors.Is(err, database.ErrDatabaseError) {
			SendError(w, "Unable to process request at this time", http.StatusInternalServerError)
			return
		}
		SendError(w, "An unexpected error occurred", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// CreateUser handles user creation requests (for admin use)
func (h *UserHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var newUser struct {
		Username    string  `json:"username"`
		DisplayName *string `json:"display_name,omitempty"`
		Password    string  `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&newUser); err != nil {
		SendError(w, "Invalid JSON format", http.StatusBadRequest)
		return
	}

	// Validate password
	if err := h.validatePassword(newUser.Password); err != nil {
		SendError(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Create user model for validation
	userModel := models.User{
		Username:    newUser.Username,
		DisplayName: newUser.DisplayName,
	}

	// Validate the user data
	if err := h.validateUser(&userModel); err != nil {
		SendError(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Hash the password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newUser.Password), bcrypt.DefaultCost)
	if err != nil {
		SendError(w, "Unable to process password", http.StatusInternalServerError)
		return
	}

	// Create the user in the database
	userWithPassword := &database.UserWithPassword{
		User:     userModel,
		Password: string(hashedPassword),
	}

	err = database.CreateUserWithPassword(h.DB, userWithPassword)
	if err != nil {
		if database.IsUsernameExistsError(err) {
			SendError(w, "Username already exists", http.StatusConflict)
			return
		}
		if errors.Is(err, database.ErrDatabaseError) {
			SendError(w, "Unable to process request at this time", http.StatusInternalServerError)
			return
		}
		SendError(w, "An unexpected error occurred", http.StatusInternalServerError)
		return
	}

	// Return user without password
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(userWithPassword.User)
}

// GetUser retrieves a single user by ID (admin use)
func (h *UserHandler) GetUser(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	userIDStr := chi.URLParam(r, "id")
	if userIDStr == "" {
		SendError(w, "User ID is required", http.StatusBadRequest)
		return
	}

	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil || userID <= 0 {
		SendError(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	var gotUser models.User
	err = database.GetUser(h.DB, userID, &gotUser)
	if err != nil {
		if database.IsUserNotFoundError(err) {
			SendError(w, "User not found", http.StatusNotFound)
			return
		}
		if errors.Is(err, database.ErrDatabaseError) {
			SendError(w, "Unable to process request at this time", http.StatusInternalServerError)
			return
		}
		SendError(w, "An unexpected error occurred", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(gotUser)
}

// GetUsers retrieves users with pagination and search (admin use)
func (h *UserHandler) GetUsers(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	query := r.URL.Query()

	// Parse page parameter
	page := 1
	if pageStr := query.Get("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	// Parse limit parameter
	limit := 20
	if limitStr := query.Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	// Get search parameter
	search := strings.TrimSpace(query.Get("search"))

	users, total, err := database.GetUsers(h.DB, page, limit, search)
	if err != nil {
		if errors.Is(err, database.ErrDatabaseError) {
			SendError(w, "Unable to process request at this time", http.StatusInternalServerError)
			return
		}
		SendError(w, "An unexpected error occurred", http.StatusInternalServerError)
		return
	}

	// Calculate pagination metadata
	totalPages := (total + limit - 1) / limit
	hasNext := page < totalPages
	hasPrev := page > 1

	response := APIResponse{
		Data: users,
		Pagination: &PaginationInfo{
			Page:       page,
			Limit:      limit,
			Total:      total,
			TotalPages: totalPages,
			HasNext:    hasNext,
			HasPrev:    hasPrev,
		},
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// UpdateUser updates an existing user (admin use)
func (h *UserHandler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	userIDStr := chi.URLParam(r, "id")
	if userIDStr == "" {
		SendError(w, "User ID is required", http.StatusBadRequest)
		return
	}

	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil || userID <= 0 {
		SendError(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	var updateUser models.User
	if err := json.NewDecoder(r.Body).Decode(&updateUser); err != nil {
		SendError(w, "Invalid JSON format", http.StatusBadRequest)
		return
	}

	if err := h.validateUser(&updateUser); err != nil {
		SendError(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = database.UpdateUser(h.DB, userID, &updateUser)
	if err != nil {
		if database.IsUserNotFoundError(err) {
			SendError(w, "User not found", http.StatusNotFound)
			return
		}
		if database.IsUsernameExistsError(err) {
			SendError(w, "Username already exists", http.StatusConflict)
			return
		}
		if errors.Is(err, database.ErrDatabaseError) {
			SendError(w, "Unable to process request at this time", http.StatusInternalServerError)
			return
		}
		SendError(w, "An unexpected error occurred", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(updateUser)
}

// DeleteUser deletes a user by ID (admin use)
func (h *UserHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	userIDStr := chi.URLParam(r, "id")
	if userIDStr == "" {
		SendError(w, "User ID is required", http.StatusBadRequest)
		return
	}

	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil || userID <= 0 {
		SendError(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	err = database.DeleteUser(h.DB, userID)
	if err != nil {
		if database.IsUserNotFoundError(err) {
			SendError(w, "User not found", http.StatusNotFound)
			return
		}
		if errors.Is(err, database.ErrDatabaseError) {
			SendError(w, "Unable to process request at this time", http.StatusInternalServerError)
			return
		}
		SendError(w, "An unexpected error occurred", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// sanitizeString sanitizes user input to prevent XSS
func sanitizeString(s string) string {
	// Escape HTML entities
	s = html.EscapeString(s)
	// Trim whitespace
	s = strings.TrimSpace(s)
	return s
}

// validateUser validates the user data before database operations
func (h *UserHandler) validateUser(user *models.User) error {
	// Validate username
	if strings.TrimSpace(user.Username) == "" {
		return errors.New("username is required")
	}

	// Sanitize and clean up username
	user.Username = sanitizeString(user.Username)

	// Username validation rules
	if utf8.RuneCountInString(user.Username) < 3 {
		return errors.New("username must be at least 3 characters long")
	}

	if utf8.RuneCountInString(user.Username) > 50 {
		return errors.New("username must be less than 50 characters")
	}

	// Basic username character validation
	for _, r := range user.Username {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') || r == '_' || r == '-') {
			return errors.New("username can only contain letters, numbers, underscores, and hyphens")
		}
	}

	// Validate display_name (optional)
	if user.DisplayName != nil {
		*user.DisplayName = sanitizeString(*user.DisplayName)
		if utf8.RuneCountInString(*user.DisplayName) > 100 {
			return errors.New("display name must be less than 100 characters")
		}
		if *user.DisplayName == "" {
			user.DisplayName = nil
		}
	}

	return nil
}

// validatePassword validates password requirements
func (h *UserHandler) validatePassword(password string) error {
	if len(password) < 12 { // Increased from 8 to 12
		return errors.New("password must be at least 12 characters long")
	}

	if len(password) > 128 {
		return errors.New("password must be less than 128 characters long")
	}

	// Check for required character types
	var (
		hasUpper   = false
		hasLower   = false
		hasNumber  = false
		hasSpecial = false
	)

	for _, r := range password {
		switch {
		case r >= 'A' && r <= 'Z':
			hasUpper = true
		case r >= 'a' && r <= 'z':
			hasLower = true
		case r >= '0' && r <= '9':
			hasNumber = true
		case strings.ContainsRune("!@#$%^&*()_+-=[]{}|;:,.<>?", r):
			hasSpecial = true
		}
	}

	if !hasUpper {
		return errors.New("password must contain at least one uppercase letter")
	}

	if !hasLower {
		return errors.New("password must contain at least one lowercase letter")
	}

	if !hasNumber {
		return errors.New("password must contain at least one number")
	}

	if !hasSpecial {
		return errors.New("password must contain at least one special character")
	}

	return nil
}

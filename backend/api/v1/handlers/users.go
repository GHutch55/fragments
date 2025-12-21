package handlers

import (
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
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

// UserHandler holds the database connection
type UserHandler struct {
	DB *pgxpool.Pool
}

// GetCurrentUser gets the authenticated user's information
func (h *UserHandler) GetCurrentUser(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

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

	err := database.UpdateUser(r.Context(), h.DB, user.ID, &updateUser)
	if err != nil {
		switch {
		case database.IsUserNotFoundError(err):
			SendError(w, "User not found", http.StatusNotFound)
		case database.IsUsernameExistsError(err):
			SendError(w, "Username already exists", http.StatusConflict)
		case errors.Is(err, database.ErrDatabaseError):
			SendError(w, "Unable to process request at this time", http.StatusInternalServerError)
		default:
			SendError(w, "An unexpected error occurred", http.StatusInternalServerError)
		}
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(updateUser)
}

// DeleteCurrentUser deletes the authenticated user's account
func (h *UserHandler) DeleteCurrentUser(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	user, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		SendError(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	err := database.DeleteUser(r.Context(), h.DB, user.ID)
	if err != nil {
		switch {
		case database.IsUserNotFoundError(err):
			SendError(w, "User not found", http.StatusNotFound)
		case errors.Is(err, database.ErrDatabaseError):
			SendError(w, "Unable to process request at this time", http.StatusInternalServerError)
		default:
			SendError(w, "An unexpected error occurred", http.StatusInternalServerError)
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// CreateUser handles user creation requests (admin use)
func (h *UserHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var newUser struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&newUser); err != nil {
		SendError(w, "Invalid JSON format", http.StatusBadRequest)
		return
	}

	if err := h.validatePassword(newUser.Password); err != nil {
		SendError(w, err.Error(), http.StatusBadRequest)
		return
	}

	userModel := models.User{
		Username: newUser.Username,
	}

	if err := h.validateUser(&userModel); err != nil {
		SendError(w, err.Error(), http.StatusBadRequest)
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword(
		[]byte(newUser.Password),
		bcrypt.DefaultCost,
	)
	if err != nil {
		SendError(w, "Unable to process password", http.StatusInternalServerError)
		return
	}

	userWithPassword := &database.UserWithPassword{
		User:     userModel,
		Password: string(hashedPassword),
	}

	err = database.CreateUserWithPassword(r.Context(), h.DB, userWithPassword)
	if err != nil {
		switch {
		case database.IsUsernameExistsError(err):
			SendError(w, "Username already exists", http.StatusConflict)
		case errors.Is(err, database.ErrDatabaseError):
			SendError(w, "Unable to process request at this time", http.StatusInternalServerError)
		default:
			SendError(w, "An unexpected error occurred", http.StatusInternalServerError)
		}
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(userWithPassword.User)
}

// GetUser retrieves a single user by ID (admin use)
func (h *UserHandler) GetUser(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	idStr := chi.URLParam(r, "id")
	userID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || userID <= 0 {
		SendError(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	var user models.User
	err = database.GetUser(r.Context(), h.DB, userID, &user)
	if err != nil {
		switch {
		case database.IsUserNotFoundError(err):
			SendError(w, "User not found", http.StatusNotFound)
		case errors.Is(err, database.ErrDatabaseError):
			SendError(w, "Unable to process request at this time", http.StatusInternalServerError)
		default:
			SendError(w, "An unexpected error occurred", http.StatusInternalServerError)
		}
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(user)
}

// sanitizeString sanitizes user input to prevent XSS
func sanitizeString(s string) string {
	s = html.EscapeString(s)
	return strings.TrimSpace(s)
}

// validateUser validates the user data before database operations
func (h *UserHandler) validateUser(user *models.User) error {
	if strings.TrimSpace(user.Username) == "" {
		return errors.New("username is required")
	}

	user.Username = sanitizeString(user.Username)

	if utf8.RuneCountInString(user.Username) < 3 {
		return errors.New("username must be at least 3 characters long")
	}

	if utf8.RuneCountInString(user.Username) > 50 {
		return errors.New("username must be less than 50 characters")
	}

	for _, r := range user.Username {
		if !((r >= 'a' && r <= 'z') ||
			(r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') ||
			r == '_' || r == '-') {
			return errors.New("username can only contain letters, numbers, underscores, and hyphens")
		}
	}

	return nil
}

// validatePassword validates password requirements
func (h *UserHandler) validatePassword(password string) error {
	if len(password) < 12 {
		return errors.New("password must be at least 12 characters long")
	}

	if len(password) > 128 {
		return errors.New("password must be less than 128 characters long")
	}

	var hasUpper, hasLower, hasNumber, hasSpecial bool

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

	switch {
	case !hasUpper:
		return errors.New("password must contain at least one uppercase letter")
	case !hasLower:
		return errors.New("password must contain at least one lowercase letter")
	case !hasNumber:
		return errors.New("password must contain at least one number")
	case !hasSpecial:
		return errors.New("password must contain at least one special character")
	}

	return nil
}


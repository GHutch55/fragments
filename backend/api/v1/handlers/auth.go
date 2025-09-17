package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"html"
	"net/http"
	"strings"
	"time"

	"github.com/GHutch55/fragments/backend/api/v1/database"
	"github.com/GHutch55/fragments/backend/api/v1/middleware"
	"github.com/GHutch55/fragments/backend/api/v1/models"
	"golang.org/x/crypto/bcrypt"
)

type AuthHandler struct {
	DB             *sql.DB
	AuthMiddleware *middleware.AuthMiddleware
}

func NewAuthHandler(db *sql.DB, authMiddleware *middleware.AuthMiddleware) *AuthHandler {
	return &AuthHandler{
		DB:             db,
		AuthMiddleware: authMiddleware,
	}
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req struct {
		Username    string  `json:"username"`
		Password    string  `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		SendError(w, "Invalid JSON format", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if err := h.validateRegistration(req.Username, req.Password); err != nil {
		SendError(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Hash the password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		SendError(w, "Failed to process password", http.StatusInternalServerError)
		return
	}

	// Create user with password
	userWithPassword := &database.UserWithPassword{
		User: models.User{
			Username:    req.Username,
		},
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
		SendError(w, "Failed to create user", http.StatusInternalServerError)
		return
	}

	// Generate JWT token
	token, err := h.AuthMiddleware.GenerateToken(&userWithPassword.User)
	if err != nil {
		SendError(w, "Failed to generate authentication token", http.StatusInternalServerError)
		return
	}

	// Create response
	userResponse := models.UserResponse{
		ID:          userWithPassword.ID,
		Username:    userWithPassword.Username,
		CreatedAt:   userWithPassword.CreatedAt,
		UpdatedAt:   userWithPassword.UpdatedAt,
	}

	response := models.AuthResponse{
		Token: token,
		User:  userResponse,
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var loginReq models.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&loginReq); err != nil {
		SendError(w, "Invalid JSON format", http.StatusBadRequest)
		return
	}

	// Validate login request
	if err := h.validateLogin(&loginReq); err != nil {
		SendError(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Get user by username
	user, err := database.GetUserByUsername(h.DB, loginReq.Username)
	if err != nil {
		// Use generic message to prevent username enumeration
		time.Sleep(100 * time.Millisecond) // delay to prevent timing attacks
		SendError(w, "Invalid username or password", http.StatusUnauthorized)
		return
	}

	// Verify password
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(loginReq.Password))
	if err != nil {
		// Add a small delay to prevent timing attacks
		time.Sleep(100 * time.Millisecond)
		SendError(w, "Invalid username or password", http.StatusUnauthorized)
		return
	}

	// Generate JWT token
	token, err := h.AuthMiddleware.GenerateToken(&user.User)
	if err != nil {
		SendError(w, "Failed to generate authentication token", http.StatusInternalServerError)
		return
	}

	// Create response
	userResponse := models.UserResponse{
		ID:          user.ID,
		Username:    user.Username,
		CreatedAt:   user.CreatedAt,
		UpdatedAt:   user.UpdatedAt,
	}

	response := models.AuthResponse{
		Token: token,
		User:  userResponse,
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	user, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		SendError(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	userResponse := models.UserResponse{
		ID:          user.ID,
		Username:    user.Username,
		CreatedAt:   user.CreatedAt,
		UpdatedAt:   user.UpdatedAt,
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(userResponse)
}

func (h *AuthHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	user, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		SendError(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	var changePasswordReq models.ChangePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&changePasswordReq); err != nil {
		SendError(w, "Invalid JSON format", http.StatusBadRequest)
		return
	}

	// Validate request
	if err := h.validateChangePassword(&changePasswordReq); err != nil {
		SendError(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Get current user with password from database
	currentUser, err := database.GetUserByUsername(h.DB, user.Username)
	if err != nil {
		SendError(w, "Unable to verify current password", http.StatusInternalServerError)
		return
	}

	// Verify current password
	err = bcrypt.CompareHashAndPassword([]byte(currentUser.Password), []byte(changePasswordReq.CurrentPassword))
	if err != nil {
		SendError(w, "Current password is incorrect", http.StatusUnauthorized)
		return
	}

	// Hash new password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(changePasswordReq.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		SendError(w, "Failed to process new password", http.StatusInternalServerError)
		return
	}

	// Update password in database
	err = database.UpdateUserPassword(h.DB, user.ID, string(hashedPassword))
	if err != nil {
		SendError(w, "Failed to update password", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Password updated successfully",
	})
}

// validateRegistration validates registration input
func (h *AuthHandler) validateRegistration(username, password string) error {
	username = html.EscapeString(strings.TrimSpace(username)) // sanitization

	if strings.TrimSpace(username) == "" {
		return errors.New("username is required")
	}

	if strings.TrimSpace(password) == "" {
		return errors.New("password is required")
	}

	// Password strength validation
	if len(password) < 12 {
		return errors.New("password must be at least 12 characters long")
	}

	// Additional password strength checks
	if !h.isStrongPassword(password) {
		return errors.New("password must contain at least one uppercase letter, one lowercase letter, one number, and one special character")
	}

	return nil
}

// validateLogin validates login input
func (h *AuthHandler) validateLogin(req *models.LoginRequest) error {
	if strings.TrimSpace(req.Username) == "" {
		return errors.New("username is required")
	}

	if strings.TrimSpace(req.Password) == "" {
		return errors.New("password is required")
	}

	return nil
}

// validateChangePassword validates change password input
func (h *AuthHandler) validateChangePassword(req *models.ChangePasswordRequest) error {
	if strings.TrimSpace(req.CurrentPassword) == "" {
		return errors.New("current password is required")
	}

	if strings.TrimSpace(req.NewPassword) == "" {
		return errors.New("new password is required")
	}

	if len(req.NewPassword) < 12 {
		return errors.New("password must be at least 12 characters long")
	}

	if req.CurrentPassword == req.NewPassword {
		return errors.New("new password must be different from current password")
	}

	if !h.isStrongPassword(req.NewPassword) {
		return errors.New("password must contain at least one uppercase letter, one lowercase letter, one number, and one special character")
	}

	return nil
}

// isStrongPassword checks if password meets strength requirements
func (h *AuthHandler) isStrongPassword(password string) bool {
	if len(password) < 12 { // Changed from implicit 8+ to 12+
		return false
	}

	hasUpper := false
	hasLower := false
	hasNumber := false
	hasSpecial := false // Add this

	for _, char := range password {
		switch {
		case char >= 'A' && char <= 'Z':
			hasUpper = true
		case char >= 'a' && char <= 'z':
			hasLower = true
		case char >= '0' && char <= '9':
			hasNumber = true
		case strings.ContainsRune("!@#$%^&*()_+-=[]{}|;:,.<>?", char): // Add this
			hasSpecial = true
		}
	}

	return hasUpper && hasLower && hasNumber && hasSpecial // Add hasSpecial
}

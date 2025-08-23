package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/GHutch55/fragments/backend/api/v1/database"
	"github.com/GHutch55/fragments/backend/api/v1/models"
	"github.com/go-chi/chi/v5"
)

type SnippetHandler struct {
	DB *sql.DB
}

func (h *SnippetHandler) CreateSnippet(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var newSnippet models.Snippet
	err := json.NewDecoder(r.Body).Decode(&newSnippet)
	if err != nil {
		h.sendError(w, "Invalid JSON format", http.StatusBadRequest)
		return
	}

	if err := h.validateSnippet(&newSnippet); err != nil {
		h.sendError(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Validate that the user exists
	if err := h.validateUserExists(newSnippet.UserID); err != nil {
		if errors.Is(err, database.ErrNoUserError) {
			h.sendError(w, "User does not exist", http.StatusBadRequest)
			return
		}
		h.sendError(w, "Unable to validate user", http.StatusInternalServerError)
		return
	}

	err = database.CreateSnippet(h.DB, &newSnippet)
	if err != nil {
		if errors.Is(err, database.ErrDatabaseError) {
			h.sendError(w, "Unable to process request at this time", http.StatusInternalServerError)
			return
		}

		h.sendError(w, "An unexpected error occurred", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(newSnippet)
}

func (h *SnippetHandler) GetSnippet(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	snippetIDStr := chi.URLParam(r, "id")
	if snippetIDStr == "" {
		h.sendError(w, "Snippet ID is required", http.StatusBadRequest)
		return
	}

	snippetID, err := strconv.ParseInt(snippetIDStr, 10, 64)
	if err != nil {
		h.sendError(w, "Invalid snippet ID", http.StatusBadRequest)
		return
	}

	gotSnippet, err := database.GetSnippet(h.DB, snippetID)
	if err != nil {
		if errors.Is(err, database.ErrNoSnippetError) {
			h.sendError(w, "Snippet not found", http.StatusNotFound)
			return
		}

		if errors.Is(err, database.ErrDatabaseError) {
			h.sendError(w, "Unable to process request at this time", http.StatusInternalServerError)
			return
		}

		h.sendError(w, "An unexpected error occurred", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(gotSnippet)
}

// GetSnippets gets snippets with optional user filtering
func (h *SnippetHandler) GetSnippets(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	query := r.URL.Query()

	// Parse user_id parameter (optional - if not provided, get all snippets)
	var userID *int64
	if userIDStr := query.Get("user_id"); userIDStr != "" {
		if id, err := strconv.ParseInt(userIDStr, 10, 64); err == nil && id > 0 {
			// Validate that the user exists
			if err := h.validateUserExists(id); err != nil {
				if errors.Is(err, database.ErrNoUserError) {
					h.sendError(w, "User does not exist", http.StatusBadRequest)
					return
				}
				h.sendError(w, "Unable to validate user", http.StatusInternalServerError)
				return
			}
			userID = &id
		} else {
			h.sendError(w, "Invalid user_id parameter", http.StatusBadRequest)
			return
		}
	}

	page := 1
	if pageStr := query.Get("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	limit := 20
	if limitStr := query.Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	search := query.Get("search")

	snippets, total, err := database.GetSnippetsWithOptionalUser(h.DB, page, limit, userID, search)
	if err != nil {
		if errors.Is(err, database.ErrDatabaseError) {
			h.sendError(w, "Unable to process request at this time", http.StatusInternalServerError)
			return
		}
		h.sendError(w, "An unexpected error occurred", http.StatusInternalServerError)
		return
	}

	totalPages := (total + limit - 1) / limit // ceiling division
	hasNext := page < totalPages
	hasPrev := page > 1

	response := map[string]interface{}{
		"data": snippets,
		"pagination": map[string]interface{}{
			"page":        page,
			"limit":       limit,
			"total":       total,
			"total_pages": totalPages,
			"has_next":    hasNext,
			"has_prev":    hasPrev,
		},
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// GetUserSnippets gets all snippets for a specific user (alternative endpoint)
func (h *SnippetHandler) GetUserSnippets(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	userIDStr := chi.URLParam(r, "user_id")
	if userIDStr == "" {
		h.sendError(w, "User ID is required", http.StatusBadRequest)
		return
	}

	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil || userID <= 0 {
		h.sendError(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	// Validate that the user exists
	if err := h.validateUserExists(userID); err != nil {
		if errors.Is(err, database.ErrNoUserError) {
			h.sendError(w, "User does not exist", http.StatusNotFound)
			return
		}
		h.sendError(w, "Unable to validate user", http.StatusInternalServerError)
		return
	}

	query := r.URL.Query()

	page := 1
	if pageStr := query.Get("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	limit := 20
	if limitStr := query.Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	search := query.Get("search")

	snippets, total, err := database.GetSnippets(h.DB, page, limit, userID, search)
	if err != nil {
		if errors.Is(err, database.ErrDatabaseError) {
			h.sendError(w, "Unable to process request at this time", http.StatusInternalServerError)
			return
		}
		h.sendError(w, "An unexpected error occurred", http.StatusInternalServerError)
		return
	}

	totalPages := (total + limit - 1) / limit
	hasNext := page < totalPages
	hasPrev := page > 1

	response := map[string]interface{}{
		"data": snippets,
		"pagination": map[string]interface{}{
			"page":        page,
			"limit":       limit,
			"total":       total,
			"total_pages": totalPages,
			"has_next":    hasNext,
			"has_prev":    hasPrev,
		},
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func (h *SnippetHandler) UpdateSnippet(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	snippetIDStr := chi.URLParam(r, "id")
	if snippetIDStr == "" {
		h.sendError(w, "Snippet ID is required", http.StatusBadRequest)
		return
	}

	snippetID, err := strconv.ParseInt(snippetIDStr, 10, 64)
	if err != nil || snippetID <= 0 {
		h.sendError(w, "Invalid snippet ID", http.StatusBadRequest)
		return
	}

	var updateSnippet models.Snippet
	err = json.NewDecoder(r.Body).Decode(&updateSnippet)
	if err != nil {
		h.sendError(w, "Invalid JSON format", http.StatusBadRequest)
		return
	}

	if err := h.validateSnippet(&updateSnippet); err != nil {
		h.sendError(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Validate that the user exists
	if err := h.validateUserExists(updateSnippet.UserID); err != nil {
		if errors.Is(err, database.ErrNoUserError) {
			h.sendError(w, "User does not exist", http.StatusBadRequest)
			return
		}
		h.sendError(w, "Unable to validate user", http.StatusInternalServerError)
		return
	}

	err = database.UpdateSnippet(h.DB, snippetID, &updateSnippet)
	if err != nil {
		if errors.Is(err, database.ErrNoSnippetError) {
			h.sendError(w, "Snippet not found", http.StatusNotFound)
			return
		}
		if errors.Is(err, database.ErrDatabaseError) {
			h.sendError(w, "Unable to process request at this time", http.StatusInternalServerError)
			return
		}
		h.sendError(w, "An unexpected error occurred", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(updateSnippet)
}

func (h *SnippetHandler) DeleteSnippet(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	snippetIDStr := chi.URLParam(r, "id")
	if snippetIDStr == "" {
		h.sendError(w, "Snippet ID is required", http.StatusBadRequest)
		return
	}

	snippetID, err := strconv.ParseInt(snippetIDStr, 10, 64)
	if err != nil || snippetID <= 0 {
		h.sendError(w, "Invalid snippet ID", http.StatusBadRequest)
		return
	}

	err = database.DeleteSnippet(h.DB, snippetID)
	if err != nil {
		if errors.Is(err, database.ErrNoSnippetError) {
			h.sendError(w, "Snippet not found", http.StatusNotFound)
			return
		}
		if errors.Is(err, database.ErrDatabaseError) {
			h.sendError(w, "Unable to process request at this time", http.StatusInternalServerError)
			return
		}
		h.sendError(w, "An unexpected error occurred", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// validateUserExists checks if a user exists in the database
func (h *SnippetHandler) validateUserExists(userID int64) error {
	var user models.User
	return database.GetUser(h.DB, userID, &user)
}

// validateSnippet validates the snippet data before database operations
func (h *SnippetHandler) validateSnippet(snippet *models.Snippet) error {
	// Validate title
	if strings.TrimSpace(snippet.Title) == "" {
		return errors.New("title is required")
	}

	// Clean up title (remove extra whitespace)
	snippet.Title = strings.TrimSpace(snippet.Title)

	// Title length validation
	if utf8.RuneCountInString(snippet.Title) > 200 {
		return errors.New("title must be less than 200 characters")
	}

	// Validate content
	if strings.TrimSpace(snippet.Content) == "" {
		return errors.New("content is required")
	}

	// Clean up content (remove extra leading/trailing whitespace but preserve internal formatting)
	snippet.Content = strings.TrimSpace(snippet.Content)

	// Content length validation (adjust limit as needed)
	if utf8.RuneCountInString(snippet.Content) > 1000000 {
		return errors.New("content must be less than 1 million characters")
	}

	// Validate language
	if strings.TrimSpace(snippet.Language) == "" {
		return errors.New("language is required")
	}

	// Clean up language
	snippet.Language = strings.TrimSpace(snippet.Language)
	snippet.Language = strings.ToLower(snippet.Language)

	// Language validation - only allow alphanumeric, hyphens, and plus signs
	for _, r := range snippet.Language {
		if !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '+') {
			return errors.New("language can only contain lowercase letters, numbers, hyphens, and plus signs")
		}
	}

	if utf8.RuneCountInString(snippet.Language) > 50 {
		return errors.New("language must be less than 50 characters")
	}

	// Validate description (optional but has limits if provided)
	if snippet.Description != nil {
		*snippet.Description = strings.TrimSpace(*snippet.Description)
		if utf8.RuneCountInString(*snippet.Description) > 500 {
			return errors.New("description must be less than 500 characters")
		}
		// If description is empty after trimming, set it to nil
		if *snippet.Description == "" {
			snippet.Description = nil
		}
	}

	// Validate user_id (should be positive)
	if snippet.UserID <= 0 {
		return errors.New("valid user ID is required")
	}

	// Validate folder_id (optional but must be positive if provided)
	if snippet.FolderID != nil && *snippet.FolderID <= 0 {
		return errors.New("folder ID must be positive if provided")
	}

	// Validate tags (optional but has limits if provided)
	if snippet.Tags != nil {
		if len(*snippet.Tags) > 20 {
			return errors.New("snippet cannot have more than 20 tags")
		}

		// Validate each tag
		for i, tag := range *snippet.Tags {
			// Clean up tag
			(*snippet.Tags)[i] = strings.TrimSpace(tag)
			cleanTag := (*snippet.Tags)[i]

			if cleanTag == "" {
				return errors.New("tags cannot be empty")
			}

			if utf8.RuneCountInString(cleanTag) > 50 {
				return errors.New("each tag must be less than 50 characters")
			}

			// Basic tag character validation (alphanumeric + underscore/hyphen/space)
			for _, r := range cleanTag {
				if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
					(r >= '0' && r <= '9') || r == '_' || r == '-' || r == ' ') {
					return errors.New("tags can only contain letters, numbers, underscores, hyphens, and spaces")
				}
			}
		}

		// Remove duplicate tags
		tagMap := make(map[string]bool)
		uniqueTags := []string{}
		for _, tag := range *snippet.Tags {
			lowerTag := strings.ToLower(tag)
			if !tagMap[lowerTag] {
				tagMap[lowerTag] = true
				uniqueTags = append(uniqueTags, tag)
			}
		}
		*snippet.Tags = uniqueTags
	}

	return nil
}

// sendError sends a JSON error response
func (h *SnippetHandler) sendError(w http.ResponseWriter, message string, statusCode int) {
	w.WriteHeader(statusCode)
	response := ErrorResponse{
		Error:   http.StatusText(statusCode),
		Message: message,
	}
	json.NewEncoder(w).Encode(response)
}

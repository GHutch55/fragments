package handlers

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/GHutch55/fragments/backend/api/v1/database"
	"github.com/GHutch55/fragments/backend/api/v1/middleware"
	"github.com/GHutch55/fragments/backend/api/v1/models"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	MaxTitleLength       = 200
	MaxContentLength     = 1000000
	MaxLanguageLength    = 50
	MaxDescriptionLength = 500
	MaxTagLength         = 50
	MaxTagsPerSnippet    = 20
)

type SnippetHandler struct {
	DB *pgxpool.Pool
}

func (h *SnippetHandler) CreateSnippet(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Get authenticated user
	user, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		SendError(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	var newSnippet models.Snippet
	err := json.NewDecoder(r.Body).Decode(&newSnippet)
	if err != nil {
		SendError(w, "Invalid JSON format", http.StatusBadRequest)
		return
	}

	// Set user ID from authenticated user (prevent user ID spoofing)
	newSnippet.UserID = user.ID

	if err := h.validateSnippet(&newSnippet); err != nil {
		SendError(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = database.CreateSnippet(r.Context(), h.DB, &newSnippet)
	if err != nil {
		log.Printf("Error creating snippet in database: %v", err)
		if errors.Is(err, database.ErrDatabaseError) {
			SendError(w, "Unable to process request at this time", http.StatusInternalServerError)
			return
		}
		SendError(w, "An unexpected error occurred", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(newSnippet)
}

func (h *SnippetHandler) GetSnippet(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Get authenticated user
	user, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		SendError(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	snippetIDStr := chi.URLParam(r, "id")
	if snippetIDStr == "" {
		SendError(w, "Snippet ID is required", http.StatusBadRequest)
		return
	}

	snippetID, err := strconv.ParseInt(snippetIDStr, 10, 64)
	if err != nil {
		SendError(w, "Invalid snippet ID", http.StatusBadRequest)
		return
	}

	gotSnippet, err := database.GetSnippet(r.Context(), h.DB, snippetID)
	if err != nil {
		if errors.Is(err, database.ErrNoSnippetError) {
			SendError(w, "Snippet not found", http.StatusNotFound)
			return
		}
		if errors.Is(err, database.ErrDatabaseError) {
			SendError(w, "Unable to process request at this time", http.StatusInternalServerError)
			return
		}
		SendError(w, "An unexpected error occurred", http.StatusInternalServerError)
		return
	}

	// Verify user owns this snippet
	if gotSnippet.UserID != user.ID {
		SendError(w, "Snippet not found", http.StatusNotFound) // Don't reveal existence
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(gotSnippet)
}

func (h *SnippetHandler) GetSnippets(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Get authenticated user
	user, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		SendError(w, "Authentication required", http.StatusUnauthorized)
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

	// Only get snippets for the authenticated user
	snippets, total, err := database.GetSnippets(r.Context(), h.DB, page, limit, user.ID, search)
	if err != nil {
		if errors.Is(err, database.ErrDatabaseError) {
			SendError(w, "Unable to process request at this time", http.StatusInternalServerError)
			return
		}
		SendError(w, "An unexpected error occurred", http.StatusInternalServerError)
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

	// Get authenticated user
	user, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		SendError(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	snippetIDStr := chi.URLParam(r, "id")
	if snippetIDStr == "" {
		SendError(w, "Snippet ID is required", http.StatusBadRequest)
		return
	}

	snippetID, err := strconv.ParseInt(snippetIDStr, 10, 64)
	if err != nil || snippetID <= 0 {
		SendError(w, "Invalid snippet ID", http.StatusBadRequest)
		return
	}

	// Check if snippet exists and user owns it
	existingSnippet, err := database.GetSnippet(r.Context(), h.DB, snippetID)
	if err != nil {
		if errors.Is(err, database.ErrNoSnippetError) {
			SendError(w, "Snippet not found", http.StatusNotFound)
			return
		}
		if errors.Is(err, database.ErrDatabaseError) {
			SendError(w, "Unable to process request at this time", http.StatusInternalServerError)
			return
		}
		SendError(w, "An unexpected error occurred", http.StatusInternalServerError)
		return
	}

	// Verify user owns this snippet
	if existingSnippet.UserID != user.ID {
		SendError(w, "Snippet not found", http.StatusNotFound) // Don't reveal existence
		return
	}

	var updateSnippet models.Snippet
	err = json.NewDecoder(r.Body).Decode(&updateSnippet)
	if err != nil {
		SendError(w, "Invalid JSON format", http.StatusBadRequest)
		return
	}

	// Set user ID from authenticated user (prevent user ID spoofing)
	updateSnippet.UserID = user.ID

	if err := h.validateSnippet(&updateSnippet); err != nil {
		SendError(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = database.UpdateSnippet(r.Context(), h.DB, snippetID, &updateSnippet)
	if err != nil {
		if errors.Is(err, database.ErrNoSnippetError) {
			SendError(w, "Snippet not found", http.StatusNotFound)
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
	json.NewEncoder(w).Encode(updateSnippet)
}

func (h *SnippetHandler) DeleteSnippet(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Get authenticated user
	user, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		SendError(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	snippetIDStr := chi.URLParam(r, "id")
	if snippetIDStr == "" {
		SendError(w, "Snippet ID is required", http.StatusBadRequest)
		return
	}

	snippetID, err := strconv.ParseInt(snippetIDStr, 10, 64)
	if err != nil || snippetID <= 0 {
		SendError(w, "Invalid snippet ID", http.StatusBadRequest)
		return
	}

	// Check if snippet exists and user owns it
	existingSnippet, err := database.GetSnippet(r.Context(), h.DB, snippetID)
	if err != nil {
		if errors.Is(err, database.ErrNoSnippetError) {
			SendError(w, "Snippet not found", http.StatusNotFound)
			return
		}
		if errors.Is(err, database.ErrDatabaseError) {
			SendError(w, "Unable to process request at this time", http.StatusInternalServerError)
			return
		}
		SendError(w, "An unexpected error occurred", http.StatusInternalServerError)
		return
	}

	// Verify user owns this snippet
	if existingSnippet.UserID != user.ID {
		SendError(w, "Snippet not found", http.StatusNotFound) // Don't reveal existence
		return
	}

	err = database.DeleteSnippet(r.Context(), h.DB, snippetID)
	if err != nil {
		if errors.Is(err, database.ErrNoSnippetError) {
			SendError(w, "Snippet not found", http.StatusNotFound)
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

func (h *SnippetHandler) validateSnippet(snippet *models.Snippet) error {
	// Validate title
	if strings.TrimSpace(snippet.Title) == "" {
		return errors.New("title is required")
	}

	// Clean up title
	snippet.Title = strings.TrimSpace(snippet.Title)

	// Title length validation
	if utf8.RuneCountInString(snippet.Title) > MaxTitleLength {
		return errors.New("title must be less than 200 characters")
	}

	// Validate content
	if strings.TrimSpace(snippet.Content) == "" {
		return errors.New("content is required")
	}

	// Clean up content
	snippet.Content = strings.TrimSpace(snippet.Content)

	// Content length validation
	if utf8.RuneCountInString(snippet.Content) > MaxContentLength {
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

	if utf8.RuneCountInString(snippet.Language) > MaxLanguageLength {
		return errors.New("language must be less than 50 characters")
	}

	// Validate description (optional)
	if snippet.Description != nil {
		*snippet.Description = strings.TrimSpace(*snippet.Description)
		if utf8.RuneCountInString(*snippet.Description) > MaxDescriptionLength {
			return errors.New("description must be less than 500 characters")
		}
		// If description is empty after trimming, set it to nil
		if *snippet.Description == "" {
			snippet.Description = nil
		}
	}

	// Validate user_id (should be positive) - this is set from auth context
	if snippet.UserID <= 0 {
		return errors.New("valid user ID is required")
	}

	// Validate folder_id (optional but must be positive if provided)
	if snippet.FolderID != nil && *snippet.FolderID <= 0 {
		return errors.New("folder ID must be positive if provided")
	}

	// Validate tags (optional)
	if snippet.Tags != nil {
		if len(*snippet.Tags) > MaxTagsPerSnippet {
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

			if utf8.RuneCountInString(cleanTag) > MaxTagLength {
				return errors.New("each tag must be less than 50 characters")
			}

			// Basic tag character validation
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

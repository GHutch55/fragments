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
	"github.com/GHutch55/fragments/backend/api/v1/middleware"
	"github.com/GHutch55/fragments/backend/api/v1/models"
	"github.com/go-chi/chi/v5"
)

const (
	MaxFolderNameLength        = 100
	MaxFolderDescriptionLength = 500
)

type FolderHandler struct {
	DB *sql.DB
}

func (h *FolderHandler) CreateFolder(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Get authenticated user
	user, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		SendError(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	var newFolder models.Folder
	err := json.NewDecoder(r.Body).Decode(&newFolder)
	if err != nil {
		SendError(w, "Invalid JSON format", http.StatusBadRequest)
		return
	}

	// Set user ID from authenticated user (prevent user ID spoofing)
	newFolder.UserID = user.ID

	if err := h.validateFolder(&newFolder); err != nil {
		SendError(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = database.CreateFolder(h.DB, &newFolder)
	if err != nil {
		if strings.Contains(err.Error(), "already exists") {
			SendError(w, "Folder name already exists in this location", http.StatusConflict)
			return
		}
		if strings.Contains(err.Error(), "parent folder") {
			SendError(w, "Invalid parent folder", http.StatusBadRequest)
			return
		}
		if strings.Contains(err.Error(), "circular reference") {
			SendError(w, "Cannot create folder: circular reference detected", http.StatusBadRequest)
			return
		}
		if errors.Is(err, database.ErrDatabaseError) {
			SendError(w, "Unable to process request at this time", http.StatusInternalServerError)
			return
		}
		SendError(w, "An unexpected error occurred", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(newFolder)
}

func (h *FolderHandler) GetFolder(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Get authenticated user
	user, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		SendError(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	folderIDStr := chi.URLParam(r, "id")
	if folderIDStr == "" {
		SendError(w, "Folder ID is required", http.StatusBadRequest)
		return
	}

	folderID, err := strconv.ParseInt(folderIDStr, 10, 64)
	if err != nil {
		SendError(w, "Invalid folder ID", http.StatusBadRequest)
		return
	}

	gotFolder, err := database.GetFolder(h.DB, folderID)
	if err != nil {
		if errors.Is(err, database.ErrNoFolderError) {
			SendError(w, "Folder not found", http.StatusNotFound)
			return
		}
		if errors.Is(err, database.ErrDatabaseError) {
			SendError(w, "Unable to process request at this time", http.StatusInternalServerError)
			return
		}
		SendError(w, "An unexpected error occurred", http.StatusInternalServerError)
		return
	}

	// Verify user owns this folder
	if gotFolder.UserID != user.ID {
		SendError(w, "Folder not found", http.StatusNotFound) // Don't reveal existence
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(gotFolder)
}

func (h *FolderHandler) GetFolders(w http.ResponseWriter, r *http.Request) {
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

	// Parse parent_id parameter (optional)
	var parentID *int64
	if parentIDStr := query.Get("parent_id"); parentIDStr != "" {
		if pid, err := strconv.ParseInt(parentIDStr, 10, 64); err == nil {
			parentID = &pid
		} else {
			SendError(w, "Invalid parent_id parameter", http.StatusBadRequest)
			return
		}
	}

	// Only get folders for the authenticated user
	folders, total, err := database.GetFolders(h.DB, page, limit, user.ID, parentID)
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
		"data": folders,
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

func (h *FolderHandler) UpdateFolder(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Get authenticated user
	user, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		SendError(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	folderIDStr := chi.URLParam(r, "id")
	if folderIDStr == "" {
		SendError(w, "Folder ID is required", http.StatusBadRequest)
		return
	}

	folderID, err := strconv.ParseInt(folderIDStr, 10, 64)
	if err != nil || folderID <= 0 {
		SendError(w, "Invalid folder ID", http.StatusBadRequest)
		return
	}

	// Check if folder exists and user owns it
	existingFolder, err := database.GetFolder(h.DB, folderID)
	if err != nil {
		if errors.Is(err, database.ErrNoFolderError) {
			SendError(w, "Folder not found", http.StatusNotFound)
			return
		}
		if errors.Is(err, database.ErrDatabaseError) {
			SendError(w, "Unable to process request at this time", http.StatusInternalServerError)
			return
		}
		SendError(w, "An unexpected error occurred", http.StatusInternalServerError)
		return
	}

	// Verify user owns this folder
	if existingFolder.UserID != user.ID {
		SendError(w, "Folder not found", http.StatusNotFound) // Don't reveal existence
		return
	}

	var updateFolder models.Folder
	err = json.NewDecoder(r.Body).Decode(&updateFolder)
	if err != nil {
		SendError(w, "Invalid JSON format", http.StatusBadRequest)
		return
	}

	// Set user ID from authenticated user (prevent user ID spoofing)
	updateFolder.UserID = user.ID

	if err := h.validateFolder(&updateFolder); err != nil {
		SendError(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = database.UpdateFolder(h.DB, folderID, &updateFolder)
	if err != nil {
		if errors.Is(err, database.ErrNoFolderError) {
			SendError(w, "Folder not found", http.StatusNotFound)
			return
		}
		if strings.Contains(err.Error(), "already exists") {
			SendError(w, "Folder name already exists in this location", http.StatusConflict)
			return
		}
		if strings.Contains(err.Error(), "parent folder") {
			SendError(w, "Invalid parent folder", http.StatusBadRequest)
			return
		}
		if strings.Contains(err.Error(), "circular reference") {
			SendError(w, "Cannot update folder: circular reference detected", http.StatusBadRequest)
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
	json.NewEncoder(w).Encode(updateFolder)
}

func (h *FolderHandler) DeleteFolder(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Get authenticated user
	user, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		SendError(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	folderIDStr := chi.URLParam(r, "id")
	if folderIDStr == "" {
		SendError(w, "Folder ID is required", http.StatusBadRequest)
		return
	}

	folderID, err := strconv.ParseInt(folderIDStr, 10, 64)
	if err != nil || folderID <= 0 {
		SendError(w, "Invalid folder ID", http.StatusBadRequest)
		return
	}

	// Check if folder exists and user owns it
	existingFolder, err := database.GetFolder(h.DB, folderID)
	if err != nil {
		if errors.Is(err, database.ErrNoFolderError) {
			SendError(w, "Folder not found", http.StatusNotFound)
			return
		}
		if errors.Is(err, database.ErrDatabaseError) {
			SendError(w, "Unable to process request at this time", http.StatusInternalServerError)
			return
		}
		SendError(w, "An unexpected error occurred", http.StatusInternalServerError)
		return
	}

	// Verify user owns this folder
	if existingFolder.UserID != user.ID {
		SendError(w, "Folder not found", http.StatusNotFound) // Don't reveal existence
		return
	}

	err = database.DeleteFolder(h.DB, folderID)
	if err != nil {
		if errors.Is(err, database.ErrNoFolderError) {
			SendError(w, "Folder not found", http.StatusNotFound)
			return
		}
		if errors.Is(err, database.ErrFolderHasChildren) {
			SendError(w, "Cannot delete folder: folder contains subfolders", http.StatusConflict)
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

func (h *FolderHandler) validateFolder(folder *models.Folder) error {
	// Validate name
	if strings.TrimSpace(folder.Name) == "" {
		return errors.New("folder name is required")
	}

	// Clean up name
	folder.Name = strings.TrimSpace(folder.Name)

	// Name length validation
	if utf8.RuneCountInString(folder.Name) > MaxFolderNameLength {
		return errors.New("folder name must be less than 100 characters")
	}

	// Basic folder name character validation
	for _, r := range folder.Name {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') || r == '_' || r == '-' || r == ' ' || r == '.') {
			return errors.New("folder name can only contain letters, numbers, underscores, hyphens, spaces, and periods")
		}
	}

	// Validate description (optional)
	if folder.Description != nil {
		*folder.Description = strings.TrimSpace(*folder.Description)
		if utf8.RuneCountInString(*folder.Description) > MaxFolderDescriptionLength {
			return errors.New("folder description must be less than 500 characters")
		}
		// If description is empty after trimming, set it to nil
		if *folder.Description == "" {
			folder.Description = nil
		}
	}

	// Validate user_id (should be positive) - this is set from auth context
	if folder.UserID <= 0 {
		return errors.New("valid user ID is required")
	}

	// Validate parent_id (optional but must be positive if provided)
	if folder.ParentID != nil && *folder.ParentID <= 0 {
		return errors.New("parent folder ID must be positive if provided")
	}

	// Prevent self-reference (though this should be caught by circular reference check)
	if folder.ParentID != nil && folder.ID != 0 && *folder.ParentID == folder.ID {
		return errors.New("folder cannot be its own parent")
	}

	return nil
}

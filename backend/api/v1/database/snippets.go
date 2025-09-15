package database

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/GHutch55/fragments/backend/api/v1/models"
)

var ErrNoSnippetError = errors.New("snippet does not exist")

func CreateSnippet(db *sql.DB, snippet *models.Snippet) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	query := `
	INSERT INTO snippets(user_id, folder_id, title, description, content, language, is_favorite, created_at, updated_at)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	RETURNING id`

	var description interface{}
	if snippet.Description != nil {
		description = *snippet.Description
	} else {
		description = nil
	}

	var folderID interface{}
	if snippet.FolderID != nil {
		folderID = *snippet.FolderID
	} else {
		folderID = nil
	}

	now := time.Now()

	var generatedID int64
	err = tx.QueryRow(
		query,
		snippet.UserID,
		folderID,
		snippet.Title,
		description,
		snippet.Content,
		snippet.Language,
		snippet.IsFavorite,
		now,
		now,
	).Scan(&generatedID)
	if err != nil {
		return fmt.Errorf("failed to insert snippet: %w", err)
	}

	// Handle tags if provided
	if snippet.Tags != nil && len(*snippet.Tags) > 0 {
		err = insertSnippetTags(tx, generatedID, snippet.UserID, *snippet.Tags)
		if err != nil {
			return fmt.Errorf("failed to insert snippet tags: %w", err)
		}
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	snippet.ID = generatedID
	snippet.CreatedAt = now
	snippet.UpdatedAt = now

	return nil
}

func GetSnippet(db *sql.DB, snippetID int64) (*models.Snippet, error) {
	query := `
		SELECT id, user_id, folder_id, title, description, content, language, 
		       is_favorite, created_at, updated_at
		FROM snippets 
		WHERE id = ?`

	var snippet models.Snippet
	var description sql.NullString
	var folderID sql.NullInt64

	err := db.QueryRow(query, snippetID).Scan(
		&snippet.ID,
		&snippet.UserID,
		&folderID,
		&snippet.Title,
		&description,
		&snippet.Content,
		&snippet.Language,
		&snippet.IsFavorite,
		&snippet.CreatedAt,
		&snippet.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNoSnippetError
		}
		return nil, fmt.Errorf("failed to get snippet: %w", err)
	}

	// Handle nullable fields
	if description.Valid {
		snippet.Description = &description.String
	}
	if folderID.Valid {
		snippet.FolderID = &folderID.Int64
	}

	// Get tags for this snippet
	tags, err := getSnippetTags(db, snippetID)
	if err != nil {
		return nil, fmt.Errorf("failed to get snippet tags: %w", err)
	}

	// Only set tags if we have any
	if len(tags) > 0 {
		snippet.Tags = &tags
	}

	return &snippet, nil
}

func GetSnippets(db *sql.DB, page, limit int, userID int64, search string) ([]models.Snippet, int, error) {
	offset := (page - 1) * limit

	// Build WHERE clause and arguments
	whereClause := "WHERE user_id = ?"
	args := []interface{}{userID}

	if search != "" {
		// Use FTS5 for search
		whereClause = `WHERE s.user_id = ? AND s.id IN (
			SELECT content_id FROM snippets_fts 
			WHERE snippets_fts MATCH ?
		)`
		args = append(args, search)
	}

	// Get total count
	var countQuery string
	if search != "" {
		countQuery = fmt.Sprintf("SELECT COUNT(*) FROM snippets s %s", whereClause)
	} else {
		countQuery = fmt.Sprintf("SELECT COUNT(*) FROM snippets %s", whereClause)
	}

	var total int
	err := db.QueryRow(countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("%w: failed to get snippet count", ErrDatabaseError)
	}

	// Get snippets data
	var dataQuery string
	if search != "" {
		dataQuery = fmt.Sprintf(`
			SELECT s.id, s.user_id, s.folder_id, s.title, s.description, s.content, s.language, s.is_favorite, s.created_at, s.updated_at
			FROM snippets s %s 
			ORDER BY s.created_at DESC 
			LIMIT ? OFFSET ?`, whereClause)
	} else {
		dataQuery = fmt.Sprintf(`
			SELECT id, user_id, folder_id, title, description, content, language, is_favorite, created_at, updated_at
			FROM snippets %s 
			ORDER BY created_at DESC 
			LIMIT ? OFFSET ?`, whereClause)
	}

	queryArgs := append(args, limit, offset)

	rows, err := db.Query(dataQuery, queryArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("%w: failed to get snippets", ErrDatabaseError)
	}
	defer rows.Close()

	var snippets []models.Snippet
	var snippetIDs []int64

	for rows.Next() {
		var snippet models.Snippet
		var description sql.NullString
		var folderID sql.NullInt64

		err := rows.Scan(
			&snippet.ID,
			&snippet.UserID,
			&folderID,
			&snippet.Title,
			&description,
			&snippet.Content,
			&snippet.Language,
			&snippet.IsFavorite,
			&snippet.CreatedAt,
			&snippet.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("%w: failed to scan snippet data", ErrDatabaseError)
		}

		// Handle nullable fields
		if description.Valid {
			snippet.Description = &description.String
		}
		if folderID.Valid {
			snippet.FolderID = &folderID.Int64
		}

		snippets = append(snippets, snippet)
		snippetIDs = append(snippetIDs, snippet.ID)
	}

	if err = rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("%w: failed to iterate snippets", ErrDatabaseError)
	}

	// Batch fetch tags for all snippets
	if len(snippetIDs) > 0 {
		err = attachTagsToSnippets(db, snippets, snippetIDs)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to attach tags: %w", err)
		}
	}

	return snippets, total, nil
}

func UpdateSnippet(db *sql.DB, snippetID int64, snippet *models.Snippet) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("%w: failed to start transaction", ErrDatabaseError)
	}
	defer tx.Rollback()

	// Check if snippet exists first
	var currentUserID int64
	err = tx.QueryRow("SELECT user_id FROM snippets WHERE id = ?", snippetID).Scan(&currentUserID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("snippet with ID %d does not exist: %w", snippetID, ErrNoSnippetError)
		}
		return fmt.Errorf("%w: failed to check snippet existence", ErrDatabaseError)
	}

	// Prepare values for update
	var descriptionValue interface{}
	if snippet.Description != nil {
		descriptionValue = *snippet.Description
	} else {
		descriptionValue = nil
	}

	var folderIDValue interface{}
	if snippet.FolderID != nil {
		folderIDValue = *snippet.FolderID
	} else {
		folderIDValue = nil
	}

	now := time.Now()

	// Update the snippet
	updateQuery := `
		UPDATE snippets 
		SET folder_id = ?, title = ?, description = ?, content = ?, language = ?, is_favorite = ?, updated_at = ?
		WHERE id = ?`

	result, err := tx.Exec(updateQuery,
		folderIDValue,
		snippet.Title,
		descriptionValue,
		snippet.Content,
		snippet.Language,
		snippet.IsFavorite,
		now,
		snippetID,
	)
	if err != nil {
		return fmt.Errorf("%w: failed to update snippet", ErrDatabaseError)
	}

	// Verify the update actually happened
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("%w: failed to verify update", ErrDatabaseError)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("snippet with ID %d does not exist: %w", snippetID, ErrNoSnippetError)
	}

	// Handle tags update if provided
	if snippet.Tags != nil {
		// Delete existing tag associations
		_, err = tx.Exec("DELETE FROM snippet_tags WHERE snippet_id = ?", snippetID)
		if err != nil {
			return fmt.Errorf("%w: failed to update snippet tags", ErrDatabaseError)
		}

		// Insert new tag associations
		if len(*snippet.Tags) > 0 {
			err = insertSnippetTags(tx, snippetID, currentUserID, *snippet.Tags)
			if err != nil {
				return fmt.Errorf("failed to update snippet tags: %w", err)
			}
		}
	}

	// Commit the transaction
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("%w: failed to commit update", ErrDatabaseError)
	}

	// Update the snippet object with the new data
	snippet.ID = snippetID
	snippet.UserID = currentUserID
	snippet.UpdatedAt = now

	// Get the updated tags if they were modified
	if snippet.Tags != nil {
		tags, err := getSnippetTags(db, snippetID)
		if err != nil {
			return fmt.Errorf("failed to retrieve updated tags: %w", err)
		}
		if len(tags) > 0 {
			snippet.Tags = &tags
		} else {
			emptyTags := []string{}
			snippet.Tags = &emptyTags
		}
	}

	return nil
}

func DeleteSnippet(db *sql.DB, snippetID int64) error {
	deleteQuery := "DELETE FROM snippets WHERE id = ?"
	result, err := db.Exec(deleteQuery, snippetID)
	if err != nil {
		return fmt.Errorf("%w: failed to delete snippet", ErrDatabaseError)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("%w: failed to verify deletion", ErrDatabaseError)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("snippet with ID %d does not exist: %w", snippetID, ErrNoSnippetError)
	}

	return nil
}

// Helper function to get tags for a single snippet
func getSnippetTags(db *sql.DB, snippetID int64) ([]string, error) {
	tagQuery := `
		SELECT t.name 
		FROM snippet_tags st
		JOIN tags t ON st.tag_id = t.id  
		WHERE st.snippet_id = ?
		ORDER BY t.name`

	rows, err := db.Query(tagQuery, snippetID)
	if err != nil {
		return nil, fmt.Errorf("failed to query snippet tags: %w", err)
	}
	defer rows.Close()

	var tags []string
	for rows.Next() {
		var tagName string
		if err := rows.Scan(&tagName); err != nil {
			return nil, fmt.Errorf("failed to scan tag: %w", err)
		}
		tags = append(tags, tagName)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate tags: %w", err)
	}

	return tags, nil
}

func attachTagsToSnippets(db *sql.DB, snippets []models.Snippet, snippetIDs []int64) error {
	if len(snippetIDs) == 0 {
		return nil
	}

	placeholders := make([]string, len(snippetIDs))
	args := make([]interface{}, len(snippetIDs))
	for i, id := range snippetIDs {
		placeholders[i] = "?"
		args[i] = id
	}

	tagQuery := fmt.Sprintf(`
		SELECT st.snippet_id, t.name 
		FROM snippet_tags st
		JOIN tags t ON st.tag_id = t.id  
		WHERE st.snippet_id IN (%s)
		ORDER BY st.snippet_id, t.name`, strings.Join(placeholders, ","))

	rows, err := db.Query(tagQuery, args...)
	if err != nil {
		return fmt.Errorf("failed to get snippet tags: %w", err)
	}
	defer rows.Close()

	// Group tags by snippet ID
	tagMap := make(map[int64][]string)
	for rows.Next() {
		var snippetID int64
		var tagName string
		if err := rows.Scan(&snippetID, &tagName); err != nil {
			return fmt.Errorf("failed to scan tag: %w", err)
		}
		tagMap[snippetID] = append(tagMap[snippetID], tagName)
	}

	if err = rows.Err(); err != nil {
		return fmt.Errorf("failed to iterate tags: %w", err)
	}

	// Attach tags to snippets
	for i := range snippets {
		if tags, exists := tagMap[snippets[i].ID]; exists {
			snippets[i].Tags = &tags
		}
	}

	return nil
}

func insertSnippetTags(tx *sql.Tx, snippetID int64, userID int64, tagNames []string) error {
	for _, tagName := range tagNames {
		// Clean up tag name
		tagName = strings.TrimSpace(tagName)
		if tagName == "" {
			continue
		}

		// First, ensure the tag exists for this user (insert if not exists)
		var tagID int64

		// Try to get existing tag first
		err := tx.QueryRow("SELECT id FROM tags WHERE user_id = ? AND name = ?", userID, tagName).Scan(&tagID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				// Tag doesn't exist, create it
				err = tx.QueryRow(`
					INSERT INTO tags (user_id, name, created_at) 
					VALUES (?, ?, ?)
					RETURNING id`,
					userID, tagName, time.Now(),
				).Scan(&tagID)
				if err != nil {
					return fmt.Errorf("failed to create tag %s: %w", tagName, err)
				}
			} else {
				return fmt.Errorf("failed to get tag %s: %w", tagName, err)
			}
		}

		// Then link the tag to the snippet (ignore duplicates)
		_, err = tx.Exec(`
			INSERT OR IGNORE INTO snippet_tags (snippet_id, tag_id) 
			VALUES (?, ?)`,
			snippetID, tagID,
		)
		if err != nil {
			return fmt.Errorf("failed to link tag %s to snippet: %w", tagName, err)
		}
	}

	return nil
}

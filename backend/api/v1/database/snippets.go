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
		folderID, // Add this parameter
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

	if snippet.Tags != nil && len(*snippet.Tags) > 0 {
		err = insertSnippetTags(tx, generatedID, snippet.UserID, *snippet.Tags)
		if err != nil {
			return fmt.Errorf("failed to insert snippet: %w", err)
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
		&snippet.ID,         // 1. id
		&snippet.UserID,     // 2. user_id
		&folderID,           // 3. folder_id
		&snippet.Title,      // 4. title
		&description,        // 5. description
		&snippet.Content,    // 6. content
		&snippet.Language,   // 7. language
		&snippet.IsFavorite, // 8. is_favorite
		&snippet.CreatedAt,  // 9. created_at
		&snippet.UpdatedAt,  // 10. updated_at
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
	tagQuery := `
		SELECT t.name 
		FROM snippet_tags st
		JOIN tags t ON st.tag_id = t.id  
		WHERE st.snippet_id = ?
		ORDER BY t.name`

	rows, err := db.Query(tagQuery, snippetID)
	if err != nil {
		return nil, fmt.Errorf("failed to get snippet tags: %w", err)
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

	// Attach tags to snippet (only if we have tags)
	if len(tags) > 0 {
		snippet.Tags = &tags
	}

	return &snippet, nil
}

func GetSnippetsWithOptionalUser(db *sql.DB, page, limit int, userID *int64, search string) ([]models.Snippet, int, error) {
	offset := (page - 1) * limit

	// Build WHERE clause - if userID is nil, get all snippets
	var whereClause string
	var args []interface{}

	if userID != nil {
		whereClause = "WHERE user_id = ?"
		args = append(args, *userID)
	}

	// Add search condition if provided
	if search != "" {
		if userID != nil {
			// Search with user filter
			whereClause = `WHERE s.user_id = ? AND s.id IN (
				SELECT content_id FROM snippets_fts 
				WHERE snippets_fts MATCH ?
			)`
			args = append(args, search)
		} else {
			// Search all snippets
			whereClause = `WHERE s.id IN (
				SELECT content_id FROM snippets_fts 
				WHERE snippets_fts MATCH ?
			)`
			args = append(args, search)
		}
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
		fmt.Printf("Database error getting snippet count: %v\n", err)
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
		fmt.Printf("Database error getting snippets: %v\n", err)
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
			fmt.Printf("Database error scanning snippet row: %v\n", err)
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
		fmt.Printf("Database error iterating snippets: %v\n", err)
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

func GetSnippets(db *sql.DB, page, limit int, userID int64, search string) ([]models.Snippet, int, error) {
	offset := (page - 1) * limit

	// Build WHERE clause
	whereClause := "WHERE user_id = ?"
	args := []interface{}{userID}

	if search != "" {
		// Use FTS5 for search - join with snippets_fts table
		whereClause = `WHERE s.user_id = ? AND s.id IN (
			SELECT content_id FROM snippets_fts 
			WHERE snippets_fts MATCH ?
		)`
		args = append(args, search)
	}

	// Get total count
	var total int
	var countQuery string
	if search != "" {
		countQuery = fmt.Sprintf("SELECT COUNT(*) FROM snippets s %s", whereClause)
	} else {
		countQuery = fmt.Sprintf("SELECT COUNT(*) FROM snippets %s", whereClause)
	}

	err := db.QueryRow(countQuery, args...).Scan(&total)
	if err != nil {
		fmt.Printf("Database error getting snippet count: %v\n", err)
		return nil, 0, fmt.Errorf("%w: failed to get snippet count", ErrDatabaseError)
	}

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
		fmt.Printf("Database error getting snippets: %v\n", err)
		return nil, 0, fmt.Errorf("%w: failed to get snippets", ErrDatabaseError)
	}
	defer rows.Close()

	var snippets []models.Snippet
	var snippetIDs []int64

	for rows.Next() {
		var snippet models.Snippet
		var description sql.NullString
		var folderID sql.NullInt64 // Add this for folder_id

		err := rows.Scan(
			&snippet.ID,
			&snippet.UserID,
			&folderID, // Add this scan parameter
			&snippet.Title,
			&description,
			&snippet.Content,
			&snippet.Language,
			&snippet.IsFavorite,
			&snippet.CreatedAt,
			&snippet.UpdatedAt,
		)
		if err != nil {
			fmt.Printf("Database error scanning snippet row: %v\n", err)
			return nil, 0, fmt.Errorf("%w: failed to scan snippet data", ErrDatabaseError)
		}

		// Handle nullable description
		if description.Valid {
			snippet.Description = &description.String
		}

		// Handle nullable folder_id
		if folderID.Valid {
			snippet.FolderID = &folderID.Int64
		}

		snippets = append(snippets, snippet)
		snippetIDs = append(snippetIDs, snippet.ID)
	}

	if err = rows.Err(); err != nil {
		fmt.Printf("Database error iterating snippets: %v\n", err)
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

func DeleteSnippet(db *sql.DB, snippetID int64) error {
	deleteQuery := "DELETE FROM snippets WHERE id = ?"
	result, err := db.Exec(deleteQuery, snippetID)
	if err != nil {
		fmt.Printf("Database error deleting snippet ID %d: %v\n", snippetID, err)
		return fmt.Errorf("%w: failed to delete snippet", ErrDatabaseError)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		fmt.Printf("Error checking rows affected for delete: %v\n", err)
		return fmt.Errorf("%w: failed to verify deletion", ErrDatabaseError)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("user with ID %d does not exist: %w", snippetID, ErrNoSnippetError)
	}

	return nil
}

func UpdateSnippet(db *sql.DB, snippetID int64, snippet *models.Snippet) error {
	tx, err := db.Begin()
	if err != nil {
		fmt.Printf("Error starting transaction: %v\n", err)
		return fmt.Errorf("%w: failed to start transaction", ErrDatabaseError)
	}
	defer tx.Rollback()

	// Check if snippet exists and get current data
	selectQuery := `
		SELECT id, user_id, folder_id, title, description, content, language, is_favorite, created_at, updated_at
		FROM snippets WHERE id = ?`

	var currentSnippet models.Snippet
	var description sql.NullString
	var folderID sql.NullInt64
	err = tx.QueryRow(selectQuery, snippetID).Scan(
		&currentSnippet.ID,
		&currentSnippet.UserID,
		&folderID,
		&currentSnippet.Title,
		&description,
		&currentSnippet.Content,
		&currentSnippet.Language,
		&currentSnippet.IsFavorite,
		&currentSnippet.CreatedAt,
		&currentSnippet.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("snippet with ID %d does not exist: %w", snippetID, ErrNoSnippetError)
		}
		fmt.Printf("Database error retrieving snippet for update: %v\n", err)
		return fmt.Errorf("%w: failed to retrieve snippet for update", ErrDatabaseError)
	}

	// Convert nullable fields for current snippet
	if description.Valid {
		currentSnippet.Description = &description.String
	}
	if folderID.Valid {
		currentSnippet.FolderID = &folderID.Int64
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

	_, err = tx.Exec(updateQuery,
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
		fmt.Printf("Database error updating snippet: %v\n", err)
		return fmt.Errorf("%w: failed to update snippet", ErrDatabaseError)
	}

	// Handle tags update if provided
	if snippet.Tags != nil {
		// Delete existing tag associations
		_, err = tx.Exec("DELETE FROM snippet_tags WHERE snippet_id = ?", snippetID)
		if err != nil {
			fmt.Printf("Database error removing old tags: %v\n", err)
			return fmt.Errorf("%w: failed to update snippet tags", ErrDatabaseError)
		}

		// Insert new tag associations
		if len(*snippet.Tags) > 0 {
			err = insertSnippetTags(tx, snippetID, currentSnippet.UserID, *snippet.Tags)
			if err != nil {
				return fmt.Errorf("failed to update snippet tags: %w", err)
			}
		}
	}

	// Get the updated snippet data
	err = tx.QueryRow(selectQuery, snippetID).Scan(
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
		fmt.Printf("Error retrieving updated snippet: %v\n", err)
		return fmt.Errorf("%w: failed to retrieve updated snippet", ErrDatabaseError)
	}

	// Convert nullable fields for response
	if description.Valid {
		snippet.Description = &description.String
	} else {
		snippet.Description = nil
	}
	if folderID.Valid {
		snippet.FolderID = &folderID.Int64
	} else {
		snippet.FolderID = nil
	}

	// If tags were updated, get the new tags
	if snippet.Tags != nil {
		tagQuery := `
			SELECT t.name 
			FROM snippet_tags st
			JOIN tags t ON st.tag_id = t.id  
			WHERE st.snippet_id = ?
			ORDER BY t.name`

		rows, err := tx.Query(tagQuery, snippetID)
		if err != nil {
			fmt.Printf("Error retrieving updated tags: %v\n", err)
			return fmt.Errorf("%w: failed to retrieve updated tags", ErrDatabaseError)
		}
		defer rows.Close()

		var tags []string
		for rows.Next() {
			var tagName string
			if err := rows.Scan(&tagName); err != nil {
				return fmt.Errorf("failed to scan updated tag: %w", err)
			}
			tags = append(tags, tagName)
		}

		if err = rows.Err(); err != nil {
			return fmt.Errorf("failed to iterate updated tags: %w", err)
		}

		// Update the snippet with new tags (only if we have tags)
		if len(tags) > 0 {
			snippet.Tags = &tags
		} else {
			emptyTags := []string{}
			snippet.Tags = &emptyTags
		}
	}

	// Commit the transaction
	if err = tx.Commit(); err != nil {
		fmt.Printf("Error committing transaction: %v\n", err)
		return fmt.Errorf("%w: failed to commit update", ErrDatabaseError)
	}

	return nil
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
		// First, ensure the tag exists for this user (insert if not exists)
		var tagID int64
		err := tx.QueryRow(`
			INSERT INTO tags (user_id, name, created_at) 
			VALUES (?, ?, ?) 
			ON CONFLICT(user_id, name) DO UPDATE SET user_id = user_id
			RETURNING id`,
			userID, tagName, time.Now(),
		).Scan(&tagID)
		if err != nil {
			return fmt.Errorf("failed to insert/get tag %s: %w", tagName, err)
		}

		// Then link the tag to the snippet
		_, err = tx.Exec(`
			INSERT INTO snippet_tags (snippet_id, tag_id) 
			VALUES (?, ?)`,
			snippetID, tagID,
		)
		if err != nil {
			return fmt.Errorf("failed to link tag %s to snippet: %w", tagName, err)
		}
	}

	return nil
}

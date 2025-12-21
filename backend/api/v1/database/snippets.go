package database

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/GHutch55/fragments/backend/api/v1/models"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNoSnippetError = errors.New("snippet does not exist")

func CreateSnippet(ctx context.Context, pool *pgxpool.Pool, snippet *models.Snippet) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	query := `
	INSERT INTO snippets(user_id, folder_id, title, description, content, language, is_favorite, created_at, updated_at)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
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
	err = tx.QueryRow(ctx,
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
		err = insertSnippetTags(ctx, tx, generatedID, snippet.UserID, *snippet.Tags)
		if err != nil {
			return fmt.Errorf("failed to insert snippet tags: %w", err)
		}
	}

	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	snippet.ID = generatedID
	snippet.CreatedAt = now
	snippet.UpdatedAt = now

	return nil
}

func GetSnippet(ctx context.Context, pool *pgxpool.Pool, snippetID int64) (*models.Snippet, error) {
	query := `
		SELECT id, user_id, folder_id, title, description, content, language, 
		       is_favorite, created_at, updated_at
		FROM snippets 
		WHERE id = $1`

	var snippet models.Snippet
	var description *string
	var folderID *int64

	err := pool.QueryRow(ctx, query, snippetID).Scan(
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
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNoSnippetError
		}
		return nil, fmt.Errorf("failed to get snippet: %w", err)
	}

	snippet.Description = description
	snippet.FolderID = folderID

	tags, err := getSnippetTags(ctx, pool, snippetID)
	if err != nil {
		return nil, fmt.Errorf("failed to get snippet tags: %w", err)
	}

	if len(tags) > 0 {
		snippet.Tags = &tags
	}

	return &snippet, nil
}

func GetSnippets(ctx context.Context, pool *pgxpool.Pool, page, limit int, userID int64, search string) ([]models.Snippet, int, error) {
	offset := (page - 1) * limit

	var whereClause string
	var args []interface{}

	if search != "" {
		whereClause = `
		WHERE s.user_id = $1
		AND s.document_with_weights @@ plainto_tsquery('english', $2)`
		args = []interface{}{userID, search}
	} else {
		whereClause = `WHERE user_id = $1`
		args = []interface{}{userID}
	}

	var countQuery string
	if search != "" {
		countQuery = fmt.Sprintf("SELECT COUNT(*) FROM snippets s %s", whereClause)
	} else {
		countQuery = fmt.Sprintf("SELECT COUNT(*) FROM snippets %s", whereClause)
	}

	var total int
	err := pool.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get snippet count: %w", err)
	}

	var dataQuery string
	argPosition := len(args) + 1
	if search != "" {
		dataQuery = fmt.Sprintf(`
			SELECT s.id, s.user_id, s.folder_id, s.title, s.description, s.content, s.language, s.is_favorite, s.created_at, s.updated_at
			FROM snippets s 
			%s 
			ORDER BY ts_rank(s.document_with_weights, plainto_tsquery('english', $2)) DESC, s.created_at DESC
			LIMIT $%d OFFSET $%d`, whereClause, argPosition, argPosition+1)
	} else {
		dataQuery = fmt.Sprintf(`
			SELECT id, user_id, folder_id, title, description, content, language, is_favorite, created_at, updated_at
			FROM snippets 
			%s 
			ORDER BY created_at DESC
			LIMIT $%d OFFSET $%d`, whereClause, argPosition, argPosition+1)
	}
	args = append(args, limit, offset)

	rows, err := pool.Query(ctx, dataQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get snippets: %w", err)
	}
	defer rows.Close()

	var snippets []models.Snippet
	var snippetIDs []int64

	for rows.Next() {
		var snippet models.Snippet
		var description *string
		var folderID *int64

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
			return nil, 0, fmt.Errorf("failed to scan snippet data: %w", err)
		}

		snippet.Description = description
		snippet.FolderID = folderID

		snippets = append(snippets, snippet)
		snippetIDs = append(snippetIDs, snippet.ID)
	}

	if err = rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("failed to iterate snippets: %w", err)
	}

	if len(snippetIDs) > 0 {
		err = attachTagsToSnippets(ctx, pool, snippets, snippetIDs)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to attach tags: %w", err)
		}
	}

	return snippets, total, nil
}

func UpdateSnippet(ctx context.Context, pool *pgxpool.Pool, snippetID int64, snippet *models.Snippet) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	var currentUserID int64
	err = tx.QueryRow(ctx, "SELECT user_id FROM snippets WHERE id = $1", snippetID).Scan(&currentUserID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("snippet with ID %d does not exist: %w", snippetID, ErrNoSnippetError)
		}
		return fmt.Errorf("failed to check snippet existence: %w", err)
	}

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

	updateQuery := `
		UPDATE snippets 
		SET folder_id = $1, title = $2, description = $3, content = $4, language = $5, is_favorite = $6, updated_at = $7
		WHERE id = $8`

	result, err := tx.Exec(ctx, updateQuery,
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
		return fmt.Errorf("failed to update snippet: %w", err)
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("snippet with ID %d does not exist: %w", snippetID, ErrNoSnippetError)
	}

	if snippet.Tags != nil {
		_, err = tx.Exec(ctx, "DELETE FROM snippet_tags WHERE snippet_id = $1", snippetID)
		if err != nil {
			return fmt.Errorf("failed to update snippet tags: %w", err)
		}

		if len(*snippet.Tags) > 0 {
			err = insertSnippetTags(ctx, tx, snippetID, currentUserID, *snippet.Tags)
			if err != nil {
				return fmt.Errorf("failed to update snippet tags: %w", err)
			}
		}
	}

	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit update: %w", err)
	}

	snippet.ID = snippetID
	snippet.UserID = currentUserID
	snippet.UpdatedAt = now

	if snippet.Tags != nil {
		tags, err := getSnippetTags(ctx, pool, snippetID)
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

func DeleteSnippet(ctx context.Context, pool *pgxpool.Pool, snippetID int64) error {
	deleteQuery := "DELETE FROM snippets WHERE id = $1"
	result, err := pool.Exec(ctx, deleteQuery, snippetID)
	if err != nil {
		return fmt.Errorf("failed to delete snippet: %w", err)
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("snippet with ID %d does not exist: %w", snippetID, ErrNoSnippetError)
	}

	return nil
}

// Helper function to get tags for a single snippet
func getSnippetTags(ctx context.Context, pool *pgxpool.Pool, snippetID int64) ([]string, error) {
	tagQuery := `
		SELECT t.name 
		FROM snippet_tags st
		JOIN tags t ON st.tag_id = t.id  
		WHERE st.snippet_id = $1
		ORDER BY t.name`

	rows, err := pool.Query(ctx, tagQuery, snippetID)
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

func attachTagsToSnippets(ctx context.Context, pool *pgxpool.Pool, snippets []models.Snippet, snippetIDs []int64) error {
	if len(snippetIDs) == 0 {
		return nil
	}

	placeholders := make([]string, len(snippetIDs))
	args := make([]interface{}, len(snippetIDs))
	for i, id := range snippetIDs {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = id
	}

	tagQuery := fmt.Sprintf(`
		SELECT st.snippet_id, t.name 
		FROM snippet_tags st
		JOIN tags t ON st.tag_id = t.id  
		WHERE st.snippet_id IN (%s)
		ORDER BY st.snippet_id, t.name`, strings.Join(placeholders, ","))

	rows, err := pool.Query(ctx, tagQuery, args...)
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

func insertSnippetTags(ctx context.Context, tx pgx.Tx, snippetID int64, userID int64, tagNames []string) error {
	for _, tagName := range tagNames {
		tagName = strings.TrimSpace(tagName)
		if tagName == "" {
			continue
		}

		var tagID int64

		err := tx.QueryRow(ctx, "SELECT id FROM tags WHERE user_id = $1 AND name = $2", userID, tagName).Scan(&tagID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				err = tx.QueryRow(ctx, `
					INSERT INTO tags (user_id, name, created_at) 
					VALUES ($1, $2, $3)
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

		_, err = tx.Exec(ctx, `
			INSERT INTO snippet_tags (snippet_id, tag_id) 
			VALUES ($1, $2)
			ON CONFLICT DO NOTHING`,
			snippetID, tagID,
		)
		if err != nil {
			return fmt.Errorf("failed to link tag %s to snippet: %w", tagName, err)
		}
	}

	return nil
}

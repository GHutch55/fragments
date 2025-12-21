package database

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/GHutch55/fragments/backend/api/v1/models"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNoFolderError     = errors.New("folder does not exist")
	ErrFolderHasChildren = errors.New("folder has child folders")
	ErrCircularReference = errors.New("circular folder reference not allowed")
)

func CreateFolder(ctx context.Context, pool *pgxpool.Pool, folder *models.Folder) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	query := `
    INSERT INTO folders(user_id, name, description, parent_id, created_at, updated_at)
    VALUES ($1, $2, $3, $4, $5, $6)
    RETURNING id`

	var description interface{}
	if folder.Description != nil {
		description = *folder.Description
	} else {
		description = nil
	}

	var parentID interface{}
	if folder.ParentID != nil {
		parentID = *folder.ParentID
	} else {
		parentID = nil
	}

	if folder.ParentID != nil {
		var parentUserID int64
		err = tx.QueryRow(ctx, "SELECT user_id FROM folders WHERE id = $1", *folder.ParentID).Scan(&parentUserID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return fmt.Errorf("parent folder does not exist")
			}
			return fmt.Errorf("failed to validate parent folder: %w", err)
		}

		if parentUserID != folder.UserID {
			return fmt.Errorf("parent folder does not belong to user")
		}

		if err := checkCircularReference(ctx, tx, folder.UserID, *folder.ParentID, 0); err != nil {
			return fmt.Errorf("circular reference detected: %w", err)
		}
	}

	var count int
	var nameCheckQuery string
	var nameCheckArgs []interface{}

	if folder.ParentID != nil {
		nameCheckQuery = "SELECT COUNT(*) FROM folders WHERE user_id = $1 AND name = $2 AND parent_id = $3"
		nameCheckArgs = []interface{}{folder.UserID, folder.Name, *folder.ParentID}
	} else {
		nameCheckQuery = "SELECT COUNT(*) FROM folders WHERE user_id = $1 AND name = $2 AND parent_id IS NULL"
		nameCheckArgs = []interface{}{folder.UserID, folder.Name}
	}

	err = tx.QueryRow(ctx, nameCheckQuery, nameCheckArgs...).Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to check for duplicate folder name: %w", err)
	}

	if count > 0 {
		return fmt.Errorf("folder name already exists in this location")
	}

	now := time.Now()

	var generatedID int64
	err = tx.QueryRow(ctx,
		query,
		folder.UserID,
		folder.Name,
		description,
		parentID,
		now,
		now,
	).Scan(&generatedID)
	if err != nil {
		return fmt.Errorf("failed to insert folder: %w", err)
	}

	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	folder.ID = generatedID
	folder.CreatedAt = now
	folder.UpdatedAt = now

	return nil
}

func GetFolder(ctx context.Context, pool *pgxpool.Pool, folderID int64) (*models.Folder, error) {
	query := `
		SELECT id, user_id, name, description, parent_id, created_at, updated_at
		FROM folders 
		WHERE id = $1`

	var folder models.Folder
	var description *string
	var parentID *int64

	err := pool.QueryRow(ctx, query, folderID).Scan(
		&folder.ID,
		&folder.UserID,
		&folder.Name,
		&description,
		&parentID,
		&folder.CreatedAt,
		&folder.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNoFolderError
		}
		return nil, fmt.Errorf("failed to get folder: %w", err)
	}

	folder.Description = description
	folder.ParentID = parentID

	return &folder, nil
}

func GetFolders(ctx context.Context, pool *pgxpool.Pool, page, limit int, userID int64, parentID *int64) ([]models.Folder, int, error) {
	offset := (page - 1) * limit

	whereClause := "WHERE user_id = $1"
	args := []interface{}{userID}

	argPosition := 2
	if parentID != nil {
		whereClause += fmt.Sprintf(" AND parent_id = $%d", argPosition)
		args = append(args, *parentID)
		argPosition++
	} else {
		whereClause += " AND parent_id IS NULL"
	}

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM folders %s", whereClause)
	var total int
	err := pool.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("%w: failed to get folder count", ErrDatabaseError)
	}

	dataQuery := fmt.Sprintf(`
		SELECT id, user_id, name, description, parent_id, created_at, updated_at
		FROM folders %s 
		ORDER BY name ASC 
		LIMIT $%d OFFSET $%d`, whereClause, argPosition, argPosition+1)

	queryArgs := append(args, limit, offset)
	rows, err := pool.Query(ctx, dataQuery, queryArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("%w: failed to get folders", ErrDatabaseError)
	}
	defer rows.Close()

	var folders []models.Folder
	for rows.Next() {
		var folder models.Folder
		var description *string
		var parentIDVal *int64

		err := rows.Scan(
			&folder.ID,
			&folder.UserID,
			&folder.Name,
			&description,
			&parentIDVal,
			&folder.CreatedAt,
			&folder.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("%w: failed to scan folder data", ErrDatabaseError)
		}

		folder.Description = description
		folder.ParentID = parentIDVal

		folders = append(folders, folder)
	}

	if err = rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("%w: failed to iterate folders", ErrDatabaseError)
	}

	return folders, total, nil
}

func UpdateFolder(ctx context.Context, pool *pgxpool.Pool, folderID int64, folder *models.Folder) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("%w: failed to start transaction", ErrDatabaseError)
	}
	defer tx.Rollback(ctx)

	var currentUserID int64
	var currentParentID *int64
	err = tx.QueryRow(ctx, "SELECT user_id, parent_id FROM folders WHERE id = $1", folderID).Scan(&currentUserID, &currentParentID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("folder with ID %d does not exist: %w", folderID, ErrNoFolderError)
		}
		return fmt.Errorf("%w: failed to check folder existence", ErrDatabaseError)
	}

	if folder.ParentID != nil {
		var parentUserID int64
		err = tx.QueryRow(ctx, "SELECT user_id FROM folders WHERE id = $1", *folder.ParentID).Scan(&parentUserID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return fmt.Errorf("parent folder does not exist")
			}
			return fmt.Errorf("failed to validate parent folder: %w", err)
		}

		if parentUserID != folder.UserID {
			return fmt.Errorf("parent folder does not belong to user")
		}

		if *folder.ParentID == folderID {
			return fmt.Errorf("folder cannot be its own parent")
		}

		needsCircularCheck := true
		if currentParentID != nil && *currentParentID == *folder.ParentID {
			needsCircularCheck = false // Parent isn't changing
		}

		if needsCircularCheck {
			if err := checkCircularReferenceForUpdate(ctx, tx, folder.UserID, folderID, *folder.ParentID, 0); err != nil {
				return fmt.Errorf("circular reference detected: %w", err)
			}
		}
	}

	var count int
	var nameCheckQuery string
	var nameCheckArgs []interface{}

	if folder.ParentID != nil {
		nameCheckQuery = "SELECT COUNT(*) FROM folders WHERE user_id = $1 AND name = $2 AND parent_id = $3 AND id != $4"
		nameCheckArgs = []interface{}{folder.UserID, folder.Name, *folder.ParentID, folderID}
	} else {
		nameCheckQuery = "SELECT COUNT(*) FROM folders WHERE user_id = $1 AND name = $2 AND parent_id IS NULL AND id != $3"
		nameCheckArgs = []interface{}{folder.UserID, folder.Name, folderID}
	}

	err = tx.QueryRow(ctx, nameCheckQuery, nameCheckArgs...).Scan(&count)
	if err != nil {
		return fmt.Errorf("%w: failed to check for duplicate folder name", ErrDatabaseError)
	}

	if count > 0 {
		return fmt.Errorf("folder name already exists in this location")
	}

	var descriptionValue interface{}
	if folder.Description != nil {
		descriptionValue = *folder.Description
	} else {
		descriptionValue = nil
	}

	var parentIDValue interface{}
	if folder.ParentID != nil {
		parentIDValue = *folder.ParentID
	} else {
		parentIDValue = nil
	}

	now := time.Now()

	updateQuery := `
		UPDATE folders 
		SET name = $1, description = $2, parent_id = $3, updated_at = $4
		WHERE id = $5`

	result, err := tx.Exec(ctx, updateQuery,
		folder.Name,
		descriptionValue,
		parentIDValue,
		now,
		folderID,
	)
	if err != nil {
		return fmt.Errorf("%w: failed to update folder", ErrDatabaseError)
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("folder with ID %d does not exist: %w", folderID, ErrNoFolderError)
	}

	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("%w: failed to commit update", ErrDatabaseError)
	}

	folder.ID = folderID
	folder.UserID = currentUserID
	folder.UpdatedAt = now

	return nil
}

func DeleteFolder(ctx context.Context, pool *pgxpool.Pool, folderID int64) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("%w: failed to start transaction", ErrDatabaseError)
	}
	defer tx.Rollback(ctx)

	var exists bool
	err = tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM folders WHERE id = $1)", folderID).Scan(&exists)
	if err != nil {
		return fmt.Errorf("%w: failed to check folder existence", ErrDatabaseError)
	}

	if !exists {
		return fmt.Errorf("folder with ID %d does not exist: %w", folderID, ErrNoFolderError)
	}

	var childCount int
	err = tx.QueryRow(ctx, "SELECT COUNT(*) FROM folders WHERE parent_id = $1", folderID).Scan(&childCount)
	if err != nil {
		return fmt.Errorf("%w: failed to check for child folders", ErrDatabaseError)
	}

	if childCount > 0 {
		return fmt.Errorf("folder has %d child folders: %w", childCount, ErrFolderHasChildren)
	}

	var snippetCount int
	err = tx.QueryRow(ctx, "SELECT COUNT(*) FROM snippets WHERE folder_id = $1", folderID).Scan(&snippetCount)
	if err != nil {
		return fmt.Errorf("%w: failed to check for snippets in folder", ErrDatabaseError)
	}

	// Move snippets to root before deleting folder
	if snippetCount > 0 {
		_, err = tx.Exec(ctx, "UPDATE snippets SET folder_id = NULL, updated_at = CURRENT_TIMESTAMP WHERE folder_id = $1", folderID)
		if err != nil {
			return fmt.Errorf("%w: failed to move snippets to root", ErrDatabaseError)
		}
	}

	result, err := tx.Exec(ctx, "DELETE FROM folders WHERE id = $1", folderID)
	if err != nil {
		return fmt.Errorf("%w: failed to delete folder", ErrDatabaseError)
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("folder with ID %d does not exist: %w", folderID, ErrNoFolderError)
	}

	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("%w: failed to commit deletion", ErrDatabaseError)
	}

	return nil
}

func checkCircularReference(ctx context.Context, tx pgx.Tx, userID int64, parentID int64, depth int) error {
	// Prevent infinite recursion
	if depth > 50 {
		return fmt.Errorf("maximum folder depth exceeded")
	}

	var grandParentID *int64
	err := tx.QueryRow(ctx, "SELECT parent_id FROM folders WHERE id = $1 AND user_id = $2", parentID, userID).Scan(&grandParentID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		return fmt.Errorf("failed to check parent folder: %w", err)
	}

	if grandParentID == nil {
		return nil
	}

	return checkCircularReference(ctx, tx, userID, *grandParentID, depth+1)
}

func checkCircularReferenceForUpdate(ctx context.Context, tx pgx.Tx, userID int64, folderID int64, newParentID int64, depth int) error {
	// Prevent infinite recursion
	if depth > 50 {
		return fmt.Errorf("maximum folder depth exceeded")
	}

	if newParentID == folderID {
		return fmt.Errorf("folder cannot be its own parent")
	}

	var grandParentID *int64
	err := tx.QueryRow(ctx, "SELECT parent_id FROM folders WHERE id = $1 AND user_id = $2", newParentID, userID).Scan(&grandParentID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		return fmt.Errorf("failed to check parent folder: %w", err)
	}

	if grandParentID == nil {
		return nil
	}

	if *grandParentID == folderID {
		return fmt.Errorf("circular reference detected")
	}

	return checkCircularReferenceForUpdate(ctx, tx, userID, folderID, *grandParentID, depth+1)
}

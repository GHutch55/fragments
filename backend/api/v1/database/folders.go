package database

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/GHutch55/fragments/backend/api/v1/models"
)

var (
	ErrNoFolderError     = errors.New("folder does not exist")
	ErrFolderHasChildren = errors.New("folder has child folders")
	ErrCircularReference = errors.New("circular folder reference not allowed")
)

func CreateFolder(db *sql.DB, folder *models.Folder) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	query := `
    INSERT INTO folders(user_id, name, description, parent_id, created_at, updated_at)
    VALUES (?, ?, ?, ?, ?, ?)
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
		err = tx.QueryRow("SELECT user_id FROM folders WHERE id = ?", *folder.ParentID).Scan(&parentUserID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return fmt.Errorf("parent folder does not exist")
			}
			return fmt.Errorf("failed to validate parent folder: %w", err)
		}

		if parentUserID != folder.UserID {
			return fmt.Errorf("parent folder does not belong to user")
		}

		if err := checkCircularReference(tx, folder.UserID, *folder.ParentID, 0); err != nil {
			return fmt.Errorf("circular reference detected: %w", err)
		}
	}

	var count int
	var nameCheckQuery string
	var nameCheckArgs []interface{}

	if folder.ParentID != nil {
		nameCheckQuery = "SELECT COUNT(*) FROM folders WHERE user_id = ? AND name = ? AND parent_id = ?"
		nameCheckArgs = []interface{}{folder.UserID, folder.Name, *folder.ParentID}
	} else {
		nameCheckQuery = "SELECT COUNT(*) FROM folders WHERE user_id = ? AND name = ? AND parent_id IS NULL"
		nameCheckArgs = []interface{}{folder.UserID, folder.Name}
	}

	err = tx.QueryRow(nameCheckQuery, nameCheckArgs...).Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to check for duplicate folder name: %w", err)
	}

	if count > 0 {
		return fmt.Errorf("folder name already exists in this location")
	}

	now := time.Now()

	var generatedID int64
	err = tx.QueryRow(
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

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	folder.ID = generatedID
	folder.CreatedAt = now
	folder.UpdatedAt = now

	return nil
}

func GetFolder(db *sql.DB, folderID int64) (*models.Folder, error) {
	query := `
		SELECT id, user_id, name, description, parent_id, created_at, updated_at
		FROM folders 
		WHERE id = ?`

	var folder models.Folder
	var description sql.NullString
	var parentID sql.NullInt64

	err := db.QueryRow(query, folderID).Scan(
		&folder.ID,
		&folder.UserID,
		&folder.Name,
		&description,
		&parentID,
		&folder.CreatedAt,
		&folder.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNoFolderError
		}
		return nil, fmt.Errorf("failed to get folder: %w", err)
	}

	if description.Valid {
		folder.Description = &description.String
	}
	if parentID.Valid {
		parentIDValue := parentID.Int64
		folder.ParentID = &parentIDValue
	}

	return &folder, nil
}

func GetFolders(db *sql.DB, page, limit int, userID int64, parentID *int64) ([]models.Folder, int, error) {
	offset := (page - 1) * limit

	whereClause := "WHERE user_id = ?"
	args := []interface{}{userID}

	if parentID != nil {
		whereClause += " AND parent_id = ?"
		args = append(args, *parentID)
	} else {
		whereClause += " AND parent_id IS NULL"
	}

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM folders %s", whereClause)
	var total int
	err := db.QueryRow(countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("%w: failed to get folder count", ErrDatabaseError)
	}

	dataQuery := fmt.Sprintf(`
		SELECT id, user_id, name, description, parent_id, created_at, updated_at
		FROM folders %s 
		ORDER BY name ASC 
		LIMIT ? OFFSET ?`, whereClause)

	queryArgs := append(args, limit, offset)
	rows, err := db.Query(dataQuery, queryArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("%w: failed to get folders", ErrDatabaseError)
	}
	defer rows.Close()

	var folders []models.Folder
	for rows.Next() {
		var folder models.Folder
		var description sql.NullString
		var parentIDVal sql.NullInt64

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

		if description.Valid {
			folder.Description = &description.String
		}
		if parentIDVal.Valid {
			folder.ParentID = &parentIDVal.Int64
		}

		folders = append(folders, folder)
	}

	if err = rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("%w: failed to iterate folders", ErrDatabaseError)
	}

	return folders, total, nil
}

func UpdateFolder(db *sql.DB, folderID int64, folder *models.Folder) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("%w: failed to start transaction", ErrDatabaseError)
	}
	defer tx.Rollback()

	var currentUserID int64
	var currentParentID sql.NullInt64
	err = tx.QueryRow("SELECT user_id, parent_id FROM folders WHERE id = ?", folderID).Scan(&currentUserID, &currentParentID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("folder with ID %d does not exist: %w", folderID, ErrNoFolderError)
		}
		return fmt.Errorf("%w: failed to check folder existence", ErrDatabaseError)
	}

	if folder.ParentID != nil {
		var parentUserID int64
		err = tx.QueryRow("SELECT user_id FROM folders WHERE id = ?", *folder.ParentID).Scan(&parentUserID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
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
		if currentParentID.Valid && currentParentID.Int64 == *folder.ParentID {
			needsCircularCheck = false // Parent isn't changing
		}

		if needsCircularCheck {
			if err := checkCircularReferenceForUpdate(tx, folder.UserID, folderID, *folder.ParentID, 0); err != nil {
				return fmt.Errorf("circular reference detected: %w", err)
			}
		}
	}

	var count int
	var nameCheckQuery string
	var nameCheckArgs []interface{}

	if folder.ParentID != nil {
		nameCheckQuery = "SELECT COUNT(*) FROM folders WHERE user_id = ? AND name = ? AND parent_id = ? AND id != ?"
		nameCheckArgs = []interface{}{folder.UserID, folder.Name, *folder.ParentID, folderID}
	} else {
		nameCheckQuery = "SELECT COUNT(*) FROM folders WHERE user_id = ? AND name = ? AND parent_id IS NULL AND id != ?"
		nameCheckArgs = []interface{}{folder.UserID, folder.Name, folderID}
	}

	err = tx.QueryRow(nameCheckQuery, nameCheckArgs...).Scan(&count)
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
		SET name = ?, description = ?, parent_id = ?, updated_at = ?
		WHERE id = ?`

	result, err := tx.Exec(updateQuery,
		folder.Name,
		descriptionValue,
		parentIDValue,
		now,
		folderID,
	)
	if err != nil {
		return fmt.Errorf("%w: failed to update folder", ErrDatabaseError)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("%w: failed to verify update", ErrDatabaseError)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("folder with ID %d does not exist: %w", folderID, ErrNoFolderError)
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("%w: failed to commit update", ErrDatabaseError)
	}

	folder.ID = folderID
	folder.UserID = currentUserID
	folder.UpdatedAt = now

	return nil
}

func DeleteFolder(db *sql.DB, folderID int64) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("%w: failed to start transaction", ErrDatabaseError)
	}
	defer tx.Rollback()

	var exists bool
	err = tx.QueryRow("SELECT EXISTS(SELECT 1 FROM folders WHERE id = ?)", folderID).Scan(&exists)
	if err != nil {
		return fmt.Errorf("%w: failed to check folder existence", ErrDatabaseError)
	}

	if !exists {
		return fmt.Errorf("folder with ID %d does not exist: %w", folderID, ErrNoFolderError)
	}

	var childCount int
	err = tx.QueryRow("SELECT COUNT(*) FROM folders WHERE parent_id = ?", folderID).Scan(&childCount)
	if err != nil {
		return fmt.Errorf("%w: failed to check for child folders", ErrDatabaseError)
	}

	if childCount > 0 {
		return fmt.Errorf("folder has %d child folders: %w", childCount, ErrFolderHasChildren)
	}

	var snippetCount int
	err = tx.QueryRow("SELECT COUNT(*) FROM snippets WHERE folder_id = ?", folderID).Scan(&snippetCount)
	if err != nil {
		return fmt.Errorf("%w: failed to check for snippets in folder", ErrDatabaseError)
	}

	// Move snippets to root before deleting folder
	if snippetCount > 0 {
		_, err = tx.Exec("UPDATE snippets SET folder_id = NULL, updated_at = CURRENT_TIMESTAMP WHERE folder_id = ?", folderID)
		if err != nil {
			return fmt.Errorf("%w: failed to move snippets to root", ErrDatabaseError)
		}
	}

	result, err := tx.Exec("DELETE FROM folders WHERE id = ?", folderID)
	if err != nil {
		return fmt.Errorf("%w: failed to delete folder", ErrDatabaseError)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("%w: failed to verify deletion", ErrDatabaseError)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("folder with ID %d does not exist: %w", folderID, ErrNoFolderError)
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("%w: failed to commit deletion", ErrDatabaseError)
	}

	return nil
}

func checkCircularReference(tx *sql.Tx, userID int64, parentID int64, depth int) error {
	// Prevent infinite recursion
	if depth > 50 {
		return fmt.Errorf("maximum folder depth exceeded")
	}

	var grandParentID sql.NullInt64
	err := tx.QueryRow("SELECT parent_id FROM folders WHERE id = ? AND user_id = ?", parentID, userID).Scan(&grandParentID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		return fmt.Errorf("failed to check parent folder: %w", err)
	}

	if !grandParentID.Valid {
		return nil
	}

	return checkCircularReference(tx, userID, grandParentID.Int64, depth+1)
}

func checkCircularReferenceForUpdate(tx *sql.Tx, userID int64, folderID int64, newParentID int64, depth int) error {
	// Prevent infinite recursion
	if depth > 50 {
		return fmt.Errorf("maximum folder depth exceeded")
	}

	if newParentID == folderID {
		return fmt.Errorf("folder cannot be its own parent")
	}

	var grandParentID sql.NullInt64
	err := tx.QueryRow("SELECT parent_id FROM folders WHERE id = ? AND user_id = ?", newParentID, userID).Scan(&grandParentID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		return fmt.Errorf("failed to check parent folder: %w", err)
	}

	if !grandParentID.Valid {
		return nil
	}

	if grandParentID.Int64 == folderID {
		return fmt.Errorf("circular reference detected")
	}

	return checkCircularReferenceForUpdate(tx, userID, folderID, grandParentID.Int64, depth+1)
}

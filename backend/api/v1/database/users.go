package database

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/GHutch55/fragments/backend/api/v1/models"
)

var (
	ErrUsernameExists = errors.New("username already exists")
	ErrDatabaseError  = errors.New("database error occurred")
	ErrNoUserError    = errors.New("user does not exist")
)

func CreateUser(db *sql.DB, user *models.User) error {
	// Check if username exists first
	var count int
	checkQuery := "SELECT COUNT(*) FROM users WHERE username = ?"
	err := db.QueryRow(checkQuery, user.Username).Scan(&count)
	if err != nil {
		fmt.Printf("Database error during username check: %v\n", err)
		return fmt.Errorf("%w: failed to check username availability", ErrDatabaseError)
	}

	if count > 0 {
		return fmt.Errorf("%w: username '%s' is already taken", ErrUsernameExists, user.Username)
	}

	// Insert the new user
	insertQuery := `
		INSERT INTO users (username, display_name)
		VALUES (?, ?)`

	var displayNameValue interface{}
	if user.DisplayName != nil {
		displayNameValue = *user.DisplayName
	} else {
		displayNameValue = nil
	}

	result, err := db.Exec(insertQuery, user.Username, displayNameValue)
	if err != nil {
		// Handle constraint violation as backup (race condition case)
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return fmt.Errorf("%w: username became unavailable", ErrUsernameExists)
		}
		fmt.Printf("Database error during user creation: %v\n", err)
		return fmt.Errorf("%w: failed to create user", ErrDatabaseError)
	}

	// Get the user ID
	userID, err := result.LastInsertId()
	if err != nil {
		fmt.Printf("Error getting last insert ID: %v\n", err)
		return fmt.Errorf("%w: failed to retrieve user ID", ErrDatabaseError)
	}

	// Populate the user struct with the created data
	return populateUserFromDB(db, userID, user)
}

func GetUser(db *sql.DB, userID int64, user *models.User) error {
	err := populateUserFromDB(db, userID, user)
	if err != nil {
		// Check if it's a "not found" error specifically
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("user with ID %d not found", userID)
		}
		return err // Already wrapped by populateUserFromDB
	}
	return nil
}

func GetUsers(db *sql.DB, page, limit int, search string) ([]models.User, int, error) {
	offset := (page - 1) * limit

	whereClause := ""
	args := []interface{}{}
	if search != "" {
		whereClause = "WHERE username LIKE ? OR display_name LIKE ?"
		searchPattern := "%" + search + "%"
		args = append(args, searchPattern, searchPattern)
	}

	// Get total count
	var total int
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM users %s", whereClause)
	err := db.QueryRow(countQuery, args...).Scan(&total)
	if err != nil {
		fmt.Printf("Database error getting user count: %v\n", err)
		return nil, 0, fmt.Errorf("%w: failed to get user count", ErrDatabaseError)
	}

	// Get actual users
	dataQuery := fmt.Sprintf(`
		SELECT id, username, display_name, created_at, updated_at 
		FROM users %s 
		ORDER BY created_at DESC 
		LIMIT ? OFFSET ?`, whereClause)
	queryArgs := append(args, limit, offset)

	rows, err := db.Query(dataQuery, queryArgs...)
	if err != nil {
		fmt.Printf("Database error getting users: %v\n", err)
		return nil, 0, fmt.Errorf("%w: failed to get users", ErrDatabaseError)
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var user models.User
		var displayName sql.NullString

		err := rows.Scan(&user.ID, &user.Username, &displayName, &user.CreatedAt, &user.UpdatedAt)
		if err != nil {
			fmt.Printf("Database error scanning user row: %v\n", err)
			return nil, 0, fmt.Errorf("%w: failed to scan user data", ErrDatabaseError)
		}

		if displayName.Valid {
			user.DisplayName = &displayName.String
		} else {
			user.DisplayName = nil
		}

		users = append(users, user)
	}

	if err = rows.Err(); err != nil {
		fmt.Printf("Database error iterating users: %v\n", err)
		return nil, 0, fmt.Errorf("%w: failed to iterate users", ErrDatabaseError)
	}

	return users, total, nil
}

func DeleteUser(db *sql.DB, userID int64) error {
	deleteQuery := "DELETE FROM users WHERE id = ?"
	result, err := db.Exec(deleteQuery, userID)
	if err != nil {
		fmt.Printf("Database error deleting user ID %d: %v\n", userID, err)
		return fmt.Errorf("%w: failed to delete user", ErrDatabaseError)
	}

	// Check if any rows were actually deleted
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		fmt.Printf("Error checking rows affected for delete: %v\n", err)
		return fmt.Errorf("%w: failed to verify deletion", ErrDatabaseError)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("user with ID %d does not exist: %w", userID, ErrNoUserError)
	}

	return nil
}

func UpdateUser(db *sql.DB, userID int64, user *models.User) error {
	// Start a transaction for atomic operations
	tx, err := db.Begin()
	if err != nil {
		fmt.Printf("Error starting transaction: %v\n", err)
		return fmt.Errorf("%w: failed to start transaction", ErrDatabaseError)
	}
	defer tx.Rollback() // This will be a no-op if we commit successfully

	// Check if user exists and get current data
	selectQuery := `
		SELECT id, username, display_name, created_at, updated_at
		FROM users WHERE id = ?`

	var currentUser models.User
	var displayName sql.NullString
	err = tx.QueryRow(selectQuery, userID).Scan(
		&currentUser.ID,
		&currentUser.Username,
		&displayName,
		&currentUser.CreatedAt,
		&currentUser.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("user with ID %d does not exist: %w", userID, ErrNoUserError)
		}
		fmt.Printf("Database error retrieving user for update: %v\n", err)
		return fmt.Errorf("%w: failed to retrieve user for update", ErrDatabaseError)
	}

	// Convert display name for comparison
	if displayName.Valid {
		currentUser.DisplayName = &displayName.String
	} else {
		currentUser.DisplayName = nil
	}

	// Check if username is changing and if new username already exists
	if user.Username != currentUser.Username {
		var count int
		checkQuery := "SELECT COUNT(*) FROM users WHERE username = ? AND id != ?"
		err = tx.QueryRow(checkQuery, user.Username, userID).Scan(&count)
		if err != nil {
			fmt.Printf("Database error checking username availability: %v\n", err)
			return fmt.Errorf("%w: failed to check username availability", ErrDatabaseError)
		}
		if count > 0 {
			return fmt.Errorf("%w: username '%s' already exists", ErrUsernameExists, user.Username)
		}
	}

	var displayNameValue interface{}
	if user.DisplayName != nil {
		displayNameValue = *user.DisplayName
	} else {
		displayNameValue = nil
	}

	updateQuery := `
		UPDATE users 
		SET username = ?, display_name = ?, updated_at = CURRENT_TIMESTAMP 
		WHERE id = ?`

	_, err = tx.Exec(updateQuery, user.Username, displayNameValue, userID)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return fmt.Errorf("%w: username became unavailable", ErrUsernameExists)
		}
		fmt.Printf("Database error updating user: %v\n", err)
		return fmt.Errorf("%w: failed to update user", ErrDatabaseError)
	}

	err = tx.QueryRow(selectQuery, userID).Scan(
		&user.ID,
		&user.Username,
		&displayName,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		fmt.Printf("Error retrieving updated user: %v\n", err)
		return fmt.Errorf("%w: failed to retrieve updated user", ErrDatabaseError)
	}

	if displayName.Valid {
		user.DisplayName = &displayName.String
	} else {
		user.DisplayName = nil
	}

	// Commit the transaction
	if err = tx.Commit(); err != nil {
		fmt.Printf("Error committing transaction: %v\n", err)
		return fmt.Errorf("%w: failed to commit update", ErrDatabaseError)
	}

	return nil
}

func populateUserFromDB(db *sql.DB, userID int64, user *models.User) error {
	selectQuery := `
		SELECT id, username, display_name, created_at, updated_at
		FROM users WHERE id = ?`

	var displayName sql.NullString
	err := db.QueryRow(selectQuery, userID).Scan(
		&user.ID,
		&user.Username,
		&displayName,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return sql.ErrNoRows
		}
		fmt.Printf("Database error retrieving user ID %d: %v\n", userID, err)
		return fmt.Errorf("%w: failed to retrieve user", ErrDatabaseError)
	}

	if displayName.Valid {
		user.DisplayName = &displayName.String
	} else {
		user.DisplayName = nil
	}

	return nil
}

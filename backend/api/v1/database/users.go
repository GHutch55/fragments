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
)

func CreateUser(db *sql.DB, user *models.User) error {
	// check if username exists first
	var count int
	checkQuery := "SELECT COUNT(*) FROM users WHERE username = ?"
	err := db.QueryRow(checkQuery, user.Username).Scan(&count)
	if err != nil {
		fmt.Printf("Database error during username check %v\n", err)
		return fmt.Errorf("%w: failed to check username availability", ErrDatabaseError)
	}
	// handle username already existing
	if count > 0 {
		return fmt.Errorf("%w: username '%s' is already taken", ErrUsernameExists, user.Username)
	}

	// username available, proceeding
	insertQuery := `
		INSERT INTO users (username, display_name)
		VALUES (?, ?)`

	// Handle *string DisplayName - pass the pointer value or nil
	var displayNameValue interface{}
	if user.DisplayName != nil {
		displayNameValue = *user.DisplayName
	} else {
		displayNameValue = nil
	}

	result, err := db.Exec(insertQuery, user.Username, displayNameValue)
	if err != nil {
		// handle constraint violation as backup (race condition case)
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return fmt.Errorf("%w: username became unavailable", ErrUsernameExists)
		}
		// other errors
		fmt.Printf("Database error during user creation: %v\n", err)
		return fmt.Errorf("%w: failed to create user", ErrDatabaseError)
	}

	// get userID
	userID, err := result.LastInsertId()
	if err != nil {
		fmt.Printf("Error getting last insert ID: %v\n", err)
		return fmt.Errorf("%w: failed to retrieve user ID", ErrDatabaseError)
	}

	// query full user record to populate full struct
	selectQuery := `
		SELECT id, username, display_name, created_at, updated_at
		FROM users WHERE id = ?`

	// Use sql.NullString to handle NULL display_name from database
	var displayName sql.NullString
	err = db.QueryRow(selectQuery, userID).Scan(
		&user.ID,
		&user.Username,
		&displayName,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		fmt.Printf("Error retrieving created user: %v\n", err)
		return fmt.Errorf("%w: failed to retrieve created user", ErrDatabaseError)
	}

	// Convert sql.NullString back to *string
	if displayName.Valid {
		user.DisplayName = &displayName.String
	} else {
		user.DisplayName = nil
	}

	return nil
}

func GetUser(db *sql.DB, userID int64, user *models.User) error {
	getQuery := `
		SELECT id, username, display_name, created_at, updated_at
		FROM users
		WHERE id = ?`

	var displayName sql.NullString

	err := db.QueryRow(getQuery, userID).Scan(
		&user.ID,
		&user.Username,
		&displayName,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	// error checking
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("user with ID %d not found", userID)
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

func GetUsers(db *sql.DB, page, limit int, search string) ([]models.User, int, error) {
	offset := (page - 1) * limit

	// Build WHERE clause for search
	whereClause := ""
	args := []interface{}{}
	if search != "" {
		whereClause = "WHERE username LIKE ? OR display_name LIKE ?"
		searchPattern := "%" + search + "%"
		args = append(args, searchPattern, searchPattern)
	}

	// get total count
	var total int
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM users %s", whereClause)
	err := db.QueryRow(countQuery, args...).Scan(&total)
	if err != nil {
		fmt.Printf("Database error getting user count: %v\n", err)
		return nil, 0, fmt.Errorf("%w: failed to get user count", ErrDatabaseError)
	}

	// get actual users
	dataQuery := fmt.Sprintf(`
		SELECT id, username, display_name, created_at, updated_at FROM users %s ORDER BY created_at
		DESC LIMIT ? OFFSET ?`, whereClause)
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
			return nil, 0, err
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

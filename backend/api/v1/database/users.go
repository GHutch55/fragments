package database

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/GHutch55/fragments/backend/api/v1/models"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrUsernameExists = errors.New("username already exists")
	ErrDatabaseError  = errors.New("database error occurred")
	ErrNoUserError    = errors.New("user does not exist")
)

func CreateUser(ctx context.Context, pool *pgxpool.Pool, user *models.User) error {
	// Check if username exists first
	var count int
	checkQuery := "SELECT COUNT(*) FROM users WHERE username = $1"
	err := pool.QueryRow(ctx, checkQuery, user.Username).Scan(&count)
	if err != nil {
		fmt.Printf("Database error during username check: %v\n", err)
		return fmt.Errorf("%w: failed to check username availability", ErrDatabaseError)
	}

	if count > 0 {
		return fmt.Errorf("%w: username '%s' is already taken", ErrUsernameExists, user.Username)
	}

	// Insert the new user and return the new ID
	insertQuery := `INSERT INTO users (username) VALUES ($1) RETURNING id, created_at, updated_at`
	err = pool.QueryRow(ctx, insertQuery, user.Username).Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate key value violates unique constraint") {
			return fmt.Errorf("%w: username became unavailable", ErrUsernameExists)
		}
		fmt.Printf("Database error during user creation: %v\n", err)
		return fmt.Errorf("%w: failed to create user", ErrDatabaseError)
	}

	return nil
}

func GetUser(ctx context.Context, pool *pgxpool.Pool, userID int64, user *models.User) error {
	selectQuery := `
		SELECT id, username, created_at, updated_at
		FROM users WHERE id = $1`

	err := pool.QueryRow(ctx, selectQuery, userID).Scan(
		&user.ID,
		&user.Username,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNoUserError
		}
		fmt.Printf("Database error retrieving user ID %d: %v\n", userID, err)
		return fmt.Errorf("%w: failed to retrieve user", ErrDatabaseError)
	}

	return nil
}

func GetUsers(ctx context.Context, pool *pgxpool.Pool, page, limit int, search string) ([]models.User, int, error) {
	offset := (page - 1) * limit
	args := []interface{}{}
	whereClause := ""

	argPosition := 1
	if search != "" {
		whereClause = fmt.Sprintf("WHERE username ILIKE $%d", argPosition)
		args = append(args, "%"+search+"%")
		argPosition++
	}

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM users %s", whereClause)
	var total int
	err := pool.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		fmt.Printf("Database error getting user count: %v\n", err)
		return nil, 0, fmt.Errorf("%w: failed to get user count", ErrDatabaseError)
	}

	dataQuery := fmt.Sprintf(`
		SELECT id, username, created_at, updated_at
		FROM users
		%s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d`, whereClause, argPosition, argPosition+1)

	args = append(args, limit, offset)

	rows, err := pool.Query(ctx, dataQuery, args...)
	if err != nil {
		fmt.Printf("Database error getting users: %v\n", err)
		return nil, 0, fmt.Errorf("%w: failed to get users", ErrDatabaseError)
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var user models.User
		err := rows.Scan(&user.ID, &user.Username, &user.CreatedAt, &user.UpdatedAt)
		if err != nil {
			fmt.Printf("Database error scanning user row: %v\n", err)
			return nil, 0, fmt.Errorf("%w: failed to scan user data", ErrDatabaseError)
		}
		users = append(users, user)
	}

	if err = rows.Err(); err != nil {
		fmt.Printf("Database error iterating users: %v\n", err)
		return nil, 0, fmt.Errorf("%w: failed to iterate users", ErrDatabaseError)
	}

	return users, total, nil
}

func DeleteUser(ctx context.Context, pool *pgxpool.Pool, userID int64) error {
	deleteQuery := "DELETE FROM users WHERE id = $1"
	result, err := pool.Exec(ctx, deleteQuery, userID)
	if err != nil {
		fmt.Printf("Database error deleting user ID %d: %v\n", userID, err)
		return fmt.Errorf("%w: failed to delete user", ErrDatabaseError)
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("user with ID %d does not exist: %w", userID, ErrNoUserError)
	}

	return nil
}

func UpdateUser(ctx context.Context, pool *pgxpool.Pool, userID int64, user *models.User) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		fmt.Printf("Error starting transaction: %v\n", err)
		return fmt.Errorf("%w: failed to start transaction", ErrDatabaseError)
	}
	defer tx.Rollback(ctx)

	selectQuery := `
		SELECT id, username, created_at, updated_at
		FROM users WHERE id = $1`

	var currentUser models.User
	err = tx.QueryRow(ctx, selectQuery, userID).Scan(
		&currentUser.ID,
		&currentUser.Username,
		&currentUser.CreatedAt,
		&currentUser.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("user with ID %d does not exist: %w", userID, ErrNoUserError)
		}
		fmt.Printf("Database error retrieving user for update: %v\n", err)
		return fmt.Errorf("%w: failed to retrieve user for update", ErrDatabaseError)
	}

	if user.Username != currentUser.Username {
		var count int
		checkQuery := "SELECT COUNT(*) FROM users WHERE username = $1 AND id != $2"
		err = tx.QueryRow(ctx, checkQuery, user.Username, userID).Scan(&count)
		if err != nil {
			fmt.Printf("Database error checking username availability: %v\n", err)
			return fmt.Errorf("%w: failed to check username availability", ErrDatabaseError)
		}
		if count > 0 {
			return fmt.Errorf("%w: username '%s' already exists", ErrUsernameExists, user.Username)
		}
	}

	updateQuery := `
		UPDATE users
		SET username = $1, updated_at = CURRENT_TIMESTAMP
		WHERE id = $2
		RETURNING created_at, updated_at`

	err = tx.QueryRow(ctx, updateQuery, user.Username, userID).Scan(&user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate key value violates unique constraint") {
			return fmt.Errorf("%w: username became unavailable", ErrUsernameExists)
		}
		fmt.Printf("Database error updating user: %v\n", err)
		return fmt.Errorf("%w: failed to update user", ErrDatabaseError)
	}

	user.ID = userID

	if err = tx.Commit(ctx); err != nil {
		fmt.Printf("Error committing transaction: %v\n", err)
		return fmt.Errorf("%w: failed to commit update", ErrDatabaseError)
	}

	return nil
}

// Helper functions for auth functionality
func IsUsernameExistsError(err error) bool {
	return errors.Is(err, ErrUsernameExists)
}

func IsUserNotFoundError(err error) bool {
	return errors.Is(err, ErrNoUserError) || errors.Is(err, pgx.ErrNoRows)
}

// Extended user model for auth with password
type UserWithPassword struct {
	models.User
	Password string `json:"-"` // Don't include in JSON responses
}

func CreateUserWithPassword(ctx context.Context, pool *pgxpool.Pool, user *UserWithPassword) error {
	// Check if username exists first
	var count int
	checkQuery := "SELECT COUNT(*) FROM users WHERE username = $1"
	err := pool.QueryRow(ctx, checkQuery, user.Username).Scan(&count)
	if err != nil {
		fmt.Printf("Database error during username check: %v\n", err)
		return fmt.Errorf("%w: failed to check username availability", ErrDatabaseError)
	}

	if count > 0 {
		return fmt.Errorf("%w: username '%s' is already taken", ErrUsernameExists, user.Username)
	}

	// Insert the new user with password and RETURNING id
	insertQuery := `
        INSERT INTO users (username, password_hash)
        VALUES ($1, $2)
        RETURNING id, created_at, updated_at`

	err = pool.QueryRow(ctx, insertQuery, user.Username, user.Password).Scan(
		&user.ID,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate key value violates unique constraint") {
			return fmt.Errorf("%w: username became unavailable", ErrUsernameExists)
		}
		fmt.Printf("Database error during user creation: %v\n", err)
		return fmt.Errorf("%w: failed to create user", ErrDatabaseError)
	}

	return nil
}

func GetUserByUsername(ctx context.Context, pool *pgxpool.Pool, username string) (*UserWithPassword, error) {
	selectQuery := `
        SELECT id, username, password_hash, created_at, updated_at
        FROM users WHERE username = $1`

	var user UserWithPassword
	err := pool.QueryRow(ctx, selectQuery, username).Scan(
		&user.ID,
		&user.Username,
		&user.Password,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("user not found")
		}
		fmt.Printf("Database error retrieving user by username: %v\n", err)
		return nil, fmt.Errorf("%w: failed to retrieve user", ErrDatabaseError)
	}

	return &user, nil
}

func UpdateUserPassword(ctx context.Context, pool *pgxpool.Pool, userID int64, hashedPassword string) error {
	updateQuery := `
        UPDATE users
        SET password_hash = $1, updated_at = CURRENT_TIMESTAMP
        WHERE id = $2`

	result, err := pool.Exec(ctx, updateQuery, hashedPassword, userID)
	if err != nil {
		fmt.Printf("Database error updating password for user ID %d: %v\n", userID, err)
		return fmt.Errorf("%w: failed to update password", ErrDatabaseError)
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("user with ID %d does not exist: %w", userID, ErrNoUserError)
	}

	return nil
}

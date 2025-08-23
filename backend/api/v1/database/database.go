package database

import (
	"database/sql"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
)

func Connect(dbPath string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	if err2 := db.Ping(); err2 != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err2)
	}

	_, err3 := db.Exec(`PRAGMA journal_mode = WAL;`)
	if err3 != nil {
		return nil, fmt.Errorf("failed to set journal mode: %w", err3)
	}

	return db, nil
}

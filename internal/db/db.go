// Package db provides SQLite database initialization and access.
package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

// DefaultPath returns the default database path: ~/.house-finder/houses.db
func DefaultPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("getting home directory: %w", err)
	}
	return filepath.Join(home, ".house-finder", "houses.db"), nil
}

// Open opens (or creates) a SQLite database at the given path,
// enables WAL mode and foreign keys, and runs migrations.
func Open(path string) (*sql.DB, error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("creating database directory %s: %w", dir, err)
	}

	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}

	if err := configure(db); err != nil {
		closeErr := db.Close()
		if closeErr != nil {
			return nil, fmt.Errorf("%w (also failed to close: %v)", err, closeErr)
		}
		return nil, err
	}

	if err := migrate(db); err != nil {
		closeErr := db.Close()
		if closeErr != nil {
			return nil, fmt.Errorf("running migrations: %w (also failed to close: %v)", err, closeErr)
		}
		return nil, fmt.Errorf("running migrations: %w", err)
	}

	return db, nil
}

// configure sets SQLite pragmas for WAL mode and foreign keys.
func configure(db *sql.DB) error {
	pragmas := []string{
		"PRAGMA journal_mode=WAL",
		"PRAGMA foreign_keys=ON",
	}

	for _, p := range pragmas {
		if _, err := db.Exec(p); err != nil {
			return fmt.Errorf("executing %s: %w", p, err)
		}
	}

	return nil
}

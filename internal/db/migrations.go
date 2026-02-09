package db

import (
	"database/sql"
	"fmt"
)

// migrations is an ordered list of SQL statements to run.
// Each migration runs inside a transaction.
var migrations = []string{
	`CREATE TABLE IF NOT EXISTS properties (
		id            INTEGER PRIMARY KEY AUTOINCREMENT,
		address       TEXT    NOT NULL,
		mpr_id        TEXT    NOT NULL UNIQUE,
		realtor_url   TEXT    NOT NULL,
		price         INTEGER,
		bedrooms      REAL,
		bathrooms     REAL,
		sqft          INTEGER,
		lot_size      REAL,
		year_built    INTEGER,
		property_type TEXT,
		status        TEXT,
		rating        INTEGER CHECK (rating IS NULL OR (rating >= 1 AND rating <= 4)),
		raw_json      TEXT    NOT NULL,
		created_at    DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at    DATETIME DEFAULT CURRENT_TIMESTAMP
	)`,
	`CREATE TABLE IF NOT EXISTS comments (
		id          INTEGER PRIMARY KEY AUTOINCREMENT,
		property_id INTEGER NOT NULL REFERENCES properties(id) ON DELETE CASCADE,
		text        TEXT    NOT NULL,
		created_at  DATETIME DEFAULT CURRENT_TIMESTAMP
	)`,
	`CREATE TABLE IF NOT EXISTS auth_tokens (
		id         INTEGER PRIMARY KEY AUTOINCREMENT,
		token      TEXT     NOT NULL UNIQUE,
		email      TEXT     NOT NULL,
		expires_at DATETIME NOT NULL,
		used       INTEGER  DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`,
	`CREATE TABLE IF NOT EXISTS sessions (
		id         TEXT     PRIMARY KEY,
		email      TEXT     NOT NULL,
		expires_at DATETIME NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`,
	`CREATE TABLE IF NOT EXISTS passkey_credentials (
		id              TEXT    PRIMARY KEY,
		email           TEXT    NOT NULL,
		name            TEXT    NOT NULL DEFAULT '',
		credential_json TEXT    NOT NULL,
		created_at      DATETIME DEFAULT CURRENT_TIMESTAMP
	)`,
	`CREATE TABLE IF NOT EXISTS api_keys (
		id           INTEGER  PRIMARY KEY AUTOINCREMENT,
		name         TEXT     NOT NULL,
		key_prefix   TEXT     NOT NULL,
		key_hash     TEXT     NOT NULL UNIQUE,
		created_at   DATETIME DEFAULT CURRENT_TIMESTAMP,
		last_used_at DATETIME
	)`,
	`CREATE TABLE IF NOT EXISTS authorized_users (
		id         INTEGER PRIMARY KEY AUTOINCREMENT,
		email      TEXT    NOT NULL UNIQUE,
		name       TEXT    NOT NULL DEFAULT '',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`,
}

// migrate runs all migrations in order.
func migrate(db *sql.DB) error {
	for i, m := range migrations {
		if _, err := db.Exec(m); err != nil {
			return fmt.Errorf("migration %d: %w", i, err)
		}
	}

	// Column additions (idempotent — checks if column exists first)
	columnMigrations := []struct {
		table, column, definition string
	}{
		{"comments", "author", "TEXT NOT NULL DEFAULT ''"},
		{"api_keys", "email", "TEXT NOT NULL DEFAULT ''"},
		{"authorized_users", "phone", "TEXT NOT NULL DEFAULT ''"},
		{"authorized_users", "is_realtor", "INTEGER NOT NULL DEFAULT 0"},
	}

	for _, cm := range columnMigrations {
		if err := addColumnIfNotExists(db, cm.table, cm.column, cm.definition); err != nil {
			return fmt.Errorf("adding %s.%s: %w", cm.table, cm.column, err)
		}
	}

	return nil
}

// addColumnIfNotExists adds a column to a table if it doesn't already exist.
func addColumnIfNotExists(db *sql.DB, table, column, definition string) error {
	rows, err := db.Query(fmt.Sprintf("PRAGMA table_info(%s)", table))
	if err != nil {
		return fmt.Errorf("checking table info: %w", err)
	}
	defer func() {
		if cerr := rows.Close(); cerr != nil {
			fmt.Printf("warning: closing rows: %v\n", cerr)
		}
	}()

	for rows.Next() {
		var cid int
		var name, colType string
		var notNull, pk int
		var dfltValue interface{}
		if err := rows.Scan(&cid, &name, &colType, &notNull, &dfltValue, &pk); err != nil {
			return fmt.Errorf("scanning column info: %w", err)
		}
		if name == column {
			return nil // column already exists
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterating columns: %w", err)
	}

	_, err = db.Exec(fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", table, column, definition))
	return err
}

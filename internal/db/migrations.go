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
}

// migrate runs all migrations in order.
func migrate(db *sql.DB) error {
	for i, m := range migrations {
		if _, err := db.Exec(m); err != nil {
			return fmt.Errorf("migration %d: %w", i, err)
		}
	}
	return nil
}

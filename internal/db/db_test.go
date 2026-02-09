package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestOpen(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(t *testing.T) string
		wantErr bool
	}{
		{
			name: "creates new database",
			setup: func(t *testing.T) string {
				return filepath.Join(t.TempDir(), "houses.db")
			},
		},
		{
			name: "creates nested directories",
			setup: func(t *testing.T) string {
				return filepath.Join(t.TempDir(), "a", "b", "houses.db")
			},
		},
		{
			name: "opens existing database",
			setup: func(t *testing.T) string {
				path := filepath.Join(t.TempDir(), "houses.db")
				d, err := Open(path)
				if err != nil {
					t.Fatalf("setup: %v", err)
				}
				if err := d.Close(); err != nil {
					t.Fatalf("setup close: %v", err)
				}
				return path
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := tt.setup(t)
			d, err := Open(path)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			defer func() {
				if err := d.Close(); err != nil {
					t.Errorf("close: %v", err)
				}
			}()

			if _, err := os.Stat(path); os.IsNotExist(err) {
				t.Error("database file was not created")
			}
		})
	}
}

func TestWALMode(t *testing.T) {
	d := openTestDB(t)

	var mode string
	if err := d.QueryRow("PRAGMA journal_mode").Scan(&mode); err != nil {
		t.Fatalf("query journal_mode: %v", err)
	}
	if mode != "wal" {
		t.Errorf("journal_mode = %q, want %q", mode, "wal")
	}
}

func TestForeignKeys(t *testing.T) {
	d := openTestDB(t)

	var fk int
	if err := d.QueryRow("PRAGMA foreign_keys").Scan(&fk); err != nil {
		t.Fatalf("query foreign_keys: %v", err)
	}
	if fk != 1 {
		t.Errorf("foreign_keys = %d, want 1", fk)
	}
}

func TestMigrations(t *testing.T) {
	tests := []struct {
		name  string
		table string
		cols  []string
	}{
		{
			name:  "properties table exists",
			table: "properties",
			cols:  []string{"id", "address", "mpr_id", "realtor_url", "price", "bedrooms", "bathrooms", "sqft", "lot_size", "year_built", "property_type", "status", "rating", "raw_json", "created_at", "updated_at"},
		},
		{
			name:  "comments table exists",
			table: "comments",
			cols:  []string{"id", "property_id", "text", "created_at"},
		},
		{
			name:  "auth_tokens table exists",
			table: "auth_tokens",
			cols:  []string{"id", "token", "email", "expires_at", "used", "created_at"},
		},
		{
			name:  "sessions table exists",
			table: "sessions",
			cols:  []string{"id", "email", "expires_at", "created_at"},
		},
		{
			name:  "passkey_credentials table exists",
			table: "passkey_credentials",
			cols:  []string{"id", "email", "name", "credential_json", "created_at"},
		},
	}

	d := openTestDB(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cols := tableColumns(t, d, tt.table)
			if len(cols) != len(tt.cols) {
				t.Fatalf("got %d columns, want %d: %v", len(cols), len(tt.cols), cols)
			}
			for i, want := range tt.cols {
				if cols[i] != want {
					t.Errorf("column %d = %q, want %q", i, cols[i], want)
				}
			}
		})
	}
}

func TestRatingConstraint(t *testing.T) {
	d := openTestDB(t)

	insert := `INSERT INTO properties (address, mpr_id, realtor_url, raw_json) VALUES (?, ?, ?, ?)`
	insertWithRating := `INSERT INTO properties (address, mpr_id, realtor_url, raw_json, rating) VALUES (?, ?, ?, ?, ?)`

	tests := []struct {
		name    string
		rating  interface{}
		wantErr bool
	}{
		{"null rating is valid", nil, false},
		{"rating 1 is valid", 1, false},
		{"rating 4 is valid", 4, false},
		{"rating 0 is invalid", 0, true},
		{"rating 5 is invalid", 5, true},
		{"rating -1 is invalid", -1, true},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mprID := fmt.Sprintf("mpr-%d", i)
			var err error
			if tt.rating == nil {
				_, err = d.Exec(insert, "123 Test St", mprID, "https://example.com", "{}")
			} else {
				_, err = d.Exec(insertWithRating, "123 Test St", mprID, "https://example.com", "{}", tt.rating)
			}
			if tt.wantErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestCascadeDelete(t *testing.T) {
	d := openTestDB(t)

	// Insert a property
	res, err := d.Exec(
		`INSERT INTO properties (address, mpr_id, realtor_url, raw_json) VALUES (?, ?, ?, ?)`,
		"123 Test St", "mpr-cascade", "https://example.com", "{}",
	)
	if err != nil {
		t.Fatalf("insert property: %v", err)
	}
	propID, err := res.LastInsertId()
	if err != nil {
		t.Fatalf("last insert id: %v", err)
	}

	// Insert comments
	for i := 0; i < 3; i++ {
		_, err = d.Exec(
			`INSERT INTO comments (property_id, text) VALUES (?, ?)`,
			propID, fmt.Sprintf("comment %d", i),
		)
		if err != nil {
			t.Fatalf("insert comment %d: %v", i, err)
		}
	}

	// Verify comments exist
	var count int
	if err := d.QueryRow(`SELECT COUNT(*) FROM comments WHERE property_id = ?`, propID).Scan(&count); err != nil {
		t.Fatalf("count comments: %v", err)
	}
	if count != 3 {
		t.Fatalf("expected 3 comments, got %d", count)
	}

	// Delete property
	if _, err := d.Exec(`DELETE FROM properties WHERE id = ?`, propID); err != nil {
		t.Fatalf("delete property: %v", err)
	}

	// Verify comments are gone
	if err := d.QueryRow(`SELECT COUNT(*) FROM comments WHERE property_id = ?`, propID).Scan(&count); err != nil {
		t.Fatalf("count comments after delete: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 comments after cascade delete, got %d", count)
	}
}

func TestMigrationsIdempotent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "houses.db")

	// Open twice — migrations should not fail on second run
	d1, err := Open(path)
	if err != nil {
		t.Fatalf("first open: %v", err)
	}
	if err := d1.Close(); err != nil {
		t.Fatalf("first close: %v", err)
	}

	d2, err := Open(path)
	if err != nil {
		t.Fatalf("second open (idempotency): %v", err)
	}
	if err := d2.Close(); err != nil {
		t.Fatalf("second close: %v", err)
	}
}

func TestDefaultPath(t *testing.T) {
	p, err := DefaultPath()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if filepath.Base(p) != "houses.db" {
		t.Errorf("expected filename houses.db, got %s", filepath.Base(p))
	}

	dir := filepath.Base(filepath.Dir(p))
	if dir != "hf" {
		t.Errorf("expected directory hf, got %s", dir)
	}
}

// openTestDB creates a temporary database for testing.
func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	path := filepath.Join(t.TempDir(), "houses.db")
	d, err := Open(path)
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() {
		if err := d.Close(); err != nil {
			t.Errorf("close test db: %v", err)
		}
	})
	return d
}

// tableColumns returns column names for a table using PRAGMA table_info.
func tableColumns(t *testing.T, d *sql.DB, table string) []string {
	t.Helper()
	rows, err := d.Query(fmt.Sprintf("PRAGMA table_info(%s)", table))
	if err != nil {
		t.Fatalf("pragma table_info(%s): %v", table, err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			t.Errorf("close rows: %v", err)
		}
	}()

	var cols []string
	for rows.Next() {
		var cid int
		var name, typ string
		var notnull int
		var dflt *string
		var pk int
		if err := rows.Scan(&cid, &name, &typ, &notnull, &dflt, &pk); err != nil {
			t.Fatalf("scan: %v", err)
		}
		cols = append(cols, name)
	}
	return cols
}

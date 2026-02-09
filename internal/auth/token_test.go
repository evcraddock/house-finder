package auth

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/evcraddock/house-finder/internal/db"
)

func TestTokenCreateAndValidate(t *testing.T) {
	store := testTokenStore(t)

	token, err := store.Create("admin@example.com")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if token == "" {
		t.Fatal("expected non-empty token")
	}

	email, err := store.Validate(token)
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	if email != "admin@example.com" {
		t.Errorf("email = %q, want %q", email, "admin@example.com")
	}
}

func TestTokenSingleUse(t *testing.T) {
	store := testTokenStore(t)

	token, err := store.Create("admin@example.com")
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	// First use succeeds
	if _, err := store.Validate(token); err != nil {
		t.Fatalf("first validate: %v", err)
	}

	// Second use fails
	if _, err := store.Validate(token); err == nil {
		t.Fatal("expected error on second use")
	}
}

func TestTokenInvalid(t *testing.T) {
	store := testTokenStore(t)

	if _, err := store.Validate("nonexistent-token"); err == nil {
		t.Fatal("expected error for invalid token")
	}
}

func TestTokenExpired(t *testing.T) {
	store := testTokenStore(t)

	// Insert a token that's already expired
	d := store.db
	if _, err := d.Exec(
		"INSERT INTO auth_tokens (token, email, expires_at) VALUES (?, ?, ?)",
		"expired-token", "admin@example.com", time.Now().Add(-1*time.Hour),
	); err != nil {
		t.Fatalf("insert expired token: %v", err)
	}

	if _, err := store.Validate("expired-token"); err == nil {
		t.Fatal("expected error for expired token")
	}
}

func TestTokenCleanup(t *testing.T) {
	store := testTokenStore(t)

	d := store.db
	// Insert expired token
	if _, err := d.Exec(
		"INSERT INTO auth_tokens (token, email, expires_at) VALUES (?, ?, ?)",
		"old-token", "admin@example.com", time.Now().Add(-1*time.Hour),
	); err != nil {
		t.Fatalf("insert: %v", err)
	}

	if err := store.Cleanup(); err != nil {
		t.Fatalf("cleanup: %v", err)
	}

	var count int
	if err := d.QueryRow("SELECT COUNT(*) FROM auth_tokens WHERE token = ?", "old-token").Scan(&count); err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 expired tokens, got %d", count)
	}
}

func testTokenStore(t *testing.T) *TokenStore {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.db")
	d, err := db.Open(path)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() {
		if err := d.Close(); err != nil {
			t.Errorf("close db: %v", err)
		}
	})
	return NewTokenStore(d)
}

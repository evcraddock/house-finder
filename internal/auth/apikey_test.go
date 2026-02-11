package auth

import (
	"path/filepath"
	"testing"

	"github.com/evcraddock/house-finder/internal/db"
)

func TestAPIKeyCreateAndValidate(t *testing.T) {
	store := testAPIKeyStore(t)

	rawKey, key, err := store.Create("Test Key", "test@example.com")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if rawKey == "" {
		t.Fatal("expected non-empty raw key")
	}
	if key.Name != "Test Key" {
		t.Errorf("name = %q, want %q", key.Name, "Test Key")
	}
	if key.KeyPrefix == "" {
		t.Error("expected non-empty key prefix")
	}
	if len(rawKey) < 10 {
		t.Error("raw key too short")
	}

	// Validate with correct key
	valid, err := store.Validate(rawKey)
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	if valid == "" {
		t.Error("expected valid key")
	}
}

func TestAPIKeyValidateInvalid(t *testing.T) {
	store := testAPIKeyStore(t)

	valid, err := store.Validate("hf_boguskey12345678")
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	if valid != "" {
		t.Error("expected invalid key")
	}
}

func TestAPIKeyList(t *testing.T) {
	store := testAPIKeyStore(t)

	if _, _, err := store.Create("Key 1", "test@example.com"); err != nil {
		t.Fatalf("create 1: %v", err)
	}
	if _, _, err := store.Create("Key 2", "test@example.com"); err != nil {
		t.Fatalf("create 2: %v", err)
	}

	keys, err := store.List("test@example.com")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(keys) != 2 {
		t.Fatalf("got %d keys, want 2", len(keys))
	}

	// Other user should see zero keys
	other, err := store.List("other@example.com")
	if err != nil {
		t.Fatalf("list other: %v", err)
	}
	if len(other) != 0 {
		t.Errorf("got %d keys for other user, want 0", len(other))
	}
}

func TestAPIKeyDelete(t *testing.T) {
	store := testAPIKeyStore(t)

	rawKey, key, err := store.Create("To Delete", "test@example.com")
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	if err := store.Delete(key.ID, "test@example.com"); err != nil {
		t.Fatalf("delete: %v", err)
	}

	// Should no longer validate
	valid, err := store.Validate(rawKey)
	if err != nil {
		t.Fatalf("validate after delete: %v", err)
	}
	if valid != "" {
		t.Error("expected invalid after delete")
	}

	// List should be empty
	keys, err := store.List("test@example.com")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(keys) != 0 {
		t.Errorf("got %d keys, want 0", len(keys))
	}
}

func TestAPIKeyDeleteNotFound(t *testing.T) {
	store := testAPIKeyStore(t)

	if err := store.Delete(999, "test@example.com"); err == nil {
		t.Fatal("expected error deleting nonexistent key")
	}
}

func TestAPIKeyDeleteWrongOwner(t *testing.T) {
	store := testAPIKeyStore(t)

	_, key, err := store.Create("Owned Key", "alice@example.com")
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	// Bob should not be able to delete Alice's key
	if err := store.Delete(key.ID, "bob@example.com"); err == nil {
		t.Fatal("expected error deleting another user's key")
	}

	// Alice can delete her own key
	if err := store.Delete(key.ID, "alice@example.com"); err != nil {
		t.Fatalf("delete own key: %v", err)
	}
}

func TestAPIKeyValidateReturnsEmail(t *testing.T) {
	store := testAPIKeyStore(t)

	rawKey, _, err := store.Create("Email Key", "alice@example.com")
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	email, err := store.Validate(rawKey)
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	if email != "alice@example.com" {
		t.Errorf("email = %q, want %q", email, "alice@example.com")
	}
}

func TestAPIKeyUpdatesLastUsed(t *testing.T) {
	store := testAPIKeyStore(t)

	rawKey, _, err := store.Create("Usage Key", "test@example.com")
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	// Before validation, last_used_at should be nil
	keys, err := store.List("test@example.com")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if keys[0].LastUsedAt != nil {
		t.Error("expected nil last_used_at before first use")
	}

	// Validate to update last_used_at
	if _, err := store.Validate(rawKey); err != nil {
		t.Fatalf("validate: %v", err)
	}

	keys, err = store.List("test@example.com")
	if err != nil {
		t.Fatalf("list after use: %v", err)
	}
	if keys[0].LastUsedAt == nil {
		t.Error("expected non-nil last_used_at after use")
	}
}

func TestAPIKeyPrefix(t *testing.T) {
	store := testAPIKeyStore(t)

	rawKey, key, err := store.Create("Prefix Key", "test@example.com")
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	// Key should start with "hf_" and prefix should be first 8 chars
	if rawKey[:3] != "hf_" {
		t.Errorf("raw key should start with hf_, got %q", rawKey[:3])
	}
	if key.KeyPrefix != rawKey[:8] {
		t.Errorf("prefix = %q, want %q", key.KeyPrefix, rawKey[:8])
	}
}

func testAPIKeyStore(t *testing.T) *APIKeyStore {
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
	return NewAPIKeyStore(d)
}

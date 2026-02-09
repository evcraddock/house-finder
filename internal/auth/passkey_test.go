package auth

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/go-webauthn/webauthn/webauthn"

	"github.com/evcraddock/house-finder/internal/db"
)

func TestPasskeySaveAndList(t *testing.T) {
	store := testPasskeyStore(t)

	cred := &webauthn.Credential{
		ID:        []byte("test-credential-id"),
		PublicKey: []byte("test-public-key"),
	}

	if err := store.Save("admin@example.com", "My Laptop", cred); err != nil {
		t.Fatalf("save: %v", err)
	}

	stored, err := store.ListByEmail("admin@example.com")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(stored) != 1 {
		t.Fatalf("got %d credentials, want 1", len(stored))
	}
	if stored[0].Name != "My Laptop" {
		t.Errorf("name = %q, want %q", stored[0].Name, "My Laptop")
	}
	if stored[0].Email != "admin@example.com" {
		t.Errorf("email = %q, want %q", stored[0].Email, "admin@example.com")
	}
	if string(stored[0].Credential.ID) != string(cred.ID) {
		t.Errorf("credential ID mismatch")
	}
}

func TestPasskeyWebAuthnCredentials(t *testing.T) {
	store := testPasskeyStore(t)

	cred1 := &webauthn.Credential{ID: []byte("cred-1"), PublicKey: []byte("key-1")}
	cred2 := &webauthn.Credential{ID: []byte("cred-2"), PublicKey: []byte("key-2")}

	if err := store.Save("admin@example.com", "Key 1", cred1); err != nil {
		t.Fatalf("save 1: %v", err)
	}
	if err := store.Save("admin@example.com", "Key 2", cred2); err != nil {
		t.Fatalf("save 2: %v", err)
	}

	creds, err := store.WebAuthnCredentials("admin@example.com")
	if err != nil {
		t.Fatalf("webauthn credentials: %v", err)
	}
	if len(creds) != 2 {
		t.Fatalf("got %d credentials, want 2", len(creds))
	}
}

func TestPasskeyListEmpty(t *testing.T) {
	store := testPasskeyStore(t)

	stored, err := store.ListByEmail("nobody@example.com")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(stored) != 0 {
		t.Errorf("got %d credentials, want 0", len(stored))
	}
}

func TestPasskeyDelete(t *testing.T) {
	store := testPasskeyStore(t)

	cred := &webauthn.Credential{
		ID:        []byte("delete-me"),
		PublicKey: []byte("key"),
	}

	if err := store.Save("admin@example.com", "To Delete", cred); err != nil {
		t.Fatalf("save: %v", err)
	}

	id := fmt.Sprintf("%x", cred.ID)
	if err := store.Delete(id, "admin@example.com"); err != nil {
		t.Fatalf("delete: %v", err)
	}

	stored, err := store.ListByEmail("admin@example.com")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(stored) != 0 {
		t.Errorf("got %d credentials after delete, want 0", len(stored))
	}
}

func TestPasskeyDeleteWrongEmail(t *testing.T) {
	store := testPasskeyStore(t)

	cred := &webauthn.Credential{
		ID:        []byte("someone-elses"),
		PublicKey: []byte("key"),
	}

	if err := store.Save("admin@example.com", "Admin Key", cred); err != nil {
		t.Fatalf("save: %v", err)
	}

	id := fmt.Sprintf("%x", cred.ID)
	if err := store.Delete(id, "intruder@example.com"); err == nil {
		t.Fatal("expected error deleting with wrong email")
	}
}

func TestPasskeyUser(t *testing.T) {
	cred := webauthn.Credential{ID: []byte("test"), PublicKey: []byte("key")}
	user := NewPasskeyUser("admin@example.com", []webauthn.Credential{cred})

	if user.WebAuthnName() != "admin@example.com" {
		t.Errorf("name = %q", user.WebAuthnName())
	}
	if user.WebAuthnDisplayName() != "admin@example.com" {
		t.Errorf("display name = %q", user.WebAuthnDisplayName())
	}
	if len(user.WebAuthnID()) != 32 {
		t.Errorf("ID length = %d, want 32 (sha256)", len(user.WebAuthnID()))
	}
	if len(user.WebAuthnCredentials()) != 1 {
		t.Errorf("credentials = %d, want 1", len(user.WebAuthnCredentials()))
	}
}

func testPasskeyStore(t *testing.T) *PasskeyStore {
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
	return NewPasskeyStore(d)
}

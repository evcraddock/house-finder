package auth

import (
	"path/filepath"
	"testing"

	"github.com/evcraddock/house-finder/internal/db"
)

func testUserStore(t *testing.T) *UserStore {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.db")
	d, err := db.Open(path)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() {
		if cerr := d.Close(); cerr != nil {
			t.Errorf("close db: %v", cerr)
		}
	})
	return NewUserStore(d, "admin@example.com")
}

func TestIsAuthorizedAdmin(t *testing.T) {
	s := testUserStore(t)

	if !s.IsAuthorized("admin@example.com") {
		t.Error("admin should be authorized")
	}
	if !s.IsAuthorized("Admin@Example.COM") {
		t.Error("admin check should be case-insensitive")
	}
}

func TestIsAuthorizedUnknown(t *testing.T) {
	s := testUserStore(t)

	if s.IsAuthorized("nobody@example.com") {
		t.Error("unknown email should not be authorized")
	}
}

func TestAddAndIsAuthorized(t *testing.T) {
	s := testUserStore(t)

	user, err := s.Add("bob@example.com", "Bob")
	if err != nil {
		t.Fatalf("add: %v", err)
	}
	if user.Email != "bob@example.com" {
		t.Errorf("email = %q", user.Email)
	}
	if user.Name != "Bob" {
		t.Errorf("name = %q", user.Name)
	}

	if !s.IsAuthorized("bob@example.com") {
		t.Error("added user should be authorized")
	}
	if !s.IsAuthorized("Bob@Example.COM") {
		t.Error("authorized check should be case-insensitive")
	}
}

func TestAddDuplicate(t *testing.T) {
	s := testUserStore(t)

	if _, err := s.Add("bob@example.com", "Bob"); err != nil {
		t.Fatalf("first add: %v", err)
	}

	_, err := s.Add("bob@example.com", "Bob Again")
	if err == nil {
		t.Fatal("expected error for duplicate")
	}
}

func TestAddEmptyEmail(t *testing.T) {
	s := testUserStore(t)

	_, err := s.Add("", "No Email")
	if err == nil {
		t.Fatal("expected error for empty email")
	}
}

func TestList(t *testing.T) {
	s := testUserStore(t)

	if _, err := s.Add("alice@example.com", "Alice"); err != nil {
		t.Fatalf("add alice: %v", err)
	}
	if _, err := s.Add("bob@example.com", "Bob"); err != nil {
		t.Fatalf("add bob: %v", err)
	}

	users, err := s.List()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(users) != 2 {
		t.Fatalf("got %d users, want 2", len(users))
	}
	// Should be ordered by email
	if users[0].Email != "alice@example.com" {
		t.Errorf("first user = %q, want alice", users[0].Email)
	}
}

func TestDeleteUser(t *testing.T) {
	s := testUserStore(t)

	user, err := s.Add("bob@example.com", "Bob")
	if err != nil {
		t.Fatalf("add: %v", err)
	}

	if err := s.Delete(user.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}

	if s.IsAuthorized("bob@example.com") {
		t.Error("deleted user should not be authorized")
	}
}

func TestDeleteNotFound(t *testing.T) {
	s := testUserStore(t)

	err := s.Delete(999)
	if err == nil {
		t.Fatal("expected error for missing user")
	}
}

func TestIsAdmin(t *testing.T) {
	s := testUserStore(t)

	if !s.IsAdmin("admin@example.com") {
		t.Error("should be admin")
	}
	if !s.IsAdmin("Admin@Example.COM") {
		t.Error("admin check should be case-insensitive")
	}
	if s.IsAdmin("bob@example.com") {
		t.Error("bob should not be admin")
	}
}

func TestAllEmails(t *testing.T) {
	s := testUserStore(t)

	if _, err := s.Add("bob@example.com", "Bob"); err != nil {
		t.Fatalf("add: %v", err)
	}

	emails, err := s.AllEmails()
	if err != nil {
		t.Fatalf("all emails: %v", err)
	}

	// Should include admin + bob
	if len(emails) != 2 {
		t.Fatalf("got %d emails, want 2", len(emails))
	}

	found := map[string]bool{}
	for _, e := range emails {
		found[e] = true
	}
	if !found["admin@example.com"] {
		t.Error("missing admin")
	}
	if !found["bob@example.com"] {
		t.Error("missing bob")
	}
}

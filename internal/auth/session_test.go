package auth

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/evcraddock/house-finder/internal/db"
)

func TestSessionCreateAndValidate(t *testing.T) {
	store := testSessionStore(t)

	w := httptest.NewRecorder()
	if err := store.Create(w, "admin@example.com"); err != nil {
		t.Fatalf("create: %v", err)
	}

	// Extract cookie from response
	cookies := w.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("expected session cookie")
	}

	var sessionCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == cookieName {
			sessionCookie = c
			break
		}
	}
	if sessionCookie == nil {
		t.Fatalf("expected cookie named %q", cookieName)
	}

	// Validate with the cookie
	r := httptest.NewRequest("GET", "/", nil)
	r.AddCookie(sessionCookie)

	email, err := store.Validate(r)
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	if email != "admin@example.com" {
		t.Errorf("email = %q, want %q", email, "admin@example.com")
	}
}

func TestSessionValidateNoCookie(t *testing.T) {
	store := testSessionStore(t)

	r := httptest.NewRequest("GET", "/", nil)
	if _, err := store.Validate(r); err == nil {
		t.Fatal("expected error with no cookie")
	}
}

func TestSessionValidateInvalidCookie(t *testing.T) {
	store := testSessionStore(t)

	r := httptest.NewRequest("GET", "/", nil)
	r.AddCookie(&http.Cookie{Name: cookieName, Value: "bogus-session-id"})

	if _, err := store.Validate(r); err == nil {
		t.Fatal("expected error for invalid session")
	}
}

func TestSessionDestroy(t *testing.T) {
	store := testSessionStore(t)

	// Create session
	w := httptest.NewRecorder()
	if err := store.Create(w, "admin@example.com"); err != nil {
		t.Fatalf("create: %v", err)
	}

	cookies := w.Result().Cookies()
	var sessionCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == cookieName {
			sessionCookie = c
			break
		}
	}

	// Destroy it
	r := httptest.NewRequest("GET", "/", nil)
	r.AddCookie(sessionCookie)
	w2 := httptest.NewRecorder()

	if err := store.Destroy(w2, r); err != nil {
		t.Fatalf("destroy: %v", err)
	}

	// Validate should fail now
	r2 := httptest.NewRequest("GET", "/", nil)
	r2.AddCookie(sessionCookie)
	if _, err := store.Validate(r2); err == nil {
		t.Fatal("expected error after destroy")
	}
}

func testSessionStore(t *testing.T) *SessionStore {
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
	return NewSessionStore(d)
}

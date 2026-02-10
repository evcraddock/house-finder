package auth

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/evcraddock/house-finder/internal/db"
)

func TestRequireAuthRedirectsUnauthenticated(t *testing.T) {
	store := testSessionStore(t)

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := RequireAuth(store, inner)

	r := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusSeeOther {
		t.Errorf("status = %d, want %d", w.Code, http.StatusSeeOther)
	}
	if w.Header().Get("Location") != "/login" {
		t.Errorf("location = %q, want /login", w.Header().Get("Location"))
	}
}

func TestRequireAuthAllowsAuthenticated(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.db")
	d, err := db.Open(path)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() {
		if err := d.Close(); err != nil {
			t.Errorf("close: %v", err)
		}
	})
	store := NewSessionStore(d, false)

	// Create a session
	w := httptest.NewRecorder()
	if err := store.Create(w, "admin@example.com"); err != nil {
		t.Fatalf("create session: %v", err)
	}
	var sessionCookie *http.Cookie
	for _, c := range w.Result().Cookies() {
		if c.Name == cookieName {
			sessionCookie = c
			break
		}
	}

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := RequireAuth(store, inner)

	r := httptest.NewRequest("GET", "/", nil)
	r.AddCookie(sessionCookie)
	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, r)

	if w2.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w2.Code, http.StatusOK)
	}
}

func TestRequireAuthAllowsPublicPaths(t *testing.T) {
	store := testSessionStore(t)

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := RequireAuth(store, inner)

	publicPaths := []string{"/health", "/login", "/auth/login", "/auth/verify", "/auth/logout", "/static/style.css", "/cli/auth", "/cli/auth/verify", "/cli/auth/complete"}
	for _, path := range publicPaths {
		t.Run(path, func(t *testing.T) {
			r := httptest.NewRequest("GET", path, nil)
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, r)

			if w.Code != http.StatusOK {
				t.Errorf("status = %d, want %d for %s", w.Code, http.StatusOK, path)
			}
		})
	}
}

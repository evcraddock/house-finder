package web

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"

	"github.com/evcraddock/house-finder/internal/auth"
	"github.com/evcraddock/house-finder/internal/db"
)

func TestLoginPageRendersForm(t *testing.T) {
	srv := testServerWithAuth(t, "admin@example.com")

	r := httptest.NewRequest("GET", "/login", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	body := w.Body.String()
	if !strings.Contains(body, "Sign In") {
		t.Error("expected login page to contain 'Sign In'")
	}
	if !strings.Contains(body, `action="/auth/login"`) {
		t.Error("expected login form action")
	}
}

func TestLoginSubmitValidEmail(t *testing.T) {
	srv := testServerWithAuth(t, "admin@example.com")

	form := url.Values{"email": {"admin@example.com"}}
	r := httptest.NewRequest("POST", "/auth/login", strings.NewReader(form.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	body := w.Body.String()
	if !strings.Contains(body, "login link has been sent") {
		t.Error("expected success message")
	}
}

func TestLoginSubmitUnknownEmail(t *testing.T) {
	srv := testServerWithAuth(t, "admin@example.com")

	form := url.Values{"email": {"stranger@example.com"}}
	r := httptest.NewRequest("POST", "/auth/login", strings.NewReader(form.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	// Same message as valid email — no info leak
	body := w.Body.String()
	if !strings.Contains(body, "login link has been sent") {
		t.Error("expected same success message for unknown email")
	}
}

func TestLoginSubmitEmptyEmail(t *testing.T) {
	srv := testServerWithAuth(t, "admin@example.com")

	form := url.Values{"email": {""}}
	r := httptest.NewRequest("POST", "/auth/login", strings.NewReader(form.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	body := w.Body.String()
	if !strings.Contains(body, "Email is required") {
		t.Error("expected error for empty email")
	}
}

func TestLoginSubmitMethodNotAllowed(t *testing.T) {
	srv := testServerWithAuth(t, "admin@example.com")

	r := httptest.NewRequest("GET", "/auth/login", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, r)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestVerifyValidToken(t *testing.T) {
	srv, d := testServerWithDBAndAuth(t, "admin@example.com")

	// Create a token directly
	tokens := auth.NewTokenStore(d)
	token, err := tokens.Create("admin@example.com")
	if err != nil {
		t.Fatalf("create token: %v", err)
	}

	r := httptest.NewRequest("GET", "/auth/verify?token="+token, nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, r)

	if w.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusSeeOther)
	}
	if w.Header().Get("Location") != "/" {
		t.Errorf("location = %q, want /", w.Header().Get("Location"))
	}

	// Should have a session cookie
	var found bool
	for _, c := range w.Result().Cookies() {
		if c.Name == "hf_session" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected session cookie after verify")
	}
}

func TestVerifyInvalidToken(t *testing.T) {
	srv := testServerWithAuth(t, "admin@example.com")

	r := httptest.NewRequest("GET", "/auth/verify?token=bogus", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	body := w.Body.String()
	if !strings.Contains(body, "Invalid or expired") {
		t.Error("expected error for invalid token")
	}
}

func TestVerifyEmptyToken(t *testing.T) {
	srv := testServerWithAuth(t, "admin@example.com")

	r := httptest.NewRequest("GET", "/auth/verify", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	body := w.Body.String()
	if !strings.Contains(body, "Invalid login link") {
		t.Error("expected error for empty token")
	}
}

func TestLogout(t *testing.T) {
	srv := testServerWithAuth(t, "admin@example.com")

	r := httptest.NewRequest("GET", "/auth/logout", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, r)

	if w.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusSeeOther)
	}
	if w.Header().Get("Location") != "/login" {
		t.Errorf("location = %q, want /login", w.Header().Get("Location"))
	}
}

func TestProtectedRouteRedirectsToLogin(t *testing.T) {
	srv := testServerWithAuth(t, "admin@example.com")

	r := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, r)

	if w.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusSeeOther)
	}
	if w.Header().Get("Location") != "/login" {
		t.Errorf("location = %q, want /login", w.Header().Get("Location"))
	}
}

func TestProtectedRouteAllowsAuthenticated(t *testing.T) {
	srv, d := testServerWithDBAndAuth(t, "admin@example.com")

	// Create a session
	sessions := auth.NewSessionStore(d, false)
	w := httptest.NewRecorder()
	if err := sessions.Create(w, "admin@example.com"); err != nil {
		t.Fatalf("create session: %v", err)
	}
	var sessionCookie *http.Cookie
	for _, c := range w.Result().Cookies() {
		if c.Name == "hf_session" {
			sessionCookie = c
			break
		}
	}

	r := httptest.NewRequest("GET", "/", nil)
	r.AddCookie(sessionCookie)
	w2 := httptest.NewRecorder()
	srv.ServeHTTP(w2, r)

	if w2.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w2.Code, http.StatusOK)
	}
}

// test helpers

func testServerWithAuth(t *testing.T, adminEmail string) *Server {
	t.Helper()
	srv, _ := testServerWithDBAndAuth(t, adminEmail)
	return srv
}

func testServerWithDBAndAuth(t *testing.T, adminEmail string) (*Server, *sql.DB) {
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

	cfg := auth.Config{
		AdminEmail: adminEmail,
		DevMode:    true,
		BaseURL:    "http://localhost:8080",
	}
	srv, err := NewServer(d, cfg)
	if err != nil {
		t.Fatalf("new server: %v", err)
	}

	return srv, d
}

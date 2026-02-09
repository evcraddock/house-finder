package web

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/evcraddock/house-finder/internal/auth"
)

func TestAuthorizedUserCanLogin(t *testing.T) {
	srv, d := testServerWithDBAndAuth(t, "admin@example.com")

	// Add an authorized user
	users := auth.NewUserStore(d, "admin@example.com")
	if _, err := users.Add("bob@example.com", "Bob", "", false); err != nil {
		t.Fatalf("add user: %v", err)
	}

	// Submit login form with bob's email
	form := url.Values{"email": {"bob@example.com"}}
	r := httptest.NewRequest("POST", "/auth/login", strings.NewReader(form.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}
	// Should show success message (magic link sent)
	body := w.Body.String()
	if !strings.Contains(body, "login link has been sent") {
		t.Error("expected success message for authorized user")
	}
}

func TestUnauthorizedUserCannotLogin(t *testing.T) {
	srv, _ := testServerWithDBAndAuth(t, "admin@example.com")

	// Submit login form with unknown email — should still show generic message
	form := url.Values{"email": {"nobody@example.com"}}
	r := httptest.NewRequest("POST", "/auth/login", strings.NewReader(form.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}
	// Should show same generic message (no email enumeration)
	body := w.Body.String()
	if !strings.Contains(body, "login link has been sent") {
		t.Error("expected generic message even for unauthorized user")
	}
}

func TestAuthorizedUserMagicLinkVerify(t *testing.T) {
	srv, d := testServerWithDBAndAuth(t, "admin@example.com")

	// Add an authorized user
	users := auth.NewUserStore(d, "admin@example.com")
	if _, err := users.Add("bob@example.com", "Bob", "", false); err != nil {
		t.Fatalf("add user: %v", err)
	}

	// Create a token for bob
	tokens := auth.NewTokenStore(d)
	token, err := tokens.Create("bob@example.com")
	if err != nil {
		t.Fatalf("create token: %v", err)
	}

	// Verify — should create session and redirect
	r := httptest.NewRequest("GET", "/auth/verify?token="+token, nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, r)

	if w.Code != http.StatusSeeOther {
		t.Fatalf("verify status = %d, want %d", w.Code, http.StatusSeeOther)
	}

	// Should have session cookie
	var found bool
	for _, c := range w.Result().Cookies() {
		if c.Name == "hf_session" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected session cookie")
	}
}

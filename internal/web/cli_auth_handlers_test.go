package web

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/evcraddock/house-finder/internal/auth"
)

func TestCLIAuthPageRendersForm(t *testing.T) {
	srv := testServerWithAuth(t, "admin@example.com")

	r := httptest.NewRequest("GET", "/cli/auth", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	body := w.Body.String()
	if !strings.Contains(body, "CLI Login") {
		t.Error("expected CLI Login title")
	}
	if !strings.Contains(body, `action="/cli/auth"`) {
		t.Error("expected form action /cli/auth")
	}
}

func TestCLIAuthSubmitEmail(t *testing.T) {
	srv := testServerWithAuth(t, "admin@example.com")

	form := url.Values{"email": {"admin@example.com"}}
	r := httptest.NewRequest("POST", "/cli/auth", strings.NewReader(form.Encode()))
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

func TestCLIAuthVerifyAndComplete(t *testing.T) {
	srv, d := testServerWithDBAndAuth(t, "admin@example.com")

	// Create a token
	tokens := auth.NewTokenStore(d)
	token, err := tokens.Create("admin@example.com")
	if err != nil {
		t.Fatalf("create token: %v", err)
	}

	// Verify — should redirect to /cli/auth/complete
	r := httptest.NewRequest("GET", "/cli/auth/verify?token="+token, nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, r)

	if w.Code != http.StatusSeeOther {
		t.Fatalf("verify status = %d, want %d", w.Code, http.StatusSeeOther)
	}
	if w.Header().Get("Location") != "/cli/auth/complete" {
		t.Errorf("location = %q, want /cli/auth/complete", w.Header().Get("Location"))
	}

	// Extract session cookie
	var sessionCookie *http.Cookie
	for _, c := range w.Result().Cookies() {
		if c.Name == "hf_session" {
			sessionCookie = c
			break
		}
	}
	if sessionCookie == nil {
		t.Fatal("expected session cookie after verify")
	}

	// Complete — should show API key
	r2 := httptest.NewRequest("GET", "/cli/auth/complete", nil)
	r2.AddCookie(sessionCookie)
	w2 := httptest.NewRecorder()
	srv.ServeHTTP(w2, r2)

	if w2.Code != http.StatusOK {
		t.Fatalf("complete status = %d, want %d", w2.Code, http.StatusOK)
	}
	body := w2.Body.String()
	if !strings.Contains(body, "hf_") {
		t.Error("expected API key starting with hf_ in response")
	}
	if !strings.Contains(body, "Copy to Clipboard") {
		t.Error("expected copy button")
	}
	if !strings.Contains(body, "paste it into your terminal") {
		t.Error("expected paste instruction")
	}
}

func TestCLIAuthCompleteRedirectsWithoutSession(t *testing.T) {
	srv := testServerWithAuth(t, "admin@example.com")

	r := httptest.NewRequest("GET", "/cli/auth/complete", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, r)

	if w.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusSeeOther)
	}
	if w.Header().Get("Location") != "/cli/auth" {
		t.Errorf("location = %q, want /cli/auth", w.Header().Get("Location"))
	}
}

func TestCLIAuthVerifyInvalidToken(t *testing.T) {
	srv := testServerWithAuth(t, "admin@example.com")

	r := httptest.NewRequest("GET", "/cli/auth/verify?token=bogus", nil)
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

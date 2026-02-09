package web

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/evcraddock/house-finder/internal/auth"
)

func TestAPIListUsersAdminOnly(t *testing.T) {
	srv, d := testServerWithDBAndAuth(t, "admin@example.com")

	// Create a session for admin
	sessions := auth.NewSessionStore(d, false)
	w := httptest.NewRecorder()
	if err := sessions.Create(w, "admin@example.com"); err != nil {
		t.Fatalf("create session: %v", err)
	}
	cookie := w.Result().Cookies()[0]

	// List users as admin
	r := httptest.NewRequest("GET", "/api/users", nil)
	r.AddCookie(cookie)
	w2 := httptest.NewRecorder()
	srv.ServeHTTP(w2, r)

	if w2.Code != http.StatusOK {
		t.Fatalf("admin list status = %d, want %d; body: %s", w2.Code, http.StatusOK, w2.Body.String())
	}
}

func TestAPIListUsersNonAdmin(t *testing.T) {
	srv, d := testServerWithDBAndAuth(t, "admin@example.com")

	// Add a non-admin user
	users := auth.NewUserStore(d, "admin@example.com")
	if _, err := users.Add("bob@example.com", "Bob", "", false); err != nil {
		t.Fatalf("add user: %v", err)
	}

	// Create a session for bob
	sessions := auth.NewSessionStore(d, false)
	w := httptest.NewRecorder()
	if err := sessions.Create(w, "bob@example.com"); err != nil {
		t.Fatalf("create session: %v", err)
	}
	cookie := w.Result().Cookies()[0]

	// List users as bob — should be forbidden
	r := httptest.NewRequest("GET", "/api/users", nil)
	r.AddCookie(cookie)
	w2 := httptest.NewRecorder()
	srv.ServeHTTP(w2, r)

	if w2.Code != http.StatusForbidden {
		t.Fatalf("non-admin list status = %d, want %d", w2.Code, http.StatusForbidden)
	}
}

func TestAPIAddAndDeleteUser(t *testing.T) {
	srv, d := testServerWithDBAndAuth(t, "admin@example.com")

	// Create admin session
	sessions := auth.NewSessionStore(d, false)
	w := httptest.NewRecorder()
	if err := sessions.Create(w, "admin@example.com"); err != nil {
		t.Fatalf("create session: %v", err)
	}
	cookie := w.Result().Cookies()[0]

	// Add user
	body := `{"email": "alice@example.com", "name": "Alice"}`
	r := httptest.NewRequest("POST", "/api/users", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r.AddCookie(cookie)
	w2 := httptest.NewRecorder()
	srv.ServeHTTP(w2, r)

	if w2.Code != http.StatusCreated {
		t.Fatalf("add status = %d, want %d; body: %s", w2.Code, http.StatusCreated, w2.Body.String())
	}

	var user auth.User
	if err := json.NewDecoder(w2.Body).Decode(&user); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if user.Email != "alice@example.com" {
		t.Errorf("email = %q", user.Email)
	}

	// Delete user
	r2 := httptest.NewRequest("DELETE", fmt.Sprintf("/api/users/%d", user.ID), nil)
	r2.AddCookie(cookie)
	w3 := httptest.NewRecorder()
	srv.ServeHTTP(w3, r2)

	if w3.Code != http.StatusOK {
		t.Fatalf("delete status = %d, want %d", w3.Code, http.StatusOK)
	}
}

func TestAPIAddUserDuplicate(t *testing.T) {
	srv, d := testServerWithDBAndAuth(t, "admin@example.com")

	sessions := auth.NewSessionStore(d, false)
	w := httptest.NewRecorder()
	if err := sessions.Create(w, "admin@example.com"); err != nil {
		t.Fatalf("create session: %v", err)
	}
	cookie := w.Result().Cookies()[0]

	body := `{"email": "alice@example.com", "name": "Alice"}`

	// Add once
	r := httptest.NewRequest("POST", "/api/users", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r.AddCookie(cookie)
	w2 := httptest.NewRecorder()
	srv.ServeHTTP(w2, r)
	if w2.Code != http.StatusCreated {
		t.Fatalf("first add: %d", w2.Code)
	}

	// Add again — should conflict
	r2 := httptest.NewRequest("POST", "/api/users", strings.NewReader(body))
	r2.Header.Set("Content-Type", "application/json")
	r2.AddCookie(cookie)
	w3 := httptest.NewRecorder()
	srv.ServeHTTP(w3, r2)
	if w3.Code != http.StatusConflict {
		t.Fatalf("duplicate add status = %d, want %d", w3.Code, http.StatusConflict)
	}
}

func TestAPIAddUserWithFields(t *testing.T) {
	srv, d := testServerWithDBAndAuth(t, "admin@example.com")

	sessions := auth.NewSessionStore(d, false)
	w := httptest.NewRecorder()
	if err := sessions.Create(w, "admin@example.com"); err != nil {
		t.Fatalf("create session: %v", err)
	}
	cookie := w.Result().Cookies()[0]

	body := `{"email": "realtor@example.com", "name": "Jane", "phone": "555-1234", "is_realtor": true}`
	r := httptest.NewRequest("POST", "/api/users", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r.AddCookie(cookie)
	w2 := httptest.NewRecorder()
	srv.ServeHTTP(w2, r)

	if w2.Code != http.StatusCreated {
		t.Fatalf("add status = %d; body: %s", w2.Code, w2.Body.String())
	}

	var user auth.User
	if err := json.NewDecoder(w2.Body).Decode(&user); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if user.Phone != "555-1234" {
		t.Errorf("phone = %q, want %q", user.Phone, "555-1234")
	}
	if !user.IsRealtor {
		t.Error("expected is_realtor = true")
	}
}

func TestAPIUpdateUser(t *testing.T) {
	srv, d := testServerWithDBAndAuth(t, "admin@example.com")

	sessions := auth.NewSessionStore(d, false)
	w := httptest.NewRecorder()
	if err := sessions.Create(w, "admin@example.com"); err != nil {
		t.Fatalf("create session: %v", err)
	}
	cookie := w.Result().Cookies()[0]

	// Add user first
	body := `{"email": "bob@example.com", "name": "Bob"}`
	r := httptest.NewRequest("POST", "/api/users", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r.AddCookie(cookie)
	w2 := httptest.NewRecorder()
	srv.ServeHTTP(w2, r)

	var user auth.User
	if err := json.NewDecoder(w2.Body).Decode(&user); err != nil {
		t.Fatalf("decode: %v", err)
	}

	// Update user
	updateBody := `{"name": "Robert", "phone": "555-9999", "is_realtor": true}`
	r2 := httptest.NewRequest("PUT", fmt.Sprintf("/api/users/%d", user.ID), strings.NewReader(updateBody))
	r2.Header.Set("Content-Type", "application/json")
	r2.AddCookie(cookie)
	w3 := httptest.NewRecorder()
	srv.ServeHTTP(w3, r2)

	if w3.Code != http.StatusOK {
		t.Fatalf("update status = %d; body: %s", w3.Code, w3.Body.String())
	}

	var updated auth.User
	if err := json.NewDecoder(w3.Body).Decode(&updated); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if updated.Name != "Robert" {
		t.Errorf("name = %q, want %q", updated.Name, "Robert")
	}
	if updated.Phone != "555-9999" {
		t.Errorf("phone = %q, want %q", updated.Phone, "555-9999")
	}
	if !updated.IsRealtor {
		t.Error("expected is_realtor = true")
	}
}

func TestAPIUsersNoAuth(t *testing.T) {
	srv := testServerWithAuth(t, "admin@example.com")

	r := httptest.NewRequest("GET", "/api/users", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("no-auth status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

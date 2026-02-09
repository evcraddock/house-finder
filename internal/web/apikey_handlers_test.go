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

func TestCreateAPIKey(t *testing.T) {
	srv, d := testServerWithDBAndAuth(t, "admin@example.com")
	cookie := createTestSession(t, d, "admin@example.com")

	body := `{"name":"Test CLI"}`
	r := httptest.NewRequest("POST", "/api/keys", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r.AddCookie(cookie)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, r)

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusCreated, w.Body.String())
	}

	var resp apiKeyCreateResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Key == "" {
		t.Error("expected raw key in response")
	}
	if resp.APIKeyResponse.Name != "Test CLI" {
		t.Errorf("name = %q, want %q", resp.APIKeyResponse.Name, "Test CLI")
	}
}

func TestListAPIKeys(t *testing.T) {
	srv, d := testServerWithDBAndAuth(t, "admin@example.com")
	cookie := createTestSession(t, d, "admin@example.com")

	// Create a key first
	store := auth.NewAPIKeyStore(d)
	if _, _, err := store.Create("Key 1"); err != nil {
		t.Fatalf("create: %v", err)
	}

	r := httptest.NewRequest("GET", "/api/keys", nil)
	r.AddCookie(cookie)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var keys []apiKeyResponse
	if err := json.NewDecoder(w.Body).Decode(&keys); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(keys) != 1 {
		t.Fatalf("got %d keys, want 1", len(keys))
	}
	if keys[0].Name != "Key 1" {
		t.Errorf("name = %q, want %q", keys[0].Name, "Key 1")
	}
}

func TestDeleteAPIKey(t *testing.T) {
	srv, d := testServerWithDBAndAuth(t, "admin@example.com")
	cookie := createTestSession(t, d, "admin@example.com")

	store := auth.NewAPIKeyStore(d)
	_, key, err := store.Create("To Revoke")
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	r := httptest.NewRequest("DELETE", fmt.Sprintf("/api/keys/%d", key.ID), nil)
	r.AddCookie(cookie)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, r)

	if w.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusNoContent)
	}

	// Verify deleted
	keys, err := store.List()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(keys) != 0 {
		t.Errorf("got %d keys after delete, want 0", len(keys))
	}
}

func TestAPIKeysRequireSession(t *testing.T) {
	srv := testServerWithAuth(t, "admin@example.com")

	// No session cookie — should get 401
	r := httptest.NewRequest("GET", "/api/keys", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestBearerTokenAuth(t *testing.T) {
	srv, d := testServerWithDBAndAuth(t, "admin@example.com")

	// Create an API key
	store := auth.NewAPIKeyStore(d)
	rawKey, _, err := store.Create("CLI Key")
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	// Access a future API endpoint with bearer token (use a non-management /api/ path)
	// For now, test that an unknown /api/ path with valid bearer returns 404 (not 401)
	r := httptest.NewRequest("GET", "/api/properties", nil)
	r.Header.Set("Authorization", "Bearer "+rawKey)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, r)

	// Should not be 401 — bearer auth passed, handler just doesn't exist yet
	if w.Code == http.StatusUnauthorized {
		t.Error("expected bearer auth to pass, got 401")
	}
}

func TestBearerTokenInvalid(t *testing.T) {
	srv := testServerWithAuth(t, "admin@example.com")

	r := httptest.NewRequest("GET", "/api/properties", nil)
	r.Header.Set("Authorization", "Bearer hf_invalidkey")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestBearerTokenMissing(t *testing.T) {
	srv := testServerWithAuth(t, "admin@example.com")

	r := httptest.NewRequest("GET", "/api/properties", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

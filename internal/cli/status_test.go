package cli

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestStatusShowsServerAndKey(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("HF_API_KEY", "hf_testapikey1234567890")
	t.Setenv("HF_SERVER_URL", "http://localhost:9999")

	// runStatus prints to stdout — just verify it doesn't panic
	// The real test is that apiKey[:8] doesn't panic with a valid key
	if err := runStatus(); err != nil {
		t.Fatalf("status: %v", err)
	}
}

func TestStatusShortAPIKey(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("HF_API_KEY", "hf_ab")
	t.Setenv("HF_SERVER_URL", "http://localhost:9999")

	// Should not panic with a short key
	if err := runStatus(); err != nil {
		t.Fatalf("status with short key: %v", err)
	}
}

func TestStatusNoAPIKey(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("HF_API_KEY", "")
	t.Setenv("HF_SERVER_URL", "http://localhost:9999")

	if err := runStatus(); err != nil {
		t.Fatalf("status with no key: %v", err)
	}
}

func TestStatusWithServer(t *testing.T) {
	// Set up a test server that returns 200 for valid bearer
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth == "Bearer hf_validkey1234567890abc" {
			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode([]string{}); err != nil {
				http.Error(w, "encode error", http.StatusInternalServerError)
			}
		} else {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
		}
	}))
	defer srv.Close()

	t.Setenv("HOME", t.TempDir())
	t.Setenv("HF_API_KEY", "hf_validkey1234567890abc")
	t.Setenv("HF_SERVER_URL", srv.URL)

	if err := runStatus(); err != nil {
		t.Fatalf("status: %v", err)
	}
}

func TestStatusWithInvalidKey(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
	}))
	defer srv.Close()

	t.Setenv("HOME", t.TempDir())
	t.Setenv("HF_API_KEY", "hf_badkey1234567890abcde")
	t.Setenv("HF_SERVER_URL", srv.URL)

	// Should not return error — just prints status
	if err := runStatus(); err != nil {
		t.Fatalf("status: %v", err)
	}
}

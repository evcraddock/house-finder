package web

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"

	"github.com/evcraddock/house-finder/internal/auth"
	"github.com/evcraddock/house-finder/internal/db"
)

func TestHealthEndpoint(t *testing.T) {
	srv := testServer(t)

	r := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("content-type = %q, want application/json", ct)
	}
	if !strings.Contains(w.Body.String(), `"status":"ok"`) {
		t.Errorf("body = %q, want status ok", w.Body.String())
	}
}

func TestHandleListEmpty(t *testing.T) {
	srv := testServer(t)

	r := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if !strings.Contains(w.Body.String(), "No properties in this tab") {
		t.Error("expected empty state message")
	}
}

func TestHandleListHasAddForm(t *testing.T) {
	srv := testServer(t)

	r := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, r)

	body := w.Body.String()
	if !strings.Contains(body, "add-property-form") {
		t.Error("expected add property form")
	}
	if !strings.Contains(body, "Enter address or MLS ID") {
		t.Error("expected address input placeholder")
	}
}

func TestHandleListWithProperties(t *testing.T) {
	srv, d := testServerWithDB(t)
	insertTestProperty(t, d, "123 Main St", "M-LIST-1")

	r := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if !strings.Contains(w.Body.String(), "123 Main St") {
		t.Error("expected property address in response")
	}
}

func TestHandleDetail(t *testing.T) {
	srv, d := testServerWithDB(t)
	insertTestProperty(t, d, "456 Oak Ave", "M-DETAIL-1")

	r := httptest.NewRequest("GET", "/property/1", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	body := w.Body.String()
	if !strings.Contains(body, "456 Oak Ave") {
		t.Error("expected property address in response")
	}
	if !strings.Contains(body, "Add Comment") {
		t.Error("expected comment form")
	}
}

func TestHandleDetailNotFound(t *testing.T) {
	srv := testServer(t)

	r := httptest.NewRequest("GET", "/property/9999", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, r)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestHandleCommentPost(t *testing.T) {
	srv, d := testServerWithDB(t)
	insertTestProperty(t, d, "789 Pine St", "M-COMMENT-1")

	form := url.Values{"text": {"Great backyard"}}
	r := httptest.NewRequest("POST", "/property/1/comment", strings.NewReader(form.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, r)

	// Non-HTMX POST should redirect
	if w.Code != http.StatusSeeOther {
		t.Errorf("status = %d, want %d", w.Code, http.StatusSeeOther)
	}
}

func TestHandleCommentPostHTMX(t *testing.T) {
	srv, d := testServerWithDB(t)
	insertTestProperty(t, d, "789 Pine St", "M-COMMENT-2")

	form := url.Values{"text": {"HTMX comment"}}
	r := httptest.NewRequest("POST", "/property/1/comment", strings.NewReader(form.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.Header.Set("HX-Request", "true")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if !strings.Contains(w.Body.String(), "HTMX comment") {
		t.Error("expected comment text in partial response")
	}
}

func TestHandleCommentPostEmptyText(t *testing.T) {
	srv, d := testServerWithDB(t)
	insertTestProperty(t, d, "789 Pine St", "M-COMMENT-3")

	form := url.Values{"text": {""}}
	r := httptest.NewRequest("POST", "/property/1/comment", strings.NewReader(form.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandleRatePost(t *testing.T) {
	srv, d := testServerWithDB(t)
	insertTestProperty(t, d, "321 Rate Dr", "M-RATE-1")

	form := url.Values{"rating": {"3"}}
	r := httptest.NewRequest("POST", "/property/1/rate", strings.NewReader(form.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, r)

	if w.Code != http.StatusSeeOther {
		t.Errorf("status = %d, want %d", w.Code, http.StatusSeeOther)
	}
}

func TestHandleRatePostHTMX(t *testing.T) {
	srv, d := testServerWithDB(t)
	insertTestProperty(t, d, "321 Rate Dr", "M-RATE-2")

	form := url.Values{"rating": {"4"}}
	r := httptest.NewRequest("POST", "/property/1/rate", strings.NewReader(form.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.Header.Set("HX-Request", "true")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if !strings.Contains(w.Body.String(), "active") {
		t.Error("expected active rating button in partial response")
	}
}

func TestHandleRatePostInvalid(t *testing.T) {
	srv, d := testServerWithDB(t)
	insertTestProperty(t, d, "321 Rate Dr", "M-RATE-3")

	tests := []struct {
		name   string
		rating string
	}{
		{"zero", "0"},
		{"five", "5"},
		{"text", "abc"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			form := url.Values{"rating": {tt.rating}}
			r := httptest.NewRequest("POST", "/property/1/rate", strings.NewReader(form.Encode()))
			r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			w := httptest.NewRecorder()
			srv.ServeHTTP(w, r)

			if w.Code != http.StatusBadRequest {
				t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
			}
		})
	}
}

func TestHandleCommentGetNotAllowed(t *testing.T) {
	srv, d := testServerWithDB(t)
	insertTestProperty(t, d, "789 Pine St", "M-METHOD-1")

	r := httptest.NewRequest("GET", "/property/1/comment", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, r)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestStaticFiles(t *testing.T) {
	srv := testServer(t)

	r := httptest.NewRequest("GET", "/static/style.css", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if !strings.Contains(w.Header().Get("Content-Type"), "css") {
		t.Errorf("content-type = %q, want css", w.Header().Get("Content-Type"))
	}
}

// test helpers

func testServer(t *testing.T) *Server {
	t.Helper()
	srv, _ := testServerWithDB(t)
	return srv
}

func testServerWithDB(t *testing.T) (*Server, *sql.DB) {
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

	// No admin email = auth disabled for tests
	cfg := auth.Config{}
	srv, err := NewServer(d, cfg)
	if err != nil {
		t.Fatalf("new server: %v", err)
	}

	return srv, d
}

func insertTestProperty(t *testing.T, d *sql.DB, address, mprID string) {
	t.Helper()
	raw := json.RawMessage(`{"data": {"list_price": 250000, "status": "for_sale", "description": {"beds": 3, "baths": 2, "sqft": 1500}}}`)
	if _, err := d.Exec(
		`INSERT INTO properties (address, mpr_id, realtor_url, price, bedrooms, bathrooms, sqft, raw_json) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		address, mprID, "https://example.com", 250000, 3.0, 2.0, 1500, string(raw),
	); err != nil {
		t.Fatalf("insert test property: %v", err)
	}
}

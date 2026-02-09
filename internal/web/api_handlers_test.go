package web

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/evcraddock/house-finder/internal/auth"
	"github.com/evcraddock/house-finder/internal/comment"
	"github.com/evcraddock/house-finder/internal/db"
	"github.com/evcraddock/house-finder/internal/property"
)

// testAPIServerWithDB creates a test server and returns the server, db, and a valid bearer token.
func testAPIServerWithDB(t *testing.T) (*Server, *sql.DB, string) {
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

	cfg := auth.Config{
		AdminEmail: "admin@example.com",
		DevMode:    true,
		BaseURL:    "http://localhost:8080",
	}
	srv, err := NewServer(d, cfg)
	if err != nil {
		t.Fatalf("new server: %v", err)
	}

	// Create an API key for testing
	rawKey, _, err := srv.apiKeys.Create("test", "admin@example.com")
	if err != nil {
		t.Fatalf("create api key: %v", err)
	}

	return srv, d, rawKey
}

func apiRequest(t *testing.T, srv *Server, method, path, token string, body interface{}) *httptest.ResponseRecorder {
	t.Helper()
	var reqBody *bytes.Buffer
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal body: %v", err)
		}
		reqBody = bytes.NewBuffer(data)
	} else {
		reqBody = &bytes.Buffer{}
	}

	r := httptest.NewRequest(method, path, reqBody)
	if body != nil {
		r.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		r.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, r)
	return w
}

var apiTestPropertyCounter int

func insertAPITestProperty(t *testing.T, d *sql.DB) int64 {
	t.Helper()
	apiTestPropertyCounter++
	repo := property.NewRepository(d)
	p, err := repo.Insert(&property.Property{
		Address:    fmt.Sprintf("123 Test St #%d", apiTestPropertyCounter),
		MprID:      fmt.Sprintf("test-mpr-%d", apiTestPropertyCounter),
		RealtorURL: "https://realtor.com/test",
		RawJSON:    json.RawMessage(`{}`),
	})
	if err != nil {
		t.Fatalf("insert test property: %v", err)
	}
	return p.ID
}

func TestAPIListProperties(t *testing.T) {
	srv, d, token := testAPIServerWithDB(t)
	insertAPITestProperty(t, d)

	w := apiRequest(t, srv, "GET", "/api/properties", token, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var props []*property.Property
	if err := json.NewDecoder(w.Body).Decode(&props); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(props) != 1 {
		t.Errorf("got %d properties, want 1", len(props))
	}
}

func TestAPIListPropertiesEmpty(t *testing.T) {
	srv, _, token := testAPIServerWithDB(t)

	w := apiRequest(t, srv, "GET", "/api/properties", token, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var props []*property.Property
	if err := json.NewDecoder(w.Body).Decode(&props); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(props) != 0 {
		t.Errorf("got %d properties, want 0", len(props))
	}
}

func TestAPIListPropertiesRequiresAuth(t *testing.T) {
	srv, _, _ := testAPIServerWithDB(t)

	w := apiRequest(t, srv, "GET", "/api/properties", "", nil)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestAPIGetProperty(t *testing.T) {
	srv, d, token := testAPIServerWithDB(t)
	id := insertAPITestProperty(t, d)

	w := apiRequest(t, srv, "GET", fmt.Sprintf("/api/properties/%d", id), token, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp struct {
		Property *property.Property `json:"property"`
		Comments []*comment.Comment `json:"comments"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Property.ID != id {
		t.Errorf("property ID = %d, want %d", resp.Property.ID, id)
	}
	if resp.Property.Address == "" {
		t.Error("expected non-empty address")
	}
}

func TestAPIGetPropertyNotFound(t *testing.T) {
	srv, _, token := testAPIServerWithDB(t)

	w := apiRequest(t, srv, "GET", "/api/properties/999", token, nil)
	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestAPIDeleteProperty(t *testing.T) {
	srv, d, token := testAPIServerWithDB(t)
	id := insertAPITestProperty(t, d)

	w := apiRequest(t, srv, "DELETE", fmt.Sprintf("/api/properties/%d", id), token, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	// Verify deleted
	w2 := apiRequest(t, srv, "GET", fmt.Sprintf("/api/properties/%d", id), token, nil)
	if w2.Code != http.StatusNotFound {
		t.Errorf("after delete: status = %d, want %d", w2.Code, http.StatusNotFound)
	}
}

func TestAPIRateProperty(t *testing.T) {
	srv, d, token := testAPIServerWithDB(t)
	id := insertAPITestProperty(t, d)

	body := map[string]int{"rating": 3}
	w := apiRequest(t, srv, "POST", fmt.Sprintf("/api/properties/%d/rate", id), token, body)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	// Verify rating
	w2 := apiRequest(t, srv, "GET", fmt.Sprintf("/api/properties/%d", id), token, nil)
	var resp struct {
		Property *property.Property `json:"property"`
	}
	if err := json.NewDecoder(w2.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Property.Rating == nil || *resp.Property.Rating != 3 {
		t.Error("expected rating 3")
	}
}

func TestAPIRatePropertyInvalid(t *testing.T) {
	srv, d, token := testAPIServerWithDB(t)
	id := insertAPITestProperty(t, d)

	body := map[string]int{"rating": 5}
	w := apiRequest(t, srv, "POST", fmt.Sprintf("/api/properties/%d/rate", id), token, body)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestAPIAddComment(t *testing.T) {
	srv, d, token := testAPIServerWithDB(t)
	id := insertAPITestProperty(t, d)

	body := map[string]string{"text": "Nice house"}
	w := apiRequest(t, srv, "POST", fmt.Sprintf("/api/properties/%d/comments", id), token, body)
	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusCreated, w.Body.String())
	}

	var c comment.Comment
	if err := json.NewDecoder(w.Body).Decode(&c); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if c.Text != "Nice house" {
		t.Errorf("text = %q, want %q", c.Text, "Nice house")
	}
}

func TestAPIAddCommentEmpty(t *testing.T) {
	srv, d, token := testAPIServerWithDB(t)
	id := insertAPITestProperty(t, d)

	body := map[string]string{"text": ""}
	w := apiRequest(t, srv, "POST", fmt.Sprintf("/api/properties/%d/comments", id), token, body)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestAPIListComments(t *testing.T) {
	srv, d, token := testAPIServerWithDB(t)
	id := insertAPITestProperty(t, d)

	// Add a comment
	commentRepo := comment.NewRepository(d)
	if _, err := commentRepo.Add(id, "Test comment", "test@example.com"); err != nil {
		t.Fatalf("add comment: %v", err)
	}

	w := apiRequest(t, srv, "GET", fmt.Sprintf("/api/properties/%d/comments", id), token, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var comments []*comment.Comment
	if err := json.NewDecoder(w.Body).Decode(&comments); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(comments) != 1 {
		t.Errorf("got %d comments, want 1", len(comments))
	}
}

func TestAPIAddPropertyWithoutMLSClient(t *testing.T) {
	srv, _, token := testAPIServerWithDB(t)

	body := map[string]string{"address": "123 Main St"}
	w := apiRequest(t, srv, "POST", "/api/properties", token, body)
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d (no MLS client)", w.Code, http.StatusServiceUnavailable)
	}
}

func TestAPIAddPropertyEmptyAddress(t *testing.T) {
	srv, _, token := testAPIServerWithDB(t)

	body := map[string]string{"address": ""}
	w := apiRequest(t, srv, "POST", "/api/properties", token, body)
	// Even without MLS client, empty address should be caught
	// (503 because MLS client check happens first)
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusServiceUnavailable)
	}
}

func TestAPIListPropertiesWithMinRating(t *testing.T) {
	srv, d, token := testAPIServerWithDB(t)
	id := insertAPITestProperty(t, d)

	// Rate it
	repo := property.NewRepository(d)
	if err := repo.UpdateRating(id, 3); err != nil {
		t.Fatalf("update rating: %v", err)
	}

	// Insert another unrated
	insertAPITestProperty(t, d)

	// Filter by min_rating=3
	w := apiRequest(t, srv, "GET", "/api/properties?min_rating=3", token, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var props []*property.Property
	if err := json.NewDecoder(w.Body).Decode(&props); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(props) != 1 {
		t.Errorf("got %d properties with min_rating=3, want 1", len(props))
	}
}

func TestAPIMethodNotAllowed(t *testing.T) {
	srv, _, token := testAPIServerWithDB(t)

	w := apiRequest(t, srv, "PUT", "/api/properties", token, nil)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

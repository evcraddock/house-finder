package client

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/evcraddock/house-finder/internal/comment"
	"github.com/evcraddock/house-finder/internal/property"
)

func TestListProperties(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/properties" {
			t.Errorf("path = %q, want /api/properties", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer testkey" {
			t.Error("expected Bearer testkey")
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode([]*property.Property{{ID: 1, Address: "123 Test"}}); err != nil {
			t.Fatalf("encode: %v", err)
		}
	}))
	defer srv.Close()

	c := New(srv.URL, "testkey")
	props, err := c.ListProperties(ListOptions{})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(props) != 1 {
		t.Fatalf("got %d props, want 1", len(props))
	}
	if props[0].Address != "123 Test" {
		t.Errorf("address = %q", props[0].Address)
	}
}

func TestListPropertiesWithMinRating(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("min_rating") != "3" {
			t.Errorf("min_rating = %q, want 3", r.URL.Query().Get("min_rating"))
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode([]*property.Property{}); err != nil {
			t.Fatalf("encode: %v", err)
		}
	}))
	defer srv.Close()

	c := New(srv.URL, "testkey")
	if _, err := c.ListProperties(ListOptions{MinRating: 3}); err != nil {
		t.Fatalf("list: %v", err)
	}
}

func TestGetProperty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/properties/42" {
			t.Errorf("path = %q", r.URL.Path)
		}
		resp := ShowResponse{
			Property: &property.Property{ID: 42, Address: "42 Elm St"},
			Comments: []*comment.Comment{{ID: 1, Text: "nice"}},
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			t.Fatalf("encode: %v", err)
		}
	}))
	defer srv.Close()

	c := New(srv.URL, "testkey")
	resp, err := c.GetProperty(42)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if resp.Property.ID != 42 {
		t.Errorf("id = %d", resp.Property.ID)
	}
	if len(resp.Comments) != 1 {
		t.Errorf("comments = %d", len(resp.Comments))
	}
}

func TestAddProperty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("method = %s", r.Method)
		}
		var req struct{ Address string }
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if req.Address != "123 Main St" {
			t.Errorf("address = %q", req.Address)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		if err := json.NewEncoder(w).Encode(&property.Property{ID: 1, Address: "123 Main St"}); err != nil {
			t.Fatalf("encode: %v", err)
		}
	}))
	defer srv.Close()

	c := New(srv.URL, "testkey")
	p, err := c.AddProperty("123 Main St")
	if err != nil {
		t.Fatalf("add: %v", err)
	}
	if p.Address != "123 Main St" {
		t.Errorf("address = %q", p.Address)
	}
}

func TestDeleteProperty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Errorf("method = %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]interface{}{"removed": true}); err != nil {
			t.Fatalf("encode: %v", err)
		}
	}))
	defer srv.Close()

	c := New(srv.URL, "testkey")
	if err := c.DeleteProperty(1); err != nil {
		t.Fatalf("delete: %v", err)
	}
}

func TestRateProperty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct{ Rating int }
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if req.Rating != 3 {
			t.Errorf("rating = %d", req.Rating)
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]interface{}{"rating": 3}); err != nil {
			t.Fatalf("encode: %v", err)
		}
	}))
	defer srv.Close()

	c := New(srv.URL, "testkey")
	if err := c.RateProperty(1, 3); err != nil {
		t.Fatalf("rate: %v", err)
	}
}

func TestAddComment(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct{ Text string }
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if req.Text != "great house" {
			t.Errorf("text = %q", req.Text)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		if err := json.NewEncoder(w).Encode(&comment.Comment{ID: 1, Text: "great house"}); err != nil {
			t.Fatalf("encode: %v", err)
		}
	}))
	defer srv.Close()

	c := New(srv.URL, "testkey")
	comm, err := c.AddComment(1, "great house")
	if err != nil {
		t.Fatalf("add comment: %v", err)
	}
	if comm.Text != "great house" {
		t.Errorf("text = %q", comm.Text)
	}
}

func TestListComments(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode([]*comment.Comment{{ID: 1, Text: "hi"}}); err != nil {
			t.Fatalf("encode: %v", err)
		}
	}))
	defer srv.Close()

	c := New(srv.URL, "testkey")
	comments, err := c.ListComments(1)
	if err != nil {
		t.Fatalf("list comments: %v", err)
	}
	if len(comments) != 1 {
		t.Errorf("got %d comments", len(comments))
	}
}

func TestServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		if err := json.NewEncoder(w).Encode(map[string]string{"error": "db exploded"}); err != nil {
			t.Fatalf("encode: %v", err)
		}
	}))
	defer srv.Close()

	c := New(srv.URL, "testkey")
	_, err := c.ListProperties(ListOptions{})
	if err == nil {
		t.Fatal("expected error")
	}
	if err.Error() != "db exploded" {
		t.Errorf("error = %q", err.Error())
	}
}

func TestUnauthorized(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
	}))
	defer srv.Close()

	c := New(srv.URL, "badkey")
	_, err := c.ListProperties(ListOptions{})
	if err == nil {
		t.Fatal("expected error")
	}
}

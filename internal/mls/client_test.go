package mls

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewClient(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		wantErr bool
	}{
		{"valid key", "test-key", false},
		{"empty key", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, err := NewClient(tt.key)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if c == nil {
				t.Fatal("expected client, got nil")
			}
		})
	}
}

func TestLookupMprID(t *testing.T) {
	tests := []struct {
		name       string
		address    string
		response   string
		statusCode int
		wantMprID  string
		wantErr    bool
	}{
		{
			name:    "successful lookup",
			address: "123 Main St",
			response: `{
				"autocomplete": [{"mpr_id": "M1234567890"}]
			}`,
			statusCode: http.StatusOK,
			wantMprID:  "M1234567890",
		},
		{
			name:       "no results",
			address:    "Nonexistent Address",
			response:   `{"autocomplete": []}`,
			statusCode: http.StatusOK,
			wantErr:    true,
		},
		{
			name:       "empty mpr_id",
			address:    "Bad Address",
			response:   `{"autocomplete": [{"mpr_id": ""}]}`,
			statusCode: http.StatusOK,
			wantErr:    true,
		},
		{
			name:       "server error",
			address:    "123 Main St",
			response:   `{}`,
			statusCode: http.StatusInternalServerError,
			wantErr:    true,
		},
		{
			name:       "invalid json",
			address:    "123 Main St",
			response:   `not json`,
			statusCode: http.StatusOK,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Query().Get("input") != tt.address {
					t.Errorf("input = %q, want %q", r.URL.Query().Get("input"), tt.address)
				}
				if r.URL.Query().Get("client_id") != "rdc-home" {
					t.Errorf("client_id = %q, want %q", r.URL.Query().Get("client_id"), "rdc-home")
				}
				w.WriteHeader(tt.statusCode)
				writeResponse(t, w, tt.response)
			}))
			defer server.Close()

			c := testClient(t, server.URL, "", "")

			mprID, err := c.lookupMprID(tt.address)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if mprID != tt.wantMprID {
				t.Errorf("mprID = %q, want %q", mprID, tt.wantMprID)
			}
		})
	}
}

func TestLookupRealtorURL(t *testing.T) {
	tests := []struct {
		name       string
		mprID      string
		response   string
		statusCode int
		wantURL    string
		wantErr    bool
	}{
		{
			name:  "successful lookup",
			mprID: "M1234567890",
			response: `{
				"data": {"home": {"href": "/realestateandhomes-detail/123-Main-St_City_ST_12345_M12345-67890", "property_id": "M1234567890"}}
			}`,
			statusCode: http.StatusOK,
			wantURL:    "/realestateandhomes-detail/123-Main-St_City_ST_12345_M12345-67890",
		},
		{
			name:       "empty href",
			mprID:      "M0000000000",
			response:   `{"data": {"home": {"href": "", "property_id": ""}}}`,
			statusCode: http.StatusOK,
			wantErr:    true,
		},
		{
			name:       "server error",
			mprID:      "M1234567890",
			response:   `{}`,
			statusCode: http.StatusInternalServerError,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "POST" {
					t.Errorf("method = %q, want POST", r.Method)
				}
				if ct := r.Header.Get("Content-Type"); ct != "application/json" {
					t.Errorf("Content-Type = %q, want application/json", ct)
				}

				var req hulkRequest
				if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
					t.Errorf("decode request body: %v", err)
				}

				w.WriteHeader(tt.statusCode)
				writeResponse(t, w, tt.response)
			}))
			defer server.Close()

			c := testClient(t, "", server.URL, "")

			href, err := c.lookupRealtorURL(tt.mprID)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if href != tt.wantURL {
				t.Errorf("href = %q, want %q", href, tt.wantURL)
			}
		})
	}
}

func TestFetchPropertyDetail(t *testing.T) {
	tests := []struct {
		name       string
		realtorURL string
		response   string
		statusCode int
		wantErr    bool
	}{
		{
			name:       "successful fetch",
			realtorURL: "/realestateandhomes-detail/123-Main-St",
			response:   `{"property": {"price": 250000, "beds": 3}}`,
			statusCode: http.StatusOK,
		},
		{
			name:       "server error",
			realtorURL: "/realestateandhomes-detail/123-Main-St",
			response:   `{}`,
			statusCode: http.StatusInternalServerError,
			wantErr:    true,
		},
		{
			name:       "invalid json response",
			realtorURL: "/realestateandhomes-detail/123-Main-St",
			response:   `not json at all`,
			statusCode: http.StatusOK,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get("x-rapidapi-key") != "test-key" {
					t.Errorf("x-rapidapi-key = %q, want %q", r.Header.Get("x-rapidapi-key"), "test-key")
				}
				if r.Header.Get("x-rapidapi-host") != "us-real-estate-listings.p.rapidapi.com" {
					t.Errorf("x-rapidapi-host = %q, want expected value", r.Header.Get("x-rapidapi-host"))
				}
				w.WriteHeader(tt.statusCode)
				writeResponse(t, w, tt.response)
			}))
			defer server.Close()

			c := testClient(t, "", "", server.URL)

			raw, err := c.fetchPropertyDetail(tt.realtorURL)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !json.Valid(raw) {
				t.Error("result is not valid JSON")
			}
		})
	}
}

func TestLookupEmptyAddress(t *testing.T) {
	c, err := NewClient("test-key")
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	_, err = c.Lookup("")
	if err == nil {
		t.Fatal("expected error for empty address, got nil")
	}
}

func TestLookupEndToEnd(t *testing.T) {
	// Mock all three endpoints
	suggestServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeResponse(t, w, `{"autocomplete": [{"mpr_id": "M9999999999"}]}`)
	}))
	defer suggestServer.Close()

	hulkServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeResponse(t, w, `{"data": {"home": {"href": "/detail/123-Test_City_ST_00000_M99999-99999", "property_id": "M9999999999"}}}`)
	}))
	defer hulkServer.Close()

	rapidServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeResponse(t, w, `{"property": {"price": 300000, "beds": 4, "baths": 2}}`)
	}))
	defer rapidServer.Close()

	c := testClient(t, suggestServer.URL, hulkServer.URL, rapidServer.URL)

	result, err := c.Lookup("123 Test St, City, ST 00000")
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}

	if result.MprID != "M9999999999" {
		t.Errorf("MprID = %q, want %q", result.MprID, "M9999999999")
	}
	if result.RealtorURL != "/detail/123-Test_City_ST_00000_M99999-99999" {
		t.Errorf("RealtorURL = %q, want expected value", result.RealtorURL)
	}
	if !json.Valid(result.RawJSON) {
		t.Error("RawJSON is not valid JSON")
	}
}

// writeResponse writes a string to an http.ResponseWriter in tests.
func writeResponse(t *testing.T, w http.ResponseWriter, s string) {
	t.Helper()
	if _, err := fmt.Fprint(w, s); err != nil {
		t.Errorf("write response: %v", err)
	}
}

// testClient creates a client with overridden URLs for testing.
func testClient(t *testing.T, suggestURL, hulkURL, rapidAPIURL string) *Client {
	t.Helper()
	c, err := NewClient("test-key")
	if err != nil {
		t.Fatalf("new client: %v", err)
	}
	if suggestURL != "" {
		c.suggestURL = suggestURL
	}
	if hulkURL != "" {
		c.hulkURL = hulkURL
	}
	if rapidAPIURL != "" {
		c.rapidAPIURL = rapidAPIURL
	}
	return c
}

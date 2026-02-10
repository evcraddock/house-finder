package web

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestAPIEmailDryRun(t *testing.T) {
	srv, d, token := testAPIServerWithDB(t)

	// Insert a property and schedule a future visit
	id := insertAPITestProperty(t, d)
	if _, err := d.Exec(
		"INSERT INTO visits (property_id, visit_date, visit_type) VALUES (?, ?, ?)",
		id, "2099-01-01", "showing",
	); err != nil {
		t.Fatalf("insert visit: %v", err)
	}

	// Add a realtor user
	if _, err := srv.users.Add("realtor@example.com", "Test Realtor", "555-1234", true); err != nil {
		t.Fatalf("add realtor: %v", err)
	}

	body := map[string]interface{}{"dry_run": true}
	w := apiRequest(t, srv, "POST", "/api/email", token, body)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp emailResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if resp.Sent {
		t.Error("expected sent=false for dry run")
	}
	if len(resp.To) < 1 {
		t.Error("expected at least one recipient")
	}
	// Admin is always included
	hasAdmin := false
	for _, r := range resp.To {
		if r == "admin@example.com" {
			hasAdmin = true
		}
	}
	if !hasAdmin {
		t.Errorf("to = %v, expected admin@example.com", resp.To)
	}
	if resp.Subject == "" {
		t.Error("expected non-empty subject")
	}
	if resp.Body == "" {
		t.Error("expected non-empty body")
	}
}

func TestAPIEmailAdminOnly(t *testing.T) {
	srv, d, token := testAPIServerWithDB(t)
	id := insertAPITestProperty(t, d)
	if _, err := d.Exec(
		"INSERT INTO visits (property_id, visit_date, visit_type) VALUES (?, ?, ?)",
		id, "2099-01-01", "showing",
	); err != nil {
		t.Fatalf("insert visit: %v", err)
	}

	// No extra users — admin is still a recipient
	body := map[string]interface{}{"dry_run": true}
	w := apiRequest(t, srv, "POST", "/api/email", token, body)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp emailResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.To) != 1 || resp.To[0] != "admin@example.com" {
		t.Errorf("to = %v, want [admin@example.com]", resp.To)
	}
}

func TestAPIEmailNoProperties(t *testing.T) {
	srv, _, token := testAPIServerWithDB(t)

	if _, err := srv.users.Add("realtor@example.com", "Test Realtor", "", true); err != nil {
		t.Fatalf("add realtor: %v", err)
	}

	body := map[string]interface{}{"dry_run": true}
	w := apiRequest(t, srv, "POST", "/api/email", token, body)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestAPIEmailWithPropertyIDs(t *testing.T) {
	srv, d, token := testAPIServerWithDB(t)

	id := insertAPITestProperty(t, d)
	insertAPITestProperty(t, d) // second property, not included

	if _, err := srv.users.Add("realtor@example.com", "Realtor", "", true); err != nil {
		t.Fatalf("add realtor: %v", err)
	}

	body := map[string]interface{}{
		"property_ids": []int64{id},
		"dry_run":      true,
	}
	w := apiRequest(t, srv, "POST", "/api/email", token, body)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d; body: %s", w.Code, w.Body.String())
	}

	var resp emailResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if resp.Subject != "Properties to visit (1)" {
		t.Errorf("subject = %q, want 1 property", resp.Subject)
	}
}

func TestAPIEmailMethodNotAllowed(t *testing.T) {
	srv, _, token := testAPIServerWithDB(t)

	w := apiRequest(t, srv, "GET", "/api/email", token, nil)
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

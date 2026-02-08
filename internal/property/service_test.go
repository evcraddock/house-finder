package property

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/evcraddock/house-finder/internal/db"
	"github.com/evcraddock/house-finder/internal/mls"
)

func TestServiceAdd(t *testing.T) {
	suggestServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeResp(t, w, `{"autocomplete": [{"mpr_id": "M9999999999"}]}`)
	}))
	defer suggestServer.Close()

	hulkServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeResp(t, w, `{"data": {"home": {"href": "/detail/123-Test_City_ST_00000_M99999-99999", "property_id": "M9999999999"}}}`)
	}))
	defer hulkServer.Close()

	rapidServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeResp(t, w, `{"list_price": 250000, "beds": 3, "baths": 2, "sqft": 1500, "year_built": 2010, "prop_type": "single_family", "prop_status": "active"}`)
	}))
	defer rapidServer.Close()

	svc := testService(t, suggestServer.URL, hulkServer.URL, rapidServer.URL)

	p, err := svc.Add("123 Test St, City, ST 00000")
	if err != nil {
		t.Fatalf("add: %v", err)
	}

	if p.ID == 0 {
		t.Error("expected non-zero ID")
	}
	if p.Address != "123 Test St, City, ST 00000" {
		t.Errorf("address = %q, want %q", p.Address, "123 Test St, City, ST 00000")
	}
	if p.MprID != "M9999999999" {
		t.Errorf("mpr_id = %q, want %q", p.MprID, "M9999999999")
	}
	if p.Price == nil || *p.Price != 250000 {
		t.Errorf("price = %v, want 250000", p.Price)
	}
	if !json.Valid(p.RawJSON) {
		t.Error("raw_json is not valid JSON")
	}
}

func TestServiceAddAPIFailure(t *testing.T) {
	// Suggest returns no results
	suggestServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeResp(t, w, `{"autocomplete": []}`)
	}))
	defer suggestServer.Close()

	svc := testService(t, suggestServer.URL, "", "")

	_, err := svc.Add("Nonexistent Address")
	if err == nil {
		t.Fatal("expected error when API fails")
	}
}

func TestServiceAddNothingSavedOnFailure(t *testing.T) {
	// Suggest works but hulk fails
	suggestServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeResp(t, w, `{"autocomplete": [{"mpr_id": "M1111111111"}]}`)
	}))
	defer suggestServer.Close()

	hulkServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer hulkServer.Close()

	d, repo := testDBAndRepo(t)
	client := testMLSClient(t, suggestServer.URL, hulkServer.URL, "")
	svc := NewService(repo, client)

	_, err := svc.Add("123 Fail St")
	if err == nil {
		t.Fatal("expected error")
	}

	// Verify nothing was saved
	var count int
	if err := d.QueryRow("SELECT COUNT(*) FROM properties").Scan(&count); err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 saved properties, got %d", count)
	}
}

func testService(t *testing.T, suggestURL, hulkURL, rapidAPIURL string) *Service {
	t.Helper()
	_, repo := testDBAndRepo(t)
	client := testMLSClient(t, suggestURL, hulkURL, rapidAPIURL)
	return NewService(repo, client)
}

func testDBAndRepo(t *testing.T) (*sql.DB, *Repository) {
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
	return d, NewRepository(d)
}

func testMLSClient(t *testing.T, suggestURL, hulkURL, rapidAPIURL string) *mls.Client {
	t.Helper()
	client, err := mls.NewClient("test-key")
	if err != nil {
		t.Fatalf("new client: %v", err)
	}
	mls.SetTestURLs(client, suggestURL, hulkURL, rapidAPIURL)
	return client
}

func writeResp(t *testing.T, w http.ResponseWriter, s string) {
	t.Helper()
	if _, err := fmt.Fprint(w, s); err != nil {
		t.Errorf("write response: %v", err)
	}
}

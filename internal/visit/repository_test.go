package visit

import (
	"path/filepath"
	"testing"

	"github.com/evcraddock/house-finder/internal/db"
)

func TestAddAndList(t *testing.T) {
	repo, propID := testSetup(t)

	v, err := repo.Add(propID, "2026-02-08", Showing, "met with Brian")
	if err != nil {
		t.Fatalf("add: %v", err)
	}
	if v.ID == 0 {
		t.Error("expected non-zero ID")
	}
	if v.PropertyID != propID {
		t.Errorf("property_id = %d, want %d", v.PropertyID, propID)
	}
	if v.VisitDate != "2026-02-08" {
		t.Errorf("visit_date = %q, want %q", v.VisitDate, "2026-02-08")
	}
	if v.VisitType != Showing {
		t.Errorf("visit_type = %q, want %q", v.VisitType, Showing)
	}
	if v.Notes != "met with Brian" {
		t.Errorf("notes = %q, want %q", v.Notes, "met with Brian")
	}

	visits, err := repo.ListByPropertyID(propID)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(visits) != 1 {
		t.Fatalf("got %d visits, want 1", len(visits))
	}
}

func TestAddInvalidType(t *testing.T) {
	repo, propID := testSetup(t)

	_, err := repo.Add(propID, "2026-02-08", "invalid_type", "")
	if err == nil {
		t.Fatal("expected error for invalid visit type")
	}
}

func TestAddInvalidDate(t *testing.T) {
	repo, propID := testSetup(t)

	_, err := repo.Add(propID, "not-a-date", Showing, "")
	if err == nil {
		t.Fatal("expected error for invalid date")
	}
}

func TestListOrderByDateDesc(t *testing.T) {
	repo, propID := testSetup(t)

	dates := []string{"2026-01-01", "2026-03-01", "2026-02-01"}
	for _, d := range dates {
		if _, err := repo.Add(propID, d, DriveBy, ""); err != nil {
			t.Fatalf("add %s: %v", d, err)
		}
	}

	visits, err := repo.ListByPropertyID(propID)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(visits) != 3 {
		t.Fatalf("got %d visits, want 3", len(visits))
	}
	if visits[0].VisitDate != "2026-03-01" {
		t.Errorf("first = %q, want newest", visits[0].VisitDate)
	}
	if visits[2].VisitDate != "2026-01-01" {
		t.Errorf("last = %q, want oldest", visits[2].VisitDate)
	}
}

func TestListEmpty(t *testing.T) {
	repo, _ := testSetup(t)

	visits, err := repo.ListByPropertyID(9999)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(visits) != 0 {
		t.Errorf("got %d visits, want 0", len(visits))
	}
}

func TestDelete(t *testing.T) {
	repo, propID := testSetup(t)

	v, err := repo.Add(propID, "2026-02-08", OpenHouse, "")
	if err != nil {
		t.Fatalf("add: %v", err)
	}

	if err := repo.Delete(v.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}

	visits, err := repo.ListByPropertyID(propID)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(visits) != 0 {
		t.Errorf("got %d visits after delete, want 0", len(visits))
	}
}

func TestDeleteNotFound(t *testing.T) {
	repo, _ := testSetup(t)

	err := repo.Delete(9999)
	if err == nil {
		t.Fatal("expected error for missing visit")
	}
}

func TestVisitTypeValid(t *testing.T) {
	tests := []struct {
		t    VisitType
		want bool
	}{
		{Showing, true},
		{DriveBy, true},
		{OpenHouse, true},
		{"invalid", false},
		{"", false},
	}
	for _, tt := range tests {
		if got := tt.t.IsValid(); got != tt.want {
			t.Errorf("VisitType(%q).IsValid() = %v, want %v", tt.t, got, tt.want)
		}
	}
}

func testSetup(t *testing.T) (*Repository, int64) {
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

	res, err := d.Exec(
		`INSERT INTO properties (address, mpr_id, realtor_url, raw_json) VALUES (?, ?, ?, ?)`,
		"123 Test St", "M-TEST", "/detail/test", "{}",
	)
	if err != nil {
		t.Fatalf("insert property: %v", err)
	}
	propID, err := res.LastInsertId()
	if err != nil {
		t.Fatalf("last insert id: %v", err)
	}

	return NewRepository(d), propID
}

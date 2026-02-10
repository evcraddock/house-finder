package property

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/evcraddock/house-finder/internal/db"
)

func TestInsertAndGetByID(t *testing.T) {
	repo := testRepo(t)

	p := &Property{
		Address:    "123 Main St",
		MprID:      "M1111111111",
		RealtorURL: "/detail/123-Main-St",
		RawJSON:    json.RawMessage(`{"price": 250000}`),
	}

	saved, err := repo.Insert(p)
	if err != nil {
		t.Fatalf("insert: %v", err)
	}
	if saved.ID == 0 {
		t.Error("expected non-zero ID")
	}
	if saved.Address != "123 Main St" {
		t.Errorf("address = %q, want %q", saved.Address, "123 Main St")
	}

	got, err := repo.GetByID(saved.ID)
	if err != nil {
		t.Fatalf("get by id: %v", err)
	}
	if got.MprID != "M1111111111" {
		t.Errorf("mpr_id = %q, want %q", got.MprID, "M1111111111")
	}
}

func TestGetByIDNotFound(t *testing.T) {
	repo := testRepo(t)

	_, err := repo.GetByID(9999)
	if err == nil {
		t.Fatal("expected error for missing property")
	}
}

func TestInsertWithFields(t *testing.T) {
	repo := testRepo(t)

	price := int64(25000000) // $250,000 in cents
	beds := 3.0
	baths := 2.5
	sqft := int64(1800)
	lot := 0.25
	year := int64(2005)
	ptype := "single_family"
	status := "active"

	p := &Property{
		Address:      "456 Oak Ave",
		MprID:        "M2222222222",
		RealtorURL:   "/detail/456-Oak-Ave",
		Price:        &price,
		Bedrooms:     &beds,
		Bathrooms:    &baths,
		Sqft:         &sqft,
		LotSize:      &lot,
		YearBuilt:    &year,
		PropertyType: &ptype,
		Status:       &status,
		RawJSON:      json.RawMessage(`{}`),
	}

	saved, err := repo.Insert(p)
	if err != nil {
		t.Fatalf("insert: %v", err)
	}

	if saved.Price == nil || *saved.Price != price {
		t.Errorf("price = %v, want %d", saved.Price, price)
	}
	if saved.Bedrooms == nil || *saved.Bedrooms != beds {
		t.Errorf("bedrooms = %v, want %f", saved.Bedrooms, beds)
	}
	if saved.Bathrooms == nil || *saved.Bathrooms != baths {
		t.Errorf("bathrooms = %v, want %f", saved.Bathrooms, baths)
	}
}

func TestInsertDuplicateMprID(t *testing.T) {
	repo := testRepo(t)

	p := &Property{
		Address:    "123 Main St",
		MprID:      "M-DUPE",
		RealtorURL: "/detail/123",
		RawJSON:    json.RawMessage(`{}`),
	}

	if _, err := repo.Insert(p); err != nil {
		t.Fatalf("first insert: %v", err)
	}

	_, err := repo.Insert(p)
	if err == nil {
		t.Fatal("expected error for duplicate mpr_id")
	}
}

func TestList(t *testing.T) {
	repo := testRepo(t)

	// Insert 3 properties
	for i, addr := range []string{"111 A St", "222 B St", "333 C St"} {
		p := &Property{
			Address:    addr,
			MprID:      fmt.Sprintf("M-LIST-%d", i),
			RealtorURL: fmt.Sprintf("/detail/%d", i),
			RawJSON:    json.RawMessage(`{}`),
		}
		if _, err := repo.Insert(p); err != nil {
			t.Fatalf("insert %d: %v", i, err)
		}
	}

	props, err := repo.List(ListOptions{})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(props) != 3 {
		t.Errorf("got %d properties, want 3", len(props))
	}
}

func TestListFilterByRating(t *testing.T) {
	repo := testRepo(t)

	// Insert properties and rate some
	for i := 0; i < 4; i++ {
		p := &Property{
			Address:    fmt.Sprintf("%d Filter St", i),
			MprID:      fmt.Sprintf("M-FILTER-%d", i),
			RealtorURL: fmt.Sprintf("/detail/filter-%d", i),
			RawJSON:    json.RawMessage(`{}`),
		}
		saved, err := repo.Insert(p)
		if err != nil {
			t.Fatalf("insert %d: %v", i, err)
		}
		if i > 0 { // rate properties 1-3 with ratings 1-3
			if err := repo.UpdateRating(saved.ID, i); err != nil {
				t.Fatalf("rate %d: %v", i, err)
			}
		}
	}

	tests := []struct {
		name      string
		minRating int
		wantCount int
	}{
		{"min rating 1", 1, 3},
		{"min rating 2", 2, 2},
		{"min rating 3", 3, 1},
		{"min rating 4", 4, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rating := tt.minRating
			props, err := repo.List(ListOptions{MinRating: &rating})
			if err != nil {
				t.Fatalf("list: %v", err)
			}
			if len(props) != tt.wantCount {
				t.Errorf("got %d properties, want %d", len(props), tt.wantCount)
			}
		})
	}
}

func TestListFilterByVisited(t *testing.T) {
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

	repo := NewRepository(d)

	// Insert 3 properties
	var ids []int64
	for i := 0; i < 3; i++ {
		p := &Property{
			Address:    fmt.Sprintf("%d Visit St", i),
			MprID:      fmt.Sprintf("M-VISIT-%d", i),
			RealtorURL: fmt.Sprintf("/detail/visit-%d", i),
			RawJSON:    json.RawMessage(`{}`),
		}
		saved, insertErr := repo.Insert(p)
		if insertErr != nil {
			t.Fatalf("insert %d: %v", i, insertErr)
		}
		ids = append(ids, saved.ID)
	}

	// Add a visit to the first property only
	if _, err := d.Exec(
		"INSERT INTO visits (property_id, visit_date, visit_type) VALUES (?, ?, ?)",
		ids[0], "2026-02-08", "showing",
	); err != nil {
		t.Fatalf("insert visit: %v", err)
	}

	// All properties
	all, err := repo.List(ListOptions{})
	if err != nil {
		t.Fatalf("list all: %v", err)
	}
	if len(all) != 3 {
		t.Errorf("all: got %d, want 3", len(all))
	}

	// Visited only
	visitedTrue := true
	visited, err := repo.List(ListOptions{Visited: &visitedTrue})
	if err != nil {
		t.Fatalf("list visited: %v", err)
	}
	if len(visited) != 1 {
		t.Errorf("visited: got %d, want 1", len(visited))
	}

	// Not visited only
	visitedFalse := false
	notVisited, err := repo.List(ListOptions{Visited: &visitedFalse})
	if err != nil {
		t.Fatalf("list not visited: %v", err)
	}
	if len(notVisited) != 2 {
		t.Errorf("not visited: got %d, want 2", len(notVisited))
	}
}

func TestListFilterByStatus(t *testing.T) {
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

	repo := NewRepository(d)

	// Insert 5 properties
	var ids []int64
	for i := 0; i < 5; i++ {
		p := &Property{
			Address:    fmt.Sprintf("%d Status St", i),
			MprID:      fmt.Sprintf("M-STATUS-%d", i),
			RealtorURL: fmt.Sprintf("/detail/status-%d", i),
			RawJSON:    json.RawMessage(`{}`),
		}
		saved, insertErr := repo.Insert(p)
		if insertErr != nil {
			t.Fatalf("insert %d: %v", i, insertErr)
		}
		ids = append(ids, saved.ID)
	}

	// ids[0]: no visits (not-visited)
	// ids[1]: past visit (visited)
	// ids[2]: future visit (scheduled)
	// ids[3]: manually marked visited (no visits, should be visited)
	// Note: if ids[3] also had a future visit, scheduled takes priority

	if _, err := d.Exec(
		"INSERT INTO visits (property_id, visit_date, visit_type) VALUES (?, ?, ?)",
		ids[1], "2020-01-01", "showing",
	); err != nil {
		t.Fatalf("insert past visit: %v", err)
	}

	if _, err := d.Exec(
		"INSERT INTO visits (property_id, visit_date, visit_type) VALUES (?, ?, ?)",
		ids[2], "2099-01-01", "showing",
	); err != nil {
		t.Fatalf("insert future visit: %v", err)
	}

	if err := repo.UpdateVisited(ids[3], true); err != nil {
		t.Fatalf("mark visited: %v", err)
	}

	// ids[4]: manually marked visited + future visit (scheduled takes priority)
	if err := repo.UpdateVisited(ids[4], true); err != nil {
		t.Fatalf("mark visited: %v", err)
	}
	if _, err := d.Exec(
		"INSERT INTO visits (property_id, visit_date, visit_type) VALUES (?, ?, ?)",
		ids[4], "2099-06-01", "showing",
	); err != nil {
		t.Fatalf("insert future visit for overlap: %v", err)
	}

	tests := []struct {
		name   string
		status PropertyStatus
		want   int
	}{
		{"not-visited", StatusNotVisited, 1}, // ids[0]
		{"scheduled", StatusScheduled, 2},    // ids[2] + ids[4] (future visit takes priority)
		{"visited", StatusVisited, 2},        // ids[1] (past visit) + ids[3] (manual, no future visit)
		{"all", StatusAll, 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			props, listErr := repo.List(ListOptions{Status: tt.status})
			if listErr != nil {
				t.Fatalf("list: %v", listErr)
			}
			if len(props) != tt.want {
				t.Errorf("got %d, want %d", len(props), tt.want)
			}
		})
	}
}

func TestUpdateVisited(t *testing.T) {
	repo := testRepo(t)

	p := &Property{
		Address:    "123 Visited St",
		MprID:      "M-VISITED-1",
		RealtorURL: "/detail/visited-1",
		RawJSON:    json.RawMessage(`{}`),
	}
	saved, err := repo.Insert(p)
	if err != nil {
		t.Fatalf("insert: %v", err)
	}

	if saved.Visited {
		t.Error("expected visited=false initially")
	}

	if err := repo.UpdateVisited(saved.ID, true); err != nil {
		t.Fatalf("set visited: %v", err)
	}

	got, err := repo.GetByID(saved.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if !got.Visited {
		t.Error("expected visited=true after update")
	}

	if err := repo.UpdateVisited(saved.ID, false); err != nil {
		t.Fatalf("unset visited: %v", err)
	}

	got, err = repo.GetByID(saved.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Visited {
		t.Error("expected visited=false after unset")
	}
}

func TestUpdateRating(t *testing.T) {
	repo := testRepo(t)

	p := &Property{
		Address:    "789 Rate St",
		MprID:      "M-RATE",
		RealtorURL: "/detail/rate",
		RawJSON:    json.RawMessage(`{}`),
	}
	saved, err := repo.Insert(p)
	if err != nil {
		t.Fatalf("insert: %v", err)
	}

	tests := []struct {
		name    string
		rating  int
		wantErr bool
	}{
		{"valid rating 1", 1, false},
		{"valid rating 4", 4, false},
		{"invalid rating 0", 0, true},
		{"invalid rating 5", 5, true},
		{"invalid rating -1", -1, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := repo.UpdateRating(saved.ID, tt.rating)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			got, err := repo.GetByID(saved.ID)
			if err != nil {
				t.Fatalf("get: %v", err)
			}
			if got.Rating == nil || *got.Rating != int64(tt.rating) {
				t.Errorf("rating = %v, want %d", got.Rating, tt.rating)
			}
		})
	}
}

func TestUpdateRatingNotFound(t *testing.T) {
	repo := testRepo(t)

	err := repo.UpdateRating(9999, 3)
	if err == nil {
		t.Fatal("expected error for missing property")
	}
}

func TestDelete(t *testing.T) {
	repo := testRepo(t)

	p := &Property{
		Address:    "999 Delete St",
		MprID:      "M-DELETE",
		RealtorURL: "/detail/delete",
		RawJSON:    json.RawMessage(`{}`),
	}
	saved, err := repo.Insert(p)
	if err != nil {
		t.Fatalf("insert: %v", err)
	}

	if err := repo.Delete(saved.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}

	_, err = repo.GetByID(saved.ID)
	if err == nil {
		t.Fatal("expected error after delete")
	}
}

func TestDeleteNotFound(t *testing.T) {
	repo := testRepo(t)

	err := repo.Delete(9999)
	if err == nil {
		t.Fatal("expected error for missing property")
	}
}

func TestParseRawJSON(t *testing.T) {
	tests := []struct {
		name     string
		raw      string
		wantFunc func(t *testing.T, f parsedFields)
	}{
		{
			name: "flat fields",
			raw:  `{"list_price": 250000, "beds": 3, "baths": 2.5, "sqft": 1800, "year_built": 2005, "prop_type": "single_family", "prop_status": "active", "lot_sqft": 10890}`,
			wantFunc: func(t *testing.T, f parsedFields) {
				assertInt64(t, "price", f.Price, 250000)
				assertFloat64(t, "beds", f.Bedrooms, 3)
				assertFloat64(t, "baths", f.Bathrooms, 2.5)
				assertInt64(t, "sqft", f.Sqft, 1800)
				assertInt64(t, "year_built", f.YearBuilt, 2005)
				assertString(t, "prop_type", f.PropertyType, "single_family")
				assertString(t, "status", f.Status, "active")
				if f.LotSize == nil {
					t.Error("lot_size is nil")
				} else if *f.LotSize < 0.24 || *f.LotSize > 0.26 {
					t.Errorf("lot_size = %f, want ~0.25 acres", *f.LotSize)
				}
			},
		},
		{
			name: "nested under data with description",
			raw:  `{"data": {"list_price": 300000, "status": "for_sale", "description": {"beds": 4, "baths": 3, "sqft": 2200, "year_built": 2010, "type": "single_family", "lot_sqft": 8712}}}`,
			wantFunc: func(t *testing.T, f parsedFields) {
				assertInt64(t, "price", f.Price, 300000)
				assertFloat64(t, "beds", f.Bedrooms, 4)
				assertFloat64(t, "baths", f.Bathrooms, 3)
				assertInt64(t, "sqft", f.Sqft, 2200)
				assertInt64(t, "year_built", f.YearBuilt, 2010)
				assertString(t, "type", f.PropertyType, "single_family")
				if f.LotSize == nil {
					t.Error("lot_size is nil")
				} else if *f.LotSize < 0.19 || *f.LotSize > 0.21 {
					t.Errorf("lot_size = %f, want ~0.20 acres", *f.LotSize)
				}
			},
		},
		{
			name: "empty json",
			raw:  `{}`,
			wantFunc: func(t *testing.T, f parsedFields) {
				if f.Price != nil {
					t.Error("expected nil price for empty json")
				}
			},
		},
		{
			name: "invalid json",
			raw:  `not json`,
			wantFunc: func(t *testing.T, f parsedFields) {
				if f.Price != nil {
					t.Error("expected nil price for invalid json")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := parseRawJSON(json.RawMessage(tt.raw))
			tt.wantFunc(t, f)
		})
	}
}

func testRepo(t *testing.T) *Repository {
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
	return NewRepository(d)
}

func assertInt64(t *testing.T, name string, got *int64, want int64) {
	t.Helper()
	if got == nil {
		t.Errorf("%s is nil, want %d", name, want)
	} else if *got != want {
		t.Errorf("%s = %d, want %d", name, *got, want)
	}
}

func assertFloat64(t *testing.T, name string, got *float64, want float64) {
	t.Helper()
	if got == nil {
		t.Errorf("%s is nil, want %f", name, want)
	} else if *got != want {
		t.Errorf("%s = %f, want %f", name, *got, want)
	}
}

func assertString(t *testing.T, name string, got *string, want string) {
	t.Helper()
	if got == nil {
		t.Errorf("%s is nil, want %q", name, want)
	} else if *got != want {
		t.Errorf("%s = %q, want %q", name, *got, want)
	}
}

package comment

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/evcraddock/house-finder/internal/db"
)

func TestAddAndListByPropertyID(t *testing.T) {
	repo, propID := testSetup(t)

	c, err := repo.Add(propID, "Nice backyard", "test@example.com")
	if err != nil {
		t.Fatalf("add: %v", err)
	}
	if c.ID == 0 {
		t.Error("expected non-zero ID")
	}
	if c.PropertyID != propID {
		t.Errorf("property_id = %d, want %d", c.PropertyID, propID)
	}
	if c.Text != "Nice backyard" {
		t.Errorf("text = %q, want %q", c.Text, "Nice backyard")
	}

	comments, err := repo.ListByPropertyID(propID)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(comments) != 1 {
		t.Fatalf("got %d comments, want 1", len(comments))
	}
	if comments[0].Text != "Nice backyard" {
		t.Errorf("text = %q, want %q", comments[0].Text, "Nice backyard")
	}
}

func TestAddEmptyText(t *testing.T) {
	repo, propID := testSetup(t)

	_, err := repo.Add(propID, "", "test@example.com")
	if err == nil {
		t.Fatal("expected error for empty text")
	}
	_ = propID
}

func TestListByPropertyIDEmpty(t *testing.T) {
	repo, _ := testSetup(t)

	comments, err := repo.ListByPropertyID(9999)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(comments) != 0 {
		t.Errorf("got %d comments, want 0", len(comments))
	}
}

func TestListOrderNewestFirst(t *testing.T) {
	repo, propID := testSetup(t)

	texts := []string{"first", "second", "third"}
	for _, text := range texts {
		if _, err := repo.Add(propID, text, "test@example.com"); err != nil {
			t.Fatalf("add %q: %v", text, err)
		}
	}

	comments, err := repo.ListByPropertyID(propID)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(comments) != 3 {
		t.Fatalf("got %d comments, want 3", len(comments))
	}

	// Newest first
	if comments[0].Text != "third" {
		t.Errorf("first comment = %q, want %q", comments[0].Text, "third")
	}
	if comments[2].Text != "first" {
		t.Errorf("last comment = %q, want %q", comments[2].Text, "first")
	}
}

func TestAuthorStored(t *testing.T) {
	repo, propID := testSetup(t)

	c, err := repo.Add(propID, "Author test", "alice@example.com")
	if err != nil {
		t.Fatalf("add: %v", err)
	}
	if c.Author != "alice@example.com" {
		t.Errorf("author = %q, want %q", c.Author, "alice@example.com")
	}

	comments, err := repo.ListByPropertyID(propID)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if comments[0].Author != "alice@example.com" {
		t.Errorf("listed author = %q, want %q", comments[0].Author, "alice@example.com")
	}
}

func TestAuthorEmpty(t *testing.T) {
	repo, propID := testSetup(t)

	c, err := repo.Add(propID, "No author", "")
	if err != nil {
		t.Fatalf("add: %v", err)
	}
	if c.Author != "" {
		t.Errorf("author = %q, want empty", c.Author)
	}
}

func TestDelete(t *testing.T) {
	repo, propID := testSetup(t)

	c, err := repo.Add(propID, "To be deleted", "test@example.com")
	if err != nil {
		t.Fatalf("add: %v", err)
	}

	if err := repo.Delete(c.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}

	comments, err := repo.ListByPropertyID(propID)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(comments) != 0 {
		t.Errorf("got %d comments after delete, want 0", len(comments))
	}
}

func TestDeleteNotFound(t *testing.T) {
	repo, _ := testSetup(t)

	err := repo.Delete(9999)
	if err == nil {
		t.Fatal("expected error for missing comment")
	}
}

func TestCascadeDeleteWithProperty(t *testing.T) {
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

	// Insert a property
	res, err := d.Exec(
		`INSERT INTO properties (address, mpr_id, realtor_url, raw_json) VALUES (?, ?, ?, ?)`,
		"123 Cascade St", "M-CASCADE", "/detail/cascade", "{}",
	)
	if err != nil {
		t.Fatalf("insert property: %v", err)
	}
	propID, err := res.LastInsertId()
	if err != nil {
		t.Fatalf("last insert id: %v", err)
	}

	repo := NewRepository(d)

	// Add comments
	for _, text := range []string{"comment 1", "comment 2"} {
		if _, err := repo.Add(propID, text, "test@example.com"); err != nil {
			t.Fatalf("add comment: %v", err)
		}
	}

	// Delete the property
	if _, err := d.Exec("DELETE FROM properties WHERE id = ?", propID); err != nil {
		t.Fatalf("delete property: %v", err)
	}

	// Comments should be gone
	comments, err := repo.ListByPropertyID(propID)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(comments) != 0 {
		t.Errorf("got %d comments after cascade delete, want 0", len(comments))
	}
}

// testSetup creates a test DB with a property and returns a comment repo + property ID.
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
		"123 Test St", "M-TEST", "/detail/test", json.RawMessage(`{}`),
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

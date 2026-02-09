package visit

import (
	"database/sql"
	"fmt"
	"time"
)

// Repository provides CRUD operations for property visits.
type Repository struct {
	db *sql.DB
}

// NewRepository creates a visit repository.
func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// Add records a new visit to a property.
func (r *Repository) Add(propertyID int64, visitDate string, visitType VisitType, notes string) (*Visit, error) {
	if !visitType.IsValid() {
		return nil, fmt.Errorf("invalid visit type: %q", visitType)
	}

	if _, err := time.Parse("2006-01-02", visitDate); err != nil {
		return nil, fmt.Errorf("invalid date format (use YYYY-MM-DD): %w", err)
	}

	result, err := r.db.Exec(
		"INSERT INTO visits (property_id, visit_date, visit_type, notes) VALUES (?, ?, ?, ?)",
		propertyID, visitDate, visitType, notes,
	)
	if err != nil {
		return nil, fmt.Errorf("inserting visit: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("getting insert id: %w", err)
	}

	var v Visit
	err = r.db.QueryRow(
		"SELECT id, property_id, visit_date, visit_type, notes, created_at FROM visits WHERE id = ?", id,
	).Scan(&v.ID, &v.PropertyID, &v.VisitDate, &v.VisitType, &v.Notes, &v.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("reading back visit: %w", err)
	}

	return &v, nil
}

// ListByPropertyID returns all visits for a property, newest first.
func (r *Repository) ListByPropertyID(propertyID int64) ([]*Visit, error) {
	rows, err := r.db.Query(
		"SELECT id, property_id, visit_date, visit_type, notes, created_at FROM visits WHERE property_id = ? ORDER BY visit_date DESC, id DESC",
		propertyID,
	)
	if err != nil {
		return nil, fmt.Errorf("listing visits: %w", err)
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			err = fmt.Errorf("closing rows: %w", closeErr)
		}
	}()

	var visits []*Visit
	for rows.Next() {
		var v Visit
		if err := rows.Scan(&v.ID, &v.PropertyID, &v.VisitDate, &v.VisitType, &v.Notes, &v.CreatedAt); err != nil {
			return nil, fmt.Errorf("scanning visit: %w", err)
		}
		visits = append(visits, &v)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating visits: %w", err)
	}

	return visits, nil
}

// LastVisitByProperty returns the most recent visit for each property that has one.
// Returns a map of property_id -> Visit.
func (r *Repository) LastVisitByProperty() (map[int64]*Visit, error) {
	rows, err := r.db.Query(
		`SELECT v.id, v.property_id, v.visit_date, v.visit_type, v.notes, v.created_at
		 FROM visits v
		 INNER JOIN (
		     SELECT property_id, MAX(visit_date) AS max_date
		     FROM visits GROUP BY property_id
		 ) latest ON v.property_id = latest.property_id AND v.visit_date = latest.max_date
		 ORDER BY v.property_id`,
	)
	if err != nil {
		return nil, fmt.Errorf("querying last visits: %w", err)
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			err = fmt.Errorf("closing rows: %w", closeErr)
		}
	}()

	result := make(map[int64]*Visit)
	for rows.Next() {
		var v Visit
		if err := rows.Scan(&v.ID, &v.PropertyID, &v.VisitDate, &v.VisitType, &v.Notes, &v.CreatedAt); err != nil {
			return nil, fmt.Errorf("scanning visit: %w", err)
		}
		result[v.PropertyID] = &v
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating visits: %w", err)
	}

	return result, nil
}

// Delete removes a visit by ID.
func (r *Repository) Delete(id int64) error {
	result, err := r.db.Exec("DELETE FROM visits WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("deleting visit: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("checking rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("visit %d not found", id)
	}

	return nil
}

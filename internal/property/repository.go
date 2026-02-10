package property

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
)

// Repository provides CRUD operations for properties.
type Repository struct {
	db *sql.DB
}

// NewRepository creates a property repository.
func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

const insertSQL = `INSERT INTO properties
	(address, mpr_id, realtor_url, price, bedrooms, bathrooms, sqft, lot_size, year_built, property_type, status, raw_json)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

const selectColumns = `id, address, mpr_id, realtor_url, price, bedrooms, bathrooms, sqft, lot_size, year_built, property_type, status, rating, visit_status, raw_json, created_at, updated_at`

// Insert adds a new property and returns it with its generated ID.
func (r *Repository) Insert(p *Property) (*Property, error) {
	result, err := r.db.Exec(insertSQL,
		p.Address, p.MprID, p.RealtorURL,
		p.Price, p.Bedrooms, p.Bathrooms, p.Sqft, p.LotSize,
		p.YearBuilt, p.PropertyType, p.Status,
		string(p.RawJSON),
	)
	if err != nil {
		return nil, fmt.Errorf("inserting property: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("getting insert id: %w", err)
	}

	return r.GetByID(id)
}

// GetByID returns a property by its ID.
func (r *Repository) GetByID(id int64) (*Property, error) {
	query := fmt.Sprintf("SELECT %s FROM properties WHERE id = ?", selectColumns)
	row := r.db.QueryRow(query, id)

	p, err := scanProperty(row)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("property %d not found", id)
	}
	if err != nil {
		return nil, fmt.Errorf("querying property %d: %w", id, err)
	}

	return p, nil
}

// ListOptions controls filtering for List.
type ListOptions struct {
	MinRating   *int
	VisitStatus VisitStatus // empty = all
}

// List returns all properties, optionally filtered.
func (r *Repository) List(opts ListOptions) ([]*Property, error) {
	query := fmt.Sprintf("SELECT %s FROM properties", selectColumns)
	var args []interface{}
	var conditions []string

	if opts.MinRating != nil {
		conditions = append(conditions, "rating >= ?")
		args = append(args, *opts.MinRating)
	}

	if opts.VisitStatus != "" {
		conditions = append(conditions, "visit_status = ?")
		args = append(args, string(opts.VisitStatus))
	}

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	query += " ORDER BY COALESCE(rating, 0) DESC, created_at DESC"

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("listing properties: %w", err)
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			err = fmt.Errorf("closing rows: %w", closeErr)
		}
	}()

	var properties []*Property
	for rows.Next() {
		p, err := scanProperty(rows)
		if err != nil {
			return nil, fmt.Errorf("scanning property: %w", err)
		}
		properties = append(properties, p)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating properties: %w", err)
	}

	return properties, nil
}

// UpdateRating sets the rating (1-4) for a property.
func (r *Repository) UpdateRating(id int64, rating int) error {
	if rating < 1 || rating > 4 {
		return fmt.Errorf("rating must be 1-4, got %d", rating)
	}

	result, err := r.db.Exec(
		"UPDATE properties SET rating = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?",
		rating, id,
	)
	if err != nil {
		return fmt.Errorf("updating rating: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("checking rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("property %d not found", id)
	}

	return nil
}

// UpdateVisitStatus sets the visit status for a property.
func (r *Repository) UpdateVisitStatus(id int64, status VisitStatus) error {
	if !ValidVisitStatus(string(status)) {
		return fmt.Errorf("invalid visit status: %s", status)
	}

	result, err := r.db.Exec(
		"UPDATE properties SET visit_status = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?",
		string(status), id,
	)
	if err != nil {
		return fmt.Errorf("updating visit status: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("checking rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("property %d not found", id)
	}

	return nil
}

// Delete removes a property by ID. Comments cascade.
func (r *Repository) Delete(id int64) error {
	result, err := r.db.Exec("DELETE FROM properties WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("deleting property: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("checking rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("property %d not found", id)
	}

	return nil
}

// parseRawJSON extracts known fields from the raw API response.
// It tries common field paths from the RapidAPI us-real-estate-listings response.
func parseRawJSON(raw json.RawMessage) parsedFields {
	var f parsedFields

	// Try to parse the top-level structure
	var data map[string]json.RawMessage
	if err := json.Unmarshal(raw, &data); err != nil {
		return f
	}

	// Navigate into nested "data" key if present
	if nested, ok := data["data"]; ok {
		var m map[string]json.RawMessage
		if err := json.Unmarshal(nested, &m); err == nil {
			data = m
		}
	}

	// Extract the "description" sub-object where beds/baths/sqft live
	var desc map[string]json.RawMessage
	if raw, ok := data["description"]; ok {
		if err := json.Unmarshal(raw, &desc); err != nil {
			desc = nil
		}
	}

	// Price lives at the top level
	f.Price = jsonInt64(data, "list_price", "price")
	f.Status = jsonString(data, "prop_status", "status")

	// Beds, baths, sqft, year_built, type, lot_sqft live in description
	if desc != nil {
		f.Bedrooms = jsonFloat64(desc, "beds", "bedrooms")
		f.Bathrooms = jsonFloat64(desc, "baths", "bathrooms", "baths_consolidated")
		f.Sqft = jsonInt64(desc, "sqft", "building_size", "living_area")
		f.YearBuilt = jsonInt64(desc, "year_built")
		f.PropertyType = jsonString(desc, "type", "prop_type", "property_type")
		f.LotSize = jsonFloat64(desc, "lot_sqft", "lot_size")
	}

	// Fallback: try top level for everything if description didn't have it
	if f.Bedrooms == nil {
		f.Bedrooms = jsonFloat64(data, "beds", "bedrooms")
	}
	if f.Bathrooms == nil {
		f.Bathrooms = jsonFloat64(data, "baths", "bathrooms", "baths_consolidated")
	}
	if f.Sqft == nil {
		f.Sqft = jsonInt64(data, "sqft", "building_size", "living_area")
	}
	if f.YearBuilt == nil {
		f.YearBuilt = jsonInt64(data, "year_built")
	}
	if f.PropertyType == nil {
		f.PropertyType = jsonString(data, "type", "prop_type", "property_type")
	}
	if f.LotSize == nil {
		f.LotSize = jsonFloat64(data, "lot_sqft", "lot_size")
	}

	// Convert lot sqft to acres if it seems like sqft (> 100)
	if f.LotSize != nil && *f.LotSize > 100 {
		acres := *f.LotSize / 43560.0
		f.LotSize = &acres
	}

	return f
}

type parsedFields struct {
	Price        *int64
	Bedrooms     *float64
	Bathrooms    *float64
	Sqft         *int64
	LotSize      *float64
	YearBuilt    *int64
	PropertyType *string
	Status       *string
}

// jsonInt64 tries multiple keys and returns the first valid int64 value.
func jsonInt64(data map[string]json.RawMessage, keys ...string) *int64 {
	for _, key := range keys {
		raw, ok := data[key]
		if !ok {
			continue
		}
		var v float64
		if err := json.Unmarshal(raw, &v); err == nil {
			i := int64(v)
			return &i
		}
	}
	return nil
}

// jsonFloat64 tries multiple keys and returns the first valid float64 value.
func jsonFloat64(data map[string]json.RawMessage, keys ...string) *float64 {
	for _, key := range keys {
		raw, ok := data[key]
		if !ok {
			continue
		}
		var v float64
		if err := json.Unmarshal(raw, &v); err == nil {
			return &v
		}
	}
	return nil
}

// jsonString tries multiple keys and returns the first valid string value.
func jsonString(data map[string]json.RawMessage, keys ...string) *string {
	for _, key := range keys {
		raw, ok := data[key]
		if !ok {
			continue
		}
		var v string
		if err := json.Unmarshal(raw, &v); err == nil && v != "" {
			return &v
		}
	}
	return nil
}

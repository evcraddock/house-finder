// Package property provides the property domain model and data access.
package property

import (
	"database/sql"
	"encoding/json"
	"time"
)

// Property represents a tracked house listing.
type Property struct {
	ID           int64           `json:"id"`
	Address      string          `json:"address"`
	MprID        string          `json:"mpr_id"`
	RealtorURL   string          `json:"realtor_url"`
	Price        *int64          `json:"price,omitempty"`
	Bedrooms     *float64        `json:"bedrooms,omitempty"`
	Bathrooms    *float64        `json:"bathrooms,omitempty"`
	Sqft         *int64          `json:"sqft,omitempty"`
	LotSize      *float64        `json:"lot_size,omitempty"`
	YearBuilt    *int64          `json:"year_built,omitempty"`
	PropertyType *string         `json:"property_type,omitempty"`
	Status       *string         `json:"status,omitempty"`
	Rating       *int64          `json:"rating,omitempty"`
	Visited      bool            `json:"visited"`
	RawJSON      json.RawMessage `json:"raw_json"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
}

// scanProperty scans a property from a database row.
func scanProperty(row interface{ Scan(...interface{}) error }) (*Property, error) {
	var p Property
	var price, sqft, yearBuilt, rating sql.NullInt64
	var bedrooms, bathrooms, lotSize sql.NullFloat64
	var propertyType, status sql.NullString
	var rawJSON string

	err := row.Scan(
		&p.ID, &p.Address, &p.MprID, &p.RealtorURL,
		&price, &bedrooms, &bathrooms, &sqft, &lotSize,
		&yearBuilt, &propertyType, &status, &rating,
		&p.Visited, &rawJSON, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	if price.Valid {
		p.Price = &price.Int64
	}
	if bedrooms.Valid {
		p.Bedrooms = &bedrooms.Float64
	}
	if bathrooms.Valid {
		p.Bathrooms = &bathrooms.Float64
	}
	if sqft.Valid {
		p.Sqft = &sqft.Int64
	}
	if lotSize.Valid {
		p.LotSize = &lotSize.Float64
	}
	if yearBuilt.Valid {
		p.YearBuilt = &yearBuilt.Int64
	}
	if propertyType.Valid {
		p.PropertyType = &propertyType.String
	}
	if status.Valid {
		p.Status = &status.String
	}
	if rating.Valid {
		p.Rating = &rating.Int64
	}
	p.RawJSON = json.RawMessage(rawJSON)

	return &p, nil
}

// Package property provides the property domain model and data access.
package property

import (
	"database/sql"
	"encoding/json"
	"time"
)

// VisitStatus represents where a property is in the visit workflow.
type VisitStatus string

const (
	VisitStatusNotVisited  VisitStatus = "not_visited"
	VisitStatusWantToVisit VisitStatus = "want_to_visit"
	VisitStatusVisited     VisitStatus = "visited"
)

// ValidVisitStatus returns true if s is a known visit status.
func ValidVisitStatus(s string) bool {
	switch VisitStatus(s) {
	case VisitStatusNotVisited, VisitStatusWantToVisit, VisitStatusVisited:
		return true
	}
	return false
}

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
	VisitStatus  VisitStatus     `json:"visit_status"`
	PhotoURL     string          `json:"photo_url,omitempty"`
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

	var visitStatus string
	err := row.Scan(
		&p.ID, &p.Address, &p.MprID, &p.RealtorURL,
		&price, &bedrooms, &bathrooms, &sqft, &lotSize,
		&yearBuilt, &propertyType, &status, &rating,
		&visitStatus, &rawJSON, &p.CreatedAt, &p.UpdatedAt,
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
	p.VisitStatus = VisitStatus(visitStatus)
	if p.VisitStatus == "" {
		p.VisitStatus = VisitStatusNotVisited
	}
	p.RawJSON = json.RawMessage(rawJSON)
	p.PhotoURL = extractPhotoURL(p.RawJSON)

	return &p, nil
}

// extractPhotoURL finds the primary exterior photo from raw API JSON.
// Prefers the first photo tagged "house_view", falls back to first photo.
func extractPhotoURL(raw json.RawMessage) string {
	var data map[string]json.RawMessage
	if err := json.Unmarshal(raw, &data); err != nil {
		return ""
	}

	// Navigate into "data" key if present
	if nested, ok := data["data"]; ok {
		var m map[string]json.RawMessage
		if err := json.Unmarshal(nested, &m); err == nil {
			data = m
		}
	}

	photosRaw, ok := data["photos"]
	if !ok {
		return ""
	}

	var photos []struct {
		Href string `json:"href"`
		Tags []struct {
			Label string `json:"label"`
		} `json:"tags"`
	}
	if err := json.Unmarshal(photosRaw, &photos); err != nil || len(photos) == 0 {
		return ""
	}

	// Prefer first photo with house_view tag
	for _, p := range photos {
		for _, t := range p.Tags {
			if t.Label == "house_view" && p.Href != "" {
				return p.Href
			}
		}
	}

	// Fallback to first photo
	return photos[0].Href
}

// Package visit provides the property visit domain model and data access.
package visit

import "time"

// VisitType represents how a property was visited.
type VisitType string

const (
	Showing   VisitType = "showing"
	DriveBy   VisitType = "drive_by"
	OpenHouse VisitType = "open_house"
)

// ValidTypes is the set of allowed visit types.
var ValidTypes = []VisitType{Showing, DriveBy, OpenHouse}

// IsValid checks if a visit type is recognized.
func (t VisitType) IsValid() bool {
	for _, v := range ValidTypes {
		if t == v {
			return true
		}
	}
	return false
}

// Label returns a human-readable label for the visit type.
func (t VisitType) Label() string {
	switch t {
	case Showing:
		return "Showing"
	case DriveBy:
		return "Drive-by"
	case OpenHouse:
		return "Open House"
	default:
		return string(t)
	}
}

// Visit represents a recorded visit to a property.
type Visit struct {
	ID         int64     `json:"id"`
	PropertyID int64     `json:"property_id"`
	VisitDate  string    `json:"visit_date"` // YYYY-MM-DD
	VisitType  VisitType `json:"visit_type"`
	Notes      string    `json:"notes"`
	CreatedAt  time.Time `json:"created_at"`
}

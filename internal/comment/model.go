// Package comment provides the comment domain model and data access.
package comment

import "time"

// Comment represents a user note on a property.
type Comment struct {
	ID         int64     `json:"id"`
	PropertyID int64     `json:"property_id"`
	Text       string    `json:"text"`
	Author     string    `json:"author"`
	CreatedAt  time.Time `json:"created_at"`
}

package comment

import (
	"database/sql"
	"fmt"
)

// Repository provides CRUD operations for comments.
type Repository struct {
	db *sql.DB
}

// NewRepository creates a comment repository.
func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// Add creates a new comment on a property.
func (r *Repository) Add(propertyID int64, text string) (*Comment, error) {
	if text == "" {
		return nil, fmt.Errorf("comment text is required")
	}

	result, err := r.db.Exec(
		"INSERT INTO comments (property_id, text) VALUES (?, ?)",
		propertyID, text,
	)
	if err != nil {
		return nil, fmt.Errorf("inserting comment: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("getting insert id: %w", err)
	}

	var c Comment
	err = r.db.QueryRow(
		"SELECT id, property_id, text, created_at FROM comments WHERE id = ?", id,
	).Scan(&c.ID, &c.PropertyID, &c.Text, &c.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("reading back comment: %w", err)
	}

	return &c, nil
}

// ListByPropertyID returns all comments for a property, newest first.
func (r *Repository) ListByPropertyID(propertyID int64) ([]*Comment, error) {
	rows, err := r.db.Query(
		"SELECT id, property_id, text, created_at FROM comments WHERE property_id = ? ORDER BY id DESC",
		propertyID,
	)
	if err != nil {
		return nil, fmt.Errorf("listing comments: %w", err)
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			err = fmt.Errorf("closing rows: %w", closeErr)
		}
	}()

	var comments []*Comment
	for rows.Next() {
		var c Comment
		if err := rows.Scan(&c.ID, &c.PropertyID, &c.Text, &c.CreatedAt); err != nil {
			return nil, fmt.Errorf("scanning comment: %w", err)
		}
		comments = append(comments, &c)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating comments: %w", err)
	}

	return comments, nil
}

// Delete removes a comment by ID.
func (r *Repository) Delete(id int64) error {
	result, err := r.db.Exec("DELETE FROM comments WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("deleting comment: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("checking rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("comment %d not found", id)
	}

	return nil
}

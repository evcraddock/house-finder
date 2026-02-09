package auth

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// User represents an authorized user.
type User struct {
	ID        int64     `json:"id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	Phone     string    `json:"phone"`
	IsRealtor bool      `json:"is_realtor"`
	CreatedAt time.Time `json:"created_at"`
}

// UserStore manages authorized users in SQLite.
type UserStore struct {
	db         *sql.DB
	adminEmail string
}

// NewUserStore creates a user store.
func NewUserStore(db *sql.DB, adminEmail string) *UserStore {
	return &UserStore{db: db, adminEmail: strings.ToLower(adminEmail)}
}

// IsAuthorized checks if an email is allowed to log in.
// The admin email is always authorized (outside the users table).
func (s *UserStore) IsAuthorized(email string) bool {
	email = strings.ToLower(email)

	if email == s.adminEmail {
		return true
	}

	var count int
	err := s.db.QueryRow(
		"SELECT COUNT(*) FROM authorized_users WHERE LOWER(email) = ?", email,
	).Scan(&count)
	if err != nil {
		return false
	}

	return count > 0
}

// IsAdmin checks if an email is the admin.
func (s *UserStore) IsAdmin(email string) bool {
	return strings.ToLower(email) == s.adminEmail
}

// Add creates a new authorized user.
func (s *UserStore) Add(email, name, phone string, isRealtor bool) (*User, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	name = strings.TrimSpace(name)
	phone = strings.TrimSpace(phone)

	if email == "" {
		return nil, fmt.Errorf("email is required")
	}

	result, err := s.db.Exec(
		"INSERT INTO authorized_users (email, name, phone, is_realtor) VALUES (?, ?, ?, ?)",
		email, name, phone, isRealtor,
	)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			return nil, fmt.Errorf("user already exists: %s", email)
		}
		return nil, fmt.Errorf("adding user: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("getting user ID: %w", err)
	}

	return s.GetByID(id)
}

// Update modifies an existing user's profile fields.
func (s *UserStore) Update(id int64, name, phone string, isRealtor bool) (*User, error) {
	name = strings.TrimSpace(name)
	phone = strings.TrimSpace(phone)

	result, err := s.db.Exec(
		"UPDATE authorized_users SET name = ?, phone = ?, is_realtor = ? WHERE id = ?",
		name, phone, isRealtor, id,
	)
	if err != nil {
		return nil, fmt.Errorf("updating user: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("checking affected rows: %w", err)
	}
	if rows == 0 {
		return nil, fmt.Errorf("user not found")
	}

	return s.GetByID(id)
}

// List returns all authorized users.
func (s *UserStore) List() ([]*User, error) {
	rows, err := s.db.Query(
		"SELECT id, email, name, phone, is_realtor, created_at FROM authorized_users ORDER BY email",
	)
	if err != nil {
		return nil, fmt.Errorf("listing users: %w", err)
	}
	defer func() {
		if cerr := rows.Close(); cerr != nil {
			fmt.Printf("warning: closing rows: %v\n", cerr)
		}
	}()

	var users []*User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Email, &u.Name, &u.Phone, &u.IsRealtor, &u.CreatedAt); err != nil {
			return nil, fmt.Errorf("scanning user: %w", err)
		}
		users = append(users, &u)
	}

	return users, rows.Err()
}

// GetByID returns a user by ID.
func (s *UserStore) GetByID(id int64) (*User, error) {
	var u User
	err := s.db.QueryRow(
		"SELECT id, email, name, phone, is_realtor, created_at FROM authorized_users WHERE id = ?", id,
	).Scan(&u.ID, &u.Email, &u.Name, &u.Phone, &u.IsRealtor, &u.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	}
	if err != nil {
		return nil, fmt.Errorf("querying user: %w", err)
	}
	return &u, nil
}

// Delete removes an authorized user by ID.
func (s *UserStore) Delete(id int64) error {
	result, err := s.db.Exec("DELETE FROM authorized_users WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("deleting user: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("checking affected rows: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("user not found")
	}

	return nil
}

// AllEmails returns all authorized emails including the admin.
// Useful for passkey login to resolve any user.
func (s *UserStore) AllEmails() ([]string, error) {
	rows, err := s.db.Query("SELECT email FROM authorized_users")
	if err != nil {
		return nil, fmt.Errorf("listing emails: %w", err)
	}
	defer func() {
		if cerr := rows.Close(); cerr != nil {
			fmt.Printf("warning: closing rows: %v\n", cerr)
		}
	}()

	emails := []string{s.adminEmail}
	for rows.Next() {
		var email string
		if err := rows.Scan(&email); err != nil {
			return nil, fmt.Errorf("scanning email: %w", err)
		}
		if strings.ToLower(email) != s.adminEmail {
			emails = append(emails, strings.ToLower(email))
		}
	}

	return emails, rows.Err()
}

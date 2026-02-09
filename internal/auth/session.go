package auth

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"net/http"
	"time"
)

const (
	sessionExpiry = 30 * 24 * time.Hour // 30 days
	cookieName    = "hf_session"
)

// SessionStore manages sessions in SQLite.
type SessionStore struct {
	db *sql.DB
}

// NewSessionStore creates a session store.
func NewSessionStore(db *sql.DB) *SessionStore {
	return &SessionStore{db: db}
}

// Create generates a new session for the given email and sets the cookie.
func (s *SessionStore) Create(w http.ResponseWriter, email string) error {
	id, err := generateSessionID()
	if err != nil {
		return fmt.Errorf("generating session ID: %w", err)
	}

	expiresAt := time.Now().Add(sessionExpiry)

	if _, err := s.db.Exec(
		"INSERT INTO sessions (id, email, expires_at) VALUES (?, ?, ?)",
		id, email, expiresAt,
	); err != nil {
		return fmt.Errorf("storing session: %w", err)
	}

	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    id,
		Path:     "/",
		Expires:  expiresAt,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	return nil
}

// Validate checks the session cookie and returns the email if valid.
func (s *SessionStore) Validate(r *http.Request) (string, error) {
	cookie, err := r.Cookie(cookieName)
	if err != nil {
		return "", fmt.Errorf("no session cookie")
	}

	var email string
	var expiresAt time.Time

	err = s.db.QueryRow(
		"SELECT email, expires_at FROM sessions WHERE id = ?",
		cookie.Value,
	).Scan(&email, &expiresAt)
	if err == sql.ErrNoRows {
		return "", fmt.Errorf("invalid session")
	}
	if err != nil {
		return "", fmt.Errorf("querying session: %w", err)
	}

	if time.Now().After(expiresAt) {
		// Clean up expired session
		if _, delErr := s.db.Exec("DELETE FROM sessions WHERE id = ?", cookie.Value); delErr != nil {
			return "", fmt.Errorf("deleting expired session: %w", delErr)
		}
		return "", fmt.Errorf("session expired")
	}

	return email, nil
}

// Destroy removes the session and clears the cookie.
func (s *SessionStore) Destroy(w http.ResponseWriter, r *http.Request) error {
	cookie, err := r.Cookie(cookieName)
	if err != nil {
		return nil // no session to destroy
	}

	if _, err := s.db.Exec("DELETE FROM sessions WHERE id = ?", cookie.Value); err != nil {
		return fmt.Errorf("deleting session: %w", err)
	}

	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	return nil
}

// Cleanup removes expired sessions.
func (s *SessionStore) Cleanup() error {
	if _, err := s.db.Exec(
		"DELETE FROM sessions WHERE expires_at < ?",
		time.Now(),
	); err != nil {
		return fmt.Errorf("cleaning up sessions: %w", err)
	}
	return nil
}

func generateSessionID() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

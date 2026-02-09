package auth

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"time"
)

const tokenExpiry = 15 * time.Minute

// TokenStore manages magic link tokens in SQLite.
type TokenStore struct {
	db *sql.DB
}

// NewTokenStore creates a token store.
func NewTokenStore(db *sql.DB) *TokenStore {
	return &TokenStore{db: db}
}

// Create generates a new magic link token for the given email.
// Returns the raw token string.
func (s *TokenStore) Create(email string) (string, error) {
	token, err := generateToken()
	if err != nil {
		return "", fmt.Errorf("generating token: %w", err)
	}

	expiresAt := time.Now().Add(tokenExpiry)

	if _, err := s.db.Exec(
		"INSERT INTO auth_tokens (token, email, expires_at) VALUES (?, ?, ?)",
		token, email, expiresAt,
	); err != nil {
		return "", fmt.Errorf("storing token: %w", err)
	}

	return token, nil
}

// Validate checks a token and returns the associated email.
// The token is marked as used and cannot be reused.
func (s *TokenStore) Validate(token string) (string, error) {
	var email string
	var used int
	var expiresAt time.Time

	err := s.db.QueryRow(
		"SELECT email, used, expires_at FROM auth_tokens WHERE token = ?",
		token,
	).Scan(&email, &used, &expiresAt)
	if err == sql.ErrNoRows {
		return "", fmt.Errorf("invalid token")
	}
	if err != nil {
		return "", fmt.Errorf("querying token: %w", err)
	}

	if used != 0 {
		return "", fmt.Errorf("token already used")
	}

	if time.Now().After(expiresAt) {
		return "", fmt.Errorf("token expired")
	}

	// Mark as used
	if _, err := s.db.Exec(
		"UPDATE auth_tokens SET used = 1 WHERE token = ?",
		token,
	); err != nil {
		return "", fmt.Errorf("marking token used: %w", err)
	}

	return email, nil
}

// Cleanup removes expired tokens.
func (s *TokenStore) Cleanup() error {
	if _, err := s.db.Exec(
		"DELETE FROM auth_tokens WHERE expires_at < ?",
		time.Now(),
	); err != nil {
		return fmt.Errorf("cleaning up tokens: %w", err)
	}
	return nil
}

func generateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

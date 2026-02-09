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
// The token is marked as used atomically and cannot be reused.
func (s *TokenStore) Validate(token string) (string, error) {
	// Atomically mark unused, unexpired token as used and return the email.
	// This avoids a TOCTOU race between SELECT and UPDATE.
	result, err := s.db.Exec(
		"UPDATE auth_tokens SET used = 1 WHERE token = ? AND used = 0 AND expires_at > ?",
		token, time.Now(),
	)
	if err != nil {
		return "", fmt.Errorf("validating token: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return "", fmt.Errorf("checking affected rows: %w", err)
	}
	if rows == 0 {
		return "", fmt.Errorf("invalid, expired, or already used token")
	}

	var email string
	err = s.db.QueryRow(
		"SELECT email FROM auth_tokens WHERE token = ?",
		token,
	).Scan(&email)
	if err != nil {
		return "", fmt.Errorf("reading token email: %w", err)
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

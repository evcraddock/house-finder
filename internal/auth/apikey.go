package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"time"
)

const apiKeyBytes = 32 // 256-bit keys

// APIKey is the stored representation of an API key (no raw key).
type APIKey struct {
	ID         int64
	Name       string
	KeyPrefix  string // first 8 chars for identification
	CreatedAt  time.Time
	LastUsedAt *time.Time
}

// APIKeyStore manages API keys in SQLite.
type APIKeyStore struct {
	db *sql.DB
}

// NewAPIKeyStore creates an API key store.
func NewAPIKeyStore(db *sql.DB) *APIKeyStore {
	return &APIKeyStore{db: db}
}

// Create generates a new API key with the given name.
// Returns the raw key (shown once to user) and the stored record.
func (s *APIKeyStore) Create(name, email string) (string, *APIKey, error) {
	raw, err := generateAPIKey()
	if err != nil {
		return "", nil, fmt.Errorf("generating key: %w", err)
	}

	prefix := raw[:8]
	hash := hashAPIKey(raw)

	result, err := s.db.Exec(
		"INSERT INTO api_keys (name, key_prefix, key_hash, email) VALUES (?, ?, ?, ?)",
		name, prefix, hash, email,
	)
	if err != nil {
		return "", nil, fmt.Errorf("storing key: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return "", nil, fmt.Errorf("getting key id: %w", err)
	}

	key := &APIKey{
		ID:        id,
		Name:      name,
		KeyPrefix: prefix,
	}

	return raw, key, nil
}

// List returns all API keys (without the raw key).
func (s *APIKeyStore) List() ([]APIKey, error) {
	rows, err := s.db.Query(
		"SELECT id, name, key_prefix, created_at, last_used_at FROM api_keys ORDER BY created_at DESC",
	)
	if err != nil {
		return nil, fmt.Errorf("querying keys: %w", err)
	}
	defer func() {
		if cerr := rows.Close(); cerr != nil {
			fmt.Printf("closing rows: %v\n", cerr)
		}
	}()

	var keys []APIKey
	for rows.Next() {
		var k APIKey
		if err := rows.Scan(&k.ID, &k.Name, &k.KeyPrefix, &k.CreatedAt, &k.LastUsedAt); err != nil {
			return nil, fmt.Errorf("scanning key: %w", err)
		}
		keys = append(keys, k)
	}

	return keys, rows.Err()
}

// Delete removes an API key by ID.
func (s *APIKeyStore) Delete(id int64) error {
	result, err := s.db.Exec("DELETE FROM api_keys WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("deleting key: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("checking affected rows: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("key not found")
	}

	return nil
}

// Validate checks a raw API key against stored hashes.
// Returns the owner's email if valid (empty string if invalid), and updates last_used_at.
func (s *APIKeyStore) Validate(rawKey string) (string, error) {
	hash := hashAPIKey(rawKey)

	var email string
	err := s.db.QueryRow(
		"SELECT email FROM api_keys WHERE key_hash = ?", hash,
	).Scan(&email)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("validating key: %w", err)
	}

	if _, err := s.db.Exec(
		"UPDATE api_keys SET last_used_at = ? WHERE key_hash = ?",
		time.Now(), hash,
	); err != nil {
		return "", fmt.Errorf("updating last_used_at: %w", err)
	}

	return email, nil
}

func generateAPIKey() (string, error) {
	b := make([]byte, apiKeyBytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return "hf_" + hex.EncodeToString(b), nil
}

func hashAPIKey(key string) string {
	h := sha256.Sum256([]byte(key))
	return hex.EncodeToString(h[:])
}

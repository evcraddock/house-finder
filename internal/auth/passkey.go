package auth

import (
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/go-webauthn/webauthn/webauthn"
)

// PasskeyUser implements webauthn.User for a single admin email.
type PasskeyUser struct {
	email       string
	credentials []webauthn.Credential
}

// NewPasskeyUser creates a PasskeyUser for the given email.
func NewPasskeyUser(email string, credentials []webauthn.Credential) *PasskeyUser {
	return &PasskeyUser{email: email, credentials: credentials}
}

// WebAuthnID returns a stable user ID derived from the email.
func (u *PasskeyUser) WebAuthnID() []byte {
	h := sha256.Sum256([]byte(u.email))
	return h[:]
}

// WebAuthnName returns the email.
func (u *PasskeyUser) WebAuthnName() string { return u.email }

// WebAuthnDisplayName returns the email.
func (u *PasskeyUser) WebAuthnDisplayName() string { return u.email }

// WebAuthnCredentials returns the stored credentials.
func (u *PasskeyUser) WebAuthnCredentials() []webauthn.Credential { return u.credentials }

// PasskeyStore manages passkey credentials in SQLite.
type PasskeyStore struct {
	db *sql.DB
}

// NewPasskeyStore creates a passkey store.
func NewPasskeyStore(db *sql.DB) *PasskeyStore {
	return &PasskeyStore{db: db}
}

// StoredCredential is a passkey credential with metadata.
type StoredCredential struct {
	ID         string
	Email      string
	Name       string
	Credential webauthn.Credential
}

// Save stores a new passkey credential.
func (s *PasskeyStore) Save(email, name string, cred *webauthn.Credential) error {
	data, err := json.Marshal(cred)
	if err != nil {
		return fmt.Errorf("marshaling credential: %w", err)
	}

	id := fmt.Sprintf("%x", cred.ID)
	if _, err := s.db.Exec(
		"INSERT INTO passkey_credentials (id, email, name, credential_json) VALUES (?, ?, ?, ?)",
		id, email, name, string(data),
	); err != nil {
		return fmt.Errorf("storing credential: %w", err)
	}

	return nil
}

// ListByEmail returns all credentials for the given email.
func (s *PasskeyStore) ListByEmail(email string) ([]StoredCredential, error) {
	rows, err := s.db.Query(
		"SELECT id, email, name, credential_json FROM passkey_credentials WHERE email = ?",
		email,
	)
	if err != nil {
		return nil, fmt.Errorf("querying credentials: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			fmt.Printf("closing rows: %v\n", err)
		}
	}()

	var result []StoredCredential
	for rows.Next() {
		var sc StoredCredential
		var data string
		if err := rows.Scan(&sc.ID, &sc.Email, &sc.Name, &data); err != nil {
			return nil, fmt.Errorf("scanning credential: %w", err)
		}
		if err := json.Unmarshal([]byte(data), &sc.Credential); err != nil {
			return nil, fmt.Errorf("unmarshaling credential: %w", err)
		}
		result = append(result, sc)
	}

	return result, rows.Err()
}

// WebAuthnCredentials returns just the webauthn.Credential slice for the given email.
func (s *PasskeyStore) WebAuthnCredentials(email string) ([]webauthn.Credential, error) {
	stored, err := s.ListByEmail(email)
	if err != nil {
		return nil, err
	}

	creds := make([]webauthn.Credential, len(stored))
	for i, sc := range stored {
		creds[i] = sc.Credential
	}

	return creds, nil
}

// Delete removes a credential by ID.
func (s *PasskeyStore) Delete(id, email string) error {
	result, err := s.db.Exec(
		"DELETE FROM passkey_credentials WHERE id = ? AND email = ?",
		id, email,
	)
	if err != nil {
		return fmt.Errorf("deleting credential: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("checking affected rows: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("credential not found")
	}

	return nil
}

package web

import (
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"

	"github.com/evcraddock/house-finder/internal/auth"
)

// passkeyHandlers holds WebAuthn-related HTTP handlers.
type passkeyHandlers struct {
	wan      *webauthn.WebAuthn
	passkeys *auth.PasskeyStore
	sessions *auth.SessionStore
	config   auth.Config

	// In-memory session data for in-flight WebAuthn ceremonies.
	// Keyed by email for registration, by challenge for login.
	mu               sync.Mutex
	regSessions      map[string]*webauthn.SessionData
	loginSessionData *webauthn.SessionData
}

func newPasskeyHandlers(cfg auth.Config, passkeys *auth.PasskeyStore, sessions *auth.SessionStore) (*passkeyHandlers, error) {
	parsed, err := url.Parse(cfg.BaseURL)
	if err != nil {
		return nil, err
	}

	rpID := parsed.Hostname()

	wan, err := webauthn.New(&webauthn.Config{
		RPDisplayName: "House Finder",
		RPID:          rpID,
		RPOrigins:     []string{cfg.BaseURL},
	})
	if err != nil {
		return nil, err
	}

	return &passkeyHandlers{
		wan:         wan,
		passkeys:    passkeys,
		sessions:    sessions,
		config:      cfg,
		regSessions: make(map[string]*webauthn.SessionData),
	}, nil
}

// handleBeginRegistration starts passkey registration (called from settings page).
// Requires an active session.
func (h *passkeyHandlers) handleBeginRegistration(w http.ResponseWriter, r *http.Request) {
	email, err := h.sessions.Validate(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	creds, err := h.passkeys.WebAuthnCredentials(email)
	if err != nil {
		log.Printf("Error loading credentials: %v", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	user := auth.NewPasskeyUser(email, creds)

	// Exclude existing credentials so user doesn't re-register the same key
	excludeList := make([]protocol.CredentialDescriptor, len(creds))
	for i, c := range creds {
		excludeList[i] = c.Descriptor()
	}

	creation, session, err := h.wan.BeginRegistration(user,
		webauthn.WithExclusions(excludeList),
	)
	if err != nil {
		log.Printf("Error beginning registration: %v", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	h.mu.Lock()
	h.regSessions[email] = session
	h.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(creation); err != nil {
		log.Printf("Error encoding registration options: %v", err)
	}
}

// handleFinishRegistration completes passkey registration.
func (h *passkeyHandlers) handleFinishRegistration(w http.ResponseWriter, r *http.Request) {
	email, err := h.sessions.Validate(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	h.mu.Lock()
	session, ok := h.regSessions[email]
	if ok {
		delete(h.regSessions, email)
	}
	h.mu.Unlock()

	if !ok {
		http.Error(w, "No registration in progress", http.StatusBadRequest)
		return
	}

	creds, err := h.passkeys.WebAuthnCredentials(email)
	if err != nil {
		log.Printf("Error loading credentials: %v", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	user := auth.NewPasskeyUser(email, creds)

	credential, err := h.wan.FinishRegistration(user, *session, r)
	if err != nil {
		log.Printf("Error finishing registration: %v", err)
		http.Error(w, "Registration failed", http.StatusBadRequest)
		return
	}

	// Get name from query param or default
	name := strings.TrimSpace(r.URL.Query().Get("name"))
	if name == "" {
		name = "Passkey"
	}

	if err := h.passkeys.Save(email, name, credential); err != nil {
		log.Printf("Error saving credential: %v", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]string{"status": "ok"}); err != nil {
		log.Printf("Error encoding response: %v", err)
	}
}

// handleBeginLogin starts passkey login (discoverable/conditional).
func (h *passkeyHandlers) handleBeginLogin(w http.ResponseWriter, r *http.Request) {
	assertion, session, err := h.wan.BeginDiscoverableLogin()
	if err != nil {
		log.Printf("Error beginning passkey login: %v", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	h.mu.Lock()
	h.loginSessionData = session
	h.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(assertion); err != nil {
		log.Printf("Error encoding login options: %v", err)
	}
}

// handleFinishLogin completes passkey login and creates a session.
func (h *passkeyHandlers) handleFinishLogin(w http.ResponseWriter, r *http.Request) {
	h.mu.Lock()
	session := h.loginSessionData
	h.loginSessionData = nil
	h.mu.Unlock()

	if session == nil {
		http.Error(w, "No login in progress", http.StatusBadRequest)
		return
	}

	handler := func(rawID, userHandle []byte) (webauthn.User, error) {
		// userHandle is the WebAuthnID (sha256 of email)
		// We only have one admin, so verify it matches
		email := h.config.AdminEmail
		user := auth.NewPasskeyUser(email, nil)

		// Verify the userHandle matches
		if string(user.WebAuthnID()) != string(userHandle) {
			return nil, protocol.ErrBadRequest.WithDetails("unknown user")
		}

		creds, err := h.passkeys.WebAuthnCredentials(email)
		if err != nil {
			return nil, err
		}

		return auth.NewPasskeyUser(email, creds), nil
	}

	_, _, err := h.wan.FinishPasskeyLogin(handler, *session, r)
	if err != nil {
		log.Printf("Error finishing passkey login: %v", err)
		http.Error(w, "Login failed", http.StatusUnauthorized)
		return
	}

	if err := h.sessions.Create(w, h.config.AdminEmail); err != nil {
		log.Printf("Error creating session: %v", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]string{"status": "ok"}); err != nil {
		log.Printf("Error encoding response: %v", err)
	}
}

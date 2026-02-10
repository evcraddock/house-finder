package web

import (
	"encoding/json"
	"log/slog"
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
	users    *auth.UserStore
	config   auth.Config

	// In-memory session data for in-flight WebAuthn ceremonies.
	// regSessions is keyed by email for registration.
	// loginSessionData holds a single login ceremony — only one concurrent
	// passkey login is supported (acceptable for small user base).
	mu               sync.Mutex
	regSessions      map[string]*webauthn.SessionData
	loginSessionData *webauthn.SessionData
}

func newPasskeyHandlers(cfg auth.Config, passkeys *auth.PasskeyStore, sessions *auth.SessionStore, users *auth.UserStore) (*passkeyHandlers, error) {
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
		users:       users,
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
		slog.Error("loading credentials", "err", err)
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
		slog.Error("beginning registration", "err", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	h.mu.Lock()
	h.regSessions[email] = session
	h.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(creation); err != nil {
		slog.Error("encoding registration options", "err", err)
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
		slog.Error("loading credentials", "err", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	user := auth.NewPasskeyUser(email, creds)

	credential, err := h.wan.FinishRegistration(user, *session, r)
	if err != nil {
		slog.Error("finishing registration", "err", err)
		http.Error(w, "Registration failed", http.StatusBadRequest)
		return
	}

	// Get name from query param or default
	name := strings.TrimSpace(r.URL.Query().Get("name"))
	if name == "" {
		name = "Passkey"
	}

	if err := h.passkeys.Save(email, name, credential); err != nil {
		slog.Error("saving credential", "err", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]string{"status": "ok"}); err != nil {
		slog.Error("encoding response", "err", err)
	}
}

// handleBeginLogin starts passkey login (discoverable/conditional).
func (h *passkeyHandlers) handleBeginLogin(w http.ResponseWriter, r *http.Request) {
	assertion, session, err := h.wan.BeginDiscoverableLogin()
	if err != nil {
		slog.Error("beginning passkey login", "err", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	h.mu.Lock()
	h.loginSessionData = session
	h.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(assertion); err != nil {
		slog.Error("encoding login options", "err", err)
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

	var loggedInEmail string

	handler := func(rawID, userHandle []byte) (webauthn.User, error) {
		// userHandle is the WebAuthnID (sha256 of email)
		// Try all authorized emails to find the matching user
		emails, emailErr := h.users.AllEmails()
		if emailErr != nil {
			return nil, emailErr
		}

		for _, email := range emails {
			user := auth.NewPasskeyUser(email, nil)
			if string(user.WebAuthnID()) == string(userHandle) {
				creds, credErr := h.passkeys.WebAuthnCredentials(email)
				if credErr != nil {
					return nil, credErr
				}
				loggedInEmail = email
				return auth.NewPasskeyUser(email, creds), nil
			}
		}

		return nil, protocol.ErrBadRequest.WithDetails("unknown user")
	}

	_, _, err := h.wan.FinishPasskeyLogin(handler, *session, r)
	if err != nil {
		slog.Error("finishing passkey login", "err", err)
		http.Error(w, "Login failed", http.StatusUnauthorized)
		return
	}

	if err := h.sessions.Create(w, loggedInEmail); err != nil {
		slog.Error("creating session", "err", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	slog.Info("login success", "email", loggedInEmail, "method", "passkey")
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]string{"status": "ok"}); err != nil {
		slog.Error("encoding response", "err", err)
	}
}

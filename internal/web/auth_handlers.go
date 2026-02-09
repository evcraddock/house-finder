package web

import (
	"log"
	"net/http"
	"strings"

	"github.com/evcraddock/house-finder/internal/auth"
)

// authHandlers holds auth-related HTTP handlers.
type authHandlers struct {
	config   auth.Config
	tokens   *auth.TokenStore
	sessions *auth.SessionStore
	passkeys *auth.PasskeyStore
	users    *auth.UserStore
	mailer   *auth.Mailer
	render   func(w http.ResponseWriter, name string, data interface{})
}

type loginData struct {
	Message     string
	Error       string
	HasPasskeys bool
}

// handleLoginPage renders the login form.
func (h *authHandlers) handleLoginPage(w http.ResponseWriter, r *http.Request) {
	h.render(w, "login.html", loginData{HasPasskeys: h.hasPasskeys()})
}

// handleLoginSubmit processes the email form submission.
func (h *authHandlers) handleLoginSubmit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	email := strings.TrimSpace(strings.ToLower(r.FormValue("email")))

	// Always show the same message regardless of whether the email is valid.
	// This prevents email enumeration.
	successMsg := "If that email is registered, a login link has been sent. Check your inbox."

	hp := h.hasPasskeys()

	if email == "" {
		h.render(w, "login.html", loginData{Error: "Email is required", HasPasskeys: hp})
		return
	}

	// Only send a real token if email is authorized (admin or in users table)
	if h.users.IsAuthorized(email) {
		token, err := h.tokens.Create(email)
		if err != nil {
			// Log internally but don't reveal to user
			log.Printf("Error creating token: %v\n", err)
			h.render(w, "login.html", loginData{Message: successMsg, HasPasskeys: hp})
			return
		}

		if _, err := h.mailer.SendMagicLink(email, token); err != nil {
			log.Printf("Error sending magic link: %v\n", err)
		}
	}

	h.render(w, "login.html", loginData{Message: successMsg, HasPasskeys: hp})
}

// handleVerify validates a magic link token and creates a session.
func (h *authHandlers) handleVerify(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	hp := h.hasPasskeys()
	if token == "" {
		h.render(w, "login.html", loginData{Error: "Invalid login link", HasPasskeys: hp})
		return
	}

	email, err := h.tokens.Validate(token)
	if err != nil {
		h.render(w, "login.html", loginData{Error: "Invalid or expired login link. Please request a new one.", HasPasskeys: hp})
		return
	}

	if err := h.sessions.Create(w, email); err != nil {
		log.Printf("Error creating session: %v\n", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// handleLogout destroys the session and redirects to login.
func (h *authHandlers) handleLogout(w http.ResponseWriter, r *http.Request) {
	if err := h.sessions.Destroy(w, r); err != nil {
		log.Printf("Error destroying session: %v\n", err)
	}
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

// hasPasskeys checks if any passkeys are registered for any authorized user.
func (h *authHandlers) hasPasskeys() bool {
	if h.passkeys == nil {
		return false
	}
	emails, err := h.users.AllEmails()
	if err != nil {
		return false
	}
	for _, email := range emails {
		creds, cerr := h.passkeys.WebAuthnCredentials(email)
		if cerr == nil && len(creds) > 0 {
			return true
		}
	}
	return false
}

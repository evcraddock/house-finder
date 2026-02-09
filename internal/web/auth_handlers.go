package web

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/evcraddock/house-finder/internal/auth"
)

// authHandlers holds auth-related HTTP handlers.
type authHandlers struct {
	config   auth.Config
	tokens   *auth.TokenStore
	sessions *auth.SessionStore
	mailer   *auth.Mailer
	render   func(w http.ResponseWriter, name string, data interface{})
}

type loginData struct {
	Message string
	Error   string
}

// handleLoginPage renders the login form.
func (h *authHandlers) handleLoginPage(w http.ResponseWriter, r *http.Request) {
	h.render(w, "login.html", loginData{})
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

	if email == "" {
		h.render(w, "login.html", loginData{Error: "Email is required"})
		return
	}

	// Only send a real token if email matches admin
	if email == strings.ToLower(h.config.AdminEmail) {
		token, err := h.tokens.Create(email)
		if err != nil {
			// Log internally but don't reveal to user
			fmt.Printf("Error creating token: %v\n", err)
			h.render(w, "login.html", loginData{Message: successMsg})
			return
		}

		if _, err := h.mailer.SendMagicLink(email, token); err != nil {
			fmt.Printf("Error sending magic link: %v\n", err)
		}
	}

	h.render(w, "login.html", loginData{Message: successMsg})
}

// handleVerify validates a magic link token and creates a session.
func (h *authHandlers) handleVerify(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		h.render(w, "login.html", loginData{Error: "Invalid login link"})
		return
	}

	email, err := h.tokens.Validate(token)
	if err != nil {
		h.render(w, "login.html", loginData{Error: "Invalid or expired login link. Please request a new one."})
		return
	}

	if err := h.sessions.Create(w, email); err != nil {
		fmt.Printf("Error creating session: %v\n", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// handleLogout destroys the session and redirects to login.
func (h *authHandlers) handleLogout(w http.ResponseWriter, r *http.Request) {
	if err := h.sessions.Destroy(w, r); err != nil {
		fmt.Printf("Error destroying session: %v\n", err)
	}
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

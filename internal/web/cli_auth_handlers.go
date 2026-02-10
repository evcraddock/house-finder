package web

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/evcraddock/house-finder/internal/auth"
)

// cliAuthHandlers handles the /cli/auth flow.
type cliAuthHandlers struct {
	config   auth.Config
	tokens   *auth.TokenStore
	sessions *auth.SessionStore
	passkeys *auth.PasskeyStore
	apiKeys  *auth.APIKeyStore
	users    *auth.UserStore
	mailer   *auth.Mailer
	render   func(w http.ResponseWriter, name string, data interface{})
}

type cliAuthData struct {
	APIKey      string
	Message     string
	Error       string
	HasPasskeys bool
}

// handleCLIAuth serves the CLI login page (GET) and processes email submission (POST).
func (h *cliAuthHandlers) handleCLIAuth(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.showLoginForm(w, r)
	case http.MethodPost:
		h.submitEmail(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *cliAuthHandlers) showLoginForm(w http.ResponseWriter, r *http.Request) {
	h.render(w, "cli_auth.html", cliAuthData{HasPasskeys: h.hasPasskeys()})
}

func (h *cliAuthHandlers) submitEmail(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	email := strings.TrimSpace(strings.ToLower(r.FormValue("email")))
	hp := h.hasPasskeys()
	successMsg := "If that email is registered, a login link has been sent. Check your inbox."

	if email == "" {
		h.render(w, "cli_auth.html", cliAuthData{Error: "Email is required", HasPasskeys: hp})
		return
	}

	// Only send token if email is authorized
	if h.users.IsAuthorized(email) {
		token, err := h.tokens.Create(email)
		if err != nil {
			slog.Error("creating token", "err", err)
			h.render(w, "cli_auth.html", cliAuthData{Message: successMsg, HasPasskeys: hp})
			return
		}

		// Magic link redirects back to /cli/auth/complete after verification
		if _, err := h.mailer.SendCLIMagicLink(email, token); err != nil {
			slog.Error("sending magic link", "err", err)
		}
	}

	h.render(w, "cli_auth.html", cliAuthData{Message: successMsg, HasPasskeys: hp})
}

// handleCLIAuthVerify validates the magic link token, creates a session,
// then redirects to /cli/auth/complete.
func (h *cliAuthHandlers) handleCLIAuthVerify(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	hp := h.hasPasskeys()

	if token == "" {
		h.render(w, "cli_auth.html", cliAuthData{Error: "Invalid login link", HasPasskeys: hp})
		return
	}

	email, err := h.tokens.Validate(token)
	if err != nil {
		h.render(w, "cli_auth.html", cliAuthData{Error: "Invalid or expired login link. Please try again.", HasPasskeys: hp})
		return
	}

	if err := h.sessions.Create(w, email); err != nil {
		slog.Error("creating session", "err", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/cli/auth/complete", http.StatusSeeOther)
}

// handleCLIAuthComplete generates an API key and displays it.
// Requires a valid session (user just logged in).
func (h *cliAuthHandlers) handleCLIAuthComplete(w http.ResponseWriter, r *http.Request) {
	email, err := h.sessions.Validate(r)
	if err != nil {
		http.Redirect(w, r, "/cli/auth", http.StatusSeeOther)
		return
	}

	rawKey, _, err := h.apiKeys.Create("CLI", email)
	if err != nil {
		slog.Error("creating api key", "err", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	h.render(w, "cli_auth.html", cliAuthData{APIKey: rawKey})
}

func (h *cliAuthHandlers) hasPasskeys() bool {
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

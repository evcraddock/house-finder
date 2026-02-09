package auth

import (
	"net/http"
	"strings"
)

// RequireAuth is middleware that redirects unauthenticated requests to the login page.
// It skips auth for public paths (login, static assets, auth endpoints).
func RequireAuth(sessions *SessionStore, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if isPublicPath(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		if _, err := sessions.Validate(r); err != nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func isPublicPath(path string) bool {
	if path == "/login" || path == "/auth/login" || path == "/auth/verify" || path == "/auth/logout" {
		return true
	}
	if strings.HasPrefix(path, "/static/") {
		return true
	}
	// Passkey login endpoints must be public (user isn't authenticated yet)
	if path == "/passkey/login/begin" || path == "/passkey/login/finish" {
		return true
	}
	return false
}

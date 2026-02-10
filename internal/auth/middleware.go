package auth

import (
	"context"
	"net/http"
	"strings"
	"sync"
	"time"
)

type contextKey string

const userEmailKey contextKey = "userEmail"

// WithUserEmail adds the authenticated user's email to the request context.
func WithUserEmail(r *http.Request, email string) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), userEmailKey, email))
}

// UserEmailFromContext extracts the authenticated user's email from the request context.
func UserEmailFromContext(r *http.Request) string {
	email, _ := r.Context().Value(userEmailKey).(string)
	return email
}

// RequireAuth is middleware that redirects unauthenticated web requests to the login page.
// It skips auth for public paths (login, static assets, auth endpoints).
// API paths (/api/...) are handled separately by RequireAPIKey.
func RequireAuth(sessions *SessionStore, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if isPublicPath(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		// API paths skip session auth — they use RequireAPIKey instead
		if strings.HasPrefix(r.URL.Path, "/api/") {
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

// rateLimiter tracks failed API key attempts per IP.
type rateLimiter struct {
	mu       sync.Mutex
	attempts map[string][]time.Time
}

var apiKeyLimiter = &rateLimiter{
	attempts: make(map[string][]time.Time),
}

const (
	rateLimitWindow  = 1 * time.Minute
	rateLimitMaxFail = 10
)

// isLimited returns true if the IP has exceeded the rate limit.
func (rl *rateLimiter) isLimited(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	cutoff := time.Now().Add(-rateLimitWindow)
	valid := rl.attempts[ip][:0]
	for _, t := range rl.attempts[ip] {
		if t.After(cutoff) {
			valid = append(valid, t)
		}
	}
	rl.attempts[ip] = valid

	return len(valid) >= rateLimitMaxFail
}

// recordFailure records a failed attempt for the IP.
func (rl *rateLimiter) recordFailure(ip string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	rl.attempts[ip] = append(rl.attempts[ip], time.Now())
}

// RequireAPIKey is middleware that validates Bearer token auth for /api/ routes.
// Non-API routes pass through untouched. API key management paths (/api/keys)
// require session auth instead of bearer tokens.
// Returns 401 for missing/invalid keys, 429 for rate-limited IPs.
func RequireAPIKey(apiKeys *APIKeyStore, sessions *SessionStore, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Only intercept /api/ paths
		if !strings.HasPrefix(r.URL.Path, "/api/") {
			next.ServeHTTP(w, r)
			return
		}

		// API key management endpoints require session auth (web UI), not bearer tokens
		if isAPIKeyManagementPath(r.URL.Path) {
			if _, err := sessions.Validate(r); err != nil {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
			return
		}

		ip := r.RemoteAddr

		authHeader := r.Header.Get("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			// No bearer token — fall back to session auth (for web UI fetch calls)
			if email, err := sessions.Validate(r); err == nil {
				next.ServeHTTP(w, WithUserEmail(r, email))
				return
			}
			http.Error(w, "Authorization required", http.StatusUnauthorized)
			return
		}

		key := strings.TrimPrefix(authHeader, "Bearer ")

		// Check rate limit before expensive validation
		if apiKeyLimiter.isLimited(ip) {
			http.Error(w, "Too many requests", http.StatusTooManyRequests)
			return
		}

		email, err := apiKeys.Validate(key)
		if err != nil {
			http.Error(w, "Internal error", http.StatusInternalServerError)
			return
		}
		if email == "" {
			apiKeyLimiter.recordFailure(ip)
			http.Error(w, "Invalid API key", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, WithUserEmail(r, email))
	})
}

func isPublicPath(path string) bool {
	if path == "/health" {
		return true
	}
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
	// CLI auth pages are public (user authenticates through them)
	if path == "/cli/auth" || path == "/cli/auth/verify" || path == "/cli/auth/complete" {
		return true
	}
	return false
}

func isAPIKeyManagementPath(path string) bool {
	return path == "/api/keys" || strings.HasPrefix(path, "/api/keys/") ||
		path == "/api/users" || strings.HasPrefix(path, "/api/users/")
}

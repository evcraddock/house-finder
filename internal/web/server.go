// Package web provides the HTTP server and handlers for the house-finder web UI.
package web

import (
	"database/sql"
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"strings"

	"github.com/evcraddock/house-finder/internal/auth"
	"github.com/evcraddock/house-finder/internal/comment"
	"github.com/evcraddock/house-finder/internal/property"
)

//go:embed templates/*.html
var templateFS embed.FS

//go:embed static/*
var staticFS embed.FS

// Server is the web UI HTTP server.
type Server struct {
	propRepo    *property.Repository
	commentRepo *comment.Repository
	sessions    *auth.SessionStore
	passkeys    *auth.PasskeyStore
	apiKeys     *auth.APIKeyStore
	templates   *template.Template
	handler     http.Handler
}

// NewServer creates a web server with the given database and auth config.
func NewServer(db *sql.DB, authCfg auth.Config) (*Server, error) {
	funcMap := template.FuncMap{
		"formatPrice":  tmplFormatPrice,
		"formatFloat":  tmplFormatFloat,
		"formatInt":    tmplFormatInt,
		"formatStr":    tmplFormatStr,
		"formatLot":    tmplFormatLot,
		"formatRating": tmplFormatRating,
		"derefRating":  tmplDerefRating,
		"seq":          tmplSeq,
	}

	tmpl, err := template.New("").Funcs(funcMap).ParseFS(templateFS, "templates/*.html")
	if err != nil {
		return nil, fmt.Errorf("parsing templates: %w", err)
	}

	tokens := auth.NewTokenStore(db)
	sessions := auth.NewSessionStore(db, !authCfg.DevMode)
	passkeys := auth.NewPasskeyStore(db)
	apiKeys := auth.NewAPIKeyStore(db)
	mailer := auth.NewMailer(authCfg)

	s := &Server{
		propRepo:    property.NewRepository(db),
		commentRepo: comment.NewRepository(db),
		sessions:    sessions,
		passkeys:    passkeys,
		apiKeys:     apiKeys,
		templates:   tmpl,
	}

	mux := http.NewServeMux()

	staticContent, err := fs.Sub(staticFS, "static")
	if err != nil {
		return nil, fmt.Errorf("creating static sub-fs: %w", err)
	}

	// Auth handlers
	ah := &authHandlers{
		config:   authCfg,
		tokens:   tokens,
		sessions: sessions,
		passkeys: passkeys,
		mailer:   mailer,
		render:   s.render,
	}

	// Public routes
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticContent))))
	mux.HandleFunc("/login", ah.handleLoginPage)
	mux.HandleFunc("/auth/login", ah.handleLoginSubmit)
	mux.HandleFunc("/auth/verify", ah.handleVerify)
	mux.HandleFunc("/auth/logout", ah.handleLogout)

	// Passkey routes (login endpoints are public, registration requires session)
	if authCfg.AdminEmail != "" {
		ph, phErr := newPasskeyHandlers(authCfg, passkeys, sessions)
		if phErr != nil {
			return nil, fmt.Errorf("creating passkey handlers: %w", phErr)
		}
		mux.HandleFunc("/passkey/login/begin", ph.handleBeginLogin)
		mux.HandleFunc("/passkey/login/finish", ph.handleFinishLogin)
		mux.HandleFunc("/passkey/register/begin", ph.handleBeginRegistration)
		mux.HandleFunc("/passkey/register/finish", ph.handleFinishRegistration)
	}

	// API key management routes (session-protected via RequireAPIKey middleware)
	akh := &apikeyHandlers{apiKeys: apiKeys}
	mux.HandleFunc("/api/keys", akh.handleAPIKeysRoute)
	mux.HandleFunc("/api/keys/", akh.handleAPIKeysRoute)

	// Protected routes
	mux.HandleFunc("/", s.handleList)
	mux.HandleFunc("/property/", s.handlePropertyRoute)
	mux.HandleFunc("/settings", s.handleSettings)
	mux.HandleFunc("/settings/passkey/delete", s.handlePasskeyDelete)

	// Wrap everything with auth middleware if admin email is configured
	if authCfg.AdminEmail != "" {
		// Web routes: session auth. API routes: bearer token or session for management.
		webAuth := auth.RequireAuth(sessions, mux)
		s.handler = auth.RequireAPIKey(apiKeys, sessions, webAuth)
	} else {
		s.handler = mux
	}

	return s, nil
}

// ServeHTTP implements http.Handler.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.handler.ServeHTTP(w, r)
}

// ListenAndServe starts the HTTP server.
func (s *Server) ListenAndServe(port int) error {
	addr := fmt.Sprintf(":%d", port)
	fmt.Printf("Starting web UI on http://localhost%s\n", addr)
	return http.ListenAndServe(addr, s)
}

// handlePropertyRoute routes /property/{id}/* requests.
func (s *Server) handlePropertyRoute(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/property/")

	if strings.HasSuffix(path, "/comment") {
		s.handleCommentPost(w, r)
		return
	}
	if strings.HasSuffix(path, "/rate") {
		s.handleRatePost(w, r)
		return
	}

	s.handleDetail(w, r)
}

// Template helper functions

func tmplFormatPrice(p *int64) string {
	if p == nil {
		return "—"
	}
	return "$" + formatWithCommas(*p)
}

func tmplFormatFloat(f *float64) string {
	if f == nil {
		return "—"
	}
	if *f == float64(int64(*f)) {
		return fmt.Sprintf("%d", int64(*f))
	}
	return fmt.Sprintf("%.1f", *f)
}

func tmplFormatInt(i *int64) string {
	if i == nil {
		return "—"
	}
	return formatWithCommas(*i)
}

func tmplFormatStr(s *string) string {
	if s == nil {
		return "—"
	}
	return *s
}

func tmplFormatLot(f *float64) string {
	if f == nil {
		return "—"
	}
	return fmt.Sprintf("%.2f acres", *f)
}

func tmplFormatRating(r *int64) string {
	if r == nil {
		return "—"
	}
	return strings.Repeat("★", int(*r)) + strings.Repeat("☆", 4-int(*r))
}

func tmplDerefRating(r *int64) int {
	if r == nil {
		return 0
	}
	return int(*r)
}

func tmplSeq(start, end int) []int {
	var s []int
	for i := start; i <= end; i++ {
		s = append(s, i)
	}
	return s
}

func formatWithCommas(n int64) string {
	s := fmt.Sprintf("%d", n)
	if len(s) <= 3 {
		return s
	}
	var parts []string
	for len(s) > 3 {
		parts = append([]string{s[len(s)-3:]}, parts...)
		s = s[:len(s)-3]
	}
	parts = append([]string{s}, parts...)
	return strings.Join(parts, ",")
}

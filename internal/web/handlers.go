package web

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/evcraddock/house-finder/internal/property"
)

type listData struct {
	Properties    []*property.Property
	IsAdmin       bool
	Tab           string // "not-visited", "scheduled", or "visited"
	NotVisitedCnt int
	ScheduledCnt  int
	VisitedCnt    int
}

type detailData struct {
	Property *property.Property
	Comments interface{}
	Visits   interface{}
	IsAdmin  bool
}

// handleList renders the property list page.
func (s *Server) handleList(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	tab := r.URL.Query().Get("tab")
	if tab != "scheduled" && tab != "visited" {
		tab = "not-visited"
	}

	// Get counts for all three tabs
	notVisitedProps, err := s.propRepo.List(property.ListOptions{Status: property.StatusNotVisited})
	if err != nil {
		http.Error(w, fmt.Sprintf("Error loading properties: %v", err), http.StatusInternalServerError)
		return
	}
	scheduledProps, err := s.propRepo.List(property.ListOptions{Status: property.StatusScheduled})
	if err != nil {
		http.Error(w, fmt.Sprintf("Error loading properties: %v", err), http.StatusInternalServerError)
		return
	}
	visitedProps, err := s.propRepo.List(property.ListOptions{Status: property.StatusVisited})
	if err != nil {
		http.Error(w, fmt.Sprintf("Error loading properties: %v", err), http.StatusInternalServerError)
		return
	}

	var props []*property.Property
	switch tab {
	case "scheduled":
		props = scheduledProps
	case "visited":
		props = visitedProps
	default:
		props = notVisitedProps
	}

	email, sessionErr := s.sessions.Validate(r)
	isAdmin := sessionErr == nil && s.users.IsAdmin(email)
	s.render(w, "list.html", listData{
		Properties:    props,
		IsAdmin:       isAdmin,
		Tab:           tab,
		NotVisitedCnt: len(notVisitedProps),
		ScheduledCnt:  len(scheduledProps),
		VisitedCnt:    len(visitedProps),
	})
}

// handleDetail renders the property detail page.
func (s *Server) handleDetail(w http.ResponseWriter, r *http.Request) {
	id, err := parsePropertyID(r.URL.Path, "")
	if err != nil {
		http.NotFound(w, r)
		return
	}

	prop, err := s.propRepo.GetByID(id)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	comments, err := s.commentRepo.ListByPropertyID(id)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error loading comments: %v", err), http.StatusInternalServerError)
		return
	}

	detailEmail, detailSessionErr := s.sessions.Validate(r)
	detailIsAdmin := detailSessionErr == nil && s.users.IsAdmin(detailEmail)
	s.render(w, "detail.html", detailData{Property: prop, Comments: comments, IsAdmin: detailIsAdmin})
}

// handleCommentPost adds a comment via HTMX or form POST.
func (s *Server) handleCommentPost(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	id, err := parsePropertyID(r.URL.Path, "/comment")
	if err != nil {
		http.NotFound(w, r)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	text := strings.TrimSpace(r.FormValue("text"))
	if text == "" {
		http.Error(w, "Comment text is required", http.StatusBadRequest)
		return
	}

	author, sessionErr := s.sessions.Validate(r)
	if sessionErr != nil {
		author = ""
	}
	if _, err := s.commentRepo.Add(id, text, author); err != nil {
		http.Error(w, fmt.Sprintf("Error adding comment: %v", err), http.StatusInternalServerError)
		return
	}

	// If HTMX request, return just the comments list partial
	if r.Header.Get("HX-Request") == "true" {
		prop, err := s.propRepo.GetByID(id)
		if err != nil {
			http.Error(w, "Error loading property", http.StatusInternalServerError)
			return
		}
		comments, err := s.commentRepo.ListByPropertyID(id)
		if err != nil {
			http.Error(w, "Error loading comments", http.StatusInternalServerError)
			return
		}
		s.renderPartial(w, "comments-partial", detailData{Property: prop, Comments: comments})
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/property/%d", id), http.StatusSeeOther)
}

// handleRatePost sets a rating via HTMX or form POST.
func (s *Server) handleRatePost(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	id, err := parsePropertyID(r.URL.Path, "/rate")
	if err != nil {
		http.NotFound(w, r)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	rating, err := strconv.Atoi(r.FormValue("rating"))
	if err != nil || rating < 1 || rating > 4 {
		http.Error(w, "Rating must be 1-4", http.StatusBadRequest)
		return
	}

	if err := s.propRepo.UpdateRating(id, rating); err != nil {
		http.Error(w, fmt.Sprintf("Error updating rating: %v", err), http.StatusInternalServerError)
		return
	}

	// If HTMX request, return just the rating card partial
	if r.Header.Get("HX-Request") == "true" {
		prop, err := s.propRepo.GetByID(id)
		if err != nil {
			http.Error(w, "Error loading property", http.StatusInternalServerError)
			return
		}
		s.renderPartial(w, "rating-partial", detailData{Property: prop})
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/property/%d", id), http.StatusSeeOther)
}

// handleAdminUsers renders the admin user management page.
func (s *Server) handleAdminUsers(w http.ResponseWriter, r *http.Request) {
	email, err := s.sessions.Validate(r)
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	if !s.users.IsAdmin(email) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	s.render(w, "admin_users.html", nil)
}

// handleSettings renders the settings page with passkey management.
func (s *Server) handleSettings(w http.ResponseWriter, r *http.Request) {
	email, err := s.sessions.Validate(r)
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	stored, err := s.passkeys.ListByEmail(email)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error loading passkeys: %v", err), http.StatusInternalServerError)
		return
	}

	type passkeyItem struct {
		ID   string
		Name string
	}
	type settingsData struct {
		Passkeys []passkeyItem
		Flash    string
		IsAdmin  bool
	}

	passkeys := make([]passkeyItem, len(stored))
	for i, sc := range stored {
		passkeys[i] = passkeyItem{ID: sc.ID, Name: sc.Name}
	}

	s.render(w, "settings.html", settingsData{
		Passkeys: passkeys,
		IsAdmin:  s.users.IsAdmin(email),
	})
}

// handlePasskeyDelete proxies to the passkey handler delete endpoint.
func (s *Server) handlePasskeyDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	email, err := s.sessions.Validate(r)
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	id := r.FormValue("id")
	if id == "" {
		http.Error(w, "Missing credential ID", http.StatusBadRequest)
		return
	}

	if err := s.passkeys.Delete(id, email); err != nil {
		http.Error(w, fmt.Sprintf("Error deleting passkey: %v", err), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/settings", http.StatusSeeOther)
}

// render executes a full page template with layout.
func (s *Server) render(w http.ResponseWriter, name string, data interface{}) {
	// Execute layout.html which includes blocks from the named template
	if err := s.templates.ExecuteTemplate(w, name, data); err != nil {
		http.Error(w, fmt.Sprintf("Error rendering template: %v", err), http.StatusInternalServerError)
	}
}

// renderPartial executes a named template block (no layout).
func (s *Server) renderPartial(w http.ResponseWriter, name string, data interface{}) {
	if err := s.templates.ExecuteTemplate(w, name, data); err != nil {
		http.Error(w, fmt.Sprintf("Error rendering partial: %v", err), http.StatusInternalServerError)
	}
}

// parsePropertyID extracts the property ID from a URL path like /property/123 or /property/123/suffix.
func parsePropertyID(path, suffix string) (int64, error) {
	s := strings.TrimPrefix(path, "/property/")
	if suffix != "" {
		s = strings.TrimSuffix(s, suffix)
	}
	return strconv.ParseInt(s, 10, 64)
}

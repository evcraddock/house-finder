package web

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/evcraddock/house-finder/internal/email"
	"github.com/evcraddock/house-finder/internal/property"
)

type emailRequest struct {
	PropertyIDs []int64 `json:"property_ids"` // specific IDs (optional)
	MinRating   *int    `json:"min_rating"`   // filter by min rating (optional)
	Visited     *bool   `json:"visited"`      // filter by visit status (optional)
	DryRun      bool    `json:"dry_run"`      // preview only, don't send
}

type emailResponse struct {
	Sent    bool     `json:"sent"`
	To      []string `json:"to"`
	Subject string   `json:"subject"`
	Body    string   `json:"body"`
}

// handleAPIEmail handles POST /api/email.
func (s *Server) handleAPIEmail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		apiError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req emailRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiError(w, "invalid JSON body", http.StatusBadRequest)
		return
	}

	// Find realtor recipients
	allUsers, err := s.users.List()
	if err != nil {
		apiError(w, fmt.Sprintf("listing users: %v", err), http.StatusInternalServerError)
		return
	}
	var recipients []string
	for _, u := range allUsers {
		if u.IsRealtor && u.Email != "" {
			recipients = append(recipients, u.Email)
		}
	}
	if len(recipients) == 0 {
		apiError(w, "no realtors configured — add a user with the realtor flag", http.StatusBadRequest)
		return
	}

	// Gather properties — default to not-visited only
	var props []*property.Property
	if len(req.PropertyIDs) > 0 {
		for _, id := range req.PropertyIDs {
			p, getErr := s.propRepo.GetByID(id)
			if getErr != nil {
				apiError(w, fmt.Sprintf("property %d not found", id), http.StatusNotFound)
				return
			}
			props = append(props, p)
		}
	} else {
		opts := property.ListOptions{
			MinRating: req.MinRating,
			Visited:   req.Visited,
		}
		// Default to not-visited if no visit filter specified
		if opts.Visited == nil {
			notVisited := false
			opts.Visited = &notVisited
		}
		listed, listErr := s.propRepo.List(opts)
		if listErr != nil {
			apiError(w, fmt.Sprintf("listing properties: %v", listErr), http.StatusInternalServerError)
			return
		}
		props = listed
	}

	if len(props) == 0 {
		apiError(w, "no properties match the criteria", http.StatusBadRequest)
		return
	}

	// Gather comments for each property
	var pwc []email.PropertyWithComments
	for _, p := range props {
		comments, commentErr := s.commentRepo.ListByPropertyID(p.ID)
		if commentErr != nil {
			apiError(w, fmt.Sprintf("loading comments for property %d: %v", p.ID, commentErr), http.StatusInternalServerError)
			return
		}
		pwc = append(pwc, email.PropertyWithComments{Property: p, Comments: comments})
	}

	subject := fmt.Sprintf("Properties to visit (%d)", len(props))
	body := email.FormatEmail(pwc, s.authCfg.BaseURL)

	resp := emailResponse{
		To:      recipients,
		Subject: subject,
		Body:    body,
	}

	if req.DryRun {
		resp.Sent = false
		apiJSON(w, resp, http.StatusOK)
		return
	}

	if !s.smtpCfg.IsConfigured() {
		apiError(w, "SMTP not configured — set HF_SMTP_HOST and HF_SMTP_FROM", http.StatusServiceUnavailable)
		return
	}

	if sendErr := email.Send(s.smtpCfg, recipients, subject, body); sendErr != nil {
		apiError(w, fmt.Sprintf("sending email: %v", sendErr), http.StatusInternalServerError)
		return
	}

	resp.Sent = true
	apiJSON(w, resp, http.StatusOK)
}

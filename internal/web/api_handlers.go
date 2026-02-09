package web

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/evcraddock/house-finder/internal/auth"
	"github.com/evcraddock/house-finder/internal/property"
	"github.com/evcraddock/house-finder/internal/visit"
)

// apiError writes a JSON error response.
func apiError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	resp := map[string]string{"error": msg}
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, `{"error":"encode failed"}`, http.StatusInternalServerError)
	}
}

// apiJSON writes a JSON response with the given status code.
func apiJSON(w http.ResponseWriter, data interface{}, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		http.Error(w, `{"error":"encode failed"}`, http.StatusInternalServerError)
	}
}

// handleAPIProperties routes /api/properties requests.
func (s *Server) handleAPIProperties(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/properties")
	path = strings.TrimPrefix(path, "/")

	// /api/properties — list or add
	if path == "" {
		switch r.Method {
		case http.MethodGet:
			s.apiListProperties(w, r)
		case http.MethodPost:
			s.apiAddProperty(w, r)
		default:
			apiError(w, "method not allowed", http.StatusMethodNotAllowed)
		}
		return
	}

	// /api/properties/{id}/comments
	if strings.HasSuffix(path, "/comments") {
		idStr := strings.TrimSuffix(path, "/comments")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			apiError(w, "invalid property ID", http.StatusBadRequest)
			return
		}
		switch r.Method {
		case http.MethodGet:
			s.apiListComments(w, id)
		case http.MethodPost:
			s.apiAddComment(w, r, id)
		default:
			apiError(w, "method not allowed", http.StatusMethodNotAllowed)
		}
		return
	}

	// /api/properties/{id}/visits
	if strings.HasSuffix(path, "/visits") {
		idStr := strings.TrimSuffix(path, "/visits")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			apiError(w, "invalid property ID", http.StatusBadRequest)
			return
		}
		switch r.Method {
		case http.MethodGet:
			s.apiListVisits(w, id)
		case http.MethodPost:
			s.apiAddVisit(w, r, id)
		default:
			apiError(w, "method not allowed", http.StatusMethodNotAllowed)
		}
		return
	}

	// /api/properties/{id}/rate
	if strings.HasSuffix(path, "/rate") {
		idStr := strings.TrimSuffix(path, "/rate")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			apiError(w, "invalid property ID", http.StatusBadRequest)
			return
		}
		if r.Method != http.MethodPost {
			apiError(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		s.apiRateProperty(w, r, id)
		return
	}

	// /api/properties/{id} — show or remove
	id, err := strconv.ParseInt(path, 10, 64)
	if err != nil {
		apiError(w, "invalid property ID", http.StatusBadRequest)
		return
	}
	switch r.Method {
	case http.MethodGet:
		s.apiGetProperty(w, id)
	case http.MethodDelete:
		s.apiDeleteProperty(w, id)
	default:
		apiError(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// apiListProperties returns all properties as JSON.
func (s *Server) apiListProperties(w http.ResponseWriter, r *http.Request) {
	opts := property.ListOptions{}
	if minStr := r.URL.Query().Get("min_rating"); minStr != "" {
		min, err := strconv.Atoi(minStr)
		if err != nil || min < 1 || min > 4 {
			apiError(w, "min_rating must be 1-4", http.StatusBadRequest)
			return
		}
		opts.MinRating = &min
	}
	if visitedStr := r.URL.Query().Get("visited"); visitedStr != "" {
		switch visitedStr {
		case "true":
			v := true
			opts.Visited = &v
		case "false":
			v := false
			opts.Visited = &v
		default:
			apiError(w, "visited must be true or false", http.StatusBadRequest)
			return
		}
	}

	props, err := s.propRepo.List(opts)
	if err != nil {
		apiError(w, fmt.Sprintf("listing properties: %v", err), http.StatusInternalServerError)
		return
	}

	apiJSON(w, props, http.StatusOK)
}

// apiAddProperty adds a property by address (does API lookup).
func (s *Server) apiAddProperty(w http.ResponseWriter, r *http.Request) {
	if s.propService == nil {
		apiError(w, "property add not available (RAPIDAPI_KEY not configured)", http.StatusServiceUnavailable)
		return
	}

	var req struct {
		Address string `json:"address"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiError(w, "invalid JSON body", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(req.Address) == "" {
		apiError(w, "address is required", http.StatusBadRequest)
		return
	}

	p, err := s.propService.Add(strings.TrimSpace(req.Address))
	if err != nil {
		apiError(w, fmt.Sprintf("adding property: %v", err), http.StatusInternalServerError)
		return
	}

	apiJSON(w, p, http.StatusCreated)
}

// apiGetProperty returns a single property with comments and visits.
func (s *Server) apiGetProperty(w http.ResponseWriter, id int64) {
	p, err := s.propRepo.GetByID(id)
	if err != nil {
		apiError(w, "property not found", http.StatusNotFound)
		return
	}

	comments, err := s.commentRepo.ListByPropertyID(id)
	if err != nil {
		apiError(w, fmt.Sprintf("loading comments: %v", err), http.StatusInternalServerError)
		return
	}

	visits, err := s.visitRepo.ListByPropertyID(id)
	if err != nil {
		apiError(w, fmt.Sprintf("loading visits: %v", err), http.StatusInternalServerError)
		return
	}

	type response struct {
		Property *property.Property `json:"property"`
		Comments interface{}        `json:"comments"`
		Visits   interface{}        `json:"visits"`
	}

	apiJSON(w, response{Property: p, Comments: comments, Visits: visits}, http.StatusOK)
}

// apiDeleteProperty removes a property and its comments.
func (s *Server) apiDeleteProperty(w http.ResponseWriter, id int64) {
	if err := s.propRepo.Delete(id); err != nil {
		apiError(w, fmt.Sprintf("deleting property: %v", err), http.StatusInternalServerError)
		return
	}
	apiJSON(w, map[string]interface{}{"id": id, "removed": true}, http.StatusOK)
}

// apiRateProperty sets a rating on a property.
func (s *Server) apiRateProperty(w http.ResponseWriter, r *http.Request, id int64) {
	var req struct {
		Rating int `json:"rating"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiError(w, "invalid JSON body", http.StatusBadRequest)
		return
	}
	if req.Rating < 1 || req.Rating > 4 {
		apiError(w, "rating must be 1-4", http.StatusBadRequest)
		return
	}

	if err := s.propRepo.UpdateRating(id, req.Rating); err != nil {
		apiError(w, fmt.Sprintf("updating rating: %v", err), http.StatusInternalServerError)
		return
	}

	apiJSON(w, map[string]interface{}{"id": id, "rating": req.Rating}, http.StatusOK)
}

// apiListComments returns comments for a property.
func (s *Server) apiListComments(w http.ResponseWriter, id int64) {
	comments, err := s.commentRepo.ListByPropertyID(id)
	if err != nil {
		apiError(w, fmt.Sprintf("loading comments: %v", err), http.StatusInternalServerError)
		return
	}
	apiJSON(w, comments, http.StatusOK)
}

// apiAddComment adds a comment to a property.
func (s *Server) apiAddComment(w http.ResponseWriter, r *http.Request, id int64) {
	var req struct {
		Text string `json:"text"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiError(w, "invalid JSON body", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(req.Text) == "" {
		apiError(w, "text is required", http.StatusBadRequest)
		return
	}

	author := auth.UserEmailFromContext(r)
	c, err := s.commentRepo.Add(id, strings.TrimSpace(req.Text), author)
	if err != nil {
		apiError(w, fmt.Sprintf("adding comment: %v", err), http.StatusInternalServerError)
		return
	}

	apiJSON(w, c, http.StatusCreated)
}

// apiListVisits returns visits for a property.
func (s *Server) apiListVisits(w http.ResponseWriter, id int64) {
	visits, err := s.visitRepo.ListByPropertyID(id)
	if err != nil {
		apiError(w, fmt.Sprintf("listing visits: %v", err), http.StatusInternalServerError)
		return
	}

	if visits == nil {
		visits = make([]*visit.Visit, 0)
	}

	apiJSON(w, visits, http.StatusOK)
}

// apiAddVisit records a visit to a property.
func (s *Server) apiAddVisit(w http.ResponseWriter, r *http.Request, id int64) {
	var req struct {
		VisitDate string `json:"visit_date"`
		VisitType string `json:"visit_type"`
		Notes     string `json:"notes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiError(w, "invalid JSON body", http.StatusBadRequest)
		return
	}
	if req.VisitDate == "" {
		apiError(w, "visit_date is required (YYYY-MM-DD)", http.StatusBadRequest)
		return
	}
	if req.VisitType == "" {
		apiError(w, "visit_type is required (showing, drive_by, open_house)", http.StatusBadRequest)
		return
	}

	v, err := s.visitRepo.Add(id, req.VisitDate, visit.VisitType(req.VisitType), req.Notes)
	if err != nil {
		if strings.Contains(err.Error(), "invalid") {
			apiError(w, err.Error(), http.StatusBadRequest)
			return
		}
		apiError(w, fmt.Sprintf("adding visit: %v", err), http.StatusInternalServerError)
		return
	}

	apiJSON(w, v, http.StatusCreated)
}

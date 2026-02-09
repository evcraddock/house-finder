package web

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/evcraddock/house-finder/internal/auth"
)

// userHandlers manages authorized users (admin-only).
type userHandlers struct {
	users    *auth.UserStore
	sessions *auth.SessionStore
}

// handleUsersRoute routes /api/users requests.
func (h *userHandlers) handleUsersRoute(w http.ResponseWriter, r *http.Request) {
	// Require session auth (admin only)
	email, err := h.sessions.Validate(r)
	if err != nil {
		apiError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	if !h.users.IsAdmin(email) {
		apiError(w, "admin access required", http.StatusForbidden)
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/api/users")
	path = strings.TrimPrefix(path, "/")

	if path == "" {
		switch r.Method {
		case http.MethodGet:
			h.listUsers(w)
		case http.MethodPost:
			h.addUser(w, r)
		default:
			apiError(w, "method not allowed", http.StatusMethodNotAllowed)
		}
		return
	}

	// /api/users/{id}
	id, parseErr := strconv.ParseInt(path, 10, 64)
	if parseErr != nil {
		apiError(w, "invalid user ID", http.StatusBadRequest)
		return
	}

	if r.Method == http.MethodDelete {
		h.deleteUser(w, id)
		return
	}

	apiError(w, "method not allowed", http.StatusMethodNotAllowed)
}

func (h *userHandlers) listUsers(w http.ResponseWriter) {
	users, err := h.users.List()
	if err != nil {
		apiError(w, "listing users: "+err.Error(), http.StatusInternalServerError)
		return
	}
	apiJSON(w, users, http.StatusOK)
}

func (h *userHandlers) addUser(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email string `json:"email"`
		Name  string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiError(w, "invalid JSON body", http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(req.Email) == "" {
		apiError(w, "email is required", http.StatusBadRequest)
		return
	}

	user, err := h.users.Add(req.Email, req.Name)
	if err != nil {
		if strings.Contains(err.Error(), "already exists") {
			apiError(w, err.Error(), http.StatusConflict)
			return
		}
		apiError(w, "adding user: "+err.Error(), http.StatusInternalServerError)
		return
	}

	apiJSON(w, user, http.StatusCreated)
}

func (h *userHandlers) deleteUser(w http.ResponseWriter, id int64) {
	if err := h.users.Delete(id); err != nil {
		if strings.Contains(err.Error(), "not found") {
			apiError(w, "user not found", http.StatusNotFound)
			return
		}
		apiError(w, "deleting user: "+err.Error(), http.StatusInternalServerError)
		return
	}

	apiJSON(w, map[string]interface{}{"id": id, "deleted": true}, http.StatusOK)
}

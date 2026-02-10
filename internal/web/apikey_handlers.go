package web

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/evcraddock/house-finder/internal/auth"
)

// apikeyHandlers holds API key management HTTP handlers.
type apikeyHandlers struct {
	apiKeys  *auth.APIKeyStore
	sessions *auth.SessionStore
}

type apiKeyResponse struct {
	ID         int64   `json:"id"`
	Name       string  `json:"name"`
	KeyPrefix  string  `json:"key_prefix"`
	CreatedAt  string  `json:"created_at"`
	LastUsedAt *string `json:"last_used_at,omitempty"`
}

type apiKeyCreateResponse struct {
	Key            string         `json:"key"` // raw key, shown once
	APIKeyResponse apiKeyResponse `json:"api_key"`
}

// handleCreateKey generates a new API key.
func (h *apikeyHandlers) handleCreateKey(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var body struct {
		Name string `json:"name"`
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	name := strings.TrimSpace(body.Name)
	if name == "" {
		name = "API Key"
	}

	email, sessionErr := h.sessions.Validate(r)
	if sessionErr != nil {
		email = "" // key will have no owner if session is missing
	}
	rawKey, key, err := h.apiKeys.Create(name, email)
	if err != nil {
		slog.Error("creating api key", "err", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	resp := apiKeyCreateResponse{
		Key: rawKey,
		APIKeyResponse: apiKeyResponse{
			ID:        key.ID,
			Name:      key.Name,
			KeyPrefix: key.KeyPrefix,
			CreatedAt: key.CreatedAt.Format("2006-01-02T15:04:05Z"),
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		slog.Error("encoding response", "err", err)
	}
}

// handleListKeys returns all API keys (without raw keys).
func (h *apikeyHandlers) handleListKeys(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	keys, err := h.apiKeys.List()
	if err != nil {
		slog.Error("listing api keys", "err", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	resp := make([]apiKeyResponse, len(keys))
	for i, k := range keys {
		resp[i] = apiKeyResponse{
			ID:        k.ID,
			Name:      k.Name,
			KeyPrefix: k.KeyPrefix,
			CreatedAt: k.CreatedAt.Format("2006-01-02T15:04:05Z"),
		}
		if k.LastUsedAt != nil {
			s := k.LastUsedAt.Format("2006-01-02T15:04:05Z")
			resp[i].LastUsedAt = &s
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		slog.Error("encoding response", "err", err)
	}
}

// handleDeleteKey revokes an API key.
func (h *apikeyHandlers) handleDeleteKey(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract ID from /api/keys/{id}
	idStr := strings.TrimPrefix(r.URL.Path, "/api/keys/")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid key ID", http.StatusBadRequest)
		return
	}

	if err := h.apiKeys.Delete(id); err != nil {
		slog.Error("deleting api key", "err", err)
		http.Error(w, "Key not found", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// handleAPIKeysRoute routes /api/keys and /api/keys/{id}.
func (h *apikeyHandlers) handleAPIKeysRoute(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/keys")

	// /api/keys (no trailing path)
	if path == "" || path == "/" {
		switch r.Method {
		case http.MethodGet:
			h.handleListKeys(w, r)
		case http.MethodPost:
			h.handleCreateKey(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
		return
	}

	// /api/keys/{id}
	if r.Method == http.MethodDelete {
		h.handleDeleteKey(w, r)
		return
	}

	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

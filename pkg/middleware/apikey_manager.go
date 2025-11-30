package middleware

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
)

// APIKeyManager handles API key CRUD operations
type APIKeyManager struct {
	store APIKeyStore
}

// NewAPIKeyManager creates a new API key manager
func NewAPIKeyManager(store APIKeyStore) *APIKeyManager {
	return &APIKeyManager{
		store: store,
	}
}

// CreateAPIKeyRequest represents a request to create a new API key
type CreateAPIKeyRequest struct {
	Name      string     `json:"name"`
	UserID    string     `json:"user_id"`
	Scopes    []string   `json:"scopes"`
	RateLimit int        `json:"rate_limit"` // requests per second
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

// APIKeyResponse represents an API key in responses
type APIKeyResponse struct {
	ID         string     `json:"id"`
	Key        string     `json:"key,omitempty"` // Only included on creation
	Name       string     `json:"name"`
	UserID     string     `json:"user_id"`
	Scopes     []string   `json:"scopes"`
	RateLimit  int        `json:"rate_limit"`
	Enabled    bool       `json:"enabled"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
}

// handleCreateAPIKey creates a new API key
func (m *APIKeyManager) handleCreateAPIKey(w http.ResponseWriter, r *http.Request) {
	var req CreateAPIKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_request", "message": err.Error()})
		return
	}

	// Validate request
	if req.Name == "" {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_request", "message": "name is required"})
		return
	}
	if req.UserID == "" {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_request", "message": "user_id is required"})
		return
	}
	if req.RateLimit <= 0 {
		req.RateLimit = 100 // default to 100 requests per second
	}

	// Generate API key
	key, err := generateAPIKey()
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "generation_failed", "message": err.Error()})
		return
	}

	// Create API key record
	apiKey := &APIKey{
		ID:        generateKeyID(),
		HashedKey: HashAPIKey(key),
		Name:      req.Name,
		UserID:    req.UserID,
		Scopes:    req.Scopes,
		RateLimit: req.RateLimit,
		Enabled:   true,
		ExpiresAt: req.ExpiresAt,
		CreatedAt: time.Now(),
	}

	// Save to store
	if err := m.store.CreateKey(r.Context(), apiKey); err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "storage_failed", "message": err.Error()})
		return
	}

	// Return response with unhashed key (only time it's visible)
	response := &APIKeyResponse{
		ID:        apiKey.ID,
		Key:       key, // Only included here
		Name:      apiKey.Name,
		UserID:    apiKey.UserID,
		Scopes:    apiKey.Scopes,
		RateLimit: apiKey.RateLimit,
		Enabled:   apiKey.Enabled,
		ExpiresAt: apiKey.ExpiresAt,
		CreatedAt: apiKey.CreatedAt,
	}

	respondJSON(w, http.StatusCreated, response)
}

// handleListAPIKeys lists all API keys
func (m *APIKeyManager) handleListAPIKeys(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")

	keys, err := m.store.ListKeys(r.Context(), userID)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "list_failed", "message": err.Error()})
		return
	}

	responses := make([]*APIKeyResponse, len(keys))
	for i, key := range keys {
		responses[i] = &APIKeyResponse{
			ID:         key.ID,
			Name:       key.Name,
			UserID:     key.UserID,
			Scopes:     key.Scopes,
			RateLimit:  key.RateLimit,
			Enabled:    key.Enabled,
			ExpiresAt:  key.ExpiresAt,
			CreatedAt:  key.CreatedAt,
			LastUsedAt: key.LastUsedAt,
		}
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"keys":  responses,
		"total": len(responses),
	})
}

// handleGetAPIKey retrieves a specific API key
func (m *APIKeyManager) handleGetAPIKey(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	keyID := vars["id"]

	key, err := m.store.GetKeyByID(r.Context(), keyID)
	if err != nil {
		respondJSON(w, http.StatusNotFound, map[string]string{"error": "key_not_found", "message": "API key not found"})
		return
	}

	response := &APIKeyResponse{
		ID:         key.ID,
		Name:       key.Name,
		UserID:     key.UserID,
		Scopes:     key.Scopes,
		RateLimit:  key.RateLimit,
		Enabled:    key.Enabled,
		ExpiresAt:  key.ExpiresAt,
		CreatedAt:  key.CreatedAt,
		LastUsedAt: key.LastUsedAt,
	}

	respondJSON(w, http.StatusOK, response)
}

// UpdateAPIKeyRequest represents a request to update an API key
type UpdateAPIKeyRequest struct {
	Name      *string    `json:"name,omitempty"`
	Scopes    *[]string  `json:"scopes,omitempty"`
	RateLimit *int       `json:"rate_limit,omitempty"`
	Enabled   *bool      `json:"enabled,omitempty"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

// handleUpdateAPIKey updates an existing API key
func (m *APIKeyManager) handleUpdateAPIKey(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	keyID := vars["id"]

	var req UpdateAPIKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_request", "message": err.Error()})
		return
	}

	// Get existing key
	key, err := m.store.GetKeyByID(r.Context(), keyID)
	if err != nil {
		respondJSON(w, http.StatusNotFound, map[string]string{"error": "key_not_found", "message": "API key not found"})
		return
	}

	// Update fields
	if req.Name != nil {
		key.Name = *req.Name
	}
	if req.Scopes != nil {
		key.Scopes = *req.Scopes
	}
	if req.RateLimit != nil {
		key.RateLimit = *req.RateLimit
	}
	if req.Enabled != nil {
		key.Enabled = *req.Enabled
	}
	if req.ExpiresAt != nil {
		key.ExpiresAt = req.ExpiresAt
	}

	// Save updated key
	if err := m.store.UpdateKey(r.Context(), key); err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "update_failed", "message": err.Error()})
		return
	}

	response := &APIKeyResponse{
		ID:         key.ID,
		Name:       key.Name,
		UserID:     key.UserID,
		Scopes:     key.Scopes,
		RateLimit:  key.RateLimit,
		Enabled:    key.Enabled,
		ExpiresAt:  key.ExpiresAt,
		CreatedAt:  key.CreatedAt,
		LastUsedAt: key.LastUsedAt,
	}

	respondJSON(w, http.StatusOK, response)
}

// handleDeleteAPIKey deletes an API key
func (m *APIKeyManager) handleDeleteAPIKey(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	keyID := vars["id"]

	if err := m.store.DeleteKey(r.Context(), keyID); err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "delete_failed", "message": err.Error()})
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "API key deleted successfully"})
}

// handleRegenerateAPIKey regenerates an API key
func (m *APIKeyManager) handleRegenerateAPIKey(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	keyID := vars["id"]

	// Get existing key
	key, err := m.store.GetKeyByID(r.Context(), keyID)
	if err != nil {
		respondJSON(w, http.StatusNotFound, map[string]string{"error": "key_not_found", "message": "API key not found"})
		return
	}

	// Generate new key
	newKey, err := generateAPIKey()
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "generation_failed", "message": err.Error()})
		return
	}

	// Update hashed key
	key.HashedKey = HashAPIKey(newKey)

	// Save updated key
	if err := m.store.UpdateKey(r.Context(), key); err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "update_failed", "message": err.Error()})
		return
	}

	// Return response with new unhashed key (only time it's visible)
	response := &APIKeyResponse{
		ID:         key.ID,
		Key:        newKey, // Only included here
		Name:       key.Name,
		UserID:     key.UserID,
		Scopes:     key.Scopes,
		RateLimit:  key.RateLimit,
		Enabled:    key.Enabled,
		ExpiresAt:  key.ExpiresAt,
		CreatedAt:  key.CreatedAt,
		LastUsedAt: key.LastUsedAt,
	}

	respondJSON(w, http.StatusOK, response)
}

// generateAPIKey generates a cryptographically secure API key
func generateAPIKey() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return "gauth_" + base64.URLEncoding.EncodeToString(b), nil
}

// generateKeyID generates a unique key ID
func generateKeyID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b) // crypto/rand.Read always succeeds on supported platforms
	return fmt.Sprintf("key_%x", b)
}

// respondJSON sends a JSON response
func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data) // Ignore error; response already committed
}

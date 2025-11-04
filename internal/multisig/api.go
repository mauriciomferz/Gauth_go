// Package multisig provides REST API handlers for multi-signature PoA workflow.
package multisig

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/internal/metrics"
)

// API implements REST endpoints for multi-signature PoA operations.
type API struct {
	manager *SignatureManager
	metrics metrics.Metrics
}

// NewAPI creates a new multi-signature API handler.
func NewAPI(manager *SignatureManager, metricsProvider metrics.Metrics) *API {
	return &API{
		manager: manager,
		metrics: metricsProvider,
	}
}

// SignRequest represents a signature submission request.
type SignRequest struct {
	PoAID     string            `json:"poa_id"`
	SignerID  string            `json:"signer_id"`
	KeyID     string            `json:"key_id"`
	Signature string            `json:"signature"` // base64-encoded
	Metadata  map[string]string `json:"metadata,omitempty"`
}

// SignResponse represents the response to a signature submission.
type SignResponse struct {
	Success         bool            `json:"success"`
	Message         string          `json:"message"`
	Status          SignatureStatus `json:"status"`
	ThresholdMet    bool            `json:"threshold_met"`
	CollectedCount  int             `json:"collected_count"`
	RequiredCount   int             `json:"required_count"`
	CollectedWeight int             `json:"collected_weight,omitempty"`
	RequiredWeight  int             `json:"required_weight,omitempty"`
}

// StatusResponse represents the multi-signature collection status.
type StatusResponse struct {
	PoAID           string                      `json:"poa_id"`
	Status          SignatureStatus             `json:"status"`
	Threshold       int                         `json:"threshold"`
	RequiredSigners []string                    `json:"required_signers"`
	Collected       []string                    `json:"collected"`
	Remaining       []string                    `json:"remaining"`
	Signatures      map[string]*SignatureRecord `json:"signatures"`
	CreatedAt       time.Time                   `json:"created_at"`
	CompletedAt     *time.Time                  `json:"completed_at,omitempty"`
	ExpiresAt       *time.Time                  `json:"expires_at,omitempty"`
	UseWeighted     bool                        `json:"use_weighted"`
	CollectedWeight int                         `json:"collected_weight,omitempty"`
	TotalWeight     int                         `json:"total_weight,omitempty"`
}

// HandleSign processes signature submissions (POST /api/v1/beta/poa/sign).
func (a *API) HandleSign(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Rate limiting check (future: implement proper rate limiter)
	// For now, rely on HTTP middleware layer

	body, err := io.ReadAll(io.LimitReader(r.Body, 1024*1024)) // 1MB limit
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var req SignRequest
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.PoAID == "" || req.SignerID == "" || req.KeyID == "" || req.Signature == "" {
		http.Error(w, "Missing required fields (poa_id, signer_id, key_id, signature)", http.StatusBadRequest)
		return
	}

	// Add client metadata for audit trail
	if req.Metadata == nil {
		req.Metadata = make(map[string]string)
	}
	req.Metadata["ip_address"] = getClientIP(r)
	req.Metadata["user_agent"] = r.UserAgent()

	// Submit signature
	err = a.manager.SubmitSignature(r.Context(), req.PoAID, req.SignerID, req.KeyID, req.Signature, req.Metadata)
	if err != nil {
		// Log audit event
		if a.metrics != nil {
			a.metrics.IncMultiSignatureVerificationFailures()
		}

		// Determine appropriate HTTP status
		status := http.StatusBadRequest
		if contains(err.Error(), "not found") || contains(err.Error(), "no signature collection") {
			status = http.StatusNotFound
		} else if contains(err.Error(), "expired") {
			status = http.StatusGone
		} else if contains(err.Error(), "already submitted") || contains(err.Error(), "already completed") {
			status = http.StatusConflict
		} else if contains(err.Error(), "verification failed") {
			status = http.StatusUnauthorized
		}

		http.Error(w, err.Error(), status)
		return
	}

	// Retrieve updated status
	state, err := a.manager.GetStatus(r.Context(), req.PoAID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get status: %v", err), http.StatusInternalServerError)
		return
	}

	// Build response
	response := SignResponse{
		Success:        true,
		Message:        "Signature submitted successfully",
		Status:         state.Status,
		ThresholdMet:   state.Status == "completed",
		CollectedCount: len(state.Signatures),
		RequiredCount:  state.Threshold,
	}

	if state.UseWeightedVoting {
		response.CollectedWeight = state.CollectedWeight
		response.RequiredWeight = state.Threshold
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		// Log encoding error but response was already started
	}

	// Record success metric
	if a.metrics != nil {
		a.metrics.IncMultiSignatureVerifications()
	}
}

// HandleStatus retrieves multi-signature collection status (GET /api/v1/beta/poa/:id/multisig/status).
func (a *API) HandleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract PoA ID from URL path
	// Assumes routing like: /api/v1/beta/poa/{id}/multisig/status
	// For now, expect it as a query parameter (router integration would handle path extraction)
	poaID := r.URL.Query().Get("poa_id")
	if poaID == "" {
		// Try extracting from path manually (simple approach)
		// Format: /api/v1/beta/poa/POA_ID/multisig/status
		pathParts := splitPath(r.URL.Path)
		if len(pathParts) >= 6 && pathParts[4] != "multisig" {
			poaID = pathParts[4]
		}
	}

	if poaID == "" {
		http.Error(w, "Missing poa_id parameter", http.StatusBadRequest)
		return
	}

	// Get status from manager
	state, err := a.manager.GetStatus(r.Context(), poaID)
	if err != nil {
		status := http.StatusInternalServerError
		if contains(err.Error(), "not found") || contains(err.Error(), "no signature collection") {
			status = http.StatusNotFound
		}
		http.Error(w, err.Error(), status)
		return
	}

	// Calculate remaining signers
	collected := make([]string, 0, len(state.Signatures))
	collectedMap := make(map[string]bool, len(state.Signatures))
	for signerID := range state.Signatures {
		collected = append(collected, signerID)
		collectedMap[signerID] = true
	}

	remaining := make([]string, 0, len(state.RequiredSigners))
	for _, signerID := range state.RequiredSigners {
		if !collectedMap[signerID] {
			remaining = append(remaining, signerID)
		}
	}

	// Build response
	response := StatusResponse{
		PoAID:           state.PoAID,
		Status:          state.Status,
		Threshold:       state.Threshold,
		RequiredSigners: state.RequiredSigners,
		Collected:       collected,
		Remaining:       remaining,
		Signatures:      state.Signatures,
		CreatedAt:       state.CreatedAt,
		CompletedAt:     state.CompletedAt,
		ExpiresAt:       state.ExpiresAt,
		UseWeighted:     state.UseWeightedVoting,
	}

	if state.UseWeightedVoting {
		response.CollectedWeight = state.CollectedWeight
		response.TotalWeight = state.TotalWeight
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		// Log encoding error but response was already started
	}
}

// HandleActivate activates a PoA after threshold completion (POST /api/v1/beta/poa/:id/activate).
func (a *API) HandleActivate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	poaID := r.URL.Query().Get("poa_id")
	if poaID == "" {
		pathParts := splitPath(r.URL.Path)
		if len(pathParts) >= 5 {
			poaID = pathParts[4]
		}
	}

	if poaID == "" {
		http.Error(w, "Missing poa_id parameter", http.StatusBadRequest)
		return
	}

	// Activate PoA
	err := a.manager.ActivatePoA(r.Context(), poaID)
	if err != nil {
		status := http.StatusBadRequest
		if contains(err.Error(), "not found") || contains(err.Error(), "no signature collection") {
			status = http.StatusNotFound
		} else if contains(err.Error(), "cannot activate") {
			status = http.StatusConflict
		}
		http.Error(w, err.Error(), status)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "PoA activated",
		"poa_id":  poaID,
	}); err != nil {
		// Log encoding error but response was already started
	}
}

// HandleListPending lists all pending multi-signature collections (GET /api/v1/beta/poa/multisig/pending).
func (a *API) HandleListPending(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	pending := a.manager.ListPending(r.Context())

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"pending": pending,
		"count":   len(pending),
	}); err != nil {
		// Response already started, log error
		_ = err
	}
}

// Helper functions

func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header (reverse proxy)
	ip := r.Header.Get("X-Forwarded-For")
	if ip != "" {
		// Take first IP if multiple
		for i := 0; i < len(ip); i++ {
			if ip[i] == ',' {
				return ip[:i]
			}
		}
		return ip
	}

	// Check X-Real-IP header
	ip = r.Header.Get("X-Real-IP")
	if ip != "" {
		return ip
	}

	// Use RemoteAddr as fallback
	return r.RemoteAddr
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || findSubstring(s, substr) >= 0)
}

func findSubstring(s, substr string) int {
	if len(substr) == 0 {
		return 0
	}
	if len(substr) > len(s) {
		return -1
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

func splitPath(path string) []string {
	var parts []string
	start := 0
	for i := 0; i < len(path); i++ {
		if path[i] == '/' {
			if i > start {
				parts = append(parts, path[start:i])
			}
			start = i + 1
		}
	}
	if start < len(path) {
		parts = append(parts, path[start:])
	}
	return parts
}

package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/mauriciomferz/AgentAuth/pkg/agentauthplus"
)

// VerificationHandler handles verification-related HTTP requests
type VerificationHandler struct {
	verificationService agentauthplus.VerificationService
}

// NewVerificationHandler creates a new verification handler
func NewVerificationHandler(verificationService agentauthplus.VerificationService) *VerificationHandler {
	return &VerificationHandler{
		verificationService: verificationService,
	}
}

// VerifyPoARequest represents a request to verify a PoA
type VerifyPoARequest struct {
	PoAID string `json:"poa_id"`
}

// VerifyPoA handles POST /api/v1/verification/poa
func (h *VerificationHandler) VerifyPoA(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req VerifyPoARequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.PoAID == "" {
		http.Error(w, "poa_id is required", http.StatusBadRequest)
		return
	}

	result, err := h.verificationService.VerifyPowerOfAttorney(r.Context(), req.PoAID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(result) // Ignore error; response already committed
}

// PublicVerifyPoA handles GET /api/v1/public/verify/poa/{poa_id}
func (h *VerificationHandler) PublicVerifyPoA(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	poaID := r.URL.Path[len("/api/v1/public/verify/poa/"):]
	if poaID == "" {
		http.Error(w, "poa_id is required", http.StatusBadRequest)
		return
	}

	result, err := h.verificationService.VerifyPowerOfAttorney(r.Context(), poaID)
	if err != nil {
		http.Error(w, "PoA not found or verification failed", http.StatusNotFound)
		return
	}

	publicResult := map[string]interface{}{
		"valid":            result.Valid,
		"poa_id":           result.PoAID,
		"status":           result.Status,
		"issuer_id":        result.IssuerID,
		"grantee_id":       result.GranteeID,
		"valid_from":       result.ValidFrom.Format(time.RFC3339),
		"valid_until":      result.ValidUntil.Format(time.RFC3339),
		"verified_at":      result.VerifiedAt.Format(time.RFC3339),
		"has_attestations": len(result.Attestations) > 0,
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=300")
	_ = json.NewEncoder(w).Encode(publicResult) // Ignore error; response already committed
}

// PublicCheckStatus handles GET /api/v1/public/verify/status/{poa_id}
func (h *VerificationHandler) PublicCheckStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	poaID := r.URL.Path[len("/api/v1/public/verify/status/"):]
	if poaID == "" {
		http.Error(w, "poa_id is required", http.StatusBadRequest)
		return
	}

	revStatus, err := h.verificationService.CheckRevocationStatus(r.Context(), poaID)
	if err != nil {
		http.Error(w, "Status check failed", http.StatusInternalServerError)
		return
	}

	result, err := h.verificationService.VerifyPowerOfAttorney(r.Context(), poaID)
	if err != nil {
		http.Error(w, "PoA not found", http.StatusNotFound)
		return
	}

	statusResponse := map[string]interface{}{
		"poa_id":     poaID,
		"active":     result.Valid && !revStatus.Revoked,
		"status":     result.Status,
		"revoked":    revStatus.Revoked,
		"checked_at": time.Now().Format(time.RFC3339),
	}

	if revStatus.Revoked {
		statusResponse["revoked_at"] = revStatus.RevokedAt.Format(time.RFC3339)
		statusResponse["revoked_by"] = revStatus.RevokedBy
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=60")
	_ = json.NewEncoder(w).Encode(statusResponse) // Ignore error; response already committed
}

// HealthCheck provides health status of verification service
func (h *VerificationHandler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	health := map[string]interface{}{
		"status":    "healthy",
		"service":   "verification",
		"timestamp": time.Now().Format(time.RFC3339),
		"version":   "1.0.0",
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(health) // Ignore error; response already committed
}

// RegisterRoutes registers all verification routes
func (h *VerificationHandler) RegisterRoutes(mux *http.ServeMux) {
	// Public verification endpoints (no auth required)
	mux.HandleFunc("/api/v1/public/verify/poa/", h.PublicVerifyPoA)
	mux.HandleFunc("/api/v1/public/verify/status/", h.PublicCheckStatus)

	// Internal verification endpoints
	mux.HandleFunc("/api/v1/verification/poa", h.VerifyPoA)
	mux.HandleFunc("/api/v1/verification/health", h.HealthCheck)
}

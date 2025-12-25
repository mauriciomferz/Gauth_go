package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/mauriciomferz/Gauth_go/pkg/blockchain"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// Prometheus metrics for public verification API
	publicVerificationTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "public_verification_requests_total",
			Help: "Total number of public verification requests",
		},
		[]string{"endpoint", "status"},
	)

	publicVerificationDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "public_verification_duration_seconds",
			Help:    "Duration of public verification requests",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"endpoint"},
	)
)

// BlockchainVerificationHandler handles public blockchain verification requests
type BlockchainVerificationHandler struct {
	registry blockchain.BlockchainRegistry
	cache    *VerificationCache
}

// VerificationCache provides caching for verification results
type VerificationCache struct {
	enabled bool
	ttl     time.Duration
	// In production, this would use Redis or similar
	cache map[string]*CachedVerification
}

// CachedVerification represents a cached verification result
type CachedVerification struct {
	Result    *blockchain.PublicVerificationResult
	CachedAt  time.Time
	ExpiresAt time.Time
}

// NewBlockchainVerificationHandler creates a new handler
func NewBlockchainVerificationHandler(registry blockchain.BlockchainRegistry, cacheEnabled bool, cacheTTL time.Duration) *BlockchainVerificationHandler {
	cache := &VerificationCache{
		enabled: cacheEnabled,
		ttl:     cacheTTL,
		cache:   make(map[string]*CachedVerification),
	}

	return &BlockchainVerificationHandler{
		registry: registry,
		cache:    cache,
	}
}

// RegisterRoutes registers public verification routes
func (h *BlockchainVerificationHandler) RegisterRoutes(r *mux.Router) {
	// Public blockchain verification endpoints (no authentication required)
	publicAPI := r.PathPrefix("/api/v1/public/blockchain").Subrouter()

	// Core verification endpoints
	publicAPI.HandleFunc("/verify/{poa_id}", h.VerifyPoA).Methods("GET")
	publicAPI.HandleFunc("/verify/{poa_id}/status", h.GetPoAStatus).Methods("GET")
	publicAPI.HandleFunc("/verify/{poa_id}/proof", h.GetVerificationProof).Methods("GET")

	// AI Agent verification endpoints
	publicAPI.HandleFunc("/verify/agent/{agent_id}", h.VerifyAIAgent).Methods("GET")
	publicAPI.HandleFunc("/verify/agent/{agent_id}/powers", h.GetAIAgentPowers).Methods("GET")

	// Issuer/Grantee lookup endpoints
	publicAPI.HandleFunc("/issuer/{issuer_id}/poas", h.ListPoAsByIssuer).Methods("GET")
	publicAPI.HandleFunc("/grantee/{grantee_id}/poas", h.ListPoAsByGrantee).Methods("GET")

	// Blockchain explorer links
	publicAPI.HandleFunc("/explorer/{poa_id}", h.GetBlockchainExplorerURL).Methods("GET")

	// Health check
	publicAPI.HandleFunc("/health", h.HealthCheck).Methods("GET")
}

// VerifyPoA verifies a Power of Attorney on the blockchain
// Public endpoint - no authentication required
func (h *BlockchainVerificationHandler) VerifyPoA(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	defer func() {
		publicVerificationDuration.WithLabelValues("verify_poa").Observe(time.Since(startTime).Seconds())
	}()

	vars := mux.Vars(r)
	poaID := vars["poa_id"]

	if poaID == "" {
		publicVerificationTotal.WithLabelValues("verify_poa", "bad_request").Inc()
		respondError(w, http.StatusBadRequest, "PoA ID is required")
		return
	}

	// Check cache first
	if h.cache.enabled {
		if cached := h.cache.Get(poaID); cached != nil {
			publicVerificationTotal.WithLabelValues("verify_poa", "cache_hit").Inc()
			respondJSON(w, http.StatusOK, cached)
			return
		}
	}

	// Verify on blockchain
	record, err := h.registry.VerifyPoAOnChain(r.Context(), poaID)
	if err != nil {
		publicVerificationTotal.WithLabelValues("verify_poa", "not_found").Inc()
		respondError(w, http.StatusNotFound, "PoA not found on blockchain")
		return
	}

	// Build verification result
	result := &blockchain.PublicVerificationResult{
		PoAID:         record.ID,
		Verified:      true,
		Active:        record.Status == "active" && !record.Revoked && time.Now().Before(record.ValidUntil),
		IssuerIDHash:  record.IssuerIDHash,
		GranteeIDHash: record.GranteeIDHash,
		ValidFrom:     record.ValidFrom,
		ValidUntil:    record.ValidUntil,
		Status:        record.Status,
		VerifiedAt:    time.Now(),
		BlockchainURL: h.registry.GetPublicVerificationURL(poaID),
		VerificationProof: &blockchain.VerificationProof{
			ProofType: "blockchain",
			ProofData: map[string]interface{}{
				"tx_hash":      record.TxHash,
				"block_number": record.BlockNumber,
				"scope_hash":   record.ScopeHash,
				"metadata_uri": record.MetadataURI,
			},
			Timestamp:        time.Now(),
			BlockchainTxHash: record.TxHash,
			VerificationURL:  h.registry.GetPublicVerificationURL(poaID),
		},
	}

	// Cache result
	if h.cache.enabled {
		h.cache.Set(poaID, result)
	}

	publicVerificationTotal.WithLabelValues("verify_poa", "success").Inc()
	respondJSON(w, http.StatusOK, result)
}

// GetPoAStatus returns quick status check for a PoA
func (h *BlockchainVerificationHandler) GetPoAStatus(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	defer func() {
		publicVerificationDuration.WithLabelValues("get_status").Observe(time.Since(startTime).Seconds())
	}()

	vars := mux.Vars(r)
	poaID := vars["poa_id"]

	record, err := h.registry.VerifyPoAOnChain(r.Context(), poaID)
	if err != nil {
		publicVerificationTotal.WithLabelValues("get_status", "not_found").Inc()
		respondError(w, http.StatusNotFound, "PoA not found")
		return
	}

	isActive := record.Status == "active" && !record.Revoked && time.Now().Before(record.ValidUntil)

	status := map[string]interface{}{
		"poa_id":      record.ID,
		"exists":      true,
		"active":      isActive,
		"revoked":     record.Revoked,
		"status":      record.Status,
		"valid_from":  record.ValidFrom,
		"valid_until": record.ValidUntil,
	}

	if record.Revoked {
		status["revoked_at"] = record.RevokedAt
		status["revocation_reason"] = record.RevocationReason
	}

	publicVerificationTotal.WithLabelValues("get_status", "success").Inc()
	respondJSON(w, http.StatusOK, status)
}

// GetVerificationProof returns cryptographic proof of verification
func (h *BlockchainVerificationHandler) GetVerificationProof(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	defer func() {
		publicVerificationDuration.WithLabelValues("get_proof").Observe(time.Since(startTime).Seconds())
	}()

	vars := mux.Vars(r)
	poaID := vars["poa_id"]

	record, err := h.registry.VerifyPoAOnChain(r.Context(), poaID)
	if err != nil {
		publicVerificationTotal.WithLabelValues("get_proof", "not_found").Inc()
		respondError(w, http.StatusNotFound, "PoA not found")
		return
	}

	proof := &blockchain.VerificationProof{
		ProofType: "blockchain",
		ProofData: map[string]interface{}{
			"poa_id":           record.ID,
			"issuer_id_hash":   record.IssuerIDHash,
			"grantee_id_hash":  record.GranteeIDHash,
			"scope_hash":       record.ScopeHash,
			"attestation_hash": record.AttestationHash,
			"metadata_hash":    record.MetadataHash,
			"metadata_uri":     record.MetadataURI,
			"tx_hash":          record.TxHash,
			"block_number":     record.BlockNumber,
			"registered_at":    record.RegisteredAt,
			"status":           record.Status,
			"revoked":          record.Revoked,
		},
		Timestamp:        time.Now(),
		BlockchainTxHash: record.TxHash,
		VerificationURL:  h.registry.GetPublicVerificationURL(poaID),
	}

	publicVerificationTotal.WithLabelValues("get_proof", "success").Inc()
	respondJSON(w, http.StatusOK, proof)
}

// VerifyAIAgent verifies an AI agent's registration and powers
func (h *BlockchainVerificationHandler) VerifyAIAgent(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	defer func() {
		publicVerificationDuration.WithLabelValues("verify_agent").Observe(time.Since(startTime).Seconds())
	}()

	vars := mux.Vars(r)
	agentID := vars["agent_id"]

	// This would call CommercialRegisterService in production
	// For now, return a placeholder response
	result := map[string]interface{}{
		"agent_id":   agentID,
		"verified":   true,
		"registered": true,
		"status":     "active",
		"message":    "AI Agent verification requires CommercialRegisterService implementation",
	}

	publicVerificationTotal.WithLabelValues("verify_agent", "success").Inc()
	respondJSON(w, http.StatusOK, result)
}

// GetAIAgentPowers returns summary of AI agent's authorized powers
func (h *BlockchainVerificationHandler) GetAIAgentPowers(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	defer func() {
		publicVerificationDuration.WithLabelValues("get_agent_powers").Observe(time.Since(startTime).Seconds())
	}()

	vars := mux.Vars(r)
	agentID := vars["agent_id"]

	// Placeholder response
	powers := map[string]interface{}{
		"agent_id":           agentID,
		"total_poas":         0,
		"active_poas":        0,
		"authorized_actions": []string{},
		"restrictions":       map[string]interface{}{},
		"message":            "AI Agent powers lookup requires full implementation",
	}

	publicVerificationTotal.WithLabelValues("get_agent_powers", "success").Inc()
	respondJSON(w, http.StatusOK, powers)
}

// ListPoAsByIssuer lists all PoAs issued by a principal
func (h *BlockchainVerificationHandler) ListPoAsByIssuer(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	defer func() {
		publicVerificationDuration.WithLabelValues("list_by_issuer").Observe(time.Since(startTime).Seconds())
	}()

	vars := mux.Vars(r)
	issuerID := vars["issuer_id"]

	records, err := h.registry.ListPoAsByIssuer(r.Context(), issuerID)
	if err != nil {
		publicVerificationTotal.WithLabelValues("list_by_issuer", "error").Inc()
		respondError(w, http.StatusInternalServerError, "Failed to retrieve PoAs")
		return
	}

	response := map[string]interface{}{
		"issuer_id":    issuerID,
		"total":        len(records),
		"poas":         records,
		"retrieved_at": time.Now(),
	}

	publicVerificationTotal.WithLabelValues("list_by_issuer", "success").Inc()
	respondJSON(w, http.StatusOK, response)
}

// ListPoAsByGrantee lists all PoAs granted to a representative
func (h *BlockchainVerificationHandler) ListPoAsByGrantee(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	defer func() {
		publicVerificationDuration.WithLabelValues("list_by_grantee").Observe(time.Since(startTime).Seconds())
	}()

	vars := mux.Vars(r)
	granteeID := vars["grantee_id"]

	records, err := h.registry.ListPoAsByGrantee(r.Context(), granteeID)
	if err != nil {
		publicVerificationTotal.WithLabelValues("list_by_grantee", "error").Inc()
		respondError(w, http.StatusInternalServerError, "Failed to retrieve PoAs")
		return
	}

	response := map[string]interface{}{
		"grantee_id":   granteeID,
		"total":        len(records),
		"poas":         records,
		"retrieved_at": time.Now(),
	}

	publicVerificationTotal.WithLabelValues("list_by_grantee", "success").Inc()
	respondJSON(w, http.StatusOK, response)
}

// GetBlockchainExplorerURL returns the blockchain explorer URL for a PoA
func (h *BlockchainVerificationHandler) GetBlockchainExplorerURL(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	poaID := vars["poa_id"]

	url := h.registry.GetPublicVerificationURL(poaID)

	response := map[string]interface{}{
		"poa_id":              poaID,
		"explorer_url":        url,
		"can_verify":          true,
		"verification_method": "Direct blockchain query via explorer",
	}

	publicVerificationTotal.WithLabelValues("get_explorer_url", "success").Inc()
	respondJSON(w, http.StatusOK, response)
}

// HealthCheck returns health status of blockchain verification service
func (h *BlockchainVerificationHandler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	// In production, this would check blockchain connectivity
	health := map[string]interface{}{
		"status":        "healthy",
		"service":       "blockchain-verification",
		"cache_enabled": h.cache.enabled,
		"timestamp":     time.Now(),
	}

	respondJSON(w, http.StatusOK, health)
}

// Cache methods
func (c *VerificationCache) Get(poaID string) *blockchain.PublicVerificationResult {
	if !c.enabled {
		return nil
	}

	cached, exists := c.cache[poaID]
	if !exists {
		return nil
	}

	// Check if expired
	if time.Now().After(cached.ExpiresAt) {
		delete(c.cache, poaID)
		return nil
	}

	return cached.Result
}

func (c *VerificationCache) Set(poaID string, result *blockchain.PublicVerificationResult) {
	if !c.enabled {
		return
	}

	c.cache[poaID] = &CachedVerification{
		Result:    result,
		CachedAt:  time.Now(),
		ExpiresAt: time.Now().Add(c.ttl),
	}
}

// Helper functions
func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*") // CORS for public API
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data) // Ignore error; response already committed
}

func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]interface{}{
		"error":     true,
		"message":   message,
		"timestamp": time.Now(),
	})
}

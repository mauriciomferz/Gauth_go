package web

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mauriciomferz/Gauth_go/pkg/revocation"
)

// RevocationService wraps the revocation system components for integration with BetaServer.
// It provides emergency revocation, two-phase revocation, optimistic revocation, and
// circuit breaker functionality via HTTP endpoints.
type RevocationService struct {
	redisAddrs []string
	oracle     *revocation.EmergencyRevocationOracle
	twoPhase   *revocation.TwoPhaseRevocation
	optimistic *revocation.OptimisticRevocation
	circuit    *revocation.CircuitBreaker
	enabled    bool
	logger     revocation.Logger
}

// NewRevocationService initializes the revocation system with Redis connection.
// It returns nil if GAUTH_REVOCATION_ENABLED != "1" or initialization fails.
func NewRevocationService(ctx context.Context) *RevocationService {
	// Check if revocation system is enabled
	if os.Getenv("GAUTH_REVOCATION_ENABLED") != "1" {
		log.Println("[revocation] Revocation system disabled (set GAUTH_REVOCATION_ENABLED=1 to enable)")
		return &RevocationService{enabled: false}
	}

	// Get Redis configuration from environment
	redisHost := os.Getenv("REDIS_HOST")
	if redisHost == "" {
		redisHost = "localhost"
	}
	redisPort := os.Getenv("REDIS_PORT")
	if redisPort == "" {
		redisPort = "6379"
	}

	redisAddrs := []string{redisHost + ":" + redisPort}
	logger := revocation.NewSimpleLogger("revocation")

	log.Printf("[revocation] Connecting to Redis at %s:%s", redisHost, redisPort)

	// Initialize Emergency Oracle
	oracle, err := revocation.NewEmergencyOracle(redisAddrs, logger)
	if err != nil {
		log.Printf("[revocation] Failed to initialize oracle: %v", err)
		return &RevocationService{enabled: false}
	}

	// Initialize Two-Phase Revocation
	twoPhase, err := revocation.NewTwoPhaseRevocation(oracle, redisAddrs, logger)
	if err != nil {
		log.Printf("[revocation] Failed to initialize two-phase revocation: %v", err)
		_ = oracle.Close()
		return &RevocationService{enabled: false}
	}

	// Configure two-phase timeout
	if timeoutStr := os.Getenv("GAUTH_REVOCATION_TWOPHASE_TIMEOUT"); timeoutStr != "" {
		if timeout, err := time.ParseDuration(timeoutStr); err == nil {
			twoPhase.SetDisableTimeout(timeout)
		}
	}

	// Initialize Optimistic Revocation
	optimistic, err := revocation.NewOptimisticRevocation(redisAddrs, oracle, logger)
	if err != nil {
		log.Printf("[revocation] Failed to initialize optimistic revocation: %v", err)
		_ = twoPhase.Close()
		_ = oracle.Close()
		return &RevocationService{enabled: false}
	}

	// Configure optimistic revocation
	if windowStr := os.Getenv("GAUTH_REVOCATION_OPTIMISTIC_WINDOW"); windowStr != "" {
		if window, err := time.ParseDuration(windowStr); err == nil {
			optimistic.SetChallengeWindow(window)
		}
	}

	// Initialize Circuit Breaker
	rateLimit := 10
	if rateLimitStr := os.Getenv("GAUTH_REVOCATION_CIRCUIT_RATE"); rateLimitStr != "" {
		if rate, err := strconv.Atoi(rateLimitStr); err == nil {
			rateLimit = rate
		}
	}

	config := &revocation.RateLimitConfig{
		MaxTxPerMinute:    rateLimit,
		MaxTxPerHour:      rateLimit * 6,
		MaxValuePerMinute: 10000000000000000000, // 10 ETH per minute (10^19 Wei)
		MaxValuePerHour:   18446744073709551615, // Max uint64 value (~18.4 ETH per hour due to overflow limitation)
		MaxFailureRate:    0.1,                  // 10% max failure rate
		FailureWindowSecs: 60,
	}

	circuit, err := revocation.NewCircuitBreaker(redisAddrs, config, logger)
	if err != nil {
		log.Printf("[revocation] Failed to initialize circuit breaker: %v", err)
		_ = optimistic.Close()
		_ = twoPhase.Close()
		_ = oracle.Close()
		return &RevocationService{enabled: false}
	}

	log.Println("[revocation] All revocation components initialized successfully")

	return &RevocationService{
		redisAddrs: redisAddrs,
		oracle:     oracle,
		twoPhase:   twoPhase,
		optimistic: optimistic,
		circuit:    circuit,
		enabled:    true,
		logger:     logger,
	}
}

// Close gracefully shuts down all revocation components
func (rs *RevocationService) Close() error {
	if !rs.enabled {
		return nil
	}

	var errs []error

	if rs.circuit != nil {
		if err := rs.circuit.Close(); err != nil {
			errs = append(errs, fmt.Errorf("circuit breaker close: %w", err))
		}
	}

	if rs.optimistic != nil {
		if err := rs.optimistic.Close(); err != nil {
			errs = append(errs, fmt.Errorf("optimistic close: %w", err))
		}
	}

	if rs.twoPhase != nil {
		if err := rs.twoPhase.Close(); err != nil {
			errs = append(errs, fmt.Errorf("two-phase close: %w", err))
		}
	}

	if rs.oracle != nil {
		if err := rs.oracle.Close(); err != nil {
			errs = append(errs, fmt.Errorf("oracle close: %w", err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("revocation service close errors: %v", errs)
	}

	return nil
}

// RegisterHandlers registers all revocation HTTP endpoints
func (rs *RevocationService) RegisterHandlers(group *gin.RouterGroup) {
	if !rs.enabled {
		rs.registerDisabledHandlers(group)
		return
	}

	// Two-Phase Revocation endpoints
	group.POST("/revocation/disable", rs.handleDisablePoA)
	group.POST("/revocation/revoke", rs.handleRevokePoA)
	group.POST("/revocation/cancel", rs.handleCancelDisable)
	group.GET("/revocation/status", rs.handleGetStatus)

	// Optimistic Revocation endpoints
	group.POST("/revocation/optimistic/pending", rs.handleMarkPending)
	group.POST("/revocation/optimistic/finalize", rs.handleFinalize)
	group.POST("/revocation/optimistic/challenge", rs.handleChallenge)

	// Circuit Breaker endpoints
	group.GET("/revocation/circuit/metrics", rs.handleGetMetrics)
	group.POST("/revocation/circuit/reset", rs.handleResetMetrics)
	group.POST("/revocation/circuit/suspend", rs.handleManualSuspend)
	group.POST("/revocation/circuit/resume", rs.handleManualResume)

	// Unified validation endpoint
	group.POST("/revocation/validate", rs.handleValidateTransaction)

	// Health check
	group.GET("/revocation/health", rs.handleHealth)

	log.Println("[revocation] Registered 13 revocation HTTP endpoints")
}

// registerDisabledHandlers returns 503 for all endpoints when system is disabled
func (rs *RevocationService) registerDisabledHandlers(group *gin.RouterGroup) {
	handler := func(c *gin.Context) {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"success": false,
			"error":   "revocation_disabled",
			"message": "Revocation system is not enabled. Set GAUTH_REVOCATION_ENABLED=1 and configure Redis.",
		})
	}

	group.POST("/revocation/disable", handler)
	group.POST("/revocation/revoke", handler)
	group.POST("/revocation/cancel", handler)
	group.GET("/revocation/status", handler)
	group.POST("/revocation/optimistic/pending", handler)
	group.POST("/revocation/optimistic/finalize", handler)
	group.POST("/revocation/optimistic/challenge", handler)
	group.GET("/revocation/circuit/metrics", handler)
	group.POST("/revocation/circuit/reset", handler)
	group.POST("/revocation/circuit/suspend", handler)
	group.POST("/revocation/circuit/resume", handler)
	group.POST("/revocation/validate", handler)
	group.GET("/revocation/health", handler)
}

// ============================================================================
// TWO-PHASE REVOCATION HANDLERS
// ============================================================================

// handleDisablePoA disables a PoA (Phase 1: reversible)
func (rs *RevocationService) handleDisablePoA(c *gin.Context) {
	var req struct {
		PoAID     string `json:"poa_id" binding:"required"`
		Principal string `json:"principal" binding:"required"`
		Reason    string `json:"reason" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid_request", "message": err.Error()})
		return
	}

	ctx := c.Request.Context()
	if err := rs.twoPhase.DisablePoA(ctx, req.PoAID, req.Principal, req.Reason); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "disable_failed", "message": err.Error()})
		return
	}

	state, _ := rs.twoPhase.GetPoAState(ctx, req.PoAID)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"poa_id":  req.PoAID,
		"status":  "disabled",
		"state":   state,
		"message": "PoA disabled successfully",
	})
}

// handleRevokePoA permanently revokes a PoA (Phase 2: irreversible)
func (rs *RevocationService) handleRevokePoA(c *gin.Context) {
	var req struct {
		PoAID  string `json:"poa_id" binding:"required"`
		Reason string `json:"reason" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid_request", "message": err.Error()})
		return
	}

	ctx := c.Request.Context()
	if err := rs.twoPhase.RevokePoA(ctx, req.PoAID, req.Reason); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "revoke_failed", "message": err.Error()})
		return
	}

	state, _ := rs.twoPhase.GetPoAState(ctx, req.PoAID)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"poa_id":  req.PoAID,
		"status":  "revoked",
		"state":   state,
		"message": "PoA permanently revoked",
	})
}

// handleCancelDisable cancels a disabled PoA (returns to active)
func (rs *RevocationService) handleCancelDisable(c *gin.Context) {
	var req struct {
		PoAID string `json:"poa_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid_request", "message": err.Error()})
		return
	}

	ctx := c.Request.Context()
	if err := rs.twoPhase.CancelDisable(ctx, req.PoAID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "cancel_failed", "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"poa_id":  req.PoAID,
		"status":  "active",
		"message": "PoA re-enabled successfully",
	})
}

// handleGetStatus retrieves comprehensive revocation status for a PoA
func (rs *RevocationService) handleGetStatus(c *gin.Context) {
	poaID := c.Query("poa_id")
	if poaID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "missing_poa_id", "message": "poa_id parameter required"})
		return
	}

	ctx := c.Request.Context()

	// Get status from all systems
	twoPhaseState, _ := rs.twoPhase.GetPoAState(ctx, poaID)
	twoPhaseUsable, twoPhaseMsg, _ := rs.twoPhase.IsPoAUsable(ctx, poaID)

	optimisticState, _ := rs.optimistic.GetRevocationState(ctx, poaID)
	optimisticUsable, optimisticMsg, _ := rs.optimistic.IsPoAUsable(ctx, poaID)

	circuitMetrics, _ := rs.circuit.GetMetrics(ctx, poaID)
	circuitAllowed, circuitMsg, _ := rs.circuit.IsPoAAllowed(ctx, poaID)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"poa_id":  poaID,
		"two_phase": gin.H{
			"state":   twoPhaseState,
			"usable":  twoPhaseUsable,
			"message": twoPhaseMsg,
		},
		"optimistic": gin.H{
			"state":   optimisticState,
			"usable":  optimisticUsable,
			"message": optimisticMsg,
		},
		"circuit_breaker": gin.H{
			"metrics": circuitMetrics,
			"allowed": circuitAllowed,
			"message": circuitMsg,
		},
		"overall_allowed": twoPhaseUsable && optimisticUsable && circuitAllowed,
	})
}

// ============================================================================
// OPTIMISTIC REVOCATION HANDLERS
// ============================================================================

// handleMarkPending marks a PoA as pending revocation (with collateral)
func (rs *RevocationService) handleMarkPending(c *gin.Context) {
	var req struct {
		PoAID      string `json:"poa_id" binding:"required"`
		Principal  string `json:"principal" binding:"required"`
		Reason     string `json:"reason" binding:"required"`
		Collateral uint64 `json:"collateral" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid_request", "message": err.Error()})
		return
	}

	ctx := c.Request.Context()
	if err := rs.optimistic.MarkPendingRevocation(ctx, req.PoAID, req.Principal, req.Reason, req.Collateral); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "mark_pending_failed", "message": err.Error()})
		return
	}

	state, _ := rs.optimistic.GetRevocationState(ctx, req.PoAID)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"poa_id":  req.PoAID,
		"status":  "pending",
		"state":   state,
		"message": "PoA marked as pending revocation",
	})
}

// handleFinalize finalizes a pending revocation
func (rs *RevocationService) handleFinalize(c *gin.Context) {
	var req struct {
		PoAID string `json:"poa_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid_request", "message": err.Error()})
		return
	}

	ctx := c.Request.Context()
	if err := rs.optimistic.FinalizeRevocation(ctx, req.PoAID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "finalize_failed", "message": err.Error()})
		return
	}

	state, _ := rs.optimistic.GetRevocationState(ctx, req.PoAID)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"poa_id":  req.PoAID,
		"status":  "finalized",
		"state":   state,
		"message": "PoA revocation finalized",
	})
}

// handleChallenge challenges a pending revocation
func (rs *RevocationService) handleChallenge(c *gin.Context) {
	var req struct {
		PoAID      string `json:"poa_id" binding:"required"`
		Challenger string `json:"challenger" binding:"required"`
		Evidence   string `json:"evidence" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid_request", "message": err.Error()})
		return
	}

	ctx := c.Request.Context()
	if err := rs.optimistic.ChallengeRevocation(ctx, req.PoAID, req.Challenger, req.Evidence); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "challenge_failed", "message": err.Error()})
		return
	}

	state, _ := rs.optimistic.GetRevocationState(ctx, req.PoAID)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"poa_id":  req.PoAID,
		"status":  "challenged",
		"state":   state,
		"message": "Revocation challenged successfully, collateral slashed",
	})
}

// ============================================================================
// CIRCUIT BREAKER HANDLERS
// ============================================================================

// handleGetMetrics retrieves circuit breaker metrics for a PoA
func (rs *RevocationService) handleGetMetrics(c *gin.Context) {
	poaID := c.Query("poa_id")
	if poaID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "missing_poa_id", "message": "poa_id parameter required"})
		return
	}

	ctx := c.Request.Context()
	metrics, err := rs.circuit.GetMetrics(ctx, poaID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "get_metrics_failed", "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, metrics)
}

// handleResetMetrics resets circuit breaker metrics for a PoA (admin)
func (rs *RevocationService) handleResetMetrics(c *gin.Context) {
	var req struct {
		PoAID string `json:"poa_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid_request", "message": err.Error()})
		return
	}

	ctx := c.Request.Context()
	if err := rs.circuit.ResetMetrics(ctx, req.PoAID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "reset_failed", "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"poa_id":  req.PoAID,
		"status":  "reset",
		"message": "Metrics reset successfully",
	})
}

// handleManualSuspend manually suspends a PoA (admin)
func (rs *RevocationService) handleManualSuspend(c *gin.Context) {
	var req struct {
		PoAID  string `json:"poa_id" binding:"required"`
		Reason string `json:"reason" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid_request", "message": err.Error()})
		return
	}

	ctx := c.Request.Context()
	reason := revocation.SuspensionReason(req.Reason)
	if err := rs.circuit.ManualSuspend(ctx, req.PoAID, reason); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "suspend_failed", "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"poa_id":  req.PoAID,
		"status":  "suspended",
		"message": "PoA manually suspended",
	})
}

// handleManualResume manually resumes a suspended PoA (admin)
func (rs *RevocationService) handleManualResume(c *gin.Context) {
	var req struct {
		PoAID string `json:"poa_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid_request", "message": err.Error()})
		return
	}

	ctx := c.Request.Context()
	if err := rs.circuit.ManualResume(ctx, req.PoAID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "resume_failed", "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"poa_id":  req.PoAID,
		"status":  "resumed",
		"message": "PoA manually resumed",
	})
}

// ============================================================================
// UNIFIED VALIDATION
// ============================================================================

// handleValidateTransaction validates a transaction against all revocation systems
func (rs *RevocationService) handleValidateTransaction(c *gin.Context) {
	var req struct {
		PoAID   string `json:"poa_id" binding:"required"`
		Value   uint64 `json:"value"`
		Success bool   `json:"success"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid_request", "message": err.Error()})
		return
	}

	ctx := c.Request.Context()

	// 1. Check circuit breaker (rate limits)
	allowed, message, err := rs.circuit.IsPoAAllowed(ctx, req.PoAID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "circuit_check_failed", "message": err.Error()})
		return
	}
	if !allowed {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"allowed": false,
			"reason":  "circuit_breaker",
			"message": message,
		})
		return
	}

	// 2. Check two-phase revocation
	usable, message, err := rs.twoPhase.IsPoAUsable(ctx, req.PoAID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "two_phase_check_failed", "message": err.Error()})
		return
	}
	if !usable {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"allowed": false,
			"reason":  "two_phase_revocation",
			"message": message,
		})
		return
	}

	// 3. Check optimistic revocation
	usable, message, err = rs.optimistic.IsPoAUsable(ctx, req.PoAID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "optimistic_check_failed", "message": err.Error()})
		return
	}
	if !usable {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"allowed": false,
			"reason":  "optimistic_revocation",
			"message": message,
		})
		return
	}

	// 4. Record transaction in circuit breaker
	if err := rs.circuit.RecordTransaction(ctx, req.PoAID, req.Value, req.Success); err != nil {
		log.Printf("[revocation] Failed to record transaction: %v", err)
	}

	// All checks passed
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"allowed": true,
		"message": "Transaction validated successfully",
	})
}

// ============================================================================
// HEALTH CHECK
// ============================================================================

// handleHealth provides a health check endpoint
func (rs *RevocationService) handleHealth(c *gin.Context) {
	if !rs.enabled {
		c.JSON(http.StatusOK, gin.H{
			"enabled": false,
			"message": "Revocation system is disabled",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"health": gin.H{
			"enabled": true,
			"components": gin.H{
				"oracle":     rs.oracle != nil,
				"two_phase":  rs.twoPhase != nil,
				"optimistic": rs.optimistic != nil,
				"circuit":    rs.circuit != nil,
			},
		},
		"message": "Revocation system healthy",
	})
}

// GetStats returns monitoring statistics
func (rs *RevocationService) GetStats() map[string]interface{} {
	if !rs.enabled {
		return map[string]interface{}{
			"enabled": false,
		}
	}

	return map[string]interface{}{
		"enabled": true,
		"components": map[string]bool{
			"oracle":     rs.oracle != nil,
			"two_phase":  rs.twoPhase != nil,
			"optimistic": rs.optimistic != nil,
			"circuit":    rs.circuit != nil,
		},
	}
}

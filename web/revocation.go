package web

import (
	"context"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/mauriciomferz/Gauth_go/pkg/revocation"
)

// RevocationService wraps the revocation system components for integration with BetaServer.
// It provides emergency revocation, two-phase revocation, optimistic revocation, and
// circuit breaker functionality via HTTP endpoints.
type RevocationService struct {
	redisClient *redis.Client
	oracle      *revocation.EmergencyRevocationOracle
	twoPhase    *revocation.TwoPhaseRevocation
	optimistic  *revocation.OptimisticRevocation
	circuit     *revocation.CircuitBreaker
	enabled     bool
	logger      *revocationLogger
}

// revocationLogger implements the revocation.Logger interface
type revocationLogger struct{}

func (l *revocationLogger) Info(msg string, args ...interface{})  { log.Printf("[revocation] INFO: "+msg, args...) }
func (l *revocationLogger) Warn(msg string, args ...interface{})  { log.Printf("[revocation] WARN: "+msg, args...) }
func (l *revocationLogger) Error(msg string, args ...interface{}) { log.Printf("[revocation] ERROR: "+msg, args...) }
func (l *revocationLogger) Debug(msg string, args ...interface{}) {
	if os.Getenv("GAUTH_REVOCATION_DEBUG") == "1" {
		log.Printf("[revocation] DEBUG: "+msg, args...)
	}
}

// NewRevocationService initializes the revocation system with Redis connection.
// It returns nil if GAUTH_REVOCATION_ENABLED != "1" or Redis connection fails.
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
	redisPassword := os.Getenv("REDIS_PASSWORD")
	redisDB := 0
	if dbStr := os.Getenv("REDIS_DB"); dbStr != "" {
		if db, err := strconv.Atoi(dbStr); err == nil {
			redisDB = db
		}
	}

	// Create Redis client
	client := redis.NewClient(&redis.Options{
		Addr:         redisHost + ":" + redisPort,
		Password:     redisPassword,
		DB:           redisDB,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		PoolSize:     10,
		MinIdleConns: 2,
	})

	// Test Redis connection
	if err := client.Ping(ctx).Err(); err != nil {
		log.Printf("[revocation] Failed to connect to Redis at %s:%s: %v", redisHost, redisPort, err)
		log.Println("[revocation] Revocation system will be disabled")
		return &RevocationService{enabled: false}
	}

	logger := &revocationLogger{}
	log.Printf("[revocation] Connected to Redis at %s:%s", redisHost, redisPort)

	// Initialize Emergency Oracle
	oracleConfig := revocation.DefaultOracleConfig()
	if channelStr := os.Getenv("GAUTH_REVOCATION_ORACLE_CHANNEL"); channelStr != "" {
		oracleConfig.Channel = channelStr
	}
	oracle, err := revocation.NewEmergencyRevocationOracle(client, oracleConfig, logger)
	if err != nil {
		log.Printf("[revocation] Failed to initialize oracle: %v", err)
		client.Close()
		return &RevocationService{enabled: false}
	}

	// Initialize Two-Phase Revocation
	twoPhaseConfig := revocation.DefaultTwoPhaseConfig()
	if timeoutStr := os.Getenv("GAUTH_REVOCATION_TWOPHASE_TIMEOUT"); timeoutStr != "" {
		if timeout, err := time.ParseDuration(timeoutStr); err == nil {
			twoPhaseConfig.DisableTimeout = timeout
		}
	}
	twoPhase, err := revocation.NewTwoPhaseRevocation(client, twoPhaseConfig, logger)
	if err != nil {
		log.Printf("[revocation] Failed to initialize two-phase: %v", err)
		oracle.Stop()
		client.Close()
		return &RevocationService{enabled: false}
	}

	// Initialize Optimistic Revocation
	optimisticConfig := revocation.DefaultOptimisticConfig()
	if windowStr := os.Getenv("GAUTH_REVOCATION_OPTIMISTIC_WINDOW"); windowStr != "" {
		if window, err := time.ParseDuration(windowStr); err == nil {
			optimisticConfig.ChallengeWindow = window
		}
	}
	optimistic, err := revocation.NewOptimisticRevocation(client, optimisticConfig, logger)
	if err != nil {
		log.Printf("[revocation] Failed to initialize optimistic: %v", err)
		twoPhase.Stop()
		oracle.Stop()
		client.Close()
		return &RevocationService{enabled: false}
	}

	// Initialize Circuit Breaker
	circuitConfig := revocation.DefaultCircuitConfig()
	if rateLimitStr := os.Getenv("GAUTH_REVOCATION_CIRCUIT_RATE"); rateLimitStr != "" {
		if rateLimit, err := strconv.Atoi(rateLimitStr); err == nil {
			circuitConfig.RateLimitPerMin = rateLimit
		}
	}
	circuit, err := revocation.NewCircuitBreaker(client, circuitConfig, logger)
	if err != nil {
		log.Printf("[revocation] Failed to initialize circuit breaker: %v", err)
		optimistic.Stop()
		twoPhase.Stop()
		oracle.Stop()
		client.Close()
		return &RevocationService{enabled: false}
	}

	log.Println("[revocation] All revocation components initialized successfully")
	log.Printf("[revocation] Oracle channel: %s", oracleConfig.Channel)
	log.Printf("[revocation] Two-phase timeout: %v", twoPhaseConfig.DisableTimeout)
	log.Printf("[revocation] Optimistic window: %v", optimisticConfig.ChallengeWindow)
	log.Printf("[revocation] Circuit rate limit: %d/min", circuitConfig.RateLimitPerMin)

	return &RevocationService{
		redisClient: client,
		oracle:      oracle,
		twoPhase:    twoPhase,
		optimistic:  optimistic,
		circuit:     circuit,
		enabled:     true,
		logger:      logger,
	}
}

// Close gracefully shuts down the revocation service and closes Redis connection.
func (rs *RevocationService) Close() error {
	if !rs.enabled {
		return nil
	}

	log.Println("[revocation] Shutting down revocation service...")
	
	if rs.circuit != nil {
		rs.circuit.Stop()
	}
	if rs.optimistic != nil {
		rs.optimistic.Stop()
	}
	if rs.twoPhase != nil {
		rs.twoPhase.Stop()
	}
	if rs.oracle != nil {
		rs.oracle.Stop()
	}
	if rs.redisClient != nil {
		if err := rs.redisClient.Close(); err != nil {
			log.Printf("[revocation] Error closing Redis: %v", err)
			return err
		}
	}

	log.Println("[revocation] Revocation service shut down successfully")
	return nil
}

// RegisterHandlers registers all revocation HTTP endpoints on the provided Gin router group.
func (rs *RevocationService) RegisterHandlers(group *gin.RouterGroup) {
	if !rs.enabled {
		// Register disabled handlers that return 503
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

	// Health check endpoint
	group.GET("/revocation/health", rs.handleHealth)

	log.Println("[revocation] Registered 13 revocation HTTP endpoints")
}

// registerDisabledHandlers registers handlers that return 503 when revocation is disabled
func (rs *RevocationService) registerDisabledHandlers(group *gin.RouterGroup) {
	disabled := func(c *gin.Context) {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"success": false,
			"error":   "revocation_disabled",
			"message": "Revocation system is not enabled. Set GAUTH_REVOCATION_ENABLED=1 and ensure Redis is available.",
		})
	}

	group.POST("/revocation/disable", disabled)
	group.POST("/revocation/revoke", disabled)
	group.POST("/revocation/cancel", disabled)
	group.GET("/revocation/status", disabled)
	group.POST("/revocation/optimistic/pending", disabled)
	group.POST("/revocation/optimistic/finalize", disabled)
	group.POST("/revocation/optimistic/challenge", disabled)
	group.GET("/revocation/circuit/metrics", disabled)
	group.POST("/revocation/circuit/reset", disabled)
	group.POST("/revocation/circuit/suspend", disabled)
	group.POST("/revocation/circuit/resume", disabled)
	group.POST("/revocation/validate", disabled)
	
	group.GET("/revocation/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"enabled": false,
			"message": "Revocation system is disabled",
		})
	})

	log.Println("[revocation] Registered disabled handlers (revocation system not enabled)")
}

// HTTP Handler implementations

func (rs *RevocationService) handleDisablePoA(c *gin.Context) {
	var req struct {
		PoAID  string `json:"poa_id" binding:"required"`
		Reason string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid_request", "message": err.Error()})
		return
	}

	if err := rs.twoPhase.Disable(c.Request.Context(), req.PoAID, req.Reason); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "disable_failed", "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"poa_id":  req.PoAID,
		"status":  "disabled",
		"message": "PoA temporarily disabled. Call /revocation/revoke to finalize or /revocation/cancel to revert.",
	})
}

func (rs *RevocationService) handleRevokePoA(c *gin.Context) {
	var req struct {
		PoAID  string `json:"poa_id" binding:"required"`
		Reason string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid_request", "message": err.Error()})
		return
	}

	if err := rs.twoPhase.Revoke(c.Request.Context(), req.PoAID, req.Reason); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "revoke_failed", "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"poa_id":  req.PoAID,
		"status":  "revoked",
		"message": "PoA permanently revoked.",
	})
}

func (rs *RevocationService) handleCancelDisable(c *gin.Context) {
	var req struct {
		PoAID string `json:"poa_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid_request", "message": err.Error()})
		return
	}

	if err := rs.twoPhase.CancelDisable(c.Request.Context(), req.PoAID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "cancel_failed", "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"poa_id":  req.PoAID,
		"status":  "active",
		"message": "Disable cancelled. PoA is now active.",
	})
}

func (rs *RevocationService) handleGetStatus(c *gin.Context) {
	poaID := c.Query("poa_id")
	if poaID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "missing_poa_id"})
		return
	}

	status, err := rs.twoPhase.GetStatus(c.Request.Context(), poaID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "status_check_failed", "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"poa_id":  poaID,
		"status":  status,
	})
}

func (rs *RevocationService) handleMarkPending(c *gin.Context) {
	var req struct {
		PoAID      string  `json:"poa_id" binding:"required"`
		Collateral float64 `json:"collateral" binding:"required"`
		Reason     string  `json:"reason"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid_request", "message": err.Error()})
		return
	}

	if err := rs.optimistic.MarkPending(c.Request.Context(), req.PoAID, req.Collateral, req.Reason); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "mark_pending_failed", "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":    true,
		"poa_id":     req.PoAID,
		"status":     "pending",
		"collateral": req.Collateral,
		"message":    "Revocation marked pending. Can be challenged within challenge window.",
	})
}

func (rs *RevocationService) handleFinalize(c *gin.Context) {
	var req struct {
		PoAID string `json:"poa_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid_request", "message": err.Error()})
		return
	}

	if err := rs.optimistic.Finalize(c.Request.Context(), req.PoAID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "finalize_failed", "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"poa_id":  req.PoAID,
		"status":  "finalized",
		"message": "Revocation finalized. PoA is now revoked.",
	})
}

func (rs *RevocationService) handleChallenge(c *gin.Context) {
	var req struct {
		PoAID    string `json:"poa_id" binding:"required"`
		Evidence string `json:"evidence"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid_request", "message": err.Error()})
		return
	}

	if err := rs.optimistic.Challenge(c.Request.Context(), req.PoAID, req.Evidence); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "challenge_failed", "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"poa_id":  req.PoAID,
		"status":  "challenged",
		"message": "Revocation challenged successfully.",
	})
}

func (rs *RevocationService) handleGetMetrics(c *gin.Context) {
	poaID := c.Query("poa_id")
	if poaID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "missing_poa_id"})
		return
	}

	metrics, err := rs.circuit.GetMetrics(c.Request.Context(), poaID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "get_metrics_failed", "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"poa_id":  poaID,
		"metrics": metrics,
	})
}

func (rs *RevocationService) handleResetMetrics(c *gin.Context) {
	var req struct {
		PoAID string `json:"poa_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid_request", "message": err.Error()})
		return
	}

	if err := rs.circuit.ResetMetrics(c.Request.Context(), req.PoAID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "reset_failed", "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"poa_id":  req.PoAID,
		"message": "Circuit breaker metrics reset.",
	})
}

func (rs *RevocationService) handleManualSuspend(c *gin.Context) {
	var req struct {
		PoAID  string `json:"poa_id" binding:"required"`
		Reason string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid_request", "message": err.Error()})
		return
	}

	if err := rs.circuit.ManualSuspend(c.Request.Context(), req.PoAID, req.Reason); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "suspend_failed", "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"poa_id":  req.PoAID,
		"status":  "suspended",
		"message": "Circuit breaker manually suspended.",
	})
}

func (rs *RevocationService) handleManualResume(c *gin.Context) {
	var req struct {
		PoAID string `json:"poa_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid_request", "message": err.Error()})
		return
	}

	if err := rs.circuit.ManualResume(c.Request.Context(), req.PoAID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "resume_failed", "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"poa_id":  req.PoAID,
		"status":  "resumed",
		"message": "Circuit breaker manually resumed.",
	})
}

func (rs *RevocationService) handleValidateTransaction(c *gin.Context) {
	var req struct {
		PoAID       string  `json:"poa_id" binding:"required"`
		TxID        string  `json:"tx_id" binding:"required"`
		TxValue     float64 `json:"tx_value" binding:"required"`
		TxType      string  `json:"tx_type"`
		FailureRate float64 `json:"failure_rate"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid_request", "message": err.Error()})
		return
	}

	ctx := c.Request.Context()

	// Check two-phase revocation status
	twoPhaseStatus, err := rs.twoPhase.GetStatus(ctx, req.PoAID)
	if err == nil && (twoPhaseStatus == "disabled" || twoPhaseStatus == "revoked") {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"error":   "poa_revoked",
			"status":  twoPhaseStatus,
			"message": "Transaction blocked: PoA is " + twoPhaseStatus,
		})
		return
	}

	// Check optimistic revocation status
	optimisticStatus, err := rs.optimistic.GetStatus(ctx, req.PoAID)
	if err == nil && (optimisticStatus == "pending" || optimisticStatus == "finalized") {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"error":   "poa_revoked",
			"status":  optimisticStatus,
			"message": "Transaction blocked: PoA revocation is " + optimisticStatus,
		})
		return
	}

	// Check circuit breaker
	allowed, err := rs.circuit.AllowTransaction(ctx, req.PoAID, req.TxValue, req.FailureRate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "validation_failed", "message": err.Error()})
		return
	}
	if !allowed {
		c.JSON(http.StatusTooManyRequests, gin.H{
			"success": false,
			"error":   "rate_limited",
			"message": "Transaction blocked by circuit breaker",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"poa_id":    req.PoAID,
		"tx_id":     req.TxID,
		"validated": true,
		"message":   "Transaction validated successfully",
	})
}

func (rs *RevocationService) handleHealth(c *gin.Context) {
	if !rs.enabled {
		c.JSON(http.StatusOK, gin.H{
			"enabled": false,
			"message": "Revocation system is disabled",
		})
		return
	}

	// Check Redis connection
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	
	redisHealthy := rs.redisClient.Ping(ctx).Err() == nil

	health := gin.H{
		"enabled": true,
		"redis":   redisHealthy,
		"components": gin.H{
			"oracle":     rs.oracle != nil,
			"two_phase":  rs.twoPhase != nil,
			"optimistic": rs.optimistic != nil,
			"circuit":    rs.circuit != nil,
		},
	}

	if !redisHealthy {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"success": false,
			"health":  health,
			"message": "Redis connection unhealthy",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"health":  health,
		"message": "Revocation system healthy",
	})
}

// GetStats returns current statistics for monitoring/metrics
func (rs *RevocationService) GetStats() map[string]interface{} {
	if !rs.enabled {
		return map[string]interface{}{
			"enabled": false,
		}
	}

	return map[string]interface{}{
		"enabled": true,
		"redis": map[string]interface{}{
			"pool_stats": rs.redisClient.PoolStats(),
		},
		"components": []string{
			"emergency_oracle",
			"two_phase_revocation",
			"optimistic_revocation",
			"circuit_breaker",
		},
	}
}

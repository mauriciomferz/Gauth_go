package admin

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mauriciomferz/Gauth_go/pkg/resilience"
)

// ResilienceHandler manages resilience patterns for the admin portal
type ResilienceHandler struct {
	repo *resilience.Repository
}

// NewResilienceHandler creates a new resilience handler instance
func NewResilienceHandler(db *pgxpool.Pool) *ResilienceHandler {
	return &ResilienceHandler{
		repo: resilience.NewRepository(db),
	}
}

// CircuitBreaker represents a circuit breaker configuration
type CircuitBreaker struct {
	ID               string  `json:"id"`
	Name             string  `json:"name"`
	Service          string  `json:"service"`
	State            string  `json:"state"` // closed, open, half-open
	FailureThreshold int     `json:"failureThreshold"`
	SuccessThreshold int     `json:"successThreshold"`
	Timeout          int     `json:"timeout"`
	Failures         int     `json:"failures"`
	Successes        int     `json:"successes"`
	LastStateChange  string  `json:"lastStateChange"`
	TotalRequests    int     `json:"totalRequests"`
	FailureRate      float64 `json:"failureRate"`
}

// RateLimiter represents a rate limiter configuration
type RateLimiter struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Resource      string `json:"resource"`
	Algorithm     string `json:"algorithm"` // token-bucket, leaky-bucket, fixed-window, sliding-window
	Limit         int    `json:"limit"`
	Window        int    `json:"window"`
	Burst         int    `json:"burst,omitempty"`
	Current       int    `json:"current"`
	Throttled     int    `json:"throttled"`
	TotalRequests int    `json:"totalRequests"`
}

// RetryPolicy represents a retry policy configuration
type RetryPolicy struct {
	ID                string `json:"id"`
	Name              string `json:"name"`
	Operation         string `json:"operation"`
	Strategy          string `json:"strategy"` // fixed, exponential, fibonacci, linear
	MaxAttempts       int    `json:"maxAttempts"`
	BaseDelay         int    `json:"baseDelay"`
	MaxDelay          int    `json:"maxDelay"`
	Jitter            bool   `json:"jitter"`
	TotalRetries      int    `json:"totalRetries"`
	SuccessfulRetries int    `json:"successfulRetries"`
	FailedRetries     int    `json:"failedRetries"`
}

// Bulkhead represents a bulkhead pattern configuration
type Bulkhead struct {
	ID                 string `json:"id"`
	Name               string `json:"name"`
	Service            string `json:"service"`
	MaxConcurrency     int    `json:"maxConcurrency"`
	MaxQueueSize       int    `json:"maxQueueSize"`
	CurrentConcurrency int    `json:"currentConcurrency"`
	QueuedRequests     int    `json:"queuedRequests"`
	RejectedRequests   int    `json:"rejectedRequests"`
	CompletedRequests  int    `json:"completedRequests"`
	Timeout            int    `json:"timeout"`
}

// CompositePattern represents a composite resilience pattern
type CompositePattern struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	Patterns     []string `json:"patterns"`
	Services     []string `json:"services"`
	Enabled      bool     `json:"enabled"`
	AppliedCount int      `json:"appliedCount"`
}

// CircuitBreakerRequest represents the request to create/update a circuit breaker
type CircuitBreakerRequest struct {
	Name             string `json:"name" binding:"required"`
	Service          string `json:"service" binding:"required"`
	FailureThreshold int    `json:"failureThreshold" binding:"required,min=1"`
	SuccessThreshold int    `json:"successThreshold" binding:"required,min=1"`
	Timeout          int    `json:"timeout" binding:"required,min=1000"`
}

// RateLimiterRequest represents the request to create a rate limiter
type RateLimiterRequest struct {
	Name      string `json:"name" binding:"required"`
	Resource  string `json:"resource" binding:"required"`
	Algorithm string `json:"algorithm" binding:"required,oneof=token-bucket leaky-bucket fixed-window sliding-window"`
	Limit     int    `json:"limit" binding:"required,min=1"`
	Window    int    `json:"window" binding:"required,min=1"`
	Burst     int    `json:"burst,omitempty"`
}

// RetryPolicyRequest represents the request to create a retry policy
type RetryPolicyRequest struct {
	Name        string `json:"name" binding:"required"`
	Operation   string `json:"operation" binding:"required"`
	Strategy    string `json:"strategy" binding:"required,oneof=fixed exponential fibonacci linear"`
	MaxAttempts int    `json:"maxAttempts" binding:"required,min=1,max=10"`
	BaseDelay   int    `json:"baseDelay" binding:"required,min=100"`
	MaxDelay    int    `json:"maxDelay" binding:"required"`
	Jitter      bool   `json:"jitter"`
}

// BulkheadRequest represents the request to create a bulkhead
type BulkheadRequest struct {
	Name           string `json:"name" binding:"required"`
	Service        string `json:"service" binding:"required"`
	MaxConcurrency int    `json:"maxConcurrency" binding:"required,min=1"`
	MaxQueueSize   int    `json:"maxQueueSize" binding:"required,min=0"`
	Timeout        int    `json:"timeout" binding:"required,min=1000"`
}

// CompositePatternRequest represents the request to create a composite pattern
type CompositePatternRequest struct {
	Name        string   `json:"name" binding:"required"`
	Description string   `json:"description"`
	Patterns    []string `json:"patterns" binding:"required,min=1"`
	Services    []string `json:"services" binding:"required,min=1"`
}

// ListCircuitBreakers returns all circuit breakers
// GET /api/admin/resilience/circuit-breakers
func (h *ResilienceHandler) ListCircuitBreakers(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tenant_id required"})
		return
	}

	breakers, err := h.repo.ListCircuitBreakers(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list circuit breakers"})
		return
	}

	// Convert to response format
	circuitBreakers := make([]CircuitBreaker, len(breakers))
	for i, b := range breakers {
		totalReqs := b.FailureCount + b.SuccessCount
		failureRate := 0.0
		if totalReqs > 0 {
			failureRate = float64(b.FailureCount) / float64(totalReqs) * 100
		}

		circuitBreakers[i] = CircuitBreaker{
			ID:               b.ID,
			Name:             b.BreakerName,
			Service:          b.ServiceName,
			State:            b.State,
			FailureThreshold: b.FailureThreshold,
			SuccessThreshold: b.SuccessThreshold,
			Timeout:          b.TimeoutSeconds * 1000, // convert to ms
			Failures:         b.FailureCount,
			Successes:        b.SuccessCount,
			LastStateChange:  b.LastStateChange.Format(time.RFC3339),
			TotalRequests:    totalReqs,
			FailureRate:      failureRate,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"circuitBreakers": circuitBreakers,
		"total":           len(circuitBreakers),
	})
}

// CreateCircuitBreaker creates a new circuit breaker
// POST /api/admin/resilience/circuit-breakers
func (h *ResilienceHandler) CreateCircuitBreaker(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tenant_id required"})
		return
	}

	var req CircuitBreakerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	breaker := &resilience.CircuitBreaker{
		TenantID:         tenantID,
		BreakerName:      req.Name,
		ServiceName:      req.Service,
		State:            "closed",
		FailureThreshold: req.FailureThreshold,
		SuccessThreshold: req.SuccessThreshold,
		TimeoutSeconds:   req.Timeout / 1000, // convert ms to seconds
	}

	if err := h.repo.CreateCircuitBreaker(c.Request.Context(), breaker); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create circuit breaker"})
		return
	}

	cb := CircuitBreaker{
		ID:               breaker.ID,
		Name:             breaker.BreakerName,
		Service:          breaker.ServiceName,
		State:            breaker.State,
		FailureThreshold: breaker.FailureThreshold,
		SuccessThreshold: breaker.SuccessThreshold,
		Timeout:          req.Timeout,
		Failures:         0,
		Successes:        0,
		LastStateChange:  breaker.CreatedAt.Format(time.RFC3339),
		TotalRequests:    0,
		FailureRate:      0.0,
	}

	c.JSON(http.StatusCreated, cb)
}

// UpdateCircuitBreaker updates a circuit breaker configuration
// PUT /api/admin/resilience/circuit-breakers/:id
func (h *ResilienceHandler) UpdateCircuitBreaker(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tenant_id required"})
		return
	}

	cbID := c.Param("id")

	var req CircuitBreakerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := h.repo.UpdateCircuitBreaker(c.Request.Context(), tenantID, cbID,
		req.FailureThreshold, req.SuccessThreshold, req.Timeout/1000)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"id":      cbID,
		"message": "Circuit breaker updated successfully",
	})
}

// ResetCircuitBreaker resets a circuit breaker to closed state
// POST /api/admin/resilience/circuit-breakers/:id/reset
func (h *ResilienceHandler) ResetCircuitBreaker(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tenant_id required"})
		return
	}

	cbID := c.Param("id")

	err := h.repo.ResetCircuitBreaker(c.Request.Context(), tenantID, cbID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"id":      cbID,
		"state":   "closed",
		"message": "Circuit breaker reset successfully",
	})
}

// DeleteCircuitBreaker removes a circuit breaker
// DELETE /api/admin/resilience/circuit-breakers/:id
func (h *ResilienceHandler) DeleteCircuitBreaker(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tenant_id required"})
		return
	}

	cbID := c.Param("id")

	err := h.repo.DeleteCircuitBreaker(c.Request.Context(), tenantID, cbID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Circuit breaker deleted successfully",
		"id":      cbID,
	})
}

// ListRateLimiters returns all rate limiters
// GET /api/admin/resilience/rate-limiters
func (h *ResilienceHandler) ListRateLimiters(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tenant_id required"})
		return
	}

	limiters, err := h.repo.ListRateLimiters(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list rate limiters"})
		return
	}

	rateLimiters := make([]RateLimiter, len(limiters))
	for i, l := range limiters {
		current := int(l.AllowedRequests % int64(l.MaxRequests))

		rateLimiters[i] = RateLimiter{
			ID:            l.ID,
			Name:          l.LimiterName,
			Resource:      l.Endpoint,
			Algorithm:     l.Algorithm,
			Limit:         l.MaxRequests,
			Window:        l.WindowSeconds,
			Current:       current,
			Throttled:     int(l.RejectedRequests),
			TotalRequests: int(l.TotalRequests),
		}
		if l.BurstSize != nil {
			rateLimiters[i].Burst = *l.BurstSize
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"rateLimiters": rateLimiters,
		"total":        len(rateLimiters),
	})
}

// CreateRateLimiter creates a new rate limiter
// POST /api/admin/resilience/rate-limiters
func (h *ResilienceHandler) CreateRateLimiter(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tenant_id required"})
		return
	}

	var req RateLimiterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Convert algorithm format
	algo := req.Algorithm
	if algo == "token-bucket" {
		algo = "token_bucket"
	} else if algo == "leaky-bucket" {
		algo = "leaky_bucket"
	} else if algo == "fixed-window" {
		algo = "fixed_window"
	} else if algo == "sliding-window" {
		algo = "sliding_window"
	}

	limiter := &resilience.RateLimiter{
		TenantID:      tenantID,
		LimiterName:   req.Name,
		Endpoint:      req.Resource,
		Algorithm:     algo,
		MaxRequests:   req.Limit,
		WindowSeconds: req.Window,
	}
	if req.Burst > 0 {
		limiter.BurstSize = &req.Burst
	}

	if err := h.repo.CreateRateLimiter(c.Request.Context(), limiter); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create rate limiter"})
		return
	}

	rl := RateLimiter{
		ID:            limiter.ID,
		Name:          limiter.LimiterName,
		Resource:      limiter.Endpoint,
		Algorithm:     req.Algorithm,
		Limit:         limiter.MaxRequests,
		Window:        limiter.WindowSeconds,
		Burst:         req.Burst,
		Current:       0,
		Throttled:     0,
		TotalRequests: 0,
	}

	c.JSON(http.StatusCreated, rl)
}

// DeleteRateLimiter removes a rate limiter
// DELETE /api/admin/resilience/rate-limiters/:id
func (h *ResilienceHandler) DeleteRateLimiter(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tenant_id required"})
		return
	}

	rlID := c.Param("id")

	err := h.repo.DeleteRateLimiter(c.Request.Context(), tenantID, rlID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Rate limiter deleted successfully",
		"id":      rlID,
	})
}

// ListRetryPolicies returns all retry policies
// GET /api/admin/resilience/retry-policies
func (h *ResilienceHandler) ListRetryPolicies(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tenant_id required"})
		return
	}

	policies, err := h.repo.ListRetryPolicies(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list retry policies"})
		return
	}

	retryPolicies := make([]RetryPolicy, len(policies))
	for i, p := range policies {
		retryPolicies[i] = RetryPolicy{
			ID:                p.ID,
			Name:              p.PolicyName,
			Operation:         p.ServiceName,
			Strategy:          p.BackoffType,
			MaxAttempts:       p.MaxAttempts,
			BaseDelay:         p.InitialDelayMs,
			MaxDelay:          p.MaxDelayMs,
			Jitter:            p.JitterEnabled,
			TotalRetries:      int(p.TotalRetries),
			SuccessfulRetries: int(p.SuccessfulRetries),
			FailedRetries:     int(p.FailedRetries),
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"retryPolicies": retryPolicies,
		"total":         len(retryPolicies),
	})
}

// CreateRetryPolicy creates a new retry policy
// POST /api/admin/resilience/retry-policies
func (h *ResilienceHandler) CreateRetryPolicy(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tenant_id required"})
		return
	}

	var req RetryPolicyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	policy := &resilience.RetryPolicy{
		TenantID:       tenantID,
		PolicyName:     req.Name,
		ServiceName:    req.Operation,
		MaxAttempts:    req.MaxAttempts,
		BackoffType:    req.Strategy,
		InitialDelayMs: req.BaseDelay,
		MaxDelayMs:     req.MaxDelay,
		Multiplier:     2.0,
		JitterEnabled:  req.Jitter,
	}

	if err := h.repo.CreateRetryPolicy(c.Request.Context(), policy); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create retry policy"})
		return
	}

	rp := RetryPolicy{
		ID:                policy.ID,
		Name:              policy.PolicyName,
		Operation:         policy.ServiceName,
		Strategy:          policy.BackoffType,
		MaxAttempts:       policy.MaxAttempts,
		BaseDelay:         policy.InitialDelayMs,
		MaxDelay:          policy.MaxDelayMs,
		Jitter:            policy.JitterEnabled,
		TotalRetries:      0,
		SuccessfulRetries: 0,
		FailedRetries:     0,
	}

	c.JSON(http.StatusCreated, rp)
}

// DeleteRetryPolicy removes a retry policy
// DELETE /api/admin/resilience/retry-policies/:id
func (h *ResilienceHandler) DeleteRetryPolicy(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tenant_id required"})
		return
	}

	rpID := c.Param("id")

	err := h.repo.DeleteRetryPolicy(c.Request.Context(), tenantID, rpID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Retry policy deleted successfully",
		"id":      rpID,
	})
}

// ListBulkheads returns all bulkheads
// GET /api/admin/resilience/bulkheads
func (h *ResilienceHandler) ListBulkheads(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tenant_id required"})
		return
	}

	bhs, err := h.repo.ListBulkheads(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list bulkheads"})
		return
	}

	bulkheads := make([]Bulkhead, len(bhs))
	for i, b := range bhs {
		bulkheads[i] = Bulkhead{
			ID:                 b.ID,
			Name:               b.BulkheadName,
			Service:            b.ServiceName,
			MaxConcurrency:     b.MaxConcurrent,
			MaxQueueSize:       b.MaxQueue,
			CurrentConcurrency: b.CurrentActive,
			QueuedRequests:     b.CurrentQueued,
			RejectedRequests:   int(b.TotalRejected),
			CompletedRequests:  int(b.TotalExecuted),
			Timeout:            b.TimeoutSeconds * 1000,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"bulkheads": bulkheads,
		"total":     len(bulkheads),
	})
}

// CreateBulkhead creates a new bulkhead
// POST /api/admin/resilience/bulkheads
func (h *ResilienceHandler) CreateBulkhead(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tenant_id required"})
		return
	}

	var req BulkheadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	bulkhead := &resilience.BulkheadRecord{
		TenantID:       tenantID,
		BulkheadName:   req.Name,
		ServiceName:    req.Service,
		MaxConcurrent:  req.MaxConcurrency,
		MaxQueue:       req.MaxQueueSize,
		TimeoutSeconds: req.Timeout / 1000,
	}

	if err := h.repo.CreateBulkhead(c.Request.Context(), bulkhead); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create bulkhead"})
		return
	}

	bh := Bulkhead{
		ID:                 bulkhead.ID,
		Name:               bulkhead.BulkheadName,
		Service:            bulkhead.ServiceName,
		MaxConcurrency:     bulkhead.MaxConcurrent,
		MaxQueueSize:       bulkhead.MaxQueue,
		CurrentConcurrency: 0,
		QueuedRequests:     0,
		RejectedRequests:   0,
		CompletedRequests:  0,
		Timeout:            req.Timeout,
	}

	c.JSON(http.StatusCreated, bh)
}

// DeleteBulkhead removes a bulkhead
// DELETE /api/admin/resilience/bulkheads/:id
func (h *ResilienceHandler) DeleteBulkhead(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tenant_id required"})
		return
	}

	bhID := c.Param("id")

	err := h.repo.DeleteBulkhead(c.Request.Context(), tenantID, bhID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Bulkhead deleted successfully",
		"id":      bhID,
	})
}

// ListCompositePatterns returns all composite patterns
// GET /api/admin/resilience/composite-patterns
func (h *ResilienceHandler) ListCompositePatterns(c *gin.Context) {
	// TODO: Composite patterns don't have database table yet
	// This is a logical combination of other patterns, may need separate implementation
	patterns := []CompositePattern{}

	c.JSON(http.StatusOK, gin.H{
		"patterns": patterns,
		"total":    0,
	})
}

// CreateCompositePattern creates a new composite pattern
// POST /api/admin/resilience/composite-patterns
func (h *ResilienceHandler) CreateCompositePattern(c *gin.Context) {
	var req CompositePatternRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// TODO: Register composite pattern
	cp := CompositePattern{
		ID:           "cp_" + generateID(),
		Name:         req.Name,
		Description:  req.Description,
		Patterns:     req.Patterns,
		Services:     req.Services,
		Enabled:      true,
		AppliedCount: 0,
	}

	c.JSON(http.StatusCreated, cp)
}

// ToggleCompositePattern enables or disables a composite pattern
// POST /api/admin/resilience/composite-patterns/:id/toggle
func (h *ResilienceHandler) ToggleCompositePattern(c *gin.Context) {
	cpID := c.Param("id")

	var req struct {
		Enabled bool `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// TODO: Enable/disable composite pattern
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"id":      cpID,
		"enabled": req.Enabled,
	})
}

// DeleteCompositePattern removes a composite pattern
// DELETE /api/admin/resilience/composite-patterns/:id
func (h *ResilienceHandler) DeleteCompositePattern(c *gin.Context) {
	cpID := c.Param("id")

	// TODO: Unregister composite pattern
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Composite pattern deleted successfully",
		"id":      cpID,
	})
}

// GetResilienceMetrics returns overall resilience metrics
// GET /api/admin/resilience/metrics
func (h *ResilienceHandler) GetResilienceMetrics(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tenant_id required"})
		return
	}

	stats, err := h.repo.GetResilienceStats(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get metrics"})
		return
	}

	metrics := gin.H{
		"circuit_breakers": gin.H{
			"total":            stats.CircuitBreakers.Total,
			"closed":           stats.CircuitBreakers.Closed,
			"open":             stats.CircuitBreakers.Open,
			"half_open":        stats.CircuitBreakers.HalfOpen,
			"total_requests":   stats.CircuitBreakers.TotalRequests,
			"avg_failure_rate": stats.CircuitBreakers.AvgFailureRate,
		},
		"rate_limiters": gin.H{
			"total":          stats.RateLimiters.Total,
			"total_requests": stats.RateLimiters.TotalRequests,
			"throttled":      stats.RateLimiters.Throttled,
			"throttle_rate":  stats.RateLimiters.ThrottleRate,
		},
		"retry_policies": gin.H{
			"total":              stats.RetryPolicies.Total,
			"total_retries":      stats.RetryPolicies.TotalRetries,
			"successful_retries": stats.RetryPolicies.SuccessfulRetries,
			"failed_retries":     stats.RetryPolicies.FailedRetries,
			"success_rate":       stats.RetryPolicies.SuccessRate,
		},
		"bulkheads": gin.H{
			"total":              stats.Bulkheads.Total,
			"total_concurrency":  stats.Bulkheads.TotalConcurrency,
			"max_concurrency":    stats.Bulkheads.MaxConcurrency,
			"completed_requests": stats.Bulkheads.CompletedRequests,
			"rejected_requests":  stats.Bulkheads.RejectedRequests,
			"avg_utilization":    stats.Bulkheads.AvgUtilization,
		},
		"composite_patterns": gin.H{
			"total":         0,
			"enabled":       0,
			"disabled":      0,
			"total_applied": 0,
		},
	}

	c.JSON(http.StatusOK, metrics)
}

// RegisterRoutes registers all resilience pattern routes
func (h *ResilienceHandler) RegisterRoutes(router *gin.RouterGroup) {
	resilience := router.Group("/resilience")
	{
		// Circuit Breakers
		resilience.GET("/circuit-breakers", h.ListCircuitBreakers)
		resilience.POST("/circuit-breakers", h.CreateCircuitBreaker)
		resilience.PUT("/circuit-breakers/:id", h.UpdateCircuitBreaker)
		resilience.POST("/circuit-breakers/:id/reset", h.ResetCircuitBreaker)
		resilience.DELETE("/circuit-breakers/:id", h.DeleteCircuitBreaker)

		// Rate Limiters
		resilience.GET("/rate-limiters", h.ListRateLimiters)
		resilience.POST("/rate-limiters", h.CreateRateLimiter)
		resilience.DELETE("/rate-limiters/:id", h.DeleteRateLimiter)

		// Retry Policies
		resilience.GET("/retry-policies", h.ListRetryPolicies)
		resilience.POST("/retry-policies", h.CreateRetryPolicy)
		resilience.DELETE("/retry-policies/:id", h.DeleteRetryPolicy)

		// Bulkheads
		resilience.GET("/bulkheads", h.ListBulkheads)
		resilience.POST("/bulkheads", h.CreateBulkhead)
		resilience.DELETE("/bulkheads/:id", h.DeleteBulkhead)

		// Composite Patterns
		resilience.GET("/composite-patterns", h.ListCompositePatterns)
		resilience.POST("/composite-patterns", h.CreateCompositePattern)
		resilience.POST("/composite-patterns/:id/toggle", h.ToggleCompositePattern)
		resilience.DELETE("/composite-patterns/:id", h.DeleteCompositePattern)

		// Metrics
		resilience.GET("/metrics", h.GetResilienceMetrics)
	}
}

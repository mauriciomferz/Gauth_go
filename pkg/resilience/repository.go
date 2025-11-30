package resilience

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository handles resilience pattern data operations
type Repository struct {
	db *pgxpool.Pool
}

// NewRepository creates a new resilience repository
func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// ============================================================================
// Circuit Breakers
// ============================================================================

// CircuitBreaker represents a circuit breaker configuration and state
type CircuitBreaker struct {
	ID                      string     `json:"id"`
	TenantID                string     `json:"tenantId"`
	BreakerName             string     `json:"breakerName"`
	ServiceName             string     `json:"serviceName"`
	State                   string     `json:"state"`
	FailureThreshold        int        `json:"failureThreshold"`
	SuccessThreshold        int        `json:"successThreshold"`
	TimeoutSeconds          int        `json:"timeoutSeconds"`
	HalfOpenMaxRequests     *int       `json:"halfOpenMaxRequests,omitempty"`
	FailureCount            int        `json:"failureCount"`
	SuccessCount            int        `json:"successCount"`
	ConsecutiveFailures     int        `json:"consecutiveFailures"`
	ConsecutiveSuccesses    int        `json:"consecutiveSuccesses"`
	LastFailureTime         *time.Time `json:"lastFailureTime,omitempty"`
	LastSuccessTime         *time.Time `json:"lastSuccessTime,omitempty"`
	LastStateChange         time.Time  `json:"lastStateChange"`
	CreatedAt               time.Time  `json:"createdAt"`
	UpdatedAt               time.Time  `json:"updatedAt"`
}

// CreateCircuitBreaker creates a new circuit breaker
func (r *Repository) CreateCircuitBreaker(ctx context.Context, cb *CircuitBreaker) error {
	query := `
		INSERT INTO circuit_breakers (
			tenant_id, breaker_name, service_name, state,
			failure_threshold, success_threshold, timeout_seconds, half_open_max_requests
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at, updated_at, last_state_change
	`
	
	err := r.db.QueryRow(ctx, query,
		cb.TenantID, cb.BreakerName, cb.ServiceName, cb.State,
		cb.FailureThreshold, cb.SuccessThreshold, cb.TimeoutSeconds, cb.HalfOpenMaxRequests,
	).Scan(&cb.ID, &cb.CreatedAt, &cb.UpdatedAt, &cb.LastStateChange)
	
	if err != nil {
		return fmt.Errorf("failed to create circuit breaker: %w", err)
	}
	
	return nil
}

// ListCircuitBreakers retrieves all circuit breakers for a tenant
func (r *Repository) ListCircuitBreakers(ctx context.Context, tenantID string) ([]CircuitBreaker, error) {
	query := `
		SELECT 
			id, tenant_id, breaker_name, service_name, state,
			failure_threshold, success_threshold, timeout_seconds, half_open_max_requests,
			failure_count, success_count, consecutive_failures, consecutive_successes,
			last_failure_time, last_success_time, last_state_change,
			created_at, updated_at
		FROM circuit_breakers
		WHERE tenant_id = $1
		ORDER BY breaker_name
	`
	
	rows, err := r.db.Query(ctx, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to list circuit breakers: %w", err)
	}
	defer rows.Close()
	
	var breakers []CircuitBreaker
	for rows.Next() {
		var cb CircuitBreaker
		err := rows.Scan(
			&cb.ID, &cb.TenantID, &cb.BreakerName, &cb.ServiceName, &cb.State,
			&cb.FailureThreshold, &cb.SuccessThreshold, &cb.TimeoutSeconds, &cb.HalfOpenMaxRequests,
			&cb.FailureCount, &cb.SuccessCount, &cb.ConsecutiveFailures, &cb.ConsecutiveSuccesses,
			&cb.LastFailureTime, &cb.LastSuccessTime, &cb.LastStateChange,
			&cb.CreatedAt, &cb.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan circuit breaker: %w", err)
		}
		breakers = append(breakers, cb)
	}
	
	return breakers, nil
}

// GetCircuitBreaker retrieves a specific circuit breaker
func (r *Repository) GetCircuitBreaker(ctx context.Context, tenantID, breakerID string) (*CircuitBreaker, error) {
	query := `
		SELECT 
			id, tenant_id, breaker_name, service_name, state,
			failure_threshold, success_threshold, timeout_seconds, half_open_max_requests,
			failure_count, success_count, consecutive_failures, consecutive_successes,
			last_failure_time, last_success_time, last_state_change,
			created_at, updated_at
		FROM circuit_breakers
		WHERE tenant_id = $1 AND id = $2
	`
	
	var cb CircuitBreaker
	err := r.db.QueryRow(ctx, query, tenantID, breakerID).Scan(
		&cb.ID, &cb.TenantID, &cb.BreakerName, &cb.ServiceName, &cb.State,
		&cb.FailureThreshold, &cb.SuccessThreshold, &cb.TimeoutSeconds, &cb.HalfOpenMaxRequests,
		&cb.FailureCount, &cb.SuccessCount, &cb.ConsecutiveFailures, &cb.ConsecutiveSuccesses,
		&cb.LastFailureTime, &cb.LastSuccessTime, &cb.LastStateChange,
		&cb.CreatedAt, &cb.UpdatedAt,
	)
	
	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("circuit breaker not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get circuit breaker: %w", err)
	}
	
	return &cb, nil
}

// UpdateCircuitBreaker updates circuit breaker configuration
func (r *Repository) UpdateCircuitBreaker(ctx context.Context, tenantID, breakerID string, failureThreshold, successThreshold, timeoutSeconds int) error {
	query := `
		UPDATE circuit_breakers
		SET failure_threshold = $3,
		    success_threshold = $4,
		    timeout_seconds = $5,
		    updated_at = CURRENT_TIMESTAMP
		WHERE tenant_id = $1 AND id = $2
		RETURNING id
	`
	
	var returnedID string
	err := r.db.QueryRow(ctx, query, tenantID, breakerID, failureThreshold, successThreshold, timeoutSeconds).Scan(&returnedID)
	
	if err == pgx.ErrNoRows {
		return fmt.Errorf("circuit breaker not found")
	}
	if err != nil {
		return fmt.Errorf("failed to update circuit breaker: %w", err)
	}
	
	return nil
}

// ResetCircuitBreaker resets a circuit breaker to closed state
func (r *Repository) ResetCircuitBreaker(ctx context.Context, tenantID, breakerID string) error {
	query := `
		UPDATE circuit_breakers
		SET state = 'closed',
		    failure_count = 0,
		    success_count = 0,
		    consecutive_failures = 0,
		    consecutive_successes = 0,
		    last_state_change = CURRENT_TIMESTAMP,
		    updated_at = CURRENT_TIMESTAMP
		WHERE tenant_id = $1 AND id = $2
		RETURNING id
	`
	
	var returnedID string
	err := r.db.QueryRow(ctx, query, tenantID, breakerID).Scan(&returnedID)
	
	if err == pgx.ErrNoRows {
		return fmt.Errorf("circuit breaker not found")
	}
	if err != nil {
		return fmt.Errorf("failed to reset circuit breaker: %w", err)
	}
	
	return nil
}

// DeleteCircuitBreaker removes a circuit breaker
func (r *Repository) DeleteCircuitBreaker(ctx context.Context, tenantID, breakerID string) error {
	query := `DELETE FROM circuit_breakers WHERE tenant_id = $1 AND id = $2`
	
	result, err := r.db.Exec(ctx, query, tenantID, breakerID)
	if err != nil {
		return fmt.Errorf("failed to delete circuit breaker: %w", err)
	}
	
	if result.RowsAffected() == 0 {
		return fmt.Errorf("circuit breaker not found")
	}
	
	return nil
}

// ============================================================================
// Rate Limiters
// ============================================================================

// RateLimiter represents a rate limiter configuration
type RateLimiter struct {
	ID               string     `json:"id"`
	TenantID         string     `json:"tenantId"`
	LimiterName      string     `json:"limiterName"`
	Endpoint         string     `json:"endpoint"`
	Algorithm        string     `json:"algorithm"`
	MaxRequests      int        `json:"maxRequests"`
	WindowSeconds    int        `json:"windowSeconds"`
	BurstSize        *int       `json:"burstSize,omitempty"`
	TotalRequests    int64      `json:"totalRequests"`
	AllowedRequests  int64      `json:"allowedRequests"`
	RejectedRequests int64      `json:"rejectedRequests"`
	LastRequestAt    *time.Time `json:"lastRequestAt,omitempty"`
	CreatedAt        time.Time  `json:"createdAt"`
}

// CreateRateLimiter creates a new rate limiter
func (r *Repository) CreateRateLimiter(ctx context.Context, rl *RateLimiter) error {
	query := `
		INSERT INTO rate_limiters (
			tenant_id, limiter_name, endpoint, algorithm,
			max_requests, window_seconds, burst_size
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at
	`
	
	err := r.db.QueryRow(ctx, query,
		rl.TenantID, rl.LimiterName, rl.Endpoint, rl.Algorithm,
		rl.MaxRequests, rl.WindowSeconds, rl.BurstSize,
	).Scan(&rl.ID, &rl.CreatedAt)
	
	if err != nil {
		return fmt.Errorf("failed to create rate limiter: %w", err)
	}
	
	return nil
}

// ListRateLimiters retrieves all rate limiters for a tenant
func (r *Repository) ListRateLimiters(ctx context.Context, tenantID string) ([]RateLimiter, error) {
	query := `
		SELECT 
			id, tenant_id, limiter_name, endpoint, algorithm,
			max_requests, window_seconds, burst_size,
			total_requests, allowed_requests, rejected_requests, last_request_at,
			created_at
		FROM rate_limiters
		WHERE tenant_id = $1
		ORDER BY limiter_name
	`
	
	rows, err := r.db.Query(ctx, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to list rate limiters: %w", err)
	}
	defer rows.Close()
	
	var limiters []RateLimiter
	for rows.Next() {
		var rl RateLimiter
		err := rows.Scan(
			&rl.ID, &rl.TenantID, &rl.LimiterName, &rl.Endpoint, &rl.Algorithm,
			&rl.MaxRequests, &rl.WindowSeconds, &rl.BurstSize,
			&rl.TotalRequests, &rl.AllowedRequests, &rl.RejectedRequests, &rl.LastRequestAt,
			&rl.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan rate limiter: %w", err)
		}
		limiters = append(limiters, rl)
	}
	
	return limiters, nil
}

// DeleteRateLimiter removes a rate limiter
func (r *Repository) DeleteRateLimiter(ctx context.Context, tenantID, limiterID string) error {
	query := `DELETE FROM rate_limiters WHERE tenant_id = $1 AND id = $2`
	
	result, err := r.db.Exec(ctx, query, tenantID, limiterID)
	if err != nil {
		return fmt.Errorf("failed to delete rate limiter: %w", err)
	}
	
	if result.RowsAffected() == 0 {
		return fmt.Errorf("rate limiter not found")
	}
	
	return nil
}

// ============================================================================
// Retry Policies
// ============================================================================

// RetryPolicy represents a retry policy configuration
type RetryPolicy struct {
	ID                string    `json:"id"`
	TenantID          string    `json:"tenantId"`
	PolicyName        string    `json:"policyName"`
	ServiceName       string    `json:"serviceName"`
	MaxAttempts       int       `json:"maxAttempts"`
	BackoffType       string    `json:"backoffType"`
	InitialDelayMs    int       `json:"initialDelayMs"`
	MaxDelayMs        int       `json:"maxDelayMs"`
	Multiplier        float64   `json:"multiplier"`
	JitterEnabled     bool      `json:"jitterEnabled"`
	RetryableErrors   []string  `json:"retryableErrors"`
	TotalRetries      int64     `json:"totalRetries"`
	SuccessfulRetries int64     `json:"successfulRetries"`
	FailedRetries     int64     `json:"failedRetries"`
	CreatedAt         time.Time `json:"createdAt"`
}

// CreateRetryPolicy creates a new retry policy
func (r *Repository) CreateRetryPolicy(ctx context.Context, rp *RetryPolicy) error {
	query := `
		INSERT INTO retry_policies (
			tenant_id, policy_name, service_name, max_attempts,
			backoff_type, initial_delay_ms, max_delay_ms, multiplier,
			jitter_enabled, retryable_errors
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, created_at
	`
	
	err := r.db.QueryRow(ctx, query,
		rp.TenantID, rp.PolicyName, rp.ServiceName, rp.MaxAttempts,
		rp.BackoffType, rp.InitialDelayMs, rp.MaxDelayMs, rp.Multiplier,
		rp.JitterEnabled, rp.RetryableErrors,
	).Scan(&rp.ID, &rp.CreatedAt)
	
	if err != nil {
		return fmt.Errorf("failed to create retry policy: %w", err)
	}
	
	return nil
}

// ListRetryPolicies retrieves all retry policies for a tenant
func (r *Repository) ListRetryPolicies(ctx context.Context, tenantID string) ([]RetryPolicy, error) {
	query := `
		SELECT 
			id, tenant_id, policy_name, service_name, max_attempts,
			backoff_type, initial_delay_ms, max_delay_ms, multiplier,
			jitter_enabled, retryable_errors,
			total_retries, successful_retries, failed_retries,
			created_at
		FROM retry_policies
		WHERE tenant_id = $1
		ORDER BY policy_name
	`
	
	rows, err := r.db.Query(ctx, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to list retry policies: %w", err)
	}
	defer rows.Close()
	
	var policies []RetryPolicy
	for rows.Next() {
		var rp RetryPolicy
		err := rows.Scan(
			&rp.ID, &rp.TenantID, &rp.PolicyName, &rp.ServiceName, &rp.MaxAttempts,
			&rp.BackoffType, &rp.InitialDelayMs, &rp.MaxDelayMs, &rp.Multiplier,
			&rp.JitterEnabled, &rp.RetryableErrors,
			&rp.TotalRetries, &rp.SuccessfulRetries, &rp.FailedRetries,
			&rp.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan retry policy: %w", err)
		}
		policies = append(policies, rp)
	}
	
	return policies, nil
}

// DeleteRetryPolicy removes a retry policy
func (r *Repository) DeleteRetryPolicy(ctx context.Context, tenantID, policyID string) error {
	query := `DELETE FROM retry_policies WHERE tenant_id = $1 AND id = $2`
	
	result, err := r.db.Exec(ctx, query, tenantID, policyID)
	if err != nil {
		return fmt.Errorf("failed to delete retry policy: %w", err)
	}
	
	if result.RowsAffected() == 0 {
		return fmt.Errorf("retry policy not found")
	}
	
	return nil
}

// ============================================================================
// Bulkheads
// ============================================================================

// BulkheadRecord represents a bulkhead configuration stored in the database
type BulkheadRecord struct {
	ID              string    `json:"id"`
	TenantID        string    `json:"tenantId"`
	BulkheadName    string    `json:"bulkheadName"`
	ServiceName     string    `json:"serviceName"`
	MaxConcurrent   int       `json:"maxConcurrent"`
	MaxQueue        int       `json:"maxQueue"`
	TimeoutSeconds  int       `json:"timeoutSeconds"`
	CurrentActive   int       `json:"currentActive"`
	CurrentQueued   int       `json:"currentQueued"`
	TotalExecuted   int64     `json:"totalExecuted"`
	TotalRejected   int64     `json:"totalRejected"`
	TotalTimeout    int64     `json:"totalTimeout"`
	PeakConcurrent  int       `json:"peakConcurrent"`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
}

// CreateBulkhead creates a new bulkhead
func (r *Repository) CreateBulkhead(ctx context.Context, bh *BulkheadRecord) error {
	query := `
		INSERT INTO bulkheads (
			tenant_id, bulkhead_name, service_name,
			max_concurrent, max_queue, timeout_seconds
		) VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at, updated_at
	`
	
	err := r.db.QueryRow(ctx, query,
		bh.TenantID, bh.BulkheadName, bh.ServiceName,
		bh.MaxConcurrent, bh.MaxQueue, bh.TimeoutSeconds,
	).Scan(&bh.ID, &bh.CreatedAt, &bh.UpdatedAt)
	
	if err != nil {
		return fmt.Errorf("failed to create bulkhead: %w", err)
	}
	
	return nil
}

// ListBulkheads retrieves all bulkheads for a tenant
func (r *Repository) ListBulkheads(ctx context.Context, tenantID string) ([]BulkheadRecord, error) {
	query := `
		SELECT 
			id, tenant_id, bulkhead_name, service_name,
			max_concurrent, max_queue, timeout_seconds,
			current_active, current_queued, total_executed, total_rejected, total_timeout,
			peak_concurrent, created_at, updated_at
		FROM bulkheads
		WHERE tenant_id = $1
		ORDER BY bulkhead_name
	`
	
	rows, err := r.db.Query(ctx, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to list bulkheads: %w", err)
	}
	defer rows.Close()
	
	var bulkheads []BulkheadRecord
	for rows.Next() {
		var bh BulkheadRecord
		err := rows.Scan(
			&bh.ID, &bh.TenantID, &bh.BulkheadName, &bh.ServiceName,
			&bh.MaxConcurrent, &bh.MaxQueue, &bh.TimeoutSeconds,
			&bh.CurrentActive, &bh.CurrentQueued, &bh.TotalExecuted, &bh.TotalRejected, &bh.TotalTimeout,
			&bh.PeakConcurrent, &bh.CreatedAt, &bh.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan bulkhead: %w", err)
		}
		bulkheads = append(bulkheads, bh)
	}
	
	return bulkheads, nil
}

// DeleteBulkhead removes a bulkhead
func (r *Repository) DeleteBulkhead(ctx context.Context, tenantID, bulkheadID string) error {
	query := `DELETE FROM bulkheads WHERE tenant_id = $1 AND id = $2`
	
	result, err := r.db.Exec(ctx, query, tenantID, bulkheadID)
	if err != nil {
		return fmt.Errorf("failed to delete bulkhead: %w", err)
	}
	
	if result.RowsAffected() == 0 {
		return fmt.Errorf("bulkhead not found")
	}
	
	return nil
}

// ============================================================================
// Statistics
// ============================================================================

// ResilienceStats represents aggregate statistics for all resilience patterns
type ResilienceStats struct {
	CircuitBreakers CircuitBreakerStats `json:"circuitBreakers"`
	RateLimiters    RateLimiterStats    `json:"rateLimiters"`
	RetryPolicies   RetryPolicyStats    `json:"retryPolicies"`
	Bulkheads       BulkheadStats       `json:"bulkheads"`
}

type CircuitBreakerStats struct {
	Total          int     `json:"total"`
	Closed         int     `json:"closed"`
	Open           int     `json:"open"`
	HalfOpen       int     `json:"halfOpen"`
	TotalRequests  int64   `json:"totalRequests"`
	AvgFailureRate float64 `json:"avgFailureRate"`
}

type RateLimiterStats struct {
	Total          int     `json:"total"`
	TotalRequests  int64   `json:"totalRequests"`
	Throttled      int64   `json:"throttled"`
	ThrottleRate   float64 `json:"throttleRate"`
}

type RetryPolicyStats struct {
	Total             int     `json:"total"`
	TotalRetries      int64   `json:"totalRetries"`
	SuccessfulRetries int64   `json:"successfulRetries"`
	FailedRetries     int64   `json:"failedRetries"`
	SuccessRate       float64 `json:"successRate"`
}

type BulkheadStats struct {
	Total              int     `json:"total"`
	TotalConcurrency   int     `json:"totalConcurrency"`
	MaxConcurrency     int     `json:"maxConcurrency"`
	CompletedRequests  int64   `json:"completedRequests"`
	RejectedRequests   int64   `json:"rejectedRequests"`
	AvgUtilization     float64 `json:"avgUtilization"`
}

// GetResilienceStats retrieves aggregate statistics for all resilience patterns
func (r *Repository) GetResilienceStats(ctx context.Context, tenantID string) (*ResilienceStats, error) {
	stats := &ResilienceStats{}
	
	// Circuit breaker stats
	cbQuery := `
		SELECT 
			COUNT(*) as total,
			COUNT(*) FILTER (WHERE state = 'closed') as closed,
			COUNT(*) FILTER (WHERE state = 'open') as open,
			COUNT(*) FILTER (WHERE state = 'half-open') as half_open,
			COALESCE(SUM(failure_count + success_count), 0) as total_requests
		FROM circuit_breakers
		WHERE tenant_id = $1
	`
	
	var totalRequests int64
	err := r.db.QueryRow(ctx, cbQuery, tenantID).Scan(
		&stats.CircuitBreakers.Total,
		&stats.CircuitBreakers.Closed,
		&stats.CircuitBreakers.Open,
		&stats.CircuitBreakers.HalfOpen,
		&totalRequests,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get circuit breaker stats: %w", err)
	}
	stats.CircuitBreakers.TotalRequests = totalRequests
	
	// Calculate average failure rate
	if totalRequests > 0 {
		var totalFailures int64
		if err := r.db.QueryRow(ctx, `SELECT COALESCE(SUM(failure_count), 0) FROM circuit_breakers WHERE tenant_id = $1`, tenantID).Scan(&totalFailures); err == nil {
			stats.CircuitBreakers.AvgFailureRate = float64(totalFailures) / float64(totalRequests) * 100
		}
	}
	
	// Rate limiter stats
	rlQuery := `
		SELECT 
			COUNT(*) as total,
			COALESCE(SUM(total_requests), 0) as total_requests,
			COALESCE(SUM(rejected_requests), 0) as throttled
		FROM rate_limiters
		WHERE tenant_id = $1
	`
	
	err = r.db.QueryRow(ctx, rlQuery, tenantID).Scan(
		&stats.RateLimiters.Total,
		&stats.RateLimiters.TotalRequests,
		&stats.RateLimiters.Throttled,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get rate limiter stats: %w", err)
	}
	
	if stats.RateLimiters.TotalRequests > 0 {
		stats.RateLimiters.ThrottleRate = float64(stats.RateLimiters.Throttled) / float64(stats.RateLimiters.TotalRequests) * 100
	}
	
	// Retry policy stats
	rpQuery := `
		SELECT 
			COUNT(*) as total,
			COALESCE(SUM(total_retries), 0) as total_retries,
			COALESCE(SUM(successful_retries), 0) as successful_retries,
			COALESCE(SUM(failed_retries), 0) as failed_retries
		FROM retry_policies
		WHERE tenant_id = $1
	`
	
	err = r.db.QueryRow(ctx, rpQuery, tenantID).Scan(
		&stats.RetryPolicies.Total,
		&stats.RetryPolicies.TotalRetries,
		&stats.RetryPolicies.SuccessfulRetries,
		&stats.RetryPolicies.FailedRetries,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get retry policy stats: %w", err)
	}
	
	if stats.RetryPolicies.TotalRetries > 0 {
		stats.RetryPolicies.SuccessRate = float64(stats.RetryPolicies.SuccessfulRetries) / float64(stats.RetryPolicies.TotalRetries) * 100
	}
	
	// Bulkhead stats
	bhQuery := `
		SELECT 
			COUNT(*) as total,
			COALESCE(SUM(current_active), 0) as total_concurrency,
			COALESCE(SUM(max_concurrent), 0) as max_concurrency,
			COALESCE(SUM(total_executed), 0) as completed_requests,
			COALESCE(SUM(total_rejected), 0) as rejected_requests
		FROM bulkheads
		WHERE tenant_id = $1
	`
	
	err = r.db.QueryRow(ctx, bhQuery, tenantID).Scan(
		&stats.Bulkheads.Total,
		&stats.Bulkheads.TotalConcurrency,
		&stats.Bulkheads.MaxConcurrency,
		&stats.Bulkheads.CompletedRequests,
		&stats.Bulkheads.RejectedRequests,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get bulkhead stats: %w", err)
	}
	
	if stats.Bulkheads.MaxConcurrency > 0 {
		stats.Bulkheads.AvgUtilization = float64(stats.Bulkheads.TotalConcurrency) / float64(stats.Bulkheads.MaxConcurrency) * 100
	}
	
	return stats, nil
}

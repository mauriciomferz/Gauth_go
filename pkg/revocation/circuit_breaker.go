package revocation

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
)

// CircuitBreakerState represents the state of the circuit breaker
type CircuitBreakerState string

const (
	// CircuitBreakerClosed indicates normal operation (PoA usable)
	CircuitBreakerClosed CircuitBreakerState = "CLOSED"

	// CircuitBreakerOpen indicates circuit is open (PoA suspended)
	CircuitBreakerOpen CircuitBreakerState = "OPEN"

	// CircuitBreakerHalfOpen indicates circuit is testing (limited transactions allowed)
	CircuitBreakerHalfOpen CircuitBreakerState = "HALF_OPEN"
)

// SuspensionReason categorizes why a PoA was suspended
type SuspensionReason string

const (
	// SuspensionRateLimitTx indicates too many transactions per time window
	SuspensionRateLimitTx SuspensionReason = "RATE_LIMIT_TX"

	// SuspensionRateLimitValue indicates too much value transferred per time window
	SuspensionRateLimitValue SuspensionReason = "RATE_LIMIT_VALUE"

	// SuspensionAnomalousPattern indicates unusual transaction patterns detected
	SuspensionAnomalousPattern SuspensionReason = "ANOMALOUS_PATTERN"

	// SuspensionFailureThreshold indicates too many failed transactions
	SuspensionFailureThreshold SuspensionReason = "FAILURE_THRESHOLD"
)

// RateLimitConfig defines rate limiting thresholds
type RateLimitConfig struct {
	MaxTxPerMinute    int     `json:"max_tx_per_minute"`    // Maximum transactions per minute
	MaxTxPerHour      int     `json:"max_tx_per_hour"`      // Maximum transactions per hour
	MaxValuePerMinute uint64  `json:"max_value_per_minute"` // Maximum Wei per minute
	MaxValuePerHour   uint64  `json:"max_value_per_hour"`   // Maximum Wei per hour
	MaxFailureRate    float64 `json:"max_failure_rate"`     // Maximum failure rate (0.0-1.0)
	FailureWindowSecs int     `json:"failure_window_secs"`  // Window for failure rate calculation
}

// CircuitBreakerMetrics tracks PoA activity metrics
type CircuitBreakerMetrics struct {
	mu                  sync.Mutex          `json:"-"` // Protects all fields from concurrent access
	PoAID               string              `json:"poa_id"`
	State               CircuitBreakerState `json:"state"`
	TxCountLastMinute   int                 `json:"tx_count_last_minute"`
	TxCountLastHour     int                 `json:"tx_count_last_hour"`
	ValueLastMinute     uint64              `json:"value_last_minute"` // Wei
	ValueLastHour       uint64              `json:"value_last_hour"`   // Wei
	FailedTxCount       int                 `json:"failed_tx_count"`
	TotalTxCount        int                 `json:"total_tx_count"`
	LastTxTimestamp     time.Time           `json:"last_tx_timestamp"`
	SuspendedAt         time.Time           `json:"suspended_at,omitempty"`
	SuspensionReason    SuspensionReason    `json:"suspension_reason,omitempty"`
	RecoveryAttemptedAt time.Time           `json:"recovery_attempted_at,omitempty"`
	TestTxAllowed       int                 `json:"test_tx_allowed"` // In HALF_OPEN state
}

// CircuitBreaker implements circuit breaker pattern with rate limiting
// Automatically suspends PoAs that exhibit suspicious behavior patterns
type CircuitBreaker struct {
	redis              redis.UniversalClient // Supports both regular and cluster clients
	logger             Logger
	config             *RateLimitConfig
	metrics            sync.Map      // poaID → *CircuitBreakerMetrics
	suspensionDuration time.Duration // How long to keep circuit open
	recoveryTestCount  int           // Number of test transactions in HALF_OPEN
}

// NewCircuitBreaker creates a new circuit breaker with rate limiting
func NewCircuitBreaker(redisAddrs []string, config *RateLimitConfig, logger Logger) (*CircuitBreaker, error) {
	if len(redisAddrs) == 0 {
		return nil, fmt.Errorf("at least one Redis address required")
	}
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	rdb := redis.NewClusterClient(&redis.ClusterOptions{
		Addrs:           redisAddrs,
		MaxRetries:      3,
		MinRetryBackoff: 8 * time.Millisecond,
		MaxRetryBackoff: 512 * time.Millisecond,
		DialTimeout:     5 * time.Second,
		ReadTimeout:     3 * time.Second,
		WriteTimeout:    3 * time.Second,
		PoolSize:        100,
		MinIdleConns:    10,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis cluster ping failed: %w", err)
	}

	cb := &CircuitBreaker{
		redis:              rdb,
		logger:             logger,
		config:             config,
		suspensionDuration: 5 * time.Minute, // Default: 5 minutes
		recoveryTestCount:  10,              // Default: 10 test transactions
	}

	logger.Info("Circuit Breaker system initialized")
	return cb, nil
}

// RecordTransaction records a transaction and checks rate limits
// Returns error if rate limits are exceeded (circuit opens)
func (cb *CircuitBreaker) RecordTransaction(ctx context.Context, poaID string, value uint64, success bool) error {
	start := time.Now()

	// Get or create metrics
	metrics, err := cb.getOrCreateMetrics(ctx, poaID)
	if err != nil {
		return fmt.Errorf("failed to get metrics: %w", err)
	}

	// Lock metrics for thread-safe access
	metrics.mu.Lock()
	defer metrics.mu.Unlock()

	// Check if circuit is open
	if metrics.State == CircuitBreakerOpen {
		// Check if suspension period has elapsed
		if time.Since(metrics.SuspendedAt) >= cb.suspensionDuration {
			// Move to HALF_OPEN for recovery testing - reset metrics
			cb.logger.Infof("Circuit for PoA %s moving to HALF_OPEN (testing recovery)", poaID)
			metrics.State = CircuitBreakerHalfOpen
			metrics.RecoveryAttemptedAt = time.Now()
			metrics.TestTxAllowed = cb.recoveryTestCount
			// Reset counters for clean slate
			metrics.TxCountLastMinute = 0
			metrics.TxCountLastHour = 0
			metrics.ValueLastMinute = 0
			metrics.ValueLastHour = 0
			metrics.FailedTxCount = 0
			metrics.TotalTxCount = 0
			// Clear cache and store reset metrics immediately
			cb.metrics.Delete(poaID)
			if storeErr := cb.storeMetrics(ctx, metrics); storeErr != nil {
				cb.logger.Errorf("Failed to store reset metrics (non-fatal): %v", storeErr)
			}
			// Reload metrics from storage to get fresh slate
			metrics, err = cb.getOrCreateMetrics(ctx, poaID)
			if err != nil {
				return fmt.Errorf("failed to reload metrics after reset: %w", err)
			}
			// Allow this transaction without further checks (transition call)
			// The next call will use the reset metrics
			metrics.LastTxTimestamp = time.Now()
			metrics.TotalTxCount++
			if success {
				metrics.TxCountLastMinute++
				metrics.TxCountLastHour++
				metrics.ValueLastMinute += value
				metrics.ValueLastHour += value
			} else {
				metrics.FailedTxCount++
			}
			if err := cb.storeMetrics(ctx, metrics); err != nil {
				cb.logger.Errorf("Failed to store metrics (non-fatal): %v", err)
			}
			return nil
		} else {
			return fmt.Errorf("circuit breaker OPEN for PoA %s (suspended: %s, reason: %s)",
				poaID, metrics.SuspensionReason, metrics.SuspensionReason)
		}
	}

	// In HALF_OPEN state, only allow limited test transactions
	if metrics.State == CircuitBreakerHalfOpen {
		// Decrement remaining test transactions
		metrics.TestTxAllowed--
		cb.logger.Infof("Circuit HALF_OPEN for PoA %s: test tx %d/%d",
			poaID, cb.recoveryTestCount-metrics.TestTxAllowed, cb.recoveryTestCount)

		// Check if all test transactions completed
		if metrics.TestTxAllowed <= 0 {
			// All test transactions completed successfully - close circuit
			cb.logger.Infof("Circuit for PoA %s CLOSED (recovery successful)", poaID)
			metrics.State = CircuitBreakerClosed
			metrics.SuspensionReason = ""
			metrics.SuspendedAt = time.Time{}
			// Reset metrics for fresh start (recovery proven successful)
			metrics.TxCountLastMinute = 0
			metrics.TxCountLastHour = 0
			metrics.ValueLastMinute = 0
			metrics.ValueLastHour = 0
			metrics.FailedTxCount = 0
			metrics.TotalTxCount = 0
		}
	}

	// Update metrics
	now := time.Now()
	metrics.LastTxTimestamp = now
	metrics.TotalTxCount++

	if success {
		// Update counters (use sliding windows)
		metrics.TxCountLastMinute++
		metrics.TxCountLastHour++
		metrics.ValueLastMinute += value
		metrics.ValueLastHour += value
	} else {
		metrics.FailedTxCount++
	}

	// Check rate limits ONLY if circuit is CLOSED
	// In HALF_OPEN state, we're testing recovery - rate limits are suspended
	if metrics.State == CircuitBreakerClosed {
		if err := cb.checkRateLimits(ctx, poaID, metrics); err != nil {
			// Rate limit exceeded - open circuit
			cb.logger.Warnf("Rate limit exceeded for PoA %s: %v", poaID, err)
			return err
		}
	}

	// Store updated metrics
	if err := cb.storeMetrics(ctx, metrics); err != nil {
		cb.logger.Errorf("Failed to store metrics (non-fatal): %v", err)
	}

	_ = time.Since(start)

	return nil
}

// checkRateLimits validates all rate limit thresholds
func (cb *CircuitBreaker) checkRateLimits(ctx context.Context, poaID string, metrics *CircuitBreakerMetrics) error {
	// Check transaction rate (per minute)
	if metrics.TxCountLastMinute > cb.config.MaxTxPerMinute {
		cb.openCircuit(ctx, poaID, metrics, SuspensionRateLimitTx)
		return fmt.Errorf("transaction rate limit exceeded: %d tx/min (max: %d)",
			metrics.TxCountLastMinute, cb.config.MaxTxPerMinute)
	}

	// Check transaction rate (per hour)
	if metrics.TxCountLastHour > cb.config.MaxTxPerHour {
		cb.openCircuit(ctx, poaID, metrics, SuspensionRateLimitTx)
		return fmt.Errorf("transaction rate limit exceeded: %d tx/hour (max: %d)",
			metrics.TxCountLastHour, cb.config.MaxTxPerHour)
	}

	// Check value rate (per minute)
	if metrics.ValueLastMinute > cb.config.MaxValuePerMinute {
		cb.openCircuit(ctx, poaID, metrics, SuspensionRateLimitValue)
		return fmt.Errorf("value rate limit exceeded: %d Wei/min (max: %d)",
			metrics.ValueLastMinute, cb.config.MaxValuePerMinute)
	}

	// Check value rate (per hour)
	if metrics.ValueLastHour > cb.config.MaxValuePerHour {
		cb.openCircuit(ctx, poaID, metrics, SuspensionRateLimitValue)
		return fmt.Errorf("value rate limit exceeded: %d Wei/hour (max: %d)",
			metrics.ValueLastHour, cb.config.MaxValuePerHour)
	}

	// Check failure rate
	if metrics.TotalTxCount >= 10 { // Need minimum sample size
		failureRate := float64(metrics.FailedTxCount) / float64(metrics.TotalTxCount)
		if failureRate > cb.config.MaxFailureRate {
			cb.openCircuit(ctx, poaID, metrics, SuspensionFailureThreshold)
			return fmt.Errorf("failure rate exceeded: %.2f%% (max: %.2f%%)",
				failureRate*100, cb.config.MaxFailureRate*100)
		}
	}

	return nil
}

// openCircuit suspends a PoA by opening its circuit breaker
func (cb *CircuitBreaker) openCircuit(
	ctx context.Context,
	poaID string,
	metrics *CircuitBreakerMetrics,
	reason SuspensionReason,
) {
	cb.logger.Warnf("🔴 Opening circuit for PoA %s (reason: %s)", poaID, reason)

	metrics.State = CircuitBreakerOpen
	metrics.SuspendedAt = time.Now()
	metrics.SuspensionReason = reason

	// Store suspension event
	if err := cb.storeMetrics(ctx, metrics); err != nil {
		cb.logger.Errorf("Failed to store suspension (non-fatal): %v", err)
	}
}

// IsPoAAllowed checks if a PoA can execute transactions
func (cb *CircuitBreaker) IsPoAAllowed(ctx context.Context, poaID string) (bool, string, error) {
	if poaID == "" {
		return false, "", fmt.Errorf("poaID cannot be empty")
	}

	metrics, err := cb.getOrCreateMetrics(ctx, poaID)
	if err != nil {
		return false, "", fmt.Errorf("failed to get metrics: %w", err)
	}

	metrics.mu.Lock()
	defer metrics.mu.Unlock()

	switch metrics.State {
	case CircuitBreakerClosed:
		return true, "Circuit CLOSED (normal operation)", nil

	case CircuitBreakerOpen:
		remainingTime := cb.suspensionDuration - time.Since(metrics.SuspendedAt)
		if remainingTime > 0 {
			return false, fmt.Sprintf("Circuit OPEN (suspended for %s, reason: %s)",
				remainingTime.Round(time.Second), metrics.SuspensionReason), nil
		}
		// Suspension expired - move to HALF_OPEN
		metrics.State = CircuitBreakerHalfOpen
		metrics.RecoveryAttemptedAt = time.Now()
		metrics.TestTxAllowed = cb.recoveryTestCount
		if err := cb.storeMetrics(ctx, metrics); err != nil {
			cb.logger.Errorf("Failed to store metrics (non-fatal): %v", err)
		}
		return true, "Circuit HALF_OPEN (testing recovery)", nil

	case CircuitBreakerHalfOpen:
		return true, fmt.Sprintf("Circuit HALF_OPEN (test tx %d/%d remaining)",
			metrics.TestTxAllowed, cb.recoveryTestCount), nil

	default:
		return false, fmt.Sprintf("Unknown circuit state: %s", metrics.State), nil
	}
}

// GetMetrics retrieves current metrics for a PoA
func (cb *CircuitBreaker) GetMetrics(ctx context.Context, poaID string) (*CircuitBreakerMetrics, error) {
	metrics, err := cb.getOrCreateMetrics(ctx, poaID)
	if err != nil {
		return nil, err
	}

	// Return a copy to prevent race conditions (but don't copy the mutex)
	metrics.mu.Lock()
	metricsCopy := CircuitBreakerMetrics{
		PoAID:               metrics.PoAID,
		State:               metrics.State,
		TxCountLastMinute:   metrics.TxCountLastMinute,
		TxCountLastHour:     metrics.TxCountLastHour,
		ValueLastMinute:     metrics.ValueLastMinute,
		ValueLastHour:       metrics.ValueLastHour,
		FailedTxCount:       metrics.FailedTxCount,
		TotalTxCount:        metrics.TotalTxCount,
		LastTxTimestamp:     metrics.LastTxTimestamp,
		SuspendedAt:         metrics.SuspendedAt,
		SuspensionReason:    metrics.SuspensionReason,
		RecoveryAttemptedAt: metrics.RecoveryAttemptedAt,
		TestTxAllowed:       metrics.TestTxAllowed,
	}
	metrics.mu.Unlock()

	return &metricsCopy, nil
}

// ResetMetrics resets all metrics for a PoA (admin operation)
func (cb *CircuitBreaker) ResetMetrics(ctx context.Context, poaID string) error {
	cb.logger.Infof("Resetting metrics for PoA %s", poaID)

	metrics := &CircuitBreakerMetrics{
		PoAID:             poaID,
		State:             CircuitBreakerClosed,
		TxCountLastMinute: 0,
		TxCountLastHour:   0,
		ValueLastMinute:   0,
		ValueLastHour:     0,
		FailedTxCount:     0,
		TotalTxCount:      0,
		LastTxTimestamp:   time.Time{},
		TestTxAllowed:     0,
	}

	if err := cb.storeMetrics(ctx, metrics); err != nil {
		return fmt.Errorf("failed to reset metrics: %w", err)
	}

	cb.logger.Infof("✅ Metrics reset for PoA %s", poaID)
	return nil
}

// ManualSuspend manually suspends a PoA (admin operation)
func (cb *CircuitBreaker) ManualSuspend(ctx context.Context, poaID string, reason SuspensionReason) error {
	cb.logger.Infof("Manually suspending PoA %s (reason: %s)", poaID, reason)

	metrics, err := cb.getOrCreateMetrics(ctx, poaID)
	if err != nil {
		return fmt.Errorf("failed to get metrics: %w", err)
	}

	metrics.mu.Lock()
	cb.openCircuit(ctx, poaID, metrics, reason)
	metrics.mu.Unlock()

	cb.logger.Infof("✅ PoA %s manually suspended", poaID)
	return nil
}

// ManualResume manually resumes a suspended PoA (admin operation)
func (cb *CircuitBreaker) ManualResume(ctx context.Context, poaID string) error {
	cb.logger.Infof("Manually resuming PoA %s", poaID)

	metrics, err := cb.getOrCreateMetrics(ctx, poaID)
	if err != nil {
		return fmt.Errorf("failed to get metrics: %w", err)
	}

	metrics.mu.Lock()
	defer metrics.mu.Unlock()

	if metrics.State != CircuitBreakerOpen {
		return fmt.Errorf("PoA %s not suspended (current state: %s)", poaID, metrics.State)
	}

	metrics.State = CircuitBreakerClosed
	metrics.SuspensionReason = ""
	metrics.SuspendedAt = time.Time{}

	if err := cb.storeMetrics(ctx, metrics); err != nil {
		return fmt.Errorf("failed to resume: %w", err)
	}

	cb.logger.Infof("✅ PoA %s manually resumed", poaID)
	return nil
}

// getOrCreateMetrics retrieves or creates metrics for a PoA
func (cb *CircuitBreaker) getOrCreateMetrics(ctx context.Context, poaID string) (*CircuitBreakerMetrics, error) {
	// Check local cache first
	if cached, ok := cb.metrics.Load(poaID); ok {
		return cached.(*CircuitBreakerMetrics), nil
	}

	// Check Redis
	key := fmt.Sprintf("circuit_breaker:%s", poaID)
	data, err := cb.redis.Get(ctx, key).Result()
	if err == redis.Nil {
		// Not found - create new
		metrics := &CircuitBreakerMetrics{
			PoAID:             poaID,
			State:             CircuitBreakerClosed,
			TxCountLastMinute: 0,
			TxCountLastHour:   0,
			ValueLastMinute:   0,
			ValueLastHour:     0,
			FailedTxCount:     0,
			TotalTxCount:      0,
			TestTxAllowed:     0,
		}
		cb.metrics.Store(poaID, metrics)
		return metrics, nil
	}
	if err != nil {
		return nil, fmt.Errorf("redis get failed: %w", err)
	}

	var metrics CircuitBreakerMetrics
	if err := json.Unmarshal([]byte(data), &metrics); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metrics: %w", err)
	}

	// Update local cache
	cb.metrics.Store(poaID, &metrics)

	return &metrics, nil
}

// storeMetrics persists metrics to Redis
func (cb *CircuitBreaker) storeMetrics(ctx context.Context, metrics *CircuitBreakerMetrics) error {
	key := fmt.Sprintf("circuit_breaker:%s", metrics.PoAID)

	data, err := json.Marshal(metrics)
	if err != nil {
		return fmt.Errorf("failed to marshal metrics: %w", err)
	}

	// Store with TTL (1 hour)
	if err := cb.redis.Set(ctx, key, data, 1*time.Hour).Err(); err != nil {
		return fmt.Errorf("redis set failed: %w", err)
	}

	// Update local cache
	cb.metrics.Store(metrics.PoAID, metrics)

	return nil
}

// SetSuspensionDuration configures how long circuits stay open
func (cb *CircuitBreaker) SetSuspensionDuration(duration time.Duration) {
	cb.suspensionDuration = duration
	cb.logger.Infof("Suspension duration set to %v", duration)
}

// GetSuspensionDuration returns the current suspension duration
func (cb *CircuitBreaker) GetSuspensionDuration() time.Duration {
	return cb.suspensionDuration
}

// SetRecoveryTestCount configures how many test transactions in HALF_OPEN
func (cb *CircuitBreaker) SetRecoveryTestCount(count int) {
	cb.recoveryTestCount = count
	cb.logger.Infof("Recovery test count set to %d", count)
}

// GetRecoveryTestCount returns the current recovery test count
func (cb *CircuitBreaker) GetRecoveryTestCount() int {
	return cb.recoveryTestCount
}

// GetConfig returns the current rate limit configuration
func (cb *CircuitBreaker) GetConfig() *RateLimitConfig {
	return cb.config
}

// UpdateConfig updates the rate limit configuration
func (cb *CircuitBreaker) UpdateConfig(config *RateLimitConfig) {
	cb.config = config
	cb.logger.Infof("Rate limit configuration updated")
}

// Close gracefully shuts down the circuit breaker
func (cb *CircuitBreaker) Close() error {
	if err := cb.redis.Close(); err != nil {
		return fmt.Errorf("failed to close Redis connection: %w", err)
	}

	cb.logger.Info("Circuit Breaker system shut down successfully")
	return nil
}

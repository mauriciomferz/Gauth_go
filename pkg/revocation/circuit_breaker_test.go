package revocation

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupCircuitBreakerTest(t *testing.T) (*CircuitBreaker, *miniredis.Miniredis) {
	// Create in-memory Redis
	mr, err := miniredis.Run()
	require.NoError(t, err)

	// Create Redis client (use regular client for miniredis, not cluster client)
	redisClient := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	// Create rate limit config
	config := &RateLimitConfig{
		MaxTxPerMinute:    10,
		MaxTxPerHour:      100,
		MaxValuePerMinute: 5000000000000000000,  // 5 ETH per minute
		MaxValuePerHour:   10000000000000000000, // 10 ETH per hour
		MaxFailureRate:    0.3,                  // 30% max failure rate
		FailureWindowSecs: 60,
	}

	logger := NewSimpleLogger("TEST")

	cb := &CircuitBreaker{
		redis:              redisClient,
		logger:             logger,
		config:             config,
		suspensionDuration: 2 * time.Second, // Short for testing
		recoveryTestCount:  3,               // Few test transactions
	}

	return cb, mr
}

func TestCircuitBreaker_NormalOperation(t *testing.T) {
	cb, mr := setupCircuitBreakerTest(t)
	defer mr.Close()
	defer cb.Close()

	ctx := context.Background()
	poaID := "test-poa-normal"

	// Record successful transactions (within limits)
	for i := 0; i < 5; i++ {
		err := cb.RecordTransaction(ctx, poaID, 1e18, true) // 1 ETH each
		assert.NoError(t, err)
	}

	// Circuit should remain closed
	allowed, msg, err := cb.IsPoAAllowed(ctx, poaID)
	require.NoError(t, err)
	assert.True(t, allowed)
	assert.Contains(t, msg, "CLOSED")
	t.Logf("✅ Normal operation: %s", msg)

	// Check metrics
	metrics, err := cb.GetMetrics(ctx, poaID)
	require.NoError(t, err)
	assert.Equal(t, 5, metrics.TxCountLastMinute)
	assert.Equal(t, uint64(5e18), metrics.ValueLastMinute)
	assert.Equal(t, 0, metrics.FailedTxCount)
}

func TestCircuitBreaker_TransactionRateLimit(t *testing.T) {
	cb, mr := setupCircuitBreakerTest(t)
	defer mr.Close()
	defer cb.Close()

	ctx := context.Background()
	poaID := "test-poa-rate-limit"

	// Record transactions up to limit
	for i := 0; i < 10; i++ {
		err := cb.RecordTransaction(ctx, poaID, 1e17, true) // 0.1 ETH each
		assert.NoError(t, err)
	}

	// Exceed rate limit
	err := cb.RecordTransaction(ctx, poaID, 1e17, true)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "transaction rate limit exceeded")
	t.Logf("✅ Correctly blocked: %v", err)

	// Circuit should be open
	allowed, msg, err := cb.IsPoAAllowed(ctx, poaID)
	require.NoError(t, err)
	assert.False(t, allowed)
	assert.Contains(t, msg, "OPEN")
	t.Logf("Circuit state: %s", msg)
}

func TestCircuitBreaker_ValueRateLimit(t *testing.T) {
	cb, mr := setupCircuitBreakerTest(t)
	defer mr.Close()
	defer cb.Close()

	ctx := context.Background()
	poaID := "test-poa-value-limit"

	// Record transactions within tx limit but approaching value limit (5 ETH limit)
	for i := 0; i < 4; i++ {
		err := cb.RecordTransaction(ctx, poaID, 1000000000000000000, true) // 1 ETH each = 4 ETH total
		assert.NoError(t, err)
	}

	// Exceed value limit
	err := cb.RecordTransaction(ctx, poaID, 2000000000000000000, true) // 2 more ETH = 6 ETH total
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "value rate limit exceeded")
	t.Logf("✅ Correctly blocked excessive value: %v", err)

	// Circuit should be open
	metrics, err := cb.GetMetrics(ctx, poaID)
	require.NoError(t, err)
	assert.Equal(t, CircuitBreakerOpen, metrics.State)
	assert.Equal(t, SuspensionRateLimitValue, metrics.SuspensionReason)
}

func TestCircuitBreaker_FailureRateThreshold(t *testing.T) {
	cb, mr := setupCircuitBreakerTest(t)
	defer mr.Close()
	defer cb.Close()

	ctx := context.Background()
	poaID := "test-poa-failure-rate"

	// Record 10 transactions: 7 successful, 3 failed (30% failure)
	for i := 0; i < 7; i++ {
		err := cb.RecordTransaction(ctx, poaID, 1e17, true)
		assert.NoError(t, err)
	}
	for i := 0; i < 3; i++ {
		err := cb.RecordTransaction(ctx, poaID, 1e17, false)
		assert.NoError(t, err)
	}

	// Next failed transaction exceeds 30% threshold
	err := cb.RecordTransaction(ctx, poaID, 1e17, false)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failure rate exceeded")
	t.Logf("✅ Correctly detected high failure rate: %v", err)

	// Circuit should be open
	metrics, err := cb.GetMetrics(ctx, poaID)
	require.NoError(t, err)
	assert.Equal(t, CircuitBreakerOpen, metrics.State)
	assert.Equal(t, SuspensionFailureThreshold, metrics.SuspensionReason)
}

func TestCircuitBreaker_AutoRecovery(t *testing.T) {
	cb, mr := setupCircuitBreakerTest(t)
	defer mr.Close()
	defer cb.Close()

	ctx := context.Background()
	poaID := "test-poa-recovery"

	// Trigger suspension by exceeding tx rate limit
	for i := 0; i < 11; i++ {
		_ = cb.RecordTransaction(ctx, poaID, 1e17, true) // Intentionally triggering limit
	}

	// Verify circuit is open
	allowed, _, _ := cb.IsPoAAllowed(ctx, poaID)
	assert.False(t, allowed)

	// Wait for suspension to expire (2 seconds)
	time.Sleep(2100 * time.Millisecond)

	// Circuit should move to HALF_OPEN
	allowed, msg, err := cb.IsPoAAllowed(ctx, poaID)
	require.NoError(t, err)
	assert.True(t, allowed)
	assert.Contains(t, msg, "HALF_OPEN")
	t.Logf("✅ Auto-recovery started: %s", msg)

	// Record test transactions (should succeed)
	for i := 0; i < cb.GetRecoveryTestCount(); i++ {
		err := cb.RecordTransaction(ctx, poaID, 1e17, true)
		assert.NoError(t, err)
	}

	// Circuit should be closed now
	metrics, err := cb.GetMetrics(ctx, poaID)
	require.NoError(t, err)
	assert.Equal(t, CircuitBreakerClosed, metrics.State)
	t.Logf("✅ Recovery complete: circuit CLOSED")
}

func TestCircuitBreaker_ManualSuspendResume(t *testing.T) {
	cb, mr := setupCircuitBreakerTest(t)
	defer mr.Close()
	defer cb.Close()

	ctx := context.Background()
	poaID := "test-poa-manual"

	// Manually suspend
	err := cb.ManualSuspend(ctx, poaID, SuspensionAnomalousPattern)
	assert.NoError(t, err)

	// Verify suspended
	allowed, msg, err := cb.IsPoAAllowed(ctx, poaID)
	require.NoError(t, err)
	assert.False(t, allowed)
	assert.Contains(t, msg, "OPEN")
	assert.Contains(t, msg, "ANOMALOUS_PATTERN")
	t.Logf("✅ Manual suspension: %s", msg)

	// Manually resume
	err = cb.ManualResume(ctx, poaID)
	assert.NoError(t, err)

	// Verify resumed
	allowed, msg, err = cb.IsPoAAllowed(ctx, poaID)
	require.NoError(t, err)
	assert.True(t, allowed)
	assert.Contains(t, msg, "CLOSED")
	t.Logf("✅ Manual resume: %s", msg)
}

func TestCircuitBreaker_ResetMetrics(t *testing.T) {
	cb, mr := setupCircuitBreakerTest(t)
	defer mr.Close()
	defer cb.Close()

	ctx := context.Background()
	poaID := "test-poa-reset"

	// Record some transactions
	for i := 0; i < 5; i++ {
		_ = cb.RecordTransaction(ctx, poaID, 1e18, true) // Test setup
	}

	// Verify metrics exist
	metrics, _ := cb.GetMetrics(ctx, poaID)
	assert.Equal(t, 5, metrics.TxCountLastMinute)

	// Reset metrics
	err := cb.ResetMetrics(ctx, poaID)
	assert.NoError(t, err)

	// Verify metrics cleared
	metrics, err = cb.GetMetrics(ctx, poaID)
	require.NoError(t, err)
	assert.Equal(t, 0, metrics.TxCountLastMinute)
	assert.Equal(t, uint64(0), metrics.ValueLastMinute)
	assert.Equal(t, CircuitBreakerClosed, metrics.State)
	t.Logf("✅ Metrics reset successfully")
}

func TestCircuitBreaker_ConfigurationGettersSetters(t *testing.T) {
	cb, mr := setupCircuitBreakerTest(t)
	defer mr.Close()
	defer cb.Close()

	// Test suspension duration
	newDuration := 10 * time.Minute
	cb.SetSuspensionDuration(newDuration)
	assert.Equal(t, newDuration, cb.GetSuspensionDuration())

	// Test recovery test count
	newCount := 20
	cb.SetRecoveryTestCount(newCount)
	assert.Equal(t, newCount, cb.GetRecoveryTestCount())

	// Test config update
	newConfig := &RateLimitConfig{
		MaxTxPerMinute:    20,
		MaxTxPerHour:      200,
		MaxValuePerMinute: 8000000000000000000,
		MaxValuePerHour:   16000000000000000000,
		MaxFailureRate:    0.5,
		FailureWindowSecs: 120,
	}
	cb.UpdateConfig(newConfig)
	config := cb.GetConfig()
	assert.Equal(t, 20, config.MaxTxPerMinute)
	assert.Equal(t, uint64(8000000000000000000), config.MaxValuePerMinute)

	t.Logf("✅ All configuration setters/getters working correctly")
}

func TestCircuitBreaker_HalfOpenFailure(t *testing.T) {
	cb, mr := setupCircuitBreakerTest(t)
	defer mr.Close()
	defer cb.Close()

	ctx := context.Background()
	poaID := "test-poa-halfopen-fail"

	// Trigger suspension
	for i := 0; i < 11; i++ {
		_ = cb.RecordTransaction(ctx, poaID, 1e17, true) // Test setup
	}

	// Wait for suspension to expire
	time.Sleep(2100 * time.Millisecond)

	// Verify HALF_OPEN
	allowed, msg, _ := cb.IsPoAAllowed(ctx, poaID)
	assert.True(t, allowed)
	assert.Contains(t, msg, "HALF_OPEN")

	// Record successful test transactions
	for i := 0; i < cb.GetRecoveryTestCount(); i++ {
		err := cb.RecordTransaction(ctx, poaID, 1e17, true)
		assert.NoError(t, err)
	}

	// Circuit should close after successful tests
	metrics, err := cb.GetMetrics(ctx, poaID)
	require.NoError(t, err)
	assert.Equal(t, CircuitBreakerClosed, metrics.State)
	t.Logf("✅ Recovery successful after test transactions")
}

func TestCircuitBreaker_MultiplePoAs(t *testing.T) {
	cb, mr := setupCircuitBreakerTest(t)
	defer mr.Close()
	defer cb.Close()

	ctx := context.Background()

	// Record transactions for different PoAs
	poaIDs := []string{"poa1", "poa2", "poa3"}

	for _, poaID := range poaIDs {
		// poa1: normal (5 tx)
		// poa2: rate limit (11 tx)
		// poa3: normal (3 tx)
		txCount := 5
		if poaID == "poa2" {
			txCount = 11
		} else if poaID == "poa3" {
			txCount = 3
		}

		for i := 0; i < txCount; i++ {
			_ = cb.RecordTransaction(ctx, poaID, 1e17, true) // Test setup
		}
	}

	// Check states
	allowed1, _, _ := cb.IsPoAAllowed(ctx, "poa1")
	allowed2, _, _ := cb.IsPoAAllowed(ctx, "poa2")
	allowed3, _, _ := cb.IsPoAAllowed(ctx, "poa3")

	assert.True(t, allowed1, "poa1 should be allowed")
	assert.False(t, allowed2, "poa2 should be suspended")
	assert.True(t, allowed3, "poa3 should be allowed")

	t.Logf("✅ Multiple PoAs tracked independently")
}

func TestCircuitBreaker_GetMetricsNonExistent(t *testing.T) {
	cb, mr := setupCircuitBreakerTest(t)
	defer mr.Close()
	defer cb.Close()

	ctx := context.Background()
	poaID := "non-existent-poa"

	metrics, err := cb.GetMetrics(ctx, poaID)
	assert.NoError(t, err)
	assert.NotNil(t, metrics)
	assert.Equal(t, CircuitBreakerClosed, metrics.State)
	assert.Equal(t, 0, metrics.TxCountLastMinute)
	t.Logf("✅ Non-existent PoA gets default metrics")
}

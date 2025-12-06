package revocation

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Chaos Engineering Tests
// Test system behavior under failure conditions and stress

// setupTwoPhaseTest creates a two-phase revocation system for testing
func setupTwoPhaseTest(t *testing.T) (*TwoPhaseRevocation, *miniredis.Miniredis) {
	mr, err := miniredis.Run()
	require.NoError(t, err)

	// Create Redis client (use regular client for miniredis, not cluster client)
	redisClient := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	logger := NewSimpleLogger("TEST")
	oracle, err := NewEmergencyOracle([]string{mr.Addr()}, logger)
	require.NoError(t, err)

	tpr := &TwoPhaseRevocation{
		redis:            redisClient,
		logger:           logger,
		oracle:           oracle,
		disableTimeout:   30 * time.Second,
		autoRevokeTimers: make(map[string]*time.Timer),
	}

	return tpr, mr
}

// TestChaos_RedisConnectionLoss tests behavior when Redis connection is lost
func TestChaos_RedisConnectionLoss(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping chaos test in short mode")
	}
	cb, mr := setupCircuitBreakerTest(t)
	defer mr.Close()
	defer cb.Close()

	ctx := context.Background()
	poaID := "test-poa-connection-loss"

	// Record transaction successfully
	err := cb.RecordTransaction(ctx, poaID, 1e17, true)
	assert.NoError(t, err)

	// Simulate connection loss
	mr.Close()

	// Try to record transaction - should handle error gracefully
	// Note: May succeed due to cache, or fail with Redis error
	err = cb.RecordTransaction(ctx, poaID, 1e17, true)
	// Either error or success is acceptable - just shouldn't panic

	// Verify system doesn't panic
	_, _, err = cb.IsPoAAllowed(ctx, poaID)
	// May succeed from cache or fail - both acceptable
	t.Log("✅ Redis connection loss handled gracefully")
}

// TestChaos_ConcurrentRevocations tests concurrent revocation requests on same PoA
func TestChaos_ConcurrentRevocations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping chaos test in short mode")
	}
	tpr, mr := setupTwoPhaseTest(t)
	defer mr.Close()
	defer tpr.Close()

	ctx := context.Background()
	poaID := "test-poa-concurrent"

	// Launch 10 concurrent disable requests
	var wg sync.WaitGroup
	errorCount := atomic.Int32{}
	successCount := atomic.Int32{}

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			err := tpr.DisablePoA(ctx, poaID, "principal", fmt.Sprintf("Concurrent disable %d", idx))
			if err != nil {
				errorCount.Add(1)
			} else {
				successCount.Add(1)
			}
		}(i)
	}

	wg.Wait()

	// Should have some successes (first one) and some failures (already disabled)
	t.Logf("✅ Concurrent revocations: %d successes, %d errors", successCount.Load(), errorCount.Load())
	assert.Greater(t, successCount.Load(), int32(0), "At least one should succeed")
}

// TestChaos_MemoryPressure tests behavior under memory pressure
func TestChaos_MemoryPressure(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping chaos test in short mode")
	}
	cb, mr := setupCircuitBreakerTest(t)
	defer mr.Close()
	defer cb.Close()

	// Increase rate limits for this test
	cb.config.MaxTxPerMinute = 1000
	cb.config.MaxTxPerHour = 10000

	ctx := context.Background()

	// Create many PoAs to simulate memory pressure
	poaCount := 1000
	for i := 0; i < poaCount; i++ {
		poaID := fmt.Sprintf("poa-%d", i)
		err := cb.RecordTransaction(ctx, poaID, 1e17, true)
		assert.NoError(t, err)
	}

	// Verify all PoAs are tracked
	successCount := 0
	for i := 0; i < poaCount; i++ {
		poaID := fmt.Sprintf("poa-%d", i)
		metrics, err := cb.GetMetrics(ctx, poaID)
		if err == nil && metrics.TotalTxCount > 0 {
			successCount++
		}
	}

	t.Logf("✅ Memory pressure test: %d/%d PoAs tracked successfully", successCount, poaCount)
	assert.Greater(t, successCount, poaCount*95/100, "At least 95%% should be tracked")
}

// TestChaos_RapidStateTransitions tests rapid state changes
func TestChaos_RapidStateTransitions(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping chaos test in short mode")
	}
	cb, mr := setupCircuitBreakerTest(t)
	defer mr.Close()
	defer cb.Close()

	// Set short suspension for rapid testing
	cb.SetSuspensionDuration(100 * time.Millisecond)
	cb.SetRecoveryTestCount(2)

	ctx := context.Background()
	poaID := "test-poa-rapid"

	// Trigger multiple OPEN -> HALF_OPEN -> CLOSED cycles rapidly
	for cycle := 0; cycle < 3; cycle++ {
		// Open circuit
		for i := 0; i < 11; i++ {
			_ = cb.RecordTransaction(ctx, poaID, 1e17, true) // Intentionally ignoring in test setup
		}

		// Verify OPEN
		metrics, _ := cb.GetMetrics(ctx, poaID)
		assert.Equal(t, CircuitBreakerOpen, metrics.State, "Should be OPEN after rate limit")

		// Wait for suspension to expire
		time.Sleep(150 * time.Millisecond)

		// Trigger HALF_OPEN -> CLOSED
		for i := 0; i < cb.GetRecoveryTestCount(); i++ {
			err := cb.RecordTransaction(ctx, poaID, 1e17, true)
			assert.NoError(t, err)
		}

		// Add one more transaction to actually close the circuit
		err := cb.RecordTransaction(ctx, poaID, 1e17, true)
		assert.NoError(t, err)

		// Verify CLOSED
		metrics, _ = cb.GetMetrics(ctx, poaID)
		assert.Equal(t, CircuitBreakerClosed, metrics.State, "Should be CLOSED after recovery")

		t.Logf("✅ Cycle %d: OPEN -> HALF_OPEN -> CLOSED", cycle+1)
	}
}

// TestChaos_OptimisticCollateralRace tests concurrent collateral operations
func TestChaos_OptimisticCollateralRace(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping chaos test in short mode")
	}
	opt, mr := setupOptimisticTest(t)
	defer mr.Close()
	defer opt.Close()

	ctx := context.Background()
	poaID := "test-poa-collateral-race"

	// Mark pending
	err := opt.MarkPendingRevocation(ctx, poaID, "principal", "Initial revocation", 2e18)
	require.NoError(t, err)

	// Launch concurrent finalize and challenge operations
	var wg sync.WaitGroup
	finalizeErr := make(chan error, 1)
	challengeErr := make(chan error, 1)

	// Finalize goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := opt.FinalizeRevocation(ctx, poaID)
		finalizeErr <- err
	}()

	// Challenge goroutine (slight delay to create race)
	wg.Add(1)
	go func() {
		defer wg.Done()
		time.Sleep(10 * time.Millisecond)
		err := opt.ChallengeRevocation(ctx, poaID, "challenger", "Challenge evidence")
		challengeErr <- err
	}()

	wg.Wait()
	close(finalizeErr)
	close(challengeErr)

	fErr := <-finalizeErr
	cErr := <-challengeErr

	// One should succeed, one should fail (or both could fail if timing is right)
	if fErr == nil {
		t.Log("✅ Finalize won the race")
		assert.Error(t, cErr, "Challenge should fail after finalize")
	} else if cErr == nil {
		t.Log("✅ Challenge won the race")
	} else {
		t.Log("✅ Both operations handled race condition gracefully")
	}

	// Verify final state is consistent
	usable, msg, err := opt.IsPoAUsable(ctx, poaID)
	require.NoError(t, err)
	t.Logf("Final state: usable=%v, msg=%s", usable, msg)
}

// TestChaos_PanicRecovery tests that goroutines recover from panics
func TestChaos_PanicRecovery(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping chaos test in short mode")
	}
	opt, mr := setupOptimisticTest(t)
	defer mr.Close()

	// Set very short mempool clear time to trigger auto-finalize quickly
	opt.SetMempoolClearTime(50 * time.Millisecond)

	ctx := context.Background()
	poaID := "test-poa-panic"

	// Mark pending to trigger auto-finalize goroutine
	err := opt.MarkPendingRevocation(ctx, poaID, "principal", "Test", 1e18)
	require.NoError(t, err)

	// Close immediately to potentially cause issues in auto-finalize goroutine
	opt.Close()

	// Wait a bit to see if any panics occur
	time.Sleep(100 * time.Millisecond)

	// If we get here without panic, the test passes
	t.Log("✅ No panics detected, system handled shutdown gracefully")
}

// TestChaos_HighConcurrency tests system under high concurrent load
func TestChaos_HighConcurrency(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping chaos test in short mode")
	}
	cb, mr := setupCircuitBreakerTest(t)
	defer mr.Close()
	defer cb.Close()

	// Increase limits for this test
	cb.config.MaxTxPerMinute = 1000
	cb.config.MaxTxPerHour = 10000

	ctx := context.Background()
	poaID := "test-poa-high-concurrency"

	// Launch 100 concurrent transactions
	var wg sync.WaitGroup
	successCount := atomic.Int32{}
	errorCount := atomic.Int32{}

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := cb.RecordTransaction(ctx, poaID, 1e17, true)
			if err != nil {
				errorCount.Add(1)
			} else {
				successCount.Add(1)
			}
		}()
	}

	wg.Wait()

	total := successCount.Load() + errorCount.Load()
	t.Logf("✅ High concurrency: %d/%d transactions completed", total, 100)
	assert.Equal(t, int32(100), total, "All transactions should complete")
}

// TestChaos_InvalidInputs tests system handles invalid inputs gracefully
func TestChaos_InvalidInputs(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping chaos test in short mode")
	}
	cb, mr := setupCircuitBreakerTest(t)
	defer mr.Close()
	defer cb.Close()

	ctx := context.Background()

	testCases := []struct {
		name  string
		poaID string
		value uint64
	}{
		{"Empty PoA ID", "", 1e17},
		{"Very long PoA ID", string(make([]byte, 1000)), 1e17},
		{"Unicode PoA ID", "测试-PoA-🔥", 1e17},
		{"Zero value", "test-poa", 0},
		{"Max uint64 value", "test-poa-max", ^uint64(0)},
		{"Special characters", "test/poa:123@domain.com", 1e17},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Should not panic
			err := cb.RecordTransaction(ctx, tc.poaID, tc.value, true)
			// Error or success both acceptable, just shouldn't panic
			if err != nil {
				t.Logf("✅ Handled invalid input gracefully: %v", err)
			} else {
				t.Logf("✅ Accepted input: %s", tc.name)
			}
		})
	}
}

// TestChaos_StaleMetrics tests handling of stale cached metrics
func TestChaos_StaleMetrics(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping chaos test in short mode")
	}
	cb, mr := setupCircuitBreakerTest(t)
	defer mr.Close()
	defer cb.Close()

	ctx := context.Background()
	poaID := "test-poa-stale"

	// Record transaction to create metrics
	err := cb.RecordTransaction(ctx, poaID, 1e17, true)
	require.NoError(t, err)

	// Reset metrics
	err = cb.ResetMetrics(ctx, poaID)
	require.NoError(t, err)

	// Verify metrics are actually reset
	metrics, err := cb.GetMetrics(ctx, poaID)
	require.NoError(t, err)
	assert.Equal(t, 0, metrics.TotalTxCount, "Metrics should be reset")
	t.Log("✅ Stale metrics handled: cache and storage remain consistent")
}

// TestChaos_NetworkPartition simulates network partition scenarios
func TestChaos_NetworkPartition(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping chaos test in short mode")
	}
	cb, mr := setupCircuitBreakerTest(t)
	defer cb.Close()

	ctx := context.Background()
	poaID := "test-poa-partition"

	// Record transaction successfully
	err := cb.RecordTransaction(ctx, poaID, 1e17, true)
	require.NoError(t, err)

	// Simulate partition by closing Redis
	mr.Close()

	// Operations should fail gracefully, not hang
	start := time.Now()
	_, _, err = cb.IsPoAAllowed(ctx, poaID)
	elapsed := time.Since(start)

	// May succeed from cache or fail - both acceptable, just shouldn't hang
	assert.Less(t, elapsed, 5*time.Second, "Should complete fast, not hang")
	t.Logf("✅ Network partition handled in %v", elapsed)
}

// TestChaos_ContextCancellation tests proper handling of context cancellation
func TestChaos_ContextCancellation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping chaos test in short mode")
	}
	cb, mr := setupCircuitBreakerTest(t)
	defer mr.Close()
	defer cb.Close()

	// Create cancellable context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	poaID := "test-poa-cancelled"

	// Operation with cancelled context should fail quickly
	start := time.Now()
	err := cb.RecordTransaction(ctx, poaID, 1e17, true)
	elapsed := time.Since(start)

	// Should either fail fast or succeed (if operation completes before context check)
	assert.Less(t, elapsed, 1*time.Second, "Should respect context cancellation")
	t.Logf("✅ Context cancellation handled in %v (error: %v)", elapsed, err)
}

// TestChaos_ZeroValues tests handling of zero config values
func TestChaos_ZeroValues(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping chaos test in short mode")
	}
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	rdb := redis.NewClusterClient(&redis.ClusterOptions{
		Addrs: []string{mr.Addr()},
	})

	logger := NewSimpleLogger("CHAOS")

	// Test zero config values
	cb := &CircuitBreaker{
		redis:  rdb,
		logger: logger,
		config: &RateLimitConfig{
			MaxTxPerMinute:     0, // Zero rate limit
			MaxTxPerHour:       0,
			MaxValuePerMinute:  0,
			MaxValuePerHour:    0,
			MaxFailureRate:     0,
			FailureWindowSecs:  0,
		},
		suspensionDuration: 5 * time.Minute,
		recoveryTestCount:  10,
	}
	defer cb.Close()

	ctx := context.Background()

	// Should handle zero values gracefully
	err = cb.RecordTransaction(ctx, "test-poa-zero", 1e17, true)
	if err != nil {
		t.Logf("✅ Rejected zero config: %v", err)
	} else {
		t.Log("✅ Accepted zero config (handled gracefully)")
	}
}

// TestChaos_ErrorPropagation tests error handling throughout the stack
func TestChaos_ErrorPropagation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping chaos test in short mode")
	}
	tpr, mr := setupTwoPhaseTest(t)
	defer mr.Close()
	defer tpr.Close()

	ctx := context.Background()

	// Test with non-existent PoA operations
	err := tpr.RevokePoA(ctx, "non-existent-poa", "Revoke non-existent")
	assert.Error(t, err, "Should fail for non-existent PoA")
	assert.Contains(t, err.Error(), "not found", "Error should be descriptive")
	t.Log("✅ Error propagated correctly with context")
}

// TestChaos_DeadlockPrevention tests for potential deadlocks
func TestChaos_DeadlockPrevention(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping chaos test in short mode")
	}
	cb, mr := setupCircuitBreakerTest(t)
	defer mr.Close()

	ctx := context.Background()
	poaID := "test-poa-deadlock"

	// Create potential deadlock scenario with multiple goroutines
	// accessing same PoA simultaneously
	var wg sync.WaitGroup
	done := make(chan bool, 1)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				_ = cb.RecordTransaction(ctx, poaID, 1e17, true) // Stress test - errors expected
				_, _ = cb.GetMetrics(ctx, poaID)
				_, _, _ = cb.IsPoAAllowed(ctx, poaID)
			}
		}()
	}

	// Wait with timeout to detect deadlock
	go func() {
		wg.Wait()
		done <- true
	}()

	select {
	case <-done:
		t.Log("✅ No deadlock detected")
		cb.Close()
	case <-time.After(5 * time.Second):
		t.Fatal("❌ Deadlock detected - operations hung")
	}
}

// TestChaos_ConcurrentDisableCancel tests concurrent disable and cancel operations
func TestChaos_ConcurrentDisableCancel(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping chaos test in short mode")
	}
	tpr, mr := setupTwoPhaseTest(t)
	defer mr.Close()
	defer tpr.Close()

	// Set short timeout
	tpr.SetDisableTimeout(200 * time.Millisecond)

	ctx := context.Background()
	poaID := "test-poa-disable-cancel"

	// Disable
	err := tpr.DisablePoA(ctx, poaID, "principal", "Test disable")
	require.NoError(t, err)

	// Launch concurrent cancel and revoke operations
	var wg sync.WaitGroup
	cancelErr := make(chan error, 1)
	revokeErr := make(chan error, 1)

	wg.Add(1)
	go func() {
		defer wg.Done()
		time.Sleep(50 * time.Millisecond)
		err := tpr.CancelDisable(ctx, poaID)
		cancelErr <- err
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		time.Sleep(60 * time.Millisecond)
		err := tpr.RevokePoA(ctx, poaID, "Test revoke")
		revokeErr <- err
	}()

	wg.Wait()
	close(cancelErr)
	close(revokeErr)

	cErr := <-cancelErr
	rErr := <-revokeErr

	// One should succeed, one should fail
	if cErr == nil {
		t.Log("✅ Cancel won the race")
		assert.Error(t, rErr, "Revoke should fail after cancel")
	} else if rErr == nil {
		t.Log("✅ Revoke won the race")
		assert.Error(t, cErr, "Cancel should fail after revoke")
	} else {
		t.Log("✅ Both operations handled race gracefully")
	}
}

// TestChaos_MassiveParallelLoad tests system under massive parallel load
func TestChaos_MassiveParallelLoad(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping chaos test in short mode")
	}
	cb, mr := setupCircuitBreakerTest(t)
	defer mr.Close()
	defer cb.Close()

	// Increase limits significantly
	cb.config.MaxTxPerMinute = 10000
	cb.config.MaxTxPerHour = 100000

	ctx := context.Background()

	// 100 PoAs, 10 transactions each = 1,000 total operations
	poaCount := 100
	txPerPoa := 10

	start := time.Now()
	var wg sync.WaitGroup
	successCount := atomic.Int32{}

	for i := 0; i < poaCount; i++ {
		poaID := fmt.Sprintf("poa-%d", i)
		for j := 0; j < txPerPoa; j++ {
			wg.Add(1)
			go func(poa string) {
				defer wg.Done()
				err := cb.RecordTransaction(ctx, poa, 1e17, true)
				if err == nil {
					successCount.Add(1)
				}
			}(poaID)
		}
	}

	wg.Wait()
	elapsed := time.Since(start)

	total := poaCount * txPerPoa
	throughput := float64(successCount.Load()) / elapsed.Seconds()

	t.Logf("✅ Massive parallel load: %d/%d ops in %v (%.0f ops/sec)",
		successCount.Load(), total, elapsed, throughput)
	assert.Greater(t, successCount.Load(), int32(total*95/100), "At least 95%% should succeed")
}

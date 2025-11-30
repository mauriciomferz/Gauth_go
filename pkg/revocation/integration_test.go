package revocation

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testPrincipalID = "test-principal"
)

// End-to-End Integration Tests
// Tests complete workflows across all revocation systems

// IntegrationTestEnv provides a complete test environment with all systems
type IntegrationTestEnv struct {
	MiniRedis           *miniredis.Miniredis
	RedisClient         *redis.ClusterClient
	CircuitBreaker      *CircuitBreaker
	TwoPhaseRevocation  *TwoPhaseRevocation
	OptimisticRevocation *OptimisticRevocation
	Logger              Logger
}

func setupIntegrationEnv(t *testing.T) *IntegrationTestEnv {
	mr, err := miniredis.Run()
	require.NoError(t, err)

	redisClient := redis.NewClusterClient(&redis.ClusterOptions{
		Addrs: []string{mr.Addr()},
	})

	logger := NewSimpleLogger("INTEGRATION")
	
	oracle, err := NewEmergencyOracle([]string{mr.Addr()}, logger)
	require.NoError(t, err)

	cb := &CircuitBreaker{
		redis:  redisClient,
		logger: logger,
		config: &RateLimitConfig{
			MaxTxPerMinute:    10,
			MaxTxPerHour:      100,
			MaxValuePerMinute: 5000000000000000000, // 5 ETH
			MaxValuePerHour:   5000000000000000000, // 5 ETH (same as per-minute for testing)
			MaxFailureRate:    0.3,
			FailureWindowSecs: 60,
		},
		metrics:            sync.Map{},
		suspensionDuration: 1 * time.Minute,
		recoveryTestCount:  3,
	}

	tpr := &TwoPhaseRevocation{
		redis:          redisClient,
		logger:         logger,
		oracle:         oracle,
		disableTimeout: 30 * time.Second,
		states:         sync.Map{},
		autoRevokeTimers: make(map[string]*time.Timer),
	}

	opt := &OptimisticRevocation{
		redis:            redisClient,
		logger:           logger,
		oracle:           oracle,
		challengeWindow:  5 * time.Minute,
		mempoolClearTime: 10 * time.Minute, // Long enough to test before auto-finalize
		minCollateral:    1000,
		states:           sync.Map{},
		shutdown:         make(chan struct{}),
	}

	return &IntegrationTestEnv{
		MiniRedis:           mr,
		RedisClient:         redisClient,
		CircuitBreaker:      cb,
		TwoPhaseRevocation:  tpr,
		OptimisticRevocation: opt,
		Logger:              logger,
	}
}

func (env *IntegrationTestEnv) Cleanup() {
	if env.CircuitBreaker != nil {
		env.CircuitBreaker.Close()
	}
	if env.TwoPhaseRevocation != nil {
		env.TwoPhaseRevocation.Close()
	}
	if env.OptimisticRevocation != nil {
		env.OptimisticRevocation.Close()
	}
	if env.MiniRedis != nil {
		env.MiniRedis.Close()
	}
}

// TestE2E_TwoPhaseToCircuitBreaker tests integration between two-phase and circuit breaker
func TestE2E_TwoPhaseToCircuitBreaker(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	env := setupIntegrationEnv(t)
	defer env.Cleanup()

	ctx := context.Background()
	poaID := "test-poa-integrated"
	principal := testPrincipalID

	// Phase 1: Disable using two-phase
	t.Log("Step 1: Disabling PoA using two-phase revocation")
	err := env.TwoPhaseRevocation.DisablePoA(ctx, poaID, principal, "integration-test")
	require.NoError(t, err)

	// Verify PoA state is disabled
	state, err := env.TwoPhaseRevocation.GetPoAState(ctx, poaID)
	require.NoError(t, err)
	assert.Equal(t, PoAStatusDisabled, state.Status)
	t.Logf("✓ PoA state: %s", state.Status)

	// Phase 2: Circuit breaker should see this PoA as problematic
	t.Log("Step 2: Testing circuit breaker awareness")
	allowed, msg, err := env.CircuitBreaker.IsPoAAllowed(ctx, poaID)
	require.NoError(t, err)
	t.Logf("✓ Circuit breaker check: allowed=%v, msg=%s", allowed, msg)

	// Phase 3: Revoke the PoA
	t.Log("Step 3: Revoking PoA permanently")
	err = env.TwoPhaseRevocation.RevokePoA(ctx, poaID, "test-revoke")
	require.NoError(t, err)

	// Verify final state
	state, err = env.TwoPhaseRevocation.GetPoAState(ctx, poaID)
	require.NoError(t, err)
	assert.Equal(t, PoAStatusRevoked, state.Status)
	t.Logf("✓ Final PoA state: %s", state.Status)
}

// TestE2E_OptimisticWithCircuitBreaker tests optimistic revocation with circuit breaker monitoring
func TestE2E_OptimisticWithCircuitBreaker(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	env := setupIntegrationEnv(t)
	defer env.Cleanup()

	ctx := context.Background()
	poaID := "test-poa-optimistic-cb"
	principal := "test-principal"
	collateral := uint64(5000)

	// Step 1: Mark as pending using optimistic revocation
	t.Log("Step 1: Initiating optimistic revocation")
	err := env.OptimisticRevocation.MarkPendingRevocation(ctx, poaID, principal, "integration-test", collateral)
	require.NoError(t, err)

	// Verify pending state
	state, err := env.OptimisticRevocation.GetRevocationState(ctx, poaID)
	require.NoError(t, err)
	assert.Equal(t, OptimisticStatusPending, state.Status)
	t.Logf("✓ Optimistic state: %s, collateral: %d", state.Status, state.Collateral)

	// Step 2: Circuit breaker records transactions on this PoA
	t.Log("Step 2: Recording transactions in circuit breaker")
	for i := 0; i < 5; i++ {
		err = env.CircuitBreaker.RecordTransaction(ctx, poaID, 1000, true)
		assert.NoError(t, err)
	}

	metrics, err := env.CircuitBreaker.GetMetrics(ctx, poaID)
	require.NoError(t, err)
	t.Logf("✓ Circuit breaker metrics: %d transactions", metrics.TotalTxCount)

	// Step 3: Finalize revocation (no challenges)
	t.Log("Step 3: Finalizing optimistic revocation")
	err = env.OptimisticRevocation.FinalizeRevocation(ctx, poaID)
	require.NoError(t, err)

	// Verify finalized state
	state, err = env.OptimisticRevocation.GetRevocationState(ctx, poaID)
	require.NoError(t, err)
	assert.Equal(t, OptimisticStatusFinalized, state.Status)
	t.Logf("✓ Final state: %s", state.Status)
}

// TestE2E_ConcurrentPrincipalsRevokingSamePoA tests race conditions with multiple principals
func TestE2E_ConcurrentPrincipalsRevokingSamePoA(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	env := setupIntegrationEnv(t)
	defer env.Cleanup()

	ctx := context.Background()
	poaID := "test-poa-concurrent-principals"
	numPrincipals := 5

	t.Logf("Testing concurrent revocation by %d principals", numPrincipals)

	var wg sync.WaitGroup
	results := make(chan error, numPrincipals)

	// Multiple principals try to disable the same PoA simultaneously
	for i := 0; i < numPrincipals; i++ {
		wg.Add(1)
		go func(principalID int) {
			defer wg.Done()
			principal := fmt.Sprintf("principal-%d", principalID)
			err := env.TwoPhaseRevocation.DisablePoA(ctx, poaID, principal, "concurrent-test")
			results <- err
		}(i)
	}

	wg.Wait()
	close(results)

	// Collect results
	successes := 0
	failures := 0
	for err := range results {
		if err == nil {
			successes++
		} else {
			failures++
		}
	}

	t.Logf("Results: %d successes, %d failures", successes, failures)

	// At least one should succeed, others may fail with "already disabled"
	assert.Greater(t, successes, 0, "At least one principal should successfully disable")
	
	// Verify final state is consistent
	state, err := env.TwoPhaseRevocation.GetPoAState(ctx, poaID)
	require.NoError(t, err)
	assert.Equal(t, PoAStatusDisabled, state.Status)
	t.Logf("✓ Consistent final state: %s", state.Status)
}

// TestE2E_RecoveryFromPartialFailures tests system recovery after partial failures
func TestE2E_RecoveryFromPartialFailures(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	env := setupIntegrationEnv(t)
	defer env.Cleanup()

	ctx := context.Background()
	poaID := "test-poa-recovery-123"
	principal := testPrincipalID

	// Step 1: Start a two-phase disable
	t.Log("Step 1: Starting two-phase disable")
	err := env.TwoPhaseRevocation.DisablePoA(ctx, poaID, principal, "recovery-test")
	require.NoError(t, err)

	// Step 2: Simulate partial failure - close Redis temporarily
	t.Log("Step 2: Simulating Redis failure")
	env.MiniRedis.Close()
	time.Sleep(100 * time.Millisecond)

	// Step 3: Try operations during failure (should fail gracefully)
	t.Log("Step 3: Attempting operations during failure")
	// Try a new operation that requires Redis (not just cached state)
	err = env.TwoPhaseRevocation.DisablePoA(ctx, "test-poa-failure", testPrincipalID, "should-fail")
	assert.Error(t, err, "Should fail when Redis is down")
	t.Logf("✓ Expected error during outage: %v", err)

	// Step 4: Restart Redis
	t.Log("Step 4: Recovering Redis")
	mr, err := miniredis.Run()
	require.NoError(t, err)
	env.MiniRedis = mr

	// Update Redis client to new instance
	env.RedisClient = redis.NewClusterClient(&redis.ClusterOptions{
		Addrs: []string{mr.Addr()},
	})
	env.CircuitBreaker.redis = env.RedisClient
	env.TwoPhaseRevocation.redis = env.RedisClient
	env.OptimisticRevocation.redis = env.RedisClient

	t.Log("Step 5: Verifying system recovery")
	// System should be able to start fresh operations
	newPoAID := "test-poa-after-recovery"
	err = env.TwoPhaseRevocation.DisablePoA(ctx, newPoAID, principal, "post-recovery-test")
	require.NoError(t, err)

	state, err := env.TwoPhaseRevocation.GetPoAState(ctx, newPoAID)
	require.NoError(t, err)
	assert.Equal(t, PoAStatusDisabled, state.Status)
	t.Logf("✓ System recovered: new PoA %s is %s", newPoAID, state.Status)
}

// TestE2E_CrossSystemConsistency tests data consistency across all systems
func TestE2E_CrossSystemConsistency(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	env := setupIntegrationEnv(t)
	defer env.Cleanup()

	ctx := context.Background()
	poaID := "test-poa-consistency"
	principal := "test-principal"

	// Create state in all systems
	t.Log("Step 1: Creating state across all systems")

	// Two-phase disable
	err := env.TwoPhaseRevocation.DisablePoA(ctx, poaID, principal, "consistency-test")
	require.NoError(t, err)

	// Circuit breaker transactions
	for i := 0; i < 3; i++ {
		err = env.CircuitBreaker.RecordTransaction(ctx, poaID, 1000, true)
		require.NoError(t, err)
	}

	// Optimistic revocation (different PoA to avoid conflicts)
	optPoAID := "test-poa-consistency-opt"
	err = env.OptimisticRevocation.MarkPendingRevocation(ctx, optPoAID, principal, "consistency-test", 3000)
	require.NoError(t, err)

	// Step 2: Verify all states are accessible
	t.Log("Step 2: Verifying cross-system state consistency")

	tprState, err := env.TwoPhaseRevocation.GetPoAState(ctx, poaID)
	require.NoError(t, err)
	assert.Equal(t, PoAStatusDisabled, tprState.Status)
	t.Logf("✓ Two-phase state: %s", tprState.Status)

	cbMetrics, err := env.CircuitBreaker.GetMetrics(ctx, poaID)
	require.NoError(t, err)
	assert.Equal(t, 3, cbMetrics.TotalTxCount)
	t.Logf("✓ Circuit breaker metrics: %d transactions", cbMetrics.TotalTxCount)

	optState, err := env.OptimisticRevocation.GetRevocationState(ctx, optPoAID)
	require.NoError(t, err)
	assert.Equal(t, OptimisticStatusPending, optState.Status)
	t.Logf("✓ Optimistic state: %s with collateral %d", optState.Status, optState.Collateral)

	// Step 3: Verify isolation - operations on one PoA don't affect others
	t.Log("Step 3: Verifying isolation between systems")
	
	// Revoke in two-phase
	err = env.TwoPhaseRevocation.RevokePoA(ctx, poaID, "final-revoke")
	require.NoError(t, err)

	// Optimistic PoA should be unaffected
	optState, err = env.OptimisticRevocation.GetRevocationState(ctx, optPoAID)
	require.NoError(t, err)
	assert.Equal(t, OptimisticStatusPending, optState.Status)
	t.Logf("✓ Isolation verified: optimistic PoA still %s", optState.Status)
}

// TestE2E_CompleteRevocationWorkflow tests full lifecycle from creation to revocation
func TestE2E_CompleteRevocationWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	env := setupIntegrationEnv(t)
	defer env.Cleanup()

	ctx := context.Background()
	poaID := "test-poa-complete-workflow"
	principal := "test-principal"

	t.Log("=== Complete Revocation Workflow Test ===")

	// Phase 1: Normal operation with circuit breaker monitoring
	t.Log("Phase 1: Normal operation (circuit breaker monitoring)")
	for i := 0; i < 5; i++ {
		err := env.CircuitBreaker.RecordTransaction(ctx, poaID, 1000, true)
		require.NoError(t, err)
	}
	
	allowed, msg, err := env.CircuitBreaker.IsPoAAllowed(ctx, poaID)
	require.NoError(t, err)
	assert.True(t, allowed)
	t.Logf("✓ PoA operational: %s", msg)

	// Phase 2: Suspicious activity detected - disable via two-phase
	t.Log("Phase 2: Suspicious activity - initiating two-phase disable")
	err = env.TwoPhaseRevocation.DisablePoA(ctx, poaID, principal, "suspicious-activity")
	require.NoError(t, err)

	usable, reason, err := env.TwoPhaseRevocation.IsPoAUsable(ctx, poaID)
	require.NoError(t, err)
	assert.False(t, usable)
	t.Logf("✓ PoA disabled: %s", reason)

	// Phase 3: Investigation period (no cancellation)
	t.Log("Phase 3: Investigation period (5 seconds)")
	time.Sleep(5 * time.Second)

	// Phase 4: Confirmed malicious - permanent revocation
	t.Log("Phase 4: Confirmed malicious - permanent revocation")
	err = env.TwoPhaseRevocation.RevokePoA(ctx, poaID, "confirmed-malicious")
	require.NoError(t, err)

	state, err := env.TwoPhaseRevocation.GetPoAState(ctx, poaID)
	require.NoError(t, err)
	assert.Equal(t, PoAStatusRevoked, state.Status)
	t.Logf("✓ Final state: %s", state.Status)

	// Phase 5: Verify PoA cannot be used anymore
	t.Log("Phase 5: Verification - PoA permanently unusable")
	usable, reason, err = env.TwoPhaseRevocation.IsPoAUsable(ctx, poaID)
	require.NoError(t, err)
	assert.False(t, usable)
	assert.Contains(t, reason, "revoked")
	t.Logf("✓ Permanent revocation confirmed: %s", reason)

	t.Log("=== Workflow Complete ===")
}

// TestE2E_CircuitBreakerUnderLoad tests circuit breaker with realistic traffic
func TestE2E_CircuitBreakerUnderLoad(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	env := setupIntegrationEnv(t)
	defer env.Cleanup()

	ctx := context.Background()
	poaID := "test-poa-cb-load"

	t.Log("Simulating realistic traffic pattern")

	// Normal traffic
	t.Log("Phase 1: Normal traffic (all successful)")
	for i := 0; i < 8; i++ {
		err := env.CircuitBreaker.RecordTransaction(ctx, poaID, 500000000000000000, true) // 0.5 ETH
		assert.NoError(t, err)
	}

	metrics, err := env.CircuitBreaker.GetMetrics(ctx, poaID)
	require.NoError(t, err)
	assert.Equal(t, CircuitBreakerClosed, metrics.State)
	t.Logf("✓ Circuit state: CLOSED, %d transactions", metrics.TotalTxCount)

	// Trigger rate limit
	t.Log("Phase 2: Rate limit exceeded")
	err = env.CircuitBreaker.RecordTransaction(ctx, poaID, 500000000000000000, true)
	assert.NoError(t, err)
	err = env.CircuitBreaker.RecordTransaction(ctx, poaID, 500000000000000000, true)
	assert.NoError(t, err)

	// Next transaction should trigger circuit
	err = env.CircuitBreaker.RecordTransaction(ctx, poaID, 500000000000000000, true)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "rate limit exceeded")
	t.Logf("✓ Circuit opened: %v", err)

	// Verify circuit is open
	metrics, err = env.CircuitBreaker.GetMetrics(ctx, poaID)
	require.NoError(t, err)
	assert.Equal(t, CircuitBreakerOpen, metrics.State)
	t.Logf("✓ Circuit state confirmed: OPEN")

	// Check if PoA is allowed
	allowed, msg, err := env.CircuitBreaker.IsPoAAllowed(ctx, poaID)
	require.NoError(t, err)
	assert.False(t, allowed)
	t.Logf("✓ PoA blocked: %s", msg)
}

// TestE2E_OptimisticRevocationWithChallenge tests optimistic revocation with a challenge
func TestE2E_OptimisticRevocationWithChallenge(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	env := setupIntegrationEnv(t)
	defer env.Cleanup()

	ctx := context.Background()
	poaID := "test-poa-optimistic-challenge"
	principal := "test-principal"
	collateral := uint64(10000)

	t.Log("=== Optimistic Revocation with Challenge Test ===")

	// Phase 1: Initiate optimistic revocation
	t.Log("Phase 1: Initiating optimistic revocation")
	err := env.OptimisticRevocation.MarkPendingRevocation(ctx, poaID, principal, "optimistic-test", collateral)
	require.NoError(t, err)

	state, err := env.OptimisticRevocation.GetRevocationState(ctx, poaID)
	require.NoError(t, err)
	assert.Equal(t, OptimisticStatusPending, state.Status)
	t.Logf("✓ Status: %s, Collateral: %d", state.Status, state.Collateral)

	// Phase 2: Challenge the revocation
	t.Log("Phase 2: Challenging revocation")
	challenger := "challenger-principal"
	err = env.OptimisticRevocation.ChallengeRevocation(ctx, poaID, challenger, "Invalid revocation - no rule violations detected")
	require.NoError(t, err)

	state, err = env.OptimisticRevocation.GetRevocationState(ctx, poaID)
	require.NoError(t, err)
	assert.Equal(t, OptimisticStatusChallenged, state.Status)
	assert.False(t, state.ChallengedAt.IsZero(), "ChallengedAt timestamp should be set")
	t.Logf("✓ Status: %s, ChallengedAt: %v", state.Status, state.ChallengedAt)

	// Phase 3: Verify cannot finalize while challenged
	t.Log("Phase 3: Attempting to finalize (should fail)")
	err = env.OptimisticRevocation.FinalizeRevocation(ctx, poaID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "challenged")
	t.Logf("✓ Cannot finalize while challenged: %v", err)

	t.Log("=== Challenge Test Complete ===")
}

// TestE2E_DataPersistenceAcrossRestarts tests that state persists in Redis
func TestE2E_DataPersistenceAcrossRestarts(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	env := setupIntegrationEnv(t)
	defer env.Cleanup()

	ctx := context.Background()
	poaID := "test-poa-persistence"
	principal := "test-principal"

	t.Log("Phase 1: Creating state")
	
	// Create state in two-phase
	err := env.TwoPhaseRevocation.DisablePoA(ctx, poaID, principal, "persistence-test")
	require.NoError(t, err)

	// Create state in circuit breaker
	err = env.CircuitBreaker.RecordTransaction(ctx, poaID, 1000, true)
	require.NoError(t, err)

	// Read initial states
	tprState1, err := env.TwoPhaseRevocation.GetPoAState(ctx, poaID)
	require.NoError(t, err)
	t.Logf("✓ Initial two-phase state: %s", tprState1.Status)

	cbMetrics1, err := env.CircuitBreaker.GetMetrics(ctx, poaID)
	require.NoError(t, err)
	t.Logf("✓ Initial circuit breaker metrics: %d transactions", cbMetrics1.TotalTxCount)

	// Phase 2: Simulate restart - close and recreate systems (keep Redis)
	t.Log("Phase 2: Simulating system restart")
	env.CircuitBreaker.Close()
	env.TwoPhaseRevocation.Close()
	env.OptimisticRevocation.Close()

	// Create new Redis cluster client (old one was closed) but connect to same miniredis
	newRedisClient := redis.NewClusterClient(&redis.ClusterOptions{
		Addrs: []string{env.MiniRedis.Addr()},
	})

	// Recreate systems with new Redis client
	logger := NewSimpleLogger("INTEGRATION-RESTART")
	oracle, err := NewEmergencyOracle([]string{env.MiniRedis.Addr()}, logger)
	require.NoError(t, err)

	env.CircuitBreaker = &CircuitBreaker{
		redis:  newRedisClient,
		logger: logger,
		config: &RateLimitConfig{
			MaxTxPerMinute:    10,
			MaxTxPerHour:      100,
			MaxValuePerMinute: 5000000000000000000,
			MaxValuePerHour:   5000000000000000000, // 5 ETH (same as per-minute)
			MaxFailureRate:    0.3,
			FailureWindowSecs: 60,
		},
		metrics:            sync.Map{},
		suspensionDuration: 1 * time.Minute,
		recoveryTestCount:  3,
	}

	env.TwoPhaseRevocation = &TwoPhaseRevocation{
		redis:          newRedisClient,
		logger:         logger,
		oracle:         oracle,
		disableTimeout: 30 * time.Second,
		states:         sync.Map{},
		autoRevokeTimers: make(map[string]*time.Timer),
	}

	env.OptimisticRevocation = &OptimisticRevocation{
		redis:            newRedisClient,
		logger:           logger,
		oracle:           oracle,
		challengeWindow:  5 * time.Minute,
		mempoolClearTime: 10 * time.Minute,
		minCollateral:    1000,
		states:           sync.Map{},
		shutdown:         make(chan struct{}),
	}

	// Update env.RedisClient to new client for cleanup
	env.RedisClient = newRedisClient

	// Phase 3: Verify state persisted
	t.Log("Phase 3: Verifying persisted state")

	tprState2, err := env.TwoPhaseRevocation.GetPoAState(ctx, poaID)
	require.NoError(t, err)
	assert.Equal(t, tprState1.Status, tprState2.Status)
	assert.Equal(t, tprState1.Principal, tprState2.Principal)
	t.Logf("✓ Two-phase state persisted: %s", tprState2.Status)

	cbMetrics2, err := env.CircuitBreaker.GetMetrics(ctx, poaID)
	require.NoError(t, err)
	assert.Equal(t, cbMetrics1.TotalTxCount, cbMetrics2.TotalTxCount)
	t.Logf("✓ Circuit breaker metrics persisted: %d transactions", cbMetrics2.TotalTxCount)

	t.Log("✓ All state persisted across restart")
}

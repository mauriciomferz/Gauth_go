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

const (
	testPrincipalAddr = "principal-address"
)

func setupOptimisticTest(t *testing.T) (*OptimisticRevocation, *miniredis.Miniredis) {
	// Create in-memory Redis
	mr, err := miniredis.Run()
	require.NoError(t, err)

	// Create Redis cluster client (single node for testing)
	redisClient := redis.NewClusterClient(&redis.ClusterOptions{
		Addrs: []string{mr.Addr()},
	})

	// Create mock oracle
	logger := NewSimpleLogger("TEST")
	oracle, err := NewEmergencyOracle([]string{mr.Addr()}, logger)
	require.NoError(t, err)

	// Create optimistic revocation
	or := &OptimisticRevocation{
		redis:            redisClient,
		logger:           logger,
		oracle:           oracle,
		challengeWindow:  5 * time.Second, // Short window for tests
		mempoolClearTime: 200 * time.Millisecond,
		minCollateral:    1e18, // 1 ETH
		shutdown:         make(chan struct{}),
	}

	return or, mr
}

func TestOptimisticRevocation_MarkPendingRevocation(t *testing.T) {
	or, mr := setupOptimisticTest(t)
	defer mr.Close()
	defer or.Close()

	ctx := context.Background()
	poaID := "test-poa-123"
	principal := testPrincipalAddr
	reason := "Suspicious activity detected"
	collateral := uint64(2e18) // 2 ETH

	start := time.Now()
	err := or.MarkPendingRevocation(ctx, poaID, principal, reason, collateral)
	duration := time.Since(start)

	assert.NoError(t, err)
	t.Logf("✅ MarkPendingRevocation completed in %v", duration)

	// Verify state
	state, err := or.GetRevocationState(ctx, poaID)
	require.NoError(t, err)
	require.NotNil(t, state)

	assert.Equal(t, poaID, state.PoAID)
	assert.Equal(t, OptimisticStatusPending, state.Status)
	assert.Equal(t, reason, state.Reason)
	assert.Equal(t, principal, state.Principal)
	assert.Equal(t, collateral, state.Collateral)
	assert.False(t, state.PendingAt.IsZero())
	assert.False(t, state.ChallengeDeadline.IsZero())

	// Verify NEW transactions are rejected
	usable, msg, err := or.IsPoAUsable(ctx, poaID)
	require.NoError(t, err)
	assert.False(t, usable)
	assert.Contains(t, msg, "pending")
	t.Logf("IsPoAUsable: %v, %s", usable, msg)
}

func TestOptimisticRevocation_InsufficientCollateral(t *testing.T) {
	or, mr := setupOptimisticTest(t)
	defer mr.Close()
	defer or.Close()

	ctx := context.Background()
	poaID := "test-poa-456"
	principal := testPrincipalAddr
	reason := "Test"
	collateral := uint64(5e17) // 0.5 ETH (below 1 ETH minimum)

	err := or.MarkPendingRevocation(ctx, poaID, principal, reason, collateral)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "insufficient collateral")
	t.Logf("✅ Correctly rejected insufficient collateral: %v", err)
}

func TestOptimisticRevocation_FinalizeRevocation(t *testing.T) {
	or, mr := setupOptimisticTest(t)
	defer mr.Close()
	defer or.Close()

	ctx := context.Background()
	poaID := "test-poa-789"
	principal := testPrincipalAddr
	reason := "Confirmed malicious behavior"
	collateral := uint64(3e18) // 3 ETH

	// Mark as pending
	err := or.MarkPendingRevocation(ctx, poaID, principal, reason, collateral)
	require.NoError(t, err)

	// Wait a bit to simulate mempool clearing
	time.Sleep(100 * time.Millisecond)

	// Finalize
	start := time.Now()
	err = or.FinalizeRevocation(ctx, poaID)
	duration := time.Since(start)

	assert.NoError(t, err)
	t.Logf("✅ FinalizeRevocation completed in %v", duration)

	// Verify state
	state, err := or.GetRevocationState(ctx, poaID)
	require.NoError(t, err)
	require.NotNil(t, state)

	assert.Equal(t, OptimisticStatusFinalized, state.Status)
	assert.False(t, state.FinalizedAt.IsZero())
	assert.True(t, state.FinalizedAt.After(state.PendingAt))

	// Verify PoA is permanently unusable
	usable, msg, err := or.IsPoAUsable(ctx, poaID)
	require.NoError(t, err)
	assert.False(t, usable)
	assert.Contains(t, msg, "permanently revoked")
	t.Logf("IsPoAUsable: %v, %s", usable, msg)
}

func TestOptimisticRevocation_ChallengeRevocation(t *testing.T) {
	or, mr := setupOptimisticTest(t)
	defer mr.Close()
	defer or.Close()

	ctx := context.Background()
	poaID := "test-poa-challenge"
	principal := testPrincipalAddr
	reason := "Potentially malicious (disputed)"
	collateral := uint64(2e18) // 2 ETH
	challenger := "validator-address"
	evidence := "AI was not compromised, revocation was malicious"

	// Mark as pending
	err := or.MarkPendingRevocation(ctx, poaID, principal, reason, collateral)
	require.NoError(t, err)

	// Challenge the revocation
	start := time.Now()
	err = or.ChallengeRevocation(ctx, poaID, challenger, evidence)
	duration := time.Since(start)

	assert.NoError(t, err)
	t.Logf("✅ ChallengeRevocation completed in %v", duration)

	// Verify state
	state, err := or.GetRevocationState(ctx, poaID)
	require.NoError(t, err)
	require.NotNil(t, state)

	assert.Equal(t, OptimisticStatusChallenged, state.Status)
	assert.False(t, state.ChallengedAt.IsZero())

	// Verify PoA is back to usable (challenge successful)
	usable, msg, err := or.IsPoAUsable(ctx, poaID)
	require.NoError(t, err)
	assert.True(t, usable)
	assert.Contains(t, msg, "challenged")
	t.Logf("IsPoAUsable: %v, %s", usable, msg)
}

func TestOptimisticRevocation_ChallengeWindowExpired(t *testing.T) {
	or, mr := setupOptimisticTest(t)
	defer mr.Close()
	defer or.Close()

	// Set very short challenge window for testing
	or.SetChallengeWindow(100 * time.Millisecond)

	ctx := context.Background()
	poaID := "test-poa-expired"
	principal := testPrincipalAddr
	reason := "Test expiration"
	collateral := uint64(1e18)

	// Mark as pending
	err := or.MarkPendingRevocation(ctx, poaID, principal, reason, collateral)
	require.NoError(t, err)

	// Wait for challenge window to expire
	time.Sleep(150 * time.Millisecond)

	// Try to challenge (should fail)
	err = or.ChallengeRevocation(ctx, poaID, "late-challenger", "too late")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "challenge window expired")
	t.Logf("✅ Correctly rejected expired challenge: %v", err)
}

func TestOptimisticRevocation_AutoFinalize(t *testing.T) {
	or, mr := setupOptimisticTest(t)
	defer mr.Close()
	defer or.Close()

	// Set short mempool clear time for testing
	or.SetMempoolClearTime(200 * time.Millisecond)

	ctx := context.Background()
	poaID := "test-poa-auto"
	principal := testPrincipalAddr
	reason := "Auto-finalize test"
	collateral := uint64(1e18)

	// Mark as pending (triggers auto-finalize goroutine)
	err := or.MarkPendingRevocation(ctx, poaID, principal, reason, collateral)
	require.NoError(t, err)

	// Verify still pending immediately
	state, err := or.GetRevocationState(ctx, poaID)
	require.NoError(t, err)
	assert.Equal(t, OptimisticStatusPending, state.Status)

	// Wait for auto-finalize
	time.Sleep(300 * time.Millisecond)

	// Verify auto-finalized
	state, err = or.GetRevocationState(ctx, poaID)
	require.NoError(t, err)
	assert.Equal(t, OptimisticStatusFinalized, state.Status)
	t.Logf("✅ Auto-finalization successful after %v", or.GetMempoolClearTime())
}

func TestOptimisticRevocation_CannotFinalizeAfterChallenge(t *testing.T) {
	or, mr := setupOptimisticTest(t)
	defer mr.Close()
	defer or.Close()

	ctx := context.Background()
	poaID := "test-poa-challenge-first"
	principal := testPrincipalAddr
	reason := "Test challenge before finalize"
	collateral := uint64(2e18)

	// Mark as pending
	err := or.MarkPendingRevocation(ctx, poaID, principal, reason, collateral)
	require.NoError(t, err)

	// Challenge immediately
	err = or.ChallengeRevocation(ctx, poaID, "challenger", "evidence")
	require.NoError(t, err)

	// Try to finalize (should fail)
	err = or.FinalizeRevocation(ctx, poaID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "challenged")
	t.Logf("✅ Correctly prevented finalization after challenge: %v", err)
}

func TestOptimisticRevocation_ConfigurationGettersSetters(t *testing.T) {
	or, mr := setupOptimisticTest(t)
	defer mr.Close()
	defer or.Close()

	// Test challenge window
	newWindow := 10 * time.Minute
	or.SetChallengeWindow(newWindow)
	assert.Equal(t, newWindow, or.GetChallengeWindow())

	// Test mempool clear time
	newClearTime := 2 * time.Minute
	or.SetMempoolClearTime(newClearTime)
	assert.Equal(t, newClearTime, or.GetMempoolClearTime())

	// Test min collateral
	newCollateral := uint64(5e18) // 5 ETH
	or.SetMinCollateral(newCollateral)
	assert.Equal(t, newCollateral, or.GetMinCollateral())

	t.Logf("✅ All configuration setters/getters working correctly")
}

func TestOptimisticRevocation_GetStateNonExistent(t *testing.T) {
	or, mr := setupOptimisticTest(t)
	defer mr.Close()
	defer or.Close()

	ctx := context.Background()
	poaID := "non-existent-poa"

	state, err := or.GetRevocationState(ctx, poaID)
	assert.NoError(t, err)
	assert.Nil(t, state)

	// Non-existent PoA should be usable (no revocation)
	usable, msg, err := or.IsPoAUsable(ctx, poaID)
	require.NoError(t, err)
	assert.True(t, usable)
	assert.Contains(t, msg, "active")
	t.Logf("✅ Non-existent PoA correctly marked as usable")
}

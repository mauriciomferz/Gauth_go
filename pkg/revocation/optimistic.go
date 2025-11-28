package revocation

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
)

// OptimisticRevocationStatus represents the state of an optimistic revocation
type OptimisticRevocationStatus string

const (
	// OptimisticStatusPending indicates revocation is pending (mempool clearing)
	OptimisticStatusPending OptimisticRevocationStatus = "PENDING"
	
	// OptimisticStatusFinalized indicates revocation is finalized (on-chain)
	OptimisticStatusFinalized OptimisticRevocationStatus = "FINALIZED"
	
	// OptimisticStatusChallenged indicates the revocation was challenged (collateral slashed)
	OptimisticStatusChallenged OptimisticRevocationStatus = "CHALLENGED"
)

// OptimisticRevocationState tracks the state of an optimistic revocation
type OptimisticRevocationState struct {
	PoAID              string                       `json:"poa_id"`
	Status             OptimisticRevocationStatus   `json:"status"`
	PendingAt          time.Time                    `json:"pending_at,omitempty"`
	FinalizedAt        time.Time                    `json:"finalized_at,omitempty"`
	ChallengedAt       time.Time                    `json:"challenged_at,omitempty"`
	Reason             string                       `json:"reason"`
	Principal          string                       `json:"principal"`
	Collateral         uint64                       `json:"collateral"`         // Wei deposited
	ChallengeDeadline  time.Time                    `json:"challenge_deadline"` // When challenge window closes
	MempoolTxCount     int                          `json:"mempool_tx_count"`   // Pending txs at revocation
	ClearedTxCount     int                          `json:"cleared_tx_count"`   // How many cleared
}

// OptimisticRevocation implements optimistic revocation with collateral
// This approach:
// 1. Immediately rejects NEW transactions for a PoA
// 2. Allows existing mempool transactions to complete (fairness)
// 3. Requires collateral deposit (slashed if revocation was malicious)
// 4. Provides challenge window for disputes
type OptimisticRevocation struct {
	redis              *redis.ClusterClient
	logger             Logger
	oracle             *EmergencyRevocationOracle
	challengeWindow    time.Duration // How long validators can challenge (default: 15 minutes)
	mempoolClearTime   time.Duration // Estimated time for mempool to clear (default: 60 seconds)
	minCollateral      uint64        // Minimum collateral required (Wei)
	states             sync.Map      // poaID → *OptimisticRevocationState
}

// NewOptimisticRevocation creates a new optimistic revocation system
func NewOptimisticRevocation(redisAddrs []string, oracle *EmergencyRevocationOracle, logger Logger) (*OptimisticRevocation, error) {
	if len(redisAddrs) == 0 {
		return nil, fmt.Errorf("at least one Redis address required")
	}
	if oracle == nil {
		return nil, fmt.Errorf("oracle cannot be nil")
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

	or := &OptimisticRevocation{
		redis:            rdb,
		logger:           logger,
		oracle:           oracle,
		challengeWindow:  15 * time.Minute,
		mempoolClearTime: 60 * time.Second,
		minCollateral:    1e18, // 1 ETH minimum
	}

	logger.Info("Optimistic Revocation system initialized")
	return or, nil
}

// MarkPendingRevocation marks a PoA as pending revocation (with collateral)
// This immediately rejects NEW transactions while allowing mempool to clear
func (o *OptimisticRevocation) MarkPendingRevocation(ctx context.Context, poaID, principal, reason string, collateral uint64) error {
	start := time.Now()
	o.logger.Infof("Marking PoA %s as pending revocation (collateral: %d Wei)", poaID, collateral)

	// Validate collateral
	if collateral < o.minCollateral {
		return fmt.Errorf("insufficient collateral: %d Wei (minimum: %d Wei)", collateral, o.minCollateral)
	}

	// Check if already pending or finalized
	currentState, err := o.GetRevocationState(ctx, poaID)
	if err != nil {
		return fmt.Errorf("failed to get current state: %w", err)
	}

	if currentState != nil {
		if currentState.Status == OptimisticStatusPending {
			return fmt.Errorf("PoA %s already pending revocation", poaID)
		}
		if currentState.Status == OptimisticStatusFinalized {
			return fmt.Errorf("PoA %s already finalized (cannot re-revoke)", poaID)
		}
	}

	// Get current mempool count (in production, query actual mempool)
	mempoolTxCount, err := o.getMempoolTxCount(ctx, poaID)
	if err != nil {
		o.logger.Warnf("Failed to get mempool count (non-fatal): %v", err)
		mempoolTxCount = 0
	}

	// Create pending state
	state := &OptimisticRevocationState{
		PoAID:             poaID,
		Status:            OptimisticStatusPending,
		PendingAt:         time.Now(),
		Reason:            reason,
		Principal:         principal,
		Collateral:        collateral,
		ChallengeDeadline: time.Now().Add(o.challengeWindow),
		MempoolTxCount:    mempoolTxCount,
		ClearedTxCount:    0,
	}

	// Store in Redis
	key := fmt.Sprintf("optimistic_revocation:%s", poaID)
	stateJSON, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	// Set with TTL (auto-expire after 24 hours)
	if err := o.redis.Set(ctx, key, stateJSON, 24*time.Hour).Err(); err != nil {
		return fmt.Errorf("redis set failed: %w", err)
	}

	// Store in local cache
	o.states.Store(poaID, state)

	// Broadcast via oracle (validators will immediately reject NEW transactions)
	event := &RevocationEvent{
		PoAID:     poaID,
		Principal: principal,
		Reason:    fmt.Sprintf("PENDING_REVOCATION: %s (collateral: %d Wei)", reason, collateral),
		Timestamp: time.Now(),
		TTL:       int64(o.challengeWindow.Seconds()),
	}

	if err := o.oracle.EmergencyRevoke(ctx, event); err != nil {
		o.logger.Errorf("Oracle broadcast failed (non-fatal): %v", err)
		// Continue - Redis state is primary source of truth
	}

	// Schedule automatic finalization after mempool clears
	go o.scheduleFinalization(poaID, o.mempoolClearTime)

	duration := time.Since(start)
	o.logger.Infof("✅ PoA %s marked as pending revocation in %v (collateral: %d Wei, mempool: %d txs)", 
		poaID, duration, collateral, mempoolTxCount)

	return nil
}

// scheduleFinalization automatically finalizes a pending revocation after mempool clears
func (o *OptimisticRevocation) scheduleFinalization(poaID string, delay time.Duration) {
	time.Sleep(delay)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Check if still pending (might have been challenged)
	state, err := o.GetRevocationState(ctx, poaID)
	if err != nil {
		o.logger.Errorf("Failed to get state for auto-finalization: %v", err)
		return
	}

	if state == nil || state.Status != OptimisticStatusPending {
		o.logger.Infof("PoA %s not in pending state, skipping auto-finalization", poaID)
		return
	}

	o.logger.Infof("Auto-finalizing PoA %s (mempool cleared)", poaID)
	if err := o.FinalizeRevocation(ctx, poaID); err != nil {
		o.logger.Errorf("Auto-finalization failed for PoA %s: %v", poaID, err)
	}
}

// FinalizeRevocation permanently revokes a PoA after mempool clears
// This can be called manually or triggered automatically after mempoolClearTime
func (o *OptimisticRevocation) FinalizeRevocation(ctx context.Context, poaID string) error {
	start := time.Now()
	o.logger.Infof("Finalizing revocation for PoA %s", poaID)

	// Check current state
	state, err := o.GetRevocationState(ctx, poaID)
	if err != nil {
		return fmt.Errorf("failed to get current state: %w", err)
	}

	if state == nil {
		return fmt.Errorf("PoA %s not found (must be pending first)", poaID)
	}

	if state.Status == OptimisticStatusFinalized {
		return fmt.Errorf("PoA %s already finalized", poaID)
	}

	if state.Status == OptimisticStatusChallenged {
		return fmt.Errorf("PoA %s was challenged (cannot finalize)", poaID)
	}

	// Get final mempool count
	mempoolTxCount, err := o.getMempoolTxCount(ctx, poaID)
	if err != nil {
		o.logger.Warnf("Failed to get mempool count (non-fatal): %v", err)
		mempoolTxCount = 0
	}

	// Update state to finalized
	state.Status = OptimisticStatusFinalized
	state.FinalizedAt = time.Now()
	state.ClearedTxCount = state.MempoolTxCount - mempoolTxCount

	// Store updated state
	key := fmt.Sprintf("optimistic_revocation:%s", poaID)
	stateJSON, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	// Set with longer TTL for finalized revocations (90 days)
	if err := o.redis.Set(ctx, key, stateJSON, 90*24*time.Hour).Err(); err != nil {
		return fmt.Errorf("redis set failed: %w", err)
	}

	// Update local cache
	o.states.Store(poaID, state)

	// Broadcast finalized revocation
	event := &RevocationEvent{
		PoAID:     poaID,
		Principal: state.Principal,
		Reason:    fmt.Sprintf("FINALIZED: %s (cleared %d txs)", state.Reason, state.ClearedTxCount),
		Timestamp: time.Now(),
		TTL:       7776000, // 90 days
	}

	if err := o.oracle.EmergencyRevoke(ctx, event); err != nil {
		o.logger.Errorf("Oracle broadcast failed (non-fatal): %v", err)
	}

	// Collateral can now be released (in production, call smart contract)
	o.logger.Infof("Collateral release: %d Wei to %s", state.Collateral, state.Principal)

	duration := time.Since(start)
	o.logger.Infof("✅ PoA %s finalized in %v (cleared %d/%d mempool txs)", 
		poaID, duration, state.ClearedTxCount, state.MempoolTxCount)

	return nil
}

// ChallengeRevocation challenges a pending revocation (if malicious)
// This slashes the collateral and returns the PoA to active status
func (o *OptimisticRevocation) ChallengeRevocation(ctx context.Context, poaID, challenger, evidence string) error {
	start := time.Now()
	o.logger.Infof("Challenging revocation for PoA %s (challenger: %s)", poaID, challenger)

	// Check current state
	state, err := o.GetRevocationState(ctx, poaID)
	if err != nil {
		return fmt.Errorf("failed to get current state: %w", err)
	}

	if state == nil {
		return fmt.Errorf("PoA %s not found", poaID)
	}

	if state.Status != OptimisticStatusPending {
		return fmt.Errorf("PoA %s not in pending state (current: %s)", poaID, state.Status)
	}

	// Check if within challenge window
	if time.Now().After(state.ChallengeDeadline) {
		return fmt.Errorf("challenge window expired (deadline was %v)", state.ChallengeDeadline)
	}

	// Update state to challenged
	state.Status = OptimisticStatusChallenged
	state.ChallengedAt = time.Now()

	// Store updated state
	key := fmt.Sprintf("optimistic_revocation:%s", poaID)
	stateJSON, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	// Set with TTL (7 days for audit)
	if err := o.redis.Set(ctx, key, stateJSON, 7*24*time.Hour).Err(); err != nil {
		return fmt.Errorf("redis set failed: %w", err)
	}

	// Update local cache
	o.states.Store(poaID, state)

	// Slash collateral (in production, send to challenger or burn)
	o.logger.Infof("Collateral slashed: %d Wei from %s (challenger: %s)", 
		state.Collateral, state.Principal, challenger)
	o.logger.Infof("Challenge evidence: %s", evidence)

	// Clear revocation from oracle (PoA returns to active)
	revokedKey := fmt.Sprintf("revoked:%s", poaID)
	if err := o.redis.Del(ctx, revokedKey).Err(); err != nil {
		o.logger.Warnf("Failed to clear oracle revocation (non-fatal): %v", err)
	}

	duration := time.Since(start)
	o.logger.Infof("✅ Challenge successful in %v: PoA %s returned to active, collateral slashed", 
		poaID, duration)

	return nil
}

// GetRevocationState retrieves the current state of an optimistic revocation
func (o *OptimisticRevocation) GetRevocationState(ctx context.Context, poaID string) (*OptimisticRevocationState, error) {
	// Check local cache first
	if cached, ok := o.states.Load(poaID); ok {
		// Return a copy to prevent race conditions when state is modified concurrently
		original := cached.(*OptimisticRevocationState)
		stateCopy := *original
		return &stateCopy, nil
	}

	// Check Redis
	key := fmt.Sprintf("optimistic_revocation:%s", poaID)
	data, err := o.redis.Get(ctx, key).Result()
	if err == redis.Nil {
		// Not in Redis = no pending revocation
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("redis get failed: %w", err)
	}

	var state OptimisticRevocationState
	if err := json.Unmarshal([]byte(data), &state); err != nil {
		return nil, fmt.Errorf("failed to unmarshal state: %w", err)
	}

	// Update local cache
	o.states.Store(poaID, &state)

	return &state, nil
}

// IsPoAUsable checks if a PoA can be used for NEW transactions
// Returns false if pending or finalized revocation exists
func (o *OptimisticRevocation) IsPoAUsable(ctx context.Context, poaID string) (bool, string, error) {
	state, err := o.GetRevocationState(ctx, poaID)
	if err != nil {
		return false, "", fmt.Errorf("failed to get state: %w", err)
	}

	// No state = active (no revocation)
	if state == nil {
		return true, "PoA is active", nil
	}

	switch state.Status {
	case OptimisticStatusPending:
		return false, fmt.Sprintf("PoA revocation pending (reason: %s, challenge deadline: %v)", 
			state.Reason, state.ChallengeDeadline), nil
	case OptimisticStatusFinalized:
		return false, fmt.Sprintf("PoA permanently revoked (reason: %s, finalized at: %v)", 
			state.Reason, state.FinalizedAt), nil
	case OptimisticStatusChallenged:
		// Challenged = returned to active
		return true, fmt.Sprintf("PoA active (revocation challenged at: %v)", state.ChallengedAt), nil
	default:
		return false, fmt.Sprintf("Unknown status: %s", state.Status), nil
	}
}

// getMempoolTxCount queries the mempool for pending transactions from this PoA
// In production, this would query the actual blockchain mempool
func (o *OptimisticRevocation) getMempoolTxCount(ctx context.Context, poaID string) (int, error) {
	// TODO: Implement actual mempool query
	// For now, return 0 (in production, query eth_pendingTransactions or similar)
	return 0, nil
}

// SetChallengeWindow configures how long validators can challenge a revocation
func (o *OptimisticRevocation) SetChallengeWindow(window time.Duration) {
	o.challengeWindow = window
	o.logger.Infof("Challenge window set to %v", window)
}

// GetChallengeWindow returns the current challenge window
func (o *OptimisticRevocation) GetChallengeWindow() time.Duration {
	return o.challengeWindow
}

// SetMempoolClearTime configures how long to wait for mempool to clear
func (o *OptimisticRevocation) SetMempoolClearTime(duration time.Duration) {
	o.mempoolClearTime = duration
	o.logger.Infof("Mempool clear time set to %v", duration)
}

// GetMempoolClearTime returns the current mempool clear time
func (o *OptimisticRevocation) GetMempoolClearTime() time.Duration {
	return o.mempoolClearTime
}

// SetMinCollateral sets the minimum collateral required for revocation
func (o *OptimisticRevocation) SetMinCollateral(amount uint64) {
	o.minCollateral = amount
	o.logger.Infof("Minimum collateral set to %d Wei", amount)
}

// GetMinCollateral returns the minimum collateral required
func (o *OptimisticRevocation) GetMinCollateral() uint64 {
	return o.minCollateral
}

// Close gracefully shuts down the optimistic revocation system
func (o *OptimisticRevocation) Close() error {
	if err := o.redis.Close(); err != nil {
		return fmt.Errorf("failed to close Redis connection: %w", err)
	}

	o.logger.Info("Optimistic Revocation system shut down successfully")
	return nil
}

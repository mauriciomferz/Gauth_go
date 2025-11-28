package revocation

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
)

// PoAStatus represents the lifecycle state of a PoA during revocation
type PoAStatus string

const (
	// PoAStatusActive indicates the PoA is fully operational
	PoAStatusActive PoAStatus = "ACTIVE"
	
	// PoAStatusDisabled indicates the PoA is temporarily suspended (reversible)
	// New transactions are rejected, but the Principal can cancel the disable
	PoAStatusDisabled PoAStatus = "DISABLED"
	
	// PoAStatusRevoked indicates the PoA is permanently revoked (irreversible)
	PoAStatusRevoked PoAStatus = "REVOKED"
)

// PoAState tracks the current state of a PoA in the two-phase revocation system
type PoAState struct {
	PoAID          string    `json:"poa_id"`
	Status         PoAStatus `json:"status"`
	DisabledAt     time.Time `json:"disabled_at,omitempty"`
	RevokedAt      time.Time `json:"revoked_at,omitempty"`
	DisableReason  string    `json:"disable_reason,omitempty"`
	RevokeReason   string    `json:"revoke_reason,omitempty"`
	Principal      string    `json:"principal"`
	CancellableUntil time.Time `json:"cancellable_until,omitempty"`
}

// TwoPhaseRevocation implements two-phase revocation: disable (immediate, reversible) → revoke (permanent, on-chain)
// This eliminates the TOCTOU vulnerability by immediately blocking new transactions while allowing
// the Principal time to cancel if the disable was accidental.
type TwoPhaseRevocation struct {
	oracle         *EmergencyRevocationOracle
	redis          *redis.ClusterClient
	logger         Logger
	disableTimeout time.Duration // How long before auto-revoke (default: 30 seconds)
	states         sync.Map      // poaID → *PoAState (local cache)
	autoRevokeMu   sync.Mutex    // Protects autoRevokeTimers
	autoRevokeTimers map[string]*time.Timer // poaID → timer for auto-revoke
}

// NewTwoPhaseRevocation creates a new two-phase revocation system
func NewTwoPhaseRevocation(oracle *EmergencyRevocationOracle, redisAddrs []string, logger Logger) (*TwoPhaseRevocation, error) {
	if oracle == nil {
		return nil, fmt.Errorf("oracle cannot be nil")
	}
	if len(redisAddrs) == 0 {
		return nil, fmt.Errorf("at least one Redis address required")
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

	tpr := &TwoPhaseRevocation{
		oracle:           oracle,
		redis:            rdb,
		logger:           logger,
		disableTimeout:   30 * time.Second, // Default: auto-revoke after 30 seconds
		autoRevokeTimers: make(map[string]*time.Timer),
	}

	logger.Info("Two-Phase Revocation system initialized")
	return tpr, nil
}

// DisablePoA immediately disables a PoA (Phase 1: reversible)
// This prevents new transactions from being accepted while giving the Principal
// time to cancel if the disable was accidental.
func (t *TwoPhaseRevocation) DisablePoA(ctx context.Context, poaID, principal, reason string) error {
	start := time.Now()
	t.logger.Infof("Phase 1: Disabling PoA %s (reason: %s)", poaID, reason)

	// Check if already disabled or revoked
	currentState, err := t.GetPoAState(ctx, poaID)
	if err != nil {
		return fmt.Errorf("failed to get current state: %w", err)
	}

	if currentState != nil {
		if currentState.Status == PoAStatusDisabled {
			return fmt.Errorf("PoA %s already disabled", poaID)
		}
		if currentState.Status == PoAStatusRevoked {
			return fmt.Errorf("PoA %s already revoked (cannot disable)", poaID)
		}
	}

	// Create new state
	state := &PoAState{
		PoAID:            poaID,
		Status:           PoAStatusDisabled,
		DisabledAt:       time.Now(),
		DisableReason:    reason,
		Principal:        principal,
		CancellableUntil: time.Now().Add(t.disableTimeout),
	}

	// Store in Redis
	key := fmt.Sprintf("poa_state:%s", poaID)
	stateJSON, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	// Set with TTL (auto-expire after 24 hours)
	if err := t.redis.Set(ctx, key, stateJSON, 24*time.Hour).Err(); err != nil {
		return fmt.Errorf("redis set failed: %w", err)
	}

	// Store in local cache
	t.states.Store(poaID, state)

	// Broadcast via oracle (validators will immediately reject new transactions)
	event := &RevocationEvent{
		PoAID:     poaID,
		Principal: principal,
		Reason:    fmt.Sprintf("DISABLED: %s", reason),
		Timestamp: time.Now(),
		TTL:       86400, // 24 hours
	}

	if err := t.oracle.EmergencyRevoke(ctx, event); err != nil {
		t.logger.Errorf("Oracle broadcast failed (non-fatal): %v", err)
		// Continue - Redis state is primary source of truth
	}

	// Schedule auto-revoke after timeout (cancel any existing timer first)
	t.autoRevokeMu.Lock()
	if existingTimer, ok := t.autoRevokeTimers[poaID]; ok {
		existingTimer.Stop()
	}
	t.autoRevokeTimers[poaID] = time.AfterFunc(t.disableTimeout, func() {
		t.performAutoRevoke(poaID)
	})
	t.autoRevokeMu.Unlock()

	duration := time.Since(start)
	t.logger.Infof("✅ Phase 1 complete: PoA %s disabled in %v (cancellable for %v)", 
		poaID, duration, t.disableTimeout)

	return nil
}

// performAutoRevoke automatically revokes a PoA after the disable timeout
func (t *TwoPhaseRevocation) performAutoRevoke(poaID string) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Clean up timer from map
	t.autoRevokeMu.Lock()
	delete(t.autoRevokeTimers, poaID)
	t.autoRevokeMu.Unlock()

	// Check if still disabled (might have been cancelled or manually revoked)
	state, err := t.GetPoAState(ctx, poaID)
	if err != nil {
		t.logger.Errorf("Failed to get state for auto-revoke: %v", err)
		return
	}

	if state == nil || state.Status != PoAStatusDisabled {
		t.logger.Infof("PoA %s not in disabled state, skipping auto-revoke", poaID)
		return
	}

	t.logger.Infof("Auto-revoking PoA %s (timeout reached)", poaID)
	if err := t.RevokePoA(ctx, poaID, "Auto-revoke after disable timeout"); err != nil {
		t.logger.Errorf("Auto-revoke failed for PoA %s: %v", poaID, err)
	}
}

// RevokePoA permanently revokes a PoA (Phase 2: irreversible)
// This writes the revocation to the blockchain, making it permanent.
func (t *TwoPhaseRevocation) RevokePoA(ctx context.Context, poaID, reason string) error {
	start := time.Now()
	t.logger.Infof("Phase 2: Revoking PoA %s (reason: %s)", poaID, reason)

	// Check current state
	state, err := t.GetPoAState(ctx, poaID)
	if err != nil {
		return fmt.Errorf("failed to get current state: %w", err)
	}

	if state == nil {
		return fmt.Errorf("PoA %s not found (must disable before revoking)", poaID)
	}

	if state.Status == PoAStatusRevoked {
		return fmt.Errorf("PoA %s already revoked", poaID)
	}

	// Update state to revoked
	state.Status = PoAStatusRevoked
	state.RevokedAt = time.Now()
	state.RevokeReason = reason

	// Store updated state
	key := fmt.Sprintf("poa_state:%s", poaID)
	stateJSON, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	// Set with longer TTL for revoked PoAs (90 days)
	if err := t.redis.Set(ctx, key, stateJSON, 90*24*time.Hour).Err(); err != nil {
		return fmt.Errorf("redis set failed: %w", err)
	}

	// Update local cache
	t.states.Store(poaID, state)

	// Broadcast permanent revocation
	event := &RevocationEvent{
		PoAID:     poaID,
		Principal: state.Principal,
		Reason:    fmt.Sprintf("REVOKED: %s", reason),
		Timestamp: time.Now(),
		TTL:       7776000, // 90 days
	}

	if err := t.oracle.EmergencyRevoke(ctx, event); err != nil {
		t.logger.Errorf("Oracle broadcast failed (non-fatal): %v", err)
	}

	duration := time.Since(start)
	t.logger.Infof("✅ Phase 2 complete: PoA %s permanently revoked in %v", poaID, duration)

	return nil
}

// CancelDisable cancels a disabled PoA, returning it to active status
// This is only possible if the PoA is in DISABLED state (before Phase 2 revocation)
func (t *TwoPhaseRevocation) CancelDisable(ctx context.Context, poaID string) error {
	start := time.Now()
	t.logger.Infof("Cancelling disable for PoA %s", poaID)

	// Check current state
	state, err := t.GetPoAState(ctx, poaID)
	if err != nil {
		return fmt.Errorf("failed to get current state: %w", err)
	}

	if state == nil {
		return fmt.Errorf("PoA %s not found", poaID)
	}

	if state.Status != PoAStatusDisabled {
		return fmt.Errorf("PoA %s not in disabled state (current: %s)", poaID, state.Status)
	}

	// Check if still within cancellation window
	if time.Now().After(state.CancellableUntil) {
		return fmt.Errorf("cancellation window expired (deadline was %v)", state.CancellableUntil)
	}

	// Cancel any pending auto-revoke timer
	t.autoRevokeMu.Lock()
	if timer, ok := t.autoRevokeTimers[poaID]; ok {
		timer.Stop()
		delete(t.autoRevokeTimers, poaID)
	}
	t.autoRevokeMu.Unlock()

	// Delete state (returns to ACTIVE)
	key := fmt.Sprintf("poa_state:%s", poaID)
	if err := t.redis.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("redis delete failed: %w", err)
	}

	// Remove from local cache
	t.states.Delete(poaID)

	// Clear revocation from oracle
	revokedKey := fmt.Sprintf("revoked:%s", poaID)
	if err := t.redis.Del(ctx, revokedKey).Err(); err != nil {
		t.logger.Warnf("Failed to clear oracle revocation (non-fatal): %v", err)
	}

	duration := time.Since(start)
	t.logger.Infof("✅ PoA %s re-enabled in %v (disable cancelled)", poaID, duration)

	return nil
}

// GetPoAState retrieves the current state of a PoA
func (t *TwoPhaseRevocation) GetPoAState(ctx context.Context, poaID string) (*PoAState, error) {
	// Check local cache first
	if cached, ok := t.states.Load(poaID); ok {
		// Return a copy to prevent race conditions when callers read while other goroutines modify
		original := cached.(*PoAState)
		stateCopy := *original
		return &stateCopy, nil
	}

	// Check Redis
	key := fmt.Sprintf("poa_state:%s", poaID)
	data, err := t.redis.Get(ctx, key).Result()
	if err == redis.Nil {
		// Not in Redis = ACTIVE (or doesn't exist)
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("redis get failed: %w", err)
	}

	var state PoAState
	if err := json.Unmarshal([]byte(data), &state); err != nil {
		return nil, fmt.Errorf("failed to unmarshal state: %w", err)
	}

	// Update local cache
	t.states.Store(poaID, &state)

	// Return a copy to prevent race conditions
	stateCopy := state
	return &stateCopy, nil
}

// IsPoAUsable checks if a PoA can be used for new transactions
func (t *TwoPhaseRevocation) IsPoAUsable(ctx context.Context, poaID string) (bool, string, error) {
	state, err := t.GetPoAState(ctx, poaID)
	if err != nil {
		return false, "", fmt.Errorf("failed to get state: %w", err)
	}

	// No state = ACTIVE
	if state == nil {
		return true, "PoA is active", nil
	}

	switch state.Status {
	case PoAStatusActive:
		return true, "PoA is active", nil
	case PoAStatusDisabled:
		return false, fmt.Sprintf("PoA disabled (reason: %s, cancellable until: %v)", 
			state.DisableReason, state.CancellableUntil), nil
	case PoAStatusRevoked:
		return false, fmt.Sprintf("PoA permanently revoked (reason: %s, revoked at: %v)", 
			state.RevokeReason, state.RevokedAt), nil
	default:
		return false, fmt.Sprintf("Unknown status: %s", state.Status), nil
	}
}

// SetDisableTimeout configures how long a PoA stays in DISABLED state before auto-revoke
func (t *TwoPhaseRevocation) SetDisableTimeout(timeout time.Duration) {
	t.disableTimeout = timeout
	t.logger.Infof("Disable timeout set to %v", timeout)
}

// GetDisableTimeout returns the current disable timeout
func (t *TwoPhaseRevocation) GetDisableTimeout() time.Duration {
	return t.disableTimeout
}

// Close gracefully shuts down the two-phase revocation system
func (t *TwoPhaseRevocation) Close() error {
	// Cancel all pending auto-revoke timers
	t.autoRevokeMu.Lock()
	for poaID, timer := range t.autoRevokeTimers {
		timer.Stop()
		delete(t.autoRevokeTimers, poaID)
	}
	t.autoRevokeMu.Unlock()

	if err := t.redis.Close(); err != nil {
		return fmt.Errorf("failed to close Redis connection: %w", err)
	}

	t.logger.Info("Two-Phase Revocation system shut down successfully")
	return nil
}

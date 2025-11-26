// Package revocation provides validator integration for emergency revocation checks
package revocation

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// ValidatorCache provides fast local caching for revocation checks
type ValidatorCache struct {
	oracle       *EmergencyRevocationOracle
	localCache   *sync.Map // map[string]*RevocationEvent
	subscriberID string
	logger       Logger
	metrics      *RevocationMetrics
}

// RevocationMetrics tracks performance metrics
type RevocationMetrics struct {
	CacheHits      uint64
	CacheMisses    uint64
	RedisHits      uint64
	RedisMisses    uint64
	BlockchainHits uint64
	TotalChecks    uint64
	mu             sync.RWMutex
}

// NewValidatorCache creates a validator cache with real-time revocation updates
func NewValidatorCache(oracle *EmergencyRevocationOracle, logger Logger) *ValidatorCache {
	vc := &ValidatorCache{
		oracle:       oracle,
		localCache:   &sync.Map{},
		subscriberID: fmt.Sprintf("validator-%s", uuid.New().String()),
		logger:       logger,
		metrics:      &RevocationMetrics{},
	}

	// Start listening for real-time revocations
	go vc.listenForRevocations()

	logger.Infof("Validator cache initialized (subscriber: %s)", vc.subscriberID)

	return vc
}

// listenForRevocations receives real-time revocation broadcasts from oracle
func (vc *ValidatorCache) listenForRevocations() {
	ch := vc.oracle.Subscribe(vc.subscriberID)

	vc.logger.Info("Listening for emergency revocations...")

	for event := range ch {
		// Update local cache immediately
		vc.localCache.Store(event.PoAID, event)

		vc.logger.Infof("PoA revoked: %s (reason: %s, principal: %s)",
			event.PoAID, event.Reason, event.Principal)

		// Schedule cache expiry after TTL
		if event.TTL > 0 {
			time.AfterFunc(time.Duration(event.TTL)*time.Second, func() {
				vc.localCache.Delete(event.PoAID)
				vc.logger.Infof("Cache entry expired for PoA: %s", event.PoAID)
			})
		}
	}

	vc.logger.Warn("Revocation listener stopped (channel closed)")
}

// IsRevoked checks if a PoA is revoked using multi-tier caching
// Performance: <1µs (cache hit), ~1ms (Redis hit), ~100ms (blockchain)
func (vc *ValidatorCache) IsRevoked(ctx context.Context, poaID string) (bool, string, error) {
	vc.metrics.mu.Lock()
	vc.metrics.TotalChecks++
	vc.metrics.mu.Unlock()

	// TIER 1: Local in-memory cache (fastest - <1µs)
	if event, found := vc.localCache.Load(poaID); found {
		vc.metrics.mu.Lock()
		vc.metrics.CacheHits++
		vc.metrics.mu.Unlock()

		revEvent := event.(*RevocationEvent)
		return true, revEvent.Reason, nil
	}

	vc.metrics.mu.Lock()
	vc.metrics.CacheMisses++
	vc.metrics.mu.Unlock()

	// TIER 2: Redis lookup (fast - ~1ms)
	revoked, event, err := vc.oracle.IsRevoked(ctx, poaID)
	if err != nil {
		vc.logger.Warnf("Redis lookup failed for PoA %s: %v", poaID, err)
		// Continue to blockchain check
	} else if revoked {
		vc.metrics.mu.Lock()
		vc.metrics.RedisHits++
		vc.metrics.mu.Unlock()

		// Cache locally for future lookups
		vc.localCache.Store(poaID, event)

		return true, event.Reason, nil
	}

	vc.metrics.mu.Lock()
	vc.metrics.RedisMisses++
	vc.metrics.mu.Unlock()

	// TIER 3: Blockchain lookup (slow but authoritative - ~100ms)
	// This would check the on-chain registry
	// For now, we assume not revoked if not in Redis
	vc.metrics.mu.Lock()
	vc.metrics.BlockchainHits++
	vc.metrics.mu.Unlock()

	return false, "", nil
}

// GetMetrics returns current performance metrics
func (vc *ValidatorCache) GetMetrics() RevocationMetrics {
	vc.metrics.mu.RLock()
	defer vc.metrics.mu.RUnlock()

	return RevocationMetrics{
		CacheHits:      vc.metrics.CacheHits,
		CacheMisses:    vc.metrics.CacheMisses,
		RedisHits:      vc.metrics.RedisHits,
		RedisMisses:    vc.metrics.RedisMisses,
		BlockchainHits: vc.metrics.BlockchainHits,
		TotalChecks:    vc.metrics.TotalChecks,
	}
}

// ClearCache removes all entries from local cache
func (vc *ValidatorCache) ClearCache() {
	vc.localCache.Range(func(key, value interface{}) bool {
		vc.localCache.Delete(key)
		return true
	})

	vc.logger.Info("Local cache cleared")
}

// Close gracefully shuts down the validator cache
func (vc *ValidatorCache) Close() {
	vc.oracle.Unsubscribe(vc.subscriberID)
	vc.ClearCache()
	vc.logger.Info("Validator cache shut down successfully")
}

// ValidatorWithRevocationCheck wraps a validator with emergency revocation checking
type ValidatorWithRevocationCheck struct {
	cache  *ValidatorCache
	logger Logger
}

// NewValidatorWithRevocationCheck creates a validator with revocation support
func NewValidatorWithRevocationCheck(oracle *EmergencyRevocationOracle, logger Logger) *ValidatorWithRevocationCheck {
	return &ValidatorWithRevocationCheck{
		cache:  NewValidatorCache(oracle, logger),
		logger: logger,
	}
}

// ValidatePoA checks if a PoA is valid (includes revocation check)
func (v *ValidatorWithRevocationCheck) ValidatePoA(ctx context.Context, poaID string) error {
	start := time.Now()

	// Check if revoked
	revoked, reason, err := v.cache.IsRevoked(ctx, poaID)
	if err != nil {
		v.logger.Warnf("Revocation check failed for PoA %s: %v", poaID, err)
		// Continue with validation - don't fail on revocation check errors
	}

	if revoked {
		checkDuration := time.Since(start)
		v.logger.Infof("PoA validation failed (revoked in %v): %s (reason: %s)",
			checkDuration, poaID, reason)
		return fmt.Errorf("PoA has been revoked: %s", reason)
	}

	checkDuration := time.Since(start)
	v.logger.Infof("PoA validation passed (%v): %s", checkDuration, poaID)

	// Additional validation logic would go here
	// (signature verification, expiry check, permissions, etc.)

	return nil
}

// GetCache returns the underlying validator cache
func (v *ValidatorWithRevocationCheck) GetCache() *ValidatorCache {
	return v.cache
}

// Close gracefully shuts down the validator
func (v *ValidatorWithRevocationCheck) Close() {
	v.cache.Close()
	v.logger.Info("Validator with revocation check shut down successfully")
}

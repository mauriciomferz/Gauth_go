package crypto

import (
	"context"
	"fmt"
	"math/rand"
	"time"
)

// NewMultiTenantKeyManager creates a new multi-tenant key manager.
func NewMultiTenantKeyManager(defaultStore KeyStore, defaultPolicy *RotationPolicy) *MultiTenantKeyManager {
	return &MultiTenantKeyManager{
		stores:        make(map[string]KeyStore),
		policies:      make(map[string]*RotationPolicy),
		schedulers:    make(map[string]*TenantScheduler),
		defaultStore:  defaultStore,
		defaultPolicy: defaultPolicy,
		statuses:      make(map[string]*RotationStatus),
		keyStore:      defaultStore,
		healthy:       true,
	}
}

// RegisterTenant registers a tenant with specific key store and rotation policy.
func (m *MultiTenantKeyManager) RegisterTenant(tenant string, store KeyStore, policy *RotationPolicy) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.stores[tenant] = store
	m.policies[tenant] = policy

	// Start scheduler if policy is enabled
	if policy.Enabled {
		scheduler, err := m.createScheduler(tenant, policy)
		if err != nil {
			return fmt.Errorf("failed to create scheduler for tenant %s: %w", tenant, err)
		}
		m.schedulers[tenant] = scheduler
		go scheduler.run()
	}

	return nil
}

// UnregisterTenant removes a tenant and stops its scheduler.
func (m *MultiTenantKeyManager) UnregisterTenant(tenant string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if scheduler, exists := m.schedulers[tenant]; exists {
		scheduler.stop()
		delete(m.schedulers, tenant)
	}

	delete(m.stores, tenant)
	delete(m.policies, tenant)
}

// GetTenantPolicy returns the rotation policy for a tenant, falling back to default.
func (m *MultiTenantKeyManager) GetTenantPolicy(tenant string) *RotationPolicy {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if policy, exists := m.policies[tenant]; exists {
		return policy
	}
	return m.defaultPolicy
}

// RotateKey manually triggers key rotation for a tenant.
func (m *MultiTenantKeyManager) RotateKey(ctx context.Context, tenant string, rotationType string) (*RotationEvent, error) {
	store := m.GetTenantStore(tenant)
	policy := m.GetTenantPolicy(tenant)

	start := time.Now()
	event := &RotationEvent{
		ID:        fmt.Sprintf("rot-%d-%s", start.UnixNano(), tenant),
		Timestamp: start,
		Tenant:    tenant,
		Type:      rotationType,
		Backend:   policy.Backend,
	}

	// Get current active key for transition tracking
	if activeKey, err := store.GetActive(ctx, tenant); err == nil && activeKey != nil {
		event.OldKeyID = activeKey.ID
	}

	// Generate new key
	newKeyID, err := store.Generate(ctx, tenant)
	if err != nil {
		event.Error = err.Error()
		event.Success = false
		event.RotationDuration = time.Since(start)
		return event, err
	}

	// Activate the new key
	if err := store.Activate(ctx, tenant, newKeyID); err != nil {
		event.Error = err.Error()
		event.Success = false
		event.RotationDuration = time.Since(start)
		return event, err
	}

	// Success
	event.NewKeyID = newKeyID
	event.Success = true
	event.RotationDuration = time.Since(start)

	// Trigger callback if set
	if m.eventCallback != nil {
		m.eventCallback(event)
	}

	return event, nil
}

// SetEventCallback sets a callback function to be called after rotation events.
func (m *MultiTenantKeyManager) SetEventCallback(callback func(*RotationEvent)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.eventCallback = callback
}

// calculateJitteredInterval adds random jitter to prevent thundering herd.
func calculateJitteredInterval(base, maxJitter time.Duration) time.Duration {
	if maxJitter <= 0 {
		return base
	}

	// Generate random jitter: -maxJitter/2 to +maxJitter/2
	//nolint:gosec // G404: weak random acceptable for rotation jitter timing
	jitter := time.Duration(rand.Int63n(int64(maxJitter))) - maxJitter/2
	result := base + jitter

	// Ensure minimum interval
	minInterval := base / 2
	if result < minInterval {
		result = minInterval
	}

	return result
}

// Additional helper methods can be added here as needed

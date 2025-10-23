package crypto

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// KeyStore defines the interface for secure key storage backends.
// This abstraction enables integration with various secure storage systems
// including HashiCorp Vault, cloud KMS services, HSMs, and file-based storage.
type KeyStore interface {
	// Generate creates a new key pair and stores it securely.
	// Returns the key ID and any error encountered.
	Generate(ctx context.Context, tenant string) (keyID string, err error)
	
	// Activate marks a key as the active signing key for a tenant.
	// This operation should be atomic to prevent signing windows.
	Activate(ctx context.Context, tenant, keyID string) error
	
	// Archive marks a key as archived (no longer active but retained for validation).
	// Archived keys should remain accessible for signature verification during grace periods.
	Archive(ctx context.Context, tenant, keyID string) error
	
	// GetActive retrieves the currently active key for a tenant.
	GetActive(ctx context.Context, tenant string) (*Key, error)
	
	// GetKey retrieves a specific key by ID for a tenant.
	GetKey(ctx context.Context, tenant, keyID string) (*Key, error)
	
	// ListKeys returns all keys (active + archived) for a tenant.
	ListKeys(ctx context.Context, tenant string) ([]*Key, error)
	
	// Delete permanently removes a key (use with extreme caution).
	Delete(ctx context.Context, tenant, keyID string) error
	
	// Health checks the connectivity and status of the key store backend.
	Health(ctx context.Context) error
}

// RotationPolicy defines the rules and schedule for key rotation.
type RotationPolicy struct {
	// Enabled indicates whether automatic rotation is active
	Enabled bool `json:"enabled"`
	
	// Interval is the base rotation interval
	Interval time.Duration `json:"interval"`
	
	// Jitter is the random variance added to the interval to prevent thundering herd
	Jitter time.Duration `json:"jitter,omitempty"`
	
	// MaxKeyAge is the maximum age before a key must be rotated
	MaxKeyAge time.Duration `json:"max_key_age,omitempty"`
	
	// GracePeriod is how long old keys remain valid for verification after rotation
	GracePeriod time.Duration `json:"grace_period,omitempty"`
	
	// Backend specifies the key storage backend to use
	Backend string `json:"backend"` // "vault", "kms", "file", "memory"
	
	// BackendConfig contains backend-specific configuration
	BackendConfig map[string]interface{} `json:"backend_config,omitempty"`
}

// RotationState represents the current state of key rotation.
type RotationState string

const (
	RotationStateIdle         RotationState = "idle"
	RotationStatePending      RotationState = "pending"
	RotationStateGenerating   RotationState = "generating"
	RotationStateInProgress   RotationState = "in_progress"
	RotationStateCompleted    RotationState = "completed"
	RotationStateFailed       RotationState = "failed"
)

// RotationStatus tracks the current state of key rotation for a tenant.
type RotationStatus struct {
	State           RotationState `json:"state"`
	LastRotation    *time.Time    `json:"last_rotation,omitempty"`
	NextRotation    *time.Time    `json:"next_rotation,omitempty"`
	LastError       string        `json:"last_error,omitempty"`
	RotationCount   int           `json:"rotation_count"`
	CurrentKeyID    string        `json:"current_key_id,omitempty"`
	PendingKeyID    string        `json:"pending_key_id,omitempty"`
}

// KeyMetadata provides key information without exposing sensitive material.
type KeyMetadata struct {
	ID        string    `json:"id"`
	Algorithm string    `json:"algorithm"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at,omitempty"`
	Active    bool      `json:"active"`
	Tenant    string    `json:"tenant,omitempty"`
	Backend   string    `json:"backend,omitempty"`
}

// RotationEvent represents a key rotation event for audit trails.
type RotationEvent struct {
	ID               string                 `json:"id"`
	Timestamp        time.Time              `json:"timestamp"`
	Tenant           string                 `json:"tenant"`
	Type             string                 `json:"type"` // "scheduled", "manual", "emergency"
	OldKeyID         string                 `json:"old_key_id,omitempty"`
	NewKeyID         string                 `json:"new_key_id"`
	Backend          string                 `json:"backend"`
	RotationDuration time.Duration          `json:"rotation_duration"`
	Success          bool                   `json:"success"`
	Error            string                 `json:"error,omitempty"`
	Metadata         map[string]interface{} `json:"metadata,omitempty"`
}

// MultiTenantKeyManager manages keys across multiple tenants with individual policies.
type MultiTenantKeyManager struct {
	stores         map[string]KeyStore    // tenant -> keystore
	policies       map[string]*RotationPolicy // tenant -> policy
	schedulers     map[string]*TenantScheduler // tenant -> scheduler
	defaultStore   KeyStore
	defaultPolicy  *RotationPolicy
	eventCallback  func(*RotationEvent)
	mu             sync.RWMutex
	keyStore       KeyStore // Unified interface for API access
	healthy        bool     // Overall health status
	statuses       map[string]*RotationStatus // tenant -> status
}

// TenantScheduler manages the rotation schedule for a specific tenant.
type TenantScheduler struct {
	tenant   string
	manager  *MultiTenantKeyManager
	policy   *RotationPolicy
	ticker   *time.Ticker
	stopCh   chan struct{}
	nextRotation time.Time
	mu       sync.RWMutex
}

// Additional methods for MultiTenantKeyManager

// GetRegisteredTenants returns a list of all registered tenants.
func (m *MultiTenantKeyManager) GetRegisteredTenants() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	tenants := make([]string, 0, len(m.stores))
	for tenant := range m.stores {
		tenants = append(tenants, tenant)
	}
	return tenants
}

// GetRotationPolicy returns the rotation policy for a tenant.
func (m *MultiTenantKeyManager) GetRotationPolicy(tenant string) *RotationPolicy {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	if policy, exists := m.policies[tenant]; exists {
		return policy
	}
	return m.defaultPolicy
}

// GetRotationStatus returns the rotation status for a tenant.
func (m *MultiTenantKeyManager) GetRotationStatus(tenant string) *RotationStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	if status, exists := m.statuses[tenant]; exists {
		return status
	}
	
	// Return default status if none exists
	return &RotationStatus{
		State: RotationStateIdle,
	}
}

// UpdateRotationPolicy updates the rotation policy for a tenant.
func (m *MultiTenantKeyManager) UpdateRotationPolicy(tenant string, policy *RotationPolicy) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.policies[tenant] = policy
	
	// Restart scheduler if needed
	if scheduler, exists := m.schedulers[tenant]; exists {
		scheduler.stop()
		delete(m.schedulers, tenant)
	}
	
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

// TriggerRotation manually triggers key rotation for a tenant.
func (m *MultiTenantKeyManager) TriggerRotation(tenant string, force bool, reason string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	store := m.GetTenantStore(tenant)
	if store == nil {
		return fmt.Errorf("no store configured for tenant %s", tenant)
	}
	
	// Update status to in-progress
	if m.statuses == nil {
		m.statuses = make(map[string]*RotationStatus)
	}
	
	status := m.statuses[tenant]
	if status == nil {
		status = &RotationStatus{State: RotationStateIdle}
		m.statuses[tenant] = status
	}
	
	// Check if rotation is already in progress
	if !force && (status.State == RotationStateInProgress || status.State == RotationStateGenerating) {
		return fmt.Errorf("rotation already in progress for tenant %s", tenant)
	}
	
	status.State = RotationStateInProgress
	
	// Perform rotation asynchronously
	go func() {
		defer func() {
			m.mu.Lock()
			if m.statuses[tenant] != nil {
				m.statuses[tenant].State = RotationStateCompleted
				now := time.Now()
				m.statuses[tenant].LastRotation = &now
				m.statuses[tenant].RotationCount++
			}
			m.mu.Unlock()
		}()
		
		ctx := context.Background()
		
		// Generate new key
		keyID, err := store.Generate(ctx, tenant)
		if err != nil {
			m.mu.Lock()
			if m.statuses[tenant] != nil {
				m.statuses[tenant].State = RotationStateFailed
				m.statuses[tenant].LastError = err.Error()
			}
			m.mu.Unlock()
			return
		}
		
		// Activate new key
		if err := store.Activate(ctx, tenant, keyID); err != nil {
			m.mu.Lock()
			if m.statuses[tenant] != nil {
				m.statuses[tenant].State = RotationStateFailed
				m.statuses[tenant].LastError = err.Error()
			}
			m.mu.Unlock()
			return
		}
		
		m.mu.Lock()
		if m.statuses[tenant] != nil {
			m.statuses[tenant].CurrentKeyID = keyID
		}
		m.mu.Unlock()
	}()
	
	return nil
}

// IsHealthy returns the overall health status of the manager.
func (m *MultiTenantKeyManager) IsHealthy() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.healthy
}

// GetTenantStore returns the key store for a tenant, falling back to default.
func (m *MultiTenantKeyManager) GetTenantStore(tenant string) KeyStore {
	if store, exists := m.stores[tenant]; exists {
		return store
	}
	return m.defaultStore
}

// createScheduler creates a new scheduler for a tenant (placeholder).
func (m *MultiTenantKeyManager) createScheduler(tenant string, policy *RotationPolicy) (*TenantScheduler, error) {
	// This would create a new scheduler - simplified for now
	return &TenantScheduler{
		tenant:  tenant,
		manager: m,
		policy:  policy,
		stopCh:  make(chan struct{}),
	}, nil
}

// TenantScheduler methods

// run starts the scheduler main loop.
func (ts *TenantScheduler) run() {
	if ts.policy.Interval <= 0 {
		return
	}
	
	ts.ticker = time.NewTicker(ts.policy.Interval)
	defer ts.ticker.Stop()
	
	for {
		select {
		case <-ts.ticker.C:
			// Trigger rotation
			ctx := context.Background()
			store := ts.manager.GetTenantStore(ts.tenant)
			if store != nil {
				// Simple rotation trigger - could be enhanced
				if _, err := store.Generate(ctx, ts.tenant); err != nil {
					// Log error but continue scheduling
					continue
				}
			}
		case <-ts.stopCh:
			return
		}
	}
}

// stop stops the scheduler.
func (ts *TenantScheduler) stop() {
	close(ts.stopCh)
	if ts.ticker != nil {
		ts.ticker.Stop()
	}
}
package authz

import (
	"context"
	"testing"
	"time"
)

// TestSetDecisionCache verifies attaching authorization cache
func TestSetDecisionCache(t *testing.T) {
	ma := NewMemoryAuthorizer()

	// Initial state: no cache
	if ma.decisionCache != nil {
		t.Error("Initial decisionCache should be nil")
	}

	// Attach cache
	cache := NewAuthorizationCache(100)
	ma.SetDecisionCache(cache)

	if ma.decisionCache != cache {
		t.Error("SetDecisionCache should attach the cache")
	}
}

// TestSetJurisdiction verifies jurisdiction setting and cache invalidation
func TestSetJurisdiction(t *testing.T) {
	ma := NewMemoryAuthorizer()
	cache := NewAuthorizationCache(100)
	ma.SetDecisionCache(cache)

	// Add cache entry
	key := makeKey("user", "read", "doc", 1, "us-east")
	cache.Set(key, AuthorizationCacheEntry{
		Decision:      Decision{Allow: true},
		PolicyVersion: 1,
		Jurisdiction:  "us-east",
	})

	// Verify entry present
	if _, found := cache.Get(key); !found {
		t.Error("Cache entry should be present before jurisdiction change")
	}

	// Change jurisdiction (should invalidate cache)
	ma.SetJurisdiction("eu-west")

	// Verify jurisdiction changed
	if ma.jurisdiction != "eu-west" {
		t.Errorf("jurisdiction = %q, want %q", ma.jurisdiction, "eu-west")
	}

	// Verify cache invalidated
	if size := cache.Size(); size != 0 {
		t.Errorf("Cache size after jurisdiction change = %d, want 0", size)
	}
}

// TestSetJurisdiction_NoChange verifies no invalidation when jurisdiction unchanged
func TestSetJurisdiction_NoChange(t *testing.T) {
	ma := NewMemoryAuthorizer()
	ma.jurisdiction = "us-east"
	cache := NewAuthorizationCache(100)
	ma.SetDecisionCache(cache)

	// Add cache entry
	key := makeKey("user", "read", "doc", 1, "us-east")
	cache.Set(key, AuthorizationCacheEntry{
		Decision:      Decision{Allow: true},
		PolicyVersion: 1,
		Jurisdiction:  "us-east",
	})

	// Set same jurisdiction (no-op)
	ma.SetJurisdiction("us-east")

	// Verify cache NOT invalidated
	if _, found := cache.Get(key); !found {
		t.Error("Cache should not be invalidated when jurisdiction unchanged")
	}
}

// TestSetObligationExecutor verifies obligation executor override
func TestSetObligationExecutor(t *testing.T) {
	ma := NewMemoryAuthorizer()

	// Default executor
	if ma.obligationExecutor == nil {
		t.Error("Default obligationExecutor should not be nil")
	}

	// Custom executor
	customExec := &MockObligationExecutor{}
	ma.SetObligationExecutor(customExec)

	// Can't directly compare interface values, check it's not nil
	if ma.obligationExecutor == nil {
		t.Error("SetObligationExecutor should set custom executor")
	}
}

// TestSetObligationExecutor_NilGuard verifies nil executor is rejected
func TestSetObligationExecutor_NilGuard(t *testing.T) {
	ma := NewMemoryAuthorizer()
	originalExec := ma.obligationExecutor

	// Try to set nil (should be no-op)
	ma.SetObligationExecutor(nil)

	if ma.obligationExecutor != originalExec {
		t.Error("SetObligationExecutor(nil) should not change executor")
	}
}

// TestSetMetricsProvider verifies metrics provider attachment
func TestSetMetricsProvider(t *testing.T) {
	ma := NewMemoryAuthorizer()

	// Initial state: no metrics provider
	if ma.metricsProvider != nil {
		t.Error("Initial metricsProvider should be nil")
	}

	// Attach provider
	mockProvider := &MockMetricsProvider{}
	ma.SetMetricsProvider(mockProvider)

	if ma.metricsProvider != mockProvider {
		t.Error("SetMetricsProvider should attach provider")
	}
}

// TestSetValidatorRegistry verifies validator registry attachment
func TestSetValidatorRegistry(t *testing.T) {
	ma := NewMemoryAuthorizer()

	// Initial state: no registry
	if ma.validatorRegistry != nil {
		t.Error("Initial validatorRegistry should be nil")
	}

	// Attach registry
	registry := NewValidatorRegistry()
	ma.SetValidatorRegistry(registry)

	if ma.validatorRegistry != registry {
		t.Error("SetValidatorRegistry should attach registry")
	}
}

// TestInvalidateOnCryptoRotation verifies cache invalidation on crypto rotation
func TestInvalidateOnCryptoRotation(t *testing.T) {
	ma := NewMemoryAuthorizer()
	cache := NewAuthorizationCache(100)
	ma.SetDecisionCache(cache)

	// Add cache entries
	for i := 0; i < 10; i++ {
		key := makeKey("user", "read", "doc", int64(i), "us")
		cache.Set(key, AuthorizationCacheEntry{
			Decision:      Decision{Allow: true},
			PolicyVersion: int64(i),
		})
	}

	// Verify entries present
	sizeBefore := cache.Size()
	if sizeBefore != 10 {
		t.Errorf("Cache size before = %d, want 10", sizeBefore)
	}

	// Trigger crypto rotation invalidation
	ma.InvalidateOnCryptoRotation()

	// Verify all cache cleared
	sizeAfter := cache.Size()
	if sizeAfter != 0 {
		t.Errorf("Cache size after rotation = %d, want 0", sizeAfter)
	}
}

// TestInvalidateOnCryptoRotation_NoCache verifies safe handling when no cache
func TestInvalidateOnCryptoRotation_NoCache(t *testing.T) {
	ma := NewMemoryAuthorizer()

	// Should not panic
	ma.InvalidateOnCryptoRotation()
}

// TestAuthorizationCacheMetrics verifies metrics snapshot retrieval
func TestAuthorizationCacheMetrics(t *testing.T) {
	ma := NewMemoryAuthorizer()

	// No cache: should return nil
	metrics := ma.AuthorizationCacheMetrics()
	if metrics != nil {
		t.Error("AuthorizationCacheMetrics with no cache should return nil")
	}

	// With cache
	cache := NewAuthorizationCache(50)
	ma.SetDecisionCache(cache)

	// Add some entries and operations
	key := makeKey("user", "read", "doc", 1, "us")
	cache.Set(key, AuthorizationCacheEntry{Decision: Decision{Allow: true}, PolicyVersion: 1})
	cache.Get(key)                // hit
	cache.Get("nonexistent-key") // miss

	metrics = ma.AuthorizationCacheMetrics()
	if metrics == nil {
		t.Fatal("AuthorizationCacheMetrics should return snapshot")
	}
	if metrics.Capacity != 50 {
		t.Errorf("Capacity = %d, want 50", metrics.Capacity)
	}
	if metrics.Size != 1 {
		t.Errorf("Size = %d, want 1", metrics.Size)
	}
	if metrics.Lookups != 2 {
		t.Errorf("Lookups = %d, want 2", metrics.Lookups)
	}
	if metrics.Hits != 1 {
		t.Errorf("Hits = %d, want 1", metrics.Hits)
	}
	if metrics.Misses != 1 {
		t.Errorf("Misses = %d, want 1", metrics.Misses)
	}
}

// TestCurrentPolicyVersion verifies version snapshot calculation
func TestCurrentPolicyVersion(t *testing.T) {
	ma := NewMemoryAuthorizer()

	// Initial version 0
	ma.version = 0
	if v := ma.currentPolicyVersion(); v != 1 {
		t.Errorf("currentPolicyVersion() for version 0 = %d, want 1", v)
	}

	// Version 1
	ma.version = 1
	if v := ma.currentPolicyVersion(); v != 1 {
		t.Errorf("currentPolicyVersion() for version 1 = %d, want 1", v)
	}

	// Version 2 (returns v-1 = 1)
	ma.version = 2
	if v := ma.currentPolicyVersion(); v != 1 {
		t.Errorf("currentPolicyVersion() for version 2 = %d, want 1", v)
	}

	// Version 10
	ma.version = 10
	if v := ma.currentPolicyVersion(); v != 9 {
		t.Errorf("currentPolicyVersion() for version 10 = %d, want 9", v)
	}
}

// TestDisableCaching verifies cache disabling
func TestDisableCaching(t *testing.T) {
	ma := NewMemoryAuthorizer()
	
	// Initially enabled
	ma.cacheEnabled = true

	// Disable
	ma.DisableCaching()

	if ma.cacheEnabled {
		t.Error("DisableCaching should set cacheEnabled to false")
	}
}

// TestAssignRoles verifies role assignment to subjects
func TestAssignRoles(t *testing.T) {
	ma := NewMemoryAuthorizer()

	// Assign roles
	ma.AssignRoles("user:123", "admin", "editor")

	// Verify assignment
	roles, ok := ma.roles["user:123"]
	if !ok {
		t.Fatal("Roles should be assigned to subject")
	}
	if len(roles) != 2 {
		t.Errorf("Role count = %d, want 2", len(roles))
	}
	if roles[0] != "admin" || roles[1] != "editor" {
		t.Errorf("Roles = %v, want [admin editor]", roles)
	}
}

// TestAssignRoles_EmptyRoles verifies assigning empty role list
func TestAssignRoles_EmptyRoles(t *testing.T) {
	ma := NewMemoryAuthorizer()

	// Assign empty roles
	ma.AssignRoles("user:456")

	// Verify assignment
	roles, ok := ma.roles["user:456"]
	if !ok {
		t.Fatal("Empty roles should be assigned")
	}
	if len(roles) != 0 {
		t.Errorf("Role count = %d, want 0", len(roles))
	}
}

// TestAssignRoles_Update verifies updating roles for existing subject
func TestAssignRoles_Update(t *testing.T) {
	ma := NewMemoryAuthorizer()

	// Initial assignment
	ma.AssignRoles("user:789", "viewer")

	// Update roles
	ma.AssignRoles("user:789", "admin", "superuser")

	// Verify update
	roles := ma.roles["user:789"]
	if len(roles) != 2 {
		t.Errorf("Role count after update = %d, want 2", len(roles))
	}
	if roles[0] != "admin" || roles[1] != "superuser" {
		t.Errorf("Updated roles = %v, want [admin superuser]", roles)
	}
}

// TestGetPermissions verifies permission retrieval for subject
func TestGetPermissions(t *testing.T) {
	ctx := context.Background()
	ma := NewMemoryAuthorizer()

	// Add policies
	ma.policies = []Policy{
		{
			ID:       "policy1",
			Subject:  "user:123",
			Resource: "document:*",
			Actions:  []string{"read", "write"},
			Effect:   Allow,
		},
		{
			ID:       "policy2",
			Subject:  "user:123",
			Resource: "file:*",
			Actions:  []string{"delete"},
			Effect:   Allow,
		},
		{
			ID:       "policy3",
			Subject:  "user:456", // Different subject
			Resource: "document:*",
			Actions:  []string{"admin"},
			Effect:   Allow,
		},
	}

	// Get permissions for user:123
	perms, err := ma.GetPermissions(ctx, "user:123")
	if err != nil {
		t.Fatalf("GetPermissions failed: %v", err)
	}

	// Should have 2 permissions (policy1 and policy2)
	if len(perms) != 2 {
		t.Errorf("Permission count = %d, want 2", len(perms))
	}

	// Verify permissions content
	var docPerm, filePerm *Permission
	for i := range perms {
		if perms[i].Resource == "document:*" {
			docPerm = &perms[i]
		} else if perms[i].Resource == "file:*" {
			filePerm = &perms[i]
		}
	}

	if docPerm == nil {
		t.Error("document:* permission not found")
	} else {
		if !docPerm.Granted {
			t.Error("document:* permission should be granted")
		}
		if len(docPerm.Actions) != 2 {
			t.Errorf("document:* action count = %d, want 2", len(docPerm.Actions))
		}
	}

	if filePerm == nil {
		t.Error("file:* permission not found")
	} else {
		if !filePerm.Granted {
			t.Error("file:* permission should be granted")
		}
		if len(filePerm.Actions) != 1 {
			t.Errorf("file:* action count = %d, want 1", len(filePerm.Actions))
		}
	}
}

// TestGetPermissions_NoPermissions verifies empty result for subject with no policies
func TestGetPermissions_NoPermissions(t *testing.T) {
	ctx := context.Background()
	ma := NewMemoryAuthorizer()

	ma.policies = []Policy{
		{
			ID:       "policy1",
			Subject:  "user:123",
			Resource: "document:*",
			Actions:  []string{"read"},
			Effect:   Allow,
		},
	}

	// Get permissions for different subject
	perms, err := ma.GetPermissions(ctx, "user:999")
	if err != nil {
		t.Fatalf("GetPermissions failed: %v", err)
	}

	if len(perms) != 0 {
		t.Errorf("Permission count = %d, want 0 for subject with no policies", len(perms))
	}
}

// Mock implementations for testing

type MockObligationExecutor struct{}

func (m *MockObligationExecutor) Execute(ob Obligation, ctx map[string]interface{}) error {
	return nil
}

func (m *MockObligationExecutor) PersistAudit(ob Obligation, ctx map[string]interface{}, result error) error {
	return nil
}

type MockMetricsProvider struct {
	ObligationsExecuted          uint64
	ObligationsFailed            uint64
	MandatoryObligationFailures  uint64
	ObligationLatencies          []time.Duration
}

func (m *MockMetricsProvider) IncObligationsExecuted() {
	m.ObligationsExecuted++
}

func (m *MockMetricsProvider) IncObligationsFailed() {
	m.ObligationsFailed++
}

func (m *MockMetricsProvider) IncMandatoryObligationFailures() {
	m.MandatoryObligationFailures++
}

func (m *MockMetricsProvider) ObserveObligationLatency(d time.Duration) {
	m.ObligationLatencies = append(m.ObligationLatencies, d)
}

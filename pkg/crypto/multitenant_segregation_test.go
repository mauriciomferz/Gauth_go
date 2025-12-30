package crypto

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestMultiTenantKeySegregation validates that keys are properly segregated per tenant
func TestMultiTenantKeySegregation(t *testing.T) {
	ctx := context.Background()

	// Create temporary directories for different tenant stores
	tempDir := filepath.Join(os.TempDir(), "AGENTAUTH-test-multitenant-seg")
	defer os.RemoveAll(tempDir)

	tenant1Store, err := NewFileKeyStore(filepath.Join(tempDir, "tenant1"), 24*time.Hour)
	if err != nil {
		t.Fatalf("Failed to create tenant1 store: %v", err)
	}
	tenant2Store, err := NewFileKeyStore(filepath.Join(tempDir, "tenant2"), 24*time.Hour)
	if err != nil {
		t.Fatalf("Failed to create tenant2 store: %v", err)
	}
	tenant3Store, err := NewFileKeyStore(filepath.Join(tempDir, "tenant3"), 24*time.Hour)
	if err != nil {
		t.Fatalf("Failed to create tenant3 store: %v", err)
	}

	defaultPolicy := &RotationPolicy{
		Enabled:  false,
		Interval: time.Hour,
		Backend:  "file",
	}

	manager := NewMultiTenantKeyManager(tenant1Store, defaultPolicy)

	// Register tenants with different key stores
	tenants := []struct {
		name  string
		store KeyStore
	}{
		{"tenant-alpha", tenant1Store},
		{"tenant-beta", tenant2Store},
		{"tenant-gamma", tenant3Store},
	}

	for _, tenant := range tenants {
		err := manager.RegisterTenant(tenant.name, tenant.store, defaultPolicy)
		if err != nil {
			t.Fatalf("Failed to register tenant %s: %v", tenant.name, err)
		}
	}

	// Generate keys for each tenant
	keys := make(map[string]string)
	for _, tenant := range tenants {
		keyID, err := tenant.store.Generate(ctx, tenant.name)
		if err != nil {
			t.Fatalf("Failed to generate key for %s: %v", tenant.name, err)
		}
		keys[tenant.name] = keyID

		// Activate the key
		err = tenant.store.Activate(ctx, tenant.name, keyID)
		if err != nil {
			t.Fatalf("Failed to activate key for %s: %v", tenant.name, err)
		}
	}

	// Verify key segregation: each tenant's store should only have their own keys
	for _, tenant := range tenants {
		t.Run(fmt.Sprintf("SegregationFor_%s", tenant.name), func(t *testing.T) {
			// Get all keys for this tenant
			allKeys, err := tenant.store.ListKeys(ctx, tenant.name)
			if err != nil {
				t.Fatalf("Failed to list keys: %v", err)
			}

			// Should have exactly 1 key
			if len(allKeys) != 1 {
				t.Errorf("Expected 1 key for %s, got %d", tenant.name, len(allKeys))
			}

			// Verify it's the correct key
			if allKeys[0].ID != keys[tenant.name] {
				t.Errorf("Key mismatch for %s: expected %s, got %s",
					tenant.name, keys[tenant.name], allKeys[0].ID)
			}

			// Verify we cannot access other tenants' keys
			for _, otherTenant := range tenants {
				if otherTenant.name == tenant.name {
					continue
				}

				// Try to get another tenant's key from this tenant's store
				_, err := tenant.store.GetKey(ctx, otherTenant.name, keys[otherTenant.name])
				if err == nil {
					t.Errorf("Security violation: tenant %s can access %s's key",
						tenant.name, otherTenant.name)
				}
			}
		})
	}
}

// TestMultiTenantDifferentBackends validates multi-tenant setup with mixed backend types
func TestMultiTenantDifferentBackends(t *testing.T) {
	ctx := context.Background()

	// Create temporary directories for different backends
	tempDir := filepath.Join(os.TempDir(), "AGENTAUTH-test-backends")
	defer os.RemoveAll(tempDir)

	// Create different file-based stores to simulate different backends
	devStore, err := NewFileKeyStore(filepath.Join(tempDir, "dev"), 24*time.Hour)
	if err != nil {
		t.Fatalf("Failed to create dev store: %v", err)
	}

	stagingStore, err := NewFileKeyStore(filepath.Join(tempDir, "staging"), 48*time.Hour)
	if err != nil {
		t.Fatalf("Failed to create staging store: %v", err)
	}

	prodStore, err := NewFileKeyStore(filepath.Join(tempDir, "production"), 168*time.Hour)
	if err != nil {
		t.Fatalf("Failed to create production store: %v", err)
	}

	defaultPolicy := &RotationPolicy{
		Enabled:  false,
		Interval: time.Hour,
		Backend:  "file",
	}

	manager := NewMultiTenantKeyManager(devStore, defaultPolicy)

	// Register tenants with different backends (simulated via different stores and TTLs)
	tenantBackends := []struct {
		tenant  string
		store   KeyStore
		backend string
		ttl     time.Duration
	}{
		{"tenant-dev", devStore, "file-dev", 24 * time.Hour},
		{"tenant-staging", stagingStore, "file-staging", 48 * time.Hour},
		{"tenant-production", prodStore, "file-production", 168 * time.Hour},
	}

	for _, tb := range tenantBackends {
		policy := &RotationPolicy{
			Enabled:  false,
			Interval: time.Hour,
			Backend:  tb.backend,
		}
		err := manager.RegisterTenant(tb.tenant, tb.store, policy)
		if err != nil {
			t.Fatalf("Failed to register %s with %s backend: %v", tb.tenant, tb.backend, err)
		}
	}

	// Generate and activate keys for each tenant
	for _, tb := range tenantBackends {
		t.Run(fmt.Sprintf("Backend_%s", tb.backend), func(t *testing.T) {
			keyID, err := tb.store.Generate(ctx, tb.tenant)
			if err != nil {
				t.Fatalf("Key generation failed for %s (%s): %v", tb.tenant, tb.backend, err)
			}

			err = tb.store.Activate(ctx, tb.tenant, keyID)
			if err != nil {
				t.Fatalf("Key activation failed for %s (%s): %v", tb.tenant, tb.backend, err)
			}

			// Verify key retrieval
			activeKey, err := tb.store.GetActive(ctx, tb.tenant)
			if err != nil {
				t.Fatalf("Failed to get active key for %s (%s): %v", tb.tenant, tb.backend, err)
			}

			if activeKey.ID != keyID {
				t.Errorf("Active key mismatch for %s (%s): expected %s, got %s",
					tb.tenant, tb.backend, keyID, activeKey.ID)
			}

			// Verify backend type
			retrievedPolicy := manager.GetRotationPolicy(tb.tenant)
			if retrievedPolicy.Backend != tb.backend {
				t.Errorf("Backend mismatch for %s: expected %s, got %s",
					tb.tenant, tb.backend, retrievedPolicy.Backend)
			}
		})
	}

	// Verify cross-tenant isolation
	for i, tb1 := range tenantBackends {
		for j, tb2 := range tenantBackends {
			if i >= j {
				continue
			}

			t.Run(fmt.Sprintf("Isolation_%s_vs_%s", tb1.backend, tb2.backend), func(t *testing.T) {
				// Get keys from tenant 1's store
				keys1, err := tb1.store.ListKeys(ctx, tb1.tenant)
				if err != nil {
					t.Fatalf("Failed to list keys for %s: %v", tb1.tenant, err)
				}

				// Try to access tenant 1's keys from tenant 2's store
				for _, key1 := range keys1 {
					_, err := tb2.store.GetKey(ctx, tb1.tenant, key1.ID)
					if err == nil && tb1.store != tb2.store {
						t.Errorf("Isolation breach: %s (%s) can access %s (%s) keys",
							tb2.tenant, tb2.backend, tb1.tenant, tb1.backend)
					}
				}
			})
		}
	}
}

// TestMultiTenantRotationSegregation validates that rotation events are tenant-specific
func TestMultiTenantRotationSegregation(t *testing.T) {
	ctx := context.Background()

	// Track rotation events per tenant
	rotationEvents := make(map[string][]*RotationEvent)

	// Create temporary directory
	tempDir := filepath.Join(os.TempDir(), "AGENTAUTH-test-rotation-seg")
	defer os.RemoveAll(tempDir)

	sharedStore, err := NewFileKeyStore(tempDir, 24*time.Hour)
	if err != nil {
		t.Fatalf("Failed to create shared store: %v", err)
	}

	defaultPolicy := &RotationPolicy{
		Enabled:  false,
		Interval: time.Hour,
		Backend:  "file",
	}

	manager := NewMultiTenantKeyManager(sharedStore, defaultPolicy)

	// Set up event callback to track rotations
	manager.SetEventCallback(func(event *RotationEvent) {
		rotationEvents[event.Tenant] = append(rotationEvents[event.Tenant], event)
	})

	// Register multiple tenants
	tenants := []string{"tenant-red", "tenant-blue", "tenant-green"}
	for _, tenant := range tenants {
		err := manager.RegisterTenant(tenant, sharedStore, defaultPolicy)
		if err != nil {
			t.Fatalf("Failed to register %s: %v", tenant, err)
		}

		// Generate initial key
		keyID, err := sharedStore.Generate(ctx, tenant)
		if err != nil {
			t.Fatalf("Failed to generate initial key for %s: %v", tenant, err)
		}
		err = sharedStore.Activate(ctx, tenant, keyID)
		if err != nil {
			t.Fatalf("Failed to activate initial key for %s: %v", tenant, err)
		}
	}

	// Trigger rotation for each tenant
	for _, tenant := range tenants {
		event, err := manager.RotateKey(ctx, tenant, "manual")
		if err != nil {
			t.Fatalf("Rotation failed for %s: %v", tenant, err)
		}

		if event.Tenant != tenant {
			t.Errorf("Event tenant mismatch: expected %s, got %s", tenant, event.Tenant)
		}

		if !event.Success {
			t.Errorf("Rotation unsuccessful for %s: %s", tenant, event.Error)
		}
	}

	// Verify each tenant has exactly 1 rotation event
	for _, tenant := range tenants {
		events := rotationEvents[tenant]
		if len(events) != 1 {
			t.Errorf("Expected 1 rotation event for %s, got %d", tenant, len(events))
		}

		if len(events) > 0 && events[0].Tenant != tenant {
			t.Errorf("Rotation event tenant mismatch for %s: got %s", tenant, events[0].Tenant)
		}
	}

	// Verify rotation event isolation (no cross-tenant contamination)
	for _, tenant := range tenants {
		events := rotationEvents[tenant]
		for _, event := range events {
			if event.Tenant != tenant {
				t.Errorf("Cross-tenant event leak: %s event in %s's events", event.Tenant, tenant)
			}
		}
	}
}

// TestMultiTenantPolicyIndependence validates that each tenant has independent rotation policies
func TestMultiTenantPolicyIndependence(t *testing.T) {
	// Create temporary directory
	tempDir := filepath.Join(os.TempDir(), "AGENTAUTH-test-policy")
	defer os.RemoveAll(tempDir)

	sharedStore, err := NewFileKeyStore(tempDir, 24*time.Hour)
	if err != nil {
		t.Fatalf("Failed to create shared store: %v", err)
	}

	defaultPolicy := &RotationPolicy{
		Enabled:  false,
		Interval: 24 * time.Hour,
		Backend:  "file",
	}

	manager := NewMultiTenantKeyManager(sharedStore, defaultPolicy)

	// Register tenants with different policies
	tenantPolicies := []struct {
		tenant   string
		interval time.Duration
		enabled  bool
	}{
		{"tenant-fast", 1 * time.Hour, true},
		{"tenant-medium", 12 * time.Hour, true},
		{"tenant-slow", 168 * time.Hour, true},
		{"tenant-manual", 0, false},
	}

	for _, tp := range tenantPolicies {
		policy := &RotationPolicy{
			Enabled:  tp.enabled,
			Interval: tp.interval,
			Backend:  "file",
		}
		err := manager.RegisterTenant(tp.tenant, sharedStore, policy)
		if err != nil {
			t.Fatalf("Failed to register %s: %v", tp.tenant, err)
		}
	}

	// Verify each tenant has independent policy
	for _, tp := range tenantPolicies {
		t.Run(tp.tenant, func(t *testing.T) {
			policy := manager.GetRotationPolicy(tp.tenant)
			if policy == nil {
				t.Fatal("Policy is nil")
			}

			if policy.Enabled != tp.enabled {
				t.Errorf("Enabled mismatch: expected %v, got %v", tp.enabled, policy.Enabled)
			}

			if policy.Interval != tp.interval {
				t.Errorf("Interval mismatch: expected %v, got %v", tp.interval, policy.Interval)
			}
		})
	}

	// Update one tenant's policy and verify others are unaffected
	newPolicy := &RotationPolicy{
		Enabled:  false,
		Interval: 999 * time.Hour,
		Backend:  "file",
	}

	updateErr := manager.UpdateRotationPolicy("tenant-fast", newPolicy)
	if updateErr != nil {
		t.Fatalf("Failed to update policy: %v", updateErr)
	}

	// Verify updated policy
	updatedPolicy := manager.GetRotationPolicy("tenant-fast")
	if updatedPolicy.Interval != 999*time.Hour {
		t.Errorf("Policy update failed: expected 999h, got %v", updatedPolicy.Interval)
	}

	// Verify other tenants' policies unchanged
	for _, tp := range tenantPolicies {
		if tp.tenant == "tenant-fast" {
			continue
		}

		policy := manager.GetRotationPolicy(tp.tenant)
		if policy.Interval != tp.interval {
			t.Errorf("Policy leak: %s interval changed from %v to %v",
				tp.tenant, tp.interval, policy.Interval)
		}
	}
}

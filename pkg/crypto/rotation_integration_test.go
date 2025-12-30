package crypto

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

//nolint:gocyclo // Integration test covering multiple rotation scenarios
func TestKeyRotationSystemIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode - test has timing issues with goroutines")
	}

	// Create temporary directory for test keys
	tempDir := filepath.Join(os.TempDir(), "AGENTAUTH-test-keys")
	defer os.RemoveAll(tempDir)

	// Create file-based key store
	store, err := NewFileKeyStore(tempDir, 24*time.Hour)
	if err != nil {
		t.Fatalf("Failed to create file key store: %v", err)
	}

	// Test 1: Basic key store operations
	t.Run("BasicKeyStoreOperations", func(t *testing.T) {
		ctx := context.Background()
		tenant := "test-tenant"

		// Generate a key
		keyID, err := store.Generate(ctx, tenant)
		if err != nil {
			t.Fatalf("Failed to generate key: %v", err)
		}

		// Activate the key
		err = store.Activate(ctx, tenant, keyID)
		if err != nil {
			t.Fatalf("Failed to activate key: %v", err)
		}

		// Get active key
		activeKey, err := store.GetActive(ctx, tenant)
		if err != nil {
			t.Fatalf("Failed to get active key: %v", err)
		}

		if activeKey.ID != keyID {
			t.Errorf("Expected active key ID %s, got %s", keyID, activeKey.ID)
		}

		// List keys
		keys, err := store.ListKeys(ctx, tenant)
		if err != nil {
			t.Fatalf("Failed to list keys: %v", err)
		}

		if len(keys) != 1 {
			t.Errorf("Expected 1 key, got %d", len(keys))
		}

		// Archive the key
		err = store.Archive(ctx, tenant, keyID)
		if err != nil {
			t.Fatalf("Failed to archive key: %v", err)
		}

		// Verify no active key after archiving
		_, err = store.GetActive(ctx, tenant)
		if err == nil {
			t.Error("Expected error when getting active key after archiving, got nil")
		}
	})

	// Test 2: Multi-tenant manager operations
	t.Run("MultiTenantManagerOperations", func(t *testing.T) {
		defaultPolicy := &RotationPolicy{
			Enabled:     true,
			Interval:    time.Hour,
			Jitter:      10 * time.Minute,
			MaxKeyAge:   24 * time.Hour,
			GracePeriod: time.Hour,
			Backend:     "file",
		}

		manager := NewMultiTenantKeyManager(store, defaultPolicy)

		// Initialize manager internal maps
		manager.stores = make(map[string]KeyStore)
		manager.policies = make(map[string]*RotationPolicy)
		manager.schedulers = make(map[string]*TenantScheduler)
		manager.statuses = make(map[string]*RotationStatus)
		manager.healthy = true

		// Register a tenant
		tenant := "multi-tenant-test"
		err := manager.RegisterTenant(tenant, store, defaultPolicy)
		if err != nil {
			t.Fatalf("Failed to register tenant: %v", err)
		}

		// Check tenant registration
		tenants := manager.GetRegisteredTenants()
		if len(tenants) != 1 || tenants[0] != tenant {
			t.Errorf("Expected tenant %s to be registered, got %v", tenant, tenants)
		}

		// Get tenant policy
		policy := manager.GetRotationPolicy(tenant)
		if policy == nil {
			t.Fatal("Expected policy for tenant, got nil")
		}

		if policy.Backend != "file" {
			t.Errorf("Expected backend 'file', got %s", policy.Backend)
		} // Update policy
		newPolicy := &RotationPolicy{
			Enabled:  true,
			Interval: 2 * time.Hour,
			Backend:  "file",
		}

		err = manager.UpdateRotationPolicy(tenant, newPolicy)
		if err != nil {
			t.Fatalf("Failed to update policy: %v", err)
		}

		// Verify policy update
		updatedPolicy := manager.GetRotationPolicy(tenant)
		if updatedPolicy.Interval != 2*time.Hour {
			t.Errorf("Expected interval 2h, got %v", updatedPolicy.Interval)
		}

		// Test manual rotation trigger
		err = manager.TriggerRotation(tenant, false, "test rotation")
		if err != nil {
			t.Fatalf("Failed to trigger rotation: %v", err)
		}

		// Give rotation some time to complete
		time.Sleep(100 * time.Millisecond)

		// Check rotation status
		status := manager.GetRotationStatus(tenant)
		if status == nil {
			t.Error("Expected rotation status, got nil")
		}

		// Check health
		if !manager.IsHealthy() {
			t.Error("Expected manager to be healthy")
		}
	})

	// Test 3: Key rotation API operations
	t.Run("KeyRotationAPI", func(t *testing.T) {
		defaultPolicy := &RotationPolicy{
			Enabled:  true,
			Interval: time.Hour,
			Backend:  "file",
		}

		manager := NewMultiTenantKeyManager(store, defaultPolicy)

		// Initialize manager
		manager.stores = make(map[string]KeyStore)
		manager.policies = make(map[string]*RotationPolicy)
		manager.schedulers = make(map[string]*TenantScheduler)
		manager.statuses = make(map[string]*RotationStatus)
		manager.healthy = true
		manager.keyStore = store

		// Register test tenant
		tenant := "api-test-tenant"
		err := manager.RegisterTenant(tenant, store, defaultPolicy)
		if err != nil {
			t.Fatalf("Failed to register tenant: %v", err)
		}

		// Create API
		api := NewKeyRotationAPI(manager)

		// Test policy conversion
		updateReq := UpdatePolicyRequest{
			Enabled:   true,
			Interval:  "2h",
			Jitter:    "15m",
			MaxKeyAge: "48h",
			Backend:   "file",
		}

		policy, err := api.requestToPolicy(updateReq)
		if err != nil {
			t.Fatalf("Failed to convert request to policy: %v", err)
		}

		if policy.Interval != 2*time.Hour {
			t.Errorf("Expected interval 2h, got %v", policy.Interval)
		}

		if policy.Jitter != 15*time.Minute {
			t.Errorf("Expected jitter 15m, got %v", policy.Jitter)
		}

		if policy.MaxKeyAge != 48*time.Hour {
			t.Errorf("Expected max key age 48h, got %v", policy.MaxKeyAge)
		}
	})

	// Test 4: Key store health checks
	t.Run("HealthChecks", func(t *testing.T) {
		ctx := context.Background()

		// Test file store health
		err := store.Health(ctx)
		if err != nil {
			t.Errorf("File store health check failed: %v", err)
		}

		// Test with inaccessible directory
		_, err = NewFileKeyStore("/invalid/path/that/does/not/exist", time.Hour)
		if err == nil {
			t.Error("Expected error creating store with invalid path, got nil")
		}
	})
}

func TestVaultKeyStoreInterface(t *testing.T) {
	// Test that VaultKeyStore implements KeyStore interface
	var _ KeyStore = (*VaultKeyStore)(nil)
}

func TestKMSKeyStoreInterface(t *testing.T) {
	// Test that KMSKeyStore implements KeyStore interface
	var _ KeyStore = (*KMSKeyStore)(nil)
}

func TestFileKeyStoreInterface(t *testing.T) {
	// Test that FileKeyStore implements KeyStore interface
	var _ KeyStore = (*FileKeyStore)(nil)
}

// Benchmark key generation performance
func BenchmarkKeyGeneration(b *testing.B) {
	tempDir := filepath.Join(os.TempDir(), "AGENTAUTH-bench-keys")
	defer os.RemoveAll(tempDir)

	store, err := NewFileKeyStore(tempDir, 24*time.Hour)
	if err != nil {
		b.Fatalf("Failed to create file key store: %v", err)
	}

	ctx := context.Background()
	tenant := "bench-tenant"

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := store.Generate(ctx, tenant)
		if err != nil {
			b.Fatalf("Failed to generate key: %v", err)
		}
	}
}

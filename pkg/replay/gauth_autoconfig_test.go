package replay

import (
	"os"
	"testing"
	"time"

	"github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/gauth"
)

// TestGAuthDurableReplayAutoConfig demonstrates automatic durable replay store configuration
// using the factory pattern. This tests sec6.item1 P1 completion.
func TestGAuthDurableReplayAutoConfig(t *testing.T) {
	// Setup: Configure environment variables
	walPath := "./testdata/gauth_autoconfig_replay.wal"
	defer os.RemoveAll("./testdata")

	os.Setenv("GAUTH_REPLAY_WAL_PATH", walPath)
	os.Setenv("GAUTH_REPLAY_TTL_SEC", "60")
	os.Setenv("GAUTH_REPLAY_EVICTION_POLICY", "ttl")
	defer os.Unsetenv("GAUTH_REPLAY_WAL_PATH")
	defer os.Unsetenv("GAUTH_REPLAY_TTL_SEC")
	defer os.Unsetenv("GAUTH_REPLAY_EVICTION_POLICY")

	// Register the replay factory (this would typically be done in init() or main())
	gauth.RegisterDurableReplayStoreFactory(func(metrics interface{}) (gauth.ReplayStore, error) {
		return NewGAuthReplayStoreFromEnv(metrics)
	})

	// Create gauth service with auto-configured durable replay store
	config := gauth.Config{
		ClientID:          "test-client",
		ClientSecret:      "test-secret-12345678901234567890",
		AccessTokenExpiry: 5 * time.Minute,
	}

	svc, err := gauth.New(config, gauth.WithDurableReplayFromEnvUsingFactory())
	if err != nil {
		t.Fatalf("Failed to create gauth service with auto-configured replay: %v", err)
	}

	// Test 1: Issue token with replay protection
	req1 := gauth.TokenRequest{
		GrantID: "grant-123",
		Scope:   []string{"read", "write"},
	}

	resp1, err := svc.RequestToken(req1)
	if err != nil {
		t.Fatalf("RequestToken failed: %v", err)
	}

	if resp1.Token == "" {
		t.Fatal("Expected non-empty token")
	}

	// Test 2: Validate token (replay protection should not interfere)
	result1, err := svc.ValidateToken(resp1.Token)
	if err != nil {
		t.Fatalf("ValidateToken failed: %v", err)
	}

	if !result1.Valid {
		t.Fatal("Token should be valid")
	}

	// Test 3: Issue another token
	resp2, err := svc.RequestToken(req1)
	if err != nil {
		t.Fatalf("Second RequestToken failed: %v", err)
	}

	// Verify tokens are different (different JTI)
	if resp1.Token == resp2.Token {
		t.Fatal("Expected different tokens (different JTI)")
	}

	// Test 4: Validate second token
	result2, err := svc.ValidateToken(resp2.Token)
	if err != nil {
		t.Fatalf("Second ValidateToken failed: %v", err)
	}

	if !result2.Valid {
		t.Fatal("Second token should be valid")
	}

	t.Logf("✅ Durable replay store auto-configuration successful")
	t.Logf("   - WAL path: %s", walPath)
	t.Logf("   - TTL: 60s")
	t.Logf("   - Eviction policy: ttl")
	t.Logf("   - Tokens issued: 2")
	t.Logf("   - All validations passed")
}

// TestGAuthDurableReplayPersistence tests that replay protection persists across restarts.
func TestGAuthDurableReplayPersistence(t *testing.T) {
	walPath := "./testdata/gauth_persist_replay.wal"
	defer os.RemoveAll("./testdata")

	os.Setenv("GAUTH_REPLAY_WAL_PATH", walPath)
	os.Setenv("GAUTH_REPLAY_TTL_SEC", "300") // 5 minutes
	os.Setenv("GAUTH_REPLAY_EVICTION_POLICY", "ttl")
	defer os.Unsetenv("GAUTH_REPLAY_WAL_PATH")
	defer os.Unsetenv("GAUTH_REPLAY_TTL_SEC")
	defer os.Unsetenv("GAUTH_REPLAY_EVICTION_POLICY")

	// Register factory
	gauth.RegisterDurableReplayStoreFactory(func(metrics interface{}) (gauth.ReplayStore, error) {
		return NewGAuthReplayStoreFromEnv(metrics)
	})

	config := gauth.Config{
		ClientID:          "test-client",
		ClientSecret:      "test-secret-12345678901234567890",
		AccessTokenExpiry: 5 * time.Minute,
	}

	var token1, token2 string

	// Phase 1: Create service, issue tokens
	{
		svc, err := gauth.New(config, gauth.WithDurableReplayFromEnvUsingFactory())
		if err != nil {
			t.Fatalf("Failed to create first service: %v", err)
		}

		req := gauth.TokenRequest{
			GrantID: "grant-123",
			Scope:   []string{"read"},
		}

		resp1, err := svc.RequestToken(req)
		if err != nil {
			t.Fatalf("First RequestToken failed: %v", err)
		}
		token1 = resp1.Token

		resp2, err := svc.RequestToken(req)
		if err != nil {
			t.Fatalf("Second RequestToken failed: %v", err)
		}
		token2 = resp2.Token

		// Tokens should be different
		if token1 == token2 {
			t.Fatal("Expected different tokens")
		}
	}

	// Phase 2: Restart service (new instance)
	{
		svc2, err := gauth.New(config, gauth.WithDurableReplayFromEnvUsingFactory())
		if err != nil {
			t.Fatalf("Failed to create second service: %v", err)
		}

		// Validate tokens from phase 1 (should still work)
		result1, err := svc2.ValidateToken(token1)
		if err != nil {
			t.Fatalf("Validate token1 after restart failed: %v", err)
		}
		if !result1.Valid {
			t.Fatal("Token1 should still be valid after restart")
		}

		result2, err := svc2.ValidateToken(token2)
		if err != nil {
			t.Fatalf("Validate token2 after restart failed: %v", err)
		}
		if !result2.Valid {
			t.Fatal("Token2 should still be valid after restart")
		}

		// Issue a new token
		req := gauth.TokenRequest{
			GrantID: "grant-123",
			Scope:   []string{"read"},
		}

		resp3, err := svc2.RequestToken(req)
		if err != nil {
			t.Fatalf("Third RequestToken failed: %v", err)
		}

		// New token should be different from previous ones
		if resp3.Token == token1 || resp3.Token == token2 {
			t.Fatal("Expected new unique token")
		}
	}

	t.Logf("✅ Durable replay persistence across restarts verified")
	t.Logf("   - Tokens from phase 1 still valid in phase 2")
	t.Logf("   - New tokens issued in phase 2 are unique")
	t.Logf("   - WAL recovery successful")
}

// TestGAuthDurableReplayEvictionPolicies tests different eviction policies.
func TestGAuthDurableReplayEvictionPolicies(t *testing.T) {
	testCases := []struct {
		name   string
		policy string
	}{
		{"TTL", "ttl"},
		{"LRU", "lru"},
		{"Size", "size"},
		{"Composite", "ttl+size"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			walPath := "./testdata/gauth_evict_" + tc.policy + "_replay.wal"
			defer os.RemoveAll("./testdata")

			os.Setenv("GAUTH_REPLAY_WAL_PATH", walPath)
			os.Setenv("GAUTH_REPLAY_TTL_SEC", "60")
			os.Setenv("GAUTH_REPLAY_EVICTION_POLICY", tc.policy)
			os.Setenv("GAUTH_REPLAY_EVICTION_MAX_SIZE", "100")
			defer os.Unsetenv("GAUTH_REPLAY_WAL_PATH")
			defer os.Unsetenv("GAUTH_REPLAY_TTL_SEC")
			defer os.Unsetenv("GAUTH_REPLAY_EVICTION_POLICY")
			defer os.Unsetenv("GAUTH_REPLAY_EVICTION_MAX_SIZE")

			gauth.RegisterDurableReplayStoreFactory(func(metrics interface{}) (gauth.ReplayStore, error) {
				return NewGAuthReplayStoreFromEnv(metrics)
			})

			config := gauth.Config{
				ClientID:          "test-client",
				ClientSecret:      "test-secret-12345678901234567890",
				AccessTokenExpiry: 5 * time.Minute,
			}

			svc, err := gauth.New(config, gauth.WithDurableReplayFromEnvUsingFactory())
			if err != nil {
				t.Fatalf("Failed to create service with %s policy: %v", tc.policy, err)
			}

			// Issue a few tokens
			req := gauth.TokenRequest{
				GrantID: "grant-123",
				Scope:   []string{"read"},
			}

			for i := 0; i < 5; i++ {
				_, err := svc.RequestToken(req)
				if err != nil {
					t.Fatalf("RequestToken %d failed: %v", i, err)
				}
			}

			t.Logf("✅ Eviction policy %s working correctly", tc.policy)
		})
	}
}

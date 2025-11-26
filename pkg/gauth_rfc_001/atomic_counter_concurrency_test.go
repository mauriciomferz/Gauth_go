package gauth_rfc_001

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
)

// TestAtomicCounter_ConcurrentCheckAndIncrement validates that the Redis Lua atomic counter
// prevents TOCTOU race conditions under high concurrency.
//
// Security Test: Critical Vulnerability - Race Condition in Constraint Enforcement (TOCTOU)
//
// Attack Scenario:
//   - Quota limit: 100.00
//   - Attacker launches 20 goroutines simultaneously
//   - Each goroutine attempts to consume 100.00 (total 2000.00)
//   - Without atomic operations: Multiple goroutines read current=0, all pass validation
//   - With atomic operations: Only 1 succeeds, remaining 19 fail
//
// Expected Behavior (SECURE):
//   - Total successful operations: 1
//   - Total consumed: 100.00
//   - Rejected operations: 19
//
// Failure Mode (VULNERABLE):
//   - Total successful operations: >1 (often 10-20)
//   - Total consumed: >100.00 (quota breach)
func TestAtomicCounter_ConcurrentCheckAndIncrement(t *testing.T) {
	// Setup in-memory Redis using miniredis
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("Failed to start miniredis: %v", err)
	}
	defer mr.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	defer redisClient.Close()

	// Create atomic counter store
	store, err := NewAtomicCounterStore(redisClient, "gauth:test"); if err != nil { t.Fatalf("Failed to create store: %v", err) }

	ctx := context.Background()
	key := "test:quota:concurrent"
	limit := 100.0
	increment := 100.0
	ttl := 1 * time.Hour

	// Launch 20 concurrent goroutines all trying to consume 100.0
	numGoroutines := 20
	var successCount atomic.Int32
	var wg sync.WaitGroup

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			// Each goroutine tries to consume full quota
			allowed, err := store.CheckAndIncrement(ctx, key, increment, limit, ttl)
			if err != nil {
				t.Errorf("Goroutine %d: unexpected error: %v", id, err)
				return
			}

			if allowed {
				successCount.Add(1)
				t.Logf("Goroutine %d: SUCCESS - consumed %.2f", id, increment)
			} else {
				t.Logf("Goroutine %d: REJECTED - would exceed limit", id)
			}
		}(i)
	}

	wg.Wait()

	// Verify only 1 goroutine succeeded (atomic enforcement)
	actualSuccesses := successCount.Load()
	if actualSuccesses != 1 {
		t.Errorf("SECURITY VULNERABILITY: Expected 1 success (atomic), got %d (race condition present)", actualSuccesses)
	} else {
		t.Logf("✅ SECURE: Atomic enforcement prevented %d quota bypass attempts", numGoroutines-1)
	}

	// Verify final counter value
	finalValue, err := store.GetValue(ctx, key)
	if err != nil {
		t.Fatalf("Failed to get final value: %v", err)
	}

	expectedValue := 100.0
	if finalValue != expectedValue {
		t.Errorf("Expected final value %.2f, got %.2f", expectedValue, finalValue)
	}
}

// TestAtomicCounter_PartialFillScenario validates that partial quota consumption works correctly
// under concurrent access.
//
// Scenario:
//   - Quota limit: 100.00
//   - 10 goroutines each try to consume 15.00 (total 150.00)
//   - Expected: 6 succeed (6 × 15 = 90), 4 fail (would reach 105, 120, 135, 150)
func TestAtomicCounter_PartialFillScenario(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("Failed to start miniredis: %v", err)
	}
	defer mr.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	defer redisClient.Close()

	store, err := NewAtomicCounterStore(redisClient, "gauth:test"); if err != nil { t.Fatalf("Failed to create store: %v", err) }

	ctx := context.Background()
	key := "test:quota:partial"
	limit := 100.0
	increment := 15.0
	ttl := 1 * time.Hour

	numGoroutines := 10
	var successCount atomic.Int32
	var wg sync.WaitGroup

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			allowed, err := store.CheckAndIncrement(ctx, key, increment, limit, ttl)
			if err != nil {
				t.Errorf("Goroutine %d: unexpected error: %v", id, err)
				return
			}

			if allowed {
				successCount.Add(1)
			}
		}(i)
	}

	wg.Wait()

	actualSuccesses := successCount.Load()
	expectedSuccesses := int32(6) // 6 × 15 = 90 (under limit)

	if actualSuccesses != expectedSuccesses {
		t.Errorf("Expected %d successes (6×15=90), got %d", expectedSuccesses, actualSuccesses)
	}

	finalValue, err := store.GetValue(ctx, key)
	if err != nil {
		t.Fatalf("Failed to get final value: %v", err)
	}

	expectedValue := 90.0
	if finalValue != expectedValue {
		t.Errorf("Expected final value %.2f, got %.2f", expectedValue, finalValue)
	} else {
		t.Logf("✅ CORRECT: Partial fills work correctly (%.2f consumed, %.2f remaining)", finalValue, limit-finalValue)
	}
}

// TestAtomicCounter_ScriptReloadOnRedisRestart validates that the atomic counter store
// automatically reloads the Lua script after Redis restarts (NOSCRIPT error).
func TestAtomicCounter_ScriptReloadOnRedisRestart(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("Failed to start miniredis: %v", err)
	}
	defer mr.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	defer redisClient.Close()

	store, err := NewAtomicCounterStore(redisClient, "gauth:test"); if err != nil { t.Fatalf("Failed to create store: %v", err) }

	ctx := context.Background()
	key := "test:quota:reload"
	limit := 100.0
	increment := 50.0
	ttl := 1 * time.Hour

	// First operation - loads script
	allowed, err := store.CheckAndIncrement(ctx, key, increment, limit, ttl)
	if err != nil {
		t.Fatalf("First operation failed: %v", err)
	}
	if !allowed {
		t.Fatal("First operation should succeed")
	}

	// Simulate Redis restart by clearing all data (including cached scripts)
	mr.FlushAll()

	// Second operation - should detect NOSCRIPT and reload automatically
	allowed, err = store.CheckAndIncrement(ctx, key, increment, limit, ttl)
	if err != nil {
		t.Fatalf("Second operation failed after script reload: %v", err)
	}
	if !allowed {
		t.Fatal("Second operation should succeed (fresh counter after flush)")
	}

	t.Log("✅ RESILIENT: Automatic script reload works correctly")
}

// TestAtomicCounter_TTLExpiration validates that quota keys expire correctly after TTL.
func TestAtomicCounter_TTLExpiration(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("Failed to start miniredis: %v", err)
	}
	defer mr.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	defer redisClient.Close()

	store, err := NewAtomicCounterStore(redisClient, "gauth:test"); if err != nil { t.Fatalf("Failed to create store: %v", err) }

	ctx := context.Background()
	key := "test:quota:ttl"
	limit := 100.0
	increment := 100.0
	ttl := 2 * time.Second

	// Consume full quota
	allowed, err := store.CheckAndIncrement(ctx, key, increment, limit, ttl)
	if err != nil {
		t.Fatalf("Initial operation failed: %v", err)
	}
	if !allowed {
		t.Fatal("Initial operation should succeed")
	}

	// Try immediately - should fail (quota exhausted)
	allowed, err = store.CheckAndIncrement(ctx, key, increment, limit, ttl)
	if err != nil {
		t.Fatalf("Second operation failed: %v", err)
	}
	if allowed {
		t.Fatal("Second operation should fail (quota exhausted)")
	}

	// Fast-forward time in miniredis
	mr.FastForward(3 * time.Second)

	// Try after TTL - should succeed (key expired, quota reset)
	allowed, err = store.CheckAndIncrement(ctx, key, increment, limit, ttl)
	if err != nil {
		t.Fatalf("Operation after TTL failed: %v", err)
	}
	if !allowed {
		t.Fatal("Operation after TTL should succeed (quota reset)")
	}

	t.Log("✅ CLEANUP: TTL-based expiration works correctly")
}

// BenchmarkAtomicCounter_CheckAndIncrement measures performance of atomic quota enforcement.
func BenchmarkAtomicCounter_CheckAndIncrement(b *testing.B) {
	mr, err := miniredis.Run()
	if err != nil {
		b.Fatalf("Failed to start miniredis: %v", err)
	}
	defer mr.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	defer redisClient.Close()

	store, err := NewAtomicCounterStore(redisClient, "gauth:test")
	if err != nil {
		b.Fatalf("Failed to create store: %v", err)
	}

	ctx := context.Background()
	limit := 1000000.0 // High limit to avoid rejection
	increment := 1.0
	ttl := 1 * time.Hour

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := "test:quota:bench"
		_, _ = store.CheckAndIncrement(ctx, key, increment, limit, ttl)
	}
}

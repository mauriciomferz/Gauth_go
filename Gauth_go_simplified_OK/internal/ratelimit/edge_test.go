package ratelimit_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/Gimel-Foundation/gauth/internal/ratelimit"
)

func TestEdgeCases(t *testing.T) {
	algorithms := map[string]ratelimit.NewAlgorithm{
		"TokenBucket":   ratelimit.WrapTokenBucket,
		"SlidingWindow": ratelimit.WrapSlidingWindow,
	}

	for name, newAlgo := range algorithms {
		t.Run(name, func(t *testing.T) {
			testZeroConfig(t, newAlgo)
			testHighLoad(t, newAlgo)
			testMultipleClients(t, newAlgo)
			testLongWindow(t, newAlgo)
		})
	}
}

func testZeroConfig(t *testing.T, newAlgo ratelimit.NewAlgorithm) {
	t.Run("Zero Config Values", func(t *testing.T) {
		// Test with zero requests per second
		config := &ratelimit.Config{
			RequestsPerSecond: 0,
			BurstSize:         1,
			WindowSize:        1,
		}
		algo := newAlgo(config)
		ctx := context.Background()

		if err := algo.Allow(ctx, "test"); err == nil {
			t.Error("Should deny requests when RequestsPerSecond is 0")
		}

		// Test with zero window size
		config = &ratelimit.Config{
			RequestsPerSecond: 1,
			BurstSize:         1,
			WindowSize:        0,
		}
		algo = newAlgo(config)

		if err := algo.Allow(ctx, "test"); err == nil {
			t.Error("Should deny requests when WindowSize is 0")
		}
	})
}

func testHighLoad(t *testing.T, newAlgo ratelimit.NewAlgorithm) {
	t.Run("High Load", func(t *testing.T) {
		config := &ratelimit.Config{
			RequestsPerSecond: 1000,
			BurstSize:         2000,
			WindowSize:        1,
		}
		algo := newAlgo(config)
		ctx := context.Background()

		// Send burst of requests
		var wg sync.WaitGroup
		errors := make(chan error, 2500)

		for i := 0; i < 2500; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				if err := algo.Allow(ctx, "high-load"); err != nil && err != ratelimit.ErrRateLimitExceeded {
					errors <- err
				}
			}()
		}

		// Wait for all requests to complete
		wg.Wait()
		close(errors)

		// Check for unexpected errors
		for err := range errors {
			t.Errorf("Unexpected error under high load: %v", err)
		}

		// Verify quota after high load
		quota := algo.GetQuota("high-load")
		if quota.Remaining < 0 {
			t.Error("Remaining quota should not be negative after high load")
		}
	})
}

func testMultipleClients(t *testing.T, newAlgo ratelimit.NewAlgorithm) {
	t.Run("Multiple Clients", func(t *testing.T) {
		config := &ratelimit.Config{
			RequestsPerSecond: 5,
			BurstSize:         10,
			WindowSize:        1,
		}
		algo := newAlgo(config)
		ctx := context.Background()

		// Test multiple clients concurrently
		clients := []string{"client1", "client2", "client3", "client4", "client5"}
		var wg sync.WaitGroup
		results := make(map[string]int)
		var mu sync.Mutex

		// Each client makes requests
		for _, client := range clients {
			wg.Add(1)
			go func(clientID string) {
				defer wg.Done()
				successCount := 0
				for i := 0; i < 15; i++ {
					if err := algo.Allow(ctx, clientID); err == nil {
						successCount++
					}
					time.Sleep(time.Millisecond * 10)
				}
				mu.Lock()
				results[clientID] = successCount
				mu.Unlock()
			}(client)
		}

		wg.Wait()

		// Verify each client's limits were enforced independently
		for client, count := range results {
			if count > config.BurstSize {
				t.Errorf("Client %s exceeded burst limit: got %d requests, want <= %d",
					client, count, config.BurstSize)
			}
		}
	})
}

func testLongWindow(t *testing.T, newAlgo ratelimit.NewAlgorithm) {
	t.Run("Long Window", func(t *testing.T) {
		config := &ratelimit.Config{
			RequestsPerSecond: 2,
			BurstSize:         6, // Increased burst size
			WindowSize:        5, // 5-second window
		}
		algo := newAlgo(config)
		ctx := context.Background()

		// Make initial requests at a rate below the limit
		for i := 0; i < 3; i++ {
			if err := algo.Allow(ctx, "long-window"); err != nil {
				t.Errorf("Initial request %d should be allowed", i+1)
			}
			time.Sleep(time.Second)
		}

		// Try burst
		burstAllowed := 0
		for i := 0; i < config.BurstSize-2; i++ {
			if err := algo.Allow(ctx, "long-window"); err == nil {
				burstAllowed++
			}
		}

		if burstAllowed == 0 {
			t.Error("Should allow some burst requests")
		}

		// Wait for partial window
		time.Sleep(2 * time.Second)

		// Should allow requests at normal rate
		if err := algo.Allow(ctx, "long-window"); err != nil {
			t.Error("Should allow request after partial window")
		}
		time.Sleep(time.Second)
		if err := algo.Allow(ctx, "long-window"); err != nil {
			t.Error("Should allow second request after partial window")
		}

		// Verify quota
		quota := algo.GetQuota("long-window")
		if quota.Window.Requests <= 0 {
			t.Error("Should have some requests in window")
		}
		if quota.Window.Duration != time.Duration(config.WindowSize)*time.Second {
			t.Error("Window duration should match config")
		}
	})
}

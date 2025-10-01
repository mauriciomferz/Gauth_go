package ratelimit_test

import (
	"context"
	"testing"
	"time"

	"github.com/Gimel-Foundation/gauth/internal/ratelimit"
)

// TestRateLimit tests both TokenBucket and SlidingWindow algorithms
func TestRateLimit(t *testing.T) {
	algorithms := map[string]ratelimit.NewAlgorithm{
		"TokenBucket":   ratelimit.WrapTokenBucket,
		"SlidingWindow": ratelimit.WrapSlidingWindow,
	}

	for name, newAlgo := range algorithms {
		t.Run(name, func(t *testing.T) {
			testBasicRateLimiting(t, newAlgo)
			testBurstHandling(t, newAlgo)
			testWindowSliding(t, newAlgo)
			testQuotaInfo(t, newAlgo)
			testReset(t, newAlgo)
		})
	}
}

func testBasicRateLimiting(t *testing.T, newAlgo ratelimit.NewAlgorithm) {
	config := &ratelimit.Config{
		RequestsPerSecond: 2,
		BurstSize:         2, // Set burst size equal to requests per second
		WindowSize:        1,
	}
	algo := newAlgo(config)
	ctx := context.Background()

	t.Run("Basic Rate Limiting", func(t *testing.T) {
		// Use up all allowed requests
		for i := 0; i < config.BurstSize; i++ {
			if err := algo.Allow(ctx, "test"); err != nil {
				t.Errorf("Request %d should be allowed, got error: %v", i+1, err)
			}
		}

		// Additional request should be denied
		if err := algo.Allow(ctx, "test"); err != ratelimit.ErrRateLimitExceeded {
			t.Errorf("Expected rate limit exceeded error, got: %v", err)
		}

		// Wait a bit and try again
		time.Sleep(time.Duration(config.WindowSize) * time.Second)
		if err := algo.Allow(ctx, "test"); err != nil {
			t.Errorf("Request after window reset should be allowed, got: %v", err)
		}
	})
}

func testBurstHandling(t *testing.T, newAlgo ratelimit.NewAlgorithm) {
	config := &ratelimit.Config{
		RequestsPerSecond: 5,
		BurstSize:         10,
		WindowSize:        2,
	}
	algo := newAlgo(config)
	ctx := context.Background()

	t.Run("Burst Handling", func(t *testing.T) {
		// Try burst of requests
		successCount := 0
		for i := 0; i < config.BurstSize+5; i++ {
			err := algo.Allow(ctx, "burst-test")
			if err == nil {
				successCount++
			}
		}

		if successCount > config.BurstSize {
			t.Errorf("Allowed more requests (%d) than burst size (%d)",
				successCount, config.BurstSize)
		}
	})
}

func testWindowSliding(t *testing.T, newAlgo ratelimit.NewAlgorithm) {
	config := &ratelimit.Config{
		RequestsPerSecond: 2,
		BurstSize:         2, // Set burst size equal to rate
		WindowSize:        1,
	}
	algo := newAlgo(config)
	ctx := context.Background()

	t.Run("Window Sliding", func(t *testing.T) {
		// Use up allowed tokens
		for i := 0; i < config.BurstSize; i++ {
			if err := algo.Allow(ctx, "slide-test"); err != nil {
				t.Errorf("Request %d should be allowed, got: %v", i+1, err)
			}
		}

		// Should be denied when exceeding burst
		if err := algo.Allow(ctx, "slide-test"); err != ratelimit.ErrRateLimitExceeded {
			t.Error("Should be denied when exceeding burst")
		}

		// Wait for tokens to replenish
		time.Sleep(1100 * time.Millisecond)

		// Should allow requests after replenishment
		if err := algo.Allow(ctx, "slide-test"); err != nil {
			t.Error("Should allow request after replenishment")
		}
	})
}

func testQuotaInfo(t *testing.T, newAlgo ratelimit.NewAlgorithm) {
	config := &ratelimit.Config{
		RequestsPerSecond: 10,
		BurstSize:         20,
		WindowSize:        5,
	}
	algo := newAlgo(config)

	t.Run("Quota Information", func(t *testing.T) {
		// Check initial quota
		quota := algo.GetQuota("quota-test")
		if quota.Total <= 0 {
			t.Error("Total quota should be positive")
		}
		if quota.Window.Duration != time.Duration(config.WindowSize)*time.Second {
			t.Error("Window duration doesn't match config")
		}

		// Verify window information
		if quota.Window.Start.IsZero() {
			t.Error("Window start time should be set")
		}
		if quota.Window.Requests != 0 {
			t.Error("Initial window should have 0 requests")
		}
	})
}

func testReset(t *testing.T, newAlgo ratelimit.NewAlgorithm) {
	config := &ratelimit.Config{
		RequestsPerSecond: 1,
		BurstSize:         1,
		WindowSize:        1,
	}
	algo := newAlgo(config)
	ctx := context.Background()

	t.Run("Reset", func(t *testing.T) {
		// Use up the limit
		if err := algo.Allow(ctx, "reset-test"); err != nil {
			t.Error("First request should be allowed")
		}
		if err := algo.Allow(ctx, "reset-test"); err != ratelimit.ErrRateLimitExceeded {
			t.Error("Second request should be denied")
		}

		// Reset and try again
		algo.Reset("reset-test")
		if err := algo.Allow(ctx, "reset-test"); err != nil {
			t.Error("Request after reset should be allowed")
		}
	})
}

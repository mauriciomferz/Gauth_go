package oidc

import (
	"context"
	"sync"
	"testing"
	"time"
)

// TestTokenBucketLimiter_BasicAllow tests basic allow functionality.
func TestTokenBucketLimiter_BasicAllow(t *testing.T) {
	// 10 requests per second with burst of 10
	limiter := NewTokenBucketLimiter(10, 10, time.Second)
	defer limiter.Close()
	ctx := context.Background()

	// First 10 requests should be allowed (burst capacity)
	for i := 0; i < 10; i++ {
		allowed, err := limiter.Allow(ctx, "test-key")
		if err != nil {
			t.Fatalf("Allow failed: %v", err)
		}
		if !allowed {
			t.Errorf("request %d should be allowed", i+1)
		}
	}

	// 11th request should be denied (bucket empty)
	allowed, err := limiter.Allow(ctx, "test-key")
	if err != nil {
		t.Fatalf("Allow failed: %v", err)
	}
	if allowed {
		t.Error("request 11 should be denied")
	}
}

// TestTokenBucketLimiter_Refill tests token refill over time.
func TestTokenBucketLimiter_Refill(t *testing.T) {
	// 5 requests per 100ms with capacity of 5
	limiter := NewTokenBucketLimiter(5, 5, 100*time.Millisecond)
	defer limiter.Close()
	ctx := context.Background()

	// Exhaust bucket
	for i := 0; i < 5; i++ {
		_, _ = _, _ = limiter.Allow(ctx, "test-key")
	}

	// Should be denied immediately
	allowed, _ := limiter.Allow(ctx, "test-key")
	if allowed {
		t.Error("request should be denied before refill")
	}

	// Wait for refill
	time.Sleep(150 * time.Millisecond)

	// Should be allowed after refill
	allowed, err := limiter.Allow(ctx, "test-key")
	if err != nil {
		t.Fatalf("Allow failed: %v", err)
	}
	if !allowed {
		t.Error("request should be allowed after refill")
	}
}

// TestTokenBucketLimiter_AllowN tests multiple token consumption.
func TestTokenBucketLimiter_AllowN(t *testing.T) {
	limiter := NewTokenBucketLimiter(10, 10, time.Second)
	defer limiter.Close()
	ctx := context.Background()

	// Consume 5 tokens at once
	allowed, err := limiter.AllowN(ctx, "test-key", 5)
	if err != nil {
		t.Fatalf("AllowN failed: %v", err)
	}
	if !allowed {
		t.Error("AllowN(5) should be allowed")
	}

	// Consume another 5 tokens
	allowed, _ = limiter.AllowN(ctx, "test-key", 5)
	if !allowed {
		t.Error("AllowN(5) should be allowed (10 tokens total)")
	}

	// Try to consume 1 more (should fail)
	allowed, _ = limiter.AllowN(ctx, "test-key", 1)
	if allowed {
		t.Error("AllowN(1) should be denied (bucket empty)")
	}
}

// TestTokenBucketLimiter_MultipleKeys tests isolation between keys.
func TestTokenBucketLimiter_MultipleKeys(t *testing.T) {
	limiter := NewTokenBucketLimiter(5, 5, time.Second)
	defer limiter.Close()
	ctx := context.Background()

	// Exhaust first key
	for i := 0; i < 5; i++ {
		_, _ = limiter.Allow(ctx, "key1")
	}

	// First key should be denied
	allowed, _ := limiter.Allow(ctx, "key1")
	if allowed {
		t.Error("key1 should be denied")
	}

	// Second key should still be allowed (different bucket)
	allowed, err := limiter.Allow(ctx, "key2")
	if err != nil {
		t.Fatalf("Allow for key2 failed: %v", err)
	}
	if !allowed {
		t.Error("key2 should be allowed (separate bucket)")
	}
}

// TestTokenBucketLimiter_Reset tests rate limit reset.
func TestTokenBucketLimiter_Reset(t *testing.T) {
	limiter := NewTokenBucketLimiter(5, 5, time.Second)
	defer limiter.Close()
	ctx := context.Background()

	// Exhaust bucket
	for i := 0; i < 5; i++ {
		_, _ = limiter.Allow(ctx, "test-key")
	}

	// Should be denied
	allowed, _ := limiter.Allow(ctx, "test-key")
	if allowed {
		t.Error("request should be denied before reset")
	}

	// Reset the key
	err := limiter.Reset(ctx, "test-key")
	if err != nil {
		t.Fatalf("Reset failed: %v", err)
	}

	// Should be allowed after reset (new full bucket)
	allowed, err = limiter.Allow(ctx, "test-key")
	if err != nil {
		t.Fatalf("Allow after reset failed: %v", err)
	}
	if !allowed {
		t.Error("request should be allowed after reset")
	}
}

// TestTokenBucketLimiter_GetLimit tests limit retrieval.
func TestTokenBucketLimiter_GetLimit(t *testing.T) {
	limiter := NewTokenBucketLimiter(10, 5, 2*time.Second)
	defer limiter.Close()

	requests, window := limiter.GetLimit()
	if requests != 5 {
		t.Errorf("requests = %d, want 5", requests)
	}
	if window != 2*time.Second {
		t.Errorf("window = %v, want 2s", window)
	}
}

// TestTokenBucketLimiter_Concurrent tests concurrent access.
func TestTokenBucketLimiter_Concurrent(t *testing.T) {
	limiter := NewTokenBucketLimiter(100, 100, time.Second)
	defer limiter.Close()
	ctx := context.Background()

	var wg sync.WaitGroup
	allowedCount := 0
	var mu sync.Mutex

	// 200 concurrent requests (only 100 should succeed)
	for i := 0; i < 200; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			allowed, _ := limiter.Allow(ctx, "test-key")
			if allowed {
				mu.Lock()
				allowedCount++
				mu.Unlock()
			}
		}()
	}

	wg.Wait()

	// Approximately 100 should be allowed (with some tolerance for refill)
	if allowedCount < 100 || allowedCount > 110 {
		t.Errorf("allowed count = %d, want ~100", allowedCount)
	}
}

// TestSlidingWindowLimiter_BasicAllow tests basic allow functionality.
func TestSlidingWindowLimiter_BasicAllow(t *testing.T) {
	// 10 requests per second
	limiter := NewSlidingWindowLimiter(10, time.Second)
	defer limiter.Close()
	ctx := context.Background()

	// First 10 requests should be allowed
	for i := 0; i < 10; i++ {
		allowed, err := limiter.Allow(ctx, "test-key")
		if err != nil {
			t.Fatalf("Allow failed: %v", err)
		}
		if !allowed {
			t.Errorf("request %d should be allowed", i+1)
		}
	}

	// 11th request should be denied
	allowed, err := limiter.Allow(ctx, "test-key")
	if err != nil {
		t.Fatalf("Allow failed: %v", err)
	}
	if allowed {
		t.Error("request 11 should be denied")
	}
}

// TestSlidingWindowLimiter_WindowSliding tests window sliding behavior.
func TestSlidingWindowLimiter_WindowSliding(t *testing.T) {
	// 5 requests per 200ms
	limiter := NewSlidingWindowLimiter(5, 200*time.Millisecond)
	defer limiter.Close()
	ctx := context.Background()

	// Exhaust limit
	for i := 0; i < 5; i++ {
		limiter.Allow(ctx, "test-key")
	}

	// Should be denied
	allowed, _ := limiter.Allow(ctx, "test-key")
	if allowed {
		t.Error("request should be denied")
	}

	// Wait for window to slide
	time.Sleep(250 * time.Millisecond)

	// Should be allowed (old requests outside window)
	allowed, err := limiter.Allow(ctx, "test-key")
	if err != nil {
		t.Fatalf("Allow failed: %v", err)
	}
	if !allowed {
		t.Error("request should be allowed after window slides")
	}
}

// TestSlidingWindowLimiter_AllowN tests multiple request allowance.
func TestSlidingWindowLimiter_AllowN(t *testing.T) {
	limiter := NewSlidingWindowLimiter(10, time.Second)
	defer limiter.Close()
	ctx := context.Background()

	// Allow 3 requests at once
	allowed, err := limiter.AllowN(ctx, "test-key", 3)
	if err != nil {
		t.Fatalf("AllowN failed: %v", err)
	}
	if !allowed {
		t.Error("AllowN(3) should be allowed")
	}

	// Allow 7 more
	allowed, _ = limiter.AllowN(ctx, "test-key", 7)
	if !allowed {
		t.Error("AllowN(7) should be allowed (10 total)")
	}

	// Try 1 more (should fail)
	allowed, _ = limiter.AllowN(ctx, "test-key", 1)
	if allowed {
		t.Error("AllowN(1) should be denied")
	}
}

// TestSlidingWindowLimiter_MultipleKeys tests isolation between keys.
func TestSlidingWindowLimiter_MultipleKeys(t *testing.T) {
	limiter := NewSlidingWindowLimiter(5, time.Second)
	defer limiter.Close()
	ctx := context.Background()

	// Exhaust first key
	for i := 0; i < 5; i++ {
		limiter.Allow(ctx, "key1")
	}

	// First key should be denied
	allowed, _ := limiter.Allow(ctx, "key1")
	if allowed {
		t.Error("key1 should be denied")
	}

	// Second key should be allowed
	allowed, err := limiter.Allow(ctx, "key2")
	if err != nil {
		t.Fatalf("Allow for key2 failed: %v", err)
	}
	if !allowed {
		t.Error("key2 should be allowed")
	}
}

// TestSlidingWindowLimiter_Reset tests reset functionality.
func TestSlidingWindowLimiter_Reset(t *testing.T) {
	limiter := NewSlidingWindowLimiter(5, time.Second)
	defer limiter.Close()
	ctx := context.Background()

	// Exhaust limit
	for i := 0; i < 5; i++ {
		limiter.Allow(ctx, "test-key")
	}

	// Should be denied
	allowed, _ := limiter.Allow(ctx, "test-key")
	if allowed {
		t.Error("request should be denied before reset")
	}

	// Reset
	err := limiter.Reset(ctx, "test-key")
	if err != nil {
		t.Fatalf("Reset failed: %v", err)
	}

	// Should be allowed after reset
	allowed, err = limiter.Allow(ctx, "test-key")
	if err != nil {
		t.Fatalf("Allow after reset failed: %v", err)
	}
	if !allowed {
		t.Error("request should be allowed after reset")
	}
}

// TestSlidingWindowLimiter_GetLimit tests limit retrieval.
func TestSlidingWindowLimiter_GetLimit(t *testing.T) {
	limiter := NewSlidingWindowLimiter(20, 3*time.Second)
	defer limiter.Close()

	requests, window := limiter.GetLimit()
	if requests != 20 {
		t.Errorf("requests = %d, want 20", requests)
	}
	if window != 3*time.Second {
		t.Errorf("window = %v, want 3s", window)
	}
}

// TestSlidingWindowLimiter_Concurrent tests concurrent access.
func TestSlidingWindowLimiter_Concurrent(t *testing.T) {
	limiter := NewSlidingWindowLimiter(50, time.Second)
	defer limiter.Close()
	ctx := context.Background()

	var wg sync.WaitGroup
	allowedCount := 0
	var mu sync.Mutex

	// 100 concurrent requests (only 50 should succeed)
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			allowed, _ := limiter.Allow(ctx, "test-key")
			if allowed {
				mu.Lock()
				allowedCount++
				mu.Unlock()
			}
		}()
	}

	wg.Wait()

	if allowedCount != 50 {
		t.Errorf("allowed count = %d, want 50", allowedCount)
	}
}

// TestSlidingWindowLimiter_AccurateCounting tests accurate request counting.
func TestSlidingWindowLimiter_AccurateCounting(t *testing.T) {
	limiter := NewSlidingWindowLimiter(3, 100*time.Millisecond)
	defer limiter.Close()
	ctx := context.Background()

	// Make 3 requests
	for i := 0; i < 3; i++ {
		allowed, _ := limiter.Allow(ctx, "test-key")
		if !allowed {
			t.Errorf("request %d should be allowed", i+1)
		}
	}

	// Wait 60ms (requests still in window)
	time.Sleep(60 * time.Millisecond)

	// Should still be denied
	allowed, _ := limiter.Allow(ctx, "test-key")
	if allowed {
		t.Error("request should be denied (still in window)")
	}

	// Wait another 50ms (total 110ms, outside window)
	time.Sleep(50 * time.Millisecond)

	// Should be allowed (old requests expired)
	allowed, _ = limiter.Allow(ctx, "test-key")
	if !allowed {
		t.Error("request should be allowed (outside window)")
	}
}

// TestTokenBucketLimiter_BurstHandling tests burst capacity.
func TestTokenBucketLimiter_BurstHandling(t *testing.T) {
	// 1 request per second, but burst of 5
	limiter := NewTokenBucketLimiter(5, 1, time.Second)
	defer limiter.Close()
	ctx := context.Background()

	// Should handle 5 requests immediately (burst)
	for i := 0; i < 5; i++ {
		allowed, err := limiter.Allow(ctx, "test-key")
		if err != nil {
			t.Fatalf("Allow failed: %v", err)
		}
		if !allowed {
			t.Errorf("burst request %d should be allowed", i+1)
		}
	}

	// 6th should be denied (burst exhausted)
	allowed, _ := limiter.Allow(ctx, "test-key")
	if allowed {
		t.Error("request beyond burst should be denied")
	}

	// Wait for 1 token to refill
	time.Sleep(1100 * time.Millisecond)

	// Should get 1 request (1 token refilled)
	allowed, _ = limiter.Allow(ctx, "test-key")
	if !allowed {
		t.Error("request should be allowed after refill")
	}
}

// TestSlidingWindowLimiter_PartialWindowOverlap tests partial window overlap.
func TestSlidingWindowLimiter_PartialWindowOverlap(t *testing.T) {
	limiter := NewSlidingWindowLimiter(5, 200*time.Millisecond)
	defer limiter.Close()
	ctx := context.Background()

	// Make 3 requests
	for i := 0; i < 3; i++ {
		limiter.Allow(ctx, "test-key")
	}

	// Wait 120ms
	time.Sleep(120 * time.Millisecond)

	// Make 2 more requests (total 5 in window)
	for i := 0; i < 2; i++ {
		allowed, _ := limiter.Allow(ctx, "test-key")
		if !allowed {
			t.Errorf("request should be allowed")
		}
	}

	// Should be denied (5 requests in window)
	allowed, _ := limiter.Allow(ctx, "test-key")
	if allowed {
		t.Error("request should be denied")
	}

	// Wait another 100ms (total 220ms from first requests)
	time.Sleep(100 * time.Millisecond)

	// First 3 requests should be outside window now
	// Should be able to make 3 more requests
	for i := 0; i < 3; i++ {
		allowed, _ := limiter.Allow(ctx, "test-key")
		if !allowed {
			t.Errorf("request %d should be allowed after partial window slide", i+1)
		}
	}
}

// TestTokenBucketLimiter_ZeroRefill tests behavior with zero refill rate.
func TestTokenBucketLimiter_ZeroRefill(t *testing.T) {
	// 10 capacity, 0 refill (one-time use)
	limiter := NewTokenBucketLimiter(10, 0, time.Second)
	defer limiter.Close()
	ctx := context.Background()

	// Should allow 10 requests
	for i := 0; i < 10; i++ {
		allowed, _ := limiter.Allow(ctx, "test-key")
		if !allowed {
			t.Errorf("request %d should be allowed", i+1)
		}
	}

	// Should deny subsequent requests even after waiting
	time.Sleep(1100 * time.Millisecond)
	allowed, _ := limiter.Allow(ctx, "test-key")
	if allowed {
		t.Error("request should be denied (no refill)")
	}
}

// TestSlidingWindowLimiter_EmptyWindow tests behavior with empty window.
func TestSlidingWindowLimiter_EmptyWindow(t *testing.T) {
	limiter := NewSlidingWindowLimiter(5, time.Second)
	defer limiter.Close()
	ctx := context.Background()

	// Reset to ensure empty
	_ = _ = limiter.Reset(ctx, "test-key")

	// Should allow up to limit
	for i := 0; i < 5; i++ {
		allowed, err := limiter.Allow(ctx, "test-key")
		if err != nil {
			t.Fatalf("Allow failed: %v", err)
		}
		if !allowed {
			t.Errorf("request %d should be allowed in empty window", i+1)
		}
	}
}

// TestTokenBucketLimiter_Close tests cleanup.
func TestTokenBucketLimiter_Close(t *testing.T) {
	limiter := NewTokenBucketLimiter(10, 10, time.Second)

	err := limiter.Close()
	if err != nil {
		t.Errorf("Close failed: %v", err)
	}
}

// TestSlidingWindowLimiter_Close tests cleanup.
func TestSlidingWindowLimiter_Close(t *testing.T) {
	limiter := NewSlidingWindowLimiter(10, time.Second)

	err := limiter.Close()
	if err != nil {
		t.Errorf("Close failed: %v", err)
	}
}

// TestRateLimitersComparison tests behavioral differences between limiters.
func TestRateLimitersComparison(t *testing.T) {
	ctx := context.Background()

	// Token bucket: 5 capacity, 1 refill per second
	tokenBucket := NewTokenBucketLimiter(5, 1, time.Second)
	defer tokenBucket.Close()

	// Sliding window: 5 requests per second
	slidingWindow := NewSlidingWindowLimiter(5, time.Second)
	defer slidingWindow.Close()

	// Both should allow initial burst of 5
	for i := 0; i < 5; i++ {
		tb, _ := tokenBucket.Allow(ctx, "test")
		sw, _ := slidingWindow.Allow(ctx, "test")
		if !tb || !sw {
			t.Errorf("request %d: both should allow initial burst", i+1)
		}
	}

	// Both should deny 6th request
	tb, _ := tokenBucket.Allow(ctx, "test")
	sw, _ := slidingWindow.Allow(ctx, "test")
	if tb || sw {
		t.Error("both should deny 6th request")
	}

	// Wait 1 second
	time.Sleep(1100 * time.Millisecond)

	// Token bucket should allow 1 request (refilled 1 token)
	tb, _ = tokenBucket.Allow(ctx, "test")
	if !tb {
		t.Error("token bucket should allow 1 request after refill")
	}

	// Sliding window should allow 5 requests (all previous outside window)
	for i := 0; i < 5; i++ {
		sw, _ := slidingWindow.Allow(ctx, "test")
		if !sw {
			t.Errorf("sliding window should allow request %d after window slides", i+1)
		}
	}
}

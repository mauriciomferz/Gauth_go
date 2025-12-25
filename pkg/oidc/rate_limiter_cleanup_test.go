package oidc

import (
	"context"
	"testing"
	"time"
)

func TestTokenBucketLimiter_Lifecycle(t *testing.T) {
	// 1. Create limiter
	l := NewTokenBucketLimiter(10, 1, time.Second)

	// 2. Use it
	ctx := context.Background()
	_, _ = l.Allow(ctx, "key1")

	// 3. Close and verify logic works (no panic/hang)
	// If cleanupLoop doesn't exit, this might not detect it immediately in unit test
	// unless run with leak detector, but we ensure Close() waits for WaitGroup.

	done := make(chan struct{})
	go func() {
		_ = l.Close()
		close(done)
	}()

	select {
	case <-done:
		// Success
	case <-time.After(1 * time.Second):
		t.Fatal("Close() timed out - likely goroutine leak in cleanupLoop")
	}
}

func TestSlidingWindowLimiter_Lifecycle(t *testing.T) {
	l := NewSlidingWindowLimiter(10, time.Second)
	ctx := context.Background()
	_, _ = l.Allow(ctx, "key1")

	done := make(chan struct{})
	go func() {
		_ = l.Close()
		close(done)
	}()

	select {
	case <-done:
		// Success
	case <-time.After(1 * time.Second):
		t.Fatal("Close() timed out - likely goroutine leak in cleanupLoop")
	}
}

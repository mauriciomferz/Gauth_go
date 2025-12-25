package ratelimit

import (
	"testing"
	"time"
)

func TestGenericLimiter_Lifecycle(t *testing.T) {
	cfg := Config{
		Algorithm: TokenBucket,
		Rate:      10,
		Period:    time.Second,
		Burst:     10,
	}
	l := NewLimiter(cfg)

	// Use it
	l.Allow("key1")

	// Verify Close logic
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

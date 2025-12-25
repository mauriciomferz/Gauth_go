package oidc

import (
	"context"
	"testing"
	"time"
)

func TestTokenRevocationService_Lifecycle(t *testing.T) {
	s := NewTokenRevocationService()

	// Simulate usage
	_ = s.RevokeToken(context.Background(), "token1", "reason", "user1", time.Now().Add(time.Hour))

	// Verify cleanup
	done := make(chan struct{})
	go func() {
		s.Stop()
		close(done)
	}()

	select {
	case <-done:
		// Success
	case <-time.After(1 * time.Second):
		t.Fatal("Stop() timed out - likely goroutine leak in cleanupLoop")
	}
}

func TestRefreshTokenService_Lifecycle(t *testing.T) {
	s := NewRefreshTokenService()

	// Simulate usage
	entry := &RefreshTokenEntry{
		RefreshToken: "refresh1",
		ExpiresAt:    time.Now().Add(time.Hour),
	}
	_ = s.StoreRefreshToken(context.Background(), "token1", entry)

	// Verify cleanup
	done := make(chan struct{})
	go func() {
		s.Stop()
		close(done)
	}()

	select {
	case <-done:
		// Success
	case <-time.After(1 * time.Second):
		t.Fatal("Stop() timed out - likely goroutine leak in cleanupLoop")
	}
}

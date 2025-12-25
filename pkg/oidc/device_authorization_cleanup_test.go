package oidc

import (
	"context"
	"testing"
	"time"
)

func TestDeviceAuthorizationService_Lifecycle(t *testing.T) {
	config := &DeviceAuthorizationConfig{
		DeviceCodeLifetime: time.Second, // fast expiry for test
		PollingInterval:    time.Second,
	}
	s := NewDeviceAuthorizationService(config)

	// Simulate usage
	req := &DeviceAuthorizationRequest{ClientID: "client1"}
	_, err := s.AuthorizeDevice(context.Background(), req)
	if err != nil {
		t.Fatalf("Failed to authorize device: %v", err)
	}

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

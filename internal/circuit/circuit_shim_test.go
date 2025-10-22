package circuit

import (
	"context"
	"errors"
	"testing"
	"time"
)

// TestShimEquivalence ensures the legacy New shim constructs a breaker equivalent
// (in initial observable state) to NewBreaker with corresponding options.
func TestShimEquivalence(t *testing.T) {
	legacy := New("shim", 2, 200*time.Millisecond)
	modern := NewBreaker(Options{Name: "shim", FailureThreshold: 2, ResetTimeout: 200 * time.Millisecond})

	if legacy.GetState() != modern.GetState() {
		t.Fatalf("expected identical initial state, got %v vs %v", legacy.GetState(), modern.GetState())
	}

	// Induce consecutive failures to open both breakers
	sentinel := errors.New("boom")
	ctx := context.TODO()
	for i := 0; i < 2; i++ {
		_ = legacy.Execute(ctx, func() error { return sentinel })
		_ = modern.Execute(ctx, func() error { return sentinel })
	}
	if legacy.GetState() == StateClosed || modern.GetState() == StateClosed {
		t.Fatalf("expected both breakers to transition to open after threshold failures: legacy=%v modern=%v", legacy.GetState(), modern.GetState())
	}

	// Wait for reset timeout and probe success to close
	time.Sleep(210 * time.Millisecond)
	if err := legacy.Execute(ctx, func() error { return nil }); err != nil {
		t.Fatalf("legacy probe execution unexpected error: %v", err)
	}
	if err := modern.Execute(ctx, func() error { return nil }); err != nil {
		t.Fatalf("modern probe execution unexpected error: %v", err)
	}
	if legacy.GetState() != StateClosed || modern.GetState() != StateClosed {
		t.Fatalf("expected both breakers to close after successful probe: legacy=%v modern=%v", legacy.GetState(), modern.GetState())
	}
}

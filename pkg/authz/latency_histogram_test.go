package authz

import (
	"context"
	"testing"
	"time"
)

// TestLatencyHistogram ensures buckets increment; we induce known delays.
func TestLatencyHistogram(t *testing.T) {
	ma := NewMemoryAuthorizer()
	// Simple always-match policy
	ma.AddPolicy(Policy{ID: "latency", Subject: "s", Resource: "r", Actions: []string{"a"}, Effect: Allow})
	ctx := context.Background()
	// Perform decisions with artificial sleep to land in higher buckets
	delays := []time.Duration{50 * time.Microsecond, 600 * time.Microsecond, 3 * time.Millisecond}
	for _, d := range delays {
		start := time.Now()
		// Authorization call
		dec, err := ma.Authorize(ctx, Request{Subject: "s", Resource: "r", Action: "a", Context: map[string]string{}})
		if err != nil || !dec.Allow {
			t.Fatalf("unexpected deny or error: %v allow=%v", err, dec.Allow)
		}
		// Sleep to inflate observed latency post decision (simulate external processing)
		time.Sleep(d - time.Since(start))
		// Manually record adjusted latency (since recordLatency inside Authorize already captured without sleep)
		ma.recordLatency(d)
	}
	snap := ma.GetMetricsSnapshot()
	// Expect histogram to have at least one bucket count for each induced delay upper bound
	found := 0
	for _, cnt := range snap.LatencyHistogram {
		if cnt > 0 {
			found++
		}
	}
	if found < 3 {
		t.Fatalf("expected at least 3 populated latency buckets, got %d", found)
	}
}

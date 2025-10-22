package authz

import (
	"context"
	"strings"
	"testing"
	"time"
)

// TestMetricsBasic verifies counters and latency accumulate.
func TestMetricsBasic(t *testing.T) {
	ma := NewMemoryAuthorizer()
	ma.EnableCaching(50 * time.Millisecond)
	ma.AddPolicy(Policy{ID: "allow1", Subject: "alice", Resource: "doc1", Actions: []string{"read"}, Effect: Allow})

	ctx := context.Background()
	// first decision (miss)
	d1, err := ma.Authorize(ctx, Request{Subject: "alice", Resource: "doc1", Action: "read"})
	if err != nil || !d1.Allow {
		t.Fatalf("expected allow decision, got err=%v allow=%v", err, d1.Allow)
	}
	if d1.Metadata["cache_hit"] != metadataCacheHitFalse {
		t.Fatalf("expected cache miss first decision")
	}

	// second decision (hit)
	d2, _ := ma.Authorize(ctx, Request{Subject: "alice", Resource: "doc1", Action: "read"})
	if d2.Metadata["cache_hit"] != metadataCacheHitTrue {
		t.Fatalf("expected cache hit second decision")
	}

	// third decision different resource (miss default deny)
	d3, _ := ma.Authorize(ctx, Request{Subject: "alice", Resource: "other", Action: "read"})
	if d3.Allow {
		t.Fatalf("expected deny for unmatched resource")
	}

	snap := ma.GetMetricsSnapshot()
	if snap.Decisions < 3 {
		t.Fatalf("expected >=3 decisions, got %d", snap.Decisions)
	}
	if snap.CacheHits < 1 {
		t.Fatalf("expected at least 1 cache hit, got %d", snap.CacheHits)
	}
	if snap.CacheMisses < 2 {
		t.Fatalf("expected at least 2 cache misses, got %d", snap.CacheMisses)
	}
	if snap.AvgLatencyNs <= 0 {
		t.Fatalf("expected positive avg latency, got %f", snap.AvgLatencyNs)
	}
	if snap.P99LatencyNs < snap.AvgLatencyNs {
		t.Fatalf("p99 should be >= mean, mean=%f p99=%f", snap.AvgLatencyNs, snap.P99LatencyNs)
	}
}

// TestMetricsConcurrency exercises concurrent authorization calls.
func TestMetricsConcurrency(t *testing.T) {
	ma := NewMemoryAuthorizer()
	ma.EnableCaching(100 * time.Millisecond)
	ma.AddPolicy(Policy{ID: "allowA", Subject: "bob", Resource: "resA", Actions: []string{"read"}, Effect: Allow})

	ctx := context.Background()
	const workers = 10
	const iterations = 50
	errCh := make(chan error, workers)
	for w := 0; w < workers; w++ {
		go func(id int) {
			for i := 0; i < iterations; i++ {
				_, err := ma.Authorize(ctx, Request{Subject: "bob", Resource: "resA", Action: "read"})
				if err != nil {
					errCh <- err
					return
				}
			}
			errCh <- nil
		}(w)
	}
	for w := 0; w < workers; w++ {
		if err := <-errCh; err != nil {
			t.Fatalf("worker error: %v", err)
		}
	}
	snap := ma.GetMetricsSnapshot()
	expected := uint64(workers * iterations)
	if snap.Decisions < expected {
		t.Fatalf("expected decisions >= %d got %d", expected, snap.Decisions)
	}
	if snap.CacheHits == 0 {
		t.Fatalf("expected some cache hits")
	}
	if snap.AvgLatencyNs <= 0 {
		t.Fatalf("expected positive latency")
	}
}

// TestFineGrainedInvalidation validates targeted cache eviction by subject and resource.
func TestFineGrainedInvalidation(t *testing.T) {
	ma := NewMemoryAuthorizer()
	ma.EnableCaching(1 * time.Minute)
	ma.AddPolicy(Policy{ID: "p1", Subject: "u1", Resource: "objA", Actions: []string{"read"}, Effect: Allow})
	ma.AddPolicy(Policy{ID: "p2", Subject: "u2", Resource: "objB", Actions: []string{"read"}, Effect: Allow})
	ctx := context.Background()
	// prime cache
	_, _ = ma.Authorize(ctx, Request{Subject: "u1", Resource: "objA", Action: "read"})
	_, _ = ma.Authorize(ctx, Request{Subject: "u2", Resource: "objB", Action: "read"})
	// confirm entries present
	if len(ma.cache) < 2 {
		t.Fatalf("expected at least 2 cache entries, got %d", len(ma.cache))
	}
	// invalidate subject u1
	ma.InvalidateSubject("u1")
	for k := range ma.cache {
		if strings.HasPrefix(k, "u1|") {
			t.Fatalf("subject u1 cache entry not evicted: %s", k)
		}
	}
	// invalidate resource objB
	ma.InvalidateResource("objB")
	for k := range ma.cache {
		parts := strings.SplitN(k, "|", 3)
		if len(parts) >= 2 && parts[1] == "objB" {
			t.Fatalf("resource objB cache entry not evicted: %s", k)
		}
	}
	// ensure other entries remain (u1 removed, objB removed -> expect 0 or 1)
	if len(ma.cache) > 1 {
		t.Fatalf("unexpected remaining cache entries: %d", len(ma.cache))
	}
}

func TestPolicyConflictDiagnostics(t *testing.T) {
	ma := NewMemoryAuthorizer()
	// conflicting allow + deny policies
	ma.AddPolicy(Policy{ID: "deny1", Subject: "u", Resource: "r", Actions: []string{"read"}, Effect: Deny})
	ma.AddPolicy(Policy{ID: "allow1", Subject: "u", Resource: "r", Actions: []string{"read"}, Effect: Allow})
	ma.SetCombiningStrategy(DenyOverrides)
	dec, _ := ma.Authorize(context.Background(), Request{Subject: "u", Resource: "r", Action: "read"})
	if dec.Allow {
		t.Fatalf("expected deny due to deny_overrides")
	}
	if dec.Metadata["policy_conflict"] == "" {
		t.Fatalf("expected policy_conflict metadata populated")
	}
	snap := ma.GetMetricsSnapshot()
	if snap.Decisions == 0 || snap.AvgLatencyNs == 0 {
		t.Fatalf("expected latency recorded for decision")
	}
	if snap.Conflicts == 0 {
		t.Fatalf("expected conflict counter to increment")
	}
}

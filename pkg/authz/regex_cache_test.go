package authz

import (
	"context"
	"testing"
)

// TestRegexCaching ensures a regex pattern is compiled only once across multiple decisions.
func TestRegexCaching(t *testing.T) {
	ma := NewMemoryAuthorizer()
	// policy with regex operator; pattern should compile once
	ma.AddPolicy(Policy{
		ID:       "rx1",
		Subject:  "alice",
		Resource: "svc",
		Actions:  []string{"call"},
		Effect:   Allow,
		Conditions: []Condition{{
			Key:      "email",
			Operator: "regex",
			Values:   []string{"^alice@.*\\.com$"},
		}},
	})
	ctx := context.Background()
	// two decisions matching same regex pattern
	for i := 0; i < 2; i++ {
		dec, err := ma.Authorize(ctx, Request{Subject: "alice", Resource: "svc", Action: "call", Context: map[string]string{"email": "alice@example.com"}})
		if err != nil {
			t.Fatalf("decision err: %v", err)
		}
		if !dec.Allow {
			t.Fatalf("expected allow, got deny: %s", dec.Reason)
		}
	}
	snap := ma.GetMetricsSnapshot()
	if snap.RegexCompiles != 1 {
		t.Fatalf("expected 1 regex compile, got %d", snap.RegexCompiles)
	}
	if snap.RegexCacheSize != 1 {
		t.Fatalf("expected cache size 1, got %d", snap.RegexCacheSize)
	}
	if snap.RegexCompileErrors != 0 {
		t.Fatalf("expected 0 compile errors, got %d", snap.RegexCompileErrors)
	}
}

// TestRegexCompileError validates failed pattern doesn't allow and increments error metric.
func TestRegexCompileError(t *testing.T) {
	ma := NewMemoryAuthorizer()
	// invalid regex pattern (unclosed group)
	ma.AddPolicy(Policy{
		ID:       "rx-bad",
		Subject:  "bob",
		Resource: "svc",
		Actions:  []string{"call"},
		Effect:   Allow,
		Conditions: []Condition{{
			Key:      "email",
			Operator: "regex",
			Values:   []string{"^bob(.*"},
		}},
	})
	dec, _ := ma.Authorize(context.Background(), Request{Subject: "bob", Resource: "svc", Action: "call", Context: map[string]string{"email": "bob@example.com"}})
	// should deny because condition evaluation fails
	if dec.Allow {
		t.Fatalf("expected deny due to invalid regex pattern")
	}
	snap := ma.GetMetricsSnapshot()
	if snap.RegexCompileErrors == 0 {
		t.Fatalf("expected regex compile error metric increment")
	}
	if snap.RegexCompiles != 0 {
		t.Fatalf("expected no successful compiles, got %d", snap.RegexCompiles)
	}
	if snap.RegexCacheSize != 0 {
		t.Fatalf("expected cache size 0 (pattern not stored), got %d", snap.RegexCacheSize)
	}
}

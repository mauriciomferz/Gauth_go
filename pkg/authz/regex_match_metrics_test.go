package authz

import (
	"context"
	"testing"
)

// TestRegexMatchFrequency ensures regex match count increments per successful evaluation.
func TestRegexMatchFrequency(t *testing.T) {
	ma := NewMemoryAuthorizer()
	ma.AddPolicy(Policy{
		ID:       "rx-match",
		Subject:  "eve",
		Resource: "svc",
		Actions:  []string{"call"},
		Effect:   Allow,
		Conditions: []Condition{{
			Key:      "email",
			Operator: "regex",
			Values:   []string{"^eve@.*\\.org$"},
		}},
	})
	ctx := context.Background()
	// execute multiple matching decisions
	for i := 0; i < 3; i++ {
		dec, err := ma.Authorize(ctx, Request{
			Subject: "eve", Resource: "svc", Action: "call",
			Context: map[string]string{"email": "eve@example.org"},
		})
		if err != nil {
			t.Fatalf("authorize err: %v", err)
		}
		if !dec.Allow {
			t.Fatalf("expected allow")
		}
	}
	snap := ma.GetMetricsSnapshot()
	if snap.RegexMatches != 3 {
		t.Fatalf("expected 3 regex matches, got %d", snap.RegexMatches)
	}
	if snap.RegexCompiles != 1 {
		t.Fatalf("expected single compile, got %d", snap.RegexCompiles)
	}
}

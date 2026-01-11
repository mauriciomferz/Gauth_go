package authz

import (
	"context"
	"testing"
	"time"
)

// TestRegexCapacityEviction validates LRU eviction when capacity exceeded.
func TestRegexCapacityEviction(t *testing.T) {
	ma := NewMemoryAuthorizer()
	ma.SetRegexCacheCapacity(2)
	// Add policies with distinct regex patterns; access first to make second oldest.
	patterns := []string{"^a.*$", "^b.*$", "^c.*$"}
	for i, p := range patterns {
		ma.AddPolicy(Policy{
			ID:       p,
			Subject:  "sub",
			Resource: "res",
			Actions:  []string{"act"},
			Effect:   Allow,
			Conditions: []Condition{{
				Key:      "val",
				Operator: "regex",
				Values:   []string{p},
			}},
		})
		// trigger compile/match
		_, err := ma.Authorize(context.Background(), Request{
			Subject: "sub", Resource: "res", Action: "act",
			Context: map[string]string{"val": string([]byte{'a' + byte(i)})},
		})
		if err != nil {
			t.Fatalf("authorize error: %v", err)
		}
	}
	snap := ma.GetMetricsSnapshot()
	if snap.RegexCacheSize != 2 {
		t.Fatalf("expected cache size 2 after eviction, got %d", snap.RegexCacheSize)
	}
	if snap.RegexEvictions == 0 {
		t.Fatalf("expected eviction metric increment")
	}
}

// TestRegexTTLEviction validates TTL expiry removes patterns.
func TestRegexTTLEviction(t *testing.T) {
	ma := NewMemoryAuthorizer()
	ma.SetRegexCacheTTL(50 * time.Millisecond)
	ma.AddPolicy(Policy{
		ID:       "ttl",
		Subject:  "s",
		Resource: "r",
		Actions:  []string{"a"},
		Effect:   Allow,
		Conditions: []Condition{{
			Key:      "val",
			Operator: "regex",
			Values:   []string{"^x.*$"},
		}},
	})
	_, err := ma.Authorize(context.Background(), Request{
		Subject: "s", Resource: "r", Action: "a", Context: map[string]string{"val": "xyz"},
	})
	if err != nil {
		t.Fatalf("authorize error: %v", err)
	}
	// wait for TTL to expire and trigger prune via another miss compile
	time.Sleep(60 * time.Millisecond)
	ma.AddPolicy(Policy{
		ID:       "other",
		Subject:  "s",
		Resource: "r",
		Actions:  []string{"a"},
		Effect:   Allow,
		Conditions: []Condition{{
			Key:      "val",
			Operator: "regex",
			Values:   []string{"^y.*$"},
		}},
	})
	_, _ = ma.Authorize(context.Background(), Request{
		Subject: "s", Resource: "r", Action: "a", Context: map[string]string{"val": "yyy"},
	})
	snap := ma.GetMetricsSnapshot()
	// TTL should have evicted first pattern; cache size should be 2 or 1 depending on prune ordering
	if snap.RegexEvictions == 0 {
		t.Fatalf("expected at least one eviction due to TTL")
	}
}

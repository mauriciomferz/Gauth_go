package test

import (
	"context"
	"testing"

	"github.com/mauriciomferz/Gauth_go/pkg/authz"
)

// TestAuthorizationCacheBasic ensures hits/misses and staleness invalidation via policy version changes.
func TestAuthorizationCacheBasic(t *testing.T) {
	ma := authz.NewMemoryAuthorizer()
	ma.AddPolicy(authz.Policy{ID: "p1", Subject: "alice", Resource: "doc:1", Actions: []string{"read"}, Effect: authz.Allow})
	version := ma.Snapshot() // establish snapshot version
	if version != 1 {
		t.Fatalf("expected first snapshot version=1 got %d", version)
	}

	cache := authz.NewAuthorizationCache(8)
	ma.SetDecisionCache(cache)
	ma.SetJurisdiction("us")

	req := authz.Request{Subject: "alice", Resource: "doc:1", Action: "read", Context: map[string]string{}}
	dec1, err := ma.Authorize(context.Background(), req)
	if err != nil {
		t.Fatalf("authorize err: %v", err)
	}
	if !dec1.Allow {
		t.Fatalf("expected allow first decision")
	}
	if dec1.Metadata["cache_hit"] != "false" {
		t.Fatalf("expected first decision cache_hit=false got %v", dec1.Metadata["cache_hit"])
	}

	dec2, err := ma.Authorize(context.Background(), req)
	if err != nil {
		t.Fatalf("authorize err: %v", err)
	}
	if dec2.Metadata["cache_hit"] != "true" {
		t.Fatalf("expected second decision cache hit true got %v", dec2.Metadata["cache_hit"])
	}

	// Change policy set (new version) -> cached entry becomes stale
	ma.AddPolicy(authz.Policy{ID: "p2", Subject: "alice", Resource: "doc:1", Actions: []string{"read"}, Effect: authz.Allow})
	newVersion := ma.Snapshot()
	if newVersion != 2 {
		t.Fatalf("expected second snapshot version=2 got %d", newVersion)
	}

	dec3, err := ma.Authorize(context.Background(), req)
	if err != nil {
		t.Fatalf("authorize err: %v", err)
	}
	if dec3.Metadata["cache_hit"] != "false" {
		t.Fatalf("expected post-version-change cache miss got %v", dec3.Metadata["cache_hit"])
	}

	// Jurisdiction change invalidates cache
	ma.SetJurisdiction("eu")
	dec4, err := ma.Authorize(context.Background(), req)
	if err != nil {
		t.Fatalf("authorize err: %v", err)
	}
	if dec4.Metadata["cache_hit"] != "false" {
		t.Fatalf("expected jurisdiction change miss got %v", dec4.Metadata["cache_hit"])
	}

	// Ensure subsequent hit again
	dec5, _ := ma.Authorize(context.Background(), req)
	if dec5.Metadata["cache_hit"] != "true" {
		t.Fatalf("expected follow-up hit got %v", dec5.Metadata["cache_hit"])
	}

	metrics := cache.Snapshot()
	if metrics.Hits == 0 || metrics.Misses == 0 {
		t.Fatalf("expected some hits and misses, got hits=%d misses=%d", metrics.Hits, metrics.Misses)
	}
}

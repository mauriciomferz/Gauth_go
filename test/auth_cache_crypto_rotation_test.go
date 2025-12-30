package test

import (
	"context"
	"testing"

	"github.com/mauriciomferz/AgentAuth/pkg/authz"
)

const (
	cacheMetadataFalse = "false"
	cacheMetadataTrue  = "true"
)

// TestAuthorizationCacheCryptoRotationInvalidation ensures rotation hook flushes cache.
func TestAuthorizationCacheCryptoRotationInvalidation(t *testing.T) {
	ma := authz.NewMemoryAuthorizer()
	ma.AddPolicy(authz.Policy{ID: "p1", Subject: "sam", Resource: "obj:1", Actions: []string{"read"}, Effect: authz.Allow})
	ma.Snapshot()
	cache := authz.NewLRUDecisionCache(4)
	ma.SetDecisionCache(cache)

	req := authz.Request{Subject: "sam", Resource: "obj:1", Action: "read"}
	dec1, _ := ma.Authorize(context.Background(), req)
	if dec1.Metadata["cache_hit"] != cacheMetadataFalse {
		t.Fatalf("expected initial miss got %v", dec1.Metadata["cache_hit"])
	}
	dec2, _ := ma.Authorize(context.Background(), req)
	if dec2.Metadata["cache_hit"] != cacheMetadataTrue {
		t.Fatalf("expected second hit got %v", dec2.Metadata["cache_hit"])
	}

	// Simulate crypto key rotation
	ma.InvalidateOnCryptoRotation()

	dec3, _ := ma.Authorize(context.Background(), req)
	if dec3.Metadata["cache_hit"] != cacheMetadataFalse {
		t.Fatalf("expected post-rotation miss got %v", dec3.Metadata["cache_hit"])
	}
	dec4, _ := ma.Authorize(context.Background(), req)
	if dec4.Metadata["cache_hit"] != cacheMetadataTrue {
		t.Fatalf("expected re-populated hit got %v", dec4.Metadata["cache_hit"])
	}

	metrics := ma.AuthorizationCacheMetrics()
	if metrics == nil {
		t.Fatalf("expected metrics snapshot")
	}
	if metrics.Invalidations == 0 {
		t.Fatalf("expected invalidation counter >0 after rotation flush")
	}
}

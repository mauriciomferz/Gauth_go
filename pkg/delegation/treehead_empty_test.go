package delegation

import (
	"testing"
	"time"

	crypto "github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/internal/crypto"
)

// TestSignTreeHeadEmptyChain ensures calling SignTreeHead on an empty revocation chain
// returns (nil,nil) and does not create a tree head or panic. This guards against
// regressions where empty chains produced confusing SignedTreeHead entries or triggered
// index errors in rotation callbacks.
func TestSignTreeHeadEmptyChain(t *testing.T) {
	// Install a no-op key manager so multi-sig / signatures path is initialized but irrelevant
	km, err := crypto.NewManager(1 * time.Hour)
	if err != nil {
		t.Fatalf("km init: %v", err)
	}
	crypto.GlobalEdDSARegistry = km

	rc := NewRevocationChain()
	// sanity: no events
	if len(rc.Events()) != 0 {
		t.Fatalf("expected zero events")
	}
	sth, err := rc.SignTreeHead()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sth != nil {
		t.Fatalf("expected nil sth for empty chain, got %+v", sth)
	}
	if rc.LatestTreeHead() != nil {
		t.Fatalf("no tree head should have been appended")
	}
}

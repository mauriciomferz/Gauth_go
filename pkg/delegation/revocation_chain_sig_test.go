package delegation

import (
	"testing"
	"time"

	"github.com/mauriciomferz/Gauth_go/pkg/crypto"
)

// helper to init a short-lived manager (1h TTL) and assign to global registry.
func initTestManager(t *testing.T) *crypto.Manager {
	km, err := crypto.NewManager(time.Hour)
	if err != nil {
		t.Fatalf("manager init: %v", err)
	}
	crypto.GlobalEdDSARegistry = km
	return km
}

func TestRevocationChainSigningAndVerification(t *testing.T) {
	initTestManager(t)
	rc := NewRevocationChain()
	e1, err := rc.Append(RevocationEvent{ID: "rev-1", DelegationID: "del-1"})
	if err != nil {
		t.Fatalf("append e1: %v", err)
	}
	if e1.Signature == "" || e1.SigKid == "" {
		t.Fatalf("expected signature fields populated")
	}
	if err := rc.Verify(); err != nil {
		t.Fatalf("verify failed: %v", err)
	}
}

func TestRevocationChainSignatureTamper(t *testing.T) {
	initTestManager(t)
	rc := NewRevocationChain()
	_, _ = rc.Append(RevocationEvent{ID: "rev-1", DelegationID: "del-1"})
	_, _ = rc.Append(RevocationEvent{ID: "rev-2", DelegationID: "del-2"})
	// Tamper with second event delegation id (changes hash => signature invalid)
	rc.events[1].DelegationID = "del-2-tamper"
	if err := rc.Verify(); err == nil {
		t.Fatalf("expected verification failure after tampering content")
	}
}

func TestRevocationChainSignatureRemoval(t *testing.T) {
	initTestManager(t)
	rc := NewRevocationChain()
	_, _ = rc.Append(RevocationEvent{ID: "rev-1", DelegationID: "del-1"})
	// Remove signature fields (legacy mode) should still verify for hash chain but event loses signature validation.
	rc.events[0].Signature = ""
	rc.events[0].SigKid = ""
	if err := rc.Verify(); err != nil {
		t.Fatalf("unexpected hash chain failure without signature: %v", err)
	}
	// Re-tamper hash content: modify delegation id -> should fail.
	rc.events[0].DelegationID = "other"
	if err := rc.Verify(); err == nil {
		t.Fatalf("expected hash chain failure after tamper")
	}
}

// Unknown kid verification should fail when the signing key is removed from the registry.
func TestRevocationChainUnknownKid(t *testing.T) {
	km := initTestManager(t)
	rc := NewRevocationChain()
	ev, err := rc.Append(RevocationEvent{ID: "rev-1", DelegationID: "del-1"})
	if err != nil {
		t.Fatalf("append: %v", err)
	}
	if ev.Signature == "" {
		t.Fatalf("expected signature present")
	}
	// Simulate key loss: remove the global manager entirely so verification cannot locate kid.
	crypto.GlobalEdDSARegistry = nil
	if err := rc.Verify(); err == nil {
		t.Fatalf("expected verification failure without key for kid %s", ev.SigKid)
	}
	// Restore original manager for other tests.
	crypto.GlobalEdDSARegistry = km
}

package delegation

import (
	"testing"
	"time"
)

func TestRevocationChainAppendAndVerify(t *testing.T) {
	rc := NewRevocationChain()
	// append first event
	e1, err := rc.Append(RevocationEvent{ID: "rev-1", DelegationID: "del-1", Reason: string(RevocationReasonCompromise)})
	if err != nil {
		t.Fatalf("append e1: %v", err)
	}
	if e1.PrevHash != "" {
		t.Fatalf("expected empty prev hash for genesis")
	}
	e2, err := rc.Append(RevocationEvent{ID: "rev-2", DelegationID: "del-2", Reason: string(RevocationReasonUserRequest)})
	if err != nil {
		t.Fatalf("append e2: %v", err)
	}
	if e2.PrevHash != e1.Hash {
		t.Fatalf("prev hash link incorrect")
	}
	if err := rc.Verify(); err != nil {
		t.Fatalf("verify chain: %v", err)
	}
}

func TestRevocationReasonValidationAndAggregateHash(t *testing.T) {
	rc := NewRevocationChain()
	// Unknown reason should fallback to revoked_by_grantor
	e1, err := rc.Append(RevocationEvent{ID: "rev-x", DelegationID: "del-x", Reason: "unknown_code"})
	if err != nil {
		t.Fatalf("append: %v", err)
	}
	if e1.Reason != string(RevocationReasonGrantorRevoked) {
		t.Fatalf("unexpected fallback reason: %s", e1.Reason)
	}
	// Add a valid reason
	e2, err := rc.Append(RevocationEvent{ID: "rev-y", DelegationID: "del-y", Reason: string(RevocationReasonCompromise)})
	if err != nil {
		t.Fatalf("append2: %v", err)
	}
	if e2.Reason != string(RevocationReasonCompromise) {
		t.Fatalf("reason mismatch: %s", e2.Reason)
	}
	if err := rc.Verify(); err != nil {
		t.Fatalf("verify: %v", err)
	}
	agg := rc.AggregateHash()
	if agg == "" {
		t.Fatalf("expected aggregate hash")
	}
	// Recompute should match (idempotent)
	agg2 := rc.AggregateHash()
	if agg != agg2 {
		t.Fatalf("aggregate hash not deterministic")
	}
}

func TestRevocationChainTamperDetect(t *testing.T) {
	rc := NewRevocationChain()
	_, _ = rc.Append(RevocationEvent{ID: "rev-1", DelegationID: "del-1"})
	_, _ = rc.Append(RevocationEvent{ID: "rev-2", DelegationID: "del-2"})
	// Tamper with stored event delegation id (which should change hash)
	rc.events[1].DelegationID = "del-2-tampered"
	if err := rc.Verify(); err == nil {
		t.Fatalf("expected verification error after tampering")
	}
}

func TestValidateDelegationChainWithRevocations(t *testing.T) {
	// build delegation chain
	c := NewChain()
	d1, err := c.Append(Delegation{ID: "del-1", Subject: "alice", Delegate: "bob", Scope: map[string]string{"action": "read"}, ExpiresAt: time.Now().Add(time.Hour)})
	if err != nil {
		t.Fatalf("append d1: %v", err)
	}
	_, err = c.Append(Delegation{ID: "del-2", Subject: "alice", Delegate: "carol", Scope: map[string]string{"action": "read"}, ExpiresAt: time.Now().Add(time.Hour)})
	if err != nil {
		t.Fatalf("append d2: %v", err)
	}

	rc := NewRevocationChain()
	// revoke first delegation
	_, err = rc.Append(RevocationEvent{ID: "rev-1", DelegationID: d1.ID, Reason: "admin revoke"})
	if err != nil {
		t.Fatalf("append revocation: %v", err)
	}

	if err := ValidateDelegationChainWithRevocations(c, rc); err == nil {
		t.Fatalf("expected revocation validation failure for revoked delegation")
	}
}

func TestValidateDelegationChainWithRevocationsNoRevoked(t *testing.T) {
	c := NewChain()
	_, err := c.Append(Delegation{ID: "del-1", Subject: "alice", Delegate: "bob", Scope: map[string]string{"action": "read"}, ExpiresAt: time.Now().Add(time.Hour)})
	if err != nil {
		t.Fatalf("append d1: %v", err)
	}
	rc := NewRevocationChain()
	if err := ValidateDelegationChainWithRevocations(c, rc); err != nil {
		t.Fatalf("unexpected validation error: %v", err)
	}
}

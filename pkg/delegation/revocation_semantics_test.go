package delegation

import (
	"testing"
	"time"
)

func TestRevocationSemantics_RFC115_C10(t *testing.T) {
	// 1. Basic Revocation Check
	t.Run("IsDelegationRevoked", func(t *testing.T) {
		rc := NewRevocationChain()
		event := RevocationEvent{
			ID:           "rev-1",
			DelegationID: "del-123",
			Reason:       "compromise",
		}
		if _, err := rc.Append(event); err != nil {
			t.Fatalf("Failed to append revocation: %v", err)
		}

		if !rc.IsDelegationRevoked("del-123", "") {
			t.Error("Expected delegation 'del-123' to be revoked")
		}
		if rc.IsDelegationRevoked("del-456", "") {
			t.Error("Did not expect 'del-456' to be revoked")
		}
	})

	// 2. Chain Validation with Revocation
	t.Run("ValidateChainWithRevocation", func(t *testing.T) {
		// Use public helpers to build a valid chain
		c := NewChain()
		d1 := Delegation{
			ID:        "del-root",
			Subject:   "did:example:alice",
			Delegate:  "did:example:bob",
			ExpiresAt: time.Now().Add(1 * time.Hour),
		}
		if _, err := c.Append(d1); err != nil {
			t.Fatalf("failed to append d1: %v", err)
		}

		d2 := Delegation{
			ID:        "del-leaf",
			Subject:   "did:example:bob",
			Delegate:  "did:example:carol",
			ExpiresAt: time.Now().Add(1 * time.Hour),
		}
		if _, err := c.Append(d2); err != nil {
			t.Fatalf("failed to append d2: %v", err)
		}

		// Verify valid chain passes empty revocation
		rcEmpty := NewRevocationChain()
		if err := ValidateDelegationChainWithRevocations(c, rcEmpty); err != nil {
			t.Errorf("Valid chain failed with empty revocation: %v", err)
		}

		// Verify revocation of leaf
		rc := NewRevocationChain()
		rc.Append(RevocationEvent{ID: "r1", DelegationID: "del-leaf", Reason: "abuse"})

		if err := ValidateDelegationChainWithRevocations(c, rc); err == nil {
			t.Error("Expected error for revoked delegation 'del-leaf'")
		} else {
			// Check error message or type if possible, but presence of error is key C10 compliance
			// (revoked chain MUST NOT validate)
			// t.Logf("Got expected error: %v", err)
		}

		// Verify revocation of root (should also fail)
		rc2 := NewRevocationChain()
		rc2.Append(RevocationEvent{ID: "r2", DelegationID: "del-root", Reason: "compromise"})
		if err := ValidateDelegationChainWithRevocations(c, rc2); err == nil {
			t.Error("Expected error for revoked delegation 'del-root'")
		}
	})

	// Let's add a robust test for IsDelegationRevoked with diverse inputs (Hash vs ID)

	rc2 := NewRevocationChain()
	rc2.Append(RevocationEvent{ID: "r2", DelegationHash: "hash-123", Reason: "superseded"})

	if !rc2.IsDelegationRevoked("", "hash-123") {
		t.Error("Expected revocation by hash")
	}
	if rc2.IsDelegationRevoked("some-id", "hash-123") {
		// Should return true if hash matches even if ID provided (OR logic usually)
		// Implementation:
		/*
			if delegationID != "" && e.DelegationID == delegationID { return true }
			if delegationHash != "" && e.DelegationHash == delegationHash { return true }
		*/
		// Yes.
	} else {
		t.Error("Expected revocation when hash matches")
	}
}

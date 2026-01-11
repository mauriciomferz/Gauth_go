package delegation

import (
	"testing"
	"time"
)

func TestRevocationChain_Integrity(t *testing.T) {
	chain := NewRevocationChain()

	// 1. Append first event
	ev1 := RevocationEvent{
		ID:           "rev-1",
		DelegationID: "del-1",
		Reason:       "compromise",
	}
	added1, err := chain.Append(ev1)
	if err != nil {
		t.Fatalf("Append failed: %v", err)
	}

	if added1.Hash == "" {
		t.Error("Hash not computed")
	}
	if added1.PrevHash != "" {
		t.Error("Genesis event should not have PrevHash")
	}

	// 2. Append second event
	ev2 := RevocationEvent{
		ID:           "rev-2",
		DelegationID: "del-2",
		Reason:       "user_request",
	}
	added2, err := chain.Append(ev2)
	if err != nil {
		t.Fatalf("Append 2 failed: %v", err)
	}

	if added2.PrevHash != added1.Hash {
		t.Errorf("Linkage broken: expected prevHash %s, got %s", added1.Hash, added2.PrevHash)
	}

	// 3. Verify Chain
	if err := chain.Verify(); err != nil {
		t.Errorf("Verify failed for valid chain: %v", err)
	}

	// 4. Test Lookup
	if !chain.IsDelegationRevoked("del-1", "") {
		t.Error("del-1 should be revoked")
	}
	if !chain.IsDelegationRevoked("del-2", "") {
		t.Error("del-2 should be revoked")
	}
	if chain.IsDelegationRevoked("del-3", "") {
		t.Error("del-3 should NOT be revoked")
	}
}

func TestRevocationChain_TamperDetection(t *testing.T) {
	// Setup valid chain
	chain := NewRevocationChain()
	if _, err := chain.Append(RevocationEvent{ID: "r1", DelegationID: "d1"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := chain.Append(RevocationEvent{ID: "r2", DelegationID: "d2"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	validEvents := chain.Events()

	// Case A: Tamper with content (Reason) without updating Hash
	tamperedChainA := NewRevocationChain()
	eventsA := make([]RevocationEvent, len(validEvents))
	copy(eventsA, validEvents)
	eventsA[0].Reason = "malicious_change"
	tamperedChainA.events = eventsA

	err := tamperedChainA.Verify()
	if err == nil {
		t.Error("Verify should fail on content tamper")
	} else if err.Error() != "revocation event hash mismatch at 0" {
		t.Errorf("Unexpected error msg: %v", err)
	}

	// Case B: Tamper with Hash matches content, but breaks linkage
	// We re-calculate hash for r1, but r2 still points to old r1 hash
	tamperedChainB := NewRevocationChain()
	eventsB := make([]RevocationEvent, len(validEvents))
	copy(eventsB, validEvents)

	// manually rehash r1
	eventsB[0].Reason = "malicious_change"
	newHash, _ := hashRevocationEvent(eventsB[0])
	eventsB[0].Hash = newHash

	tamperedChainB.events = eventsB

	err = tamperedChainB.Verify()
	if err == nil {
		t.Error("Verify should fail on broken linkage")
	} else if err.Error() != "broken revocation prev hash link at 1" {
		t.Errorf("Unexpected error msg: %v", err)
	}
}

func TestRevocationChain_FutureTimestamp(t *testing.T) {
	chain := NewRevocationChain()
	futureEv := RevocationEvent{
		ID:           "future-1",
		DelegationID: "d1",
		RevokedAt:    time.Now().Add(24 * time.Hour),
	}

	// Bypass Append validation by manually constructing (Append sets time to Now)
	// But wait, Append sets RevokedAt to Now().UTC(). We can't easily force future time via Append public API.
	// We have to inject it manually to test Verify.

	// Calculate hash manually
	h, _ := hashRevocationEvent(futureEv)
	futureEv.Hash = h

	chain.events = append(chain.events, futureEv)

	err := chain.Verify()
	if err == nil {
		t.Error("Verify should fail on future timestamp")
	}
}

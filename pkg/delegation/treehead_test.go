package delegation

import (
	"encoding/base64"
	"fmt"
	"testing"
	"time"

	"github.com/mauriciomferz/Gauth_go/pkg/crypto"
)

// TestSignTreeHeadBasic ensures signing produces a signature when key manager active.
func TestSignTreeHeadBasic(t *testing.T) {
	// Initialize ephemeral key manager for test isolation.
	km, err := crypto.NewManager(1 * time.Hour)
	if err != nil {
		t.Fatalf("km init: %v", err)
	}
	crypto.GlobalEdDSARegistry = km
	rc := NewRevocationChain()
	// Append a couple events to make non-empty merkle root
	_, _ = rc.Append(RevocationEvent{ID: "r1", DelegationID: "d1"})
	_, _ = rc.Append(RevocationEvent{ID: "r2", DelegationID: "d2"})
	sth, err := rc.SignTreeHead()
	if err != nil {
		t.Fatalf("sign tree head error: %v", err)
	}
	if sth.MerkleRoot == "" {
		t.Fatalf("expected merkle root non-empty")
	}
	if len(sth.Signatures) == 0 {
		t.Fatalf("expected at least one signature")
	}
	// Basic signature format check
	if _, err := base64.RawURLEncoding.DecodeString(sth.Signatures[0].Sig); err != nil {
		t.Fatalf("invalid signature encoding: %v", err)
	}
	if sth.Signatures[0].Alg != "EdDSA" {
		t.Fatalf("unexpected alg %s", sth.Signatures[0].Alg)
	}
}

// TestVerifyTreeHeadSignature validates verification helper.
func TestVerifyTreeHeadSignature(t *testing.T) {
	km, err := crypto.NewManager(1 * time.Hour)
	if err != nil {
		t.Fatalf("km init: %v", err)
	}
	crypto.GlobalEdDSARegistry = km
	rc := NewRevocationChain()
	_, _ = rc.Append(RevocationEvent{ID: "ra", DelegationID: "da"})
	sth, err := rc.SignTreeHead()
	if err != nil {
		t.Fatalf("sign error: %v", err)
	}
	if err := VerifyTreeHeadSignature(sth, km); err != nil {
		t.Fatalf("verify signature failed: %v", err)
	}
	// Tamper merkle root
	sth.MerkleRoot = deadbeefValue
	if err := VerifyTreeHeadSignature(sth, km); err == nil {
		t.Fatalf("expected failure after tamper")
	}
}

// TestConsistencyProof builds two signed tree heads and generates/ verifies consistency proof.
func TestConsistencyProof(t *testing.T) {
	km, err := crypto.NewManager(1 * time.Hour)
	if err != nil {
		t.Fatalf("km init: %v", err)
	}
	crypto.GlobalEdDSARegistry = km
	rc := NewRevocationChain()
	// append initial events
	for i := 1; i <= 3; i++ {
		_, _ = rc.Append(RevocationEvent{ID: fmt.Sprintf("e%d", i), DelegationID: fmt.Sprintf("d%d", i)})
	}
	sth1, err := rc.SignTreeHead()
	if err != nil {
		t.Fatalf("sign1: %v", err)
	}
	// append more
	for i := 4; i <= 7; i++ {
		_, _ = rc.Append(RevocationEvent{ID: fmt.Sprintf("e%d", i), DelegationID: fmt.Sprintf("d%d", i)})
	}
	sth2, err := rc.SignTreeHead()
	if err != nil {
		t.Fatalf("sign2: %v", err)
	}
	if sth1.ChainLength >= sth2.ChainLength {
		t.Fatalf("expected growth")
	}
	proof, err := rc.GenerateConsistencyProof(0)
	if err != nil {
		t.Fatalf("generate proof: %v", err)
	}
	// Gather all event hashes
	allHashes := []string{}
	for _, ev := range rc.Events() {
		allHashes = append(allHashes, ev.Hash)
	}
	if err := VerifyConsistencyProof(proof, allHashes); err != nil {
		t.Fatalf("verify consistency failed: %v", err)
	}
	// Tamper new leaves
	proof.NewLeaves[0] = deadbeefValue
	if err := VerifyConsistencyProof(proof, allHashes); err == nil {
		t.Fatalf("expected failure after tamper")
	}
}

// GlobalKeyManagerUnavailable detects absence of global eddsa registry.
// Previous helper removed; direct manager initialization used per test.

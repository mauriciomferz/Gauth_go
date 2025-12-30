package test

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"

	auditor "github.com/mauriciomferz/AgentAuth/pkg/auditor"
	"github.com/mauriciomferz/AgentAuth/pkg/delegation"
)

// helper to produce deterministic event hash
func h(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

func TestRevocationProofVerification(t *testing.T) {
	// Build synthetic merkle tree with 5 events
	tree := delegation.NewMerkleTree()
	events := []string{h("event0"), h("event1"), h("event2"), h("event3"), h("event4")}
	for _, evh := range events {
		tree.AppendLeaf(evh)
	}
	// Generate proof for index 2
	proof, root, err := tree.GenerateProof(2)
	if err != nil {
		t.Fatalf("generate proof: %v", err)
	}
	res := auditor.VerifyRevocationProof(events[2], proof, root)
	if !res.Included || res.Reason != "" {
		t.Fatalf("expected inclusion true with empty reason; got %+v", res)
	}
	if res.LeafDigest == "" {
		t.Fatalf("leaf digest empty")
	}
	// Tamper first proof step sibling
	bad := make([]delegation.MerkleProofStep, len(proof))
	copy(bad, proof)
	bad[0].Sibling = h("tamper") // different digest
	resBad := auditor.VerifyRevocationProof(events[2], bad, root)
	if resBad.Included || resBad.Reason == "" {
		t.Fatalf("expected failure; got %+v", resBad)
	}
}

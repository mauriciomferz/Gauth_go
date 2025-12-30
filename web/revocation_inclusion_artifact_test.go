package web

import (
	"encoding/json"
	"os"
	"testing"
)

// Structures matching the example artifact (only fields we assert on).
type revocationInclusionArtifact struct {
	Version         int    `json:"version"`
	GeneratedAt     string `json:"generated_at"`
	RevocationEvent struct {
		ID           string `json:"id"`
		DelegationID string `json:"delegation_id"`
		Reason       string `json:"reason"`
		Hash         string `json:"hash"`
	} `json:"revocation_event"`
	MerkleProof struct {
		LeafHash     string   `json:"leaf_hash"`
		Siblings     []string `json:"siblings"`
		Index        int      `json:"index"`
		TreeSize     int      `json:"tree_size"`
		ComputedRoot string   `json:"computed_root"`
	} `json:"merkle_proof"`
	SignedTreeHead struct {
		ChainLength     int    `json:"chain_length"`
		MerkleRoot      string `json:"merkle_root"`
		AggregateHash   string `json:"aggregate_hash"`
		GeneratedAt     string `json:"generated_at"`
		SatisfiedWeight int    `json:"satisfied_weight"`
		Threshold       int    `json:"threshold"`
		Signatures      []struct {
			KID       string `json:"kid"`
			Mode      string `json:"mode"`
			Signature string `json:"signature"`
		} `json:"signatures"`
	} `json:"signed_tree_head"`
	RFCRefs []string `json:/aap_refs"`
}

// TestExampleRevocationInclusionArtifactConsistency validates structural consistency
// of the example inclusion proof artifact. Full cryptographic verification is not
// performed because sibling hashes are truncated placeholders. This test ensures
// future edits preserve required semantic fields and internal consistency.
func TestExampleRevocationInclusionArtifactConsistency(t *testing.T) {
	// Resolve artifact relative to repository root (tests run from package dir)
	path := "../examples/revocation_inclusion_proof.json"
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read artifact: %v", err)
		return
	}
	var art revocationInclusionArtifact
	if err := json.Unmarshal(raw, &art); err != nil {
		t.Fatalf("unmarshal artifact: %v", err)
	}
	if art.Version != 1 {
		t.Fatalf("expected version=1 got %d", art.Version)
	}
	if art.RevocationEvent.ID == "" || art.RevocationEvent.Hash == "" {
		t.Fatalf("revocation_event missing id/hash")
	}
	if art.MerkleProof.LeafHash == "" || art.MerkleProof.ComputedRoot == "" {
		t.Fatalf("merkle_proof missing leaf/computed root")
	}
	if art.MerkleProof.ComputedRoot != art.SignedTreeHead.MerkleRoot {
		t.Fatalf("computed_root mismatch signed_tree_head.merkle_root")
	}
	if art.SignedTreeHead.Threshold != art.SignedTreeHead.SatisfiedWeight {
		t.Fatalf("threshold != satisfied_weight (expected equal for single-sig demo)")
	}
	if len(art.SignedTreeHead.Signatures) == 0 {
		t.Fatalf("expected at least one signature")
	}
	if len(art.RFCRefs) == 0 {
		t.Fatalf("expected/aap_refs to be present")
	}
	// Ensure placeholder siblings are present but we don't enforce count beyond non-empty.
	if len(art.MerkleProof.Siblings) == 0 {
		t.Fatalf("expected at least one sibling hash placeholder")
	}
}

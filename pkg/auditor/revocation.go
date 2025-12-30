package auditor

import (
	"fmt"

	"github.com/mauriciomferz/AgentAuth/pkg/delegation"
)

// RevocationProofResult captures inclusion verification outcome for a single revocation event.
type RevocationProofResult struct {
	EventHash  string `json:"event_hash"`
	LeafDigest string `json:"leaf_digest"`
	MerkleRoot string `json:"merkle_root"`
	Steps      int    `json:"steps"`
	Included   bool   `json:"included"`
	Reason     string `json:"reason,omitempty"`
}

// VerifyRevocationProof verifies Merkle inclusion of a revocation event given its raw event hash, proof steps and expected root.
// The eventHash must be the original revocation event Hash (hex sha256) prior to leaf domain separation.
// Returns a RevocationProofResult detailing success/failure and diagnostic reason on failure.
func VerifyRevocationProof(eventHash string, proof []delegation.MerkleProofStep, expectedRoot string) *RevocationProofResult {
	res := &RevocationProofResult{EventHash: eventHash, MerkleRoot: expectedRoot, Steps: len(proof)}
	if eventHash == "" {
		res.Reason = "empty_event_hash"
		return res
	}
	if expectedRoot == "" {
		res.Reason = "empty_merkle_root"
		return res
	}
	if len(proof) == 0 {
		res.Reason = "empty_proof"
		return res
	}
	leaf := delegation.LeafDigestForEventHash(eventHash)
	res.LeafDigest = leaf
	if delegation.VerifyProof(leaf, proof, expectedRoot) {
		res.Included = true
		return res
	}
	res.Included = false
	res.Reason = "verification_failed"
	// Lightweight heuristic diagnostics: check for malformed position values.
	for i, step := range proof {
		if step.Position != "L" && step.Position != "R" {
			res.Reason = fmt.Sprintf("invalid_position_at_%d", i)
			break
		}
	}
	return res
}

package delegation

// Merkle accumulator for revocation events (Phase 3 prototype).
// Design goals:
// - Append-only tree (no deletions) over event Hash values (hex SHA-256 strings)
// - Efficient proof generation for any leaf (O(log n))
// - Stable root after each append exposed for external anchoring
// - Deterministic layer ordering (left-to-right) and domain-separated hashing
// - Compact proof structure: list of (siblingHash, position) pairs; position is 'L' or 'R'
// - Verification given leaf hash, proof, and expected root
//
// Hash function: SHA-256 over prefix + concatenation of child digests:
//   leaf:   SHA256("GAUTH_MERKLE_LEAF:" + hexHash)
//   parent: SHA256("GAUTH_MERKLE_NODE:" + left + right)  (left/right are raw hex strings of child nodes)
// We retain hex strings throughout for consistency with existing event Hash.

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
)

// merkleNode holds computed digest as hex string.
type merkleNode struct{ digest string }

// MerkleTree maintains leaves and cached levels.
type MerkleTree struct {
	leaves []merkleNode
	// levels[i] is slice of nodes at height i (0 = leaves). Rebuilt lazily.
	levels [][]merkleNode
	dirty  bool
}

// NewMerkleTree constructs empty tree.
func NewMerkleTree() *MerkleTree { return &MerkleTree{leaves: make([]merkleNode, 0)} }

// AppendLeaf adds a new leaf from raw event hash (hex string) and marks tree dirty.
func (t *MerkleTree) AppendLeaf(eventHash string) {
	h := sha256.Sum256(append([]byte("GAUTH_MERKLE_LEAF:"), []byte(eventHash)...))
	t.leaves = append(t.leaves, merkleNode{digest: hex.EncodeToString(h[:])})
	t.dirty = true
}

// Root returns current Merkle root (empty string if no leaves).
func (t *MerkleTree) Root() string {
	if len(t.leaves) == 0 {
		return ""
	}
	t.rebuildIfNeeded()
	top := t.levels[len(t.levels)-1]
	if len(top) == 0 {
		return ""
	}
	return top[0].digest
}

// rebuildIfNeeded builds levels from leaves if dirty.
func (t *MerkleTree) rebuildIfNeeded() {
	if !t.dirty {
		return
	}
	// Initialize level 0 (leaves)
	levels := make([][]merkleNode, 0)
	levels = append(levels, t.leaves)
	cur := t.leaves
	for {
		if len(cur) == 0 {
			break
		}
		if len(cur) == 1 {
			break
		} // single root
		// Next level
		next := make([]merkleNode, 0, (len(cur)+1)/2)
		for i := 0; i < len(cur); i += 2 {
			if i+1 == len(cur) {
				// odd count: promote last directly (copy-up strategy ensures determinism)
				next = append(next, cur[i])
				continue
			}
			left := cur[i].digest
			right := cur[i+1].digest
			parent := sha256.Sum256(append(append([]byte("GAUTH_MERKLE_NODE:"), []byte(left)...), []byte(right)...))
			next = append(next, merkleNode{digest: hex.EncodeToString(parent[:])})
		}
		levels = append(levels, next)
		cur = next
	}
	t.levels = levels
	t.dirty = false
}

// MerkleProofStep holds sibling digest and position relative to current hash.
type MerkleProofStep struct {
	Sibling  string `json:"sibling"`
	Position string `json:"position"` // "L" or "R" (sibling position relative to evolving hash)
}

// GenerateProof returns proof steps for leaf index (0-based) or error if out of range.
func (t *MerkleTree) GenerateProof(leafIndex int) ([]MerkleProofStep, string, error) {
	if leafIndex < 0 || leafIndex >= len(t.leaves) {
		return nil, "", errors.New("leaf_index_out_of_range")
	}
	t.rebuildIfNeeded()
	proof := []MerkleProofStep{}
	// Start from leaf level upward
	idx := leafIndex
	for level := 0; level < len(t.levels)-1; level++ {
		nodes := t.levels[level]
		// If odd last and idx is that last, it was promoted—no sibling at this level.
		if idx == len(nodes)-1 && len(nodes)%2 == 1 {
			// promoted node: just move to next level index = idx/2
			idx = idx / 2
			continue
		}
		// Determine sibling
		var siblingIdx int
		var pos string
		if idx%2 == 0 { // left node
			siblingIdx = idx + 1
			pos = "R" // sibling is right of current hash
		} else {
			siblingIdx = idx - 1
			pos = "L" // sibling is left of current hash
		}
		if siblingIdx < len(nodes) {
			proof = append(proof, MerkleProofStep{Sibling: nodes[siblingIdx].digest, Position: pos})
		}
		idx = idx / 2 // move up
	}
	return proof, t.Root(), nil
}

// VerifyProof verifies inclusion of leafDigest (already leaf-form digest) given proof steps and expected root.
// leafDigest should be the post-leaf-domain digest (not raw event hash). Caller obtains from tree or recompute.
func VerifyProof(leafDigest string, proof []MerkleProofStep, expectedRoot string) bool {
	if expectedRoot == "" {
		return false
	}
	cur := leafDigest
	for _, step := range proof {
		if step.Position == "R" { // sibling is right of cur: parent = HASH(cur + sibling)
			parent := sha256.Sum256(append(append([]byte("GAUTH_MERKLE_NODE:"), []byte(cur)...), []byte(step.Sibling)...))
			cur = hex.EncodeToString(parent[:])
		} else if step.Position == "L" { // sibling is left: parent = HASH(sibling + cur)
			parent := sha256.Sum256(append(append([]byte("GAUTH_MERKLE_NODE:"), []byte(step.Sibling)...), []byte(cur)...))
			cur = hex.EncodeToString(parent[:])
		} else {
			return false
		}
	}
	return cur == expectedRoot
}

// LeafDigestForEventHash returns leaf-domain digest for given event hash (helper for external verification)
func LeafDigestForEventHash(eventHash string) string {
	h := sha256.Sum256(append([]byte("GAUTH_MERKLE_LEAF:"), []byte(eventHash)...))
	return hex.EncodeToString(h[:])
}

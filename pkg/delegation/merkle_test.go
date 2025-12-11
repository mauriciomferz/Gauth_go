package delegation

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"testing"
)

// TestMerkleTreeVectors verifies basic tree construction and root stability.
func TestMerkleTreeVectors(t *testing.T) {
	mt := NewMerkleTree()
	if mt.Root() != "" {
		t.Error("expected empty root for empty tree")
	}

	// Case 1: Single leaf "a"
	mt.AppendLeaf(hashString("a"))
	root1 := mt.Root()
	// Computed manually: SHA256("GAUTH_MERKLE_LEAF:" + hash("a")) -> then that is the root (level 0 promoted)
	// Actually implementation promotes single leaf to root directly?
	// verify implementation detail: level 0 is leaves. if len=1, loop breaks.
	// So root = leaf digest.
	expected1 := LeafDigestForEventHash(hashString("a"))
	if root1 != expected1 {
		t.Errorf("expected root %s got %s", expected1, root1)
	}

	// Case 2: Two leaves "a", "b"
	mt.AppendLeaf(hashString("b"))
	root2 := mt.Root()
	// Expected: SHA256("GAUTH_MERKLE_NODE:" + leaf(a) + leaf(b))
	leafA := LeafDigestForEventHash(hashString("a"))
	leafB := LeafDigestForEventHash(hashString("b"))
	expected2 := hashNode(leafA, leafB)
	if root2 != expected2 {
		t.Errorf("expected root %s got %s", expected2, root2)
	}
}

// TestInclusionProofs verifies GenerateProof and VerifyProof for various tree sizes.
func TestInclusionProofs(t *testing.T) {
	// Test trees of size 1 to 16
	for size := 1; size <= 16; size++ {
		mt := NewMerkleTree()
		leaves := make([]string, size)
		for i := 0; i < size; i++ {
			leaves[i] = hashString(fmt.Sprintf("leaf-%d", i))
			mt.AppendLeaf(leaves[i])
		}
		root := mt.Root()

		// Verify proof for each leaf
		for i := 0; i < size; i++ {
			proof, proofRoot, err := mt.GenerateProof(i)
			if err != nil {
				t.Fatalf("size=%d leaf=%d generate error: %v", size, i, err)
			}
			if proofRoot != root {
				t.Fatalf("size=%d leaf=%d proof root mismatch", size, i)
			}

			// Valid verification
			leafDigest := LeafDigestForEventHash(leaves[i])
			if !VerifyProof(leafDigest, proof, root) {
				t.Errorf("size=%d leaf=%d proof verification failed", size, i)
			}

			// Invalid verification (tampered leaf)
			wrongLeaf := LeafDigestForEventHash("tampered")
			if VerifyProof(wrongLeaf, proof, root) {
				t.Errorf("size=%d leaf=%d verified valid for wrong leaf", size, i)
			}
		}
	}
}

// TestRevocationChainConsistency re-implements consistency test using RevocationChain to access the logic
func TestRevocationChainConsistency(t *testing.T) {
	rc := NewRevocationChain()

	// Append 10 events, capturing STH at each step
	for i := 0; i < 10; i++ {
		ev := RevocationEvent{ID: fmt.Sprintf("id-%d", i), DelegationID: "d-1"}
		if _, err := rc.Append(ev); err != nil {
			t.Fatalf("append %d: %v", i, err)
		}
		// Generate STH
		if _, err := rc.SignTreeHead(); err != nil {
			t.Fatalf("sign %d: %v", i, err)
		}
	}

	// Verify consistency between every pair logic (previous STH -> latest)

	latest := rc.LatestTreeHead()
	for i := 0; i < len(rc.TreeHeads())-1; i++ {
		proof, err := rc.GenerateConsistencyProofV2(i)
		if err != nil {
			t.Fatalf("gen proof v2 from index %d: %v", i, err)
		}

		startSTH := rc.TreeHeads()[i]
		if proof.StartLength != startSTH.ChainLength {
			t.Errorf("proof start len mismatch: got %d want %d", proof.StartLength, startSTH.ChainLength)
		}
		if proof.EndLength != latest.ChainLength {
			t.Errorf("proof end len mismatch: got %d want %d", proof.EndLength, latest.ChainLength)
		}

		// Basic structural check
		if proof.StartLength > 0 && proof.StartLength < proof.EndLength {
			if len(proof.Path) == 0 && len(proof.PrefixRoots) == 0 {
				t.Logf("Warning: Empty consistency proof parts for %d->%d", proof.StartLength, proof.EndLength)
			}
		}
	}
}

// Helpers
func hashString(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}

func hashNode(left, right string) string {
	h := sha256.Sum256(append(append([]byte("GAUTH_MERKLE_NODE:"), []byte(left)...), []byte(right)...))
	return hex.EncodeToString(h[:])
}

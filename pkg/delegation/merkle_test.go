package delegation

import "testing"

func TestMerkleTreeBasic(t *testing.T) {
	mt := NewMerkleTree()
	if mt.Root() != "" {
		t.Fatalf("expected empty root")
	}
	hashes := []string{"aa", "bb", "cc", "dd"}
	for _, h := range hashes {
		mt.AppendLeaf(h)
	}
	root := mt.Root()
	if root == "" {
		t.Fatalf("expected non-empty root")
	}
	// Proof first leaf
	proof, expectedRoot, err := mt.GenerateProof(0)
	if err != nil {
		t.Fatalf("proof error: %v", err)
	}
	leafDigest := LeafDigestForEventHash("aa")
	if !VerifyProof(leafDigest, proof, expectedRoot) {
		t.Fatalf("proof verification failed for first leaf")
	}
	// Proof last leaf
	proof2, expectedRoot2, err2 := mt.GenerateProof(len(hashes) - 1)
	if err2 != nil {
		t.Fatalf("proof error last: %v", err2)
	}
	if expectedRoot2 != expectedRoot {
		t.Fatalf("root mismatch across proofs")
	}
	leafDigest2 := LeafDigestForEventHash("dd")
	if !VerifyProof(leafDigest2, proof2, expectedRoot) {
		t.Fatalf("proof verification failed for last leaf")
	}
}

func TestMerkleOddPromotion(t *testing.T) {
	mt := NewMerkleTree()
	hashes := []string{"x1", "x2", "x3"} // odd count triggers promotion
	for _, h := range hashes {
		mt.AppendLeaf(h)
	}
	root := mt.Root()
	if root == "" {
		t.Fatalf("root expected")
	}
	for i, h := range hashes {
		proof, r, err := mt.GenerateProof(i)
		if err != nil {
			t.Fatalf("proof error idx=%d: %v", i, err)
		}
		if r != root {
			t.Fatalf("root mismatch for idx=%d", i)
		}
		if !VerifyProof(LeafDigestForEventHash(h), proof, root) {
			t.Fatalf("verification failed idx=%d", i)
		}
	}
}

func TestMerkleProofTamper(t *testing.T) {
	mt := NewMerkleTree()
	mt.AppendLeaf("a")
	mt.AppendLeaf("b")
	proof, root, err := mt.GenerateProof(0)
	if err != nil {
		t.Fatalf("proof gen: %v", err)
	}
	// Tamper: modify first sibling hash
	if len(proof) == 0 {
		t.Fatalf("expected at least one proof step")
	}
	proof[0].Sibling = deadbeefValue // invalid digest
	if VerifyProof(LeafDigestForEventHash("a"), proof, root) {
		t.Fatalf("expected tampered proof to fail")
	}
}

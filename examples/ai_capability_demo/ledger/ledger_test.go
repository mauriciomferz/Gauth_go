package ledger

import (
	"fmt"
	"testing"
)

func TestLedgerAppendAndRoot(t *testing.T) {
	l := New()
	if l.LatestRoot() != "" { t.Fatalf("expected empty root initially") }
	id, root := l.Append("decision", "payload1", "")
	if id != 0 { t.Fatalf("expected id 0 got %d", id) }
	if root == "" { t.Fatalf("expected non-empty root after first append") }
	id2, root2 := l.Append("decision", "payload2", "")
	if id2 != 1 { t.Fatalf("expected id 1 got %d", id2) }
	if root2 == root { t.Fatalf("expected root change after second append") }
}

func TestLedgerProofIntegrity(t *testing.T) {
	l := New()
	for i := 0; i < 5; i++ { l.Append("decision", "p"+string(rune('a'+i)), "") }
	proof, err := l.Proof(2)
	if err != nil { t.Fatalf("unexpected proof error: %v", err) }
	if proof.EntryID != 2 { t.Fatalf("expected entry id 2") }
	if len(proof.Siblings) == 0 { t.Fatalf("expected non-empty siblings path") }
	if proof.Root != l.LatestRoot() { t.Fatalf("proof root mismatch latest") }
	if len(proof.Orientations) != len(proof.Siblings) { t.Fatalf("orientations length mismatch siblings") }
	// Recompute Merkle root from proof
	// We lack explicit orientation; current construction always pairs deterministic left/right.
	// We emulate tree rebuild by reconstructing layers starting from all leaves to validate.
	// (Simpler approach: recompute full root and compare; ensures proof logical consistency indirectly.)
	fullRoot := recomputeFullRootFromLedger(l)
	if fullRoot != proof.Root { t.Fatalf("recomputed full root mismatch: got %s want %s", fullRoot, proof.Root) }
}

func TestLedgerProofOrientationRebuild(t *testing.T) {
	l := New()
	for i := 0; i < 6; i++ { l.Append("decision", fmt.Sprintf("entry-%d", i), "") }
	proof, err := l.Proof(5)
	if err != nil { t.Fatalf("proof error: %v", err) }
	// Rebuild root using orientations + siblings starting from leaf
	leaf := leafHash(l.entries[5].Digest)
	acc := leaf
	for i, sib := range proof.Siblings {
		if proof.Orientations[i] == "L" { // node was left child; hash(node,left->right)
			acc = hash(acc, sib)
		} else { // node was right child; parent = hash(sibling,left-right ordering = sibling||acc)
			acc = hash(sib, acc)
		}
	}
	if acc != proof.Root { t.Fatalf("orientation rebuild root mismatch got %s want %s", acc, proof.Root) }
}

// recomputeFullRootFromLedger duplicates logic in computeRoot for test verification
func recomputeFullRootFromLedger(l *Ledger) string {
	if l.Size() == 0 { return "" }
	l.mu.RLock(); defer l.mu.RUnlock()
	layer := make([]string, len(l.entries))
	for i, e := range l.entries { layer[i] = leafHash(e.Digest) }
	for len(layer) > 1 {
		var next []string
		for i := 0; i < len(layer); i += 2 {
			if i+1 == len(layer) { next = append(next, hash(layer[i], layer[i])) } else { next = append(next, hash(layer[i], layer[i+1])) }
		}
		layer = next
	}
	return layer[0]
}

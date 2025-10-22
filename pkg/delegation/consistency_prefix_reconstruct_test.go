package delegation


import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/rand"
	"os"
	"testing"
	"time"
)

const deadbeefValue = "deadbeef"

// buildPrefixDecomposition replicates generation logic for tests (power-of-two blocks left to right).
func buildPrefixDecomposition(events []RevocationEvent, startLen int) ([]string, []int) {
	tmpTree := NewMerkleTree()
	for _, ev := range events {
		tmpTree.AppendLeaf(ev.Hash)
	}
	tmpTree.rebuildIfNeeded()
	remaining := startLen
	offset := 0
	roots := []string{}
	sizes := []int{}
	largestPow2 := func(n int) int {
		p := 1
		for p<<1 <= n {
			p = p << 1
		}
		return p
	}
	for remaining > 0 {
		blk := largestPow2(remaining)
		k := 0
		for (1 << k) < blk {
			k++
		}
		levelNodes := tmpTree.levels[k]
		idx := offset >> k
		roots = append(roots, levelNodes[idx].digest)
		sizes = append(sizes, blk)
		offset += blk
		remaining -= blk
	}
	return roots, sizes
}

func TestFastReconstructionMatchesFullTree(t *testing.T) {
	os.Setenv("GAUTH_CONSISTENCY_V2_FAST", "1")
	defer os.Unsetenv("GAUTH_CONSISTENCY_V2_FAST")
	for n := 1; n <= 50; n++ {
		c := NewRevocationChain()
		for i := 0; i < n; i++ {
			id := fmt.Sprintf("X%d", i)
			c.Append(RevocationEvent{ID: id, DelegationID: id})
		}
		roots, sizes := buildPrefixDecomposition(c.Events(), n)
		fullTree := NewMerkleTree()
		for i := 0; i < n; i++ {
			fullTree.AppendLeaf(c.Events()[i].Hash)
		}
		startRoot := fullTree.Root()
		bridges := buildBridges(roots, sizes)
		fast := ReconstructStartRootFromPrefixBlocks(roots, sizes, n, bridges)
		if fast == "" {
			t.Fatalf("expected fast reconstruction for n=%d", n)
		}
		if fast != startRoot {
			t.Fatalf("mismatch for n=%d: expected %s got %s", n, startRoot, fast)
		}
	}
}

func TestFastReconstructionRejectsBadSizes(t *testing.T) {
	os.Setenv("GAUTH_CONSISTENCY_V2_FAST", "1")
	defer os.Unsetenv("GAUTH_CONSISTENCY_V2_FAST")
	// invalid size (3 not power-of-two) should fail despite single root because prefixSizes invariants violated
	bad := ReconstructStartRootFromPrefixBlocks([]string{"abc"}, []int{3}, 3, nil)
	if bad != "" {
		t.Fatal("expected empty for invalid decomposition size")
	}
}

func TestFastReconstructionRandomSizes(t *testing.T) {
	os.Setenv("GAUTH_CONSISTENCY_V2_FAST", "1")
	defer os.Unsetenv("GAUTH_CONSISTENCY_V2_FAST")
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := 0; i < 200; i++ {
		n := 1 + rng.Intn(200)
		c := NewRevocationChain()
		for j := 0; j < n; j++ {
			id := fmt.Sprintf("R%d_%d", i, j)
			c.Append(RevocationEvent{ID: id, DelegationID: id})
		}
		roots, sizes := buildPrefixDecomposition(c.Events(), n)
		bridges := buildBridges(roots, sizes)
		full := NewMerkleTree()
		for j := 0; j < n; j++ {
			full.AppendLeaf(c.Events()[j].Hash)
		}
		startRoot := full.Root()
		fast := ReconstructStartRootFromPrefixBlocks(roots, sizes, n, bridges)
		if fast == "" {
			t.Fatalf("expected fast reconstruction for n=%d", n)
		}
		if fast != startRoot {
			t.Fatalf("mismatch fast vs full for n=%d", n)
		}
	}
}

// buildBridges replicates prefix bridge computation from GenerateConsistencyProofV2 for isolated tests
func buildBridges(prefixRoots []string, prefixSizes []int) []string {
	if len(prefixRoots) <= 1 {
		return nil
	}
	type seg struct {
		h  string
		sz int
	}
	blocks := make([]seg, len(prefixRoots))
	for i := range prefixRoots {
		blocks[i] = seg{h: prefixRoots[i], sz: prefixSizes[i]}
	}
	bridges := []string{}
	for len(blocks) > 1 {
		last := blocks[len(blocks)-1]
		prev := blocks[len(blocks)-2]
		parent := sha256.Sum256(append(append([]byte("GAUTH_MERKLE_NODE:"), []byte(prev.h)...), []byte(last.h)...))
		phex := hex.EncodeToString(parent[:])
		bridges = append(bridges, phex)
		blocks[len(blocks)-2] = seg{h: phex, sz: prev.sz + last.sz}
		blocks = blocks[:len(blocks)-1]
		for len(blocks) >= 2 {
			a := blocks[len(blocks)-2]
			b := blocks[len(blocks)-1]
			if a.sz != b.sz {
				break
			}
			parent2 := sha256.Sum256(append(append([]byte("GAUTH_MERKLE_NODE:"), []byte(a.h)...), []byte(b.h)...))
			phex2 := hex.EncodeToString(parent2[:])
			bridges = append(bridges, phex2)
			blocks[len(blocks)-2] = seg{h: phex2, sz: a.sz * 2}
			blocks = blocks[:len(blocks)-1]
		}
	}
	return bridges
}

func TestFastReconstructionTamperBridge(t *testing.T) {
	os.Setenv("GAUTH_CONSISTENCY_V2_FAST", "1")
	defer os.Unsetenv("GAUTH_CONSISTENCY_V2_FAST")
	c := NewRevocationChain()
	for i := 0; i < 7; i++ {
		id := fmt.Sprintf("TB%d", i)
		c.Append(RevocationEvent{ID: id, DelegationID: id})
	}
	roots, sizes := buildPrefixDecomposition(c.Events(), 7)
	bridges := buildBridges(roots, sizes)
	bridges[0] = deadbeefValue // tamper
	fast := ReconstructStartRootFromPrefixBlocks(roots, sizes, 7, bridges)
	if fast != "" {
		t.Fatal("expected empty due to tampered bridge hash")
	}
}

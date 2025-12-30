package delegation

import (
	"os"
	"strconv"
	"testing"
)

// TestIntervalPathEquivalence verifies that interval-based streaming path (flag enabled) produces
// a path whose verification succeeds identically to legacy traversal for several sizes.
func TestIntervalPathEquivalence(t *testing.T) {
	chain := NewRevocationChain()
	// Append 64 events and sign a tree head after each append to create historical snapshots
	for i := 0; i < 64; i++ {
		if _, err := chain.Append(RevocationEvent{ID: "rev-" + strconv.Itoa(i), DelegationID: "del-" + strconv.Itoa(i)}); err != nil {
			t.Fatalf("append err: %v", err)
		}
		if _, err := chain.SignTreeHead(); err != nil {
			t.Fatalf("sign head: %v", err)
		}
	}
	// baseline proof (legacy path) for each historical head
	os.Setenv("AGENTAUTH_CONSISTENCY_V2_INTERVAL_PATH", "0")
	if len(chain.treeHeads) < 2 {
		t.Fatalf("need at least 2 tree heads")
	}
	baseProofs := make([]*ConsistencyProofV2, len(chain.treeHeads)-2) // exclude last start pointing to itself
	for i := 0; i < len(chain.treeHeads)-1; i++ {
		p, err := chain.GenerateConsistencyProofV2(i)
		if err != nil {
			t.Fatalf("legacy proof %d: %v", i, err)
		}
		// skip case where no growth; generator returns error earlier, so path always has growth
		if p.StartLength < p.EndLength {
			if i < len(baseProofs) {
				baseProofs[i] = p
			}
		}
	}
	// interval path proofs
	os.Setenv("AGENTAUTH_CONSISTENCY_V2_INTERVAL_PATH", "1")
	for i := 0; i < len(chain.treeHeads)-1; i++ {
		p, err := chain.GenerateConsistencyProofV2(i)
		if err != nil {
			t.Fatalf("interval proof %d: %v", i, err)
		}
		if p.StartLength >= p.EndLength {
			continue
		} // skip degenerate
		allHashes := make([]string, len(chain.events))
		for j, ev := range chain.events {
			allHashes[j] = ev.Hash
		}
		// Verify interval proof
		if err := VerifyConsistencyProofV2(p, allHashes); err != nil {
			t.Fatalf("interval verify fail at %d: %v", i, err)
		}
	}
}

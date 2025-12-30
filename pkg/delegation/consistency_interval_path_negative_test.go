package delegation

import (
	"os"
	"strconv"
	"testing"
)

// TestIntervalPathTamper ensures verification fails if a sibling digest in the interval path proof is altered.
func TestIntervalPathTamper(t *testing.T) {
	os.Setenv("AGENTAUTH_CONSISTENCY_V2_INTERVAL_PATH", "1")
	chain := NewRevocationChain()
	// Build first segment: 128 events, then snapshot
	for i := 0; i < 128; i++ {
		_, _ = chain.Append(RevocationEvent{ID: "rev-" + strconv.Itoa(i), DelegationID: "del-" + strconv.Itoa(i)})
	}
	if _, err := chain.SignTreeHead(); err != nil {
		t.Fatalf("sign head initial: %v", err)
	}
	// Append second segment: 128 more events (total 256) and snapshot
	for i := 128; i < 256; i++ {
		_, _ = chain.Append(RevocationEvent{ID: "rev-" + strconv.Itoa(i), DelegationID: "del-" + strconv.Itoa(i)})
	}
	if _, err := chain.SignTreeHead(); err != nil {
		t.Fatalf("sign head final: %v", err)
	}
	startIdx := len(chain.treeHeads) - 2
	proof, err := chain.GenerateConsistencyProofV2(startIdx)
	if err != nil {
		t.Fatalf("generate proof: %v", err)
	}
	if len(proof.Path) == 0 {
		t.Fatalf("expected non-empty path")
	}
	// Tamper first sibling
	orig := proof.Path[0]
	proof.Path[0] = "deadbeef" + orig
	allHashes := make([]string, len(chain.events))
	for i, ev := range chain.events {
		allHashes[i] = ev.Hash
	}
	if err := VerifyConsistencyProofV2(proof, allHashes); err == nil {
		t.Fatalf("expected verification failure after tamper")
	}
}

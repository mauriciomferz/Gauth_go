package delegation

import (
	mrand "math/rand"
	"testing"
	"time"
)

// legacyGenerateConsistencyProofV2 emulates old behavior by forcing a full tree rebuild before calling current generator.
func legacyGenerateConsistencyProofV2(c *RevocationChain, startIndex int) (*ConsistencyProofV2, error) {
	tmp := NewMerkleTree()
	for _, ev := range c.events {
		tmp.AppendLeaf(ev.Hash)
	}
	tmp.rebuildIfNeeded() // discard result; just emulate prior cost
	return c.GenerateConsistencyProofV2(startIndex)
}

func buildChainWithN(n int) *RevocationChain {
	rc := NewRevocationChain()
	for i := 0; i < n; i++ {
		ev := RevocationEvent{ID: randStr(12), DelegationID: randStr(8), Reason: string(RevocationReasonUserRequest)}
		rc.Append(ev)
		if (i+1)%16 == 0 { // manually record a SignedTreeHead snapshot
			rc.treeHeads = append(rc.treeHeads, newSTH(rc))
		}
	}
	if len(rc.treeHeads) == 0 { // ensure at least one tree head
		rc.treeHeads = append(rc.treeHeads, newSTH(rc))
	}
	// final latest head
	rc.treeHeads = append(rc.treeHeads, newSTH(rc))
	return rc
}

// newSTH constructs a SignedTreeHead snapshot from current chain state.
func newSTH(rc *RevocationChain) *SignedTreeHead {
	return &SignedTreeHead{Version: 1, MerkleRoot: rc.MerkleRoot(), ChainLength: len(rc.events), AggregateHash: rc.AggregateHash(), Timestamp: time.Now().UTC()}
}

func randStr(n int) string {
	// Use a local deterministic RNG to avoid mutating global source; randomness quality not critical for benchmark.
	c := []rune("abcdefghijklmnopqrstuvwxyz0123456789")
	// Seed chosen for reproducibility; different length requests create new instance for isolation.
	r := mrand.New(mrand.NewSource(int64(n) * 1337))
	out := make([]rune, n)
	for i := range out {
		out[i] = c[r.Intn(len(c))]
	}
	return string(out)
}

func BenchmarkConsistencyProofGeneration(b *testing.B) {
	// No global seeding; chain construction uses randStr which has its own deterministic RNG.
	sizes := []int{64, 256, 1024, 4096}
	for _, sz := range sizes {
		rc := buildChainWithN(sz)
		startIdx := 0
		if len(rc.treeHeads) > 1 {
			startIdx = len(rc.treeHeads)/2 - 1
		}
		b.Run("new_gen_"+itoa(sz), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				if _, err := rc.GenerateConsistencyProofV2(startIdx); err != nil {
					b.Fatalf("new gen error: %v", err)
				}
			}
		})
		b.Run("legacy_gen_"+itoa(sz), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				if _, err := legacyGenerateConsistencyProofV2(rc, startIdx); err != nil {
					b.Fatalf("legacy gen error: %v", err)
				}
			}
		})
	}
}

func itoa(v int) string { // avoid strconv import in benchmark file
	switch v {
	case 64:
		return "64"
	case 256:
		return "256"
	case 1024:
		return "1024"
	case 4096:
		return "4096"
	default:
		return ""
	}
}

package web

import (
	"fmt"
	"testing"
	"time"

	delegation "github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/delegation"
)

// BenchmarkRevocationConsistencyV2Proof measures generation + verification latency for
// logarithmic consistency proofs (GenerateConsistencyProofV2 + VerifyConsistencyProofV2).
// Provides baseline for RB10 performance tracking as event count scales.
func BenchmarkRevocationConsistencyV2Proof(b *testing.B) {
	sizes := []int{64, 256, 1024}
	for _, n := range sizes {
		b.Run("size="+localItoa(n), func(b *testing.B) {
			chain := delegation.NewRevocationChain()
			// Seed n events
			for i := 0; i < n; i++ {
				ev := delegation.RevocationEvent{ID: formatID(i), DelegationID: formatDelegID(i), Reason: string(delegation.RevocationReasonUserRequest)}
				if _, err := chain.Append(ev); err != nil {
					b.Fatalf("append: %v", err)
				}
			}
			// Sign a tree head midway and final to populate c.treeHeads
			_, _ = chain.SignTreeHead()
			time.Sleep(1 * time.Millisecond) // ensure timestamp ordering
			// Append one extra batch to grow
			for i := 0; i < 5; i++ {
				ev := delegation.RevocationEvent{ID: formatID(n + i), DelegationID: formatDelegID(n + i), Reason: string(delegation.RevocationReasonUserRequest)}
				if _, err := chain.Append(ev); err != nil {
					b.Fatalf("append late: %v", err)
				}
			}
			_, _ = chain.SignTreeHead()
			startIndex := 0 // first tree head
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				proof, err := chain.GenerateConsistencyProofV2(startIndex)
				if err != nil {
					b.Fatalf("generate proof: %v", err)
				}
				// Collect hashes
				hashes := make([]string, 0, chain.LatestTreeHead().ChainLength)
				for _, ev := range chain.Events() {
					hashes = append(hashes, ev.Hash)
				}
				if err := delegation.VerifyConsistencyProofV2(proof, hashes); err != nil {
					b.Fatalf("verify: %v", err)
				}
			}
		})
	}
}

// Helper small inline formatting to avoid fmt overhead in tight loops.
// localItoa avoids collision with existing test helper itoa in policy_diff_test.go
func localItoa(i int) string     { return fmt.Sprintf("%d", i) }
func formatID(i int) string      { return "rev-" + fmt.Sprintf("%d", i) }
func formatDelegID(i int) string { return "deleg-" + fmt.Sprintf("%d", i) }

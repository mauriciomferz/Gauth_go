package delegation

import (
	"os"
	"strconv"
	"testing"
)

// BenchmarkConsistencyIntervalPath compares legacy path generation (full temp tree) vs interval-based streaming.
// Sizes selected to exercise several tree depths while remaining fast.
func BenchmarkConsistencyIntervalPath(b *testing.B) {
	sizes := []int{512, 1024, 2048, 4096}
	for _, n := range sizes {
		b.Run("legacy_"+strconv.Itoa(n), func(b *testing.B) {
			os.Setenv("AGENTAUTH_CONSISTENCY_V2_INTERVAL_PATH", "0")
			chain := NewRevocationChain()
			// Build first half
			for i := 0; i < n/2; i++ {
				_, _ = chain.Append(RevocationEvent{ID: "rev-" + strconv.Itoa(i), DelegationID: "del-" + strconv.Itoa(i)})
			}
			if _, err := chain.SignTreeHead(); err != nil {
				b.Fatalf("sign start head: %v", err)
			}
			// Build second half
			for i := n / 2; i < n; i++ {
				_, _ = chain.Append(RevocationEvent{ID: "rev-" + strconv.Itoa(i), DelegationID: "del-" + strconv.Itoa(i)})
			}
			if _, err := chain.SignTreeHead(); err != nil {
				b.Fatalf("sign end head: %v", err)
			}
			startIdx := len(chain.treeHeads) - 2
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if _, err := chain.GenerateConsistencyProofV2(startIdx); err != nil {
					b.Fatalf("legacy gen err: %v", err)
				}
			}
		})
		b.Run("interval_"+strconv.Itoa(n), func(b *testing.B) {
			os.Setenv("AGENTAUTH_CONSISTENCY_V2_INTERVAL_PATH", "1")
			chain := NewRevocationChain()
			for i := 0; i < n/2; i++ {
				_, _ = chain.Append(RevocationEvent{ID: "rev-" + strconv.Itoa(i), DelegationID: "del-" + strconv.Itoa(i)})
			}
			if _, err := chain.SignTreeHead(); err != nil {
				b.Fatalf("sign start head: %v", err)
			}
			for i := n / 2; i < n; i++ {
				_, _ = chain.Append(RevocationEvent{ID: "rev-" + strconv.Itoa(i), DelegationID: "del-" + strconv.Itoa(i)})
			}
			if _, err := chain.SignTreeHead(); err != nil {
				b.Fatalf("sign end head: %v", err)
			}
			startIdx := len(chain.treeHeads) - 2
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if _, err := chain.GenerateConsistencyProofV2(startIdx); err != nil {
					b.Fatalf("interval gen err: %v", err)
				}
			}
		})
	}
}

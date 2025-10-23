package delegation

import (
	"fmt"
	"os"
	"testing"
)

// seedBenchChain creates a chain with n events and signs a tree head.
func seedBenchChain(n int) *RevocationChain {
	c := NewRevocationChain()
	for i := 0; i < n; i++ {
		id := fmt.Sprintf("B%06d", i)
		_, _ = c.Append(RevocationEvent{ID: id, DelegationID: id})
	}
	_, _ = c.SignTreeHead()
	return c
}

// BenchmarkConsistencyProofV2 measures verification cost for various start/end growth scenarios.
// It focuses on the current O(n) start root rebuild path.
func BenchmarkConsistencyProofV2(b *testing.B) {
	sizes := []int{64, 256, 1024, 4096}
	for _, base := range sizes {
		chain := seedBenchChain(base)
		// Grow by +1 to simulate minimal extension
		if err := chain.Append(RevocationEvent{ID: fmt.Sprintf("grow-%d", base), DelegationID: fmt.Sprintf("grow-%d", base)}); err != nil {
			b.Fatalf("Failed to append revocation event: %v", err)
		}
		if err := chain.SignTreeHead(); err != nil {
			b.Fatalf("Failed to sign tree head: %v", err)
		}
		proof, err := chain.GenerateConsistencyProofV2(0)
		if err != nil {
			b.Fatalf("proof gen base %d: %v", base, err)
		}
		hashes := make([]string, 0, chain.LatestTreeHead().ChainLength)
		for _, ev := range chain.Events() {
			hashes = append(hashes, ev.Hash)
		}
		b.Run(fmt.Sprintf("start_%d_end_%d", proof.StartLength, proof.EndLength), func(sb *testing.B) {
			sb.ResetTimer()
			for i := 0; i < sb.N; i++ {
				if err := VerifyConsistencyProofV2(proof, hashes); err != nil {
					sb.Fatalf("verify failed: %v", err)
				}
			}
		})
	}
}

// BenchmarkConsistencyProofV2PathSize checks path size growth over larger chains.
func BenchmarkConsistencyProofV2PathSize(b *testing.B) {
	chain := seedBenchChain(8192)
	if err := chain.Append(RevocationEvent{ID: "grow-final", DelegationID: "grow-final"}); err != nil {
		b.Fatalf("Failed to append revocation event: %v", err)
	}
	if err := chain.SignTreeHead(); err != nil {
		b.Fatalf("Failed to sign tree head: %v", err)
	}
	proof, err := chain.GenerateConsistencyProofV2(0)
	if err != nil {
		b.Fatalf("proof gen: %v", err)
	}
	b.ReportMetric(float64(len(proof.Path)), "path_nodes")
	b.ReportMetric(float64(len(proof.PrefixRoots)), "prefix_blocks")
	hashes := make([]string, 0, chain.LatestTreeHead().ChainLength)
	for _, ev := range chain.Events() {
		hashes = append(hashes, ev.Hash)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := VerifyConsistencyProofV2(proof, hashes); err != nil {
			b.Fatalf("verify failed: %v", err)
		}
	}
}

// BenchmarkConsistencyProofV2FutureFast (placeholder) will attempt fast reconstruction when implemented.
func BenchmarkConsistencyProofV2FutureFast(b *testing.B) {
	// Enable flag (algorithm not yet implemented). Should behave identical to normal path.
	tval := os.Getenv("GAUTH_CONSISTENCY_V2_FAST")
	os.Setenv("GAUTH_CONSISTENCY_V2_FAST", "1")
	defer os.Setenv("GAUTH_CONSISTENCY_V2_FAST", tval)
	chain := seedBenchChain(1024)
	if err := chain.Append(RevocationEvent{ID: "grow-fast", DelegationID: "grow-fast"}); err != nil {
		b.Fatalf("Failed to append revocation event: %v", err)
	}
	if err := chain.SignTreeHead(); err != nil {
		b.Fatalf("Failed to sign tree head: %v", err)
	}
	proof, err := chain.GenerateConsistencyProofV2(0)
	if err != nil {
		b.Fatalf("proof gen: %v", err)
	}
	hashes := make([]string, 0, chain.LatestTreeHead().ChainLength)
	for _, ev := range chain.Events() {
		hashes = append(hashes, ev.Hash)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := VerifyConsistencyProofV2(proof, hashes); err != nil {
			b.Fatalf("verify failed: %v", err)
		}
	}
}

// BenchmarkStartRootReconstruction contrasts legacy full tree rebuild vs prefix fast path.
// It benchmarks only the start root portion (excluding path verification) to isolate improvement.
func BenchmarkStartRootReconstruction(b *testing.B) {
	sizes := []int{64, 256, 1024, 4096}
	for _, n := range sizes {
		chain := seedBenchChain(n)
		// Gather hashes for legacy full rebuild simulation.
		hashes := make([]string, n)
		for i, ev := range chain.Events() {
			hashes[i] = ev.Hash
		}
		// Generate proof (provides prefix blocks & bridges) for start snapshot relative to itself by appending one element.
		chain.Append(RevocationEvent{ID: "bench-extra", DelegationID: "bench-extra"})
		chain.SignTreeHead()
		proof, err := chain.GenerateConsistencyProofV2(0)
		if err != nil {
			b.Fatalf("proof gen: %v", err)
		}
		// Legacy approach: rebuild full start tree each iteration.
		b.Run(fmt.Sprintf("legacy_full_rebuild_%d", n), func(sb *testing.B) {
			for i := 0; i < sb.N; i++ {
				mt := NewMerkleTree()
				for j := 0; j < proof.StartLength; j++ {
					mt.AppendLeaf(hashes[j])
				}
				_ = mt.Root()
			}
		})
		// Fast path: reconstruct via prefix blocks.
		b.Run(fmt.Sprintf("fast_prefix_reconstruct_%d", n), func(sb *testing.B) {
			os.Setenv("GAUTH_CONSISTENCY_V2_FAST", "1")
			defer os.Unsetenv("GAUTH_CONSISTENCY_V2_FAST")
			for i := 0; i < sb.N; i++ {
				r := ReconstructStartRootFromPrefixBlocks(proof.PrefixRoots, proof.PrefixSizes, proof.StartLength, proof.PrefixBridges)
				if r == "" {
					sb.Fatalf("empty fast root for size %d", n)
				}
			}
		})
	}
}

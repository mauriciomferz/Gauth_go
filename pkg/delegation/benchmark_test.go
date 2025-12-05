package delegation

import (
	crand "crypto/rand"
	mrand "math/rand"
	"testing"
	"time"

	cryptoInt "github.com/mauriciomferz/Gauth_go/pkg/crypto"
)

// BenchmarkMerkleAppend measures leaf append throughput for a moderately sized tree.
func BenchmarkMerkleAppend(b *testing.B) {
	mt := NewMerkleTree()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf := make([]byte, 32)
		_, _ = crand.Read(buf)
		mt.AppendLeaf(hexEncode(buf))
	}
}

// BenchmarkMerkleGenerateProof builds a tree once then benchmarks proof generation for random indices.
func BenchmarkMerkleGenerateProof(b *testing.B) {
	size := 2000
	mt := NewMerkleTree()
	//nolint:gosec // G404: weak random acceptable for benchmark test
	idxRng := mrand.New(mrand.NewSource(42))
	for i := 0; i < size; i++ {
		mt.AppendLeaf(randomHex(32))
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		idx := idxRng.Intn(size)
		if _, _, err := mt.GenerateProof(idx); err != nil {
			b.Fatalf("Failed to generate proof: %v", err)
		}
	}
}

// BenchmarkSignTreeHeadSingleSig benchmarks signing a tree head for a chain with pre-populated events.
func BenchmarkSignTreeHeadSingleSig(b *testing.B) {
	km, _ := cryptoInt.NewManager(24 * time.Hour)
	cryptoInt.GlobalEdDSARegistry = km
	chain := NewRevocationChain()
	for i := 0; i < 1500; i++ {
		_, _ = chain.Append(RevocationEvent{ID: randomHex(8), DelegationID: randomHex(6)})
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = chain.SignTreeHead()
	}
}

// BenchmarkGenerateConsistencyProof benchmarks generating a consistency proof between an older and current tree head.
func BenchmarkGenerateConsistencyProof(b *testing.B) {
	km, _ := cryptoInt.NewManager(24 * time.Hour)
	cryptoInt.GlobalEdDSARegistry = km
	chain := NewRevocationChain()
	// Build initial events and sign first tree head
	for i := 0; i < 1000; i++ {
		_, _ = chain.Append(RevocationEvent{ID: randomHex(8), DelegationID: randomHex(6)})
	}
	_, _ = chain.SignTreeHead()
	// Append additional events and sign latest
	for i := 0; i < 500; i++ {
		_, _ = chain.Append(RevocationEvent{ID: randomHex(8), DelegationID: randomHex(6)})
	}
	_, _ = chain.SignTreeHead()
	if len(chain.TreeHeads()) < 2 {
		b.Fatalf("expected at least two tree heads")
	}
	startIndex := 0
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = chain.GenerateConsistencyProof(startIndex)
	}
}

// Helper: hex encode bytes (local minimal to avoid importing encoding/hex repeatedly here)
func hexEncode(b []byte) string {
	const hexdigits = "0123456789abcdef"
	out := make([]byte, len(b)*2)
	for i, by := range b {
		out[i*2] = hexdigits[by>>4]
		out[i*2+1] = hexdigits[by&0x0f]
	}
	return string(out)
}

func randomHex(n int) string {
	buf := make([]byte, n)
	_, _ = crand.Read(buf)
	return hexEncode(buf)
}

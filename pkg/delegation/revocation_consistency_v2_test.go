package delegation

import (
	"fmt"
	"strings"
	"testing"
	"time"

	cryptoInt "github.com/mauriciomferz/AgentAuth/pkg/crypto"
)

// helper to seed revocation chain with n events
func seedChain(n int, opts ...Option) *RevocationChain {
	c := NewRevocationChain(opts...)
	for i := 0; i < n; i++ {
		id := time.Now().UTC().Format("150405.000000") + string(rune('a'+(i%26))) + "-" + string(rune('A'+(i%26)))
		_, _ = c.Append(RevocationEvent{ID: id, DelegationID: id})
		time.Sleep(200 * time.Microsecond)
	}
	return c
}

func TestConsistencyProofV2BasicGrowth(t *testing.T) {
	chain := seedChain(5)
	// Sign two heads to populate treeHeads slice
	if _, err := chain.SignTreeHead(); err != nil {
		t.Fatalf("sign head1: %v", err)
	}
	if _, err := chain.Append(RevocationEvent{ID: "extra-1", DelegationID: "extra-1"}); err != nil {
		t.Fatalf("Failed to append extra-1: %v", err)
	}
	if _, err := chain.SignTreeHead(); err != nil {
		t.Fatalf("sign head2: %v", err)
	}
	proof, err := chain.GenerateConsistencyProofV2(0)
	if err != nil {
		t.Fatalf("generate v2 proof: %v", err)
	}
	if proof.EndLength <= proof.StartLength {
		t.Fatalf("expected growth")
	}
	if len(proof.Path) == 0 {
		t.Fatalf("expected non-empty path")
	}
	// Basic verification (prototype)
	hashes := []string{}
	for _, ev := range chain.Events() {
		hashes = append(hashes, ev.Hash)
	}
	if len(proof.Positions) != len(proof.Path) {
		t.Fatalf("positions length mismatch")
	}
	if verr := VerifyConsistencyProofV2(proof, hashes); verr != nil {
		t.Fatalf("verify proof v2 failed: %v", verr)
	}
}

func TestConsistencyProofV2VariousSizes(t *testing.T) {
	sizes := []int{3, 8, 9}
	for _, sz := range sizes {
		chain := seedChain(sz)
		if _, err := chain.SignTreeHead(); err != nil {
			t.Fatalf("sign initial: %v", err)
		}
		// grow
		if _, err := chain.Append(RevocationEvent{ID: "grow-x", DelegationID: "grow-x"}); err != nil {
			t.Fatalf("Failed to append grow-x: %v", err)
		}
		if _, err := chain.SignTreeHead(); err != nil {
			t.Fatalf("sign second: %v", err)
		}
		proof, err := chain.GenerateConsistencyProofV2(0)
		if err != nil {
			t.Fatalf("size %d proof err: %v", sz, err)
		}
		hashes := []string{}
		for _, ev := range chain.Events() {
			hashes = append(hashes, ev.Hash)
		}
		if len(proof.Positions) != len(proof.Path) {
			t.Fatalf("positions mismatch size %d", sz)
		}
		// Validate prefix decomposition invariants
		total := 0
		for _, s := range proof.PrefixSizes {
			total += s
		}
		if total != proof.StartLength {
			t.Fatalf("prefix sizes sum mismatch size %d", sz)
		}
		if len(proof.PrefixRoots) != len(proof.PrefixSizes) {
			t.Fatalf("prefix roots length mismatch size %d", sz)
		}
		if verr := VerifyConsistencyProofV2(proof, hashes); verr != nil {
			t.Fatalf("size %d verify err: %v", sz, verr)
		}
	}
}

// Multi-sig unrelated; ensure key manager presence doesn't break consistency functions
func TestConsistencyProofV2WithKeyManager(t *testing.T) {
	km, err := cryptoInt.NewManager(24 * time.Hour)
	if err != nil {
		t.Fatalf("km init: %v", err)
	}
	chain := seedChain(4, WithKeyProvider(km))
	if _, err2 := chain.SignTreeHead(); err2 != nil {
		t.Fatalf("sign 1: %v", err2)
	}
	_, _ = chain.Append(RevocationEvent{ID: "x2", DelegationID: "x2"})
	if _, err2 := chain.SignTreeHead(); err2 != nil {
		t.Fatalf("sign 2: %v", err2)
	}
	proof, err := chain.GenerateConsistencyProofV2(0)
	if err != nil {
		t.Fatalf("proof gen: %v", err)
	}
	hashes := []string{}
	for _, ev := range chain.Events() {
		hashes = append(hashes, ev.Hash)
	}
	if len(proof.Positions) != len(proof.Path) {
		t.Fatalf("positions len mismatch")
	}
	if verr := VerifyConsistencyProofV2(proof, hashes); verr != nil {
		t.Fatalf("verify: %v", verr)
	}
}

func TestConsistencyProofV2Tamper(t *testing.T) {
	chain := NewRevocationChain()
	for i := 0; i < 6; i++ {
		id := fmt.Sprintf("E%d", i)
		if _, err := chain.Append(RevocationEvent{ID: id, DelegationID: id}); err != nil {
			t.Fatalf("append: %v", err)
		}
	}
	if _, err := chain.SignTreeHead(); err != nil {
		t.Fatalf("sign initial: %v", err)
	}
	if _, err := chain.Append(RevocationEvent{ID: "extra", DelegationID: "extra"}); err != nil {
		t.Fatalf("append extra: %v", err)
	}
	if _, err := chain.SignTreeHead(); err != nil {
		t.Fatalf("sign second: %v", err)
	}
	proof, err := chain.GenerateConsistencyProofV2(0)
	if err != nil {
		t.Fatalf("gen proof: %v", err)
	}
	hashes := []string{}
	for _, ev := range chain.Events() {
		hashes = append(hashes, ev.Hash)
	}
	if verr := VerifyConsistencyProofV2(proof, hashes); verr != nil {
		t.Fatalf("baseline verify failed: %v", verr)
	}
	if len(proof.Path) == 0 {
		t.Skip("no path elements")
	}
	orig := proof.Path[0]
	proof.Path[0] = strings.Repeat("a", len(orig))
	if verr := VerifyConsistencyProofV2(proof, hashes); verr == nil {
		t.Fatalf("expected verification failure after tamper")
	}
}

// Tamper with a prefix root to ensure start root reconstruction fails.
func TestConsistencyProofV2PrefixTamper(t *testing.T) {
	chain := seedChain(7) // start length will be 7 (decomposes to 4+2+1)
	if _, err := chain.SignTreeHead(); err != nil {
		t.Fatalf("sign initial: %v", err)
	}
	_, _ = chain.Append(RevocationEvent{ID: "extra-a", DelegationID: "extra-a"})
	if _, err := chain.SignTreeHead(); err != nil {
		t.Fatalf("sign second: %v", err)
	}
	proof, err := chain.GenerateConsistencyProofV2(0)
	if err != nil {
		t.Fatalf("gen proof: %v", err)
	}
	hashes := []string{}
	for _, ev := range chain.Events() {
		hashes = append(hashes, ev.Hash)
	}
	if verr := VerifyConsistencyProofV2(proof, hashes); verr != nil {
		t.Fatalf("baseline verify failed: %v", verr)
	}
	if len(proof.PrefixRoots) == 0 {
		t.Skip("no prefix roots provided")
	}
	orig := proof.PrefixRoots[0]
	proof.PrefixRoots[0] = strings.Repeat("b", len(orig))
	if verr := VerifyConsistencyProofV2(proof, hashes); verr == nil {
		t.Fatalf("expected failure after prefix root tamper")
	}
}

package policy

import (
	"strings"
	"testing"
)

// TestRegistryTamperDetection_AAP_001_C1_C2 verifies correct detection of chain tampering
// specifically addressing multi-hop integrity (C1) and error code mapping (C2).
func TestRegistryTamperDetection_AAP_001_C1_C2(t *testing.T) {
	r := NewRegistry()

	// 1. Add Bundles A -> B -> C
	add := func(id string) {
		p := Policy{ID: id, Subjects: []string{"s"}, Rules: []Rule{{Effect: Allow}}}
		if _, err := r.AddBundle(Bundle{ID: id, Policies: []Policy{p}}); err != nil {
			t.Fatalf("AddBundle %s failed: %v", id, err)
		}
	}
	add("A")
	add("B")
	add("C")

	if err := r.VerifyChain(); err != nil {
		t.Fatalf("Initial chain invalid: %v", err)
	}

	// 2. Multi-hop Tamper: Modify Bundle B (index 1) which is in the middle
	// Direct access to bundles slice (allowed since we are in package policy)
	if len(r.bundles) != 3 {
		t.Fatalf("Expected 3 bundles, got %d", len(r.bundles))
	}

	// Preserve original hash to restore later if needed, or just tamper
	// Tamper: Modify Hash field to something else, breaking link from C (C.PrevHash == originalHash)
	// OR modify C.PrevHash?
	// but the stored Hash in the chain remains consistent with C.PrevHash?
	// If we substitute B with B', B'.Hash != B.Hash.
	// C.PrevHash expects B.Hash.
	// So VerifyChain checks: bundles[i].PrevHash == bundles[i-1].Hash.

	// Scenario 1: Integrity Failure (Content Tamper)
	// Modify content without updating Hash
	r.bundles[1].ID = "tampered_B"
	if err := r.VerifyChain(); err == nil {
		t.Fatal("VerifyChain passed after content tamper")
	} else if !strings.Contains(err.Error(), "hash mismatch") {
		t.Errorf("Expected hash mismatch error, got: %v", err)
	}

	// Restore B status (still has broken ID but we reset Hash?)
	// Let's restore fully
	r.bundles[1].ID = "B"

	// Scenario 2: Substitution (New valid bundle in middle)
	// B' has different content AND correct hash, but doesn't match C's PrevHash expectation.
	originalB := r.bundles[1]

	bPrime := originalB
	bPrime.ID = "B_PRIME"
	hPrime, _ := hashBundle(bPrime)
	bPrime.Hash = hPrime

	r.bundles[1] = bPrime

	if err := r.VerifyChain(); err == nil {
		t.Fatal("VerifyChain passed after substitution")
	} else if !strings.Contains(err.Error(), "broken") {
		// Expect "broken prev hash link at 2" because C.PrevHash points to B, not B'
		t.Errorf("Expected broken link error (substitution), got: %v", err)
	}

	// Restore B
	r.bundles[1] = originalB
	if err := r.VerifyChain(); err != nil {
		t.Fatal("Chain failed to recover after restore")
	}

	// Scenario 3: Broken Link (Explicit PrevHash tampering)
	// Modify C.PrevHash to point clearly wrong, update C.Hash so it looks valid itself
	r.bundles[2].PrevHash = "malicious_prev_hash"
	hC, _ := hashBundle(r.bundles[2])
	r.bundles[2].Hash = hC

	if err := r.VerifyChain(); err == nil {
		t.Fatal("VerifyChain passed after C.PrevHash tamper")
	} else if !strings.Contains(err.Error(), "broken") {
		t.Errorf("Expected broken link error, got: %v", err)
	}
}

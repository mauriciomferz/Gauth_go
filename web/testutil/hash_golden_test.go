package testutil

import "testing"

// TestCanonicalRegistryHashGolden asserts stable canonical hashes for a representative
// subset of capability registry fixtures. If a hash changes unexpectedly this likely
// indicates a semantic change in canonicalization logic or fixture content.
func TestCanonicalRegistryHashGolden(t *testing.T) {
	// Expected hashes captured from initial logging pass (sha256 of canonical JSON).
	// To intentionally update, re-run logging locally and adjust values here in a single commit.
	expected := map[string]string{
		"CapAlphaV1":          "3f75890a75a5c856e3027876ae7e05dc5f47569c51c7f5612ef5f8b481a1fd98",
		"CapAlphaBetaIssueV1": "c9efab04a4b6e74be7f3c0668b4abe039a29dec7523f60076ce839d392422fc0",
		// Permutation fixtures should hash differently (they are NOT semantically identical after canonicalization
		// because ordering inside action_mappings' value lists is preserved). Demonstrate both values for regression.
		"CapABDelegationIssuePerm1V1": "bccd9d013c5ba68e840e9c26f68b4dac4f00c1783ac3ed684901a93e9a8910e0",
		"CapABDelegationIssuePerm2V1": "1c20e5f8a106e1649518be4afab4d636ce5d51d269eb63dccdc54dc1892bed5e",
		// Semantic change (added cap.c) should differ from permutation hashes.
		"CapABCDelegationIssueV1": "07781cd6b285b56465009ba1b79c17c4e43ec145d157d5b2cb64c583fbad6b78",
	}
	fixtures := map[string]string{
		"CapAlphaV1":                  CapAlphaV1,
		"CapAlphaBetaIssueV1":         CapAlphaBetaIssueV1,
		"CapABDelegationIssuePerm1V1": CapABDelegationIssuePerm1V1,
		"CapABDelegationIssuePerm2V1": CapABDelegationIssuePerm2V1,
		"CapABCDelegationIssueV1":     CapABCDelegationIssueV1,
	}
	for name, raw := range fixtures {
		got := CanonicalRegistryHash(raw)
		want := expected[name]
		if got != want {
			t.Fatalf("canonical hash mismatch for %s: want=%s got=%s", name, want, got)
		}
	}
}

package testutil

import "testing"

// TestCanonicalPolicyBundleHashGolden asserts stable canonical hashes for policy bundle fixtures:
// - Permutation variants must hash identically (canonicalization sorts policies by ID)
// - Semantic addition (p3) must produce a distinct hash and populate change detection if used.
func TestCanonicalPolicyBundleHashGolden(t *testing.T) {
    expected := map[string]string{
        "PolicyBundleB1V1":          "0f4828f2d48daaca4a7cacb905d015fbedebc1a8e93a80ba0591a74323e20092",
        "PolicyBundleMultiPerm1V1":  "415ef7e5328fead1469224134a0c880185a46360f5471da9a415301398c86931",
        "PolicyBundleMultiPerm2V1":  "415ef7e5328fead1469224134a0c880185a46360f5471da9a415301398c86931",
        "PolicyBundleMultiPlusP3V1": "a9cd9b6a460a926a20fec3b3d79fe45f243c2f68ee86b2bc9a68a6a341dfcc79",
    }
    fixtures := map[string]string{
        "PolicyBundleB1V1":          PolicyBundleB1V1,
        "PolicyBundleMultiPerm1V1":  PolicyBundleMultiPerm1V1,
        "PolicyBundleMultiPerm2V1":  PolicyBundleMultiPerm2V1,
        "PolicyBundleMultiPlusP3V1": PolicyBundleMultiPlusP3V1,
    }
    for name, raw := range fixtures {
        got := CanonicalPolicyBundleHash(raw)
        want := expected[name]
        if got != want {
            t.Fatalf("canonical policy bundle hash mismatch for %s: want=%s got=%s", name, want, got)
        }
    }
}

package testutil

import "testing"

func TestIterateValidPolicyBundles(t *testing.T) {
    count := 0
    IterateValidPolicyBundles(func(name, raw string) bool {
        b, err := ParsePolicyBundle(raw)
        if err != nil {
            t.Fatalf("expected valid bundle %s; got error %v", name, err)
        }
        if b.ID == "" {
            t.Fatalf("expected non-empty id for %s", name)
        }
        count++
        return true
    })
    if count != len(ValidPolicyBundleFixtures) {
        t.Fatalf("expected %d policy fixtures iterated; got %d", len(ValidPolicyBundleFixtures), count)
    }
}

func TestCanonicalizePolicyBundleDeterministic(t *testing.T) {
    raw := PolicyBundleB1V1
    c1 := CanonicalizePolicyBundle(raw)
    c2 := CanonicalizePolicyBundle(raw)
    if c1 != c2 {
        t.Fatalf("canonical policy bundle output not deterministic")
    }
    // Should parse
    if _, err := ParsePolicyBundle(c1); err != nil {
        t.Fatalf("canonical policy bundle failed to parse: %v", err)
    }
}

func TestCanonicalPolicyBundleHashConsistency(t *testing.T) {
    h1 := CanonicalPolicyBundleHash(PolicyBundleB2V1)
    h2 := CanonicalPolicyBundleHash(PolicyBundleB2V1)
    if h1 != h2 {
        t.Fatalf("canonical policy bundle hash not deterministic: %s vs %s", h1, h2)
    }
    if len(h1) != 64 {
        t.Fatalf("expected 64-char hash; got %d", len(h1))
    }
}

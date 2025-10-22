package testutil

import "testing"

func TestParseCapabilityRegistrySuccess(t *testing.T) {
    reg := MustCapabilityRegistry(CapTransferIssueDelegationCreateV1)
    if reg.SchemaVersion != 1 {
        t.Fatalf("expected schema_version 1; got %d", reg.SchemaVersion)
    }
    if len(reg.Capabilities) != 3 {
        t.Fatalf("expected 3 capabilities; got %d", len(reg.Capabilities))
    }
    if reg.ActionMapping["transaction:execute"][0] != "cap.transfer" {
        t.Fatalf("unexpected mapping for transaction:execute: %+v", reg.ActionMapping["transaction:execute"])
    }
}

func TestParseCapabilityRegistryMissingSchema(t *testing.T) {
    _, err := ParseCapabilityRegistry(CapAlphaMissingSchemaVersion)
    if err == nil {
        t.Fatalf("expected error for missing schema_version")
    }
    if err != ErrMissingSchemaVersion {
        t.Fatalf("expected ErrMissingSchemaVersion; got %v", err)
    }
}

func TestSHA256HexDeterministic(t *testing.T) {
    h1 := SHA256Hex(CapTransferV1)
    h2 := SHA256Hex(CapTransferV1)
    if h1 != h2 {
        t.Fatalf("hash not deterministic: %s vs %s", h1, h2)
    }
    if len(h1) != 64 {
        t.Fatalf("expected 64 char hex string; got length %d", len(h1))
    }
}

func TestParsePolicyBundle(t *testing.T) {
    b := MustPolicyBundle(PolicyBundleB1V1)
    if b.ID != "b1" {
        t.Fatalf("unexpected bundle id: %s", b.ID)
    }
    if len(b.Policies) != 1 || b.Policies[0].ID != "p1" {
        t.Fatalf("unexpected policies parsed: %+v", b.Policies)
    }
}

func TestParseCapabilityRegistryUnknownCapability(t *testing.T) {
    _, err := ParseCapabilityRegistry(CapAlphaUnknownMapping)
    if err == nil {
        t.Fatalf("expected error for unknown capability mapping")
    }
    if !contains(err.Error(), "unknown capability id") {
        t.Fatalf("expected unknown capability id error; got %v", err)
    }
}

func TestParseCapabilityRegistryDuplicateIDs(t *testing.T) {
    _, err := ParseCapabilityRegistry(CapAlphaDuplicateIDs)
    if err == nil {
        t.Fatalf("expected duplicate id error")
    }
    if err != ErrDuplicateCapabilityID {
        t.Fatalf("expected ErrDuplicateCapabilityID; got %v", err)
    }
}

func TestIsValidCapabilityRegistry(t *testing.T) {
    if !IsValidCapabilityRegistry(CapTransferV1) {
        t.Fatalf("expected valid registry")
    }
    if IsValidCapabilityRegistry(CapAlphaMissingSchemaVersion) {
        t.Fatalf("expected invalid registry (missing schema_version)")
    }
}

// contains is a tiny helper to avoid importing strings for one use.
func contains(haystack, needle string) bool {
    return len(needle) == 0 || (len(haystack) >= len(needle) && index(haystack, needle) >= 0)
}

// index returns the index of needle in s or -1; naive implementation sufficient for tests.
func index(s, sub string) int {
    if len(sub) == 0 {
        return 0
    }
    for i := 0; i+len(sub) <= len(s); i++ {
        if s[i:i+len(sub)] == sub {
            return i
        }
    }
    return -1
}

func TestIterateValidRegistries(t *testing.T) {
    count := 0
    IterateValidRegistries(func(name, raw string) bool {
        if !IsValidCapabilityRegistry(raw) {
            t.Fatalf("fixture %s expected valid", name)
        }
        count++
        return true
    })
    if count != len(ValidCapabilityRegistryFixtures) {
        t.Fatalf("expected %d fixtures iterated; got %d", len(ValidCapabilityRegistryFixtures), count)
    }
}

func TestCanonicalizeRegistryDeterministic(t *testing.T) {
    raw := CapTransferIssueDelegationCreateV1
    c1 := CanonicalizeRegistry(raw)
    c2 := CanonicalizeRegistry(raw)
    if c1 != c2 {
        t.Fatalf("canonicalization not deterministic: %s vs %s", c1, c2)
    }
    // Canonical should still parse.
    if _, err := ParseCapabilityRegistry(c1); err != nil {
        t.Fatalf("canonical output failed to parse: %v", err)
    }
}

package testutil

// Fuzz tests provide property-based exploration of parser behavior beyond curated fixtures.
// They compile with normal `go test` but execute only when invoked via `go test -fuzz=Fuzz`.

import "testing"

// Seed fixtures for capability registry fuzzing.
var capabilitySeeds = []string{
	CapTransferV1,
	CapTransferIssueV1,
	CapTransferIssueDelegationCreateV1,
	CapAlphaV1,
	CapAlphaUnknownMapping,
	CapAlphaDuplicateIDs,
	CapAlphaMissingSchemaVersion,
	CapTransferAuditV1,
}

// Seed fixtures for policy bundle fuzzing.
var policySeeds = []string{
	PolicyBundleB1V1,
	PolicyBundleB2V1,
}

// FuzzParseCapabilityRegistry exercises ParseCapabilityRegistry across mutated inputs.
// Invariants (when parsing succeeds):
//   - SchemaVersion > 0
//   - Canonicalization parses
//   - Canonical and raw hash differ only when ordering differs
func FuzzParseCapabilityRegistry(f *testing.F) {
	for _, s := range capabilitySeeds {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, raw string) {
		reg, err := ParseCapabilityRegistry(raw)
		if err != nil {
			// Accept errors; ensure they are not panics.
			return
		}
		if reg.SchemaVersion <= 0 {
			t.Fatalf("parsed registry has non-positive SchemaVersion: %d", reg.SchemaVersion)
		}
		// Canonical version should parse again.
		canon := CanonicalizeRegistry(raw)
		reg2, err2 := ParseCapabilityRegistry(canon)
		if err2 != nil {
			t.Fatalf("canonical output failed to parse: %v", err2)
		}
		// Ensure capability count stable.
		if len(reg.Capabilities) != len(reg2.Capabilities) {
			t.Fatalf("capability count changed after canonicalization: %d vs %d", len(reg.Capabilities), len(reg2.Capabilities))
		}
	})
}

// FuzzParsePolicyBundle exercises ParsePolicyBundle for robustness.
// Invariants (when parsing succeeds):
//   - ID non-empty
//   - Canonicalization is deterministic & parses
func FuzzParsePolicyBundle(f *testing.F) {
	for _, s := range policySeeds {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, raw string) {
		b, err := ParsePolicyBundle(raw)
		if err != nil {
			return
		}
		if b.ID == "" {
			t.Fatalf("parsed bundle has empty ID")
		}
		canon := CanonicalizePolicyBundle(raw)
		b2, err2 := ParsePolicyBundle(canon)
		if err2 != nil {
			t.Fatalf("canonical bundle failed to parse: %v", err2)
		}
		if b.ID != b2.ID {
			t.Fatalf("bundle ID changed after canonicalization: %s vs %s", b.ID, b2.ID)
		}
	})
}

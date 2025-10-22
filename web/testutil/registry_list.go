package testutil

// registry_list.go provides convenience iteration and canonicalization helpers
// for capability registry fixtures.

import (
	"encoding/json"
	"sort"
)

// ValidCapabilityRegistryFixtures enumerates only the valid (non-negative test) registry fixtures.
// Keep order stable to avoid churn in table-driven tests.
var ValidCapabilityRegistryFixtures = []struct {
	Name string
	Raw  string
}{
	{Name: "CapTransferV1", Raw: CapTransferV1},
	{Name: "CapTransferIssueV1", Raw: CapTransferIssueV1},
	{Name: "CapTransferIssueDelegationCreateV1", Raw: CapTransferIssueDelegationCreateV1},
	{Name: "CapAlphaV1", Raw: CapAlphaV1},
	{Name: "CapTransferAuditV1", Raw: CapTransferAuditV1},
}

// IterateValidRegistries calls fn for each valid fixture. If fn returns false iteration stops early.
func IterateValidRegistries(fn func(name, raw string) bool) {
	for _, f := range ValidCapabilityRegistryFixtures {
		if !fn(f.Name, f.Raw) {
			return
		}
	}
}

// CanonicalizeRegistry parses then re-serializes a registry fixture with deterministic ordering:
// - Capabilities sorted by ID then Version
// - Action mapping keys sorted
// - Capability lists under each action left as-is (they reflect semantic order)
// Returns the canonical compact JSON or the original raw string if parsing fails.
func CanonicalizeRegistry(raw string) string {
	reg, err := ParseCapabilityRegistry(raw)
	if err != nil {
		// For negative fixtures or parse failure, just return the raw for stability.
		return raw
	}
	sort.SliceStable(reg.Capabilities, func(i, j int) bool {
		if reg.Capabilities[i].ID == reg.Capabilities[j].ID {
			return reg.Capabilities[i].Version < reg.Capabilities[j].Version
		}
		return reg.Capabilities[i].ID < reg.Capabilities[j].ID
	})
	// Sort action mapping keys; build a new map to guarantee ordering on marshal.
	type canonical struct {
		SchemaVersion int                 `json:"schema_version"`
		Capabilities  []Capability        `json:"capabilities"`
		ActionMapping map[string][]string `json:"action_mappings"`
	}
	keys := make([]string, 0, len(reg.ActionMapping))
	for k := range reg.ActionMapping {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	ordered := make(map[string][]string, len(keys))
	for _, k := range keys {
		ordered[k] = reg.ActionMapping[k]
	}
	c := canonical{
		SchemaVersion: reg.SchemaVersion,
		Capabilities:  reg.Capabilities,
		ActionMapping: ordered,
	}
	out, err := json.Marshal(c)
	if err != nil {
		return raw // Fallback: should not usually happen.
	}
	return string(out)
}

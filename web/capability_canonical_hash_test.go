package web

import (
	"crypto/sha256"
	"encoding/json"
	mrand "math/rand"
	"sort"
	"testing"
	"time"

	"github.com/mauriciomferz/AgentAuth/internal/capability"
)

// canonicalSerialize replicates the canonical capability registry serialization used for hashing.
func canonicalSerialize(caps []capability.Capability, actionMappings map[string][]string, schemaVersion int) ([]byte, error) {
	// Sort capabilities by ID
	orderedCaps := make([]capability.Capability, len(caps))
	copy(orderedCaps, caps)
	sort.Slice(orderedCaps, func(i, j int) bool { return orderedCaps[i].ID < orderedCaps[j].ID })
	// Sort actions and each capability list
	actions := make([]string, 0, len(actionMappings))
	for act := range actionMappings {
		actions = append(actions, act)
	}
	sort.Strings(actions)
	canonActions := make(map[string][]string, len(actions))
	for _, a := range actions {
		lst := make([]string, len(actionMappings[a]))
		copy(lst, actionMappings[a])
		sort.Strings(lst)
		canonActions[a] = lst
	}
	payload := struct {
		SchemaVersion  int                     `json:"schema_version"`
		Capabilities   []capability.Capability `json:"capabilities"`
		ActionMappings map[string][]string     `json:"action_mappings"`
	}{SchemaVersion: schemaVersion, Capabilities: orderedCaps, ActionMappings: canonActions}
	return json.Marshal(payload)
}

// randomPerm returns a new slice with the elements of src randomly permuted.
func randomPerm[T any](src []T) []T {
	out := make([]T, len(src))
	copy(out, src)
	//nolint:gosec // G404: weak random acceptable for test permutation generation
	r := mrand.New(mrand.NewSource(time.Now().UnixNano()))
	r.Shuffle(len(out), func(i, j int) { out[i], out[j] = out[j], out[i] })
	return out
}

// TestCapabilityCanonicalHashStability generates permutations of capabilities and action mappings
// to assert that canonical hash computation remains stable regardless of input ordering.
func TestCapabilityCanonicalHashStability(t *testing.T) {
	// Disable background polls to minimize interference / timing variance.
	t.Setenv("GAUTH_DISABLE_BG_POLLS", "1")
	t.Setenv("GAUTH_SKIP_SMOKETEST", "1")
	// Use local RNG instances for each shuffle to avoid global state mutation.
	baseCaps := []capability.Capability{
		{ID: "cap.transfer", Version: "1.0", Stable: true},
		{ID: "cap.issue", Version: "1.0", Stable: true},
		{ID: "cap.delegation.create", Version: "1.0", Stable: true},
		{ID: "cap.delegation.revoke", Version: "1.0", Stable: true},
	}
	baseActions := map[string][]string{
		"transaction:execute": {"cap.transfer"},
		"transaction:pay":     {"cap.transfer"},
		"transaction:issue":   {"cap.issue"},
		"delegation:create":   {"cap.delegation.create"},
		"delegation:revoke":   {"cap.delegation.revoke"},
	}
	// Compute reference hash from base ordering.
	refBytes, err := canonicalSerialize(baseCaps, baseActions, 1)
	if err != nil {
		t.Fatalf("canonical serialize ref: %v", err)
	}
	refHash := sha256.Sum256(refBytes)
	// Perform multiple trials with random permutations.
	trials := 100
	for i := 0; i < trials; i++ {
		capsPerm := randomPerm(baseCaps)
		// Randomly permute action mapping keys and individual capability lists.
		// We rebuild an unordered map in shuffled insertion order.
		actionKeys := make([]string, 0, len(baseActions))
		for k := range baseActions {
			actionKeys = append(actionKeys, k)
		}
		// Local shuffle to avoid global RNG.
		//nolint:gosec // G404: weak random acceptable for test permutation generation
		local := mrand.New(mrand.NewSource(time.Now().UnixNano()))
		local.Shuffle(len(actionKeys), func(i, j int) { actionKeys[i], actionKeys[j] = actionKeys[j], actionKeys[i] })
		permActions := make(map[string][]string, len(baseActions))
		for _, k := range actionKeys {
			permActions[k] = randomPerm(baseActions[k])
		}
		b, err := canonicalSerialize(capsPerm, permActions, 1)
		if err != nil {
			t.Fatalf("canonical serialize trial %d: %v", i, err)
		}
		h := sha256.Sum256(b)
		if h != refHash {
			t.Fatalf("canonical hash instability detected trial=%d ref=%x got=%x", i, refHash, h)
		}
	}
}

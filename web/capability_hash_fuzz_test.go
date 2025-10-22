package web

import (
	"crypto/sha256"
	"encoding/json"
	"math/rand"
	"sort"
	"strconv"
	"testing"

	"github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/internal/capability"
)

// canonicalRegistryHash computes the deterministic hash of the registry (mirrors server seed logic).
func canonicalRegistryHash(caps []capability.Capability, action map[string][]string) string {
	sorted := make([]capability.Capability, len(caps))
	copy(sorted, caps)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].ID < sorted[j].ID })
	canon := struct {
		SchemaVersion  int                     `json:"schema_version"`
		Capabilities   []capability.Capability `json:"capabilities"`
		ActionMappings map[string][]string     `json:"action_mappings"`
	}{SchemaVersion: 1, Capabilities: sorted, ActionMappings: action}
	enc, _ := json.Marshal(canon)
	h := sha256.Sum256(enc)
	return string([]byte("sha256:" + fmtHex(h[:])))
}

// fmtHex provides lowercase hex without importing fmt for every fuzz iteration.
func fmtHex(b []byte) string {
	const hexdigits = "0123456789abcdef"
	out := make([]byte, len(b)*2)
	for i, v := range b {
		out[i*2] = hexdigits[v>>4]
		out[i*2+1] = hexdigits[v&0x0f]
	}
	return string(out)
}

// FuzzCapabilityCanonicalHash ensures order insensitivity of hash (set semantics) and stability under random input permutations.
func FuzzCapabilityCanonicalHash(f *testing.F) {
	// Seed corpus with small deterministic sets
	f.Add(3) // number of capabilities baseline
	f.Add(5) // larger set
	f.Fuzz(func(t *testing.T, n int) {
		if n < 0 {
			n = -n
		}
		if n == 0 {
			n = 1
		}
		if n > 50 {
			n = 50
		} // cap to keep test fast
		caps := make([]capability.Capability, 0, n)
		action := map[string][]string{"delegation:create": {"cap.delegation.create"}, "delegation:revoke": {"cap.delegation.revoke"}}
		for i := 0; i < n; i++ {
			caps = append(caps, capability.Capability{ID: "cap.test." + strconv.Itoa(i), Version: "1.0", Stable: true})
		}
		// Compute baseline hash
		h1 := canonicalRegistryHash(caps, action)
		// Permute capabilities
		rand.Shuffle(len(caps), func(i, j int) { caps[i], caps[j] = caps[j], caps[i] })
		h2 := canonicalRegistryHash(caps, action)
		if h1 != h2 {
			t.Fatalf("canonical hash instability: h1=%s h2=%s", h1, h2)
		}
	})
}

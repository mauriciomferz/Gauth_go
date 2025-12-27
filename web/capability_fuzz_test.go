package web

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"testing"
)

// attemptLoadCapabilities is a helper wrapping the reload endpoint; returns (ok, hashAfter)
func attemptLoadCapabilities(srv *BetaServer) (bool, string) {
	resp := performRequest(srv.router, "POST", "/api/v1/beta/capabilities/reload")
	if resp.Code != 200 {
		// On failure, hash should remain previous
		disc := performRequest(srv.router, "GET", "/.well-known/gauth-configuration")
		var doc map[string]any
		_ = json.Unmarshal(disc.Body.Bytes(), &doc)
		h, _ := doc["capability_registry_hash"].(string)
		return false, h
	}
	disc := performRequest(srv.router, "GET", "/.well-known/gauth-configuration")
	var doc map[string]any
	_ = json.Unmarshal(disc.Body.Bytes(), &doc)
	h, _ := doc["capability_registry_hash"].(string)
	return true, h
}

// mutateJSON attempts random permutations / invalidations of a base capability file.
func mutateJSON(base map[string]any, r *rand.Rand) []byte {
	// Randomly decide mutation type
	choice := r.Intn(6)
	// Clone base
	clone := map[string]any{}
	for k, v := range base {
		clone[k] = v
	}
	switch choice {
	case 0: // reorder capabilities
		caps := clone["capabilities"].([]map[string]any)
		r.Shuffle(len(caps), func(i, j int) { caps[i], caps[j] = caps[j], caps[i] })
	case 1: // duplicate capability id
		caps := clone["capabilities"].([]map[string]any)
		if len(caps) > 0 {
			caps = append(caps, caps[0])
			clone["capabilities"] = caps
		}
	case 2: // dangling reference in action_mappings
		am := clone["action_mappings"].(map[string][]string)
		am[fmt.Sprintf("action:%d", r.Intn(1000))] = []string{"cap.nonexistent"}
	case 3: // remove schema_version
		delete(clone, "schema_version")
	case 4: // add new valid capability + mapping
		caps := clone["capabilities"].([]map[string]any)
		newID := fmt.Sprintf("cap.%d", r.Intn(10000))
		caps = append(caps, map[string]any{"id": newID, "version": "1.0"})
		clone["capabilities"] = caps
		am := clone["action_mappings"].(map[string][]string)
		am[fmt.Sprintf("custom:%d", r.Intn(10000))] = []string{newID}
	case 5: // reorder mapping list
		am := clone["action_mappings"].(map[string][]string)
		for k, v := range am {
			r.Shuffle(len(v), func(i, j int) { v[i], v[j] = v[j], v[i] })
			am[k] = v
		}
	}
	// Marshal manual (avoid dependency): build simple structure
	// Convert caps back to generic slice
	actionMappings := map[string][]string{}
	for k, v := range clone["action_mappings"].(map[string][]string) {
		actionMappings[k] = v
	}
	out := map[string]any{}
	for k, v := range clone {
		out[k] = v
	}
	// Use encoding/json for final output
	b, _ := json.Marshal(out)
	return b
}

// FuzzCapabilityReload validates atomicity & hash stability under random mutations of capability file.
// Requires Go 1.18+ fuzzing support. Short fuzz runs in CI; extended runs locally.
func FuzzCapabilityReload(f *testing.F) {
	seed := []byte(`{"schema_version":1,"capabilities":[{"id":"cap.alpha","version":"1.0"},{"id":"cap.beta","version":"1.0"}],"action_mappings":{"token:issue":["cap.alpha"],"delegation:create":["cap.beta"]}}`)
	f.Add(seed)
	f.Fuzz(func(t *testing.T, input []byte) {
		// Write seed base file first
		tmpDir := t.TempDir()
		capFile := fmt.Sprintf("%s/caps.json", tmpDir)
		if err := os.WriteFile(capFile, seed, 0o600); err != nil {
			t.Fatal(err)
		}
		t.Setenv("GAUTH_CAPABILITIES_PATH", capFile)
		srv := NewBetaServer(":0")
		t.Cleanup(func() { srv.Shutdown() })
		disc := performRequest(srv.router, "GET", "/.well-known/gauth-configuration")
		var doc map[string]any
		if err := json.Unmarshal(disc.Body.Bytes(), &doc); err != nil {
			t.Fatal(err)
		}
		originalHash, _ := doc["capability_registry_hash"].(string)
		if originalHash == "" {
			t.Fatalf("missing original hash")
		}

		// Derive deterministic pseudo-random source from input bytes to avoid time-dependent flakiness.
		// This stabilizes fuzz behavior across reruns while still exploring varied mutations based on corpus evolution.
		h := sha256.Sum256(input)
		// #nosec G115: deterministic seed from SHA256 hash, bounded by uint64 range
		seedInt := int64(binary.LittleEndian.Uint64(h[:8]))
		//nolint:gosec // G404: weak random acceptable for fuzz test mutation
		r := rand.New(rand.NewSource(seedInt))
		// Use provided input as potential override of base, fallback to seed if invalid JSON
		var base map[string]any
		if err := json.Unmarshal(input, &base); err != nil {
			// fallback to seed decode
			_ = json.Unmarshal(seed, &base)
		}
		// Ensure required structural fields for mutation helper
		if _, ok := base["capabilities"].([]any); !ok {
			base["capabilities"] = []any{map[string]any{"id": "cap.fallback", "version": "1.0"}}
		}
		// Normalize capabilities slice to []map[string]any
		capsRaw := base["capabilities"].([]any)
		normCaps := []map[string]any{}
		for _, c := range capsRaw {
			if m, ok := c.(map[string]any); ok {
				normCaps = append(normCaps, m)
			}
		}
		base["capabilities"] = normCaps
		if _, ok := base["action_mappings"].(map[string]any); !ok {
			base["action_mappings"] = map[string]any{"token:issue": []any{"cap.fallback"}}
		}
		// Normalize mapping values
		amGeneric := base["action_mappings"].(map[string]any)
		normalized := map[string][]string{}
		for k, v := range amGeneric {
			switch vv := v.(type) {
			case []any:
				lst := []string{}
				for _, x := range vv {
					if s, ok := x.(string); ok {
						lst = append(lst, s)
					}
				}
				normalized[k] = lst
			case []string:
				normalized[k] = vv
			default:
				normalized[k] = []string{"cap.fallback"}
			}
		}
		base["action_mappings"] = normalized
		mutBytes := mutateJSON(map[string]any{"schema_version": base["schema_version"], "capabilities": base["capabilities"], "action_mappings": normalized}, r)
		//nolint:gosec // G306: test file permissions
		if err := os.WriteFile(capFile, mutBytes, 0o644); err != nil {
			t.Fatal(err)
		}
		ok, newHash := attemptLoadCapabilities(srv)
		if !ok {
			// hash must remain identical on failed reload
			if newHash != originalHash {
				t.Fatalf("hash changed on failed reload original=%s new=%s", originalHash, newHash)
			}
			return
		}
		// Successful reload: hash either unchanged (ordering only) or changed (semantic). Never empty.
		if newHash == "" {
			t.Fatalf("empty hash after reload")
		}
		// If semantic differences introduced (e.g., added capability) we expect hash change; heuristic: count capabilities.
		disc2 := performRequest(srv.router, "GET", "/.well-known/gauth-configuration")
		var doc2 map[string]any
		_ = json.Unmarshal(disc2.Body.Bytes(), &doc2)
		capsAny, _ := doc2["capability_registry"].([]any)
		// If capability count differs from original (base had 2) then hash must differ.
		if len(capsAny) != 2 && newHash == originalHash {
			t.Fatalf("hash unchanged despite capability count diff=%d", len(capsAny))
		}
	})
}

package web

import (
	"encoding/json"
	"os"
	"testing"

	testutil "github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/web/testutil"
)

// TestCapabilityRegistryPrevHashPermutationSemantics verifies that a permutation-only reload
// (no semantic change) does NOT populate capability_registry_prev_hash, while a subsequent
// semantic modification DOES populate prev_hash with the prior (stable) hash.
func TestCapabilityRegistryPrevHashPermutationSemantics(t *testing.T) {
    tmp, err := os.CreateTemp(t.TempDir(), "caps-prevhash-*.json")
    if err != nil {
        t.Fatal(err)
    }
    // Initial file (Permutation variant 1)
    if _, err := tmp.WriteString(testutil.CapABDelegationIssuePerm1V1); err != nil {
        t.Fatal(err)
    }
    if err := tmp.Close(); err != nil {
        t.Fatal(err)
    }
    t.Setenv("GAUTH_CAPABILITIES_PATH", tmp.Name())
    srv := NewBetaServer(":0")

    // Initial discovery
    d1 := performRequest(srv.router, "GET", "/.well-known/gauth-configuration")
    if d1.Code != 200 {
        t.Fatalf("discovery status=%d", d1.Code)
    }
    var doc1 map[string]any
    if err := json.Unmarshal(d1.Body.Bytes(), &doc1); err != nil {
        t.Fatal(err)
    }
    h1, _ := doc1["capability_registry_hash"].(string)
    if h1 == "" {
        t.Fatalf("expected initial hash present")
    }
    if doc1["capability_registry_prev_hash"] != nil {
        t.Fatalf("expected prev hash nil on first load got %v", doc1["capability_registry_prev_hash"])
    }

    // Overwrite with permutation-only variant (same semantics, different ordering)
    if err := os.WriteFile(tmp.Name(), []byte(testutil.CapABDelegationIssuePerm2V1), 0o644); err != nil {
        t.Fatal(err)
    }
    rPerm := performRequest(srv.router, "POST", "/api/v1/beta/capabilities/reload")
    if rPerm.Code != 200 {
        t.Fatalf("perm reload status=%d body=%s", rPerm.Code, rPerm.Body.String())
    }
    d2 := performRequest(srv.router, "GET", "/.well-known/gauth-configuration")
    if d2.Code != 200 {
        t.Fatalf("discovery2 status=%d", d2.Code)
    }
    var doc2 map[string]any
    if err := json.Unmarshal(d2.Body.Bytes(), &doc2); err != nil {
        t.Fatal(err)
    }
    h2, _ := doc2["capability_registry_hash"].(string)
    if h2 != h1 {
        t.Fatalf("hash changed on permutation-only reload h1=%s h2=%s", h1, h2)
    }
    if doc2["capability_registry_prev_hash"] != nil {
        t.Fatalf("prev hash populated unexpectedly after permutation reload: %v", doc2["capability_registry_prev_hash"])
    }

    // Now introduce semantic change (add cap.c referenced in mapping)
    if err := os.WriteFile(tmp.Name(), []byte(testutil.CapABCDelegationIssueV1), 0o644); err != nil {
        t.Fatal(err)
    }
    rSem := performRequest(srv.router, "POST", "/api/v1/beta/capabilities/reload")
    if rSem.Code != 200 {
        t.Fatalf("semantic reload status=%d body=%s", rSem.Code, rSem.Body.String())
    }
    d3 := performRequest(srv.router, "GET", "/.well-known/gauth-configuration")
    if d3.Code != 200 {
        t.Fatalf("discovery3 status=%d", d3.Code)
    }
    var doc3 map[string]any
    if err := json.Unmarshal(d3.Body.Bytes(), &doc3); err != nil {
        t.Fatal(err)
    }
    h3, _ := doc3["capability_registry_hash"].(string)
    if h3 == h2 {
        t.Fatalf("hash did not change after semantic modification h2=%s h3=%s", h2, h3)
    }
    prev, _ := doc3["capability_registry_prev_hash"].(string)
    if prev != h2 {
        t.Fatalf("prev hash mismatch expected previous stable hash=%s got=%s", h2, prev)
    }
}

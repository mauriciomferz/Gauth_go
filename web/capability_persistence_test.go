package web

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	testutil "github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/web/testutil"
)

// TestCapabilityPersistenceAndReload verifies loading from file and reload behavior, including provenance fields.
func TestCapabilityPersistenceAndReload(t *testing.T) {
	// Write a temporary capabilities file
	tmp, err := os.CreateTemp(t.TempDir(), "caps-*.json")
	if err != nil {
		t.Fatal(err)
	}
	// initial file with two capabilities (subset) and one action
	if _, err := tmp.WriteString(testutil.CapAlphaBetaIssueV1); err != nil {
		t.Fatal(err)
	}
	if err := tmp.Close(); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GAUTH_CAPABILITIES_PATH", tmp.Name())
	srv := NewBetaServer(":0")

	// Discovery should show source = file and capabilities from file
	resp := performRequest(srv.router, "GET", "/.well-known/gauth-configuration")
	if resp.Code != 200 {
		t.Fatalf("discovery status=%d", resp.Code)
	}
	var doc map[string]any
	if err := json.Unmarshal(resp.Body.Bytes(), &doc); err != nil {
		t.Fatal(err)
	}
	if doc["capability_registry_source"] != "file" {
		t.Fatalf("expected source file got %v", doc["capability_registry_source"])
	}
	capsAny := doc["capability_registry"].([]any)
	if len(capsAny) != 2 {
		t.Fatalf("expected 2 capabilities from file got %d", len(capsAny))
	}
	firstHash, _ := doc["capability_registry_hash"].(string)
	if firstHash == "" || !strings.HasPrefix(firstHash, "sha256:") {
		t.Fatalf("expected capability_registry_hash set got %v", doc["capability_registry_hash"])
	}
	// Initial load should have no previous hash
	if doc["capability_registry_prev_hash"] != nil {
		t.Fatalf("expected prev hash nil on initial load got %v", doc["capability_registry_prev_hash"])
	}
	// Initial change timestamp should exist
	if doc["capability_registry_last_changed_at"] == nil {
		t.Fatalf("expected initial last_changed_at timestamp present")
	}

	// Modify file to add new capability and mapping
	if err := os.WriteFile(tmp.Name(), []byte(testutil.CapAlphaBetaGammaDelegationIssueV1), 0o644); err != nil {
		t.Fatal(err)
	}

	// Reload endpoint
	reloadResp := performRequest(srv.router, "POST", "/api/v1/beta/capabilities/reload")
	if reloadResp.Code != 200 {
		t.Fatalf("reload status=%d body=%s", reloadResp.Code, reloadResp.Body.String())
	}
	var reload map[string]any
	if err := json.Unmarshal(reloadResp.Body.Bytes(), &reload); err != nil {
		t.Fatal(err)
	}
	if reload["capabilities_after"].(float64) != 3 {
		t.Fatalf("expected 3 capabilities after reload got %v", reload["capabilities_after"])
	}

	// Discovery again should reflect 3 capabilities and updated hash
	resp2 := performRequest(srv.router, "GET", "/.well-known/gauth-configuration")
	if resp2.Code != 200 {
		t.Fatalf("discovery2 status=%d", resp2.Code)
	}
	var doc2 map[string]any
	if err := json.Unmarshal(resp2.Body.Bytes(), &doc2); err != nil {
		t.Fatal(err)
	}
	caps2 := doc2["capability_registry"].([]any)
	if len(caps2) != 3 {
		t.Fatalf("expected 3 capabilities after reload got %d", len(caps2))
	}
	if doc2["capability_registry_last_loaded"] == nil {
		t.Fatalf("expected last_loaded timestamp present")
	}
	if doc2["capability_registry_last_changed_at"] == nil {
		t.Fatalf("expected last_changed_at timestamp after reload")
	}
	secondHash, _ := doc2["capability_registry_hash"].(string)
	if secondHash == "" || !strings.HasPrefix(secondHash, "sha256:") {
		t.Fatalf("expected hash after reload got %v", doc2["capability_registry_hash"])
	}
	if secondHash == firstHash {
		t.Fatalf("expected hash to change after capability set modification")
	}
	// Previous hash should now equal the first hash after semantic change
	prevAfter, _ := doc2["capability_registry_prev_hash"].(string)
	if prevAfter == "" {
		t.Fatalf("expected prev hash populated after semantic change")
	}
	if prevAfter != firstHash {
		t.Fatalf("prev hash mismatch expected=%s got=%s", firstHash, prevAfter)
	}
}

// TestCapabilityRegistryHashPreservedOnFailedReload ensures hash does not change when reload fails.
func TestCapabilityRegistryHashPreservedOnFailedReload(t *testing.T) {
	tmp, err := os.CreateTemp(t.TempDir(), "caps-*.json")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tmp.WriteString(testutil.CapAlphaV1); err != nil {
		t.Fatal(err)
	}
	tmp.Close()
	t.Setenv("GAUTH_CAPABILITIES_PATH", tmp.Name())
	srv := NewBetaServer(":0")
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
	// Write invalid file (unknown capability in mapping)
	if err := os.WriteFile(tmp.Name(), []byte(testutil.CapAlphaUnknownMapping), 0o644); err != nil {
		t.Fatal(err)
	}
	r := performRequest(srv.router, "POST", "/api/v1/beta/capabilities/reload")
	if r.Code != 500 {
		t.Fatalf("expected 500 reload failure got %d", r.Code)
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
		t.Fatalf("hash changed unexpectedly on failed reload: before=%s after=%s", h1, h2)
	}
	// prev hash should remain empty after failed reload
	if doc2["capability_registry_prev_hash"] != nil {
		t.Fatalf("expected prev hash nil after failed reload got=%v", doc2["capability_registry_prev_hash"])
	}
}

// TestCapabilityRegistryHashDeterminism ensures canonical hash is stable across equivalent permutations
// and changes only with semantic differences (added/removed capability or mapping change).
func TestCapabilityRegistryHashDeterminism(t *testing.T) {
	tmp1, err := os.CreateTemp(t.TempDir(), "caps-a-*.json")
	if err != nil {
		t.Fatal(err)
	}
	// Same semantic content but different ordering in capabilities and action_mappings
	if _, err := tmp1.WriteString(testutil.CapABDelegationIssuePerm1V1); err != nil {
		t.Fatal(err)
	}
	tmp1.Close()
	t.Setenv("GAUTH_CAPABILITIES_PATH", tmp1.Name())
	srv := NewBetaServer(":0")
	dA := performRequest(srv.router, "GET", "/.well-known/gauth-configuration")
	if dA.Code != 200 {
		t.Fatalf("status %d", dA.Code)
	}
	var docA map[string]any
	if err := json.Unmarshal(dA.Body.Bytes(), &docA); err != nil {
		t.Fatal(err)
	}
	hA, _ := docA["capability_registry_hash"].(string)
	if hA == "" {
		t.Fatalf("hash missing")
	}
	// Overwrite file with permutation
	if err := os.WriteFile(tmp1.Name(), []byte(testutil.CapABDelegationIssuePerm2V1), 0o644); err != nil {
		t.Fatal(err)
	}
	r := performRequest(srv.router, "POST", "/api/v1/beta/capabilities/reload")
	if r.Code != 200 {
		t.Fatalf("reload status %d body=%s", r.Code, r.Body.String())
	}
	dB := performRequest(srv.router, "GET", "/.well-known/gauth-configuration")
	if dB.Code != 200 {
		t.Fatalf("status %d", dB.Code)
	}
	var docB map[string]any
	if err := json.Unmarshal(dB.Body.Bytes(), &docB); err != nil {
		t.Fatal(err)
	}
	hB, _ := docB["capability_registry_hash"].(string)
	if hB != hA {
		t.Fatalf("expected hash stable across permutations hA=%s hB=%s", hA, hB)
	}

	// Now introduce semantic change: add new capability referenced by mapping
	if err := os.WriteFile(tmp1.Name(), []byte(testutil.CapABCDelegationIssueV1), 0o644); err != nil {
		t.Fatal(err)
	}
	r2 := performRequest(srv.router, "POST", "/api/v1/beta/capabilities/reload")
	if r2.Code != 200 {
		t.Fatalf("reload2 status %d body=%s", r2.Code, r2.Body.String())
	}
	dC := performRequest(srv.router, "GET", "/.well-known/gauth-configuration")
	if dC.Code != 200 {
		t.Fatalf("status %d", dC.Code)
	}
	var docC map[string]any
	if err := json.Unmarshal(dC.Body.Bytes(), &docC); err != nil {
		t.Fatal(err)
	}
	hC, _ := docC["capability_registry_hash"].(string)
	if hC == hA {
		t.Fatalf("expected hash change on semantic modification hA=%s hC=%s", hA, hC)
	}
	// prev hash should now be hA
	prevC, _ := docC["capability_registry_prev_hash"].(string)
	if prevC != hA {
		t.Fatalf("expected prev hash to equal previous head hA=%s prevC=%s", hA, prevC)
	}
}

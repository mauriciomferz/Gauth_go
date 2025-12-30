package web

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"testing"

	"github.com/mauriciomferz/AgentAuth/internal/capability"
)

const testPastTime = "2025-01-01T00:00:00Z"

// TestCapabilityNegotiationStrictLifecycle ensures deprecated capabilities are excluded when strict flag set.
func TestCapabilityNegotiationStrictLifecycle(t *testing.T) {
	// Set lifecycle strict flag and start with clean registry
	t.Setenv("GAUTH_CAP_LIFECYCLE_STRICT", "1")
	capability.Reset([]capability.Capability{})
	srv := NewBetaServer(":0")
	t.Cleanup(func() { srv.Shutdown() })
	// Manually register a capability with deprecated_after in the past.
	past := testPastTime
	capability.Register(capability.Capability{ID: "cap.deprecated.demo", Version: "1.0", Stable: false, DeprecatedAfter: past})
	// Recompute registry hash to reflect change (simplified: invoke load logic manually by mimicking hash recompute used for static seed).
	caps := capability.DefaultRegistry().List()
	sorted := make([]capability.Capability, len(caps))
	copy(sorted, caps)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].ID < sorted[j].ID })
	canon := struct {
		SchemaVersion  int                     `json:"schema_version"`
		Capabilities   []capability.Capability `json:"capabilities"`
		ActionMappings map[string][]string     `json:"action_mappings"`
	}{SchemaVersion: 1, Capabilities: sorted, ActionMappings: srv.capabilitiesHandler.GetActionMappings()}
	enc, _ := json.Marshal(canon)
	h := sha256.Sum256(enc)
	srv.capabilitiesHandler.RegistryHash = fmt.Sprintf("sha256:%x", h[:])

	// Client requests versions including the deprecated capability.
	reqBody := map[string]any{"client_versions": map[string][]string{"cap.deprecated.demo": {"1.0"}}}
	resp := doPost(srv, "/api/v1/beta/capabilities/negotiate", reqBody)
	if resp.Code != 200 {
		t.Fatalf("expected 200 got %d body=%s", resp.Code, resp.Body.String())
	}
	var doc map[string]any
	if err := json.Unmarshal(resp.Body.Bytes(), &doc); err != nil {
		t.Fatalf("json decode: %v", err)
	}
	if !doc["success"].(bool) {
		t.Fatalf("success=false")
	}
	agreed, _ := doc["agreed"].(map[string]any)
	unsupported, _ := doc["unsupported"].(map[string]any)
	if _, ok := agreed["cap.deprecated.demo"]; ok {
		t.Fatalf("deprecated capability should not be agreed under strict lifecycle")
	}
	if _, ok := unsupported["cap.deprecated.demo"]; !ok {
		t.Fatalf("expected deprecated capability listed as unsupported")
	}
	// Ensure lifecycle_strict flag present and true
	if v, ok := doc["lifecycle_strict"].(bool); !ok || !v {
		t.Fatalf("lifecycle_strict flag missing or false in response")
	}
}

// TestCapabilityNegotiationNonStrictLifecycle ensures deprecated capabilities still negotiate when strict disabled.
func TestCapabilityNegotiationNonStrictLifecycle(t *testing.T) {
	os.Unsetenv("GAUTH_CAP_LIFECYCLE_STRICT")
	capability.Reset([]capability.Capability{})
	srv := NewBetaServer(":0")
	t.Cleanup(func() { srv.Shutdown() })
	past := testPastTime
	capability.Register(capability.Capability{ID: "cap.deprecated.demo2", Version: "1.0", Stable: false, DeprecatedAfter: past})
	// Recompute registry hash
	caps := capability.DefaultRegistry().List()
	sorted := make([]capability.Capability, len(caps))
	copy(sorted, caps)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].ID < sorted[j].ID })
	canon := struct {
		SchemaVersion  int                     `json:"schema_version"`
		Capabilities   []capability.Capability `json:"capabilities"`
		ActionMappings map[string][]string     `json:"action_mappings"`
	}{SchemaVersion: 1, Capabilities: sorted, ActionMappings: srv.capabilitiesHandler.GetActionMappings()}
	enc, _ := json.Marshal(canon)
	h := sha256.Sum256(enc)
	srv.capabilitiesHandler.RegistryHash = fmt.Sprintf("sha256:%x", h[:])

	reqBody := map[string]any{"client_versions": map[string][]string{"cap.deprecated.demo2": {"1.0"}}}
	resp := doPost(srv, "/api/v1/beta/capabilities/negotiate", reqBody)
	if resp.Code != 200 {
		t.Fatalf("expected 200 got %d", resp.Code)
	}
	var doc map[string]any
	if err := json.Unmarshal(resp.Body.Bytes(), &doc); err != nil {
		t.Fatalf("json decode: %v", err)
	}
	agreed, _ := doc["agreed"].(map[string]any)
	if _, ok := agreed["cap.deprecated.demo2"]; !ok {
		t.Fatalf("expected deprecated capability to be agreed when strict disabled")
	}
}

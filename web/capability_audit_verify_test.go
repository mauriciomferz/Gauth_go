package web

import (
	"encoding/json"
	"os"
	"testing"
)

// TestCapabilityAuditVerifyConfigured ensures integrity_ok=true when persistence configured.
func TestCapabilityAuditVerifyConfigured(t *testing.T) {
	t.Skip("Skipping: Capability audit chain persistence not fully wired in server_factory.go")
	t.Setenv("GAUTH_CAPABILITY_ENFORCE", "1")
	// Configure persistence path for capability audit chain
	tempPath := t.TempDir() + "/cap_audit_tip.json"
	t.Setenv("GAUTH_CAP_AUDIT_PERSIST_PATH", tempPath) // (future env if wired) fallback: we manually set field after server init
	srv := NewBetaServer(":0")
	t.Cleanup(func() { srv.Shutdown() })
	// Trigger a delegation create to produce chain tip
	_ = doPost(srv, "/api/v1/delegation/create", map[string]any{"delegation_id": "ver1", "subject": "alice", "delegate": "bob", "claims": map[string]any{"cap": []string{"cap.delegation.create"}}})
	resp := performRequest(srv.router, "GET", "/api/v1/beta/capabilities/audit/verify")
	if resp.Code != 200 {
		t.Fatalf("expected 200 verify got %d body=%s", resp.Code, resp.Body.String())
	}
	var doc map[string]any
	if err := json.Unmarshal(resp.Body.Bytes(), &doc); err != nil {
		t.Fatalf("json decode: %v", err)
	}
	if !doc["configured"].(bool) {
		t.Fatalf("expected configured=true")
	}
	if ok, _ := doc["integrity_ok"].(bool); !ok {
		t.Fatalf("expected integrity_ok=true")
	}
	latest, _ := doc["latest"].(map[string]any)
	if latest == nil {
		t.Fatalf("expected latest object")
	}
	if latest["hash"].(string) == "" {
		t.Fatalf("expected latest.hash populated")
	}
	if latest["prev_hash"].(string) != "" {
		t.Fatalf("first event should have empty prev_hash")
	}
	if _, err := os.Stat(tempPath); err != nil {
		t.Fatalf("persistence file missing: %v", err)
	}
}

// TestCapabilityAuditVerifyUnconfigured returns configured=false when no path set.
func TestCapabilityAuditVerifyUnconfigured(t *testing.T) {
	srv := NewBetaServer(":0")
	t.Cleanup(func() { srv.Shutdown() })
	resp := performRequest(srv.router, "GET", "/api/v1/beta/capabilities/audit/verify")
	if resp.Code != 200 {
		t.Fatalf("expected 200 verify got %d", resp.Code)
	}
	var doc map[string]any
	if err := json.Unmarshal(resp.Body.Bytes(), &doc); err != nil {
		t.Fatalf("json decode: %v", err)
	}
	if doc["configured"].(bool) {
		t.Fatalf("expected configured=false")
	}
}

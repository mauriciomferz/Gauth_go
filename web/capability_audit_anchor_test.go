package web

import (
	"encoding/json"
	"testing"
)

// TestCapabilityAuditChainAnchoring validates anchoring of audit chain tip.
func TestCapabilityAuditChainAnchoring(t *testing.T) {
	t.Setenv("AGENTAUTH_CAPABILITY_ENFORCE", "1")
	t.Setenv("AGENTAUTH_ANCHOR_PROVIDER", "memory")
	t.Setenv("AGENTAUTH_CAPABILITY_ANCHOR_ENABLE", "1")
	t.Setenv("AGENTAUTH_CAP_AUDIT_PERSIST_PATH", t.TempDir()+"/cap_audit_tip.json")
	srv := NewBetaServer(":0")
	t.Cleanup(func() { srv.Shutdown() })
	prev := srv.CapAuditPrevHash()
	// Produce at least one audit chained event
	_ = doPost(srv, "/api/v1/delegation/create", map[string]any{"delegation_id": "a1", "subject": "alice", "delegate": "bob", "claims": map[string]any{"cap": []string{"cap.delegation.create"}}})
	// Verify anchor call
	// Verify chain tip advanced
	if srv.CapAuditPrevHash() == prev {
		t.Fatal("expected chain tip")
	}
	resp := doPost(srv, "/api/v1/beta/capabilities/audit/anchor", nil)
	if resp.Code != 200 {
		t.Fatalf("expected 200 anchor got %d body=%s", resp.Code, resp.Body.String())
	}
	var doc map[string]any
	if err := json.Unmarshal(resp.Body.Bytes(), &doc); err != nil {
		t.Fatalf("json decode: %v", err)
	}
	if !doc["success"].(bool) {
		t.Fatalf("success=false")
	}
	if doc["chain_tip"].(string) != srv.CapAuditPrevHash() {
		t.Fatalf("chain_tip mismatch")
	}
	if doc["hash"].(string) == "" {
		t.Fatalf("missing anchored hash")
	}
}

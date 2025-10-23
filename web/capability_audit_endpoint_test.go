package web

import (
	"encoding/json"
	"testing"
)

// TestCapabilityAuditEndpoint ensures capability enforcement audit entries are exported.
func TestCapabilityAuditEndpoint(t *testing.T) {
	t.Setenv("GAUTH_CAPABILITY_ENFORCE", "1")
	bs := NewBetaServer(":0")
	// Trigger denial (missing capability)
	resp := doPost(bs, "/api/v1/delegation/create", map[string]any{"delegation_id": "audit1", "subject": "alice", "delegate": "bob"})
	if resp.Code != 403 {
		t.Fatalf("expected 403 denial got %d", resp.Code)
	}
	// Trigger success with caps
	resp = doPost(bs, "/api/v1/delegation/create", map[string]any{"delegation_id": "audit2", "subject": "alice", "delegate": "bob", "claims": map[string]any{"cap": []string{"cap.delegation.create"}}})
	if resp.Code != 200 {
		t.Fatalf("expected 200 success got %d", resp.Code)
	}
	// Query audit capabilities endpoint
	list := performRequest(bs.router, "GET", "/api/v1/audit/capabilities")
	if list.Code != 200 {
		t.Fatalf("expected 200 list got %d", list.Code)
	}
	var doc map[string]any
	if err := json.Unmarshal(list.Body.Bytes(), &doc); err != nil {
		t.Fatal(err)
	}
	entries, ok := doc["entries"].([]any)
	if !ok {
		t.Fatalf("expected entries slice")
	}
	if len(entries) < 2 {
		t.Fatalf("expected at least 2 entries got %d", len(entries))
	}
	var foundDenied, foundCreate bool
	for _, raw := range entries {
		e := raw.(map[string]any)
		if e["action"] == actionCapabilityEnforce && e["outcome"] == "denied" {
			foundDenied = true
		}
		if e["action"] == "delegation_create" && e["outcome"] == "active" {
			foundCreate = true
		}
	}
	if !foundDenied {
		t.Fatalf("expected denied capability enforcement entry")
	}
	if !foundCreate {
		t.Fatalf("expected successful delegation_create entry")
	}
}

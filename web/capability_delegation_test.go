package web

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// helper to perform POST with JSON body
func doPost(bs *BetaServer, path string, body any) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	b, _ := json.Marshal(body)
	req, _ := http.NewRequest(http.MethodPost, path, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	bs.router.ServeHTTP(w, req)
	return w
}

// TestDelegationCreateCapabilityEnforcement ensures capability is required when enforcement flag enabled.
func TestDelegationCreateCapabilityEnforcement(t *testing.T) {
	t.Setenv("AGENTAUTH_CAPABILITY_ENFORCE", "1")
	bs := NewBetaServer("")
	t.Cleanup(func() { bs.Shutdown() })
	// Missing capability should yield 403
	resp := doPost(bs, "/api/v1/delegation/create", map[string]any{"delegation_id": "d1", "subject": "s1", "delegate": "dlt"})
	if resp.Code != 403 {
		t.Fatalf("expected 403 missing capability got %d body=%s", resp.Code, resp.Body.String())
	}
	if !bytes.Contains(resp.Body.Bytes(), []byte("capability_denied")) {
		t.Fatalf("expected capability_denied error body=%s", resp.Body.String())
	}
	// Provide required capability
	resp = doPost(bs, "/api/v1/delegation/create", map[string]any{"delegation_id": "d2", "subject": "s1", "delegate": "dlt", "claims": map[string]any{"cap": []string{"cap.delegation.create"}}})
	if resp.Code != 200 {
		t.Fatalf("expected 200 success got %d body=%s", resp.Code, resp.Body.String())
	}
	if !bytes.Contains(resp.Body.Bytes(), []byte("\"status\":\"active\"")) {
		t.Fatalf("expected active status body=%s", resp.Body.String())
	}
}

// TestDelegationRevokeCapabilityEnforcement ensures revoke requires capability.
func TestDelegationRevokeCapabilityEnforcement(t *testing.T) {
	t.Setenv("AGENTAUTH_CAPABILITY_ENFORCE", "1")
	bs := NewBetaServer("")
	t.Cleanup(func() { bs.Shutdown() })
	// First create delegation with required capability
	createResp := doPost(bs, "/api/v1/delegation/create", map[string]any{"delegation_id": "d1", "subject": "s1", "delegate": "dlt", "claims": map[string]any{"cap": []string{"cap.delegation.create"}}})
	if createResp.Code != 200 {
		// If create fails test cannot proceed
		t.Fatalf("setup create failed code=%d body=%s", createResp.Code, createResp.Body.String())
	}
	// Missing revoke capability should fail
	resp := doPost(bs, "/api/v1/delegation/revoke", map[string]any{"delegation_id": "d1"})
	if resp.Code != 403 {
		t.Fatalf("expected 403 revoke missing capability got %d body=%s", resp.Code, resp.Body.String())
	}
	// Provide required revoke capability
	resp = doPost(bs, "/api/v1/delegation/revoke", map[string]any{"delegation_id": "d1", "claims": map[string]any{"cap": []string{"cap.delegation.revoke"}}})
	if resp.Code != 200 {
		t.Fatalf("expected 200 revoke success got %d body=%s", resp.Code, resp.Body.String())
	}
	if !bytes.Contains(resp.Body.Bytes(), []byte("terminated")) {
		t.Fatalf("expected terminated status body=%s", resp.Body.String())
	}
}

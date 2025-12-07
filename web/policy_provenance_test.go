package web

import (
	"encoding/json"
	"net/http/httptest"
	"testing"
)

// TestPolicyProvenanceEndpoint verifies provenance returns expected keys even when empty.
func TestPolicyProvenanceEndpoint(t *testing.T) {
	srv := NewBetaServer(":0")
	t.Cleanup(func() { srv.Shutdown() })
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/policy/provenance", nil)
	srv.router.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("status %d", w.Code)
	}
	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("json: %v", err)
	}
	required := []string{"success", "head_hash", "chain", "verified", "length"}
	for _, k := range required {
		if _, ok := body[k]; !ok {
			t.Fatalf("missing key %s", k)
		}
	}
}

// TestPolicyProvenanceUnknownHash queries an unknown hash and expects success=false or not verified indicators.
func TestPolicyProvenanceUnknownHash(t *testing.T) {
	srv := NewBetaServer(":0")
	t.Cleanup(func() { srv.Shutdown() })
	// Force empty registry scenario (if seeding disabled this remains empty); request with hash param
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/policy/provenance?hash=nonexistentdeadbeef", nil)
	srv.router.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("status %d", w.Code)
	}
	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("json: %v", err)
	}
	// Accept either success=false or verified=false depending on implementation
	if s, ok := body["success"].(bool); ok && s {
		if v, vok := body["verified"].(bool); vok && v {
			t.Fatalf("expected provenance not verified for unknown hash")
		}
	}
}

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
	// Use a valid SHA-256 hex string that definitely doesn't exist
	unknownHash := "000000000000000000000000000000000000000000000000000000000000dead"
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/policy/provenance?hash="+unknownHash, nil)
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

// TestPolicyProvenanceMalformedHash ensures system handles invalid hash formats gracefully.
func TestPolicyProvenanceMalformedHash(t *testing.T) {
	srv := NewBetaServer(":0")
	t.Cleanup(func() { srv.Shutdown() })
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/policy/provenance?hash=!!!badhash!!!", nil)
	srv.router.ServeHTTP(w, req)

	// Should return 400 Bad Request or 200 with error/success=false
	if w.Code == 200 {
		var body map[string]any
		if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
			t.Fatalf("unmarshal response: %v", err)
		}
		if s, ok := body["success"].(bool); ok && s {
			t.Fatal("expected success=false for malformed hash")
		}
	} else if w.Code != 400 {
		t.Fatalf("expected 400 or 200/error, got %d", w.Code)
	}
}

// TestPolicyProvenanceUnseeded checks explicit empty state.
func TestPolicyProvenanceUnseeded(t *testing.T) {
	t.Setenv("POLICY_CHAIN_STATE_PATH", "") // Ensure no state loaded
	srv := NewBetaServer(":0")
	t.Cleanup(func() { srv.Shutdown() })

	w := httptest.NewRecorder()
	srv.router.ServeHTTP(w, httptest.NewRequest("GET", "/api/v1/policy/provenance", nil))

	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if l, ok := body["length"].(float64); !ok || l != 0 {
		t.Errorf("expected length 0 for unseeded registry, got %v", body["length"])
	}
}

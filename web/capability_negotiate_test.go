package web

import (
	"encoding/json"
	"testing"
)

// TestCapabilityNegotiationBasic covers agreed and unsupported capability versions.
func TestCapabilityNegotiationBasic(t *testing.T) {
	// Seed server (static capabilities already registered in NewBetaServer)
	srv := NewBetaServer(":0")
	t.Cleanup(func() { srv.Shutdown() })
	// Request with supported versions (1.0) for existing capabilities plus an unknown capability.
	reqBody := map[string]any{
		"client_versions": map[string][]string{
			"cap.transfer": {"1.0"},
			"cap.issue":    {"1.0"},
			"cap.unknown":  {"1.0"},
		},
	}
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
	if agreed == nil || unsupported == nil {
		t.Fatalf("missing agreed/unsupported maps")
	}
	if _, ok := agreed["cap.transfer"]; !ok {
		t.Fatalf("expected cap.transfer agreed")
	}
	if _, ok := agreed["cap.issue"]; !ok {
		t.Fatalf("expected cap.issue agreed")
	}
	if _, ok := unsupported["cap.unknown"]; !ok {
		t.Fatalf("expected cap.unknown unsupported")
	}
}

// TestCapabilityNegotiationInvalidPayload ensures 400 on empty client_versions.
func TestCapabilityNegotiationInvalidPayload(t *testing.T) {
	srv := NewBetaServer(":0")
	t.Cleanup(func() { srv.Shutdown() })
	resp := doPost(srv, "/api/v1/beta/capabilities/negotiate", map[string]any{"client_versions": map[string][]string{}})
	if resp.Code != 400 {
		t.Fatalf("expected 400 invalid payload got %d", resp.Code)
	}
}

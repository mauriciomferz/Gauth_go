package web

import (
	"encoding/json"
	"net/http/httptest"
	"os"
	"testing"
)

func TestWellKnownDiscovery(t *testing.T) {
	srv := NewBetaServer(":0")
	t.Cleanup(func() { srv.Shutdown() })
	// Build request
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/.well-known/agentauth-configuration", nil)
	srv.router.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json decode failed: %v body=%s", err, w.Body.String())
	}
	// Required keys (expanded)
	required := []string{"version", "implementation", "issuer", "token_algorithms", "policy_endpoints", "poa_endpoints", "revocation_endpoints", "revocation_endpoint", "revocation_endpoint_v1", "revocation_support", "multi_signature_poa", "capabilities", "documentation", "jwks_uri", "introspection_endpoint", "key_rotation", "anchoring"}
	for _, k := range required {
		if _, ok := payload[k]; !ok {
			t.Errorf("missing key %s in discovery payload", k)
		}
	}
	// token_algorithms should be slice non-empty
	algs, ok := payload["token_algorithms"].([]any)
	if !ok || len(algs) == 0 {
		t.Errorf("token_algorithms missing or empty: %#v", payload["token_algorithms"])
	}
	if algs, ok := payload["token_algorithms"].([]any); ok {
		if len(algs) < 1 {
			t.Fatalf("expected at least one algorithm")
		}
	}
	// schema_version should be integer >=1
	if v, ok := payload["schema_version"].(float64); !ok || int(v) < 1 {
		t.Fatalf("invalid schema_version: %#v", payload["schema_version"])
	}
}

// TestDiscoveryExactKeys ensures discovery payload contains required keys and jwks_uri empties when JWT disabled.
func TestDiscoveryExactKeys(t *testing.T) {
	// Scenario 1: Neither JWT library nor EdDSA mode active -> jwks_uri should still be populated (Dynamic Identity default).
	t.Setenv("AGENTAUTH_USE_JWT_LIB", "")
	t.Setenv("AGENTAUTH_TOKEN_SIG_MODE", "hmac") // force legacy HMAC
	srv := NewBetaServer(":0")
	t.Cleanup(func() { srv.Shutdown() })
	w := performRequest(srv.router, "GET", "/.well-known/agentauth-configuration")
	if w.Code != 200 {
		t.Fatalf("status %d", w.Code)
	}
	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("json: %v", err)
	}
	required := []string{"version", "schema_version", "future_version", "deprecated_fields", "implementation", "issuer", "token_algorithms", "policy_endpoints", "poa_endpoints", "audit_endpoints", "revocation_endpoints", "revocation_endpoint", "revocation_endpoint_v1", "revocation_support", "multi_signature_poa", "capabilities", "documentation", "jwks_uri", "introspection_endpoint", "key_rotation", "anchoring"}
	if len(body) < len(required) {
		t.Fatalf("expected at least %d keys got %d", len(required), len(body))
	}
	for _, k := range required {
		if _, ok := body[k]; !ok {
			t.Fatalf("missing key %s", k)
		}
	}
	if body["jwks_uri"] == "" {
		t.Fatalf("jwks_uri should be populated even in legacy mode (Dynamic Identity support)")
	}

	// Scenario 2: Enable JWT library -> jwks_uri populated.
	t.Setenv("AGENTAUTH_USE_JWT_LIB", "1")
	t.Setenv("AGENTAUTH_JWT_KID", "demo-key")
	srv2 := NewBetaServer(":0")
	t.Cleanup(func() { srv2.Shutdown() })
	w2 := performRequest(srv2.router, "GET", "/.well-known/agentauth-configuration")
	var body2 map[string]any
	if err := json.Unmarshal(w2.Body.Bytes(), &body2); err != nil {
		t.Fatalf("json2: %v", err)
	}
	if body2["jwks_uri"] == "" {
		t.Fatalf("jwks_uri should be populated when JWT enabled")
	}

	// Scenario 3: Disable JWT but enable EdDSA -> jwks_uri populated (new behavior after EdDSA introduction).
	_ = os.Unsetenv("AGENTAUTH_USE_JWT_LIB")
	t.Setenv("AGENTAUTH_TOKEN_SIG_MODE", "eddsa")
	srv3 := NewBetaServer(":0")
	t.Cleanup(func() { srv3.Shutdown() })
	w3 := performRequest(srv3.router, "GET", "/.well-known/agentauth-configuration")
	var body3 map[string]any
	if err := json.Unmarshal(w3.Body.Bytes(), &body3); err != nil {
		t.Fatalf("json3: %v", err)
	}
	if body3["jwks_uri"] == "" {
		t.Fatalf("jwks_uri should be populated when eddsa mode enabled")
	}

	// Scenario 4: AGENTAUTH_ISSUER set -> jwks_uri and revocation endponts prefixed
	t.Setenv("AGENTAUTH_ISSUER", "https://agentauth.example.com")
	srv4 := NewBetaServer(":0")
	t.Cleanup(func() { srv4.Shutdown() })
	w4 := performRequest(srv4.router, "GET", "/.well-known/agentauth-configuration")
	var body4 map[string]any
	if err := json.Unmarshal(w4.Body.Bytes(), &body4); err != nil {
		t.Fatalf("json4: %v", err)
	}
	jwks := body4["jwks_uri"].(string)
	if jwks != "https://agentauth.example.com/.well-known/jwks.json" {
		t.Errorf("expected prefixed jwks_uri, got %q", jwks)
	}
	rev := body4["revocation_endpoint"].(string)
	if rev != "https://agentauth.example.com/api/v1/token/revoke" {
		t.Errorf("expected prefixed revocation_endpoint, got %q", rev)
	}
}

// TestDiscoveryAnchoringFields ensures anchoring fields populate when memory anchor enabled.
func TestDiscoveryAnchoringFields(t *testing.T) {
	t.Setenv("AGENTAUTH_ANCHOR_PROVIDER", "memory")
	srv := NewBetaServer(":0")
	t.Cleanup(func() { srv.Shutdown() })
	w := performRequest(srv.router, "GET", "/.well-known/agentauth-configuration")
	if w.Code != 200 {
		t.Fatalf("status %d", w.Code)
	}
	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("json: %v", err)
	}
	anch, ok := body["anchoring"].(map[string]any)
	if !ok {
		t.Fatalf("anchoring field missing or wrong type: %#v", body["anchoring"])
	}
	if enabled, ok := anch["enabled"].(bool); !ok || !enabled {
		t.Fatalf("expected anchoring enabled")
	}
	if _, ok := anch["total"]; !ok {
		t.Fatalf("missing total in anchoring")
	}
	if _, ok := anch["latest_hash"]; !ok {
		t.Fatalf("missing latest_hash in anchoring")
	}
}

func TestDiscoveryMultiSigWeights(t *testing.T) {
	t.Setenv("AGENTAUTH_MULTI_SIG_THRESHOLD", "5")
	t.Setenv("AGENTAUTH_MULTI_SIG_WEIGHTS", "signerA=3,signerB=2")
	srv := NewBetaServer(":0")
	t.Cleanup(func() { srv.Shutdown() })
	w := performRequest(srv.router, "GET", "/.well-known/agentauth-configuration")
	if w.Code != 200 {
		t.Fatalf("status %d", w.Code)
	}
	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("json: %v", err)
	}
	msRaw, ok := body["multi_signature_poa"].(map[string]any)
	if !ok {
		t.Fatalf("multi_signature_poa missing or wrong type")
	}
	weightsValid, ok := msRaw["weights_valid"].(bool)
	if !ok || !weightsValid {
		t.Fatalf("expected weights_valid=true got %#v", msRaw["weights_valid"])
	}
	if tv, ok := msRaw["weights_total"].(float64); !ok || int(tv) != 5 {
		t.Fatalf("weights_total mismatch %#v", msRaw["weights_total"])
	}
}

func TestDiscoveryMultiSigWeightsInvalid(t *testing.T) {
	t.Setenv("AGENTAUTH_MULTI_SIG_THRESHOLD", "5")
	t.Setenv("AGENTAUTH_MULTI_SIG_WEIGHTS", "signerA=2,signerB=2") // total 4 < threshold 5
	srv := NewBetaServer(":0")
	t.Cleanup(func() { srv.Shutdown() })
	w := performRequest(srv.router, "GET", "/.well-known/agentauth-configuration")
	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("json: %v", err)
	}
	msRaw, ok := body["multi_signature_poa"].(map[string]any)
	if !ok {
		t.Fatalf("multi_signature_poa missing or wrong type")
	}
	if weightsValid, ok := msRaw["weights_valid"].(bool); ok && weightsValid {
		t.Fatalf("expected weights_valid=false got %#v", msRaw["weights_valid"])
	}
}

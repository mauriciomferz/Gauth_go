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
	req := httptest.NewRequest("GET", "/.well-known/gauth-configuration", nil)
	srv.router.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json decode failed: %v body=%s", err, w.Body.String())
	}
	// Required keys (expanded)
	required := []string{"version", "implementation", "issuer", "token_algorithms", "policy_endpoints", "poa_endpoints", "revocation_endpoints", "revocation_support", "multi_signature_poa", "capabilities", "documentation", "jwks_uri", "introspection_endpoint", "key_rotation", "anchoring"}
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
	// Scenario 1: Neither JWT library nor EdDSA mode active -> jwks_uri should be empty.
	os.Unsetenv("GAUTH_USE_JWT_LIB")
	t.Setenv("GAUTH_TOKEN_SIG_MODE", "hmac") // force legacy HMAC to ensure JWKS suppression
	srv := NewBetaServer(":0")
	t.Cleanup(func() { srv.Shutdown() })
	w := performRequest(srv.router, "GET", "/.well-known/gauth-configuration")
	if w.Code != 200 {
		t.Fatalf("status %d", w.Code)
	}
	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("json: %v", err)
	}
	required := []string{"version", "schema_version", "future_version", "deprecated_fields", "implementation", "issuer", "token_algorithms", "policy_endpoints", "poa_endpoints", "audit_endpoints", "revocation_endpoints", "revocation_support", "multi_signature_poa", "capabilities", "documentation", "jwks_uri", "introspection_endpoint", "key_rotation", "anchoring"}
	if len(body) < len(required) {
		t.Fatalf("expected at least %d keys got %d", len(required), len(body))
	}
	for _, k := range required {
		if _, ok := body[k]; !ok {
			t.Fatalf("missing key %s", k)
		}
	}
	if body["jwks_uri"] != "" {
		t.Fatalf("jwks_uri should be empty when neither JWT nor eddsa mode enabled; got %v", body["jwks_uri"])
	}

	// Scenario 2: Enable JWT library -> jwks_uri populated.
	t.Setenv("GAUTH_USE_JWT_LIB", "1")
	t.Setenv("GAUTH_JWT_KID", "demo-key")
	srv2 := NewBetaServer(":0")
	t.Cleanup(func() { srv2.Shutdown() })
	w2 := performRequest(srv2.router, "GET", "/.well-known/gauth-configuration")
	var body2 map[string]any
	if err := json.Unmarshal(w2.Body.Bytes(), &body2); err != nil {
		t.Fatalf("json2: %v", err)
	}
	if body2["jwks_uri"] == "" {
		t.Fatalf("jwks_uri should be populated when JWT enabled")
	}

	// Scenario 3: Disable JWT but enable EdDSA -> jwks_uri populated (new behavior after EdDSA introduction).
	os.Unsetenv("GAUTH_USE_JWT_LIB")
	t.Setenv("GAUTH_TOKEN_SIG_MODE", "eddsa")
	srv3 := NewBetaServer(":0")
	t.Cleanup(func() { srv3.Shutdown() })
	w3 := performRequest(srv3.router, "GET", "/.well-known/gauth-configuration")
	var body3 map[string]any
	if err := json.Unmarshal(w3.Body.Bytes(), &body3); err != nil {
		t.Fatalf("json3: %v", err)
	}
	if body3["jwks_uri"] == "" {
		t.Fatalf("jwks_uri should be populated when eddsa mode enabled")
	}
}

// TestDiscoveryAnchoringFields ensures anchoring fields populate when memory anchor enabled.
func TestDiscoveryAnchoringFields(t *testing.T) {
	t.Setenv("GAUTH_ANCHOR_PROVIDER", "memory")
	srv := NewBetaServer(":0")
	t.Cleanup(func() { srv.Shutdown() })
	w := performRequest(srv.router, "GET", "/.well-known/gauth-configuration")
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
	t.Setenv("GAUTH_MULTI_SIG_THRESHOLD", "5")
	t.Setenv("GAUTH_MULTI_SIG_WEIGHTS", "signerA=3,signerB=2")
	srv := NewBetaServer(":0")
	t.Cleanup(func() { srv.Shutdown() })
	w := performRequest(srv.router, "GET", "/.well-known/gauth-configuration")
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
	t.Setenv("GAUTH_MULTI_SIG_THRESHOLD", "5")
	t.Setenv("GAUTH_MULTI_SIG_WEIGHTS", "signerA=2,signerB=2") // total 4 < threshold 5
	srv := NewBetaServer(":0")
	t.Cleanup(func() { srv.Shutdown() })
	w := performRequest(srv.router, "GET", "/.well-known/gauth-configuration")
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

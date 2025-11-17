package web

import (
	"encoding/json"
	"testing"
)

// TestEdDSAPublicKeyEndpointDisabled ensures endpoint returns configured=false when EdDSA mode not enabled.
func TestEdDSAPublicKeyEndpointDisabled(t *testing.T) {
	// Explicitly force non-EdDSA mode to avoid leakage from previous tests that may have set it.
	// Some other package tests use os.Setenv without unset, so we defensively override here.
	t.Setenv("GAUTH_TOKEN_SIG_MODE", "hmac")
	srv := NewBetaServer(":0")
	t.Cleanup(func() { srv.Shutdown() })
	w := performRequest(srv.router, "GET", "/api/v1/beta/keys/eddsa")
	if w.Code != 200 {
		t.Fatalf("status %d", w.Code)
	}
	var resp struct {
		Success    bool   `json:"success"`
		Configured bool   `json:"configured"`
		Kid        string `json:"kid"`
		PublicKey  string `json:"public_key"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !resp.Success {
		t.Fatalf("success false")
	}
	if resp.Configured {
		t.Fatalf("expected configured=false when eddsa disabled")
	}
	if resp.Kid != "" || resp.PublicKey != "" {
		t.Fatalf("unexpected key data when disabled")
	}
}

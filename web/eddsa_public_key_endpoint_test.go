package web

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"testing"
)

// TestEdDSAPublicKeyEndpoint validates exposure of active EdDSA public key when mode enabled.
func TestEdDSAPublicKeyEndpoint(t *testing.T) {
	t.Setenv("GAUTH_TOKEN_SIG_MODE", "eddsa")
	srv := NewBetaServer(":0")
	w := performRequest(srv.router, "GET", "/api/v1/beta/keys/eddsa")
	if w.Code != 200 {
		t.Fatalf("status %d body=%s", w.Code, w.Body.String())
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
	if !resp.Success || !resp.Configured {
		t.Fatalf("expected configured success response: %#v", resp)
	}
	if resp.Kid == "" {
		t.Fatalf("kid empty")
	}
	if resp.PublicKey == "" {
		t.Fatalf("public_key empty")
	}
	pkBytes, err := base64.RawStdEncoding.DecodeString(resp.PublicKey)
	if err != nil {
		t.Fatalf("public_key base64 decode: %v", err)
	}
	if len(pkBytes) != ed25519.PublicKeySize {
		t.Fatalf("public key size mismatch got=%d expected=%d", len(pkBytes), ed25519.PublicKeySize)
	}
}

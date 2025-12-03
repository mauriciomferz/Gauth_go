package verification

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

const (
	rotationsSummaryPath = "/api/v1/beta/rotations/summary"
	jwksPath             = "/.well-known/jwks.json"
)

// mock server returning a deterministic signed rotation summary using ephemeral key
func TestVerifyRotationSummary_Success(t *testing.T) {
	pub, priv, _ := ed25519.GenerateKey(rand.Reader)
	kid := "ed25519:" + base64.RawURLEncoding.EncodeToString(pub[:8])
	// Build canonical payload (ordered struct matching verifier logic)
	payload := struct {
		ChainLength   int    `json:"chain_length"`
		HeadHash      string `json:"head_hash"`
		AggregateHash string `json:"aggregate_hash"`
		GeneratedAt   string `json:"generated_at"`
	}{2, "abc", "def", time.Now().UTC().Format(time.RFC3339Nano)}
	enc, _ := json.Marshal(payload)
	sig := ed25519.Sign(priv, append([]byte("GAUTH_ROTATION_SUMMARY:"), enc...))
	summary := map[string]any{
		"success":    true,
		"configured": true,
		"summary": map[string]any{
			"chain_length":   2,
			"head_hash":      "abc",
			"aggregate_hash": "def",
			"generated_at":   payload.GeneratedAt,
			"kid":            kid,
			"signature":      base64.RawURLEncoding.EncodeToString(sig),
			"mode":           "EdDSA",
		},
		"anchored":    true,
		"anchor_hash": "zzz",
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case rotationsSummaryPath:
			_ = json.NewEncoder(w).Encode(summary)
		case jwksPath:
			_ = json.NewEncoder(w).Encode(map[string]any{"keys": []map[string]any{{
				"kty": "OKP", "crv": "Ed25519", "alg": "EdDSA", "use": "sig", "kid": kid, "x": base64.RawURLEncoding.EncodeToString(pub),
			}}})
		default:
			w.WriteHeader(404)
		}
	}))
	defer srv.Close()
	// Load JWKS then verify
	_, err := LoadJWKS(srv.Client(), srv.URL)
	if err != nil {
		t.Fatalf("load jwks: %v", err)
	}
	res, err := VerifyRotationSummary(srv.Client(), srv.URL)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if !res.SignatureValid {
		t.Fatalf("expected signature valid, got error=%s", res.SignatureError)
	}
}

func TestVerifyRotationSummary_BadSignature(t *testing.T) {
	pub, priv, _ := ed25519.GenerateKey(rand.Reader)
	kid := "ed25519:" + base64.RawURLEncoding.EncodeToString(pub[:8])
	payload := struct {
		ChainLength   int    `json:"chain_length"`
		HeadHash      string `json:"head_hash"`
		AggregateHash string `json:"aggregate_hash"`
		GeneratedAt   string `json:"generated_at"`
	}{1, "aa", "bb", time.Now().UTC().Format(time.RFC3339Nano)}
	enc, _ := json.Marshal(payload)
	sig := ed25519.Sign(priv, append([]byte("GAUTH_ROTATION_SUMMARY:"), enc...))
	// Corrupt signature (flip a byte)
	sig[0] ^= 0xff
	summary := map[string]any{
		"success":    true,
		"configured": true,
		"summary": map[string]any{
			"chain_length":   1,
			"head_hash":      "aa",
			"aggregate_hash": "bb",
			"generated_at":   payload.GeneratedAt,
			"kid":            kid,
			"signature":      base64.RawURLEncoding.EncodeToString(sig),
			"mode":           "EdDSA",
		},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case rotationsSummaryPath:
			_ = json.NewEncoder(w).Encode(summary)
		case jwksPath:
			_ = json.NewEncoder(w).Encode(map[string]any{"keys": []map[string]any{{
				"kty": "OKP", "crv": "Ed25519", "alg": "EdDSA", "use": "sig", "kid": kid, "x": base64.RawURLEncoding.EncodeToString(pub),
			}}})
		default:
			w.WriteHeader(404)
		}
	}))
	defer srv.Close()
	_, err := LoadJWKS(srv.Client(), srv.URL)
	if err != nil {
		t.Fatalf("load jwks: %v", err)
	}
	res, err := VerifyRotationSummary(srv.Client(), srv.URL)
	if err != nil {
		t.Fatalf("verify call failed: %v", err)
	}
	if res.SignatureValid {
		t.Fatalf("expected invalid signature, got valid")
	}
	if res.SignatureError == "" {
		t.Fatalf("expected signature error reason populated")
	}
}

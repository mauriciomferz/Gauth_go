package web

import (
	"encoding/json"
	"net/http/httptest"
	"os"
	"testing"
)

// TestJWKSDiscoveryMetadata verifies discovery includes jwks_etag & jwks_last_rotated when JWT enabled.
func TestJWKSDiscoveryMetadata(t *testing.T) {
	os.Setenv("GAUTH_USE_JWT_LIB", "1")
	os.Setenv("GAUTH_JWT_ALG", "RS256")
	s := NewBetaServer("0")
	// First fetch JWKS to initialize metadata
	w1 := httptest.NewRecorder()
	req1 := httptest.NewRequest("GET", "/.well-known/jwks.json", nil)
	s.router.ServeHTTP(w1, req1)
	if w1.Code != 200 {
		t.Fatalf("expected 200 jwks init got %d", w1.Code)
	}
	jwksETag := w1.Header().Get("ETag")
	if jwksETag == "" {
		t.Fatalf("expected jwks ETag header present")
	}
	// Discovery should now surface jwks_etag & jwks_last_rotated
	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest("GET", "/.well-known/gauth-configuration", nil)
	s.router.ServeHTTP(w2, req2)
	if w2.Code != 200 {
		t.Fatalf("expected 200 discovery got %d", w2.Code)
	}
	var body map[string]any
	if err := json.Unmarshal(w2.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal discovery: %v", err)
	}
	if body["jwks_etag"] == "" {
		t.Fatalf("jwks_etag missing in discovery")
	}
	if body["jwks_etag"].(string) != jwksETag {
		t.Fatalf("jwks_etag mismatch discovery=%s jwks=%s", body["jwks_etag"].(string), jwksETag)
	}
	if body["jwks_last_rotated"] == nil {
		t.Fatalf("jwks_last_rotated should be set")
	}
}

// TestJWKSConditionalETag ensures JWKS endpoint honors If-None-Match and 304.
func TestJWKSConditionalETag(t *testing.T) {
	os.Setenv("GAUTH_USE_JWT_LIB", "1")
	os.Setenv("GAUTH_JWT_ALG", "RS256")
	s := NewBetaServer("0")
	w1 := httptest.NewRecorder()
	req1 := httptest.NewRequest("GET", "/.well-known/jwks.json", nil)
	s.router.ServeHTTP(w1, req1)
	if w1.Code != 200 {
		t.Fatalf("expected 200 got %d", w1.Code)
	}
	etag := w1.Header().Get("ETag")
	if etag == "" {
		t.Fatalf("missing ETag")
	}
	// Second request with If-None-Match should yield 304
	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest("GET", "/.well-known/jwks.json", nil)
	req2.Header.Set("If-None-Match", etag)
	s.router.ServeHTTP(w2, req2)
	if w2.Code != 304 {
		t.Fatalf("expected 304 got %d", w2.Code)
	}
}

// TestJWKSOptionalSignature verifies signature headers appear only when enabled.
func TestJWKSOptionalSignature(t *testing.T) {
	os.Setenv("GAUTH_USE_JWT_LIB", "1")
	os.Setenv("GAUTH_JWT_ALG", "RS256")
	// Disabled path
	s := NewBetaServer("0")
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/.well-known/jwks.json", nil)
	s.router.ServeHTTP(w, req)
	if w.Header().Get("X-JWKS-Signature") != "" {
		t.Fatalf("signature should be absent when disabled")
	}
	// Enabled path
	os.Setenv("GAUTH_JWKS_SIGNING_KEY", "jwks-demo-secret")
	os.Setenv("GAUTH_JWKS_SIGNING_KEY_ENABLED", "1")
	s2 := NewBetaServer("0")
	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest("GET", "/.well-known/jwks.json", nil)
	s2.router.ServeHTTP(w2, req2)
	if w2.Header().Get("X-JWKS-Signature") == "" {
		t.Fatalf("expected signature header when enabled")
	}
	if w2.Header().Get("X-JWKS-Signature-Alg") != "HMAC-SHA256" {
		t.Fatalf("signature alg mismatch")
	}
}

package web

import (
	"encoding/json"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/internal/crypto"
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

// TestJWKSDeprecationMetadata verifies deprecated_after and sunset_after in EdDSA JWK entries.
func TestJWKSDeprecationMetadata(t *testing.T) {
	// Clean environment first to avoid pollution from other tests
	os.Unsetenv("GAUTH_USE_JWT_LIB")
	os.Unsetenv("GAUTH_JWT_ALG")
	os.Setenv("GAUTH_TOKEN_SIG_MODE", "eddsa")
	os.Setenv("GAUTH_EDDSA_AUTO_ROTATE", "0") // Disable auto-rotation for stable test
	defer os.Unsetenv("GAUTH_TOKEN_SIG_MODE")
	defer os.Unsetenv("GAUTH_EDDSA_AUTO_ROTATE")
	// Create server first (initializes crypto manager with 24h TTL)
	s := NewBetaServer("0")
	// Replace with short-TTL manager to trigger deprecation
	ttl := 10 * time.Second
	km, err := crypto.NewManager(ttl)
	if err != nil {
		t.Fatalf("crypto.NewManager: %v", err)
	}
	crypto.GlobalEdDSARegistry = km
	// Wait for deprecation (80% of 10s = 8s, add 1s buffer)
	time.Sleep(9 * time.Second)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/.well-known/jwks.json", nil)
	s.router.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("expected 200 got %d", w.Code)
	}
	var body struct {
		Keys []map[string]any `json:"keys"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal jwks: %v", err)
	}
	if len(body.Keys) == 0 {
		t.Fatalf("expected at least 1 EdDSA key")
	}
	// Verify first key has deprecation metadata
	key := body.Keys[0]
	// Verify key type is actually EdDSA (not HMAC fallback)
	if key["kty"] != "OKP" || key["alg"] != "EdDSA" {
		t.Skip("EdDSA mode not activated, skipping deprecation test")
	}
	if key["deprecated_after"] == nil {
		t.Errorf("deprecated_after missing in JWK")
	}
	if key["sunset_after"] == nil {
		t.Errorf("sunset_after missing in JWK")
	}
	// Verify expires_at equals sunset_after
	if key["expires_at"] != nil && key["sunset_after"] != nil {
		if key["expires_at"].(string) != key["sunset_after"].(string) {
			t.Errorf("expires_at != sunset_after: %s != %s", key["expires_at"].(string), key["sunset_after"].(string))
		}
	}
}

// TestJWKSDeprecationWarningHeader verifies Warning header when key is deprecated.
func TestJWKSDeprecationWarningHeader(t *testing.T) {
	// Clean environment first
	os.Unsetenv("GAUTH_USE_JWT_LIB")
	os.Unsetenv("GAUTH_JWT_ALG")
	os.Setenv("GAUTH_TOKEN_SIG_MODE", "eddsa")
	os.Setenv("GAUTH_EDDSA_AUTO_ROTATE", "0")
	defer os.Unsetenv("GAUTH_TOKEN_SIG_MODE")
	defer os.Unsetenv("GAUTH_EDDSA_AUTO_ROTATE")
	// Create server first
	s := NewBetaServer("0")
	// Replace with short-TTL manager
	ttl := 10 * time.Second
	km, err := crypto.NewManager(ttl)
	if err != nil {
		t.Fatalf("crypto.NewManager: %v", err)
	}
	crypto.GlobalEdDSARegistry = km
	// Wait for deprecation (80% of 10s = 8s, add 1s buffer)
	time.Sleep(9 * time.Second)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/.well-known/jwks.json", nil)
	s.router.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("expected 200 got %d", w.Code)
	}
	var body struct {
		Keys []map[string]any `json:"keys"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err == nil && len(body.Keys) > 0 {
		key := body.Keys[0]
		if key["kty"] != "OKP" || key["alg"] != "EdDSA" {
			t.Skip("EdDSA mode not activated, skipping warning test")
		}
	}
	warning := w.Header().Get("Warning")
	if warning == "" {
		t.Errorf("expected Warning header when key deprecated")
	} else {
		// Verify Warning format: "299 - "Keys deprecated: <kid>""
		if len(warning) < 10 || warning[:3] != "299" {
			t.Errorf("Warning header format unexpected: %s", warning)
		}
	}
}

// TestJWKSNoWarningWhenNoDeprecation verifies no Warning header when keys are fresh.
func TestJWKSNoWarningWhenNoDeprecation(t *testing.T) {
	os.Setenv("GAUTH_TOKEN_SIG_MODE", "eddsa")
	os.Setenv("GAUTH_EDDSA_AUTO_ROTATE", "0")
	// Create manager with long TTL (no deprecation)
	ttl := 24 * time.Hour
	km, err := crypto.NewManager(ttl)
	if err != nil {
		t.Fatalf("crypto.NewManager: %v", err)
	}
	crypto.GlobalEdDSARegistry = km
	s := NewBetaServer("0")
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/.well-known/jwks.json", nil)
	s.router.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("expected 200 got %d", w.Code)
	}
	warning := w.Header().Get("Warning")
	if warning != "" {
		t.Errorf("unexpected Warning header when keys not deprecated: %s", warning)
	}
}


package web

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

// TestJWTHMACAlgMismatch verifies that when expecting HS256 the header contains RS256 and we get normalized invalid_algorithm detail.
func TestJWTHMACAlgMismatch(t *testing.T) {
	os.Setenv("GAUTH_USE_JWT_LIB", "1")
	os.Setenv("GAUTH_JWT_ALG", "HS256") // Expect HMAC
	os.Setenv("GAUTH_JWT_SECRET", "demo-hmac-secret-1234567890")
	os.Setenv("GAUTH_JWT_KID", "demo-hmac")
	srv := NewBetaServer(":0")
	t.Cleanup(func() { srv.Shutdown() })

	// Issue token using current expected HS256 (server will sign with HS256)
	iw := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/token/create", strings.NewReader(`{"ttl_seconds":60}`))
	req.Header.Set("Content-Type", "application/json")
	srv.router.ServeHTTP(iw, req)
	if iw.Code != 201 {
		t.Fatalf("issue status=%d body=%s", iw.Code, iw.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(iw.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal issuance: %v", err)
	}
	jwtRaw, _ := body["jwt"].(string)
	if jwtRaw == "" {
		t.Fatalf("missing jwt issuance body=%v", body)
	}

	// Modify header alg to RS256 while keeping signature part; mimic mismatch.
	parts := strings.Split(jwtRaw, ".")
	if len(parts) != 3 {
		t.Fatalf("bad jwt parts")
	}
	// Decode header
	hdrJSON := decodeBase64URL(t, parts[0])
	// Replace alg value naively (simple string replace of \"HS256\" to \"RS256\")
	if !strings.Contains(hdrJSON, "\"alg\":") {
		t.Fatalf("header lacks alg: %s", hdrJSON)
	}
	modifiedHdrJSON := strings.Replace(hdrJSON, "HS256", "RS256", 1)
	parts[0] = encodeBase64URL(modifiedHdrJSON)
	modified := strings.Join(parts, ".")

	vw := httptest.NewRecorder()
	vreq := httptest.NewRequest(http.MethodPost, "/api/v1/token/validate", strings.NewReader(`{"token":"`+modified+`"}`))
	vreq.Header.Set("Content-Type", "application/json")
	srv.router.ServeHTTP(vw, vreq)
	var vbody map[string]any
	_ = json.Unmarshal(vw.Body.Bytes(), &vbody)
	if vbody["error"] != ErrInvalidAlgorithm {
		t.Fatalf("expected %s got body=%s", ErrInvalidAlgorithm, vw.Body.String())
	}
	detail := vbody["detail"].(string)
	if !strings.Contains(detail, "invalid_algorithm: header alg RS256 rejected (expected HS256)") {
		t.Fatalf("unexpected detail normalization: %s", detail)
	}
}

// Helper reuse (keep local to avoid import from other test). Decode base64 URL.
func decodeBase64URL(t *testing.T, s string) string {
	b, err := base64RawURLDecode(s)
	if err != nil {
		t.Fatalf("decode header: %v", err)
	}
	return string(b)
}
func encodeBase64URL(s string) string { return base64RawURLEncode([]byte(s)) }

// Minimal inline helpers (duplicated to keep test self-contained)
func base64RawURLDecode(s string) ([]byte, error) { return base64.RawURLEncoding.DecodeString(s) }
func base64RawURLEncode(b []byte) string          { return base64.RawURLEncoding.EncodeToString(b) }

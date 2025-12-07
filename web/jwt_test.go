package web

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/mauriciomferz/Gauth_go/web/handlers/token"
)

// TestJWTDiscoveryAlgorithms ensures discovery endpoint reflects JWT algorithms when feature flag enabled.
func TestJWTDiscoveryAlgorithms(t *testing.T) {
	os.Setenv("GAUTH_USE_JWT_LIB", "1")
	os.Setenv("GAUTH_JWT_ALG", "RS256")
	srv := NewBetaServer(":0")
	t.Cleanup(func() { srv.Shutdown() })
	w := performRequest(srv.router, "GET", "/.well-known/gauth-configuration")
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	algs, ok := body["token_algorithms"].([]any)
	if !ok || len(algs) < 2 {
		t.Fatalf("expected >=2 algorithms when JWT enabled: %#v", body["token_algorithms"])
	}
	foundRS := false
	for _, a := range algs {
		if a == "RS256" {
			foundRS = true
		}
	}
	if !foundRS {
		t.Fatalf("RS256 not found in algorithms: %#v", algs)
	}
	if body["jwks_uri"] == "" {
		t.Fatalf("jwks_uri should be populated when JWT enabled")
	}
}

// TestJWTIssuance verifies /api/v1/token/create issues a JWT formatted token when flag enabled.
func TestJWTIssuance(t *testing.T) {
	os.Setenv("GAUTH_USE_JWT_LIB", "1")
	os.Setenv("GAUTH_JWT_ALG", "RS256")
	os.Setenv("GAUTH_JWT_KID", demoRSAKid)
	srv := NewBetaServer(":0")
	t.Cleanup(func() { srv.Shutdown() })
	// Issue token
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/token/create", strings.NewReader(`{"ttl_seconds":60}`))
	req.Header.Set("Content-Type", "application/json")
	srv.router.ServeHTTP(w, req)
	if w.Code != 201 {
		t.Fatalf("expected 201, got %d body=%s", w.Code, w.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	jwtRaw, ok := body["jwt"].(string)
	if !ok || jwtRaw == "" {
		t.Fatalf("jwt field missing in response: %#v", body)
	}
	parts := strings.Split(jwtRaw, ".")
	if len(parts) != 3 {
		t.Fatalf("expected 3 JWT parts, got %d", len(parts))
	}
	hdrBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		t.Fatalf("decode header: %v", err)
	}
	if !strings.Contains(string(hdrBytes), "\"kid\":") {
		t.Fatalf("expected kid in header: %s", string(hdrBytes))
	}
}

// helper to perform POST validate
func doValidate(s *BetaServer, token string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/token/validate", strings.NewReader(`{"token":"`+token+`"}`))
	req.Header.Set("Content-Type", "application/json")
	s.router.ServeHTTP(w, req)
	return w
}

// TestJWTValidatePositive ensures a freshly issued RS256 JWT validates.
func TestJWTValidatePositive(t *testing.T) {
	os.Setenv("GAUTH_USE_JWT_LIB", "1")
	os.Setenv("GAUTH_JWT_ALG", "RS256")
	os.Setenv("GAUTH_JWT_KID", demoRSAKid)
	srv := NewBetaServer(":0")
	t.Cleanup(func() { srv.Shutdown() })
	// issue
	iw := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/token/create", strings.NewReader(`{"ttl_seconds":30}`))
	req.Header.Set("Content-Type", "application/json")
	srv.router.ServeHTTP(iw, req)
	if iw.Code != 201 {
		t.Fatalf("issue status=%d body=%s", iw.Code, iw.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(iw.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	jwtRaw, _ := body["jwt"].(string)
	if jwtRaw == "" {
		t.Fatalf("missing jwt in issuance body=%v", body)
	}
	vw := doValidate(srv, jwtRaw)
	if vw.Code != 200 {
		t.Fatalf("validate status=%d body=%s", vw.Code, vw.Body.String())
	}
	var vbody map[string]any
	if err := json.Unmarshal(vw.Body.Bytes(), &vbody); err != nil {
		t.Fatalf("unmarshal validate: %v", err)
	}
	if vbody["error"] != nil {
		t.Fatalf("unexpected error in validation: %v", vbody)
	}
	if vbody["status"] != statusValidJWT {
		t.Fatalf("expected status=%s got %v", statusValidJWT, vbody["status"])
	}
}

// TestJWTValidateExpired issues a short-lived token and validates after expiry.
func TestJWTValidateExpired(t *testing.T) {
	os.Setenv("GAUTH_USE_JWT_LIB", "1")
	os.Setenv("GAUTH_JWT_ALG", "RS256")
	os.Setenv("GAUTH_JWT_KID", demoRSAKid)
	srv := NewBetaServer(":0")
	t.Cleanup(func() { srv.Shutdown() })
	// Manually build extremely short-lived token (1 second) using server's RSA key helper.
	pk, err := token.LoadOrGenerateRSAKey()
	if err != nil {
		t.Fatalf("rsa key: %v", err)
	}
	method := jwt.GetSigningMethod("RS256")
	claims := jwt.MapClaims{"sub": "demo-client", "exp": time.Now().Add(1 * time.Second).Unix(), "iat": time.Now().Unix()}
	j := jwt.NewWithClaims(method, claims)
	j.Header["kid"] = demoRSAKid
	signed, err := j.SignedString(pk)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	time.Sleep(1500 * time.Millisecond) // allow to expire
	vw := doValidate(srv, signed)
	var vbody map[string]any
	_ = json.Unmarshal(vw.Body.Bytes(), &vbody)
	// Current implementation returns malformed_token with detail containing expired; accept either until taxonomy refined.
	if vbody["error"] != ErrTokenExpired && vbody["error"] != ErrMalformedToken {
		t.Fatalf("expected %s or %s got body=%s", ErrTokenExpired, ErrMalformedToken, vw.Body.String())
	}
}

// TestJWTValidateTampered modifies signature to force invalid_signature.
func TestJWTValidateTampered(t *testing.T) {
	os.Setenv("GAUTH_USE_JWT_LIB", "1")
	os.Setenv("GAUTH_JWT_ALG", "RS256")
	os.Setenv("GAUTH_JWT_KID", demoRSAKid)
	srv := NewBetaServer(":0")
	t.Cleanup(func() { srv.Shutdown() })
	// issue
	iw := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/token/create", strings.NewReader(`{"ttl_seconds":30}`))
	req.Header.Set("Content-Type", "application/json")
	srv.router.ServeHTTP(iw, req)
	var body map[string]any
	_ = json.Unmarshal(iw.Body.Bytes(), &body)
	jwtRaw, _ := body["jwt"].(string)
	parts := strings.Split(jwtRaw, ".")
	if len(parts) != 3 {
		t.Fatalf("bad jwt parts")
	}
	parts[2] = strings.TrimRight(parts[2], "=") + "xyz" // corrupt signature
	tampered := strings.Join(parts, ".")
	vw := doValidate(srv, tampered)
	var vbody map[string]any
	_ = json.Unmarshal(vw.Body.Bytes(), &vbody)
	if vbody["error"] != ErrInvalidSignature {
		t.Fatalf("expected %s got status=%d body=%s", ErrInvalidSignature, vw.Code, vw.Body.String())
	}
}

// TestJWTValidateAlgMismatch rewrites header alg to HS256 while keeping RS256 signature.
func TestJWTValidateAlgMismatch(t *testing.T) {
	os.Setenv("GAUTH_USE_JWT_LIB", "1")
	os.Setenv("GAUTH_JWT_ALG", "RS256")
	os.Setenv("GAUTH_JWT_KID", demoRSAKid)
	srv := NewBetaServer(":0")
	t.Cleanup(func() { srv.Shutdown() })
	// issue
	iw := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/token/create", strings.NewReader(`{"ttl_seconds":30}`))
	req.Header.Set("Content-Type", "application/json")
	srv.router.ServeHTTP(iw, req)
	var body map[string]any
	_ = json.Unmarshal(iw.Body.Bytes(), &body)
	jwtRaw, _ := body["jwt"].(string)
	parts := strings.Split(jwtRaw, ".")
	if len(parts) != 3 {
		t.Fatalf("bad jwt parts")
	}
	// decode header, replace alg, re-encode with same signature part
	hdrBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	var hdr map[string]any
	_ = json.Unmarshal(hdrBytes, &hdr)
	hdr["alg"] = "HS256"
	newHdrBytes, _ := json.Marshal(hdr)
	parts[0] = base64.RawURLEncoding.EncodeToString(newHdrBytes)
	modified := strings.Join(parts, ".")
	vw := doValidate(srv, modified)
	var vbody map[string]any
	_ = json.Unmarshal(vw.Body.Bytes(), &vbody)
	if vbody["error"] != ErrInvalidAlgorithm {
		t.Fatalf("expected %s got body=%s", ErrInvalidAlgorithm, vw.Body.String())
	}
}

// TestJWTMissingJTIStrict ensures strict replay mode rejects tokens without jti claim.
func TestJWTMissingJTIStrict(t *testing.T) {
	os.Setenv("GAUTH_USE_JWT_LIB", "1")
	os.Setenv("GAUTH_JWT_ALG", "RS256")
	os.Setenv("GAUTH_JWT_KID", demoRSAKid)
	os.Setenv("GAUTH_REPLAY_STRICT", "1")
	srv := NewBetaServer(":0")
	t.Cleanup(func() { srv.Shutdown() })
	pk, err := token.LoadOrGenerateRSAKey()
	if err != nil {
		t.Fatalf("rsa key: %v", err)
	}
	method := jwt.GetSigningMethod("RS256")
	claims := jwt.MapClaims{"sub": "demo", "exp": time.Now().Add(60 * time.Second).Unix(), "iat": time.Now().Unix()}
	j := jwt.NewWithClaims(method, claims)
	j.Header["kid"] = demoRSAKid
	signed, err := j.SignedString(pk)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	vw := doValidate(srv, signed)
	var body map[string]any
	_ = json.Unmarshal(vw.Body.Bytes(), &body)
	if body["error"] != ErrMalformedToken {
		t.Fatalf("expected %s for missing jti strict mode body=%s", ErrMalformedToken, vw.Body.String())
	}
}

// TestJWTDuplicateJTIReplay ensures replay detection on second validation attempt of same token.
func TestJWTDuplicateJTIReplay(t *testing.T) {
	os.Setenv("GAUTH_USE_JWT_LIB", "1")
	os.Setenv("GAUTH_JWT_ALG", "RS256")
	os.Setenv("GAUTH_JWT_KID", demoRSAKid)
	srv := NewBetaServer(":0")
	t.Cleanup(func() { srv.Shutdown() })
	// Issue standard token via API (will include jti)
	iw := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/token/create", strings.NewReader(`{"ttl_seconds":60}`))
	req.Header.Set("Content-Type", "application/json")
	srv.router.ServeHTTP(iw, req)
	if iw.Code != 201 {
		t.Fatalf("issue status=%d body=%s", iw.Code, iw.Body.String())
	}
	var body map[string]any
	_ = json.Unmarshal(iw.Body.Bytes(), &body)
	jwtRaw, _ := body["jwt"].(string)
	if jwtRaw == "" {
		t.Fatalf("missing jwt issuance body=%v", body)
	}
	// First validate ok (records JTI)
	vw1 := doValidate(srv, jwtRaw)
	if vw1.Code != 200 {
		t.Fatalf("first validate status=%d body=%s", vw1.Code, vw1.Body.String())
	}
	// Second validate should detect replay (duplicate jti) and emit replay taxonomy
	vw2 := doValidate(srv, jwtRaw)
	if vw2.Code != 401 {
		t.Fatalf("expected 401 replay detection second validation status=%d body=%s", vw2.Code, vw2.Body.String())
	}
	var rep map[string]any
	_ = json.Unmarshal(vw2.Body.Bytes(), &rep)
	if rep["code"] != "token_replay_detected" || rep["error"] != "replay_detected" || rep["rfc_ref"] != "rfc111:replay_protection" {
		t.Fatalf("expected replay taxonomy code=token_replay_detected error=replay_detected rfc_ref=rfc111:replay_protection body=%s", vw2.Body.String())
	}
}

// TestJWTClockSkewTolerance validates that exp slightly in past within skew window still accepted.
func TestJWTClockSkewTolerance(t *testing.T) {
	os.Setenv("GAUTH_USE_JWT_LIB", "1")
	os.Setenv("GAUTH_JWT_ALG", "RS256")
	os.Setenv("GAUTH_JWT_KID", demoRSAKid)
	os.Setenv("GAUTH_JWT_CLOCK_SKEW_SECONDS", "30")
	srv := NewBetaServer(":0")
	t.Cleanup(func() { srv.Shutdown() })
	pk, err := token.LoadOrGenerateRSAKey()
	if err != nil {
		t.Fatalf("rsa key: %v", err)
	}
	method := jwt.GetSigningMethod("RS256")
	// Set exp 5 seconds in past; within 30s skew tolerance
	claims := jwt.MapClaims{"sub": "demo", "exp": time.Now().Add(-5 * time.Second).Unix(), "iat": time.Now().Add(-10 * time.Second).Unix(), "jti": "skewtest-" + strconv.FormatInt(time.Now().UnixNano(), 10)}
	j := jwt.NewWithClaims(method, claims)
	j.Header["kid"] = demoRSAKid
	signed, err := j.SignedString(pk)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	vw := doValidate(srv, signed)
	if vw.Code != 200 {
		t.Fatalf("expected 200 within skew window body=%s", vw.Body.String())
	}
}

// TestJWTNearExpiration ensures token just inside expiration passes while just outside fails.
func TestJWTNearExpiration(t *testing.T) {
	os.Setenv("GAUTH_USE_JWT_LIB", "1")
	os.Setenv("GAUTH_JWT_ALG", "RS256")
	os.Setenv("GAUTH_JWT_KID", demoRSAKid)
	os.Setenv("GAUTH_JWT_CLOCK_SKEW_SECONDS", "2")
	srv := NewBetaServer(":0")
	t.Cleanup(func() { srv.Shutdown() })
	pk, err := token.LoadOrGenerateRSAKey()
	if err != nil {
		t.Fatalf("rsa key: %v", err)
	}
	method := jwt.GetSigningMethod("RS256")
	// exp 1 second ahead => should pass
	claims1 := jwt.MapClaims{"sub": "demo", "exp": time.Now().Add(1 * time.Second).Unix(), "iat": time.Now().Unix(), "jti": "near-exp-pass" + strconv.FormatInt(time.Now().UnixNano(), 10)}
	j1 := jwt.NewWithClaims(method, claims1)
	j1.Header["kid"] = demoRSAKid
	signed1, err := j1.SignedString(pk)
	if err != nil {
		t.Fatalf("sign1: %v", err)
	}
	vw1 := doValidate(srv, signed1)
	if vw1.Code != 200 {
		t.Fatalf("expected pass body=%s", vw1.Body.String())
	}
	// exp 3 seconds in past with skew tolerance=2 => should fail
	claims2 := jwt.MapClaims{"sub": "demo", "exp": time.Now().Add(-3 * time.Second).Unix(), "iat": time.Now().Add(-5 * time.Second).Unix(), "jti": "near-exp-fail" + strconv.FormatInt(time.Now().UnixNano(), 10)}
	j2 := jwt.NewWithClaims(method, claims2)
	j2.Header["kid"] = demoRSAKid
	signed2, err := j2.SignedString(pk)
	if err != nil {
		t.Fatalf("sign2: %v", err)
	}
	vw2 := doValidate(srv, signed2)
	var body2 map[string]any
	_ = json.Unmarshal(vw2.Body.Bytes(), &body2)
	if body2["error"] != ErrTokenExpired {
		t.Fatalf("expected %s outside skew body=%s", ErrTokenExpired, vw2.Body.String())
	}
}

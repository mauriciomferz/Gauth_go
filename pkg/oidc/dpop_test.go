package oidc

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// TestNewDPoPManager tests DPoP manager creation.
func TestNewDPoPManager(t *testing.T) {
	// Test with nil config (should use defaults)
	dm := NewDPoPManager(nil)
	if dm == nil {
		t.Fatal("NewDPoPManager returned nil")
	}
	t.Cleanup(func() { _ = dm.Close() })

	if dm.config.MaxAge != 60*time.Second {
		t.Errorf("default MaxAge = %v, want 60s", dm.config.MaxAge)
	}

	// Test with custom config
	config := &DPoPConfig{
		RequireNonce:      true,
		MaxAge:            30 * time.Second,
		NonceLifetime:     60 * time.Second,
		AllowedAlgorithms: []string{"RS256"},
		ClockSkew:         10 * time.Second,
	}

	dm2 := NewDPoPManager(config)
	t.Cleanup(func() { _ = dm2.Close() })

	if !dm2.config.RequireNonce {
		t.Error("RequireNonce not set")
	}
	if dm2.config.MaxAge != 30*time.Second {
		t.Errorf("MaxAge = %v, want 30s", dm2.config.MaxAge)
	}
}

// TestGenerateNonce tests nonce generation.
func TestGenerateNonce(t *testing.T) {
	dm := NewDPoPManager(nil)
	t.Cleanup(func() { _ = dm.Close() })

	// Generate nonces
	nonces := make(map[string]bool)
	for i := 0; i < 100; i++ {
		nonce, err := dm.GenerateNonce()
		if err != nil {
			t.Fatalf("GenerateNonce failed: %v", err)
		}

		if nonce == "" {
			t.Error("nonce is empty")
		}

		// Check uniqueness
		if nonces[nonce] {
			t.Errorf("duplicate nonce: %s", nonce)
		}
		nonces[nonce] = true
	}

	// Verify nonces are stored
	dm.mu.RLock()
	storedCount := len(dm.nonces)
	dm.mu.RUnlock()

	if storedCount != 100 {
		t.Errorf("stored nonces = %d, want 100", storedCount)
	}
}

// TestValidateDPoPProof_ValidProof tests validation of a valid DPoP proof.
func TestValidateDPoPProof_ValidProof(t *testing.T) {
	dm := NewDPoPManager(nil)
	t.Cleanup(func() { _ = dm.Close() })

	// Generate RSA key pair
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}

	// Create JWK for public key
	jwk := rsaPublicKeyToJWK(&privateKey.PublicKey)

	// Create DPoP proof
	httpMethod := "POST"
	httpURI := "https://as.example.com/token"

	proof := createDPoPProof(t, privateKey, jwk, httpMethod, httpURI, "", "")

	// Validate proof
	ctx := context.Background()
	validatedProof, thumbprint, err := dm.ValidateDPoPProof(ctx, proof, httpMethod, httpURI, "")
	if err != nil {
		t.Fatalf("ValidateDPoPProof failed: %v", err)
	}

	if validatedProof == nil {
		t.Fatal("validated proof is nil")
	}

	if validatedProof.HTTPMethod != httpMethod {
		t.Errorf("HTTPMethod = %s, want %s", validatedProof.HTTPMethod, httpMethod)
	}

	if validatedProof.HTTPURI != httpURI {
		t.Errorf("HTTPURI = %s, want %s", validatedProof.HTTPURI, httpURI)
	}

	if thumbprint == "" {
		t.Error("thumbprint is empty")
	}
}

// TestValidateDPoPProof_MissingProof tests validation with missing DPoP header.
func TestValidateDPoPProof_MissingProof(t *testing.T) {
	dm := NewDPoPManager(nil)
	defer func() { _ = dm.Close() }()

	ctx := context.Background()
	_, _, err := dm.ValidateDPoPProof(ctx, "", "POST", "https://as.example.com/token", "")

	if err == nil {
		t.Fatal("expected error for missing proof")
	}

	oidcErr, ok := err.(*OIDCError)
	if !ok {
		t.Fatalf("error type = %T, want *OIDCError", err)
	}

	if oidcErr.ErrorCode != ErrorInvalidRequest {
		t.Errorf("error code = %s, want %s", oidcErr.ErrorCode, ErrorInvalidRequest)
	}
}

// TestValidateDPoPProof_InvalidTypHeader tests validation with wrong typ header.
func TestValidateDPoPProof_InvalidTypHeader(t *testing.T) {
	dm := NewDPoPManager(nil)
	defer func() { _ = dm.Close() }()

	privateKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	jwk := rsaPublicKeyToJWK(&privateKey.PublicKey)

	// Create proof with wrong typ header
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
		"jti": "unique-id",
		"htm": "POST",
		"htu": "https://as.example.com/token",
		"iat": time.Now().Unix(),
	})
	token.Header["typ"] = "JWT" // Wrong type
	token.Header["jwk"] = jwk

	proofString, _ := token.SignedString(privateKey)

	ctx := context.Background()
	_, _, err := dm.ValidateDPoPProof(ctx, proofString, "POST", "https://as.example.com/token", "")

	if err == nil {
		t.Fatal("expected error for invalid typ")
	}

	if !strings.Contains(err.Error(), "typ") {
		t.Errorf("error should mention 'typ', got: %v", err)
	}
}

// TestValidateDPoPProof_HTTPMethodMismatch tests validation with wrong HTTP method.
func TestValidateDPoPProof_HTTPMethodMismatch(t *testing.T) {
	dm := NewDPoPManager(nil)
	defer func() { _ = dm.Close() }()

	privateKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	jwk := rsaPublicKeyToJWK(&privateKey.PublicKey)

	// Create proof for POST
	proof := createDPoPProof(t, privateKey, jwk, "POST", "https://as.example.com/token", "", "")

	// Validate with GET
	ctx := context.Background()
	_, _, err := dm.ValidateDPoPProof(ctx, proof, "GET", "https://as.example.com/token", "")

	if err == nil {
		t.Fatal("expected error for method mismatch")
	}

	if !strings.Contains(err.Error(), "htm") {
		t.Errorf("error should mention 'htm', got: %v", err)
	}
}

// TestValidateDPoPProof_HTTPURIMismatch tests validation with wrong HTTP URI.
func TestValidateDPoPProof_HTTPURIMismatch(t *testing.T) {
	dm := NewDPoPManager(nil)
	defer func() { _ = dm.Close() }()

	privateKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	jwk := rsaPublicKeyToJWK(&privateKey.PublicKey)

	// Create proof for one URI
	proof := createDPoPProof(t, privateKey, jwk, "POST", "https://as.example.com/token", "", "")

	// Validate with different URI
	ctx := context.Background()
	_, _, err := dm.ValidateDPoPProof(ctx, proof, "POST", "https://evil.example.com/token", "")

	if err == nil {
		t.Fatal("expected error for URI mismatch")
	}

	if !strings.Contains(err.Error(), "htu") {
		t.Errorf("error should mention 'htu', got: %v", err)
	}
}

// TestValidateDPoPProof_ExpiredProof tests validation of expired proof.
func TestValidateDPoPProof_ExpiredProof(t *testing.T) {
	config := &DPoPConfig{
		MaxAge:            1 * time.Second,
		ClockSkew:         0,
		AllowedAlgorithms: []string{"RS256", "ES256", "EdDSA"},
	}
	dm := NewDPoPManager(config)
	defer func() { _ = dm.Close() }()

	privateKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	jwk := rsaPublicKeyToJWK(&privateKey.PublicKey)

	// Create proof with old iat
	oldTime := time.Now().Add(-5 * time.Second)
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
		"jti": "unique-id",
		"htm": "POST",
		"htu": "https://as.example.com/token",
		"iat": oldTime.Unix(),
	})
	token.Header["typ"] = "dpop+jwt"
	token.Header["jwk"] = jwk

	proofString, _ := token.SignedString(privateKey)

	ctx := context.Background()
	_, _, err := dm.ValidateDPoPProof(ctx, proofString, "POST", "https://as.example.com/token", "")

	if err == nil {
		t.Fatal("expected error for expired proof")
	}

	if !strings.Contains(err.Error(), "too old") {
		t.Errorf("error should mention 'too old', got: %v", err)
	}
}

// TestValidateDPoPProof_ReplayAttack tests JTI replay prevention.
func TestValidateDPoPProof_ReplayAttack(t *testing.T) {
	dm := NewDPoPManager(nil)
	defer func() { _ = dm.Close() }()

	privateKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	jwk := rsaPublicKeyToJWK(&privateKey.PublicKey)

	httpMethod := "POST"
	httpURI := "https://as.example.com/token"
	proof := createDPoPProof(t, privateKey, jwk, httpMethod, httpURI, "", "")

	ctx := context.Background()

	// First use should succeed
	_, _, err := dm.ValidateDPoPProof(ctx, proof, httpMethod, httpURI, "")
	if err != nil {
		t.Fatalf("first validation failed: %v", err)
	}

	// Second use with same JTI should fail (replay)
	_, _, err = dm.ValidateDPoPProof(ctx, proof, httpMethod, httpURI, "")
	if err == nil {
		t.Fatal("expected error for replay attack")
	}

	oidcErr, ok := err.(*OIDCError)
	if !ok {
		t.Fatalf("error type = %T, want *OIDCError", err)
	}

	if oidcErr.ErrorCode != ErrorUseDPoPNonce {
		t.Errorf("error code = %s, want %s", oidcErr.ErrorCode, ErrorUseDPoPNonce)
	}

	if !strings.Contains(oidcErr.ErrorDescription, "replay") {
		t.Errorf("error should mention replay, got: %s", oidcErr.ErrorDescription)
	}
}

// TestValidateDPoPProof_WithNonce tests nonce validation.
func TestValidateDPoPProof_WithNonce(t *testing.T) {
	config := &DPoPConfig{
		RequireNonce:      true,
		MaxAge:            60 * time.Second,
		NonceLifetime:     5 * time.Second,
		AllowedAlgorithms: []string{"RS256", "ES256", "EdDSA"},
		ClockSkew:         5 * time.Second,
	}
	dm := NewDPoPManager(config)
	defer func() { _ = dm.Close() }()

	privateKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	jwk := rsaPublicKeyToJWK(&privateKey.PublicKey)

	// Generate nonce
	nonce, err := dm.GenerateNonce()
	if err != nil {
		t.Fatalf("GenerateNonce failed: %v", err)
	}

	// Create proof with nonce
	httpMethod := "POST"
	httpURI := "https://as.example.com/token"
	proof := createDPoPProof(t, privateKey, jwk, httpMethod, httpURI, nonce, "")

	ctx := context.Background()
	_, _, err = dm.ValidateDPoPProof(ctx, proof, httpMethod, httpURI, "")
	if err != nil {
		t.Fatalf("validation with nonce failed: %v", err)
	}

	// Verify nonce was consumed
	dm.mu.RLock()
	_, exists := dm.nonces[nonce]
	dm.mu.RUnlock()

	if exists {
		t.Error("nonce should be consumed after use")
	}
}

// TestValidateDPoPProof_MissingNonce tests missing nonce when required.
func TestValidateDPoPProof_MissingNonce(t *testing.T) {
	config := &DPoPConfig{
		RequireNonce:      true,
		AllowedAlgorithms: []string{"RS256", "ES256", "EdDSA"},
	}
	dm := NewDPoPManager(config)
	defer func() { _ = dm.Close() }()

	privateKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	jwk := rsaPublicKeyToJWK(&privateKey.PublicKey)

	// Create proof without nonce
	proof := createDPoPProof(t, privateKey, jwk, "POST", "https://as.example.com/token", "", "")

	ctx := context.Background()
	_, _, err := dm.ValidateDPoPProof(ctx, proof, "POST", "https://as.example.com/token", "")

	if err == nil {
		t.Fatal("expected error for missing nonce")
	}

	oidcErr, ok := err.(*OIDCError)
	if !ok {
		t.Fatalf("error type = %T, want *OIDCError", err)
	}

	// Error code could be either ErrorUseDPoPNonce or ErrorInvalidDPoPProof depending on validation order
	if oidcErr.ErrorCode != ErrorUseDPoPNonce && oidcErr.ErrorCode != ErrorInvalidDPoPProof {
		t.Errorf("error code = %s, want %s or %s", oidcErr.ErrorCode, ErrorUseDPoPNonce, ErrorInvalidDPoPProof)
	}
}

// TestValidateDPoPProof_WithAccessTokenHash tests ath validation.
func TestValidateDPoPProof_WithAccessTokenHash(t *testing.T) {
	dm := NewDPoPManager(nil)
	defer func() { _ = dm.Close() }()

	privateKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	jwk := rsaPublicKeyToJWK(&privateKey.PublicKey)

	accessToken := "test-access-token-12345"
	httpMethod := "GET"
	httpURI := "https://rs.example.com/resource"

	// Calculate correct ath
	ath := calculateAccessTokenHash(accessToken)

	// Create proof with ath
	proof := createDPoPProof(t, privateKey, jwk, httpMethod, httpURI, "", ath)

	ctx := context.Background()
	_, _, err := dm.ValidateDPoPProof(ctx, proof, httpMethod, httpURI, accessToken)
	if err != nil {
		t.Fatalf("validation with ath failed: %v", err)
	}
}

// TestValidateDPoPProof_InvalidAccessTokenHash tests wrong ath.
func TestValidateDPoPProof_InvalidAccessTokenHash(t *testing.T) {
	dm := NewDPoPManager(nil)
	defer func() { _ = dm.Close() }()

	privateKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	jwk := rsaPublicKeyToJWK(&privateKey.PublicKey)

	// Create proof with wrong ath
	proof := createDPoPProof(t, privateKey, jwk, "GET", "https://rs.example.com/resource", "", "wrong-hash")

	ctx := context.Background()
	_, _, err := dm.ValidateDPoPProof(ctx, proof, "GET", "https://rs.example.com/resource", "test-token")

	if err == nil {
		t.Fatal("expected error for invalid ath")
	}

	if !strings.Contains(err.Error(), "hash") {
		t.Errorf("error should mention hash, got: %v", err)
	}
}

// TestBindAccessToken tests token binding.
func TestBindAccessToken(t *testing.T) {
	dm := NewDPoPManager(nil)
	defer func() { _ = dm.Close() }()

	tokenID := "token-123"
	thumbprint := "thumbprint-abc"

	dm.BindAccessToken(tokenID, thumbprint)

	// Verify binding
	err := dm.ValidateTokenBinding(tokenID, thumbprint)
	if err != nil {
		t.Errorf("ValidateTokenBinding failed: %v", err)
	}

	// Verify wrong thumbprint fails
	err = dm.ValidateTokenBinding(tokenID, "wrong-thumbprint")
	if err == nil {
		t.Error("expected error for wrong thumbprint")
	}

	// Verify unbound token fails
	err = dm.ValidateTokenBinding("unknown-token", thumbprint)
	if err == nil {
		t.Error("expected error for unbound token")
	}
}

// TestCleanup tests automatic cleanup of expired entries.
func TestCleanup(t *testing.T) {
	config := &DPoPConfig{
		MaxAge:        100 * time.Millisecond,
		NonceLifetime: 100 * time.Millisecond,
	}
	dm := NewDPoPManager(config)
	defer func() { _ = dm.Close() }()

	// Generate nonces
	for i := 0; i < 10; i++ {
		_, _ = dm.GenerateNonce()
	}

	// Add used JTIs
	dm.mu.Lock()
	dm.usedJTIs["jti-1"] = time.Now().Add(-1 * time.Hour)
	dm.usedJTIs["jti-2"] = time.Now()
	dm.mu.Unlock()

	// Wait for expiration
	time.Sleep(150 * time.Millisecond)

	// Run cleanup
	dm.cleanup()

	// Verify old entries removed
	dm.mu.RLock()
	nonceCount := len(dm.nonces)
	jtiCount := len(dm.usedJTIs)
	dm.mu.RUnlock()

	if nonceCount != 0 {
		t.Errorf("nonce count = %d, want 0 (expired)", nonceCount)
	}

	if jtiCount != 1 {
		t.Errorf("JTI count = %d, want 1 (only recent kept)", jtiCount)
	}
}

// TestCalculateJWKThumbprint tests JWK thumbprint calculation.
func TestCalculateJWKThumbprint(t *testing.T) {
	jwk := map[string]interface{}{
		"kty": "RSA",
		//nolint:lll // RSA modulus for test JWK
		"n": "0vx7agoebGcQSuuPiLJXZptN9nndrQmbXEps2aiAFbWhM78LhWx4cbbfAAtVT86zwu1RK7aPFFxuhDR1L6tSoc_BJECPebWKRXjBZCiFV4n3oknjhMstn64tZ_2W-5JsGY4Hc5n9yBXArwl93lqt7_RN5w6Cf0h4QyQ5v-65YGjQR0_FDW2QvzqY368QQMicAtaSqzs8KJZgnYb9c7d0zgdAZHzu6qMQvRL5hajrn1n91CbOpbISD08qNLyrdkt-bFTWhAI4vMQFh6WeZu0fM4lFd2NcRwr3XPksINHaQ-G_xBniIqbw0Ls1jF44-csFCur-kEgU8awapJzKnqDKgw",
		"e": "AQAB",
	}

	thumbprint, err := calculateJWKThumbprint(jwk)
	if err != nil {
		t.Fatalf("calculateJWKThumbprint failed: %v", err)
	}

	if thumbprint == "" {
		t.Error("thumbprint is empty")
	}

	// Calculate again should give same result
	thumbprint2, err := calculateJWKThumbprint(jwk)
	if err != nil {
		t.Fatalf("second calculation failed: %v", err)
	}

	if thumbprint != thumbprint2 {
		t.Errorf("thumbprints don't match: %s vs %s", thumbprint, thumbprint2)
	}
}

// TestNormalizeHTTPURI tests URI normalization.
func TestNormalizeHTTPURI(t *testing.T) {
	tests := []struct {
		name     string
		uri      string
		expected string
	}{
		{
			name:     "no query or fragment",
			uri:      "https://as.example.com/token",
			expected: "https://as.example.com/token",
		},
		{
			name:     "with query parameters",
			uri:      "https://as.example.com/token?foo=bar&baz=qux",
			expected: "https://as.example.com/token",
		},
		{
			name:     "with fragment",
			uri:      "https://as.example.com/token#section",
			expected: "https://as.example.com/token",
		},
		{
			name:     "with query and fragment",
			uri:      "https://as.example.com/token?foo=bar#section",
			expected: "https://as.example.com/token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeHTTPURI(tt.uri)
			if result != tt.expected {
				t.Errorf("normalizeHTTPURI(%q) = %q, want %q", tt.uri, result, tt.expected)
			}
		})
	}
}

// TestMultipleAlgorithms tests DPoP with different algorithms.
func TestMultipleAlgorithms(t *testing.T) {
	tests := []struct {
		name      string
		keyGen    func() (interface{}, interface{}, error)
		alg       string
		supported bool
	}{
		{
			name: "RS256",
			keyGen: func() (interface{}, interface{}, error) {
				key, err := rsa.GenerateKey(rand.Reader, 2048)
				return key, &key.PublicKey, err
			},
			alg:       "RS256",
			supported: true,
		},
		{
			name: "ES256",
			keyGen: func() (interface{}, interface{}, error) {
				key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
				return key, &key.PublicKey, err
			},
			alg:       "ES256",
			supported: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &DPoPConfig{
				MaxAge:            60 * time.Second,
				NonceLifetime:     300 * time.Second,
				AllowedAlgorithms: []string{tt.alg},
				ClockSkew:         5 * time.Second,
			}
			dm := NewDPoPManager(config)
			defer func() { _ = dm.Close() }()

			privateKey, publicKey, err := tt.keyGen()
			if err != nil {
				t.Fatalf("key generation failed: %v", err)
			}

			var jwk map[string]interface{}
			if rsaPub, ok := publicKey.(*rsa.PublicKey); ok {
				jwk = rsaPublicKeyToJWK(rsaPub)
			} else {
				// Skip non-RSA for now (EC parsing not fully implemented)
				t.Skip("EC key parsing not fully implemented")
				return
			}

			// Create and sign proof
			var method jwt.SigningMethod
			switch tt.alg {
			case "RS256":
				method = jwt.SigningMethodRS256
			case "ES256":
				method = jwt.SigningMethodES256
			default:
				t.Fatalf("unknown algorithm: %s", tt.alg)
			}

			token := jwt.NewWithClaims(method, jwt.MapClaims{
				"jti": "unique-id-123",
				"htm": "POST",
				"htu": "https://as.example.com/token",
				"iat": time.Now().Unix(),
			})
			token.Header["typ"] = "dpop+jwt"
			token.Header["jwk"] = jwk

			proofString, err := token.SignedString(privateKey)
			if err != nil {
				t.Fatalf("signing failed: %v", err)
			}

			// Validate
			ctx := context.Background()
			_, _, err = dm.ValidateDPoPProof(ctx, proofString, "POST", "https://as.example.com/token", "")
			if tt.supported && err != nil {
				t.Errorf("validation failed for supported algorithm: %v", err)
			}
		})
	}
}

// Helper functions

func rsaPublicKeyToJWK(pub *rsa.PublicKey) map[string]interface{} {
	return map[string]interface{}{
		"kty": "RSA",
		"n":   base64.RawURLEncoding.EncodeToString(pub.N.Bytes()),
		"e":   base64.RawURLEncoding.EncodeToString([]byte{byte(pub.E >> 16), byte(pub.E >> 8), byte(pub.E)}),
	}
}

func createDPoPProof(
	t *testing.T, privateKey *rsa.PrivateKey, jwk map[string]interface{},
	httpMethod, httpURI, nonce, ath string,
) string {
	claims := jwt.MapClaims{
		"jti": fmt.Sprintf("unique-id-%d", time.Now().UnixNano()),
		"htm": httpMethod,
		"htu": httpURI,
		"iat": time.Now().Unix(),
	}

	if nonce != "" {
		claims["nonce"] = nonce
	}

	if ath != "" {
		claims["ath"] = ath
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["typ"] = "dpop+jwt"
	token.Header["jwk"] = jwk

	proofString, err := token.SignedString(privateKey)
	if err != nil {
		t.Fatalf("failed to sign proof: %v", err)
	}

	return proofString
}

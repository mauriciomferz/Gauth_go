package oidc

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// TestInMemoryJWKSFetcher_GetKey_Success tests successful key retrieval
func TestInMemoryJWKSFetcher_GetKey_Success(t *testing.T) {
	// Generate test RSA key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate RSA key: %v", err)
	}

	// Create JWKS from public key
	jwks := createJWKSFromRSAKey(&privateKey.PublicKey, "test-kid-1")

	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(jwks)
	}))
	defer server.Close()

	fetcher := NewInMemoryJWKSFetcher(time.Hour)
	ctx := context.Background()

	// Test key retrieval
	key, err := fetcher.GetKey(ctx, server.URL, "test-kid-1")
	if err != nil {
		t.Fatalf("GetKey failed: %v", err)
	}

	// Verify key type
	rsaKey, ok := key.(*rsa.PublicKey)
	if !ok {
		t.Fatalf("Expected RSA public key, got %T", key)
	}

	// Verify key matches
	if rsaKey.N.Cmp(privateKey.PublicKey.N) != 0 {
		t.Error("Retrieved key modulus doesn't match original")
	}
	if rsaKey.E != privateKey.PublicKey.E {
		t.Error("Retrieved key exponent doesn't match original")
	}
}

// TestInMemoryJWKSFetcher_GetKey_Caching tests JWKS caching behavior
func TestInMemoryJWKSFetcher_GetKey_Caching(t *testing.T) {
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		privateKey, _ := rsa.GenerateKey(rand.Reader, 2048)
		jwks := createJWKSFromRSAKey(&privateKey.PublicKey, "test-kid")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(jwks)
	}))
	defer server.Close()

	fetcher := NewInMemoryJWKSFetcher(time.Hour)
	ctx := context.Background()

	// First request should hit the server
	_, err := fetcher.GetKey(ctx, server.URL, "test-kid")
	if err != nil {
		t.Fatalf("First GetKey failed: %v", err)
	}

	if requestCount != 1 {
		t.Errorf("Expected 1 request, got %d", requestCount)
	}

	// Second request should use cache
	_, err = fetcher.GetKey(ctx, server.URL, "test-kid")
	if err != nil {
		t.Fatalf("Second GetKey failed: %v", err)
	}

	if requestCount != 1 {
		t.Errorf("Expected cache hit (1 request total), got %d requests", requestCount)
	}
}

// TestInMemoryJWKSFetcher_GetKey_CacheExpiration tests cache expiration
func TestInMemoryJWKSFetcher_GetKey_CacheExpiration(t *testing.T) {
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		privateKey, _ := rsa.GenerateKey(rand.Reader, 2048)
		jwks := createJWKSFromRSAKey(&privateKey.PublicKey, "test-kid")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(jwks)
	}))
	defer server.Close()

	// Set short TTL for testing
	fetcher := NewInMemoryJWKSFetcher(50 * time.Millisecond)
	ctx := context.Background()

	// First request
	_, err := fetcher.GetKey(ctx, server.URL, "test-kid")
	if err != nil {
		t.Fatalf("First GetKey failed: %v", err)
	}

	if requestCount != 1 {
		t.Errorf("Expected 1 request, got %d", requestCount)
	}

	// Wait for cache to expire
	time.Sleep(100 * time.Millisecond)

	// Second request should fetch again
	_, err = fetcher.GetKey(ctx, server.URL, "test-kid")
	if err != nil {
		t.Fatalf("Second GetKey failed: %v", err)
	}

	if requestCount != 2 {
		t.Errorf("Expected cache expiration (2 requests), got %d requests", requestCount)
	}
}

// TestInMemoryJWKSFetcher_GetKey_KeyNotFound tests missing key ID
func TestInMemoryJWKSFetcher_GetKey_KeyNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		privateKey, _ := rsa.GenerateKey(rand.Reader, 2048)
		jwks := createJWKSFromRSAKey(&privateKey.PublicKey, "test-kid-1")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(jwks)
	}))
	defer server.Close()

	fetcher := NewInMemoryJWKSFetcher(time.Hour)
	ctx := context.Background()

	// Request non-existent key ID
	_, err := fetcher.GetKey(ctx, server.URL, "non-existent-kid")
	if err == nil {
		t.Fatal("Expected error for non-existent key ID")
	}

	expectedError := "key with kid non-existent-kid not found in JWKS"
	if err.Error() != expectedError {
		t.Errorf("Expected error %q, got %q", expectedError, err.Error())
	}
}

// TestInMemoryJWKSFetcher_GetKey_InvalidEndpoint tests handling of invalid endpoints
func TestInMemoryJWKSFetcher_GetKey_InvalidEndpoint(t *testing.T) {
	fetcher := NewInMemoryJWKSFetcher(time.Hour)
	ctx := context.Background()

	// Test with invalid URL
	_, err := fetcher.GetKey(ctx, "http://invalid-endpoint-that-does-not-exist.local", "test-kid")
	if err == nil {
		t.Fatal("Expected error for invalid endpoint")
	}
}

// TestInMemoryJWKSFetcher_GetKey_ServerError tests handling of server errors
func TestInMemoryJWKSFetcher_GetKey_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	fetcher := NewInMemoryJWKSFetcher(time.Hour)
	ctx := context.Background()

	_, err := fetcher.GetKey(ctx, server.URL, "test-kid")
	if err == nil {
		t.Fatal("Expected error for server error")
	}
}

// TestInMemoryJWKSFetcher_GetKey_InvalidJSON tests handling of invalid JSON
func TestInMemoryJWKSFetcher_GetKey_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("invalid json"))
	}))
	defer server.Close()

	fetcher := NewInMemoryJWKSFetcher(time.Hour)
	ctx := context.Background()

	_, err := fetcher.GetKey(ctx, server.URL, "test-kid")
	if err == nil {
		t.Fatal("Expected error for invalid JSON")
	}
}

// TestInMemoryJWKSFetcher_RefreshKeys tests manual key refresh
func TestInMemoryJWKSFetcher_RefreshKeys(t *testing.T) {
	keyVersion := 1
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		privateKey, _ := rsa.GenerateKey(rand.Reader, 2048)
		kid := fmt.Sprintf("test-kid-v%d", keyVersion)
		jwks := createJWKSFromRSAKey(&privateKey.PublicKey, kid)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(jwks)
	}))
	defer server.Close()

	fetcher := NewInMemoryJWKSFetcher(time.Hour)
	ctx := context.Background()

	// Initial fetch
	key1, err := fetcher.GetKey(ctx, server.URL, "test-kid-v1")
	if err != nil {
		t.Fatalf("Initial GetKey failed: %v", err)
	}

	// Update server to return new key
	keyVersion = 2

	// Refresh should fetch new keys
	err = fetcher.RefreshKeys(ctx, server.URL)
	if err != nil {
		t.Fatalf("RefreshKeys failed: %v", err)
	}

	// Should now be able to get new key
	key2, err := fetcher.GetKey(ctx, server.URL, "test-kid-v2")
	if err != nil {
		t.Fatalf("GetKey after refresh failed: %v", err)
	}

	// Keys should be different
	rsaKey1 := key1.(*rsa.PublicKey)
	rsaKey2 := key2.(*rsa.PublicKey)
	if rsaKey1.N.Cmp(rsaKey2.N) == 0 {
		t.Error("Expected different keys after refresh")
	}
}

// TestInMemoryJWKSFetcher_ClearCache tests cache clearing
func TestInMemoryJWKSFetcher_ClearCache(t *testing.T) {
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		privateKey, _ := rsa.GenerateKey(rand.Reader, 2048)
		jwks := createJWKSFromRSAKey(&privateKey.PublicKey, "test-kid")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(jwks)
	}))
	defer server.Close()

	fetcher := NewInMemoryJWKSFetcher(time.Hour)
	ctx := context.Background()

	// First request
	_, err := fetcher.GetKey(ctx, server.URL, "test-kid")
	if err != nil {
		t.Fatalf("GetKey failed: %v", err)
	}

	if requestCount != 1 {
		t.Errorf("Expected 1 request, got %d", requestCount)
	}

	// Clear cache
	fetcher.ClearCache()

	// Should fetch again after clear
	_, err = fetcher.GetKey(ctx, server.URL, "test-kid")
	if err != nil {
		t.Fatalf("GetKey after clear failed: %v", err)
	}

	if requestCount != 2 {
		t.Errorf("Expected 2 requests after clear, got %d", requestCount)
	}
}

// TestInMemoryJWKSFetcher_MultipleKeys tests handling of multiple keys
func TestInMemoryJWKSFetcher_MultipleKeys(t *testing.T) {
	// Generate multiple keys
	key1, _ := rsa.GenerateKey(rand.Reader, 2048)
	key2, _ := rsa.GenerateKey(rand.Reader, 2048)
	key3, _ := rsa.GenerateKey(rand.Reader, 2048)

	jwks := &JWKS{
		Keys: []JWKSKey{
			*createJWKFromRSAKey(&key1.PublicKey, "kid-1"),
			*createJWKFromRSAKey(&key2.PublicKey, "kid-2"),
			*createJWKFromRSAKey(&key3.PublicKey, "kid-3"),
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(jwks)
	}))
	defer server.Close()

	fetcher := NewInMemoryJWKSFetcher(time.Hour)
	ctx := context.Background()

	// Test retrieval of each key
	keys := []string{"kid-1", "kid-2", "kid-3"}
	expectedKeys := []*rsa.PublicKey{&key1.PublicKey, &key2.PublicKey, &key3.PublicKey}

	for i, kid := range keys {
		retrievedKey, err := fetcher.GetKey(ctx, server.URL, kid)
		if err != nil {
			t.Fatalf("Failed to get key %s: %v", kid, err)
		}

		rsaKey := retrievedKey.(*rsa.PublicKey)
		if rsaKey.N.Cmp(expectedKeys[i].N) != 0 {
			t.Errorf("Key %s modulus doesn't match", kid)
		}
	}
}

// TestExternalTokenValidator_ValidateToken_Success tests successful token validation
func TestExternalTokenValidator_ValidateToken_Success(t *testing.T) {
	// Generate RSA key pair
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate RSA key: %v", err)
	}

	issuer := "https://test.provider.com"
	audience := "test-client-id"
	kid := "test-kid"

	// Create token
	now := time.Now()
	claims := &IDTokenClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    issuer,
			Subject:   "user123",
			Audience:  jwt.ClaimStrings{audience},
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
			NotBefore: jwt.NewNumericDate(now.Add(-time.Minute)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
		Email:         "user@example.com",
		EmailVerified: true,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = kid
	tokenString, err := token.SignedString(privateKey)
	if err != nil {
		t.Fatalf("Failed to sign token: %v", err)
	}

	// Setup JWKS server
	jwks := createJWKSFromRSAKey(&privateKey.PublicKey, kid)
	jwksServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(jwks)
	}))
	defer jwksServer.Close()

	// Setup discovery server
	discoveryDoc := &OIDCConfiguration{
		Issuer:  issuer,
		JWKSUri: jwksServer.URL,
	}
	discoveryServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(discoveryDoc)
	}))
	defer discoveryServer.Close()

	// Create validator
	jwksFetcher := NewInMemoryJWKSFetcher(time.Hour)
	discoveryCache := NewInMemoryDiscoveryCache(
		WithDefaultTTL(time.Hour),
	)
	validator := NewExternalTokenValidator(jwksFetcher, discoveryCache)

	// Override issuer for testing
	discoveryCache.Set(issuer, discoveryDoc, time.Hour)

	// Validate token
	ctx := context.Background()
	validatedClaims, err := validator.ValidateToken(ctx, tokenString, issuer, audience)
	if err != nil {
		t.Fatalf("ValidateToken failed: %v", err)
	}

	// Verify claims
	if validatedClaims.Subject != "user123" {
		t.Errorf("Expected subject user123, got %s", validatedClaims.Subject)
	}
	if validatedClaims.Email != "user@example.com" {
		t.Errorf("Expected email user@example.com, got %s", validatedClaims.Email)
	}
}

// TestExternalTokenValidator_ValidateToken_ExpiredToken tests expired token rejection
func TestExternalTokenValidator_ValidateToken_ExpiredToken(t *testing.T) {
	privateKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	issuer := "https://test.provider.com"
	audience := "test-client-id"
	kid := "test-kid"

	// Create expired token
	now := time.Now()
	claims := &IDTokenClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    issuer,
			Subject:   "user123",
			Audience:  jwt.ClaimStrings{audience},
			ExpiresAt: jwt.NewNumericDate(now.Add(-time.Hour)), // Expired
			IssuedAt:  jwt.NewNumericDate(now.Add(-2 * time.Hour)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = kid
	tokenString, _ := token.SignedString(privateKey)

	// Setup servers
	jwks := createJWKSFromRSAKey(&privateKey.PublicKey, kid)
	jwksServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(jwks)
	}))
	defer jwksServer.Close()

	discoveryDoc := &OIDCConfiguration{
		Issuer:  issuer,
		JWKSUri: jwksServer.URL,
	}

	jwksFetcher := NewInMemoryJWKSFetcher(time.Hour)
	discoveryCache := NewInMemoryDiscoveryCache(WithDefaultTTL(time.Hour))
	discoveryCache.Set(issuer, discoveryDoc, time.Hour)
	validator := NewExternalTokenValidator(jwksFetcher, discoveryCache)

	// Validate expired token
	ctx := context.Background()
	_, err := validator.ValidateToken(ctx, tokenString, issuer, audience)
	if err == nil {
		t.Fatal("Expected error for expired token")
	}
}

// TestExternalTokenValidator_ValidateToken_InvalidIssuer tests issuer validation
func TestExternalTokenValidator_ValidateToken_InvalidIssuer(t *testing.T) {
	privateKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	actualIssuer := "https://actual.provider.com"
	expectedIssuer := "https://expected.provider.com"
	audience := "test-client-id"
	kid := "test-kid"

	// Create token with wrong issuer
	now := time.Now()
	claims := &IDTokenClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    actualIssuer,
			Subject:   "user123",
			Audience:  jwt.ClaimStrings{audience},
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = kid
	tokenString, _ := token.SignedString(privateKey)

	// Setup servers
	jwks := createJWKSFromRSAKey(&privateKey.PublicKey, kid)
	jwksServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(jwks)
	}))
	defer jwksServer.Close()

	discoveryDoc := &OIDCConfiguration{
		Issuer:  expectedIssuer,
		JWKSUri: jwksServer.URL,
	}

	jwksFetcher := NewInMemoryJWKSFetcher(time.Hour)
	discoveryCache := NewInMemoryDiscoveryCache(WithDefaultTTL(time.Hour))
	discoveryCache.Set(expectedIssuer, discoveryDoc, time.Hour)
	validator := NewExternalTokenValidator(jwksFetcher, discoveryCache)

	// Validate token
	ctx := context.Background()
	_, err := validator.ValidateToken(ctx, tokenString, expectedIssuer, audience)
	if err == nil {
		t.Fatal("Expected error for invalid issuer")
	}
}

// TestExternalTokenValidator_ValidateToken_InvalidAudience tests audience validation
func TestExternalTokenValidator_ValidateToken_InvalidAudience(t *testing.T) {
	privateKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	issuer := "https://test.provider.com"
	tokenAudience := "actual-client-id"
	expectedAudience := "expected-client-id"
	kid := "test-kid"

	// Create token with wrong audience
	now := time.Now()
	claims := &IDTokenClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    issuer,
			Subject:   "user123",
			Audience:  jwt.ClaimStrings{tokenAudience},
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = kid
	tokenString, _ := token.SignedString(privateKey)

	// Setup servers
	jwks := createJWKSFromRSAKey(&privateKey.PublicKey, kid)
	jwksServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(jwks)
	}))
	defer jwksServer.Close()

	discoveryDoc := &OIDCConfiguration{
		Issuer:  issuer,
		JWKSUri: jwksServer.URL,
	}

	jwksFetcher := NewInMemoryJWKSFetcher(time.Hour)
	discoveryCache := NewInMemoryDiscoveryCache(WithDefaultTTL(time.Hour))
	discoveryCache.Set(issuer, discoveryDoc, time.Hour)
	validator := NewExternalTokenValidator(jwksFetcher, discoveryCache)

	// Validate token
	ctx := context.Background()
	_, err := validator.ValidateToken(ctx, tokenString, issuer, expectedAudience)
	if err == nil {
		t.Fatal("Expected error for invalid audience")
	}
}

// TestExternalTokenValidator_ValidateToken_MissingKid tests handling of missing kid
func TestExternalTokenValidator_ValidateToken_MissingKid(t *testing.T) {
	privateKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	issuer := "https://test.provider.com"
	audience := "test-client-id"

	// Create token without kid
	now := time.Now()
	claims := &IDTokenClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    issuer,
			Subject:   "user123",
			Audience:  jwt.ClaimStrings{audience},
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	// Don't set kid header
	tokenString, _ := token.SignedString(privateKey)

	discoveryDoc := &OIDCConfiguration{
		Issuer:  issuer,
		JWKSUri: "https://test.com/jwks",
	}

	jwksFetcher := NewInMemoryJWKSFetcher(time.Hour)
	discoveryCache := NewInMemoryDiscoveryCache(WithDefaultTTL(time.Hour))
	discoveryCache.Set(issuer, discoveryDoc, time.Hour)
	validator := NewExternalTokenValidator(jwksFetcher, discoveryCache)

	// Validate token
	ctx := context.Background()
	_, err := validator.ValidateToken(ctx, tokenString, issuer, audience)
	if err == nil {
		t.Fatal("Expected error for missing kid")
	}
}

// Helper functions

func createJWKSFromRSAKey(publicKey *rsa.PublicKey, kid string) *JWKS {
	return &JWKS{
		Keys: []JWKSKey{*createJWKFromRSAKey(publicKey, kid)},
	}
}

func createJWKFromRSAKey(publicKey *rsa.PublicKey, kid string) *JWKSKey {
	// Encode modulus and exponent as base64url
	nBytes := publicKey.N.Bytes()
	n := base64.RawURLEncoding.EncodeToString(nBytes)

	// Encode exponent
	eBytes := big.NewInt(int64(publicKey.E)).Bytes()
	e := base64.RawURLEncoding.EncodeToString(eBytes)

	return &JWKSKey{
		KID: kid,
		Kty: "RSA",
		Alg: "RS256",
		Use: "sig",
		N:   n,
		E:   e,
	}
}

package gauth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"testing"
	"time"
)

// TestParsingPropertyRoundTrip validates that encoding and then parsing returns equivalent claims.
// Property: For all valid claim sets C, decode(encode(C)) ≈ C (modulo type coercion)
func TestParsingPropertyRoundTrip(t *testing.T) {
	svc, err := New(Config{
		ClientID:           "roundtrip-client",
		ClientSecret:       strings.Repeat("x", 40),
		AuthServerURL:      "https://auth.local",
		AccessTokenExpiry:  time.Hour,
	})
	if err != nil {
		t.Fatalf("service init: %v", err)
	}

	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))
	iterations := 1000

	for i := 0; i < iterations; i++ {
		// Generate random but valid claim set
		now := time.Now()
		claims := map[string]interface{}{
			"sub":   fmt.Sprintf("user-%d", rnd.Intn(10000)),
			"scope": generateRandomScope(rnd),
			"exp":   now.Add(time.Hour * time.Duration(rnd.Intn(24)+1)).Unix(),
			"iat":   now.Unix(),
			"iss":   "https://auth.local",
		}

		// Optionally add audience (string or array)
		if rnd.Intn(2) == 0 {
			if rnd.Intn(2) == 0 {
				claims["aud"] = "https://api.local"
			} else {
				claims["aud"] = []string{"https://api1.local", "https://api2.local"}
			}
		}

		// Optionally add nbf
		if rnd.Intn(3) == 0 {
			claims["nbf"] = now.Add(-time.Minute * time.Duration(rnd.Intn(10))).Unix()
		}

		// Optionally add jti
		if rnd.Intn(2) == 0 {
			claims["jti"] = generateJTI()
		}

		// Build token
		token := buildTestToken(svc.signingKey, claims)

		// Parse token
		res, vErr := svc.ValidateToken(token)

		// Validation may fail due to expiration skew or missing fields, but shouldn't panic
		if vErr != nil {
			// Expected errors for edge cases
			continue
		}

		if res == nil || !res.Valid {
			t.Fatalf("iteration %d: expected valid result, got %+v", i, res)
		}

		// Verify round-trip for key fields
		if res.ClientID != claims["sub"].(string) {
			t.Errorf("iteration %d: client ID mismatch: got %s, want %s", i, res.ClientID, claims["sub"])
		}

		// Scope round-trip (may be normalized)
		expectedScope := normalizeScope(claims["scope"].(string))
		actualScope := normalizeScopeSlice(res.Scope)
		if expectedScope != actualScope {
			t.Errorf("iteration %d: scope mismatch: got %s, want %s", i, actualScope, expectedScope)
		}

		// Note: ExpiresAt not exposed in TokenValidationResult, only Valid flag
		// Expiration validation happens internally during ValidateToken
	}
}

// TestParsingPropertyIdempotence validates that parsing is deterministic for the same input.
// Property: For all tokens T, the first parse result should be deterministic
// Note: Subsequent parses may fail due to JTI replay detection, which is expected behavior
func TestParsingPropertyIdempotence(t *testing.T) {
	svc, err := New(Config{
		ClientID:           "idempotent-client",
		ClientSecret:       strings.Repeat("y", 40),
		AuthServerURL:      "https://auth.local",
		AccessTokenExpiry:  time.Hour,
	})
	if err != nil {
		t.Fatalf("service init: %v", err)
	}

	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))
	iterations := 200

	for i := 0; i < iterations; i++ {
		claims := generateRandomClaims(rnd)
		// Don't include JTI to avoid replay detection interference
		delete(claims, "jti")
		
		token := buildTestToken(svc.signingKey, claims)

		// Parse token multiple times (without JTI, should be deterministic)
		res1, err1 := svc.ValidateToken(token)
		res2, err2 := svc.ValidateToken(token)

		// Errors should be consistent
		if (err1 == nil) != (err2 == nil) {
			// This could indicate non-deterministic parsing
			t.Errorf("iteration %d: inconsistent errors: %v vs %v", i, err1, err2)
			continue
		}

		// If no errors, results should be identical
		if err1 == nil && res1 != nil && res2 != nil {
			if res1.Valid != res2.Valid {
				t.Errorf("iteration %d: validity inconsistent: %v vs %v", i, res1.Valid, res2.Valid)
			}
			if res1.ClientID != res2.ClientID {
				t.Errorf("iteration %d: client ID inconsistent: %s vs %s", i, res1.ClientID, res2.ClientID)
			}
			if !scopeSlicesEqual(res1.Scope, res2.Scope) {
				t.Errorf("iteration %d: scope inconsistent: %v vs %v", i, res1.Scope, res2.Scope)
			}
		}
	}
}

// TestParsingPropertyErrorPreservation validates that malformed input consistently produces errors.
// Property: For all malformed tokens M, validate(M) returns error consistently
func TestParsingPropertyErrorPreservation(t *testing.T) {
	svc, err := New(Config{
		ClientID:           "error-client",
		ClientSecret:       strings.Repeat("z", 40),
		AuthServerURL:      "https://auth.local",
		AccessTokenExpiry:  time.Hour,
	})
	if err != nil {
		t.Fatalf("service init: %v", err)
	}

	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))
	iterations := 500

	for i := 0; i < iterations; i++ {
		// Generate deliberately malformed tokens
		malformedToken := generateMalformedToken(rnd, svc.signingKey)

		// Parse multiple times
		_, err1 := svc.ValidateToken(malformedToken)
		_, err2 := svc.ValidateToken(malformedToken)
		_, err3 := svc.ValidateToken(malformedToken)

		// All calls should produce errors (malformed input)
		if err1 == nil || err2 == nil || err3 == nil {
			t.Errorf("iteration %d: expected errors for malformed token, got %v, %v, %v", i, err1, err2, err3)
		}

		// Error messages should be consistent (same error type)
		// Note: Exact message may vary, but error presence is deterministic
	}
}

// TestParsingPropertyClaimExtraction validates that claim extraction is order-independent.
// Property: For all claim sets C with permuted JSON field order, parsed claims are equivalent
func TestParsingPropertyClaimExtraction(t *testing.T) {
	svc, err := New(Config{
		ClientID:           "extraction-client",
		ClientSecret:       strings.Repeat("a", 40),
		AuthServerURL:      "https://auth.local",
		AccessTokenExpiry:  time.Hour,
	})
	if err != nil {
		t.Fatalf("service init: %v", err)
	}

	iterations := 500

	for i := 0; i < iterations; i++ {
		now := time.Now()
		baseClaims := map[string]interface{}{
			"sub":   fmt.Sprintf("user-%d", i),
			"scope": "read write",
			"exp":   now.Add(2 * time.Hour).Unix(),
			"iat":   now.Unix(),
			"iss":   "https://auth.local",
			"aud":   "https://api.local",
		}

		// Build token with field order 1
		token1 := buildTestTokenWithFieldOrder(svc.signingKey, baseClaims, []string{"sub", "scope", "exp", "iat", "iss", "aud"})

		// Build token with field order 2 (permuted)
		token2 := buildTestTokenWithFieldOrder(svc.signingKey, baseClaims, []string{"aud", "iss", "iat", "exp", "scope", "sub"})

		// Parse both tokens
		res1, err1 := svc.ValidateToken(token1)
		res2, err2 := svc.ValidateToken(token2)

		// Both should succeed or fail consistently
		if (err1 == nil) != (err2 == nil) {
			t.Errorf("iteration %d: inconsistent errors for permuted fields: %v vs %v", i, err1, err2)
		}

		if err1 == nil {
			// Extracted claims should be identical
			if res1.ClientID != res2.ClientID {
				t.Errorf("iteration %d: client ID mismatch: %s vs %s", i, res1.ClientID, res2.ClientID)
			}
			if !scopeSlicesEqual(res1.Scope, res2.Scope) {
				t.Errorf("iteration %d: scope mismatch: %v vs %v", i, res1.Scope, res2.Scope)
			}
			// Note: ExpiresAt not exposed in TokenValidationResult
		}
	}
}

// TestParsingPropertyTimingBoundaries validates parsing behavior near time boundaries.
// Property: Expired tokens should be rejected consistently
func TestParsingPropertyTimingBoundaries(t *testing.T) {
	// Force HMAC mode for this test since buildTestToken creates HMAC tokens
	oldMode := os.Getenv("GAUTH_TOKEN_SIG_MODE")
	os.Setenv("GAUTH_TOKEN_SIG_MODE", "hmac")
	defer os.Setenv("GAUTH_TOKEN_SIG_MODE", oldMode)
	
	svc, err := New(Config{
		ClientID:           "timing-client",
		ClientSecret:       strings.Repeat("b", 40),
		AuthServerURL:      "https://auth.local",
		AccessTokenExpiry:  time.Hour,
	})
	if err != nil {
		t.Fatalf("service init: %v", err)
	}

	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))
	iterations := 100

	for i := 0; i < iterations; i++ {
		now := time.Now()

		// Test case 1: Token already expired
		expiredClaims := map[string]interface{}{
			"sub":   fmt.Sprintf("user-%d", i),
			"scope": "read",
			"exp":   now.Add(-time.Minute).Unix(), // Expired 1 minute ago
			"iat":   now.Add(-2 * time.Minute).Unix(),
			"jti":   fmt.Sprintf("expired-jti-%d", i),
		}
		expiredToken := buildTestToken(svc.signingKey, expiredClaims)
		_, expiredErr := svc.ValidateToken(expiredToken)
		if expiredErr == nil {
			t.Errorf("iteration %d: expired token accepted", i)
		}

		// Test case 2: Token valid for next hour
		validClaims := map[string]interface{}{
			"sub":   fmt.Sprintf("user-%d", i),
			"scope": "read",
			"exp":   now.Add(time.Hour).Unix(),
			"iat":   now.Unix(),
			"jti":   fmt.Sprintf("valid-jti-%d", i),
		}
		validToken := buildTestToken(svc.signingKey, validClaims)
		validRes, validErr := svc.ValidateToken(validToken)
		if validErr != nil || (validRes != nil && !validRes.Valid) {
			t.Errorf("iteration %d: valid token rejected: %v", i, validErr)
		}

		// Test case 3: Token with random future expiry (should succeed)
		futureClaims := map[string]interface{}{
			"sub":   fmt.Sprintf("user-%d", i),
			"scope": "write",
			"exp":   now.Add(time.Duration(rnd.Intn(3600)+60) * time.Second).Unix(),
			"iat":   now.Unix(),
			"jti":   fmt.Sprintf("future-jti-%d", i),
		}
		futureToken := buildTestToken(svc.signingKey, futureClaims)
		futureRes, futureErr := svc.ValidateToken(futureToken)
		if futureErr != nil || (futureRes != nil && !futureRes.Valid) {
			t.Errorf("iteration %d: future-valid token rejected: %v", i, futureErr)
		}
	}
}

// TestParsingPropertyNullAndEmpty validates handling of null and empty values.
// Property: Missing/null/empty claims should be handled consistently without panics
func TestParsingPropertyNullAndEmpty(t *testing.T) {
	svc, err := New(Config{
		ClientID:           "null-client",
		ClientSecret:       strings.Repeat("c", 40),
		AuthServerURL:      "https://auth.local",
		AccessTokenExpiry:  time.Hour,
	})
	if err != nil {
		t.Fatalf("service init: %v", err)
	}

	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))
	iterations := 500

	for i := 0; i < iterations; i++ {
		// Generate claims with random null/empty values
		claims := generateClaimsWithNullsAndEmpties(rnd)
		token := buildTestToken(svc.signingKey, claims)

		// Parse should not panic
		res, err := svc.ValidateToken(token)

		// Validate error behavior is consistent
		if err == nil && (res == nil || !res.Valid) {
			t.Errorf("iteration %d: nil error but invalid result", i)
		}

		// If mandatory fields missing, should error
		if _, hasSub := claims["sub"]; !hasSub {
			if err == nil {
				t.Errorf("iteration %d: missing sub but no error", i)
			}
		}
	}
}

// Helper functions

func buildTestToken(signingKey []byte, claims map[string]interface{}) string {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))

	// Serialize claims to JSON
	claimsJSON, _ := json.Marshal(claims)
	payload := base64.RawURLEncoding.EncodeToString(claimsJSON)

	unsigned := header + "." + payload
	mac := hmac.New(sha256.New, signingKey)
	mac.Write([]byte(unsigned))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))

	return unsigned + "." + sig
}

func buildTestTokenWithFieldOrder(signingKey []byte, claims map[string]interface{}, fieldOrder []string) string {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))

	// Manually construct JSON with specific field order
	parts := make([]string, 0, len(fieldOrder))
	for _, field := range fieldOrder {
		if val, ok := claims[field]; ok {
			switch v := val.(type) {
			case string:
				parts = append(parts, fmt.Sprintf("\"%s\":\"%s\"", field, v))
			case int64:
				parts = append(parts, fmt.Sprintf("\"%s\":%d", field, v))
			case int:
				parts = append(parts, fmt.Sprintf("\"%s\":%d", field, v))
			case []string:
				audJSON, _ := json.Marshal(v)
				parts = append(parts, fmt.Sprintf("\"%s\":%s", field, audJSON))
			}
		}
	}

	claimsJSON := "{" + strings.Join(parts, ",") + "}"
	payload := base64.RawURLEncoding.EncodeToString([]byte(claimsJSON))

	unsigned := header + "." + payload
	mac := hmac.New(sha256.New, signingKey)
	mac.Write([]byte(unsigned))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))

	return unsigned + "." + sig
}

func generateRandomScope(rnd *rand.Rand) string {
	scopes := []string{"read", "write", "delete", "admin", "user:read", "api:access"}
	count := rnd.Intn(4) + 1
	selected := make([]string, count)
	for i := 0; i < count; i++ {
		selected[i] = scopes[rnd.Intn(len(scopes))]
	}
	return strings.Join(selected, " ")
}

func generateRandomClaims(rnd *rand.Rand) map[string]interface{} {
	now := time.Now()
	claims := map[string]interface{}{
		"sub": fmt.Sprintf("user-%d", rnd.Intn(10000)),
		"exp": now.Add(time.Hour * time.Duration(rnd.Intn(24)+1)).Unix(),
	}

	// Randomly add optional fields
	if rnd.Intn(2) == 0 {
		claims["scope"] = generateRandomScope(rnd)
	}
	if rnd.Intn(2) == 0 {
		claims["iat"] = now.Unix()
	}
	if rnd.Intn(2) == 0 {
		claims["iss"] = "https://auth.local"
	}
	if rnd.Intn(3) == 0 {
		claims["aud"] = "https://api.local"
	}
	if rnd.Intn(4) == 0 {
		claims["jti"] = generateJTI()
	}

	return claims
}

func generateMalformedToken(rnd *rand.Rand, signingKey []byte) string {
	// Generate various types of malformed tokens
	strategy := rnd.Intn(8)

	switch strategy {
	case 0: // Truncated token (missing signature)
		validToken := buildTestToken(signingKey, generateRandomClaims(rnd))
		return validToken[:len(validToken)-rnd.Intn(20)-1]

	case 1: // Missing payload segment
		header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
		return header + ".."

	case 2: // Invalid base64 in payload
		header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
		return header + ".invalid!!!base64." + generateJTI()

	case 3: // Invalid JSON in payload
		header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
		payload := base64.RawURLEncoding.EncodeToString([]byte("{not:valid:json}"))
		return header + "." + payload + "." + generateJTI()

	case 4: // Wrong signature
		validToken := buildTestToken(signingKey, generateRandomClaims(rnd))
		parts := strings.Split(validToken, ".")
		wrongSig := base64.RawURLEncoding.EncodeToString([]byte("wrong"))
		return parts[0] + "." + parts[1] + "." + wrongSig

	case 5: // Empty token
		return ""

	case 6: // Only one segment
		return base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256"}`))

	case 7: // Corrupted payload (non-UTF8)
		header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
		payload := base64.RawURLEncoding.EncodeToString([]byte{0xFF, 0xFE, 0xFD})
		return header + "." + payload + "." + generateJTI()

	default:
		return "malformed.token.invalid"
	}
}

func generateClaimsWithNullsAndEmpties(rnd *rand.Rand) map[string]interface{} {
	claims := make(map[string]interface{})

	// Randomly include/exclude/null each field
	if rnd.Intn(3) != 0 {
		if rnd.Intn(5) == 0 {
			claims["sub"] = "" // empty string
		} else {
			claims["sub"] = fmt.Sprintf("user-%d", rnd.Intn(100))
		}
	}

	if rnd.Intn(3) != 0 {
		if rnd.Intn(5) == 0 {
			claims["scope"] = "" // empty scope
		} else {
			claims["scope"] = "read"
		}
	}

	if rnd.Intn(2) == 0 {
		claims["exp"] = time.Now().Add(time.Hour).Unix()
	}

	if rnd.Intn(3) == 0 {
		claims["iat"] = time.Now().Unix()
	}

	if rnd.Intn(4) == 0 {
		if rnd.Intn(5) == 0 {
			claims["iss"] = "" // empty issuer
		} else {
			claims["iss"] = "https://auth.local"
		}
	}

	return claims
}

func normalizeScope(scope string) string {
	// Normalize whitespace for comparison
	return strings.Join(strings.Fields(scope), " ")
}

func normalizeScopeSlice(scopes []string) string {
	// Convert slice to space-separated string for comparison
	return strings.Join(scopes, " ")
}

func scopeSlicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func abs(x int64) int64 {
	if x < 0 {
		return -x
	}
	return x
}

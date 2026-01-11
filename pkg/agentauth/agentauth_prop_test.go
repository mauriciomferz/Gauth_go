package agentauth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"math/rand"
	"strings"
	"testing"
	"time"
)

// buildManualToken constructs a legacy HMAC token using the same logic as Service.RequestToken
// but allows overriding / omitting certain fields to exercise edge cases.
func buildManualToken(signingKey []byte, claims map[string]any) string {
	// Header always constant for legacy path
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
	// Build payload in a deterministic order to keep signing stable.
	parts := []string{}
	for k, v := range claims {
		switch val := v.(type) {
		case string:
			parts = append(parts, fmt.Sprintf("\"%s\":\"%s\"", k, val))
		case int64:
			parts = append(parts, fmt.Sprintf("\"%s\":%d", k, val))
		case int:
			parts = append(parts, fmt.Sprintf("\"%s\":%d", k, val))
		case float64:
			parts = append(parts, fmt.Sprintf("\"%s\":%d", k, int64(val)))
		}
	}
	// Stable ordering by key for reproducibility
	// (Simple bubble; size small)
	changed := true
	for changed {
		changed = false
		for i := 0; i < len(parts)-1; i++ {
			// compare by key lexicographically: first char after opening quote
			ki := parts[i][2 : strings.Index(parts[i][2:], "\"")+2]
			kj := parts[i+1][2 : strings.Index(parts[i+1][2:], "\"")+2]
			if kj < ki {
				parts[i], parts[i+1] = parts[i+1], parts[i]
				changed = true
			}
		}
	}
	payloadJSON := "{" + strings.Join(parts, ",") + "}"
	payload := base64.RawURLEncoding.EncodeToString([]byte(payloadJSON))
	unsigned := header + "." + payload
	mac := hmac.New(sha256.New, signingKey)
	mac.Write([]byte(unsigned))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return unsigned + "." + sig
}

// TestJSONParseProperty runs a lightweight property-style loop constructing random tokens
// and ensures ValidateToken never panics and returns logically consistent validity
// relative to exp (if present and in the future) while handling missing/extra fields.
func TestJSONParseProperty(t *testing.T) {
	svc, err := New(Config{
		ClientID: "prop-client", ClientSecret: strings.Repeat("x", 40),
		AuthServerURL: "https://auth.local", AccessTokenExpiry: time.Hour,
	})
	if err != nil {
		t.Fatalf("service init: %v", err)
	}
	//nolint:gosec // G404: weak random acceptable for property-based testing
	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))
	iterations := 300
	for i := 0; i < iterations; i++ {
		// Random claim set
		claims := map[string]any{}
		// Mandatory sub sometimes omitted
		if rnd.Intn(5) != 0 {
			claims["sub"] = "prop-client"
		}
		// scope sometimes empty or large
		switch {
		case rnd.Intn(4) == 0:
			claims["scope"] = ""
		case rnd.Intn(7) == 0:
			claims["scope"] = strings.Repeat("a", rnd.Intn(50)+1)
		default:
			claims["scope"] = "read write"
		}
		now := time.Now()
		// exp variations: valid future, past, missing, huge, near-future
		switch rnd.Intn(6) {
		case 0: // valid future
			claims["exp"] = now.Add(time.Minute * time.Duration(rnd.Intn(120)+2)).Unix()
		case 1: // past
			claims["exp"] = now.Add(-time.Minute * time.Duration(rnd.Intn(30)+1)).Unix()
		case 2: // near future (boundary)
			claims["exp"] = now.Add(5 * time.Second).Unix()
		case 3: // far future
			claims["exp"] = now.Add(24 * time.Hour).Unix()
		case 4: // huge (overflow risk not realistic but test large value)
			claims["exp"] = now.Add(365 * 24 * time.Hour).Unix()
		case 5: // missing: do not set exp
		}
		// iat sometimes omitted
		if rnd.Intn(3) != 0 {
			claims["iat"] = now.Unix()
		}

		token := buildManualToken(svc.signingKey, claims)
		// Occasionally corrupt token structure (remove signature or payload)
		switch rnd.Intn(8) {
		case 0: // truncate signature
			token = token[:len(token)-rnd.Intn(10)-1]
		case 1: // drop last segment entirely
			token = token[:strings.LastIndex(token, ".")]
		case 2: // corrupt base64 by appending '=' causing decode mismatch maybe
			token += "="
		}
		// Execute validation
		res, vErr := svc.ValidateToken(token)
		// Invariant: no panic (implicit) and if error nil then Valid true
		if vErr == nil && (res == nil || !res.Valid) {
			t.Fatalf("iteration %d: expected valid result when error nil", i)
		}
		// If exp present and in the past, service should reject (error != nil)
		if expV, ok := claims["exp"].(int64); ok && vErr == nil {
			if time.Now().Unix() > expV {
				t.Fatalf("iteration %d: token expired but accepted exp=%d", i, expV)
			}
		}
	}
}

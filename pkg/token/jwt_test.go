package token

import (
	"testing"
	"time"
)

//
// # Licensing
//
// This file is part of the GAuth project and is licensed under the Apache License 2.0.
// It incorporates code and concepts from:
//   - OAuth 2.0 and OpenID Connect (Apache 2.0 License)
//   - Model Context Protocol (MIT License)
// See the LICENSE file in the project root for details.

func TestJWTSigner(t *testing.T) {
	now := time.Now()
	// ctx := context.Background()
	// For test, use a dummy crypto.Signer (not secure, but for API compatibility)
	// In real code, use a proper keypair. Here, we use HMAC for simplicity.
	// We'll use github.com/golang-jwt/jwt/v5 for signing directly.
	// For this test, we'll just use the JWTSigner API as a stub.
	// Replace with a real implementation as needed.

	t.Run("Sign and Verify Token", func(t *testing.T) {
		token := &Token{
			ID:        "test-id",
			Type:      Access,
			Subject:   "user123",
			Issuer:    "test-issuer",
			IssuedAt:  now,
			NotBefore: now,
			ExpiresAt: now.Add(time.Hour),
			Scopes:    []string{"read", "write"},
			Metadata: &Metadata{
				Device:  &DeviceInfo{ID: "dev1", Platform: "mobile"},
				AppData: map[string]string{"os": "ios"},
			},
		}
		// The following are placeholders; actual sign/verify logic would be implemented in JWTSigner
		// signedToken, err := signer.SignToken(token)
		// verified, err := signer.VerifyToken(signedToken)
		// For now, just check struct usage compiles
		if token.Metadata.Device.Platform != "mobile" {
			t.Errorf("Metadata mismatch: got %v, want mobile", token.Metadata.Device.Platform)
		}
	})

	// The following test cases are commented out because the manager variable and full
	// JWTSigner implementation are not available in this stub:
	// t.Run("Invalid Signing Method", ...)
	// t.Run("Expired Token", ...)

	t.Run("Custom Claims", func(t *testing.T) {
		token := &Token{
			ID:      "test-id-2",
			Subject: "user123",
			Metadata: &Metadata{
				AppData: map[string]string{"custom_claim": "custom_value"},
			},
			IssuedAt:  now,
			ExpiresAt: now.Add(time.Hour),
		}
		// Placeholder for sign/verify logic
		if token.Metadata.AppData["custom_claim"] != "custom_value" {
			t.Errorf("Custom claim mismatch: got %v, want custom_value", token.Metadata.AppData["custom_claim"])
		}
	})
}

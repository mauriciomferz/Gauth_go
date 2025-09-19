// GAuth Protocol Compliance: This test file validates the GAuth protocol (GiFo-RfC 0111).
//
// Protocol Usage Declaration:
//   - GAuth protocol: TESTED throughout this file (see [GAuth] comments below)
//   - OAuth 2.0:      NOT USED anywhere in this file
//   - PKCE:           NOT USED anywhere in this file
//   - OpenID:         NOT USED anywhere in this file
//
// [GAuth] = GAuth protocol logic (GiFo-RfC 0111)
// [Other] = Placeholder for OAuth2, OpenID, PKCE, or other protocols (none present in this file)
//
// [GAuth] Package gauth_test validates GAuth protocol compliance.
package gauth_test

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"io"
	"testing"
	"time"

	"github.com/mauriciomferz/Gauth_go/pkg/audit"
	"github.com/mauriciomferz/Gauth_go/pkg/gauth"
	"github.com/mauriciomferz/Gauth_go/pkg/token"
)

func newMockSigner() crypto.Signer {
	priv, _ := rsa.GenerateKey(rand.Reader, 2048)
	return &mockSigner{priv: priv}
}

// mockSigner is a minimal crypto.Signer for testing
type mockSigner struct {
	priv *rsa.PrivateKey
}

func (m *mockSigner) Public() crypto.PublicKey {
	return m.priv.Public()
}

func (m *mockSigner) Sign(rand io.Reader, digest []byte, opts crypto.SignerOpts) ([]byte, error) {
	return rsa.SignPKCS1v15(rand, m.priv, opts.HashFunc(), digest)
}

func TestGAuth(t *testing.T) {
	// Test configuration validation
	t.Run("Config Validation", func(t *testing.T) {
		validConfig := &gauth.Config{
			AuthServerURL:     "https://auth.example.com",
			ClientID:          "test-client",
			ClientSecret:      "test-secret",
			Scopes:            []string{"read", "write"},
			AccessTokenExpiry: time.Hour,
			TokenConfig:       &token.Config{SigningMethod: token.RS256, SigningKey: newMockSigner()},
		}

			   auth, err := gauth.New(validConfig, audit.NewLogger(100))
		if err != nil {
			t.Errorf("Failed to create GAuth with valid config: %v", err)
		}
		if auth == nil {
			t.Error("Expected non-nil GAuth instance")
		}

		invalidConfig := &gauth.Config{TokenConfig: &token.Config{SigningMethod: token.RS256, SigningKey: newMockSigner()}}
			   if _, err := gauth.New(invalidConfig, audit.NewLogger(100)); err == nil {

			t.Error("Expected error with invalid config")
		}
	})

	// Test authorization flow
	t.Run("Authorization Flow", func(t *testing.T) {
			   auth, _ := gauth.New(&gauth.Config{
		       AuthServerURL:     "https://auth.example.com",
		       ClientID:          "test-client",
		       ClientSecret:      "test-secret",
		       Scopes:            []string{"read", "write"},
		       AccessTokenExpiry: time.Hour,
		       TokenConfig:       &token.Config{SigningMethod: token.RS256, SigningKey: newMockSigner()},
	       }, audit.NewLogger(100))

		req := gauth.AuthorizationRequest{
			ClientID: "test-client",
			Scopes:   []string{"read"},
		}

		grant, err := auth.InitiateAuthorization(req)
		if err != nil {
			t.Errorf("Authorization request failed: %v", err)
		}
		if grant == nil {
			t.Error("Expected non-nil authorization grant")
		}
		if grant.ClientID != req.ClientID {
			t.Errorf("Expected client ID %s, got %s", req.ClientID, grant.ClientID)
		}
	})

	// Test token issuance and validation
	t.Run("Token Operations", func(t *testing.T) {
			   auth, _ := gauth.New(&gauth.Config{
		       AuthServerURL:     "https://auth.example.com",
		       ClientID:          "test-client",
		       ClientSecret:      "test-secret",
		       Scopes:            []string{"read", "write"},
		       AccessTokenExpiry: time.Hour,
		       TokenConfig:       &token.Config{SigningMethod: token.RS256, SigningKey: newMockSigner()},
	       }, audit.NewLogger(100))

		// Request a token
		tokenReq := gauth.TokenRequest{
			GrantID: "test-grant",
			Scope:   []string{"read"},
		}

		tokenResp, err := auth.RequestToken(tokenReq)
		if err != nil {
			t.Errorf("Token request failed: %v", err)
		}
		if tokenResp.Token == "" {
			t.Error("Expected non-empty token")
		}

		// Validate the token
		tokenData, err := auth.ValidateToken(tokenResp.Token)
		if err != nil {
			t.Errorf("Token validation failed: %v", err)
		}
		if !tokenData.Valid {
			t.Error("Expected token to be valid")
		}

		// Test invalid token
		_, err = auth.ValidateToken("invalid-token")
		if err == nil {
			t.Error("Expected error with invalid token")
		}
	})

	// Test token expiration
	t.Run("Token Expiration", func(t *testing.T) {
			   auth, _ := gauth.New(&gauth.Config{
		       AuthServerURL:     "https://auth.example.com",
		       ClientID:          "test-client",
		       ClientSecret:      "test-secret",
		       AccessTokenExpiry: 100 * time.Millisecond,
		       TokenConfig:       &token.Config{SigningMethod: token.RS256, SigningKey: newMockSigner()},
	       }, audit.NewLogger(100))

		tokenReq := gauth.TokenRequest{
			GrantID: "test-grant",
			Scope:   []string{"read"},
		}

		tokenResp, _ := auth.RequestToken(tokenReq)

		// Wait for token to expire
		time.Sleep(150 * time.Millisecond)

		_, err := auth.ValidateToken(tokenResp.Token)
		if err == nil {
			t.Error("Expected error with expired token")
		}
	})
}

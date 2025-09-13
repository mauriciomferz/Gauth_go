package gauth_test

import (
	"testing"
	"time"

	"github.com/Gimel-Foundation/gauth/pkg/gauth"
)

func TestGAuth(t *testing.T) {
	// Test configuration validation
	t.Run("Config Validation", func(t *testing.T) {
		validConfig := gauth.Config{
			AuthServerURL:     "https://auth.example.com",
			ClientID:          "test-client",
			ClientSecret:      "test-secret",
			Scopes:            []string{"read", "write"},
			AccessTokenExpiry: time.Hour,
		}

		auth, err := gauth.New(validConfig)
		if err != nil {
			t.Errorf("Failed to create GAuth with valid config: %v", err)
		}
		if auth == nil {
			t.Error("Expected non-nil GAuth instance")
		}

		invalidConfig := gauth.Config{}
		if _, err := gauth.New(invalidConfig); err == nil {
			t.Error("Expected error with invalid config")
		}
	})

	// Test authorization flow
	t.Run("Authorization Flow", func(t *testing.T) {
		auth, _ := gauth.New(gauth.Config{
			AuthServerURL:     "https://auth.example.com",
			ClientID:          "test-client",
			ClientSecret:      "test-secret",
			Scopes:            []string{"read", "write"},
			AccessTokenExpiry: time.Hour,
		})

		req := gauth.AuthorizationRequest{
			ClientID:        "test-client",
			ClientOwnerID:   "owner-1",
			ResourceOwnerID: "resource-1",
			Scopes:          []string{"read"},
			Timestamp:       time.Now().UnixNano() / 1e6,
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
		auth, _ := gauth.New(gauth.Config{
			AuthServerURL:     "https://auth.example.com",
			ClientID:          "test-client",
			ClientSecret:      "test-secret",
			Scopes:            []string{"read", "write"},
			AccessTokenExpiry: time.Hour,
		})

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
		auth, _ := gauth.New(gauth.Config{
			AuthServerURL:     "https://auth.example.com",
			ClientID:          "test-client",
			ClientSecret:      "test-secret",
			AccessTokenExpiry: 100 * time.Millisecond,
		})

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

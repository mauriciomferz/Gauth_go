package gauth

import (
	"testing"
	"time"
)

func TestGAuth(t *testing.T) {
	// Test configuration validation
	t.Run("Config Validation", func(t *testing.T) {
		validConfig := Config{
			AuthServerURL:     "https://auth.example.com",
			ClientID:          "test-client",
			ClientSecret:      "test-secret",
			Scopes:            []string{"read", "write"},
			AccessTokenExpiry: time.Hour,
		}

		auth, err := New(validConfig)
		if err != nil {
			t.Errorf("Failed to create GAuth with valid config: %v", err)
		}
		if auth == nil {
			t.Error("Expected non-nil GAuth instance")
		}

		invalidConfig := Config{}
		if _, err := New(invalidConfig); err == nil {
			t.Error("Expected error with invalid config")
		}
	})

	// Test authorization flow
	t.Run("Authorization Flow", func(t *testing.T) {
		auth, _ := New(Config{
			AuthServerURL:     "https://auth.example.com",
			ClientID:          "test-client",
			ClientSecret:      "test-secret",
			Scopes:            []string{"read", "write"},
			AccessTokenExpiry: time.Hour,
		})

		req := AuthorizationRequest{
			ClientID: "test-client",
			Scopes:   []string{"read"},
		}

		grant, err := auth.InitiateAuthorization(req)
		if err != nil {
			t.Errorf("Authorization request failed: %v", err)
		}
		if grant == nil {
			t.Error("Expected non-nil authorization grant")
		} else {
			if grant.ClientID != req.ClientID {
				t.Errorf("Expected client ID %s, got %s", req.ClientID, grant.ClientID)
			}
		}
	})

	// Test token issuance and validation
	t.Run("Token Operations", func(t *testing.T) {
		auth, _ := New(Config{
			AuthServerURL:     "https://auth.example.com",
			ClientID:          "test-client",
			ClientSecret:      "test-secret",
			Scopes:            []string{"read", "write"},
			AccessTokenExpiry: time.Hour,
		})

		// Request a token
		tokenReq := TokenRequest{
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
		validResp, err := auth.ValidateToken(tokenResp.Token)
		if err != nil {
			t.Errorf("Token validation failed: %v", err)
		}
		if validResp == nil {
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
		auth, _ := New(Config{
			AuthServerURL:     "https://auth.example.com",
			ClientID:          "test-client",
			ClientSecret:      "test-secret",
			AccessTokenExpiry: 100 * time.Millisecond,
		})

		tokenReq := TokenRequest{
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

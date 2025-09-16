package gauth

import (
	"context"
	"testing"
	"time"

	"github.com/Gimel-Foundation/gauth/pkg/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name        string
		config      Config
		expectError bool
	}{
		{
			name: "Valid configuration",
			config: Config{
				AuthServerURL:     "http://localhost:8080",
				ClientID:          "test-client",
				ClientSecret:      "test-secret",
				Scopes:            []string{"read", "write"},
				AccessTokenExpiry: time.Hour,
				   RateLimit: common.RateLimitConfig{
					RequestsPerSecond: 100,
					BurstSize:         10,
					WindowSize:        60,
				},
			},
			expectError: false,
		},
		{
			name: "Missing auth server URL",
			config: Config{
				ClientID:          "test-client",
				ClientSecret:      "test-secret",
				AccessTokenExpiry: time.Hour,
			},
			expectError: true,
		},
		{
			name: "Missing client ID",
			config: Config{
				AuthServerURL:     "http://localhost:8080",
				ClientSecret:      "test-secret",
				AccessTokenExpiry: time.Hour,
			},
			expectError: true,
		},
		{
			name: "Invalid token expiry",
			config: Config{
				AuthServerURL:     "http://localhost:8080",
				ClientID:          "test-client",
				ClientSecret:      "test-secret",
				AccessTokenExpiry: -time.Hour,
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, err := New(tt.config)
			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, svc)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, svc)
				assert.NoError(t, svc.Close())
			}
		})
	}
}

func TestService_Authorize(t *testing.T) {
       svc := setupTestService(t)
       t.Cleanup(func() {
	       err := svc.Close()
	       if err != nil {
		       t.Errorf("error closing service: %v", err)
	       }
       })

	ctx := context.Background()

	tests := []struct {
		name        string
		request     *AuthorizationRequest
		expectError bool
	}{
		{
			name: "Valid request",
			request: &AuthorizationRequest{
				ClientID: "test-client",
				Scopes:   []string{"read", "write"},
			},
			expectError: false,
		},
		{
			name: "Missing client ID",
			request: &AuthorizationRequest{
				Scopes: []string{"read"},
			},
			expectError: true,
		},
		{
			name: "Empty scopes",
			request: &AuthorizationRequest{
				ClientID: "test-client",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			grant, err := svc.Authorize(ctx, tt.request)
			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, grant)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, grant)
				assert.Equal(t, tt.request.ClientID, grant.ClientID)
				assert.ElementsMatch(t, tt.request.Scopes, grant.Scope)
				assert.True(t, grant.ValidUntil.After(time.Now()))
			}
		})
	}
}

func TestService_RequestToken(t *testing.T) {
       svc := setupTestService(t)
       t.Cleanup(func() {
	       err := svc.Close()
	       if err != nil {
		       t.Errorf("error closing service: %v", err)
	       }
       })

	ctx := context.Background()

	// First, get a valid grant
	grant, err := svc.Authorize(ctx, &AuthorizationRequest{
		ClientID: "test-client",
		Scopes:   []string{"read", "write"},
	})
	require.NoError(t, err)
	require.NotNil(t, grant)

	tests := []struct {
		name        string
		request     *TokenRequest
		expectError bool
	}{
		{
			name: "Valid request",
			request: &TokenRequest{
				GrantID: grant.GrantID,
				Scope:   []string{"read", "write"},
			},
			expectError: false,
		},
		{
			name: "Invalid grant ID",
			request: &TokenRequest{
				GrantID: "invalid-grant",
				Scope:   []string{"read"},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := svc.RequestToken(ctx, tt.request)
			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, resp)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, resp)
				assert.NotEmpty(t, resp.Token)
				assert.True(t, resp.ValidUntil.After(time.Now()))
				assert.ElementsMatch(t, tt.request.Scope, resp.Scope)
			}
		})
	}
}

func TestService_RevokeToken(t *testing.T) {
       svc := setupTestService(t)
       t.Cleanup(func() {
	       err := svc.Close()
	       if err != nil {
		       t.Errorf("error closing service: %v", err)
	       }
       })

	ctx := context.Background()

	// Get a valid token
	grant, err := svc.Authorize(ctx, &AuthorizationRequest{
		ClientID: "test-client",
		Scopes:   []string{"read"},
	})
	require.NoError(t, err)

	resp, err := svc.RequestToken(ctx, &TokenRequest{
		GrantID: grant.GrantID,
		Scope:   []string{"read"},
	})
	require.NoError(t, err)

	// Test token revocation
	err = svc.RevokeToken(ctx, resp.Token)
	assert.NoError(t, err)

	// Verify token is revoked by trying to use it
	_, err = svc.RequestToken(ctx, &TokenRequest{
		GrantID: grant.GrantID,
		Scope:   []string{"read"},
	})
	assert.Error(t, err)
}

func TestService_RateLimiting(t *testing.T) {
	config := Config{
		AuthServerURL:     "http://localhost:8080",
		ClientID:          "test-client",
		ClientSecret:      "test-secret",
		AccessTokenExpiry: time.Hour,
		RateLimit: common.RateLimitConfig{
			RequestsPerSecond: 2,
			BurstSize:         1,
			WindowSize:        1,
		},
	}

	svc, err := New(config)
	require.NoError(t, err)
	defer svc.Close()

	ctx := context.Background()
	req := &AuthorizationRequest{
		ClientID: "test-client",
		Scopes:   []string{"read"},
	}

	// First request should succeed
	_, err = svc.Authorize(ctx, req)
	assert.NoError(t, err)

	// Second request should fail due to rate limiting
	_, err = svc.Authorize(ctx, req)
	assert.Error(t, err)

	// Wait for rate limit window to reset
	time.Sleep(time.Second)

	// Request should succeed again
	_, err = svc.Authorize(ctx, req)
	assert.NoError(t, err)
}

func setupTestService(t *testing.T) *Service {
	config := Config{
		AuthServerURL:     "http://localhost:8080",
		ClientID:          "test-client",
		ClientSecret:      "test-secret",
		AccessTokenExpiry: time.Hour,
		RateLimit: common.RateLimitConfig{
			RequestsPerSecond: 100,
			BurstSize:         10,
			WindowSize:        60,
		},
	}

	svc, err := NewService(config)
	require.NoError(t, err)
	return svc
}

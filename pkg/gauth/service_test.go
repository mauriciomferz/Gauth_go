package gauth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"testing"
	"time"

	"github.com/mauriciomferz/Gauth_go/pkg/common"
	"github.com/mauriciomferz/Gauth_go/pkg/token"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func generateTestSigningKey(t *testing.T) *rsa.PrivateKey {
       key, err := rsa.GenerateKey(rand.Reader, 2048)
       require.NoError(t, err)
       return key
}

func TestNew(t *testing.T) {
       signingKey := generateTestSigningKey(t)
       tests := []struct {
	       name        string
	       config      *Config
	       expectError bool
       }{
	       {
		       name: "Valid configuration",
		       config: &Config{
			       AuthServerURL:     "http://localhost:8080",
			       ClientID:          "test-client",
			       ClientSecret:      "test-secret",
			       Scopes:            []string{"read", "write"},
			       AccessTokenExpiry: time.Hour,
			       RateLimit:         common.RateLimitConfig{},
			       TokenConfig: &token.Config{
				       SigningKey:        signingKey,
				       SigningMethod:     "RS256",
				       ValidityPeriod:    time.Hour,
				       RefreshPeriod:     time.Hour,
				       CleanupInterval:   time.Hour,
				       MaxTokens:         1000,
			       },
		       },
		       expectError: false,
	       },
	       {
		       name: "Missing signing key",
		       config: &Config{
			       AuthServerURL:     "http://localhost:8080",
			       ClientID:          "test-client",
			       ClientSecret:      "test-secret",
			       Scopes:            []string{"read", "write"},
			       AccessTokenExpiry: time.Hour,
			       RateLimit:         common.RateLimitConfig{},
			       TokenConfig: &token.Config{
				       SigningMethod:     "RS256",
				       ValidityPeriod:    time.Hour,
				       RefreshPeriod:     time.Hour,
				       CleanupInterval:   time.Hour,
				       MaxTokens:         1000,
			       },
		       },
		       expectError: true,
	       },
       }

       for _, tt := range tests {
	       t.Run(tt.name, func(t *testing.T) {
		       svc, err := NewService(tt.config)
		       if tt.expectError {
			       assert.Error(t, err)
		       } else {
			       assert.NoError(t, err)
			       assert.NotNil(t, svc)
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
       t.Logf("DEBUG: SigningKey in service after setup: %v, addr: %p", svc.config.TokenConfig.SigningKey, svc.config.TokenConfig.SigningKey)
       if svc.config.TokenConfig.SigningKey == nil {
	       t.Errorf("DEBUG: SigningKey is nil immediately after setupTestService!")
       } else {
	       t.Logf("DEBUG: SigningKey type: %T", svc.config.TokenConfig.SigningKey)
       }
       require.NotNil(t, svc.config.TokenConfig.SigningKey, "SigningKey should not be nil after setupTestService")
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

	// Assert SigningKey is still present before RequestToken
	require.NotNil(t, svc.config.TokenConfig.SigningKey, "SigningKey should not be nil before RequestToken call")

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


func setupTestService(t *testing.T) *Service {
       signingKey := generateTestSigningKey(t)
       t.Logf("DEBUG: signingKey generated: %v, addr: %p", signingKey, signingKey)
       t.Logf("DEBUG: about to assign signingKey to TokenConfig.SigningKey")
       tokenCfg := &token.Config{
	       SigningKey:    signingKey,
	       SigningMethod: token.RS256,
	       ValidityPeriod: time.Hour,
       }
       config := &Config{
	       AuthServerURL:     "http://localhost:8080", // required for validation
	       ClientID:          "test-client",
	       ClientSecret:      "test-secret",
	       Scopes:            []string{"read", "write"},
	       AccessTokenExpiry: time.Hour,
	       RateLimit:         common.RateLimitConfig{},
	       TokenConfig:       tokenCfg,
       }
       t.Logf("DEBUG: Config.SigningKey after assignment: %v, addr: %p", config.TokenConfig.SigningKey, config.TokenConfig.SigningKey)
       svc, err := NewService(config)
       require.NoError(t, err)
       return svc
}

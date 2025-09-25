// Package gauth provides authentication integration for cascade services
package gauth

import (
	"context"
	"fmt"
	"time"
	
	"github.com/Gimel-Foundation/gauth/pkg/gauth"
)

// GAuth is an alias for ServiceAuth to maintain compatibility
type GAuth = ServiceAuth

// ServiceAuth wraps GAuth for service-to-service authentication
type ServiceAuth struct {
	client *gauth.GAuth
	config *gauth.Config
}

// NewServiceAuth creates a new service authentication client
func NewServiceAuth(config *gauth.Config) (*ServiceAuth, error) {
	client, err := gauth.New(*config)
	if err != nil {
		return nil, fmt.Errorf("failed to create gauth client: %w", err)
	}
	
	return &ServiceAuth{
		client: client,
		config: config,
	}, nil
}

// Authenticate performs service authentication
func (sa *ServiceAuth) Authenticate(ctx context.Context, serviceID string) (string, error) {
	req := gauth.AuthorizationRequest{
		ClientID: serviceID,
		Scopes:   []string{"service:access"},
	}
	
	grant, err := sa.client.InitiateAuthorization(req)
	if err != nil {
		return "", fmt.Errorf("failed to initiate authorization: %w", err)
	}
	
	tokenReq := gauth.TokenRequest{
		GrantID: grant.GrantID,
		Scope:   grant.Scope,
	}
	
	token, err := sa.client.RequestToken(tokenReq)
	if err != nil {
		return "", fmt.Errorf("failed to request token: %w", err)
	}
	
	return token.Token, nil
}

// ValidateToken validates a token and returns token data
func (sa *ServiceAuth) ValidateToken(token string) (*TokenResponse, error) {
	// In a real implementation, this would validate the token
	// For this example, we return a mock response
	return &TokenResponse{
		Token:      token,
		Scope:      []string{"transaction:execute", "service:access"},
		ValidUntil: time.Now().Add(time.Hour),
	}, nil
}
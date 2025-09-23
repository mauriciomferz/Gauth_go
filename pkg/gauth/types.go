package gauth

import (
	"context"
	"time"

	"github.com/Gimel-Foundation/gauth/pkg/common"
)

// AuthorizationRequest represents a request to initiate authorization (delegation)
type AuthorizationRequest struct {
	ClientID string
	Scopes   []string
}

// TokenType represents the type of token issued
// (access_token, refresh_token, etc.)
type TokenType string

const (
	// AccessToken represents a short-lived token for resource access
	AccessToken TokenType = "access_token"
	// RefreshToken represents a long-lived token for obtaining new access tokens
	RefreshToken TokenType = "refresh_token"
)

// AuthorizationGrant represents the granted authorization
type AuthorizationGrant struct {
	GrantID      string
	ClientID     string
	Scope        []string
	Restrictions []Restriction
	ValidUntil   time.Time
}

// TokenRequest represents a request for a token
type TokenRequest struct {
	GrantID      string
	Scope        []string
	Restrictions []Restriction
	Context      context.Context
}

// TokenResponse represents the response to a token request
type TokenResponse struct {
	Token        string
	ValidUntil   time.Time
	Scope        []string
	Restrictions []Restriction
}

// Config represents the configuration for GAuth
type Config struct {
	AuthServerURL     string                 // URL of the authorization server
	ClientID          string                 // Client identifier
	ClientSecret      string                 // Client secret
	Scopes            []string               // Default scopes
	RateLimit         common.RateLimitConfig // Rate limiting configuration
	AccessTokenExpiry time.Duration          // Token expiry duration
	SigningKey        interface{}            // Signing key for token generation (crypto.Signer)
}

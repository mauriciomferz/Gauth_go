// Package auth provides authentication and authorization functionality

package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/Gimel-Foundation/gauth/pkg/audit"
)

type Type string

const (
	// TypeBasic represents basic auth
	TypeBasic Type = "basic"
	// TypeOAuth2 represents OAuth2 flow
	TypeOAuth2 Type = "oauth2"
	// TypeJWT represents JWT token auth
	TypeJWT Type = "jwt"
	// TypePaseto represents PASETO token auth
	TypePaseto Type = "paseto"
)

// Config holds the configuration for auth operations
type Config struct {
	// Type is the type of authorization to use
	Type Type

	// AuthServerURL is the URL of the auth server
	AuthServerURL string

	// ClientID is the ID of the client
	ClientID string

	// ClientSecret is the secret of the client
	ClientSecret string

	// Scopes are the default scopes to request
	Scopes []string

	// AccessTokenExpiry is the expiry duration for access tokens
	AccessTokenExpiry time.Duration

	// AuditLogger is the logger for audit events
	AuditLogger *audit.AuditLogger

	// TokenValidation holds token validation config
	TokenValidation TokenValidationConfig

	// ApprovalRules for compliance (added for RFC111/core example compatibility)
	ApprovalRules []ApprovalRule

	// ExtraConfig for OAuth2 and other advanced flows
	ExtraConfig interface{}
}

// TokenValidationConfig holds token validation configuration
type TokenValidationConfig struct {
	AllowedIssuers    []string
	AllowedAudiences  []string
	RequiredScopes    []string
	RequiredClaims    Claims
	ClockSkew         time.Duration
	ValidateSignature bool
}

// Authenticator defines the interface for authentication operations
type Authenticator interface {
	// Initialize sets up any necessary resources
	Initialize(ctx context.Context) error

	// Close releases any held resources
	Close() error

	// ValidateCredentials validates the provided credentials
	ValidateCredentials(ctx context.Context, creds interface{}) error

	// GenerateToken generates a new token for the authenticated user/client
	GenerateToken(ctx context.Context, req TokenRequest) (*TokenResponse, error)

	// ValidateToken validates a token and returns its data
	ValidateToken(ctx context.Context, token string) (*TokenData, error)

	// RevokeToken revokes a token
	RevokeToken(ctx context.Context, token string) error
}

// Metadata represents additional metadata for a token request
type Metadata struct {
	IPAddress  string
	Device     string
	UserAgent  string
	CustomData map[string]string
}

// TokenRequest represents a request for a new token
type TokenRequest struct {
	GrantType string
	Scopes    []string
	Audience  string
	Subject   string
	ExpiresIn time.Duration
	Metadata  map[string]interface{}
}

// TokenResponse represents a successful token generation
type TokenResponse struct {
	Token     string
	TokenType string
	ExpiresIn int64
	Scope     []string
	Claims    Claims
}

// TokenData represents validated token data
type TokenData struct {
	Valid     bool
	Subject   string
	Issuer    string
	Audience  string
	IssuedAt  time.Time
	ExpiresAt time.Time
	Scope     []string
	Claims    Claims
}

// Claims is a type alias for map[string]interface{} for unified claims/metadata handling
// This replaces the previous *Claims struct usage everywhere

type Claims map[string]interface{}

// NewAuthenticator creates a new authenticator based on the config
func NewAuthenticator(config Config) (Authenticator, error) {
	switch config.Type {
	case TypeBasic:
		return newBasicAuthenticator(config)
	case TypeOAuth2:
		return newOAuth2Authenticator(config)
	case TypeJWT:
		return newJWTAuthenticator(config)
	case TypePaseto:
		return newPasetoAuthenticator(config)
	default:
		return nil, fmt.Errorf("unsupported auth type: %s", config.Type)
	}
}

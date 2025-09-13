
// Package auth provides authentication and authorization functionality
package auth

import (
    "context"
    "fmt"
)

type TokenResponse struct {
    // Token is the generated token string
    Token string

    // TokenType is the type of token (e.g., "Bearer")
    TokenType string

    // ExpiresIn is the token lifetime in seconds
    ExpiresIn int64

    // Scope contains the granted scopes
    Scope []string

    // Claims contains additional token claims
	Claims *claims.Claims
}m/mauricio.fernandez_fernandezsiemens.co/gauth/pkg/audit"
	"github.com/Gimel-Foundation/gauth/pkg/auth/claims"
)

// Type represents the type of authorization
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
	AuditLogger *audit.Logger

	// TokenValidation holds token validation config
	TokenValidation TokenValidationConfig
}

// TokenValidationConfig holds token validation configuration
type TokenValidationConfig struct {
	// AllowedIssuers is a list of valid token issuers
	AllowedIssuers []string

	// AllowedAudiences is a list of valid token audiences
	AllowedAudiences []string

	// RequiredScopes are scopes that must be present
	RequiredScopes []string

	// RequiredClaims are claims that must be present and match
	RequiredClaims map[string]interface{}

	// ClockSkew allows for small time differences
	ClockSkew time.Duration

	// ValidateSignature indicates if signature validation is required
	ValidateSignature bool
}

// Authenticator defines the interface for authentication operations
type Authenticator interface {
	// Initialize sets up any necessary resources
	Initialize(context.Context) error

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

// TokenRequest represents a request for a new token
type TokenRequest struct {
	// GrantType is the type of grant being requested
	GrantType string

	// Scopes are the requested scopes
	Scopes []string

	// Audience is the intended audience for the token
	Audience string

	// Subject is the subject of the token
	Subject string

	// ExpiresIn is the requested token lifetime
	ExpiresIn time.Duration

	// Metadata is additional metadata to include
	Metadata *Claims
}

// TokenResponse represents a successful token generation
type TokenResponse struct {
	// Token is the generated token string
	Token string

	// TokenType is the type of token (e.g., "Bearer")
	TokenType string

	// ExpiresIn is the token lifetime in seconds
	ExpiresIn int64

	// Scope contains the granted scopes
	Scope []string

	// Claims contains additional token claims
	Claims *Claims
}

// TokenData represents validated token data
type TokenData struct {
	// Valid indicates if the token is valid
	Valid bool

	// Subject is the token subject
	Subject string

	// Issuer is the token issuer
	Issuer string

	// Audience is the token audience
	Audience string

	// IssuedAt is when the token was issued
	IssuedAt time.Time

	// ExpiresAt is when the token expires
	ExpiresAt time.Time

	// Scope contains the granted scopes
	Scope []string

	// Claims contains additional token claims
	Claims map[string]interface{}
}

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

// These are placeholder functions that will be implemented in separate files
func newBasicAuthenticator(config Config) (Authenticator, error)  { return nil, nil }
func newOAuth2Authenticator(config Config) (Authenticator, error) { return nil, nil }
func newJWTAuthenticator(config Config) (Authenticator, error)    { return nil, nil }
func newPasetoAuthenticator(config Config) (Authenticator, error) { return nil, nil }

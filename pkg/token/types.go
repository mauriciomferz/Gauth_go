//
// # Licensing
//
// This file is part of the GAuth project and is licensed under the Apache License 2.0.
// It incorporates code and concepts from:
//   - OAuth 2.0 and OpenID Connect (Apache 2.0 License)
//   - Model Context Protocol (MIT License)
// See the LICENSE file in the project root for details.

// Package token provides token management functionality
package token

import (
	"context"
	"crypto"
	"time"
)

// Type represents the type of a token.
// RFC111: Used to distinguish access, refresh, and ID tokens for protocol compliance.
type Type string

const (
	// Access represents an OAuth2 access token
	Access Type = "access_token"
	// Refresh represents an OAuth2 refresh token
	Refresh Type = "refresh_token"
	// ID represents an OpenID Connect ID token
	ID Type = "id_token"
)

// Algorithm represents a token signing algorithm.
// RFC111: Ensures cryptographic compliance and auditability.
type Algorithm string

const (
	// RS256 is RSA with SHA-256
	RS256 Algorithm = "RS256"
	// ES256 is ECDSA with SHA-256
	ES256 Algorithm = "ES256"
	// HS256 is HMAC with SHA-256
	HS256 Algorithm = "HS256"
	// PS256 is RSA-PSS with SHA-256
	PS256 Algorithm = "PS256"
)

// DeviceInfo contains information about the device using the token.
// RFC111: Device binding is optional but recommended for traceability.
type DeviceInfo struct {
	ID        string `json:"id"`         // Unique device identifier
	UserAgent string `json:"user_agent"` // Device user agent string
	IPAddress string `json:"ip_address"` // Device IP address
	Platform  string `json:"platform,omitempty"`
	Version   string `json:"version,omitempty"`
}

// RevocationStatus contains information about token revocation.
// RFC111: Required for transparent and auditable revocation.
type RevocationStatus struct {
	RevokedAt time.Time `json:"revoked_at"`           // When the token was revoked
	Reason    string    `json:"reason"`               // Reason for revocation
	RevokedBy string    `json:"revoked_by,omitempty"` // Who revoked the token
}

// Metadata contains strongly-typed token metadata.
// RFC111: Used for delegation, attestation, and application context.
type Metadata struct {
	Device     *DeviceInfo         `json:"device,omitempty"`      // Device info
	AppID      string              `json:"app_id,omitempty"`      // Application ID
	AppVersion string              `json:"app_version,omitempty"` // Application version
	AppData    map[string]string   `json:"app_data,omitempty"`    // Arbitrary app data (e.g., power-of-attorney)
	Labels     map[string]string   `json:"labels,omitempty"`
	Tags       []string            `json:"tags,omitempty"`
	Attributes map[string][]string `json:"attributes,omitempty"`
}

// Token represents a security token with metadata.
// RFC111: Central type for all protocol flows, including grant, attestation, and revocation.
type Token struct {
	ID               string            `json:"id"`    // Unique token identifier
	Value            string            `json:"token"` // Token value (e.g., JWT)
	Type             Type              `json:"type"`  // Token type
	IssuedAt         time.Time         `json:"iat"`   // Issuance time
	ExpiresAt        time.Time         `json:"exp"`   // Expiry time
	NotBefore        time.Time         `json:"nbf"`   // Not valid before
	LastUsedAt       *time.Time        `json:"last_used_at,omitempty"`
	Issuer           string            `json:"iss"` // Token issuer (must be central authority per RFC111)
	Subject          string            `json:"sub"` // Token subject (user, agent, etc.)
	Audience         []string          `json:"aud"`
	Scopes           []string          `json:"scope"`
	Algorithm        Algorithm         `json:"alg"`
	Metadata         *Metadata         `json:"metadata,omitempty"`
	RevocationStatus *RevocationStatus `json:"revocation_status,omitempty"`
}

// Claims represents standard JWT claims
type Claims struct {
	// Issuer identifies who created the token
	Issuer string `json:"iss"`

	// Subject identifies the principal that is the subject of the token
	Subject string `json:"sub"`

	// Audience identifies the recipients that the token is intended for
	Audience []string `json:"aud"`

	// ExpiresAt is when the token expires
	ExpiresAt time.Time `json:"exp"`

	// NotBefore is when the token becomes valid
	NotBefore time.Time `json:"nbf"`

	// IssuedAt is when the token was created
	IssuedAt time.Time `json:"iat"`

	// ID provides a unique identifier for the token
	ID string `json:"jti"`

	// Scopes are the permissions granted by this token
	Scopes []string `json:"scope"`

	// Additional claims can be added via this map (now type-safe)
	Extra map[string]string `json:"extra,omitempty"`
}

// Store defines the interface for token storage and management
type Store interface {
	// Save stores a token with the given key
	Save(ctx context.Context, key string, token *Token) error

	// Get retrieves a token by key
	Get(ctx context.Context, key string) (*Token, error)

	// Delete removes a token
	Delete(ctx context.Context, key string) error

	// List returns all tokens matching the filter
	List(ctx context.Context, filter Filter) ([]*Token, error)

	// Rotate replaces an existing token with a new one
	Rotate(ctx context.Context, old, new *Token) error

	// Revoke invalidates a token before its natural expiration
	Revoke(ctx context.Context, token *Token) error

	// Validate checks if a token is valid and active
	Validate(ctx context.Context, token *Token) error

	// Refresh generates a new access token from a refresh token
	Refresh(ctx context.Context, refreshToken *Token) (*Token, error)

	// Count returns the number of tokens matching the filter
	Count(ctx context.Context, filter Filter) (int64, error)

	// Cleanup removes expired tokens
	Cleanup(ctx context.Context) error

	// Close releases resources used by the store
	Close() error
}

// Filter defines criteria for querying tokens
type Filter struct {
	// Types are the token types to include
	Types []Type `json:"types"`

	// Subject filters by token subject
	Subject string `json:"subject"`

	// Issuer filters by token issuer
	Issuer string `json:"issuer"`

	// ExpiresAfter filters tokens expiring after this time
	ExpiresAfter time.Time `json:"expires_after"`

	// ExpiresBefore filters tokens expiring before this time
	ExpiresBefore time.Time `json:"expires_before"`

	// IssuedAfter filters tokens issued after this time
	IssuedAfter time.Time `json:"issued_after"`

	// IssuedBefore filters tokens issued before this time
	IssuedBefore time.Time `json:"issued_before"`

	// Scopes filters tokens with any of these scopes
	Scopes []string `json:"scopes"`

	// RequireAllScopes indicates if all scopes must be present
	RequireAllScopes bool `json:"require_all_scopes"`

	// Active filters for only currently valid tokens
	Active bool `json:"active"`

	// Metadata filters by token metadata matching all key-value pairs
	Metadata map[string]string `json:"metadata"`
}

// Config contains token configuration options
type Config struct {
	// Store is the token storage backend
	Store Store

	// SigningMethod is the algorithm used to sign tokens
	SigningMethod Algorithm

	// SigningKey is the key used to sign tokens
	SigningKey crypto.Signer

	// ValidityPeriod is how long tokens are valid for
	ValidityPeriod time.Duration

	// RefreshPeriod is how long refresh tokens are valid for
	RefreshPeriod time.Duration

	// CleanupInterval is how often to clean up expired tokens
	CleanupInterval time.Duration

	// MaxTokens is the maximum number of active tokens allowed
	MaxTokens int

	// DefaultScopes are the default scopes for new tokens
	DefaultScopes []string

	// ValidateAudience indicates if audience validation is required
	ValidateAudience bool

	// ValidateIssuer indicates if issuer validation is required
	ValidateIssuer bool

	// AllowedIssuers are the allowed token issuers
	AllowedIssuers []string

	// AllowedAudiences are the allowed token audiences
	AllowedAudiences []string
}

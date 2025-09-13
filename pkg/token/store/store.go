/*
Package store provides token storage implementations for secure token management.

DEPRECATED: Use github.com/Gimel-Foundation/gauth/pkg/token package instead.

This package is kept for backward compatibility with older code.
New code should use the token package directly.

The store package offers:
  - Multiple storage backends
  - Secure token encryption
  - TTL support
  - Distributed storage options
  - Automatic cleanup
  - Type-safe operations

For more examples, see examples/token_management/
*/
package store

import (
	"context"
	"fmt"
	"time"

	"github.com/Gimel-Foundation/gauth/pkg/errors"
)

// TokenMetadata contains structured metadata about a token
type TokenMetadata struct {
	UserID      string    // Associated user ID
	ClientID    string    // Client that requested the token
	Scopes      []string  // Token scopes
	IssuedAt    time.Time // When the token was issued
	ExpiresAt   time.Time // When the token expires
	IPAddress   string    // IP address of token request
	DeviceInfo  string    // Device/user agent info
	IsRevoked   bool      // Whether token is revoked
	RevokedAt   time.Time // When token was revoked
	RevokedBy   string    // Who revoked the token
	LastUsed    time.Time // Last time token was used
	UseCount    int64     // Number of times used
	SessionID   string    // Associated session ID
	Fingerprint string    // Device fingerprint
}

// Store defines the interface for token storage backends
type Store interface {
	// Save stores a token with metadata and TTL
	Save(ctx context.Context, token string, metadata TokenMetadata) error

	// Get retrieves token metadata
	Get(ctx context.Context, token string) (*TokenMetadata, error)

	// Delete removes a token
	Delete(ctx context.Context, token string) error

	// Revoke marks a token as revoked
	Revoke(ctx context.Context, token string, revokedBy string) error

	// IsRevoked checks if a token is revoked
	IsRevoked(ctx context.Context, token string) (bool, error)

	// UpdateLastUsed updates token usage info
	UpdateLastUsed(ctx context.Context, token string) error

	// ListExpired returns tokens that have expired
	ListExpired(ctx context.Context) ([]string, error)

	// Cleanup removes expired tokens
	Cleanup(ctx context.Context) error

	// Close cleans up resources
	Close() error
}

// Config contains common configuration for token stores
type Config struct {
	// Required fields
	EncryptionKey []byte // Key for encrypting tokens

	// Optional fields with defaults
	TokenTTL        time.Duration // Default token lifetime
	CleanupInterval time.Duration // How often to remove expired tokens
	MaxTokens       int64         // Maximum number of tokens to store
	EnableMetrics   bool          // Whether to collect metrics
}

// Validate checks the config is valid
func (c *Config) Validate() error {
	if len(c.EncryptionKey) == 0 {
		return errors.ErrMissingEncryptionKey
	}
	if c.TokenTTL == 0 {
		c.TokenTTL = 24 * time.Hour // Default 1 day
	}
	if c.CleanupInterval == 0 {
		c.CleanupInterval = time.Hour // Default hourly cleanup
	}
	if c.MaxTokens == 0 {
		c.MaxTokens = 1_000_000 // Default 1M tokens
	}
	return nil
}

// Validate checks token metadata is valid
func (m *TokenMetadata) Validate() error {
	if m.UserID == "" {
		return errors.ErrMissingUserID
	}
	if m.ClientID == "" {
		return errors.ErrMissingClientID
	}
	if m.ExpiresAt.IsZero() {
		return errors.ErrMissingExpiry
	}
	if m.ExpiresAt.Before(time.Now()) {
		return errors.ErrTokenExpired
	}
	return nil
}

// IsExpired checks if the token has expired
func (m *TokenMetadata) IsExpired() bool {
	return !m.ExpiresAt.IsZero() && time.Now().After(m.ExpiresAt)
}

// TimeToExpiry returns how long until the token expires
func (m *TokenMetadata) TimeToExpiry() time.Duration {
	if m.ExpiresAt.IsZero() {
		return 0
	}
	return time.Until(m.ExpiresAt)
}

// String returns a string representation of token metadata
func (m *TokenMetadata) String() string {
	return fmt.Sprintf(
		"Token{user=%s, client=%s, expires=%s, revoked=%v}",
		m.UserID,
		m.ClientID,
		m.ExpiresAt.Format(time.RFC3339),
		m.IsRevoked,
	)
}

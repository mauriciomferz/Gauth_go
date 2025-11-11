package gauth

import (
	"context"
	"errors"
	"time"
)

// ExtendedTokenStore defines the interface for storing and retrieving extended tokens.
// This supports RFC-0111 token lifecycle management including validation,
// introspection, and revocation (RFC 7009).
type ExtendedTokenStore interface {
	// SaveToken stores a new extended token
	SaveToken(ctx context.Context, token *ExtendedToken) error
	
	// GetToken retrieves a token by its access token value
	GetToken(ctx context.Context, accessToken string) (*ExtendedToken, error)
	
	// GetTokenByRefreshToken retrieves a token by its refresh token value
	GetTokenByRefreshToken(ctx context.Context, refreshToken string) (*ExtendedToken, error)
	
	// RevokeToken marks a token as revoked
	RevokeToken(ctx context.Context, accessToken string) error
	
	// IsRevoked checks if a token has been revoked
	IsRevoked(ctx context.Context, accessToken string) (bool, error)
	
	// DeleteExpiredTokens removes expired tokens (cleanup operation)
	DeleteExpiredTokens(ctx context.Context) (int, error)
	
	// ListTokensByClient returns all tokens for a specific client
	ListTokensByClient(ctx context.Context, clientID string) ([]*ExtendedToken, error)
}

// Common errors for token store operations
var (
	ErrTokenNotFound      = errors.New("token not found")
	ErrTokenAlreadyExists = errors.New("token already exists")
	ErrTokenRevoked       = errors.New("token has been revoked")
	// ErrTokenExpired is defined in gauth.go
)

// TokenMetadata holds additional metadata for token storage
type TokenMetadata struct {
	CreatedAt   time.Time
	RevokedAt   *time.Time
	LastUsedAt  *time.Time
	UseCount    int
}

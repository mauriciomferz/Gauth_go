// Package tokenstore provides token storage functionality for GAuth

package tokenstore

import "time"

// TokenData represents the data associated with a token
type TokenData struct {
	Valid      bool
	ValidUntil time.Time
	ClientID   string
	OwnerID    string
	Scope      []string
}

// Store defines the interface for token storage implementations
type Store interface {
	// Store stores a token with its associated data
	Store(token string, data TokenData) error

	// Get retrieves token data for a given token
	Get(token string) (TokenData, bool)

	// Delete removes a token from the store
	Delete(token string) error

	// Cleanup removes expired tokens
	Cleanup() error
}



// Package tokenstore provides token storage functionality for GAuth

package tokenstore

import "time"

// TokenData represents the data associated with a token.
// Valid indicates if the token is currently valid.
// ValidUntil is the expiration time.
// ClientID and OwnerID identify the client and owner.
// Scope lists the granted scopes.
type TokenData struct {
	Valid      bool
	ValidUntil time.Time
	ClientID   string
	OwnerID    string
	Scope      []string
}


// Store defines the interface for token storage implementations.
// Implementations must be safe for concurrent use.
//
// Example usage:
//   var s tokenstore.Store = tokenstore.NewMemoryStore()
//   err := s.Store("token123", tokenstore.TokenData{Valid: true, ValidUntil: time.Now().Add(time.Hour)})
//   data, ok := s.Get("token123")
//   err = s.Delete("token123")
type Store interface {
	// Store stores a token with its associated data.
	Store(token string, data TokenData) error

	// Get retrieves token data for a given token.
	Get(token string) (TokenData, bool)

	// Delete removes a token from the store.
	Delete(token string) error

	// Cleanup removes expired tokens.
	Cleanup() error
}



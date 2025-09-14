package store

import (
	"context"
	"time"
)

// TokenMetadata contains metadata about a stored token
type TokenMetadata struct {
	ID        string    `json:"id"`
	Subject   string    `json:"subject"`
	Issuer    string    `json:"issuer"`
	IssuedAt  time.Time `json:"issued_at"`
	ExpiresAt time.Time `json:"expires_at"`
	KeyID     string    `json:"key_id"`
	Type      string    `json:"type"`
	Status    string    `json:"status"`
}

// TokenStore defines the interface for token storage backends
type TokenStore interface {
	// Store stores a token with its metadata
	Store(ctx context.Context, token string, metadata TokenMetadata) error

	// Get retrieves token metadata by token string
	Get(ctx context.Context, token string) (*TokenMetadata, error)

	// GetByID retrieves token metadata by token ID
	GetByID(ctx context.Context, id string) (*TokenMetadata, error)

	// Delete removes a token from storage
	Delete(ctx context.Context, token string) error

	// List returns all tokens for a subject
	List(ctx context.Context, subject string) ([]TokenMetadata, error)

	// Revoke marks a token as revoked
	Revoke(ctx context.Context, token string) error

	// IsRevoked checks if a token is revoked
	IsRevoked(ctx context.Context, token string) (bool, error)

	// Cleanup removes expired tokens
	Cleanup(ctx context.Context) error
}

// StorageError represents token store specific errors
type StorageError struct {
	Op     string // Operation that failed
	Key    string // Key that caused the error
	Err    error  // Underlying error
	Detail string // Additional context about the error
}

func (e *StorageError) Error() string {
	if e.Detail != "" {
		if e.Key != "" {
			return "store." + e.Op + ": " + e.Key + ": " + e.Err.Error() + " - " + e.Detail
		}
		return "store." + e.Op + ": " + e.Err.Error() + " - " + e.Detail
	}
	if e.Key != "" {
		return "store." + e.Op + ": " + e.Key + ": " + e.Err.Error()
	}
	return "store." + e.Op + ": " + e.Err.Error()
}

// StorageConfig contains configuration for token stores
type StorageConfig struct {
	// Type specifies the type of storage backend
	Type string `json:"type"`

	// ConnectionString for the storage backend
	ConnectionString string `json:"connection_string"`

	// Namespace for token storage
	Namespace string `json:"namespace"`

	// MaxTokens is the maximum number of tokens to store
	MaxTokens int `json:"max_tokens"`

	// CleanupInterval is how often to run cleanup
	CleanupInterval time.Duration `json:"cleanup_interval"`

	// BackendOptions contains backend-specific options as a typed struct.
	// NOTE: interface{} is used here only for backend plugin config flexibility.
	// All public APIs use type-safe alternatives. Do not expose in new APIs.
	// Use a concrete struct for each backend (e.g., MemoryOptions, RedisOptions, etc.)
	BackendOptions interface{} `json:"backend_options,omitempty"`
}

// Example: MemoryOptions for in-memory backend
type MemoryOptions struct {
	EvictionPolicy string        `json:"eviction_policy"`
	MaxIdle        time.Duration `json:"max_idle"`
}

// Example: RedisOptions for Redis backend
type RedisOptions struct {
	Addr     string        `json:"addr"`
	Password string        `json:"password"`
	DB       int           `json:"db"`
	Timeout  time.Duration `json:"timeout"`
}

// TokenQuery defines criteria for token searches
type TokenQuery struct {
	Subject   string    `json:"subject"`
	Issuer    string    `json:"issuer"`
	Type      string    `json:"type"`
	Status    string    `json:"status"`
	IssuedAt  time.Time `json:"issued_at"`
	ExpiresAt time.Time `json:"expires_at"`
	KeyID     string    `json:"key_id"`
}

// StoreStats contains statistics about the token store
type StoreStats struct {
	TotalTokens     int       `json:"total_tokens"`
	ActiveTokens    int       `json:"active_tokens"`
	RevokedTokens   int       `json:"revoked_tokens"`
	ExpiredTokens   int       `json:"expired_tokens"`
	LastCleanup     time.Time `json:"last_cleanup"`
	StorageSize     int64     `json:"storage_size"`
	StorageCapacity int64     `json:"storage_capacity"`
}

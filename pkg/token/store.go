package token

import (
	"time"
)

// StoreOption is a function that configures a Store
type StoreOption func(Store) error

// WithTTL sets the default TTL for tokens
func WithTTL(ttl time.Duration) StoreOption {
	return func(s Store) error {
		if configurable, ok := s.(interface{ SetDefaultTTL(time.Duration) }); ok {
			configurable.SetDefaultTTL(ttl)
		}
		return nil
	}
}

// WithCleanup enables automatic cleanup of expired tokens
func WithCleanup(interval time.Duration) StoreOption {
	return func(s Store) error {
		if configurable, ok := s.(interface{ EnableCleanup(time.Duration) }); ok {
			configurable.EnableCleanup(interval)
		}
		return nil
	}
}

// WithCapacity sets the maximum number of tokens the store can hold
func WithCapacity(n int) StoreOption {
	return func(s Store) error {
		if configurable, ok := s.(interface{ SetCapacity(int) }); ok {
			configurable.SetCapacity(n)
		}
		return nil
	}
}

// StoreConfig holds common store configuration
type StoreConfig struct {
	// Default TTL for tokens
	DefaultTTL time.Duration

	// Maximum number of tokens
	MaxTokens int

	// Cleanup configuration
	CleanupEnabled  bool
	CleanupInterval time.Duration
}

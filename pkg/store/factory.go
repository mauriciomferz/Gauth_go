// Package store provides token storage implementations for GAuth
package store

import (
	"errors"
	"fmt"
)

// Type represents the type of store
type Type string

const (
	// MemoryStoreType represents an in-memory store
	MemoryStoreType Type = "memory"
	// Redis represents a Redis-backed token store
	Redis Type = "redis"
	// Database represents a database-backed token store
	Database Type = "database"
)

// Common factory errors
var (
	ErrInvalidStoreType = errors.New("invalid store type")
	ErrMissingConfig    = errors.New("missing configuration")
	ErrInvalidConfig    = errors.New("invalid configuration")
)

// NewTokenStore creates a new token store based on the provided configuration
func NewTokenStore(storeType Type, config interface{}) (TokenStore, error) {
	switch storeType {
	case MemoryStoreType:
		// Handle memory store configuration
		var cfg Config
		if config != nil {
			if memCfg, ok := config.(Config); ok {
				cfg = memCfg
			} else {
				return nil, fmt.Errorf("%w: expected Config for memory store", ErrInvalidConfig)
			}
		} else {
			cfg = DefaultConfig()
		}
		return NewMemoryStore(cfg)

	case Redis:
		// Handle Redis store configuration
		var cfg RedisConfig
		if config != nil {
			if redisCfg, ok := config.(RedisConfig); ok {
				cfg = redisCfg
			} else {
				return nil, fmt.Errorf("%w: expected RedisConfig for Redis store", ErrInvalidConfig)
			}
		} else {
			cfg = DefaultRedisConfig()
		}
		return NewRedisStore(cfg)

	case Database:
		// Database store not yet implemented
		return nil, fmt.Errorf("%w: database store not yet implemented", ErrInvalidStoreType)

	default:
		return nil, fmt.Errorf("%w: %s", ErrInvalidStoreType, storeType)
	}
}

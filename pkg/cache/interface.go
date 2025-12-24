package cache

import (
	"context"
	"fmt"
	"os"
	"time"
)

// Cache defines the interface for caching operations
type Cache interface {
	// Get retrieves a value from the cache
	Get(ctx context.Context, key string) ([]byte, error)

	// Set stores a value in the cache with the given TTL
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error

	// Delete removes a value from the cache
	Delete(ctx context.Context, key string) error

	// DeletePattern removes all keys matching the pattern
	DeletePattern(ctx context.Context, pattern string) error

	// Exists checks if a key exists in the cache
	Exists(ctx context.Context, key string) (bool, error)

	// GetStats returns cache statistics
	GetStats(ctx context.Context) (*Stats, error)

	// Close closes the cache connection
	Close() error

	// Ping checks if the cache is reachable
	Ping(ctx context.Context) error
}

// Stats represents cache statistics
type Stats struct {
	Hits        int64   `json:"hits"`
	Misses      int64   `json:"misses"`
	Keys        int64   `json:"keys"`
	Memory      int64   `json:"memory_bytes"`
	HitRate     float64 `json:"hit_rate"`
	Uptime      int64   `json:"uptime_seconds"`
	Connections int     `json:"connections"`
}

// Config represents cache configuration
type Config struct {
	// Type of cache: "redis" or "memory"
	Type string

	// Redis configuration
	RedisURL      string
	RedisPassword string
	RedisDB       int
	PoolSize      int
	MaxRetries    int

	// TTL configuration
	VerificationTTL time.Duration
	PoATTL          time.Duration
	StatsTTL        time.Duration

	// Memory cache configuration (for fallback)
	MaxSize int
}

// CacheType represents the type of cached data
type CacheType string

const (
	CacheTypeVerification CacheType = "verification"
	CacheTypePoA          CacheType = "poa"
	CacheTypeStats        CacheType = "stats"
	CacheTypeUser         CacheType = "user"
)

// GetTTL returns the appropriate TTL for a cache type
func (c *Config) GetTTL(cacheType CacheType) time.Duration {
	switch cacheType {
	case CacheTypeVerification:
		if c.VerificationTTL > 0 {
			return c.VerificationTTL
		}
		return 5 * time.Minute
	case CacheTypePoA:
		if c.PoATTL > 0 {
			return c.PoATTL
		}
		return 1 * time.Minute
	case CacheTypeStats:
		if c.StatsTTL > 0 {
			return c.StatsTTL
		}
		return 30 * time.Second
	case CacheTypeUser:
		return 5 * time.Minute
	default:
		return 1 * time.Minute
	}
}

// DefaultConfig returns a default cache configuration
func DefaultConfig() *Config {
	host := "localhost"
	if h := os.Getenv("REDIS_HOST"); h != "" {
		host = h
	}
	port := "6379"
	if p := os.Getenv("REDIS_PORT"); p != "" {
		port = p
	}
	return &Config{
		Type:            "redis",
		RedisURL:        fmt.Sprintf("redis://%s:%s", host, port),
		RedisPassword:   "",
		RedisDB:         0,
		PoolSize:        10,
		MaxRetries:      3,
		VerificationTTL: 5 * time.Minute,
		PoATTL:          1 * time.Minute,
		StatsTTL:        30 * time.Second,
		MaxSize:         1000,
	}
}

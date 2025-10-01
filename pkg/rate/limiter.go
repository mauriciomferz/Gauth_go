// Package rate provides rate limiting functionality
package rate

import (
	"context"
	"errors"
	"sync"
	"time"
)

// Common errors
var (
	ErrRateLimitExceeded = errors.New("rate limit exceeded")
	ErrInvalidConfig     = errors.New("invalid rate limiter configuration")
)

// Config defines the configuration for rate limiters
type Config struct {
	// Rate defines the number of requests allowed per time window
	Rate int64

	// Window defines the time window for rate limiting
	Window time.Duration

	// BurstSize defines the maximum burst size for token bucket algorithm
	BurstSize int64

	// DistributedConfig holds configuration for distributed rate limiting
	DistributedConfig *RedisConfig
}

// RedisConfig defines Redis configuration for distributed rate limiting
type RedisConfig struct {
	// Addresses holds the Redis server addresses
	Addresses []string

	// Password for Redis authentication
	Password string

	// DB number to use
	DB int

	// KeyPrefix for Redis keys
	KeyPrefix string
}

// Limiter defines the interface for rate limiting
type Limiter interface {
	// Allow checks if a request is allowed for the given ID
	Allow(ctx context.Context, id string) error

	// GetRemainingRequests returns the number of remaining requests for the ID
	GetRemainingRequests(id string) int64

	// Reset resets the rate limit for the given ID
	Reset(id string)
}

// TokenBucket implements the token bucket algorithm
type TokenBucket struct {
	rate      int64
	burstSize int64
	tokens    sync.Map
	lastTime  sync.Map
	mu        sync.RWMutex
}

// NewTokenBucket creates a new token bucket rate limiter
func NewTokenBucket(cfg Config) *TokenBucket {
	return &TokenBucket{
		rate:      cfg.Rate,
		burstSize: cfg.BurstSize,
	}
}

// Allow implements the Limiter interface
func (tb *TokenBucket) Allow(_ context.Context, id string) error {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	now := time.Now()

	// Load or initialize token count
	tokensIface, _ := tb.tokens.LoadOrStore(id, tb.burstSize)
	tokens := tokensIface.(int64)

	// Load last update time
	lastIface, _ := tb.lastTime.LoadOrStore(id, now)
	last := lastIface.(time.Time)

	// Calculate token replenishment
	elapsed := now.Sub(last)
	newTokens := tokens + int64(elapsed.Seconds())*tb.rate
	if newTokens > tb.burstSize {
		newTokens = tb.burstSize
	}

	// Check if request can be allowed
	if newTokens < 1 {
		return ErrRateLimitExceeded
	}

	// Update state
	tb.tokens.Store(id, newTokens-1)
	tb.lastTime.Store(id, now)

	return nil
}

// GetRemainingRequests implements the Limiter interface
func (tb *TokenBucket) GetRemainingRequests(id string) int64 {
	tb.mu.RLock()
	defer tb.mu.RUnlock()

	tokensIface, ok := tb.tokens.Load(id)
	if !ok {
		return tb.burstSize
	}
	return tokensIface.(int64)
}

// Reset implements the Limiter interface
func (tb *TokenBucket) Reset(id string) {
	tb.tokens.Store(id, tb.burstSize)
	tb.lastTime.Store(id, time.Now())
}

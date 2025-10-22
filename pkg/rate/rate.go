// Package rate provides rate limiting functionality
package rate

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// NewLimiterFromConfig allows creating a Limiter from a Config struct
func NewLimiterFromConfig(config *Config) *Limiter {
	return NewLimiter(*config)
}

// Allow checks if a request is allowed for the given id and context

// Config represents rate limiting configuration
type Config struct {
	RequestsPerSecond int           `json:"requests_per_second"`
	BurstSize         int           `json:"burst_size"`
	WindowSize        time.Duration `json:"window_size"`
}

// DefaultConfig returns the default rate limiting configuration
func DefaultConfig() Config {
	return Config{
		RequestsPerSecond: 100,
		BurstSize:         200,
		WindowSize:        time.Minute,
	}
}

// Limiter represents a rate limiter
type Limiter struct {
	config     Config
	tokens     int
	lastRefill time.Time
	now        func() time.Time // injectable clock for deterministic tests
	mu         sync.Mutex
}

// NewLimiter creates a new rate limiter
func NewLimiter(config Config) *Limiter {
	return &Limiter{
		config:     config,
		tokens:     config.BurstSize,
		lastRefill: time.Now(),
		now:        time.Now,
	}
}

// Allow checks if a request is allowed under the rate limit
func (l *Limiter) Allow() bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Refill tokens based on time elapsed
	now := l.now()
	elapsed := now.Sub(l.lastRefill)
	tokensToAdd := int(elapsed.Seconds() * float64(l.config.RequestsPerSecond))

	if tokensToAdd > 0 {
		l.tokens += tokensToAdd
		if l.tokens > l.config.BurstSize {
			l.tokens = l.config.BurstSize
		}
		l.lastRefill = now
	}

	// Check if we have tokens available
	if l.tokens > 0 {
		l.tokens--
		return true
	}

	return false
}

// Wait waits until a request is allowed
func (l *Limiter) Wait(ctx context.Context) error {
	for {
		if l.Allow() {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Millisecond * 10):
			// Continue waiting
		}
	}
}

// Algorithm represents different rate limiting algorithms
type Algorithm int

const (
	// TokenBucket represents the token bucket algorithm
	TokenBucket Algorithm = iota
	// SlidingWindow represents the sliding window algorithm
	SlidingWindow
	// FixedWindow represents the fixed window algorithm
	FixedWindow
)

// AllowClient checks if a request is allowed for a specific client ID
// This is a simple implementation that uses the same bucket for all clients
// In a production system, you would maintain per-client buckets
func (l *Limiter) AllowClient(ctx context.Context, clientID string) error {
	if !l.Allow() {
		return ErrLimitExceeded
	}
	return nil
}

// GetRemainingRequests returns the number of remaining requests in the current window
func (l *Limiter) GetRemainingRequests(clientID string) int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.tokens
}

// Reset resets the rate limiter for a specific client
func (l *Limiter) Reset(clientID string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.tokens = l.config.BurstSize
	l.lastRefill = l.now()
}

// Close closes the rate limiter and cleans up resources
func (l *Limiter) Close() error {
	// Simple implementation - just reset state
	l.mu.Lock()
	defer l.mu.Unlock()
	l.tokens = 0
	return nil
}

// String returns a string representation of the algorithm
func (a Algorithm) String() string {
	switch a {
	case TokenBucket:
		return "token_bucket"
	case SlidingWindow:
		return "sliding_window"
	case FixedWindow:
		return "fixed_window"
	default:
		return "unknown"
	}
}

// WrapTokenBucket wraps a function with token bucket rate limiting
func WrapTokenBucket(config Config, fn func() error) func() error {
	limiter := NewLimiter(config)

	return func() error {
		if !limiter.Allow() {
			return fmt.Errorf("rate limit exceeded")
		}
		return fn()
	}
}

// Demo demonstrates rate limiting functionality
func Demo() error {
	fmt.Println("=== Rate Limiting Demo ===")

	// Create a rate limiter
	config := Config{
		RequestsPerSecond: 5,  // 5 requests per second
		BurstSize:         10, // Allow burst of 10 requests
		WindowSize:        time.Second,
	}

	limiter := NewLimiter(config)

	fmt.Printf("Rate limiter configured for %d req/sec with burst size %d\n",
		config.RequestsPerSecond, config.BurstSize)

	// Test rate limiting
	allowed := 0
	denied := 0

	for i := 0; i < 20; i++ {
		if limiter.Allow() {
			allowed++
			fmt.Printf("✅ Request %d: ALLOWED\n", i+1)
		} else {
			denied++
			fmt.Printf("❌ Request %d: RATE LIMITED\n", i+1)
		}

		// Small delay between requests
		time.Sleep(time.Millisecond * 50)
	}

	fmt.Printf("\nResults: %d allowed, %d denied\n", allowed, denied)

	return nil
}

// Enhanced Config with additional fields for monitoring example compatibility
type EnhancedConfig struct {
	Config
	Rate   int           `json:"rate"`   // Alias for RequestsPerSecond
	Window time.Duration `json:"window"` // Alias for WindowSize
}

// NewEnhancedConfig creates an enhanced config with aliases
func NewEnhancedConfig(rate int, window time.Duration) EnhancedConfig {
	return EnhancedConfig{
		Config: Config{
			RequestsPerSecond: rate,
			BurstSize:         rate * 2,
			WindowSize:        window,
		},
		Rate:   rate,
		Window: window,
	}
}

// ErrLimitExceeded is returned when rate limit is exceeded
var ErrLimitExceeded = fmt.Errorf("rate limit exceeded")

// NewTokenBucket creates a new token bucket rate limiter
func NewTokenBucket(config EnhancedConfig) *TokenBucketWrapper {
	limiter := NewLimiter(config.Config)
	return &TokenBucketWrapper{limiter: limiter}
} // TokenBucket represents a token bucket rate limiter
type TokenBucketWrapper struct {
	limiter *Limiter
}

// Allow checks if a request should be allowed
func (tb *TokenBucketWrapper) Allow() error {
	if !tb.limiter.Allow() {
		return ErrLimitExceeded
	}
	return nil
}

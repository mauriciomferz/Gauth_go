package ratelimit

import (
	"context"
	"sync"
	"time"
)

// Algorithm represents different rate limiting algorithms
type Algorithm string

const (
	TokenBucket   Algorithm = "token_bucket"
	LeakyBucket   Algorithm = "leaky_bucket"
	FixedWindow   Algorithm = "fixed_window"
	SlidingWindow Algorithm = "sliding_window"
)

// Config represents rate limiter configuration
type Config struct {
	Algorithm Algorithm     `json:"algorithm"`
	Rate      int           `json:"rate"`   // requests per period
	Period    time.Duration `json:"period"` // time period
	Burst     int           `json:"burst"`  // maximum burst size
	KeyFunc   func() string `json:"-"`      // function to generate keys
}

// Limiter defines the rate limiter interface
type Limiter interface {
	Allow(key string) bool
	AllowN(key string, n int) bool
	Wait(ctx context.Context, key string) error
	WaitN(ctx context.Context, key string, n int) error
	Reset(key string)
	Close() error
}

// TokenBucketLimiter implements a token bucket rate limiter
type TokenBucketLimiter struct {
	config  Config
	buckets map[string]*bucket
	mutex   sync.RWMutex
}

type bucket struct {
	tokens     int
	lastRefill time.Time
	mutex      sync.Mutex
}

// NewLimiter creates a new rate limiter
func NewLimiter(config Config) Limiter {
	return &TokenBucketLimiter{
		config:  config,
		buckets: make(map[string]*bucket),
	}
}

// Allow checks if a request is allowed
func (l *TokenBucketLimiter) Allow(key string) bool {
	return l.AllowN(key, 1)
}

// AllowN checks if n requests are allowed
func (l *TokenBucketLimiter) AllowN(key string, n int) bool {
	l.mutex.Lock()
	b, exists := l.buckets[key]
	if !exists {
		b = &bucket{
			tokens:     l.config.Burst,
			lastRefill: time.Now(),
		}
		l.buckets[key] = b
	}
	l.mutex.Unlock()

	b.mutex.Lock()
	defer b.mutex.Unlock()

	now := time.Now()
	elapsed := now.Sub(b.lastRefill)

	// Calculate tokens to add
	tokensToAdd := int(elapsed.Nanoseconds() * int64(l.config.Rate) / l.config.Period.Nanoseconds())
	if tokensToAdd > 0 {
		b.tokens += tokensToAdd
		if b.tokens > l.config.Burst {
			b.tokens = l.config.Burst
		}
		b.lastRefill = now
	}

	if b.tokens >= n {
		b.tokens -= n
		return true
	}
	return false
}

// Wait waits until a request can be allowed
func (l *TokenBucketLimiter) Wait(ctx context.Context, key string) error {
	return l.WaitN(ctx, key, 1)
}

// WaitN waits until n requests can be allowed
func (l *TokenBucketLimiter) WaitN(ctx context.Context, key string, n int) error {
	for {
		if l.AllowN(key, n) {
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

// Reset resets the rate limiter for a specific key
func (l *TokenBucketLimiter) Reset(key string) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	delete(l.buckets, key)
}

// Close closes the rate limiter
func (l *TokenBucketLimiter) Close() error {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	l.buckets = make(map[string]*bucket)
	return nil
}

// WrapTokenBucket wraps a function with rate limiting
func WrapTokenBucket(fn func() error, config Config) func() error {
	limiter := NewLimiter(config)
	key := "default"
	if config.KeyFunc != nil {
		key = config.KeyFunc()
	}

	return func() error {
		if !limiter.Allow(key) {
			return &RateLimitError{Key: key, Limit: config.Rate}
		}
		return fn()
	}
}

// RateLimitError represents a rate limit exceeded error
type RateLimitError struct {
	Key   string
	Limit int
}

func (e *RateLimitError) Error() string {
	return "rate limit exceeded"
}

// DefaultConfig returns a default rate limiter configuration
func DefaultConfig() Config {
	return Config{
		Algorithm: TokenBucket,
		Rate:      100,
		Period:    time.Minute,
		Burst:     10,
	}
}

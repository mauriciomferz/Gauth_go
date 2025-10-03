// Package ratelimit provides rate limiting functionality for the GAuth protocol.
package ratelimit

import (
	"context"
	"sync"
	"time"
)

// RateLimiterConfig defines the configuration for a rate limiter
type RateLimiterConfig struct {
	// RequestLimit is the maximum number of requests allowed in the reset interval
	RequestLimit int

	// ResetInterval is the duration after which the request count is reset
	ResetInterval time.Duration

	// OnRateExceeded is called when the rate limit is exceeded
	OnRateExceeded func()
}

// RateLimiter provides basic rate limiting functionality
type RateLimiter struct {
	mu             sync.RWMutex
	requestCount   int
	lastReset      time.Time
	resetInterval  time.Duration
	requestLimit   int
	onRateExceeded func()
}

// NewRateLimiter creates a new rate limiter with the given configuration
func NewRateLimiter(config RateLimiterConfig) *RateLimiter {
	return &RateLimiter{
		requestLimit:   config.RequestLimit,
		resetInterval:  config.ResetInterval,
		lastReset:      time.Now(),
		onRateExceeded: config.OnRateExceeded,
	}
}

// Allow checks if a request is allowed and increments the counter if it is
func (r *RateLimiter) Allow(_ context.Context) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Reset if needed
	if time.Since(r.lastReset) > r.resetInterval {
		r.requestCount = 0
		r.lastReset = time.Now()
	}

	// Check limit
	if r.requestCount >= r.requestLimit {
		if r.onRateExceeded != nil {
			r.onRateExceeded()
		}
		return false
	}

	// Increment count
	r.requestCount++
	return true
}

// Reset resets the rate limiter's counter
func (r *RateLimiter) Reset() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.requestCount = 0
	r.lastReset = time.Now()
}

// RemainingRequests returns the number of requests remaining in the current interval
func (r *RateLimiter) RemainingRequests() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if time.Since(r.lastReset) > r.resetInterval {
		return r.requestLimit
	}

	if r.requestCount >= r.requestLimit {
		return 0
	}

	return r.requestLimit - r.requestCount
}

// TimeUntilReset returns the duration until the rate limiter resets
func (r *RateLimiter) TimeUntilReset() time.Duration {
	r.mu.RLock()
	defer r.mu.RUnlock()

	elapsed := time.Since(r.lastReset)
	if elapsed >= r.resetInterval {
		return 0
	}

	return r.resetInterval - elapsed
}

// SetRequestLimit changes the request limit
func (r *RateLimiter) SetRequestLimit(limit int) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.requestLimit = limit
}

// SetResetInterval changes the reset interval
func (r *RateLimiter) SetResetInterval(interval time.Duration) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.resetInterval = interval
}

// SetRateExceededCallback changes the callback function called when rate is exceeded
func (r *RateLimiter) SetRateExceededCallback(callback func()) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.onRateExceeded = callback
}

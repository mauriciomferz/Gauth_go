// Package gauth provides rate limiting functionality for the GAuth protocol.
package gauth

import (
	"context"
	"sync"
	"time"
)

// RateLimiter provides rate limiting functionality
type RateLimiter struct {
	mu             sync.RWMutex
	requestCount   int
	lastReset      time.Time
	resetInterval  time.Duration
	requestLimit   int
	onRateExceeded func()
}

// NewRateLimiter creates a new rate limiter with the given configuration
func NewRateLimiter(requestLimit int, resetInterval time.Duration, onRateExceeded func()) *RateLimiter {
	return &RateLimiter{
		requestLimit:   requestLimit,
		resetInterval:  resetInterval,
		lastReset:      time.Now(),
		onRateExceeded: onRateExceeded,
	}
}

// Allow checks if a request should be allowed under the current rate limit
func (r *RateLimiter) Allow(_ context.Context) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if we need to reset the counter
	now := time.Now()
	if now.Sub(r.lastReset) >= r.resetInterval {
		r.requestCount = 0
		r.lastReset = now
	}

	// Check if we're at the limit
	if r.requestCount >= r.requestLimit {
		if r.onRateExceeded != nil {
			r.onRateExceeded()
		}
		return false
	}

	// Increment the counter and allow the request
	r.requestCount++
	return true
}

// Reset resets the rate limiter state
func (r *RateLimiter) Reset() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.requestCount = 0
	r.lastReset = time.Now()
}

// RemainingRequests returns the number of remaining requests allowed
func (r *RateLimiter) RemainingRequests() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Check if we need to reset
	if time.Since(r.lastReset) >= r.resetInterval {
		return r.requestLimit
	}

	remaining := r.requestLimit - r.requestCount
	if remaining < 0 {
		return 0
	}
	return remaining
}

// TimeUntilReset returns the duration until the rate limit resets
func (r *RateLimiter) TimeUntilReset() time.Duration {
	r.mu.RLock()
	defer r.mu.RUnlock()

	nextReset := r.lastReset.Add(r.resetInterval)
	return time.Until(nextReset)
}

// SetRequestLimit updates the request limit
func (r *RateLimiter) SetRequestLimit(limit int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.requestLimit = limit
}

// SetResetInterval updates the reset interval
func (r *RateLimiter) SetResetInterval(interval time.Duration) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.resetInterval = interval
}

// SetRateExceededCallback updates the callback for rate limit exceeded events
func (r *RateLimiter) SetRateExceededCallback(callback func()) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.onRateExceeded = callback
}

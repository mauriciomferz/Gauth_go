// Package ratelimit provides rate limiting functionality for the GAuth protocol.
package ratelimit

import (
	"context"
	"sync"
	"time"
)

// RateLimiterConfig defines the configuration for a rate limiter
type RateLimiterConfig struct {
	RequestLimit int
	ResetInterval time.Duration
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
func (r *RateLimiter) Allow(ctx context.Context) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	if time.Since(r.lastReset) > r.resetInterval {
		r.requestCount = 0
		r.lastReset = time.Now()
	}

       if r.requestCount >= r.requestLimit {
	       if r.onRateExceeded != nil {
		       r.onRateExceeded()
	       }
	       return false
       }
       r.requestCount++
       return true
}

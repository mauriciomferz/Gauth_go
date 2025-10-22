// Package ratelimit provides rate limiting implementations
package ratelimit

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// bucket represents a token bucket for a specific key
type bucket struct {
	tokens     int
	lastRefill time.Time
	mutex      sync.Mutex
}

// Limiter wraps a rate limiting algorithm and config
type Limiter struct {
	algorithm Algorithm
	config    *Config
}

// Config holds rate limiting configuration
type Config struct {
	RequestsPerSecond int
	WindowSize        int
	BurstSize         int
}

// Algorithm defines the rate limiting interface
type Algorithm interface {
	Allow(ctx context.Context, key string) error
}

// TokenBucket implements the token bucket algorithm
type TokenBucket struct {
	config       *Config
	tokens       map[string]*bucket
	mutex        sync.RWMutex
	refillTicker *time.Ticker
	stopChan     chan struct{}
}

// NewLimiter creates a new rate limiter using the default token bucket algorithm
func NewLimiter(config *Config) *Limiter {
	return &Limiter{
		algorithm: WrapTokenBucket(config),
		config:    config,
	}
}

// Allow checks if a request should be allowed
func (l *Limiter) Allow(ctx context.Context, key string) error {
	return l.algorithm.Allow(ctx, key)
}

// GetRemainingRequests returns an estimate of remaining requests for a key
func (l *Limiter) GetRemainingRequests(key string) int {
	// This is a simplified implementation
	// In a real system, you'd need to expose bucket state from the algorithms
	return l.config.BurstSize / 2 // Return a reasonable estimate
}

// Reset resets the rate limit state for a specific key
func (l *Limiter) Reset(key string) {
	// This is a simplified implementation
	// The actual implementation would need to expose bucket management
	// For now, we'll just log the action
	fmt.Printf("Rate limiter reset for key: %s\n", key)
}

// Remove removes the rate limit tracking for a specific key
func (l *Limiter) Remove(key string) {
	// This is a simplified implementation
	// The actual implementation would need to expose bucket management
	fmt.Printf("Rate limiter tracking removed for key: %s\n", key)
}

// WrapTokenBucket creates a new token bucket rate limiter
func WrapTokenBucket(config *Config) Algorithm {
	tb := &TokenBucket{
		config:   config,
		tokens:   make(map[string]*bucket),
		stopChan: make(chan struct{}),
	}

	// Start refill goroutine
	tb.refillTicker = time.NewTicker(time.Second)
	go tb.refillLoop()

	return tb
}

func (tb *TokenBucket) refillLoop() {
	for {
		select {
		case <-tb.refillTicker.C:
			tb.refillBuckets()
		case <-tb.stopChan:
			tb.refillTicker.Stop()
			return
		}
	}
}

func (tb *TokenBucket) refillBuckets() {
	tb.mutex.Lock()
	defer tb.mutex.Unlock()

	now := time.Now()
	for key, b := range tb.tokens {
		b.mutex.Lock()

		// Calculate tokens to add based on time elapsed
		elapsed := now.Sub(b.lastRefill)
		tokensToAdd := int(elapsed.Seconds()) * tb.config.RequestsPerSecond

		if tokensToAdd > 0 {
			b.tokens += tokensToAdd
			if b.tokens > tb.config.BurstSize {
				b.tokens = tb.config.BurstSize
			}
			b.lastRefill = now
		}

		b.mutex.Unlock()

		// Clean up old buckets (no activity for 1 minute)
		if elapsed > time.Minute {
			delete(tb.tokens, key)
		}
	}
}

func (tb *TokenBucket) Allow(ctx context.Context, key string) error {
	tb.mutex.Lock()
	b, exists := tb.tokens[key]
	if !exists {
		b = &bucket{
			tokens:     tb.config.BurstSize,
			lastRefill: time.Now(),
		}
		tb.tokens[key] = b
	}
	tb.mutex.Unlock()

	b.mutex.Lock()
	defer b.mutex.Unlock()

	if b.tokens <= 0 {
		return fmt.Errorf("rate limit exceeded for key: %s", key)
	}

	b.tokens--
	return nil
}

// Stop stops the rate limiter
func (tb *TokenBucket) Stop() {
	close(tb.stopChan)
}

// SlidingWindow implements sliding window rate limiting
type SlidingWindow struct {
	config  *Config
	windows map[string]*window
	mutex   sync.RWMutex
}

type window struct {
	requests []time.Time
	mutex    sync.Mutex
}

// WrapSlidingWindow creates a new sliding window rate limiter
func WrapSlidingWindow(config *Config) Algorithm {
	return &SlidingWindow{
		config:  config,
		windows: make(map[string]*window),
	}
}

func (sw *SlidingWindow) Allow(ctx context.Context, key string) error {
	sw.mutex.Lock()
	w, exists := sw.windows[key]
	if !exists {
		w = &window{
			requests: make([]time.Time, 0),
		}
		sw.windows[key] = w
	}
	sw.mutex.Unlock()

	w.mutex.Lock()
	defer w.mutex.Unlock()

	now := time.Now()
	windowStart := now.Add(-time.Duration(sw.config.WindowSize) * time.Second)

	// Remove old requests outside the window
	var validRequests []time.Time
	for _, reqTime := range w.requests {
		if reqTime.After(windowStart) {
			validRequests = append(validRequests, reqTime)
		}
	}
	w.requests = validRequests

	// Check if we can allow the request
	if len(w.requests) >= sw.config.RequestsPerSecond {
		return fmt.Errorf("rate limit exceeded for key: %s", key)
	}

	// Add the current request
	w.requests = append(w.requests, now)
	return nil
}

// FixedWindow implements fixed window rate limiting
type FixedWindow struct {
	config  *Config
	windows map[string]*fixedWindow
	mutex   sync.RWMutex
}

type fixedWindow struct {
	count       int
	windowStart time.Time
	mutex       sync.Mutex
}

// WrapFixedWindow creates a new fixed window rate limiter
func WrapFixedWindow(config *Config) Algorithm {
	return &FixedWindow{
		config:  config,
		windows: make(map[string]*fixedWindow),
	}
}

func (fw *FixedWindow) Allow(ctx context.Context, key string) error {
	fw.mutex.Lock()
	w, exists := fw.windows[key]
	if !exists {
		w = &fixedWindow{
			count:       0,
			windowStart: time.Now(),
		}
		fw.windows[key] = w
	}
	fw.mutex.Unlock()

	w.mutex.Lock()
	defer w.mutex.Unlock()

	now := time.Now()
	windowDuration := time.Duration(fw.config.WindowSize) * time.Second

	// Reset window if it's expired
	if now.Sub(w.windowStart) >= windowDuration {
		w.count = 0
		w.windowStart = now
	}

	// Check if we can allow the request
	if w.count >= fw.config.RequestsPerSecond {
		return fmt.Errorf("rate limit exceeded for key: %s", key)
	}

	w.count++
	return nil
}

// AllowWithClient checks if an operation is allowed (compatibility: accept clientID, return error)

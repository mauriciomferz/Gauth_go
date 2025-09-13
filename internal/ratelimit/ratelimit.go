// Package ratelimit provides rate limiting functionality for GAuth
package ratelimit

import (
	"context"
	"errors"
	"sync"
	"time"
)

var (
	// ErrRateLimitExceeded is returned when the rate limit is exceeded
	ErrRateLimitExceeded = errors.New("rate limit exceeded")
)

// Config represents rate limiting configuration
type Config struct {
	RequestsPerSecond int `json:"requests_per_second"` // Maximum requests per second
	BurstSize         int `json:"burst_size"`          // Maximum burst size
	WindowSize        int `json:"window_size"`         // Time window in seconds
}

// Window represents a sliding window of requests
type Window struct {
	requests  []time.Time
	config    *Config
	lastClean time.Time
}

// Limiter provides thread-safe rate limiting using sliding windows
type Limiter struct {
	mu      sync.RWMutex
	windows map[string]*Window
	config  *Config
}

// NewLimiter creates a new rate limiter with the given configuration
func NewLimiter(config *Config) *Limiter {
	l := &Limiter{
		windows: make(map[string]*Window),
		config:  config,
	}
	go l.startCleanup()
	return l
}

// Allow checks if a request should be allowed based on the rate limit configuration
func (l *Limiter) Allow(ctx context.Context, id string) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	window, exists := l.windows[id]
	if !exists {
		window = &Window{
			requests:  make([]time.Time, 0),
			config:    l.config,
			lastClean: now,
		}
		l.windows[id] = window
	}

	// Clean old requests from window
	windowStart := now.Add(-time.Duration(l.config.WindowSize) * time.Second)
	validRequests := make([]time.Time, 0)
	for _, t := range window.requests {
		if t.After(windowStart) {
			validRequests = append(validRequests, t)
		}
	}

	// Check if we've exceeded the rate limit
	if len(validRequests) >= l.config.RequestsPerSecond*l.config.WindowSize {
		return ErrRateLimitExceeded
	}

	// Add current request
	validRequests = append(validRequests, now)
	window.requests = validRequests

	return nil
}

// GetRemainingRequests returns the number of remaining requests allowed
func (l *Limiter) GetRemainingRequests(id string) int {
	l.mu.RLock()
	defer l.mu.RUnlock()

	window, exists := l.windows[id]
	if !exists {
		return l.config.RequestsPerSecond * l.config.WindowSize
	}

	now := time.Now()
	windowStart := now.Add(-time.Duration(l.config.WindowSize) * time.Second)
	validCount := 0
	for _, t := range window.requests {
		if t.After(windowStart) {
			validCount++
		}
	}

	maxRequests := l.config.RequestsPerSecond * l.config.WindowSize
	remaining := maxRequests - validCount
	if remaining < 0 {
		remaining = 0
	}
	return remaining
}

// Reset resets the window for a given ID
func (l *Limiter) Reset(id string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if window, exists := l.windows[id]; exists {
		window.requests = make([]time.Time, 0)
		window.lastClean = time.Now()
	}
}

// Remove removes the window for a given ID
func (l *Limiter) Remove(id string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.windows, id)
}

// startCleanup starts a background goroutine to periodically clean up old windows
func (l *Limiter) startCleanup() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		l.cleanup()
	}
}

// cleanup removes old requests and inactive windows
func (l *Limiter) cleanup() {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	threshold := now.Add(-time.Duration(l.config.WindowSize) * time.Second)

	for id, window := range l.windows {
		validRequests := make([]time.Time, 0)
		for _, t := range window.requests {
			if t.After(threshold) {
				validRequests = append(validRequests, t)
			}
		}
		if len(validRequests) == 0 && now.Sub(window.lastClean) > time.Hour {
			delete(l.windows, id)
		} else {
			window.requests = validRequests
			window.lastClean = now
		}
	}
}

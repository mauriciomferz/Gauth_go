package ratelimit

import (
	"context"
	"sync"
	"time"
)

// TokenBucket implements the token bucket algorithm
type TokenBucket struct {
	mu      sync.RWMutex
	buckets map[string]*bucket
	config  *Config
}

// bucket represents a token bucket for a client
type bucket struct {
	tokens   float64
	lastFill time.Time
	config   *Config
}

// NewTokenBucket creates a new token bucket rate limiter
func NewTokenBucket(config *Config) *TokenBucket {
	return &TokenBucket{
		buckets: make(map[string]*bucket),
		config:  config,
	}
}

// Allow implements Algorithm.Allow
func (tb *TokenBucket) Allow(ctx context.Context, id string) error {
	// Check for invalid configuration
	if tb.config.RequestsPerSecond <= 0 || tb.config.WindowSize <= 0 {
		return ErrRateLimitExceeded
	}

	tb.mu.Lock()
	defer tb.mu.Unlock()

	now := time.Now()
	b, exists := tb.buckets[id]
	if !exists {
		maxTokens := float64(tb.config.BurstSize)
		if maxTokens == 0 {
			maxTokens = float64(tb.config.RequestsPerSecond)
		}
		b = &bucket{
			tokens:   maxTokens,
			lastFill: now,
			config:   tb.config,
		}
		tb.buckets[id] = b
	} else {
		// Calculate tokens to add based on time passed
		timePassed := now.Sub(b.lastFill).Seconds()
		tokensToAdd := timePassed * float64(tb.config.RequestsPerSecond)

		// Add tokens up to burst size
		maxTokens := float64(tb.config.BurstSize)
		if maxTokens == 0 {
			maxTokens = float64(tb.config.RequestsPerSecond)
		}
		b.tokens = min(maxTokens, b.tokens+tokensToAdd)
		b.lastFill = now
	}

	// Check if we have enough tokens
	if b.tokens < 1 {
		return ErrRateLimitExceeded
	}

	// Consume one token
	b.tokens--
	return nil
}

// GetQuota implements Algorithm.GetQuota
func (tb *TokenBucket) GetQuota(id string) Quota {
	tb.mu.RLock()
	defer tb.mu.RUnlock()

	now := time.Now()
	windowDuration := time.Duration(tb.config.WindowSize) * time.Second

	b, exists := tb.buckets[id]
	if !exists {
		return Quota{
			Remaining: tb.config.BurstSize,
			Total:     tb.config.BurstSize,
			ResetAt:   now.Add(windowDuration),
			Window: struct {
				Start    time.Time     `json:"start"`
				Duration time.Duration `json:"duration"`
				Requests int           `json:"requests"`
			}{
				Start:    now,
				Duration: windowDuration,
				Requests: 0,
			},
		}
	}

	// Calculate current tokens
	timePassed := now.Sub(b.lastFill).Seconds()
	tokensToAdd := timePassed * float64(tb.config.RequestsPerSecond)
	currentTokens := min(float64(tb.config.BurstSize), b.tokens+tokensToAdd)

	// Time until tokens are fully replenished
	timeToReset := time.Duration(
		(float64(tb.config.BurstSize) - currentTokens) /
			float64(tb.config.RequestsPerSecond) * float64(time.Second))
	if timeToReset < 0 {
		timeToReset = 0
	}

	return Quota{
		Remaining: int(currentTokens),
		Total:     tb.config.BurstSize,
		ResetAt:   now.Add(timeToReset),
		Window: struct {
			Start    time.Time     `json:"start"`
			Duration time.Duration `json:"duration"`
			Requests int           `json:"requests"`
		}{
			Start:    b.lastFill,
			Duration: windowDuration,
			Requests: tb.config.BurstSize - int(currentTokens),
		},
	}
}

// Reset implements Algorithm.Reset
func (tb *TokenBucket) Reset(id string) {
	tb.mu.Lock()
	defer tb.mu.Unlock()
	delete(tb.buckets, id)
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

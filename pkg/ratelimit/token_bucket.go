package ratelimit

import (
	"context"
	"sync"
	"time"
)

type TokenBucket struct {
	mu      sync.RWMutex
	buckets map[string]*bucket
	config  *Config
}

type bucket struct {
	tokens   float64
	lastFill time.Time
	config   *Config
}

func NewTokenBucket(config *Config) *TokenBucket {
	return &TokenBucket{
		buckets: make(map[string]*bucket),
		config:  config,
	}
}

func (tb *TokenBucket) Allow(ctx context.Context, id string) error {
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
		timePassed := now.Sub(b.lastFill).Seconds()
		tokensToAdd := timePassed * float64(tb.config.RequestsPerSecond)
		maxTokens := float64(tb.config.BurstSize)
		if maxTokens == 0 {
			maxTokens = float64(tb.config.RequestsPerSecond)
		}
		b.tokens = min(maxTokens, b.tokens+tokensToAdd)
		b.lastFill = now
	}
	if b.tokens < 1 {
		return ErrRateLimitExceeded
	}
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
	timePassed := now.Sub(b.lastFill).Seconds()
	tokensToAdd := timePassed * float64(tb.config.RequestsPerSecond)
	currentTokens := min(float64(tb.config.BurstSize), b.tokens+tokensToAdd)
	timeToReset := time.Duration((float64(tb.config.BurstSize)-currentTokens)/float64(tb.config.RequestsPerSecond)*float64(time.Second))
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

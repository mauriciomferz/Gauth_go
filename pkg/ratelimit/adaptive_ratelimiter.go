package ratelimit

import (
	"sync"
	"time"
)

type AdaptiveConfig struct {
	InitialLimit int
	MaxLimit int
	MinLimit int
	Window time.Duration
	ScaleUpFactor float64
	ScaleDownFactor float64
}

type AdaptiveRateLimiter struct {
	mu           sync.RWMutex
	currentLimit int
	config       AdaptiveConfig
	requestCount int
	lastReset    time.Time
	usageHistory []float64
}

func NewAdaptiveRateLimiter(config AdaptiveConfig) *AdaptiveRateLimiter {
	if config.ScaleUpFactor == 0 {
		config.ScaleUpFactor = 1.1
	}
	if config.ScaleDownFactor == 0 {
		config.ScaleDownFactor = 0.9
	}
	return &AdaptiveRateLimiter{
		currentLimit: config.InitialLimit,
		config:       config,
		lastReset:    time.Now(),
		usageHistory: make([]float64, 0, 10),
	}
}

func (a *AdaptiveRateLimiter) Allow() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	// ...existing code...
	return true
}

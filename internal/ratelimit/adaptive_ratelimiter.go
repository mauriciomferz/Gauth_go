package ratelimit

import (
	"sync"
	"time"
)

// AdaptiveConfig defines configuration for an adaptive rate limiter
type AdaptiveConfig struct {
	// Initial limit of requests per window
	InitialLimit int

	// Maximum limit of requests per window
	MaxLimit int

	// Minimum limit of requests per window
	MinLimit int

	// Time window for rate limiting
	Window time.Duration

	// ScaleUpFactor determines how quickly to increase limits when capacity available
	ScaleUpFactor float64

	// ScaleDownFactor determines how quickly to decrease limits when capacity limited
	ScaleDownFactor float64
}

// AdaptiveRateLimiter implements a rate limiter that can adapt to system load
type AdaptiveRateLimiter struct {
	mu           sync.RWMutex
	currentLimit int
	config       AdaptiveConfig
	requestCount int
	lastReset    time.Time
	usageHistory []float64 // Tracks recent usage for adaptive adjustments
}

// NewAdaptiveRateLimiter creates a new adaptive rate limiter
func NewAdaptiveRateLimiter(config AdaptiveConfig) *AdaptiveRateLimiter {
	// Set default values if not specified
	if config.ScaleUpFactor == 0 {
		config.ScaleUpFactor = 1.1 // Increase by 10% when underutilized
	}

	if config.ScaleDownFactor == 0 {
		config.ScaleDownFactor = 0.9 // Decrease by 10% when overutilized
	}

	return &AdaptiveRateLimiter{
		currentLimit: config.InitialLimit,
		config:       config,
		lastReset:    time.Now(),
		usageHistory: make([]float64, 0, 10),
	}
}

// Allow checks if a request is allowed and records usage statistics
func (a *AdaptiveRateLimiter) Allow() bool {
	a.mu.Lock()
	defer a.mu.Unlock()

	now := time.Now()

	// Check if window has passed and we need to adjust limits
	if now.Sub(a.lastReset) > a.config.Window {
		a.adjustLimits()
		a.requestCount = 0
		a.lastReset = now
	}

	// Check if we're still within limits
	if a.requestCount >= a.currentLimit {
		return false
	}

	a.requestCount++
	return true
}

// adjustLimits modifies rate limits based on recent usage patterns
func (a *AdaptiveRateLimiter) adjustLimits() {
	// Calculate usage ratio for the just-completed window
	usageRatio := float64(a.requestCount) / float64(a.currentLimit)

	// Store usage history (up to 10 points)
	a.usageHistory = append(a.usageHistory, usageRatio)
	if len(a.usageHistory) > 10 {
		a.usageHistory = a.usageHistory[1:]
	}

	// Calculate average usage
	var sum float64
	for _, usage := range a.usageHistory {
		sum += usage
	}
	avgUsage := sum / float64(len(a.usageHistory))

	// Adjust limits based on usage
	if avgUsage > 0.8 {
		// High usage - scale down
		newLimit := int(float64(a.currentLimit) * a.config.ScaleDownFactor)
		if newLimit >= a.config.MinLimit {
			a.currentLimit = newLimit
		} else {
			a.currentLimit = a.config.MinLimit
		}
	} else if avgUsage < 0.5 {
		// Low usage - scale up
		newLimit := int(float64(a.currentLimit) * a.config.ScaleUpFactor)
		if newLimit <= a.config.MaxLimit {
			a.currentLimit = newLimit
		} else {
			a.currentLimit = a.config.MaxLimit
		}
	}
}

// GetCurrentLimit returns the current request limit
func (a *AdaptiveRateLimiter) GetCurrentLimit() int {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.currentLimit
}

// GetUsage returns the current usage ratio
func (a *AdaptiveRateLimiter) GetUsage() float64 {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if a.currentLimit == 0 {
		return 0
	}

	return float64(a.requestCount) / float64(a.currentLimit)
}

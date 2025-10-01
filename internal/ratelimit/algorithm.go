// Package ratelimit provides interfaces for rate limiting algorithms.
package ratelimit

import (
	"context"
	"time"
)

// Algorithm defines the interface for rate limiting algorithms
type Algorithm interface {
	// Allow checks if a request should be allowed
	Allow(ctx context.Context, id string) error

	// GetQuota returns the current quota status
	GetQuota(id string) Quota

	// Reset resets the rate limiter for an ID
	Reset(id string)
}

// Quota represents rate limit quota information
type Quota struct {
	// Remaining is the number of requests remaining
	Remaining int `json:"remaining"`

	// Total is the total requests allowed
	Total int `json:"total"`

	// ResetAt is when the quota resets
	ResetAt time.Time `json:"reset_at"`

	// Window is the current time window
	Window struct {
		// Start is the window start time
		Start time.Time `json:"start"`

		// Duration is the window duration
		Duration time.Duration `json:"duration"`

		// Requests is requests in window
		Requests int `json:"requests"`
	} `json:"window"`
}

// Strategy represents a rate limiting strategy
type Strategy string

const (
	// StrategyFixedWindow uses fixed time windows
	StrategyFixedWindow Strategy = "fixed-window"
	// StrategySlidingWindow uses sliding time windows
	StrategySlidingWindow Strategy = "sliding-window"
	// StrategyLeakyBucket uses the leaky bucket algorithm
	StrategyLeakyBucket Strategy = "leaky-bucket"
	// StrategyTokenBucket uses the token bucket algorithm
	StrategyTokenBucket Strategy = "token-bucket"
)

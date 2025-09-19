// Package ratelimit provides interfaces for rate limiting algorithms.
package ratelimit

import (
	"context"
	"time"
)

// Algorithm defines the interface for rate limiting algorithms
type Algorithm interface {
	Allow(ctx context.Context, id string) error
	GetQuota(id string) Quota
	Reset(id string)
}

// Quota represents rate limit quota information
type Quota struct {
	Remaining int `json:"remaining"`
	Total int `json:"total"`
	ResetAt time.Time `json:"reset_at"`
	Window struct {
		Start time.Time `json:"start"`
		Duration time.Duration `json:"duration"`
		Requests int `json:"requests"`
	} `json:"window"`
}

// Strategy represents a rate limiting strategy
type Strategy string

const (
	StrategyFixedWindow Strategy = "fixed-window"
	StrategySlidingWindow Strategy = "sliding-window"
	StrategyLeakyBucket Strategy = "leaky-bucket"
	StrategyTokenBucket Strategy = "token-bucket"
)

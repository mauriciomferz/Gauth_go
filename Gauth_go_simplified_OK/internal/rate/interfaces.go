package rate

import (
	"context"
	"time"
)

// Algorithm defines the interface for rate limiting algorithms.
type Algorithm interface {
	// Allow checks if a request should be allowed.
	// Returns nil if allowed, ErrRateLimitExceeded if not.
	Allow(ctx context.Context, id string) error

	// GetRemainingQuota returns the number of requests remaining
	// in the current window.
	GetRemainingQuota(id string) int

	// Reset clears all tracking data for the given ID.
	Reset(id string)
}

// Store defines the interface for rate limit data storage.
type Store interface {
	// Add adds a request to the store.
	// Returns ErrRateLimitExceeded if the addition would exceed limits.
	Add(ctx context.Context, id string, t time.Time) error

	// GetWindow returns all requests in the window for the given ID.
	GetWindow(id string, start time.Time) []time.Time

	// Reset clears all data for the given ID.
	Reset(id string)

	// Remove completely removes all data for the given ID.
	Remove(id string)

	// Cleanup removes expired data across all IDs.
	Cleanup(threshold time.Time)
}

// Config represents rate limiting configuration.
// It provides settings that control the rate limiting behavior.
type Config struct {
	// RequestsPerSecond is the maximum number of requests allowed per second
	// when averaged over the window.
	RequestsPerSecond int `json:"requests_per_second"`

	// BurstSize is the maximum number of requests allowed in a single burst.
	// This can temporarily exceed RequestsPerSecond but will be constrained
	// by the WindowSize.
	BurstSize int `json:"burst_size"`

	// WindowSize is the duration in seconds over which the rate is calculated.
	// A longer window allows more flexibility in request distribution while
	// still maintaining the average rate.
	WindowSize int `json:"window_size"`
}

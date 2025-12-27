package pdp

import (
	"context"
)

// DecisionCache defines the interface for caching PDP decisions.
// Abstraction allows switching between in-memory, Redis, or other distributed backends.
type DecisionCache interface {
	// Get retrieves a cached decision for a given request.
	// Returns decision, true if found, and error if any backend failure occurred.
	Get(ctx context.Context, req Request) (Decision, bool, error)

	// Set stores a decision in the cache.
	Set(ctx context.Context, req Request, decision Decision) error

	// InvalidateAll clears the entire cache or increments the global version.
	InvalidateAll(ctx context.Context) error

	// InvalidateSubject removes all cached decisions for a specific subject.
	InvalidateSubject(ctx context.Context, subject string) error

	// InvalidateResource removes all cached decisions for a specific resource.
	InvalidateResource(ctx context.Context, resource string) error

	// InvalidateAction removes all cached decisions for a specific action.
	InvalidateAction(ctx context.Context, action string) error

	// GetMetrics returns cache performance statistics.
	GetMetrics() PDPCacheMetrics

	// Close cleans up resources (e.g., stopping background routines or connections).
	Close() error
}

// PDPCacheMetrics defines common metrics for cache implementations.
type PDPCacheMetrics struct {
	Capacity      int     `json:"capacity"`
	Size          int     `json:"size"`
	Lookups       uint64  `json:"lookups"`
	Hits          uint64  `json:"hits"`
	Misses        uint64  `json:"misses"`
	HitRate       float64 `json:"hit_rate"`
	Evictions     uint64  `json:"evictions"`
	Expirations   uint64  `json:"expirations"`
	Invalidations uint64  `json:"invalidations"`
	TTL           string  `json:"ttl"`
	Backend       string  `json:"backend"` // "memory", "redis", etc.
}

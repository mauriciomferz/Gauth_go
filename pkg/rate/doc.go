/*
Package rate provides comprehensive rate limiting functionality for controlling access to resources.

The package implements several rate limiting algorithms designed for different use cases:

  - Token Bucket: For smooth rate limiting with configurable burst capacity
  - Sliding Window: For accurate rate limiting over precise time periods
  - Fixed Window: For simpler implementation with periodic counter resets
  - Leaky Bucket: For controlled rate of processing with overflow protection
  - Distributed Rate Limiting: For coordinated limits across multiple instances

Core Types:

Limiter is the main interface for rate limiting:

	type Limiter interface {
	    Allow(ctx context.Context, id string) error
	    GetRemainingRequests(id string) int64
	    Reset(id string)
	}

Algorithms:

TokenBucket implements the token bucket algorithm:

	bucket := rate.NewTokenBucket(rate.Config{
	    Rate:      100,  // tokens per second
	    BurstSize: 10,   // maximum burst size
	})

SlidingWindow implements a sliding window counter:
*/
package rate

//
//	window := rate.NewSlidingWindow(rate.Config{
//	    Requests: 1000,  // requests per window
//	    Window:   time.Minute,
//	})
//
// # Usage Examples
//
// Basic usage:
//
//	limiter := rate.NewTokenBucket(rate.Config{
//	    Rate:      10,
//	    BurstSize: 5,
//	})
//
//	err := limiter.Allow(ctx, "user-123")
//	if err != nil {
//	    // Rate limit exceeded
//	}
//
// With middleware:
//
//	http.Handle("/api", rate.Middleware(limiter)(handler))
//
// # Distributed Rate Limiting
//
// The package supports distributed rate limiting through Redis:
//
//	limiter := rate.NewDistributedLimiter(rate.RedisConfig{
//	    Addresses: []string{"localhost:6379"},
//	    Password:  "secret",
//	})
//
// # Error Types
//
// The package defines specific error types:
//
//	var ErrRateLimitExceeded = errors.New("rate limit exceeded")
//	var ErrInvalidConfig = errors.New("invalid rate limiter configuration")
//
// # Monitoring
//
// Built-in metrics for monitoring rate limiting:
//
// - Request counts
// - Rejection counts
// - Response latency
// - Remaining quota
//
// Example with Prometheus:
//
//	metrics := rate.NewPrometheusMetrics()
//	limiter := rate.NewTokenBucket(rate.Config{
//	    Rate:      100,
//	    BurstSize: 10,
//	    Metrics:   metrics,
//	})
//
// For more examples, see the examples/rate directory.

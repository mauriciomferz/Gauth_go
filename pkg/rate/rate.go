/*
Package rate provides rate limiting functionality with support for multiple algorithms and storage backends.

Key features:
  - Multiple rate limiting algorithms
  - Distributed rate limiting support
  - Custom storage backends
  - Dynamic limit adjustments
  - Quota management
  - Request throttling
  - Burst handling

The package supports several rate limiting strategies:
  - Token Bucket
  - Sliding Window
  - Fixed Window
  - Leaky Bucket
  - Dynamic Rate Limiting

Example usage:

	import "github.com/Gimel-Foundation/gauth/pkg/rate"

	// Create a rate limiter
	limiter := rate.NewTokenBucket(rate.Config{
			Tokens:    100,    // Max tokens
			Interval:  time.Minute,  // Refill interval
			BurstSize: 10,     // Max burst
	})

	// Check rate limit
	quota, err := limiter.Allow(ctx, "client1")
	if err == rate.ErrLimitExceeded {
			// Handle rate limit exceeded
	}

See examples/rate/ for more examples.
*/
package rate

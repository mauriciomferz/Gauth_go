// Package rate provides rate limiting functionality for GAuth.
//
// The package implements a sliding window rate limiter that offers:
//   - Precise request counting over configurable time windows
//   - Thread-safe operations for concurrent use
//   - Per-client rate limit tracking
//   - Automatic cleanup of stale data
//
// Basic Usage:
//
//	import "github.com/Gimel-Foundation/gauth/internal/rate"
//
//	// Create a rate limiter with configuration
//	limiter := rate.NewLimiter(&rate.Config{
//	    RequestsPerSecond: 100,
//	    WindowSize:       60, // 60-second window
//	})
//
//	// Check if a request is allowed
//	if err := limiter.Allow(ctx, "client-123"); err != nil {
//	    // Handle rate limit exceeded
//	}
//
// Thread Safety:
//
// All operations are thread-safe and can be called from multiple goroutines.
// The limiter uses fine-grained locking to minimize contention.
//
// Memory Management:
//
// The limiter automatically cleans up stale data to prevent memory leaks:
//   - Expired requests are removed from tracking windows
//   - Inactive clients are removed after extended periods
//   - Cleanup runs periodically in the background
//
// Extension Points:
//
// The package is designed to be extensible:
//   - Implement custom storage backends by implementing the Store interface
//   - Add new rate limiting algorithms by implementing the Algorithm interface
//   - Override cleanup behavior through the Cleaner interface
//
// See the examples directory for detailed usage examples.
package rate

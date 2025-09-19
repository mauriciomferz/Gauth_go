// Package ratelimit provides various rate limiting implementations for the GAuth protocol.
//
// This package offers several rate limiting strategies:
//
// 1. Basic Rate Limiting - Simple token bucket style rate limiter for a single source
//   - Thread-safe operation
//   - Configurable limits and intervals
//   - Customizable behavior on limit exceeded
//
// 2. Client Rate Limiting - Per-client rate limiting with automatic cleanup
//   - Support for multiple clients with individual limits
//   - Automatic cleanup of expired entries
//   - Configurable time windows per client
//
// 3. Adaptive Rate Limiting - Adjusts limits based on observed usage patterns
//   - Dynamically scales limits up and down
//   - Configurable scaling factors and thresholds
//   - Historical usage tracking for smarter adjustments
//
// Usage Examples:
//
// Basic Rate Limiter:
//
//	limiter := ratelimit.NewRateLimiter(ratelimit.RateLimiterConfig{
//	    RequestLimit: 100,
//	    ResetInterval: time.Minute,
//	    OnRateExceeded: func() { log.Println("Rate limit exceeded") },
//	})
//
//	if limiter.Allow(ctx) {
//	    // Process the request
//	} else {
//	    // Return rate limit error
//	}
//
// Client Rate Limiter:
//
//	clientLimiter := ratelimit.NewClientRateLimiter(time.Minute, 100)
//
//	clientID := "user-123"
package ratelimit

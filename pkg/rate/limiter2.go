package rate

import (
	"context"
	"time"
)

// Algorithm represents a rate limiting algorithm
type Algorithm interface {
	// Allow checks if a request should be allowed
	Allow(ctx context.Context) (bool, time.Duration)

	// Reset resets the rate limiter state
	Reset()
}

package integration

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/Gimel-Foundation/gauth/pkg/rate"
)

func TestRateLimiterIntegration(t *testing.T) {
	configs := []struct {
		name  string
		setup func(cfg rate.Config) rate.Limiter
		cfg   rate.Config
	}{
		{
			name: "TokenBucket",
			setup: func(cfg rate.Config) rate.Limiter {
				return rate.NewTokenBucket(cfg)
			},
			cfg: rate.Config{
				Rate:      10,
				Window:    time.Second,
				BurstSize: 3,
			},
		},
		{
			name: "SlidingWindow",
			setup: func(cfg rate.Config) rate.Limiter {
				return rate.NewSlidingWindow(cfg)
			},
			cfg: rate.Config{
				Rate:   10,
				Window: time.Second,
			},
		},
	}

	for _, tc := range configs {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			limiter := tc.setup(tc.cfg)

			key := "test-client"

			// Should allow up to burst size for token bucket, or rate for sliding window
			maxRequests := int(tc.cfg.Rate)
			if tc.cfg.BurstSize > 0 {
				maxRequests = int(tc.cfg.BurstSize)
			}
			
			for i := 0; i < maxRequests; i++ {
				err := limiter.Allow(ctx, key)
				assert.NoError(t, err)
				remaining := limiter.GetRemainingRequests(key)
				assert.GreaterOrEqual(t, remaining, int64(0))
			}

			// Should deny after rate
			err := limiter.Allow(ctx, key)
			assert.ErrorIs(t, err, rate.ErrRateLimitExceeded)
		})
	}
}

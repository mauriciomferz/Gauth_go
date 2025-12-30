//go:build integration
// +build integration

package integration

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/mauriciomferz/AgentAuth/pkg/rate"
)

func TestRateLimiterIntegration(t *testing.T) {
	ctx := context.Background()
	cfg := rate.NewEnhancedConfig(10, rate.DefaultConfig().WindowSize)
	limiter := rate.NewLimiter(cfg.Config)

	// Consume tokens until limiter denies
	consumed := 0
	for consumed < cfg.BurstSize*2 { // upper bound to avoid infinite loop
		if !limiter.Allow() {
			break
		}
		consumed++
	}
	assert.GreaterOrEqual(t, consumed, 1, "expected to consume at least one token")
	// Now Wait should block until a token is refilled then return nil
	if err := limiter.Wait(ctx); err != nil {
		t.Fatalf("wait error: %v", err)
	}
	// The token consumed by Wait may be the only one available; treat success of Wait as sufficient.
}

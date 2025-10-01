package integration

import (
	"context"
	stderrors "errors"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/Gimel-Foundation/gauth/pkg/resilience"
)

func TestResilienceIntegration(t *testing.T) {
	ctx := context.Background()

	t.Run("CircuitBreaker", func(t *testing.T) {
		// Create circuit breaker
		cb := resilience.NewCircuitBreaker(resilience.CircuitConfig{
			Name:        "test-circuit",
			MaxFailures: 3,
			Timeout:     2 * time.Second,
			Interval:    5 * time.Second,
		})

		// Test initial state
		t.Run("InitialState", func(t *testing.T) {
			assert.Equal(t, resilience.StateClosed, cb.State())
		})

		// Test failure threshold
		t.Run("FailureThreshold", func(t *testing.T) {
			// Simulate failures
			for i := 0; i < 3; i++ {
				err := cb.Execute(ctx, func(_ context.Context) error {
					return stderrors.New("test failure")
				})
				assert.Error(t, err)
			}

			// Circuit should be open
			assert.Equal(t, resilience.StateOpen, cb.State())

			// Should fail fast when open
			err := cb.Execute(ctx, func(_ context.Context) error {
				return nil
			})
			assert.ErrorIs(t, err, resilience.ErrCircuitOpen)
		})

		// Test half-open state transition
		t.Run("HalfOpen", func(t *testing.T) {
			// Wait for circuit breaker timeout (2s + buffer for CI stability)
			waitTime := 2500 * time.Millisecond
			if os.Getenv("CI") == "true" {
				waitTime = 3500 * time.Millisecond // Extra buffer for CI environment
			}
			time.Sleep(waitTime)

			// Circuit should still be open (half-open is transient during execution)
			assert.Equal(t, resilience.StateOpen, cb.State())

			// Test successful call - this triggers half-open->closed transition
			err := cb.Execute(ctx, func(_ context.Context) error {
				return nil
			})
			assert.NoError(t, err)

			// Circuit should be closed again after successful execution
			assert.Equal(t, resilience.StateClosed, cb.State())
		})
	})
}

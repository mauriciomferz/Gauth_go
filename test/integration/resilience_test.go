package integration

import (
	"context"
	stderrors "errors"
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
				err := cb.Execute(ctx, func(ctx context.Context) error {
					return stderrors.New("test failure")
				})
				assert.Error(t, err)
			}

			// Circuit should be open
			assert.Equal(t, resilience.StateOpen, cb.State())

			// Should fail fast when open
			err := cb.Execute(ctx, func(ctx context.Context) error {
				return nil
			})
			assert.ErrorIs(t, err, resilience.ErrCircuitOpen)
		})

		// Test half-open state
		t.Run("HalfOpen", func(t *testing.T) {
			// Wait for timeout
			time.Sleep(2500 * time.Millisecond)

			// Circuit should be half-open
			assert.Equal(t, resilience.StateHalfOpen, cb.State())

			// Test successful call
			err := cb.Execute(ctx, func(ctx context.Context) error {
				return nil
			})
			assert.NoError(t, err)

			// Circuit should be closed again
			assert.Equal(t, resilience.StateClosed, cb.State())
		})
	})
}

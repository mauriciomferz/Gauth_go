package integration

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Gimel-Foundation/gauth/internal/circuit"
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
			assert.Equal(t, circuit.StateClosed, cb.State())
		})

		// Test failure threshold
		t.Run("FailureThreshold", func(t *testing.T) {
			// Simulate failures
			for i := 0; i < 3; i++ {
				err := cb.Execute(ctx, func(ctx context.Context) error {
					return errors.New("test failure")
				})
				assert.Error(t, err)
			}

			// Circuit should be open
			assert.Equal(t, circuit.StateOpen, cb.State())

			// Should fail fast when open
			err := cb.Execute(ctx, func(ctx context.Context) error {
				return nil
			})
			assert.ErrorIs(t, err, circuit.ErrCircuitOpen)
		})

		// Test half-open state
		t.Run("HalfOpen", func(t *testing.T) {
			// Wait for timeout
			time.Sleep(2 * time.Second)

			// Circuit should be half-open
			assert.Equal(t, circuit.StateHalfOpen, cb.State())

			// Successful execution should close circuit
			err := cb.Execute(ctx, func(ctx context.Context) error {
				return nil
			})
			require.NoError(t, err)
			assert.Equal(t, circuit.StateClosed, cb.State())
		})
	})

	t.Run("RetryWithBackoff", func(t *testing.T) {
		// Create retry handler
		retry := resilience.NewRetry(resilience.RetryConfig{
			MaxAttempts:  3,
			InitialDelay: 100 * time.Millisecond,
			MaxDelay:     1 * time.Second,
			Multiplier:   2.0,
		})

		// Test successful retry
		t.Run("SuccessfulRetry", func(t *testing.T) {
			attempts := 0
			err := retry.Execute(ctx, func(ctx context.Context) error {
				attempts++
				if attempts < 2 {
					return errors.New("temporary error")
				}
				return nil
			})

			require.NoError(t, err)
			assert.Equal(t, 2, attempts)
		})

		// Test max attempts exceeded
		t.Run("MaxAttemptsExceeded", func(t *testing.T) {
			attempts := 0
			err := retry.Execute(ctx, func(ctx context.Context) error {
				attempts++
				return errors.New("persistent error")
			})

			require.Error(t, err)
			assert.Equal(t, 3, attempts)
		})
	})

	t.Run("Timeout", func(t *testing.T) {
		// Create timeout handler
		timeout := resilience.NewTimeout(resilience.TimeoutConfig{
			Timeout: 100 * time.Millisecond,
		})

		// Test successful execution
		t.Run("SuccessfulExecution", func(t *testing.T) {
			err := timeout.Execute(ctx, func(ctx context.Context) error {
				return nil
			})
			require.NoError(t, err)
		})

		// Test timeout
		t.Run("TimeoutExceeded", func(t *testing.T) {
			err := timeout.Execute(ctx, func(ctx context.Context) error {
				time.Sleep(200 * time.Millisecond)
				return nil
			})
			assert.ErrorIs(t, err, context.DeadlineExceeded)
		})
	})

	t.Run("Bulkhead", func(t *testing.T) {
		// Create bulkhead
		bulkhead := resilience.NewBulkhead(resilience.BulkheadConfig{
			MaxConcurrent: 2,
			MaxWaitTime:   100 * time.Millisecond,
		})

		// Test concurrent execution
		t.Run("ConcurrentExecution", func(t *testing.T) {
			running := make(chan struct{}, 2)
			complete := make(chan error, 3)

			// Launch three concurrent executions
			for i := 0; i < 3; i++ {
				go func() {
					err := bulkhead.Execute(ctx, func(ctx context.Context) error {
						running <- struct{}{}
						time.Sleep(200 * time.Millisecond)
						<-running
						return nil
					})
					complete <- err
				}()
			}

			// Collect results
			var errors []error
			for i := 0; i < 3; i++ {
				errors = append(errors, <-complete)
			}

			// Should have one rejection and two successes
			successes := 0
			rejections := 0
			for _, err := range errors {
				if err == nil {
					successes++
				} else if errors.Is(err, resilience.ErrBulkheadFull) {
					rejections++
				}
			}

			assert.Equal(t, 2, successes)
			assert.Equal(t, 1, rejections)
		})
	})

	t.Run("Combined", func(t *testing.T) {
		// Create combined resilience pattern
		pattern := resilience.Combine(
			resilience.NewCircuitBreaker(resilience.CircuitConfig{
				MaxFailures: 2,
				Timeout:     1 * time.Second,
			}),
			resilience.NewRetry(resilience.RetryConfig{
				MaxAttempts:  2,
				InitialDelay: 50 * time.Millisecond,
			}),
			resilience.NewTimeout(resilience.TimeoutConfig{
				Timeout: 200 * time.Millisecond,
			}),
		)

		// Test successful execution
		t.Run("SuccessfulExecution", func(t *testing.T) {
			err := pattern.Execute(ctx, func(ctx context.Context) error {
				return nil
			})
			require.NoError(t, err)
		})

		// Test circuit breaker triggering
		t.Run("CircuitBreakerTriggering", func(t *testing.T) {
			// Force circuit breaker to open
			for i := 0; i < 3; i++ {
				_ = pattern.Execute(ctx, func(ctx context.Context) error {
					return errors.New("failure")
				})
			}

			// Should fail fast
			start := time.Now()
			err := pattern.Execute(ctx, func(ctx context.Context) error {
				return nil
			})
			assert.Error(t, err)
			assert.Less(t, time.Since(start), 50*time.Millisecond)
		})
	})
}

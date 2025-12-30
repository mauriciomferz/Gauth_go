//go:build integration
// +build integration

package integration

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/mauriciomferz/AgentAuth/internal/circuit"
	assert "github.com/stretchr/testify/assert"
)

// Circuit breaker constants using actual implementation
var (
	StateClosed = circuit.StateClosed
	StateOpen   = circuit.StateOpen
)

var ErrCircuitOpen = errors.New("circuit breaker is open")

func NewCircuitBreaker(cfg struct {
	Name        string
	MaxFailures int
	Timeout     time.Duration
	Interval    time.Duration
},
) *circuit.CircuitBreaker {
	return circuit.NewBreaker(circuit.Options{
		Name:             cfg.Name,
		FailureThreshold: cfg.MaxFailures,
		ResetTimeout:     cfg.Timeout,
	})
}

// Test cases for the circuit breaker
func TestCircuitBreaker(t *testing.T) {
	cb := NewCircuitBreaker(struct {
		Name        string
		MaxFailures int
		Timeout     time.Duration
		Interval    time.Duration
	}{
		Name:        "TestCircuitBreaker",
		MaxFailures: 3,
		Timeout:     2 * time.Second,
		Interval:    1 * time.Second,
	})

	// Circuit should be closed initially
	assert.Equal(t, StateClosed, cb.GetState())

	// Simulate failures to open the circuit
	for i := 0; i < 3; i++ {
		err := cb.Execute(context.Background(), func() error {
			return errors.New("failure")
		})
		// We expect errors here to trigger circuit opening
		_ = err
	}

	// Circuit should be open
	assert.Equal(t, StateOpen, cb.GetState())

	// Should fail fast when open
	err := cb.Execute(context.Background(), func() error {
		return nil
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "circuit breaker open")

	// Test half-open state transition
	t.Run("HalfOpen", func(t *testing.T) {
		// Wait for circuit breaker timeout (2s + buffer for CI stability)
		waitTime := 2500 * time.Millisecond
		if os.Getenv("CI") == "true" {
			waitTime = 3500 * time.Millisecond // Extra buffer for CI environment
		}
		time.Sleep(waitTime)

		// Circuit should still be open (half-open is transient during execution)
		assert.Equal(t, StateOpen, cb.GetState())

		// Test successful call - this triggers half-open->closed transition
		err := cb.Execute(context.Background(), func() error {
			return nil
		})
		assert.NoError(t, err)

		// Circuit should be closed again after successful execution
		assert.Equal(t, StateClosed, cb.GetState())
	})
}

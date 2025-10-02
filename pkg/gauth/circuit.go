// Package gauth provides circuit breaker functionality for resilient services.
// NOTE: This is a stub implementation for educational purposes.
// Real circuit breaker functionality is provided by pkg/resilience

package gauth

import (
	"context"
	"time"

	"github.com/Gimel-Foundation/gauth/pkg/resilience"
)

// CircuitBreakerState is an alias for resilience.CircuitState
type CircuitBreakerState = resilience.CircuitState

// State constants
const (
	CircuitClosed   = resilience.StateClosed
	CircuitOpen     = resilience.StateOpen
	CircuitHalfOpen = resilience.StateHalfOpen
)

// CircuitBreakerMetrics provides stub metrics
type CircuitBreakerMetrics struct {
	// Core metrics (stubbed)
	Requests     uint64              `json:"requests"`
	Failures     uint64              `json:"failures"`
	FailureRate  float64             `json:"failure_rate"`
	CurrentState CircuitBreakerState `json:"current_state"`

	// State information (stubbed)
	Name            string    `json:"name"`
	LastFailure     time.Time `json:"last_failure"`
	LastSuccess     time.Time `json:"last_success"`
	LastStateChange time.Time `json:"last_state_change"`
}

// CircuitBreakerConfig provides stub configuration
type CircuitBreakerConfig struct {
	Name             string        `json:"name"`
	FailureRatio     float64       `json:"failure_ratio"`
	RequestCount     uint64        `json:"request_count"`
	SuccessThreshold uint64        `json:"success_threshold"`
	ResetTimeout     time.Duration `json:"reset_timeout"`
}

// CircuitBreaker is a stub that always succeeds
type CircuitBreaker struct {
	name   string
	config CircuitBreakerConfig
}

// NewCircuitBreaker creates a stub circuit breaker
func NewCircuitBreaker(config CircuitBreakerConfig) *CircuitBreaker {
	return &CircuitBreaker{
		name:   config.Name,
		config: config,
	}
}

// Call executes function without protection (stub)
func (cb *CircuitBreaker) Call(ctx context.Context, fn func() (interface{}, error)) (interface{}, error) {
	return fn()
}

// Metrics returns fake metrics
func (cb *CircuitBreaker) Metrics() CircuitBreakerMetrics {
	return CircuitBreakerMetrics{
		Requests:     0,
		Failures:     0,
		FailureRate:  0.0,
		CurrentState: CircuitClosed,
		Name:         cb.name,
		LastFailure:  time.Time{},
		LastSuccess:  time.Now(),
	}
}

// State always returns closed
func (cb *CircuitBreaker) State() CircuitBreakerState {
	return CircuitClosed
}

// Reset does nothing (stub)
func (cb *CircuitBreaker) Reset() {
	// no-op
}

// Name returns the circuit name
func (cb *CircuitBreaker) Name() string {
	return cb.name
}

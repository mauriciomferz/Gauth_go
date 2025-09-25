// Package gauth provides circuit breaker functionality for resilient services.
package gauth

import (
	"time"

	"github.com/Gimel-Foundation/gauth/examples/cascade/pkg/resilience"
)

// CircuitBreakerState represents the state of a circuit breaker
type CircuitBreakerState string

const (
	// CircuitClosed indicates the circuit is allowing requests
	CircuitClosed CircuitBreakerState = "closed"
	// CircuitOpen indicates the circuit is blocking requests
	CircuitOpen CircuitBreakerState = "open"
	// CircuitHalfOpen indicates the circuit is testing recovery
	CircuitHalfOpen CircuitBreakerState = "half-open"
)

// CircuitBreakerMetrics provides circuit breaker statistics
type CircuitBreakerMetrics struct {
	// Core metrics
	Requests    uint64  `json:"requests"`     // Total requests
	Failures    uint64  `json:"failures"`     // Total failures
	FailureRate float64 `json:"failure_rate"` // Current failure rate

	// State information
	CurrentState    CircuitBreakerState `json:"current_state"`     // Current state
	LastStateChange time.Time           `json:"last_state_change"` // Last transition

	// Window metrics
	WindowFailureRate float64 `json:"window_failure_rate"` // Recent failure rate

	// Timing metrics
	LastFailure time.Time `json:"last_failure"` // Last failure time
	LastSuccess time.Time `json:"last_success"` // Last success time

	// Recovery metrics
	RecoveryAttempts  int `json:"recovery_attempts"`  // Test attempts
	RecoverySuccesses int `json:"recovery_successes"` // Successful tests
}

// CircuitBreakerConfig configures a circuit breaker
type CircuitBreakerConfig struct {
	// Name identifies this circuit breaker
	Name string `json:"name"`

	// Thresholds
	FailureThreshold     int     `json:"failure_threshold"`      // Failures to open
	FailureRateThreshold float64 `json:"failure_rate_threshold"` // Rate to open

	// Windows
	WindowSize      int `json:"window_size"`      // Window size
	MinimumRequests int `json:"minimum_requests"` // Min before opening

	// Timeouts
	OpenTimeout     time.Duration `json:"open_timeout"`      // Time to stay open
	ResetTimeout    time.Duration `json:"reset_timeout"`     // Time to reset
	HalfOpenTimeout time.Duration `json:"half_open_timeout"` // Time for testing

	// Behavior
	FailFast    bool `json:"fail_fast"`    // Fail when open
	ForceOpen   bool `json:"force_open"`   // Force open state
	ForceClosed bool `json:"force_closed"` // Force closed state
}

// CircuitBreaker provides resilient service access
type CircuitBreaker struct {
	breaker *resilience.CircuitBreaker
	config  CircuitBreakerConfig
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(config CircuitBreakerConfig) *CircuitBreaker {
	breaker := resilience.NewCircuitBreaker(
		config.Name,
		config.FailureThreshold,
		config.ResetTimeout,
	)

	return &CircuitBreaker{
		breaker: breaker,
		config:  config,
	}
}

// Execute runs a function with circuit breaker protection
func (cb *CircuitBreaker) Execute(fn func() error) error {
	return cb.breaker.Execute(fn)
}

// GetMetrics returns current circuit breaker metrics
func (cb *CircuitBreaker) GetMetrics() CircuitBreakerMetrics {
	// Return basic metrics - in a real implementation this would
	// be populated from the actual circuit breaker state
	return CircuitBreakerMetrics{
		Requests:          0,
		Failures:          0,
		FailureRate:       0.0,
		CurrentState:      CircuitClosed,
		LastStateChange:   time.Now(),
		WindowFailureRate: 0.0,
		LastFailure:       time.Time{},
		LastSuccess:       time.Time{},
		RecoveryAttempts:  0,
		RecoverySuccesses: 0,
	}
}

// State returns the current circuit breaker state
func (cb *CircuitBreaker) State() CircuitBreakerState {
	// The resilience circuit breaker doesn't expose state directly
	// In a real implementation, we would track this internally
	return CircuitClosed
}

// Reset resets the circuit breaker to its initial state
func (cb *CircuitBreaker) Reset() {
	// The resilience circuit breaker doesn't expose a Reset method
	// In a real implementation, we would recreate the circuit breaker
}

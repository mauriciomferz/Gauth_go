// Package gauth provides circuit breaker functionality for resilient services.
package gauth

import (
	"time"

	"github.com/Gimel-Foundation/gauth/internal/circuit"
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
	breaker *circuit.Breaker
	monitor *circuit.Monitor
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(config CircuitBreakerConfig) *CircuitBreaker {
	opts := circuit.Options{
		Name:             config.Name,
		FailureThreshold: config.FailureThreshold,
		ResetTimeout:     config.ResetTimeout,
		HalfOpenLimit:    config.MinimumRequests,
	}

	breaker := circuit.NewBreaker(opts)
	monitor := circuit.NewMonitor()

	// Wire up monitoring
	opts.OnStateChange = func(name string, from, to circuit.State) {
		monitor.OnStateChange(name, from, to, time.Time{})
	}

	return &CircuitBreaker{
		breaker: breaker,
		monitor: monitor,
	}
}

// Execute runs a function with circuit breaker protection
func (cb *CircuitBreaker) Execute(fn func() error) error {
	return cb.breaker.Execute(fn)
}

// GetMetrics returns current circuit breaker metrics
func (cb *CircuitBreaker) GetMetrics() CircuitBreakerMetrics {
	stats := cb.monitor.GetStats(cb.breaker.Name())
	if stats == nil {
		return CircuitBreakerMetrics{}
	}

	return CircuitBreakerMetrics{
		Requests:          stats.Requests,
		Failures:          stats.Failures,
		FailureRate:       stats.FailureRate,
		CurrentState:      CircuitBreakerState(stats.CurrentState.String()),
		LastStateChange:   stats.LastStateChange,
		WindowFailureRate: stats.WindowFailureRate,
		LastFailure:       stats.LastFailure,
		LastSuccess:       stats.LastSuccess,
		RecoveryAttempts:  stats.HalfOpenAttempts,
		RecoverySuccesses: stats.HalfOpenSuccesses,
	}
}

// State returns the current circuit breaker state
func (cb *CircuitBreaker) State() CircuitBreakerState {
	state := cb.breaker.State()
	switch state {
	case circuit.StateClosed:
		return CircuitClosed
	case circuit.StateOpen:
		return CircuitOpen
	case circuit.StateHalfOpen:
		return CircuitHalfOpen
	default:
		return CircuitClosed
	}
}

// Reset resets the circuit breaker to its initial state
func (cb *CircuitBreaker) Reset() {
	cb.breaker.Reset()
	cb.monitor.Reset(cb.breaker.Name())
}

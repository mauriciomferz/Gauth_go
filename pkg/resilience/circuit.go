package resilience

import (
	"context"
	"errors"
	"sync"
	"time"
)

// ErrCircuitOpen is returned when the circuit breaker is open and requests are not allowed
var ErrCircuitOpen = errors.New("circuit breaker is open")

// CircuitState represents the state of a circuit breaker
type CircuitState int

const (
	// StateClosed indicates the circuit is closed and allowing requests
	StateClosed CircuitState = iota

	// StateOpen indicates the circuit is open and failing fast
	StateOpen

	// StateHalfOpen indicates the circuit is testing if the service has recovered
	StateHalfOpen
)

// CircuitConfig configures a circuit breaker
type CircuitConfig struct {
	// Name identifies the circuit breaker
	Name string

	// MaxFailures is how many failures trigger opening the circuit
	MaxFailures int

	// Timeout is how long to wait before trying again
	Timeout time.Duration

	// Interval is how often to reset failure counts
	Interval time.Duration

	// OnStateChange is called when circuit state changes
	OnStateChange func(from, to CircuitState)
}

// CircuitBreaker implements the circuit breaker pattern
type CircuitBreaker struct {
	config CircuitConfig

	mu          sync.RWMutex
	state       CircuitState
	failures    int
	lastFailure time.Time
	lastReset   time.Time
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(config CircuitConfig) *CircuitBreaker {
	return &CircuitBreaker{
		config:    config,
		state:     StateClosed,
		lastReset: time.Now(),
	}
}

// Execute runs an operation with circuit breaker protection
func (cb *CircuitBreaker) Execute(ctx context.Context, op func(context.Context) error) error {
	if err := cb.beforeExecute(); err != nil {
		return err
	}

	err := op(ctx)

	cb.afterExecute(err)
	return err
}

// State returns the current circuit state
func (cb *CircuitBreaker) State() CircuitState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

func (cb *CircuitBreaker) beforeExecute() error {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case StateOpen:
		if !cb.shouldAttemptReset() {
			return ErrCircuitOpen
		}
		cb.transitionTo(StateHalfOpen)
		return nil

	case StateHalfOpen:
		return ErrCircuitOpen

	default:
		return nil
	}
}

func (cb *CircuitBreaker) afterExecute(err error) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case StateHalfOpen:
		if err != nil {
			cb.transitionTo(StateOpen)
		} else {
			cb.transitionTo(StateClosed)
		}

	case StateClosed:
		if err != nil {
			cb.recordFailure()
		}

		cb.checkFailureThreshold()
	}
}

func (cb *CircuitBreaker) shouldAttemptReset() bool {
	return time.Since(cb.lastFailure) > cb.config.Timeout
}

func (cb *CircuitBreaker) recordFailure() {
	// Reset failure count if interval has elapsed
	if time.Since(cb.lastReset) > cb.config.Interval {
		cb.failures = 0
		cb.lastReset = time.Now()
	}

	cb.failures++
	cb.lastFailure = time.Now()
}

func (cb *CircuitBreaker) checkFailureThreshold() {
	if cb.failures >= cb.config.MaxFailures {
		cb.transitionTo(StateOpen)
	}
}

func (cb *CircuitBreaker) transitionTo(newState CircuitState) {
	if cb.state == newState {
		return
	}

	oldState := cb.state
	cb.state = newState

	if cb.config.OnStateChange != nil {
		cb.config.OnStateChange(oldState, newState)
	}

	// Reset counters on state change
	cb.failures = 0
	cb.lastReset = time.Now()
}

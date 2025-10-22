// Package circuit provides circuit breaker pattern implementation
package circuit

import (
	"context"
	"errors"
	"sync"
	"time"
)

// Options represents circuit breaker configuration options
type Options struct {
	Name             string
	FailureThreshold int           // number of consecutive failures to open the circuit
	ResetTimeout     time.Duration // how long to wait before allowing a trial (half-open)
	HalfOpenLimit    int           // reserved for future (number of concurrent probes)
}

// Breaker is an alias kept for backwards compatibility
type Breaker = CircuitBreaker

// State represents the circuit breaker state
type State int

const (
	StateClosed   State = iota // normal operation
	StateOpen                  // failures exceed threshold, fail-fast
	StateHalfOpen              // trial period to see if system recovered
)

// String converts state to human readable text
func (s State) String() string {
	switch s {
	case StateClosed:
		return "closed"
	case StateOpen:
		return "open"
	case StateHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

// CircuitBreaker implements a simple count-based circuit breaker with half-open probing.
type CircuitBreaker struct {
	mu             sync.Mutex
	name           string
	state          State
	failureCount   int
	threshold      int
	timeout        time.Duration
	lastFailure    time.Time
	halfOpenActive bool
}

// NewBreaker creates a new circuit breaker.
func NewBreaker(o Options) *CircuitBreaker {
	if o.FailureThreshold <= 0 {
		o.FailureThreshold = 5
	}
	if o.ResetTimeout <= 0 {
		o.ResetTimeout = 5 * time.Second
	}
	return &CircuitBreaker{
		name:      o.Name,
		state:     StateClosed,
		threshold: o.FailureThreshold,
		timeout:   o.ResetTimeout,
	}
}

// New is a backward compatibility shim returning a breaker with provided parameters.
// Deprecated: use NewBreaker with Options instead.
func New(name string, failureThreshold int, resetTimeout time.Duration) *CircuitBreaker {
	return NewBreaker(Options{Name: name, FailureThreshold: failureThreshold, ResetTimeout: resetTimeout})
}

// GetState returns the current state (helper for tests/inspection).
func (cb *CircuitBreaker) GetState() State {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	return cb.state
}

// Reset forces the breaker back to closed, clearing counts.
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.state = StateClosed
	cb.failureCount = 0
	cb.halfOpenActive = false
}

// Execute runs fn with circuit breaker protection.
func (cb *CircuitBreaker) Execute(ctx context.Context, fn func() error) error {
	// Fast path: check/open state decisions
	cb.mu.Lock()
	now := time.Now()
	switch cb.state {
	case StateOpen:
		if now.Sub(cb.lastFailure) >= cb.timeout {
			// move to half-open and allow one probe
			cb.state = StateHalfOpen
			cb.halfOpenActive = false // will be set true by probe path
		} else {
			cb.mu.Unlock()
			return errors.New("circuit breaker open")
		}
	}

	// If half-open, allow only one active probe at a time
	if cb.state == StateHalfOpen {
		if cb.halfOpenActive { // another probe in-flight
			cb.mu.Unlock()
			return errors.New("circuit breaker half-open busy")
		}
		cb.halfOpenActive = true
	}
	cb.mu.Unlock()

	// Run the protected function outside lock
	err := fn()

	// Update state
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if err != nil {
		cb.failureCount++
		cb.lastFailure = time.Now()
		if cb.state == StateHalfOpen {
			// failure during probe -> open
			cb.state = StateOpen
			cb.halfOpenActive = false
			cb.failureCount = cb.threshold // mark as at threshold
		} else if cb.failureCount >= cb.threshold {
			cb.state = StateOpen
		}
		return err
	}

	// success path
	if cb.state == StateHalfOpen {
		// success -> close and reset
		cb.state = StateClosed
		cb.failureCount = 0
		cb.halfOpenActive = false
	} else if cb.state == StateClosed {
		// successful execution in closed state resets failure counter
		cb.failureCount = 0
	}
	return nil
}

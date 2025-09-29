// Package circuit provides circuit breaker functionality for GAuth
package circuit

import (
	"sync"
	"time"

	"github.com/Gimel-Foundation/gauth/internal/errors"
)

// State represents the state of the circuit breaker
type State int

const (
	StateClosed   State = iota // Circuit is closed (allowing requests)
	StateOpen                  // Circuit is open (blocking requests)
	StateHalfOpen              // Circuit is half-open (testing if service is healthy)
)

// String returns a string representation of the state
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

// Breaker implements the circuit breaker pattern
type Breaker struct {
	name             string
	failureThreshold int
	resetTimeout     time.Duration
	halfOpenLimit    int
	failureCount     int
	state            State
	lastStateChange  time.Time
	halfOpenCount    int
	mu               sync.RWMutex
	onStateChange    func(name string, from, to State)
}

// Options configures a circuit breaker
type Options struct {
	Name             string
	FailureThreshold int
	ResetTimeout     time.Duration
	HalfOpenLimit    int
	OnStateChange    func(name string, from, to State)
}

// NewBreaker creates a new circuit breaker
func NewBreaker(opts Options) *Breaker {
	if opts.FailureThreshold <= 0 {
		opts.FailureThreshold = 5
	}
	if opts.ResetTimeout == 0 {
		opts.ResetTimeout = 10 * time.Second
	}
	if opts.HalfOpenLimit <= 0 {
		opts.HalfOpenLimit = 1
	}

	return &Breaker{
		name:             opts.Name,
		failureThreshold: opts.FailureThreshold,
		resetTimeout:     opts.ResetTimeout,
		halfOpenLimit:    opts.HalfOpenLimit,
		state:            StateClosed,
		lastStateChange:  time.Now(),
		onStateChange:    opts.OnStateChange,
	}
}

// Name returns the circuit breaker name
func (cb *Breaker) Name() string {
	return cb.name
}

// State returns the current circuit breaker state
func (cb *Breaker) State() State {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// Reset resets the circuit breaker state
func (cb *Breaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	oldState := cb.state
	cb.state = StateClosed
	cb.failureCount = 0
	cb.halfOpenCount = 0
	cb.lastStateChange = time.Now()

	if cb.onStateChange != nil {
		cb.onStateChange(cb.name, oldState, StateClosed)
	}
}

// Execute attempts to run the given function with circuit breaker protection
func (cb *Breaker) Execute(fn func() error) error {
	if !cb.allowRequest() {
		return errors.ErrCircuitOpen
	}

	err := fn()

	cb.mu.Lock()
	defer cb.mu.Unlock()

	if err != nil {
		cb.recordFailure()
		return err
	}

	cb.recordSuccess()
	return nil
}

// allowRequest checks if a request should be allowed
func (cb *Breaker) allowRequest() bool {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	switch cb.state {
	case StateClosed:
		return true
	case StateOpen:
		if time.Since(cb.lastStateChange) > cb.resetTimeout {
			cb.mu.RUnlock()
			cb.mu.Lock()
			cb.toHalfOpen()
			cb.mu.Unlock()
			cb.mu.RLock()
			return true
		}
		return false
	case StateHalfOpen:
		return cb.halfOpenCount < cb.halfOpenLimit
	default:
		return false
	}
}

// recordSuccess records a successful request
func (cb *Breaker) recordSuccess() {
	switch cb.state {
	case StateHalfOpen:
		cb.toClosed()
	case StateClosed:
		cb.failureCount = 0
	}
}

// recordFailure records a failed request
func (cb *Breaker) recordFailure() {
	cb.failureCount++

	switch cb.state {
	case StateClosed:
		if cb.failureCount >= cb.failureThreshold {
			cb.toOpen()
		}
	case StateHalfOpen:
		cb.toOpen()
	}
}

// toOpen changes the state to open
func (cb *Breaker) toOpen() {
	if cb.state != StateOpen {
		oldState := cb.state
		cb.state = StateOpen
		cb.lastStateChange = time.Now()
		if cb.onStateChange != nil {
			cb.onStateChange(cb.name, oldState, StateOpen)
		}
	}
}

// toHalfOpen changes the state to half-open
func (cb *Breaker) toHalfOpen() {
	if cb.state != StateHalfOpen {
		oldState := cb.state
		cb.state = StateHalfOpen
		cb.halfOpenCount = 0
		cb.lastStateChange = time.Now()
		if cb.onStateChange != nil {
			cb.onStateChange(cb.name, oldState, StateHalfOpen)
		}
	}
}

// toClosed changes the state to closed
func (cb *Breaker) toClosed() {
	if cb.state != StateClosed {
		oldState := cb.state
		cb.state = StateClosed
		cb.failureCount = 0
		cb.halfOpenCount = 0
		cb.lastStateChange = time.Now()
		if cb.onStateChange != nil {
			cb.onStateChange(cb.name, oldState, StateClosed)
		}
	}
}

// GetState returns the current state of the circuit breaker
func (cb *Breaker) GetState() State {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// Reset is now handled by the main Reset method above

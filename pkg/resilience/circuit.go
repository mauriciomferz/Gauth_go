package resilience

import (
	"context"
	"errors"
	"fmt"
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
	config            CircuitConfig
	mu                sync.RWMutex
	state             CircuitState
	failures          int
	lastFailure       time.Time
	lastReset         time.Time
	halfOpenMax       int
	halfOpenResults   []error // stores results of half-open attempts
	halfOpenCount     int     // number of attempts in current half-open cycle
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
	fmt.Printf("[DEBUG] beforeExecute: state=%v, halfOpenCount=%d, halfOpenMax=%d\n", cb.state, cb.halfOpenCount, cb.halfOpenMax)
       cb.mu.Lock()
       defer cb.mu.Unlock()

       for {
	       switch cb.state {
	       case StateOpen:
		       if !cb.shouldAttemptReset() {
			       return ErrCircuitOpen
		       }
		       cb.transitionTo(StateHalfOpen)
		       // Loop to re-check state as half-open
		       continue
	       case StateHalfOpen:
		       if cb.halfOpenCount >= cb.halfOpenMax {
			       return ErrCircuitOpen
		       }
		       // Increment after check so exactly halfOpenMax executions are allowed
		       cb.halfOpenCount++
		       return nil
	       default:
		       return nil
	       }
       }
}

func (cb *CircuitBreaker) afterExecute(err error) {
	fmt.Printf("[DEBUG] afterExecute: state=%v, halfOpenCount=%d, halfOpenMax=%d, err=%v\n", cb.state, cb.halfOpenCount, cb.halfOpenMax, err)
       cb.mu.Lock()
       defer cb.mu.Unlock()

       switch cb.state {
       case StateHalfOpen:
	       cb.halfOpenResults = append(cb.halfOpenResults, err)
	       if err != nil {
		       cb.transitionTo(StateOpen)
		       cb.halfOpenResults = nil
		       cb.halfOpenCount = 0
		       return
	       }
	       // Only transition after all allowed requests
	       if len(cb.halfOpenResults) == cb.halfOpenMax {
		       allSuccess := true
		       for _, res := range cb.halfOpenResults {
			       if res != nil {
				       allSuccess = false
				       break
			       }
		       }
		       if allSuccess {
			       cb.transitionTo(StateClosed)
		       } else {
			       cb.transitionTo(StateOpen)
		       }
		       cb.halfOpenResults = nil
		       cb.halfOpenCount = 0
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
       fmt.Printf("[DEBUG] recordFailure: failures=%d, maxFailures=%d\n", cb.failures, cb.config.MaxFailures)
}

func (cb *CircuitBreaker) checkFailureThreshold() {
       fmt.Printf("[DEBUG] checkFailureThreshold: failures=%d, maxFailures=%d\n", cb.failures, cb.config.MaxFailures)
       if cb.failures >= cb.config.MaxFailures {
	       fmt.Printf("[DEBUG] checkFailureThreshold: threshold reached, opening circuit\n")
	       cb.transitionTo(StateOpen)
       }
}

func (cb *CircuitBreaker) transitionTo(newState CircuitState) {
	fmt.Printf("[DEBUG] transitionTo: from=%v to=%v\n", cb.state, newState)
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
       // Only reset half-open tracking fields when leaving half-open
       if oldState == StateHalfOpen && newState != StateHalfOpen {
	       cb.halfOpenMax = 0
	       cb.halfOpenResults = nil
	       cb.halfOpenCount = 0
       }
       // Initialize half-open tracking fields when entering half-open
       if newState == StateHalfOpen {
	       cb.halfOpenMax = cb.config.MaxFailures
	       cb.halfOpenResults = make([]error, 0, cb.halfOpenMax)
	       cb.halfOpenCount = 0
       }
}

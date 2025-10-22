package circuit

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// State represents the circuit breaker state
type State int

const (
	Closed State = iota
	Open
	HalfOpen
)

// Options configures a circuit breaker
type Options struct {
	MaxFailures      int           `json:"max_failures"`
	FailureThreshold int           `json:"failure_threshold"`
	Timeout          time.Duration `json:"timeout"`
	ResetTimeout     time.Duration `json:"reset_timeout"`
	ReadyToTrip      func(counts Counts) bool
	OnStateChange    func(name string, from State, to State)
	MaxRequests      uint32        `json:"max_requests"`
	HalfOpenLimit    int           `json:"half_open_limit"`
	Interval         time.Duration `json:"interval"`
}

// Counts holds the statistics for a circuit breaker
type Counts struct {
	Requests             uint32
	TotalSuccesses       uint32
	TotalFailures        uint32
	ConsecutiveSuccesses uint32
	ConsecutiveFailures  uint32
}

// Breaker represents a circuit breaker
type Breaker struct {
	name          string
	maxFailures   int
	timeout       time.Duration
	maxRequests   uint32
	interval      time.Duration
	mutex         sync.Mutex
	state         State
	counts        Counts
	expiry        time.Time
	onStateChange func(name string, from State, to State)
}

// NewBreaker creates a new circuit breaker with the given options
func NewBreaker(opts Options) *Breaker {
	return &Breaker{
		maxFailures: opts.MaxFailures,
		timeout:     opts.Timeout,
		state:       Closed,
	}
}

// Execute executes the given function within the circuit breaker
func (cb *Breaker) Execute(fn func() error) error {
	return cb.ExecuteContext(context.Background(), fn)
}

// ExecuteContext executes the given function within the circuit breaker with context
func (cb *Breaker) ExecuteContext(ctx context.Context, fn func() error) error {
	if !cb.allow() {
		return fmt.Errorf("circuit breaker is open")
	}

	err := fn()
	cb.onResult(err == nil)
	return err
}

// allow checks if the request is allowed
func (cb *Breaker) allow() bool {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	now := time.Now()

	switch cb.state {
	case Closed:
		if cb.expiry.Before(now) {
			cb.counts = Counts{}
			cb.expiry = now.Add(cb.interval)
		}
		return true

	case Open:
		if cb.expiry.Before(now) {
			cb.setState(HalfOpen)
			cb.expiry = now.Add(cb.timeout)
			cb.counts = Counts{}
			return true
		}
		return false

	case HalfOpen:
		return cb.counts.Requests < cb.maxRequests

	default:
		return false
	}
}

// onResult handles the result of a request
func (cb *Breaker) onResult(success bool) {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	cb.counts.Requests++
	if success {
		cb.counts.TotalSuccesses++
		cb.counts.ConsecutiveSuccesses++
		cb.counts.ConsecutiveFailures = 0

		if cb.state == HalfOpen {
			cb.setState(Closed)
		}
	} else {
		cb.counts.TotalFailures++
		cb.counts.ConsecutiveFailures++
		cb.counts.ConsecutiveSuccesses = 0

		if cb.shouldTrip() {
			cb.setState(Open)
			cb.expiry = time.Now().Add(cb.timeout)
		}
	}
}

// shouldTrip checks if the circuit should trip to open state
func (cb *Breaker) shouldTrip() bool {
	return cb.counts.ConsecutiveFailures >= uint32(cb.maxFailures)
}

// setState changes the state and calls the state change callback
func (cb *Breaker) setState(state State) {
	if cb.state == state {
		return
	}

	prev := cb.state
	cb.state = state

	if cb.onStateChange != nil {
		cb.onStateChange(cb.name, prev, state)
	}
}

// State returns the current state of the circuit breaker
func (cb *Breaker) State() State {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()
	return cb.state
}

// Counts returns the current counts
func (cb *Breaker) Counts() Counts {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()
	return cb.counts
}

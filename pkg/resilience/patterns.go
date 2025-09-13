// Package resilience provides type-safe implementations of common resilience patterns
// like circuit breakers, rate limiting, retry with backoff, and bulkheads.
package resilience

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// State represents the state of a resilience pattern

// Patterns encapsulates all resilience patterns
type Patterns struct {
	name      string
	mu        sync.RWMutex
	failures  int
	successes int

	// Circuit breaker
	threshold   int
	timeout     time.Duration
	lastFailure time.Time
	state       CircuitState
	onState     func(name string, from, to CircuitState)

	// Rate limiter
	reqPerSec   int
	burst       int
	lastRequest time.Time
	tokens      int
	onRateLimit func(name string)

	// Retry
	maxAttempts  int
	baseInterval time.Duration
	maxInterval  time.Duration

	// Bulkhead
	maxConcurrent    int
	activeRequests   int
	requestSemaphore chan struct{}
}

// PatternsOption is a function that configures a Patterns instance
type PatternsOption func(*Patterns)

// NewPatterns creates a new Patterns instance

func NewPatterns(name string, opts ...PatternsOption) *Patterns {
	p := &Patterns{
		name:             name,
		state:            StateClosed,
		threshold:        5,
		timeout:          time.Second * 10,
		reqPerSec:        100,
		burst:            20,
		maxAttempts:      3,
		baseInterval:     time.Millisecond * 100,
		maxInterval:      time.Second,
		maxConcurrent:    10,
		requestSemaphore: make(chan struct{}, 10),
	}

	for _, opt := range opts {
		opt(p)
	}

	return p
}

// WithCircuitBreaker configures circuit breaker settings
func WithCircuitBreaker(threshold int, timeout time.Duration, onState func(name string, from, to CircuitState)) PatternsOption {
	return func(p *Patterns) {
		p.threshold = threshold
		p.timeout = timeout
		p.onState = onState
	}
}

// WithRateLimit configures rate limiting settings
func WithRateLimit(reqPerSec, burst int, onLimit func(name string)) PatternsOption {
	return func(p *Patterns) {
		p.reqPerSec = reqPerSec
		p.burst = burst
		p.onRateLimit = onLimit
	}
}

// WithRetry configures retry settings
func WithRetry(maxAttempts int, baseInterval, maxInterval time.Duration) PatternsOption {
	return func(p *Patterns) {
		p.maxAttempts = maxAttempts
		p.baseInterval = baseInterval
		p.maxInterval = maxInterval
	}
}

// WithBulkhead configures bulkhead settings
func WithBulkhead(maxConcurrent int) PatternsOption {
	return func(p *Patterns) {
		p.maxConcurrent = maxConcurrent
		p.requestSemaphore = make(chan struct{}, maxConcurrent)
	}
}

// Execute runs a function with all resilience patterns applied
func (p *Patterns) Execute(ctx context.Context, fn func() error) error {
	// Try bulkhead
	select {
	case p.requestSemaphore <- struct{}{}:
		defer func() { <-p.requestSemaphore }()
	case <-ctx.Done():
		return ctx.Err()
	default:
		return fmt.Errorf("bulkhead full for %s", p.name)
	}

	// Check rate limit
	p.mu.Lock()
	now := time.Now()
	elapsed := now.Sub(p.lastRequest)
	newTokens := int(float64(p.reqPerSec) * elapsed.Seconds())
	p.tokens = min(p.burst, p.tokens+newTokens)
	if p.tokens <= 0 {
		p.mu.Unlock()
		if p.onRateLimit != nil {
			p.onRateLimit(p.name)
		}
		return fmt.Errorf("rate limit exceeded for %s", p.name)
	}
	p.tokens--
	p.lastRequest = now
	p.mu.Unlock()

	// Check circuit breaker
	p.mu.RLock()
	state := p.state
	p.mu.RUnlock()

	switch state {
	case StateOpen:
		if time.Since(p.lastFailure) > p.timeout {
			p.changeState(StateHalfOpen)
		} else {
			return fmt.Errorf("circuit breaker open for %s", p.name)
		}
	case StateHalfOpen:
		select {
		case p.requestSemaphore <- struct{}{}:
			defer func() { <-p.requestSemaphore }()
		default:
			return fmt.Errorf("circuit breaker half-open, at test limit for %s", p.name)
		}
	}

	// Apply retry with backoff
	var lastErr error
	for attempt := 0; attempt < p.maxAttempts; attempt++ {
		if err := fn(); err != nil {
			lastErr = err
			p.recordFailure()

			// If not final attempt, wait with exponential backoff
			if attempt < p.maxAttempts-1 {
				backoff := p.baseInterval * time.Duration(1<<uint(attempt))
				if backoff > p.maxInterval {
					backoff = p.maxInterval
				}
				select {
				case <-time.After(backoff):
				case <-ctx.Done():
					return ctx.Err()
				}
				continue
			}
			return err
		}

		p.recordSuccess()
		return nil
	}

	return lastErr
}

func (p *Patterns) changeState(newState CircuitState) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.state != newState {
		oldState := p.state
		p.state = newState
		if p.onState != nil {
			p.onState(p.name, oldState, newState)
		}
	}
}

func (p *Patterns) recordSuccess() {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.successes++
	if p.state == StateHalfOpen && p.successes >= p.threshold {
		p.changeState(StateClosed)
		p.successes = 0
		p.failures = 0
	}
}

func (p *Patterns) recordFailure() {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.failures++
	p.lastFailure = time.Now()
	if (p.state == StateClosed || p.state == StateHalfOpen) && p.failures >= p.threshold {
		p.changeState(StateOpen)
		p.successes = 0
		p.failures = 0
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

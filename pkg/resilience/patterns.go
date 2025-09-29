package resilience

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// --- BEGIN STUBS FOR EXAMPLES AND DOCS ---
// RetryConfig is a stub for retry configuration
type RetryConfig struct {
	MaxAttempts  int
	InitialDelay time.Duration
	MaxDelay     time.Duration
	Multiplier   float64
}

// TimeoutConfig is a stub for timeout configuration
type TimeoutConfig struct {
	Timeout time.Duration
}

// BulkheadConfig is a stub for bulkhead configuration
type BulkheadConfig struct {
	MaxConcurrent int
	MaxWaitTime   time.Duration
}

// ErrBulkheadFull is a sentinel error for bulkhead full
var ErrBulkheadFull = fmt.Errorf("bulkhead capacity exceeded")

// Retry is a stub for retry pattern
type Retry struct {
	config RetryConfig
}

func NewRetry(config RetryConfig) *Retry {
	return &Retry{config: config}
}

func (r *Retry) Execute(ctx context.Context, fn func(ctx context.Context) error) error {
	// Minimal stub: just call fn once
	return fn(ctx)
}

// Timeout is a stub for timeout pattern
type Timeout struct {
	config TimeoutConfig
}

func NewTimeout(config TimeoutConfig) *Timeout {
	return &Timeout{config: config}
}

func (t *Timeout) Execute(ctx context.Context, fn func(ctx context.Context) error) error {
	// Minimal stub: just call fn once
	return fn(ctx)
}

// Bulkhead is a stub for bulkhead pattern
type Bulkhead struct {
	config BulkheadConfig
}

func NewBulkhead(config BulkheadConfig) *Bulkhead {
	return &Bulkhead{config: config}
}

func (b *Bulkhead) Execute(ctx context.Context, fn func(ctx context.Context) error) error {
	// Minimal stub: just call fn once
	return fn(ctx)
}

// Combined is a stub for pattern composition
type Combined struct {
	patterns []interface{}
}

func Combine(patterns ...interface{}) *Combined {
	return &Combined{patterns: patterns}
}

func (c *Combined) Execute(ctx context.Context, fn func(ctx context.Context) error) error {
	// Minimal stub: just call fn once
	return fn(ctx)
}

// --- END STUBS FOR EXAMPLES AND DOCS ---
// Package resilience provides type-safe implementations of common resilience patterns
// like circuit breakers, rate limiting, retry with backoff, and bulkheads.

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
	activeRequests   int //nolint:unused // reserved for future monitoring
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
	if err := p.checkBulkhead(ctx); err != nil {
		return err
	}
	defer func() { <-p.requestSemaphore }()

	if err := p.checkRateLimit(); err != nil {
		return err
	}

	if err := p.checkCircuitBreaker(ctx); err != nil {
		return err
	}

	return p.executeWithRetry(ctx, fn)
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

// checkBulkhead checks if the bulkhead allows the request
func (p *Patterns) checkBulkhead(ctx context.Context) error {
	select {
	case p.requestSemaphore <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	default:
		return fmt.Errorf("bulkhead full for %s", p.name)
	}
}

// checkRateLimit checks if the rate limit allows the request
func (p *Patterns) checkRateLimit() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	now := time.Now()
	elapsed := now.Sub(p.lastRequest)
	newTokens := int(float64(p.reqPerSec) * elapsed.Seconds())
	p.tokens = minInt(p.burst, p.tokens+newTokens)
	
	if p.tokens <= 0 {
		if p.onRateLimit != nil {
			p.onRateLimit(p.name)
		}
		return fmt.Errorf("rate limit exceeded for %s", p.name)
	}
	
	p.tokens--
	p.lastRequest = now
	return nil
}

// checkCircuitBreaker checks if the circuit breaker allows the request
func (p *Patterns) checkCircuitBreaker(ctx context.Context) error {
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
	return nil
}

// executeWithRetry executes the function with retry and exponential backoff
func (p *Patterns) executeWithRetry(ctx context.Context, fn func() error) error {
	var lastErr error
	for attempt := 0; attempt < p.maxAttempts; attempt++ {
		if err := fn(); err != nil {
			lastErr = err
			p.recordFailure()

			if attempt < p.maxAttempts-1 {
				backoff := p.calculateBackoff(attempt)
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

// calculateBackoff calculates exponential backoff duration
func (p *Patterns) calculateBackoff(attempt int) time.Duration {
	safeAttempt := attempt
	if safeAttempt > 30 { // 2^30 is large enough, prevents overflow
		safeAttempt = 30
	}
	
	var shiftAmount uint
	if safeAttempt >= 0 && safeAttempt <= 63 {
		shiftAmount = uint(safeAttempt)
	} else {
		shiftAmount = 30 // Cap at maximum safe shift
	}
	
	backoff := p.baseInterval * time.Duration(1<<shiftAmount)
	if backoff > p.maxInterval {
		backoff = p.maxInterval
	}
	return backoff
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

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

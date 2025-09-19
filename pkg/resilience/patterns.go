package resilience

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

// RetryStrategy is a legacy alias for RetryConfig for integration test compatibility
type RetryStrategy struct {
	MaxAttempts     int
	InitialInterval time.Duration
	MaxInterval     time.Duration
	Multiplier      float64
}

// NewRetry creates a new retry handler from RetryStrategy or RetryConfig
func NewRetry(strategy interface{}) *Retry {
       switch cfg := strategy.(type) {
       case RetryStrategy:
	       return NewRetry(RetryConfig{
		       MaxAttempts:  cfg.MaxAttempts,
		       InitialDelay:  cfg.InitialInterval,
		       MaxDelay:     cfg.MaxInterval,
		       Multiplier:    cfg.Multiplier,
	       })
       case RetryConfig:
	       return newRetryFromConfig(cfg)
       default:
	       return nil // unsupported type for NewRetry
       }
}

// newRetryFromConfig is the original NewRetry implementation for RetryConfig
func newRetryFromConfig(cfg RetryConfig) *Retry {
	if cfg.MaxAttempts <= 0 {
		cfg.MaxAttempts = 3
	}
	if cfg.InitialDelay <= 0 {
		cfg.InitialDelay = time.Second
	}
	if cfg.MaxDelay <= 0 {
		cfg.MaxDelay = 30 * time.Second
	}
	if cfg.Multiplier <= 1.0 {
		cfg.Multiplier = 2.0
	}
	return &Retry{config: cfg}
}


// RetryConfig defines configuration for Retry
type RetryConfig struct {
	MaxAttempts  int
	InitialDelay time.Duration
	MaxDelay     time.Duration
	Multiplier   float64
}

// TimeoutConfig defines configuration for Timeout
type TimeoutConfig struct {
	Timeout time.Duration
}

// ErrBulkheadFull is returned when bulkhead is full
var ErrBulkheadFull = errors.New("bulkhead full")

// Retry.Execute runs fn with retry logic and context
func (r *Retry) Execute(ctx context.Context, fn func(ctx context.Context) error) error {
       interval := r.config.InitialDelay
       for attempt := 1; attempt <= r.config.MaxAttempts; attempt++ {
	       err := fn(ctx)
	       if err == nil {
		       return nil
	       }
	       if attempt == r.config.MaxAttempts {
		       return err
	       }
	       select {
	       case <-time.After(interval):
	       case <-ctx.Done():
		       return ctx.Err()
	       }
	       interval = time.Duration(float64(interval) * r.config.Multiplier)
	       if interval > r.config.MaxDelay {
		       interval = r.config.MaxDelay
	       }
       }
       return errors.New("retry attempts exhausted")
}

// Timeout implements a timeout pattern
type Timeout struct {
	timeout time.Duration
}

// NewTimeout creates a new Timeout handler
func NewTimeout(cfg TimeoutConfig) *Timeout {
	return &Timeout{timeout: cfg.Timeout}
}

// Timeout.Execute runs fn with a timeout using context
func (t *Timeout) Execute(ctx context.Context, fn func(ctx context.Context) error) error {
       ctxTimeout, cancel := context.WithTimeout(ctx, t.timeout)
       defer cancel()
       done := make(chan error, 1)
       go func() {
	       done <- fn(ctxTimeout)
       }()
       select {
       case err := <-done:
	       return err
       case <-ctxTimeout.Done():
	       return ctxTimeout.Err()
       }
}
// BulkheadConfig defines configuration for Bulkhead
type BulkheadConfig struct {
	MaxConcurrent int
	MaxWaitTime   time.Duration
}

// Retry implements retry with exponential backoff
type Retry struct {
       config RetryConfig
}



// Do executes the function with retry logic (backward compatibility)
func (r *Retry) Do(fn func() error) error {
       interval := r.config.InitialDelay
       for attempt := 1; attempt <= r.config.MaxAttempts; attempt++ {
	       err := fn()
	       if err == nil {
		       return nil
	       }
	       if attempt == r.config.MaxAttempts {
		       return err
	       }
	       time.Sleep(interval)
	       interval = time.Duration(float64(interval) * r.config.Multiplier)
	       if interval > r.config.MaxDelay {
		       interval = r.config.MaxDelay
	       }
       }
       return errors.New("retry attempts exhausted")
}

// Bulkhead is a stub for bulkhead pattern
type Bulkhead struct {
	maxConcurrent int
	maxWaitTime   time.Duration
	semaphore     chan struct{}
}

func NewBulkhead(config BulkheadConfig) *Bulkhead {
	if config.MaxConcurrent <= 0 {
		 config.MaxConcurrent = 1
	}
	return &Bulkhead{
		 maxConcurrent: config.MaxConcurrent,
		 maxWaitTime:   config.MaxWaitTime,
		 semaphore:     make(chan struct{}, config.MaxConcurrent),
	}
}

func (b *Bulkhead) Execute(ctx context.Context, fn func(ctx context.Context) error) error {
	timer := time.NewTimer(b.maxWaitTime)
	defer timer.Stop()
	select {
	case b.semaphore <- struct{}{}:
		 defer func() { <-b.semaphore }()
		 return fn(ctx)
	case <-timer.C:
		 return ErrBulkheadFull
	case <-ctx.Done():
		 return ctx.Err()
	}
}

// Combined is a stub for pattern composition
type Combined struct {
	patterns []interface{}
}

func Combine(patterns ...interface{}) *Combined {
	return &Combined{patterns: patterns}
}

func (c *Combined) Execute(ctx context.Context, fn func(ctx context.Context) error) error {
       wrapped := fn
       // Apply patterns in reverse order so the first pattern wraps the rest
       for i := len(c.patterns) - 1; i >= 0; i-- {
	       p := c.patterns[i]
	       switch pat := p.(type) {
	       case interface{ Execute(context.Context, func(context.Context) error) error }:
		       next := wrapped
		       wrapped = func(ctx context.Context) error {
			       err := pat.Execute(ctx, next)
			       // If the error is ErrCircuitOpen, propagate immediately (do not retry)
			       if errors.Is(err, ErrCircuitOpen) {
				       return err
			       }
			       return err
		       }
	       default:
		       // Ignore unknown pattern types
	       }
       }
       return wrapped(ctx)
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
		// Do NOT reset p.failures here; keep failures count to keep circuit open
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

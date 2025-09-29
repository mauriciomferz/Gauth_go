// Package resilience provides additional resilience patterns for GAuth
package resilience

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/Gimel-Foundation/gauth/internal/circuit"
)

// RetryStrategy defines how retries should be performed
type RetryStrategy struct {
	MaxAttempts     int
	InitialInterval time.Duration
	MaxInterval     time.Duration
	Multiplier      float64
}

// Bulkhead limits concurrent operations
type Bulkhead struct {
	sem    chan struct{}
	maxCon int
}

// NewBulkhead creates a new bulkhead with max concurrent operations
func NewBulkhead(maxConcurrent int) *Bulkhead {
	return &Bulkhead{
		sem:    make(chan struct{}, maxConcurrent),
		maxCon: maxConcurrent,
	}
}

// Execute runs an operation with bulkhead protection
func (b *Bulkhead) Execute(ctx context.Context, fn func() error) error {
	select {
	case b.sem <- struct{}{}:
		defer func() { <-b.sem }()
		return fn()
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Retry implements retry with exponential backoff
type Retry struct {
	strategy RetryStrategy
}

// NewRetry creates a new retry handler
func NewRetry(strategy RetryStrategy) *Retry {
	if strategy.MaxAttempts <= 0 {
		strategy.MaxAttempts = 3
	}
	if strategy.InitialInterval <= 0 {
		strategy.InitialInterval = time.Second
	}
	if strategy.MaxInterval <= 0 {
		strategy.MaxInterval = 30 * time.Second
	}
	if strategy.Multiplier <= 0 {
		strategy.Multiplier = 2.0
	}

	return &Retry{strategy: strategy}
}

// Execute runs an operation with retry
func (r *Retry) Execute(ctx context.Context, fn func() error) error {
	var lastErr error
	interval := r.strategy.InitialInterval

	for attempt := 0; attempt < r.strategy.MaxAttempts; attempt++ {
		err := fn()
		if err == nil {
			return nil
		}

		lastErr = err
		if !r.shouldRetry(err) {
			return err
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(interval):
			interval = time.Duration(float64(interval) * r.strategy.Multiplier)
			if interval > r.strategy.MaxInterval {
				interval = r.strategy.MaxInterval
			}
		}
	}

	return lastErr
}

func (r *Retry) shouldRetry(err error) bool {
	// Add custom logic to determine if error is retryable
	return !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded)
}

// RateLimiter implements token bucket rate limiting
type RateLimiter struct {
	rate       float64
	burstSize  int
	tokens     float64
	lastRefill time.Time
	mu         sync.Mutex
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(rate float64, burstSize int) *RateLimiter {
	return &RateLimiter{
		rate:       rate,
		burstSize:  burstSize,
		tokens:     float64(burstSize),
		lastRefill: time.Now(),
	}
}

// Allow checks if a request should be allowed
func (rl *RateLimiter) Allow() bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(rl.lastRefill).Seconds()
	rl.tokens = min(float64(rl.burstSize), rl.tokens+elapsed*rl.rate)
	rl.lastRefill = now

	if rl.tokens < 1 {
		return false
	}

	rl.tokens--
	return true
}

// Composite combines multiple resilience patterns
type Composite struct {
	breaker   *circuit.Breaker
	bulkhead  *Bulkhead
	retry     *Retry
	rateLimit *RateLimiter
}

// NewComposite creates a new composite resilience handler
func NewComposite(opts CompositeOptions) *Composite {
	return &Composite{
		breaker:   circuit.NewBreaker(opts.CircuitOptions),
		bulkhead:  NewBulkhead(opts.MaxConcurrent),
		retry:     NewRetry(opts.RetryStrategy),
		rateLimit: NewRateLimiter(opts.RateLimit, opts.BurstSize),
	}
}

// CompositeOptions configures the composite handler
type CompositeOptions struct {
	CircuitOptions circuit.Options
	MaxConcurrent  int
	RetryStrategy  RetryStrategy
	RateLimit      float64
	BurstSize      int
}

// Execute runs an operation with all resilience patterns
func (c *Composite) Execute(ctx context.Context, fn func() error) error {
	if !c.rateLimit.Allow() {
		return errors.New("rate limit exceeded")
	}

	return c.bulkhead.Execute(ctx, func() error {
		return c.breaker.Execute(func() error {
			return c.retry.Execute(ctx, fn)
		})
	})
}

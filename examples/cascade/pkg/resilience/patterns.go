package resilience

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// CircuitBreaker implements the circuit breaker pattern
type CircuitBreaker struct {
	name             string
	failureThreshold int
	resetTimeout     time.Duration
	failures         int
	lastFailure      time.Time
	state            State
	mu               sync.RWMutex
}

type State int

const (
	StateClosed State = iota
	StateOpen
	StateHalfOpen
)

func (s State) String() string {
	switch s {
	case StateClosed:
		return "Closed"
	case StateOpen:
		return "Open"
	case StateHalfOpen:
		return "HalfOpen"
	default:
		return "Unknown"
	}
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(name string, failureThreshold int, resetTimeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		name:             name,
		failureThreshold: failureThreshold,
		resetTimeout:     resetTimeout,
		state:            StateClosed,
	}
}

// Execute runs the given function with circuit breaker protection
func (cb *CircuitBreaker) Execute(fn func() error) error {
	cb.mu.Lock()
	if cb.state == StateOpen {
		if time.Since(cb.lastFailure) > cb.resetTimeout {
			cb.state = StateHalfOpen
			fmt.Printf("[%s] Circuit half-open\n", cb.name)
		} else {
			cb.mu.Unlock()
			return fmt.Errorf("circuit breaker is open")
		}
	}
	cb.mu.Unlock()

	err := fn()

	cb.mu.Lock()
	defer cb.mu.Unlock()

	if err != nil {
		cb.failures++
		cb.lastFailure = time.Now()
		if cb.failures >= cb.failureThreshold {
			cb.state = StateOpen
			fmt.Printf("[%s] Circuit opened\n", cb.name)
		}
		return err
	}

	if cb.state == StateHalfOpen {
		cb.state = StateClosed
		fmt.Printf("[%s] Circuit closed\n", cb.name)
	}
	cb.failures = 0
	return nil
}

// RateLimiter implements a simple token bucket rate limiter
type RateLimiter struct {
	tokens          int
	capacity        int
	refillRate      int
	lastRefill      time.Time
	tokensPerSecond int
	mu              sync.Mutex
}

func NewRateLimiter(tokensPerSecond, burstSize int) *RateLimiter {
	return &RateLimiter{
		tokens:          burstSize,
		capacity:        burstSize,
		refillRate:      tokensPerSecond,
		tokensPerSecond: tokensPerSecond,
		lastRefill:      time.Now(),
	}
}

func (rl *RateLimiter) Allow() error {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(rl.lastRefill)
	rl.lastRefill = now

	// Refill tokens based on elapsed time
	newTokens := int(elapsed.Seconds() * float64(rl.tokensPerSecond))
	if newTokens > 0 {
		rl.tokens = min(rl.capacity, rl.tokens+newTokens)
	}

	if rl.tokens > 0 {
		rl.tokens--
		return nil
	}
	return fmt.Errorf("rate limit exceeded")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Bulkhead implements the bulkhead pattern using a semaphore
type Bulkhead struct {
	sem chan struct{}
}

func NewBulkhead(maxConcurrent int) *Bulkhead {
	return &Bulkhead{
		sem: make(chan struct{}, maxConcurrent),
	}
}

func (b *Bulkhead) Execute(ctx context.Context, fn func() error) error {
	select {
	case b.sem <- struct{}{}:
		defer func() { <-b.sem }()
		return fn()
	case <-ctx.Done():
		return ctx.Err()
	default:
		return fmt.Errorf("bulkhead full")
	}
}

// RetryStrategy defines retry behavior
type RetryStrategy struct {
	MaxAttempts     int
	InitialInterval time.Duration
	MaxInterval     time.Duration
	Multiplier      float64
}

// Retry implements the retry pattern with exponential backoff
type Retry struct {
	strategy RetryStrategy
}

func NewRetry(strategy RetryStrategy) *Retry {
	return &Retry{strategy: strategy}
}

func (r *Retry) Execute(ctx context.Context, fn func() error) error {
	var err error
	interval := r.strategy.InitialInterval

	for attempt := 1; attempt <= r.strategy.MaxAttempts; attempt++ {
		if err = fn(); err == nil {
			return nil
		}

		if attempt == r.strategy.MaxAttempts {
			break
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

	return fmt.Errorf("retry failed after %d attempts: %w", r.strategy.MaxAttempts, err)
}

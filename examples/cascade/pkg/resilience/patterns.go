// Package resilience provides type-safe implementations of common resilience patterns.
package resilience

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/Gimel-Foundation/gauth/examples/cascade/pkg/events"
)

// CircuitBreaker implements the circuit breaker pattern
type CircuitBreaker struct {
	name           string
	errorThreshold int
	resetTimeout   time.Duration
	failures       int
	lastFailure    time.Time
	state          State
	mu             sync.RWMutex
	eventPublisher *events.EventPublisher
}

// State represents circuit breaker states
type State int

const (
	StateClosed State = iota
	StateHalfOpen
	StateOpen
)

// CircuitBreakerError indicates that the circuit is open
type CircuitBreakerError struct {
	Service     string
	OpenedSince time.Time
}

func (e CircuitBreakerError) Error() string {
	return "circuit breaker is open for service: " + e.Service
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(name string, threshold int, timeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		name:           name,
		errorThreshold: threshold,
		resetTimeout:   timeout,
		state:         StateClosed,
		eventPublisher: &events.EventPublisher{},
	}
}

// Execute runs the provided function with circuit breaker protection
func (cb *CircuitBreaker) Execute(ctx context.Context, fn func() error) error {
	cb.mu.RLock()
	if cb.state == StateOpen {
		if time.Since(cb.lastFailure) > cb.resetTimeout {
			cb.mu.RUnlock()
			cb.mu.Lock()
			cb.state = StateHalfOpen
			cb.mu.Unlock()
		} else {
			cb.mu.RUnlock()
			return CircuitBreakerError{
				Service:     cb.name,
				OpenedSince: cb.lastFailure,
			}
		}
	} else {
		cb.mu.RUnlock()
	}

	err := fn()
	
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if err != nil {
		cb.failures++
		cb.lastFailure = time.Now()

		if cb.state == StateHalfOpen || cb.failures >= cb.errorThreshold {
			cb.state = StateOpen
			cb.eventPublisher.Publish(events.Event{
				Type:      events.CircuitOpened,
				Timestamp: time.Now(),
				Service:   cb.name,
				Data: events.CircuitBreakerData{
					FailureCount:    cb.failures,
					LastFailureTime: cb.lastFailure,
					ResetTimeout:    cb.resetTimeout,
				},
			})
		}
	} else if cb.state == StateHalfOpen {
		cb.state = StateClosed
		cb.failures = 0
		cb.eventPublisher.Publish(events.Event{
			Type:      events.CircuitClosed,
			Timestamp: time.Now(),
			Service:   cb.name,
		})
	}

	return err
}

// RateLimiter implements rate limiting pattern
type RateLimiter struct {
	name            string
	requestsPerSec  float64
	burstSize      int
	requests       []time.Time
	mu             sync.Mutex
	eventPublisher *events.EventPublisher
}

// RateLimitExceededError indicates rate limit has been exceeded
type RateLimitExceededError struct {
	Service       string
	CurrentRate   float64
	Limit         float64
}

func (e RateLimitExceededError) Error() string {
	return "rate limit exceeded for service: " + e.Service
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(name string, rps float64, burst int) *RateLimiter {
	return &RateLimiter{
		name:           name,
		requestsPerSec: rps,
		burstSize:     burst,
		requests:      make([]time.Time, 0, burst),
		eventPublisher: &events.EventPublisher{},
	}
}

// Allow checks if a request can proceed under current rate limits
func (rl *RateLimiter) Allow() error {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	window := time.Second
	
	// Remove old requests
	i := 0
	for ; i < len(rl.requests); i++ {
		if now.Sub(rl.requests[i]) <= window {
			break
		}
	}
	rl.requests = rl.requests[i:]

	// Check rate limit
	if len(rl.requests) >= rl.burstSize {
		currentRate := float64(len(rl.requests))
		rl.eventPublisher.Publish(events.Event{
			Type:      events.RateLimitExceeded,
			Timestamp: now,
			Service:   rl.name,
			Data: events.RateLimitData{
				CurrentRate:     currentRate,
				Limit:          rl.requestsPerSec,
				WindowDuration: window,
			},
		})
		return RateLimitExceededError{
			Service:     rl.name,
			CurrentRate: currentRate,
			Limit:       rl.requestsPerSec,
		}
	}

	// Record request
	rl.requests = append(rl.requests, now)
	return nil
}

// Bulkhead implements the bulkhead pattern
type Bulkhead struct {
	name          string
	maxConcurrent int
	active        int
	mu            sync.Mutex
	eventPublisher *events.EventPublisher
}

// BulkheadFullError indicates all resources are in use
type BulkheadFullError struct {
	Service       string
	MaxConcurrent int
}

func (e BulkheadFullError) Error() string {
	return "bulkhead is full for service: " + e.Service
}

// NewBulkhead creates a new bulkhead
func NewBulkhead(name string, max int) *Bulkhead {
	return &Bulkhead{
		name:          name,
		maxConcurrent: max,
		eventPublisher: &events.EventPublisher{},
	}
}

// Execute runs the provided function with bulkhead protection
func (b *Bulkhead) Execute(ctx context.Context, fn func() error) error {
	b.mu.Lock()
	if b.active >= b.maxConcurrent {
		b.mu.Unlock()
		b.eventPublisher.Publish(events.Event{
			Type:      events.ResourceExhausted,
			Timestamp: time.Now(),
			Service:   b.name,
			Data: events.ResourceData{
				ResourceType: "bulkhead",
				CurrentUsage: float64(b.active),
				MaxCapacity: float64(b.maxConcurrent),
			},
		})
		return BulkheadFullError{
			Service:       b.name,
			MaxConcurrent: b.maxConcurrent,
		}
	}
	b.active++
	b.mu.Unlock()

	defer func() {
		b.mu.Lock()
		b.active--
		b.mu.Unlock()
	}()

	return fn()
}
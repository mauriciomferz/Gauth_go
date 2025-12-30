package external

import (
	"sync"
	"time"
)

// CircuitBreakerState represents the state of a circuit breaker
type CircuitBreakerState int

const (
	// CircuitBreakerClosed normal operation, requests pass through
	CircuitBreakerClosed CircuitBreakerState = iota
	// CircuitBreakerOpen circuit is open, requests fail fast
	CircuitBreakerOpen
	// CircuitBreakerHalfOpen testing if service has recovered
	CircuitBreakerHalfOpen
)

// CircuitBreaker implements the circuit breaker pattern for fault tolerance
type CircuitBreaker struct {
	maxFailures  int
	timeout      time.Duration
	resetTimeout time.Duration
	state        CircuitBreakerState
	failures     int
	lastFailTime time.Time
	mu           sync.RWMutex
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(maxFailures int, timeout, resetTimeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		maxFailures:  maxFailures,
		timeout:      timeout,
		resetTimeout: resetTimeout,
		state:        CircuitBreakerClosed,
	}
}

// Call executes a function with circuit breaker protection
func (cb *CircuitBreaker) Call(fn func() error) error {
	cb.mu.Lock()

	// Check if we should transition from Open to Half-Open
	if cb.state == CircuitBreakerOpen {
		if time.Since(cb.lastFailTime) > cb.resetTimeout {
			cb.state = CircuitBreakerHalfOpen
			cb.failures = 0
		}
	}

	// Fail fast if circuit is open
	if cb.state == CircuitBreakerOpen {
		cb.mu.Unlock()
		return ErrCircuitBreakerOpen
	}

	cb.mu.Unlock()

	// Execute the function
	err := fn()

	cb.mu.Lock()
	defer cb.mu.Unlock()

	if err != nil {
		cb.failures++
		cb.lastFailTime = time.Now()

		// Open circuit if max failures reached
		if cb.failures >= cb.maxFailures {
			cb.state = CircuitBreakerOpen
		}

		return err
	}

	// Success - reset to closed state
	if cb.state == CircuitBreakerHalfOpen {
		cb.state = CircuitBreakerClosed
	}
	cb.failures = 0

	return nil
}

// GetState returns the current state of the circuit breaker
func (cb *CircuitBreaker) GetState() CircuitBreakerState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// Reset manually resets the circuit breaker
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.state = CircuitBreakerClosed
	cb.failures = 0
}

// ResponseCache implements a simple TTL-based cache
type ResponseCache struct {
	cache map[string]*cacheEntry
	ttl   time.Duration
	mu    sync.RWMutex
}

type cacheEntry struct {
	data      interface{}
	timestamp time.Time
}

// NewResponseCache creates a new response cache
func NewResponseCache(ttl time.Duration) *ResponseCache {
	cache := &ResponseCache{
		cache: make(map[string]*cacheEntry),
		ttl:   ttl,
	}

	// Start cleanup goroutine
	go cache.cleanup()

	return cache
}

// Get retrieves a value from the cache
func (rc *ResponseCache) Get(key string) (interface{}, bool) {
	rc.mu.RLock()
	defer rc.mu.RUnlock()

	entry, exists := rc.cache[key]
	if !exists {
		return nil, false
	}

	// Check if entry has expired
	if time.Since(entry.timestamp) > rc.ttl {
		return nil, false
	}

	return entry.data, true
}

// Set stores a value in the cache
func (rc *ResponseCache) Set(key string, value interface{}) {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	rc.cache[key] = &cacheEntry{
		data:      value,
		timestamp: time.Now(),
	}
}

// Delete removes a value from the cache
func (rc *ResponseCache) Delete(key string) {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	delete(rc.cache, key)
}

// Clear removes all entries from the cache
func (rc *ResponseCache) Clear() {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	rc.cache = make(map[string]*cacheEntry)
}

// cleanup periodically removes expired entries
func (rc *ResponseCache) cleanup() {
	ticker := time.NewTicker(rc.ttl)
	defer ticker.Stop()

	for range ticker.C {
		rc.mu.Lock()
		now := time.Now()
		for key, entry := range rc.cache {
			if now.Sub(entry.timestamp) > rc.ttl {
				delete(rc.cache, key)
			}
		}
		rc.mu.Unlock()
	}
}

// Size returns the number of entries in the cache
func (rc *ResponseCache) Size() int {
	rc.mu.RLock()
	defer rc.mu.RUnlock()
	return len(rc.cache)
}

// Common errors
var (
	ErrCircuitBreakerOpen = &CircuitBreakerError{Message: "circuit breaker is open"}
)

// CircuitBreakerError represents a circuit breaker error
type CircuitBreakerError struct {
	Message string
}

func (e *CircuitBreakerError) Error() string {
	return e.Message
}

// Common shared types across connectors
// Note: Each country connector defines its own specific request/response types
// to accommodate country-specific validation requirements and data structures

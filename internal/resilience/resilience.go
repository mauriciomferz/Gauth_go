// Package resilience provides resilience patterns implementation
package resilience

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// RetryPolicy defines retry behavior
type RetryPolicy struct {
	MaxRetries    int
	Delay         time.Duration
	BackoffFactor float64
}

// RetryStrategy represents retry configuration
type RetryStrategy struct {
	MaxAttempts     int
	InitialInterval time.Duration
	MaxInterval     time.Duration
	Multiplier      float64
}

// ExecuteWithRetry executes a function with retry logic
func ExecuteWithRetry(ctx context.Context, policy RetryPolicy, fn func() error) error {
	var lastErr error
	delay := policy.Delay

	for i := 0; i <= policy.MaxRetries; i++ {
		lastErr = fn()
		if lastErr == nil {
			return nil
		}

		if i < policy.MaxRetries {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
				delay = time.Duration(float64(delay) * policy.BackoffFactor)
			}
		}
	}

	return lastErr
}

// Retry represents a retry mechanism with configurable strategy
type Retry struct {
	Strategy RetryStrategy
}

// NewRetry creates a new retry mechanism
func NewRetry(strategy RetryStrategy) *Retry {
	return &Retry{
		Strategy: strategy,
	}
}

// Execute executes a function with retry logic
func (r *Retry) Execute(ctx context.Context, fn func() error) error {
	policy := RetryPolicy{
		MaxRetries:    r.Strategy.MaxAttempts - 1, // MaxAttempts includes the initial attempt
		Delay:         r.Strategy.InitialInterval,
		BackoffFactor: r.Strategy.Multiplier,
	}
	return ExecuteWithRetry(ctx, policy, fn)
}

// The calculateDelay function has been removed as it is unused.

// Bulkhead implements the bulkhead pattern for resource isolation
type Bulkhead struct {
	semaphore     chan struct{}
	maxConcurrent int
}

// NewBulkhead creates a new bulkhead with the specified max concurrent operations
func NewBulkhead(maxConcurrent int) *Bulkhead {
	return &Bulkhead{
		semaphore:     make(chan struct{}, maxConcurrent),
		maxConcurrent: maxConcurrent,
	}
}

// Execute runs the given function with bulkhead protection
func (b *Bulkhead) Execute(ctx context.Context, fn func() error) error {
	// Acquire semaphore
	select {
	case b.semaphore <- struct{}{}:
		defer func() { <-b.semaphore }() // Release semaphore
	case <-ctx.Done():
		return ctx.Err()
	}

	// Execute the function
	return fn()
}

// Timeout implements timeout pattern
type Timeout struct {
	duration time.Duration
}

// NewTimeout creates a new timeout instance
func NewTimeout(duration time.Duration) *Timeout {
	return &Timeout{
		duration: duration,
	}
}

// Execute runs the given function with timeout protection
func (t *Timeout) Execute(ctx context.Context, fn func() error) error {
	// Create context with timeout
	timeoutCtx, cancel := context.WithTimeout(ctx, t.duration)
	defer cancel()

	// Channel to receive the result
	done := make(chan error, 1)

	// Execute the function in a goroutine
	go func() {
		done <- fn()
	}()

	// Wait for either completion or timeout
	select {
	case err := <-done:
		return err
	case <-timeoutCtx.Done():
		return fmt.Errorf("operation timed out after %v", t.duration)
	}
}

// Fallback implements the fallback pattern
type Fallback struct {
	fallbackFn func() error
}

// NewFallback creates a new fallback instance
func NewFallback(fallbackFn func() error) *Fallback {
	return &Fallback{
		fallbackFn: fallbackFn,
	}
}

// Execute runs the primary function, falling back to the fallback function on error
func (f *Fallback) Execute(primaryFn func() error) error {
	if err := primaryFn(); err != nil {
		if f.fallbackFn != nil {
			return f.fallbackFn()
		}
		return err
	}
	return nil
}

// HealthChecker implements health checking functionality
type HealthChecker struct {
	checks  map[string]HealthCheck
	timeout time.Duration
	mutex   sync.RWMutex
}

// HealthCheck represents a single health check
type HealthCheck struct {
	Name    string
	CheckFn func(ctx context.Context) error
	Timeout time.Duration
}

// HealthResult represents the result of a health check
type HealthResult struct {
	Name     string
	Status   string
	Error    error
	Duration time.Duration
}

// NewHealthChecker creates a new health checker
func NewHealthChecker(timeout time.Duration) *HealthChecker {
	return &HealthChecker{
		checks:  make(map[string]HealthCheck),
		timeout: timeout,
	}
}

// AddCheck adds a health check
func (hc *HealthChecker) AddCheck(name string, checkFn func(ctx context.Context) error) {
	hc.mutex.Lock()
	defer hc.mutex.Unlock()

	hc.checks[name] = HealthCheck{
		Name:    name,
		CheckFn: checkFn,
		Timeout: hc.timeout,
	}
}

// CheckAll runs all health checks
func (hc *HealthChecker) CheckAll(ctx context.Context) []HealthResult {
	hc.mutex.RLock()
	checks := make([]HealthCheck, 0, len(hc.checks))
	for _, check := range hc.checks {
		checks = append(checks, check)
	}
	hc.mutex.RUnlock()

	results := make([]HealthResult, len(checks))
	var wg sync.WaitGroup

	for i, check := range checks {
		wg.Add(1)
		go func(i int, check HealthCheck) {
			defer wg.Done()
			results[i] = hc.runSingleCheck(ctx, check)
		}(i, check)
	}

	wg.Wait()
	return results
}

func (hc *HealthChecker) runSingleCheck(ctx context.Context, check HealthCheck) HealthResult {
	start := time.Now()

	// Create timeout context for the check
	checkCtx, cancel := context.WithTimeout(ctx, check.Timeout)
	defer cancel()

	// Run the check
	err := check.CheckFn(checkCtx)
	duration := time.Since(start)

	status := "healthy"
	if err != nil {
		status = "unhealthy"
	}

	return HealthResult{
		Name:     check.Name,
		Status:   status,
		Error:    err,
		Duration: duration,
	}
}

// Cache implements a simple in-memory cache with TTL
type Cache struct {
	items map[string]*cacheItem
	mutex sync.RWMutex
	ttl   time.Duration
}

type cacheItem struct {
	value     interface{}
	expiresAt time.Time
}

// NewCache creates a new cache with the specified TTL
func NewCache(ttl time.Duration) *Cache {
	cache := &Cache{
		items: make(map[string]*cacheItem),
		ttl:   ttl,
	}

	// Start cleanup goroutine
	go cache.cleanupExpired()

	return cache
}

// Get retrieves a value from the cache
func (c *Cache) Get(key string) (interface{}, bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	item, exists := c.items[key]
	if !exists {
		return nil, false
	}

	if time.Now().After(item.expiresAt) {
		delete(c.items, key)
		return nil, false
	}

	return item.value, true
}

// Set stores a value in the cache
func (c *Cache) Set(key string, value interface{}) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.items[key] = &cacheItem{
		value:     value,
		expiresAt: time.Now().Add(c.ttl),
	}
}

// Delete removes a value from the cache
func (c *Cache) Delete(key string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	delete(c.items, key)
}

func (c *Cache) cleanupExpired() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		c.mutex.Lock()
		now := time.Now()
		for key, item := range c.items {
			if now.After(item.expiresAt) {
				delete(c.items, key)
			}
		}
		c.mutex.Unlock()
	}
}

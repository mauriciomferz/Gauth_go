// # Licensing
//
// This file is part of the GAuth project and is licensed under the Apache License 2.0.
// It incorporates code and concepts from:
//   - OAuth 2.0 and OpenID Connect (Apache 2.0 License)
//   - Model Context Protocol (MIT License)
// See the LICENSE file in the project root for details.

package auth

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/time/rate"
)

// ProperConcurrencyManager provides thread-safe operations with proper synchronization
// This replaces the coarse-grained mutex approach throughout the codebase
type ProperConcurrencyManager struct {
	// Use fine-grained locking instead of coarse locks
	userLocks    sync.Map // map[string]*sync.RWMutex for per-user locking
	rateLimiters sync.Map // map[string]*rate.Limiter for per-user rate limiting
	
	// Atomic counters for statistics
	activeRequests int64
	totalRequests  int64
	
	// Configuration
	maxConcurrentRequests int64
	defaultRateLimit      rate.Limit
	defaultBurst          int
}

// NewProperConcurrencyManager creates a new concurrency manager
func NewProperConcurrencyManager(maxConcurrent int64, rateLimit rate.Limit, burst int) *ProperConcurrencyManager {
	return &ProperConcurrencyManager{
		maxConcurrentRequests: maxConcurrent,
		defaultRateLimit:      rateLimit,
		defaultBurst:          burst,
	}
}

// getUserLock gets or creates a per-user lock for fine-grained synchronization
func (pcm *ProperConcurrencyManager) getUserLock(userID string) *sync.RWMutex {
	if lock, exists := pcm.userLocks.Load(userID); exists {
		return lock.(*sync.RWMutex)
	}
	
	// Create new lock
	newLock := &sync.RWMutex{}
	actual, _ := pcm.userLocks.LoadOrStore(userID, newLock)
	return actual.(*sync.RWMutex)
}

// getUserRateLimiter gets or creates a per-user rate limiter
func (pcm *ProperConcurrencyManager) getUserRateLimiter(userID string) *rate.Limiter {
	if limiter, exists := pcm.rateLimiters.Load(userID); exists {
		return limiter.(*rate.Limiter)
	}
	
	// Create new rate limiter
	newLimiter := rate.NewLimiter(pcm.defaultRateLimit, pcm.defaultBurst)
	actual, _ := pcm.rateLimiters.LoadOrStore(userID, newLimiter)
	return actual.(*rate.Limiter)
}

// WithUserLock executes a function with a per-user read lock
func (pcm *ProperConcurrencyManager) WithUserReadLock(userID string, fn func() error) error {
	lock := pcm.getUserLock(userID)
	lock.RLock()
	defer lock.RUnlock()
	
	return fn()
}

// WithUserWriteLock executes a function with a per-user write lock
func (pcm *ProperConcurrencyManager) WithUserWriteLock(userID string, fn func() error) error {
	lock := pcm.getUserLock(userID)
	lock.Lock()
	defer lock.Unlock()
	
	return fn()
}

// CheckRateLimit checks if a user has exceeded their rate limit
func (pcm *ProperConcurrencyManager) CheckRateLimit(ctx context.Context, userID string) error {
	limiter := pcm.getUserRateLimiter(userID)
	
	if !limiter.Allow() {
		return fmt.Errorf("rate limit exceeded for user %s", userID)
	}
	
	return nil
}

// WaitForRateLimit waits for rate limit availability with context cancellation
func (pcm *ProperConcurrencyManager) WaitForRateLimit(ctx context.Context, userID string) error {
	limiter := pcm.getUserRateLimiter(userID)
	
	return limiter.Wait(ctx)
}

// TryAcquireRequest attempts to acquire a request slot (for global concurrency limiting)
func (pcm *ProperConcurrencyManager) TryAcquireRequest() bool {
	current := atomic.LoadInt64(&pcm.activeRequests)
	if current >= pcm.maxConcurrentRequests {
		return false
	}
	
	// Try to increment atomically
	return atomic.CompareAndSwapInt64(&pcm.activeRequests, current, current+1)
}

// AcquireRequest waits for a request slot with timeout
func (pcm *ProperConcurrencyManager) AcquireRequest(ctx context.Context) error {
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if pcm.TryAcquireRequest() {
				atomic.AddInt64(&pcm.totalRequests, 1)
				return nil
			}
		}
	}
}

// ReleaseRequest releases a request slot
func (pcm *ProperConcurrencyManager) ReleaseRequest() {
	atomic.AddInt64(&pcm.activeRequests, -1)
}

// GetStats returns concurrency statistics
func (pcm *ProperConcurrencyManager) GetStats() ConcurrencyStats {
	return ConcurrencyStats{
		ActiveRequests: atomic.LoadInt64(&pcm.activeRequests),
		TotalRequests:  atomic.LoadInt64(&pcm.totalRequests),
		MaxConcurrent:  pcm.maxConcurrentRequests,
	}
}

// ConcurrencyStats holds concurrency statistics
type ConcurrencyStats struct {
	ActiveRequests int64 `json:"active_requests"`
	TotalRequests  int64 `json:"total_requests"`
	MaxConcurrent  int64 `json:"max_concurrent"`
}

// RequestScope manages the lifecycle of a request with proper cleanup
type RequestScope struct {
	manager   *ProperConcurrencyManager
	userID    string
	acquired  bool
	startTime time.Time
}

// NewRequestScope creates a new request scope
func (pcm *ProperConcurrencyManager) NewRequestScope(ctx context.Context, userID string) (*RequestScope, error) {
	scope := &RequestScope{
		manager:   pcm,
		userID:    userID,
		startTime: time.Now(),
	}
	
	// Check rate limit first
	if err := pcm.CheckRateLimit(ctx, userID); err != nil {
		return nil, err
	}
	
	// Acquire request slot
	if err := pcm.AcquireRequest(ctx); err != nil {
		return nil, err
	}
	
	scope.acquired = true
	return scope, nil
}

// Close releases the request scope
func (rs *RequestScope) Close() {
	if rs.acquired {
		rs.manager.ReleaseRequest()
		rs.acquired = false
	}
}

// Duration returns how long the request has been active
func (rs *RequestScope) Duration() time.Duration {
	return time.Since(rs.startTime)
}

// SafeMap provides a thread-safe map implementation with proper locking
type SafeMap[K comparable, V any] struct {
	mu   sync.RWMutex
	data map[K]V
}

// NewSafeMap creates a new thread-safe map
func NewSafeMap[K comparable, V any]() *SafeMap[K, V] {
	return &SafeMap[K, V]{
		data: make(map[K]V),
	}
}

// Get retrieves a value with read lock
func (sm *SafeMap[K, V]) Get(key K) (V, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	
	value, exists := sm.data[key]
	return value, exists
}

// Set stores a value with write lock
func (sm *SafeMap[K, V]) Set(key K, value V) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	
	sm.data[key] = value
}

// Delete removes a value with write lock
func (sm *SafeMap[K, V]) Delete(key K) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	
	delete(sm.data, key)
}

// Len returns the number of elements with read lock
func (sm *SafeMap[K, V]) Len() int {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	
	return len(sm.data)
}

// Keys returns all keys with read lock
func (sm *SafeMap[K, V]) Keys() []K {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	
	keys := make([]K, 0, len(sm.data))
	for k := range sm.data {
		keys = append(keys, k)
	}
	return keys
}

// ForEach iterates over all key-value pairs with read lock
func (sm *SafeMap[K, V]) ForEach(fn func(K, V) bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	
	for k, v := range sm.data {
		if !fn(k, v) {
			break
		}
	}
}

// SafeCounter provides a thread-safe counter
type SafeCounter struct {
	value int64
}

// NewSafeCounter creates a new thread-safe counter
func NewSafeCounter() *SafeCounter {
	return &SafeCounter{}
}

// Increment atomically increments the counter
func (sc *SafeCounter) Increment() int64 {
	return atomic.AddInt64(&sc.value, 1)
}

// Decrement atomically decrements the counter
func (sc *SafeCounter) Decrement() int64 {
	return atomic.AddInt64(&sc.value, -1)
}

// Get atomically gets the current value
func (sc *SafeCounter) Get() int64 {
	return atomic.LoadInt64(&sc.value)
}

// Set atomically sets the value
func (sc *SafeCounter) Set(value int64) {
	atomic.StoreInt64(&sc.value, value)
}

// WorkerPool provides a proper worker pool implementation
type WorkerPool struct {
	workers   int
	taskQueue chan func()
	wg        sync.WaitGroup
	quit      chan bool
	started   bool
	mu        sync.Mutex
}

// NewWorkerPool creates a new worker pool
func NewWorkerPool(workers int, queueSize int) *WorkerPool {
	return &WorkerPool{
		workers:   workers,
		taskQueue: make(chan func(), queueSize),
		quit:      make(chan bool),
	}
}

// Start starts the worker pool
func (wp *WorkerPool) Start() {
	wp.mu.Lock()
	defer wp.mu.Unlock()
	
	if wp.started {
		return
	}
	
	wp.started = true
	for i := 0; i < wp.workers; i++ {
		wp.wg.Add(1)
		go wp.worker()
	}
}

// worker is the worker goroutine
func (wp *WorkerPool) worker() {
	defer wp.wg.Done()
	
	for {
		select {
		case task := <-wp.taskQueue:
			if task != nil {
				task()
			}
		case <-wp.quit:
			return
		}
	}
}

// Submit submits a task to the worker pool
func (wp *WorkerPool) Submit(task func()) error {
	wp.mu.Lock()
	started := wp.started
	wp.mu.Unlock()
	
	if !started {
		return fmt.Errorf("worker pool not started")
	}
	
	select {
	case wp.taskQueue <- task:
		return nil
	default:
		return fmt.Errorf("task queue is full")
	}
}

// Stop stops the worker pool
func (wp *WorkerPool) Stop() {
	wp.mu.Lock()
	defer wp.mu.Unlock()
	
	if !wp.started {
		return
	}
	
	close(wp.quit)
	wp.wg.Wait()
	wp.started = false
}

// NOTE: CircuitBreaker implementation removed to avoid conflicts
// Real circuit breaker functionality is available in:
// - pkg/resilience/circuit.go (working implementation)
// - internal/circuit/breaker.go (internal implementation)
// 
// This duplicate definition was causing naming conflicts.
// Use pkg/resilience.CircuitBreaker for actual circuit breaking functionality.
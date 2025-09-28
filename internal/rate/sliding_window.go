package rate

import (
	"context"
	"errors"
	"sync"
	"time"
)

var (
	// ErrRateLimitExceeded is returned when the rate limit is exceeded.
	ErrRateLimitExceeded = errors.New("rate limit exceeded")
)

// window implements a sliding window of requests.
type window struct {
	requests  []time.Time
	lastClean time.Time
}

// memoryStore implements Store using in-memory storage.
type memoryStore struct {
	mu      sync.RWMutex
	windows map[string]*window
	config  *Config
}

func newMemoryStore(config *Config) *memoryStore {
	return &memoryStore{
		windows: make(map[string]*window),
		config:  config,
	}
}

func (s *memoryStore) Add(_ context.Context, id string, t time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	w, exists := s.windows[id]
	if !exists {
		w = &window{
			requests:  make([]time.Time, 0),
			lastClean: t,
		}
		s.windows[id] = w
	}

	// Clean old requests from window
	windowStart := t.Add(-time.Duration(s.config.WindowSize) * time.Second)
	validRequests := make([]time.Time, 0)
	for _, rt := range w.requests {
		if rt.After(windowStart) {
			validRequests = append(validRequests, rt)
		}
	}

	// Check if we've exceeded the rate limit
	if len(validRequests) >= s.config.RequestsPerSecond*s.config.WindowSize {
		return ErrRateLimitExceeded
	}

	// Add current request
	validRequests = append(validRequests, t)
	w.requests = validRequests

	return nil
}

func (s *memoryStore) GetWindow(id string, start time.Time) []time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()

	w, exists := s.windows[id]
	if !exists {
		return nil
	}

	validRequests := make([]time.Time, 0)
	for _, t := range w.requests {
		if t.After(start) {
			validRequests = append(validRequests, t)
		}
	}
	return validRequests
}

func (s *memoryStore) Reset(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if w, exists := s.windows[id]; exists {
		w.requests = make([]time.Time, 0)
		w.lastClean = time.Now()
	}
}

func (s *memoryStore) Remove(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.windows, id)
}

func (s *memoryStore) Cleanup(threshold time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for id, w := range s.windows {
		validRequests := make([]time.Time, 0)
		for _, t := range w.requests {
			if t.After(threshold) {
				validRequests = append(validRequests, t)
			}
		}
		if len(validRequests) == 0 && time.Since(w.lastClean) > time.Hour {
			delete(s.windows, id)
		} else {
			w.requests = validRequests
			w.lastClean = time.Now()
		}
	}
}

// slidingWindow implements Algorithm using a sliding window approach.
type slidingWindow struct {
	store  Store
	config *Config
}

func newSlidingWindow(config *Config) *slidingWindow {
	return &slidingWindow{
		store:  newMemoryStore(config),
		config: config,
	}
}

func (w *slidingWindow) Allow(ctx context.Context, id string) error {
	return w.store.Add(ctx, id, time.Now())
}

func (w *slidingWindow) GetRemainingQuota(id string) int {
	now := time.Now()
	windowStart := now.Add(-time.Duration(w.config.WindowSize) * time.Second)
	requests := w.store.GetWindow(id, windowStart)

	maxRequests := w.config.RequestsPerSecond * w.config.WindowSize
	remaining := maxRequests - len(requests)
	if remaining < 0 {
		remaining = 0
	}
	return remaining
}

func (w *slidingWindow) Reset(id string) {
	w.store.Reset(id)
}

// Limiter provides thread-safe rate limiting using a sliding window algorithm.
type Limiter struct {
	algorithm Algorithm
	cleaner   *time.Ticker
	done      chan struct{}
}

// NewLimiter creates a new rate limiter with the given configuration.
func NewLimiter(config *Config) *Limiter {
	l := &Limiter{
		algorithm: newSlidingWindow(config),
		done:      make(chan struct{}),
	}
	l.startCleanup()
	return l
}

// Allow checks if a request should be allowed based on the rate limit configuration.
func (l *Limiter) Allow(ctx context.Context, id string) error {
	return l.algorithm.Allow(ctx, id)
}

// GetRemainingRequests returns the number of remaining requests allowed.
func (l *Limiter) GetRemainingRequests(id string) int {
	return l.algorithm.GetRemainingQuota(id)
}

// Reset resets the window for a given ID.
func (l *Limiter) Reset(id string) {
	l.algorithm.Reset(id)
}

func (l *Limiter) startCleanup() {
	l.cleaner = time.NewTicker(time.Minute)
	go func() {
		for {
			select {
			case <-l.cleaner.C:
				if store, ok := l.algorithm.(interface{ Cleanup(time.Time) }); ok {
					store.Cleanup(time.Now().Add(-time.Hour))
				}
			case <-l.done:
				l.cleaner.Stop()
				return
			}
		}
	}()
}

// Close stops the cleaner goroutine. Should be called when the limiter is no longer needed.
func (l *Limiter) Close() {
	close(l.done)
}

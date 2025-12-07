package policy

import (
	"sync"
	"time"
)

// Metrics tracks policy evaluation statistics
type Metrics struct {
	Total          uint64
	Allow          uint64
	Deny           uint64
	LastReason     string
	LastAt         time.Time
	LastMatched    int
	LastDeniedBy   int
	LatencyBuckets map[int64]*uint64 // upper bound ns -> *count (atomic)
	P99LatencyNS   int64
	Revisions      uint64 // total appended bundles (monotonic)
	ActiveVersion  int    // current effective bundle version (after rollback)
	RollbackCount  uint64 // number of successful rollback operations
	DiffRequests   uint64 // number of diff endpoint requests (successful)
	sync.RWMutex
}

// Config holds configuration for the policy handler
type Config struct {
	PersistPath string
}

// SimpleRateLimiter implements a basic in-memory rate limiter
type SimpleRateLimiter struct {
	limit  int
	window time.Duration
	mu     sync.Mutex
	slots  map[string]*RLSlot
}

type RLSlot struct {
	count int
	reset time.Time
}

func NewSimpleRateLimiter(limit int, window time.Duration) *SimpleRateLimiter {
	return &SimpleRateLimiter{limit: limit, window: window, slots: make(map[string]*RLSlot)}
}

func (rl *SimpleRateLimiter) Allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	s := rl.slots[key]
	now := time.Now()
	if s == nil || now.After(s.reset) {
		s = &RLSlot{count: 0, reset: now.Add(rl.window)}
		rl.slots[key] = s
	}
	if s.count < rl.limit {
		s.count++
		return true
	}
	return false
}

// ForceReset clears all rate limit slots (for testing).
func (rl *SimpleRateLimiter) ForceReset() {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	rl.slots = make(map[string]*RLSlot)
}

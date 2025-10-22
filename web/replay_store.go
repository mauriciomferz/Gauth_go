package web

import (
	"os"
	"strconv"
	"sync"
	"time"
)

// ReplayNonceStore provides minimal in-memory nonce/JTI replay protection for token issuance.
// Not production-ready: single-process only, periodic TTL sweep on access.
type ReplayNonceStore struct {
	mu   sync.Mutex
	ttl  time.Duration
	seen map[string]time.Time
	cap  int // optional capacity limit; evict oldest on insert when exceeded
}

func NewReplayNonceStore(ttl time.Duration) *ReplayNonceStore {
	cap := 0
	if raw := os.Getenv("GAUTH_REPLAY_CAP"); raw != "" {
		if v, err := strconv.Atoi(raw); err == nil && v > 0 {
			cap = v
		}
	}
	return &ReplayNonceStore{ttl: ttl, seen: make(map[string]time.Time), cap: cap}
}

// Seen returns true if nonce already recorded (and not expired). Performs lazy TTL cleanup.
func (r *ReplayNonceStore) Seen(n string, now time.Time) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	// Cleanup
	for k, t := range r.seen {
		if now.Sub(t) > r.ttl {
			delete(r.seen, k)
		}
	}
	t, ok := r.seen[n]
	if !ok {
		return false
	}
	if now.Sub(t) > r.ttl {
		delete(r.seen, n)
		return false
	}
	return true
}

// Record stores nonce occurrence time.
func (r *ReplayNonceStore) Record(n string, now time.Time) {
	r.mu.Lock()
	r.seen[n] = now
	r.mu.Unlock()
}

// RecordWithEvict records and applies capacity eviction if configured.
func (r *ReplayNonceStore) RecordWithEvict(n string, now time.Time) {
	r.mu.Lock()
	defer r.mu.Unlock()
	// TTL cleanup first
	for k, t := range r.seen {
		if now.Sub(t) > r.ttl {
			delete(r.seen, k)
		}
	}
	r.seen[n] = now
	if r.cap > 0 && len(r.seen) > r.cap {
		// Evict oldest (linear scan sufficient for small demo; optimize with min-heap if needed)
		var oldestKey string
		var oldestTime time.Time
		first := true
		for k, t := range r.seen {
			if first || t.Before(oldestTime) {
				oldestKey = k
				oldestTime = t
				first = false
			}
		}
		delete(r.seen, oldestKey)
	}
}

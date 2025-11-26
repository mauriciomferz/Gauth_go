package web

import (
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/mauriciomferz/Gauth_go/internal/metrics"
	"github.com/mauriciomferz/Gauth_go/pkg/replay"
)

// ReplayNonceStore provides minimal in-memory nonce/JTI replay protection for token issuance.
// Not production-ready: single-process only, periodic TTL sweep on access.
type ReplayNonceStore struct {
	mu      sync.Mutex
	ttl     time.Duration
	seen    map[string]time.Time
	cap     int // optional capacity limit; evict oldest on insert when exceeded
	wal     *replay.WALStore
	metrics metrics.Metrics // optional; nil-safe
}

// NewReplayNonceStore creates a store without metrics (backward compatibility).
func NewReplayNonceStore(ttl time.Duration) *ReplayNonceStore {
	return newReplayNonceStore(ttl, nil)
}

// NewReplayNonceStoreWithMetrics creates a store with metrics instrumentation.
func NewReplayNonceStoreWithMetrics(ttl time.Duration, m metrics.Metrics) *ReplayNonceStore {
	return newReplayNonceStore(ttl, m)
}

// internal shared constructor logic.
func newReplayNonceStore(ttl time.Duration, m metrics.Metrics) *ReplayNonceStore {
	cap := 0
	if raw := os.Getenv("GAUTH_REPLAY_CAP"); raw != "" {
		if v, err := strconv.Atoi(raw); err == nil && v > 0 {
			cap = v
		}
	}
	walPath := os.Getenv("GAUTH_REPLAY_WAL")
	var wal *replay.WALStore
	if walPath != "" {
		w, err := replay.NewWALStore(walPath)
		if err == nil {
			wal = w
		} else if m != nil {
			m.IncReplayStoreErrors()
		}
	}
	store := &ReplayNonceStore{ttl: ttl, seen: make(map[string]time.Time), cap: cap, wal: wal, metrics: m}
	// Recover from WAL if available (corruption-tolerant)
	if wal != nil {
		start := time.Now()
		skipped := 0
		_, skipped, _ = wal.RecoverWithStats(func(rec replay.WALRecord) error {
			if rec.Op == "Record" {
				// Clamp future timestamps to now to avoid negative TTL windows
				ts := time.Unix(rec.TS, 0)
				if ts.After(time.Now()) {
					ts = time.Now()
				}
				store.seen[string(rec.Key)] = ts
			}
			return nil
		})
		if m != nil {
			m.ObserveReplayStoreLatency(time.Since(start))
			for i := 0; i < skipped; i++ { // count each skipped malformed line
				m.IncReplayStoreErrors()
			}
		}
	}
	return store
}

// NewReplayNonceStoreWithConfig creates a store with explicit capacity and WAL path instead of relying
// on environment variables. This enables isolated durable replay stores (e.g. attestation vs token issuance)
// without mutating global process environment. If walPath is empty durability is disabled.
func NewReplayNonceStoreWithConfig(ttl time.Duration, capacity int, walPath string, m metrics.Metrics) *ReplayNonceStore {
	store := &ReplayNonceStore{ttl: ttl, seen: make(map[string]time.Time), cap: capacity, metrics: m}
	if walPath != "" {
		if w, err := replay.NewWALStore(walPath); err == nil {
			store.wal = w
			// Attempt recovery
			start := time.Now()
			skipped := 0
			_, skipped, _ = store.wal.RecoverWithStats(func(rec replay.WALRecord) error {
				if rec.Op == "Record" {
					ts := time.Unix(rec.TS, 0)
					if ts.After(time.Now()) {
						ts = time.Now()
					}
					store.seen[string(rec.Key)] = ts
				}
				return nil
			})
			if m != nil {
				m.ObserveReplayStoreLatency(time.Since(start))
				for i := 0; i < skipped; i++ {
					m.IncReplayStoreErrors()
				}
			}
		} else if m != nil {
			m.IncReplayStoreErrors()
		}
	}
	return store
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
	if r.wal != nil {
		start := time.Now()
		err := r.wal.AppendRecord(replay.WALRecord{
			Op:    "Record",
			Key:   []byte(n),
			Value: nil,
			TS:    now.Unix(),
		})
		if r.metrics != nil {
			if err != nil {
				r.metrics.IncReplayStoreErrors()
			} else {
				r.metrics.ObserveReplayStoreLatency(time.Since(start))
				// Pending WAL entries approximated by active seen size
				r.metrics.SetReplayWALPending(len(r.seen))
			}
		}
	}
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
	if r.wal != nil {
		start := time.Now()
		err := r.wal.AppendRecord(replay.WALRecord{
			Op:    "Record",
			Key:   []byte(n),
			Value: nil,
			TS:    now.Unix(),
		})
		if r.metrics != nil {
			if err != nil {
				r.metrics.IncReplayStoreErrors()
			} else {
				r.metrics.ObserveReplayStoreLatency(time.Since(start))
				// Update pending WAL entries gauge
				r.metrics.SetReplayWALPending(len(r.seen))
			}
		}
	}
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

// Size returns current number of active (non-expired) entries. Performs a lazy TTL cleanup first.
func (r *ReplayNonceStore) Size() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	now := time.Now()
	for k, t := range r.seen {
		if now.Sub(t) > r.ttl {
			delete(r.seen, k)
		}
	}
	return len(r.seen)
}

// SnapshotAndCompact persists current active entries to snapshot file and rotates WAL.
// After rotation all active entries are re-appended to a fresh WAL for minimal recovery surface.
// This reduces WAL growth and bounds recovery time.
func (r *ReplayNonceStore) SnapshotAndCompact() error {
	if r.wal == nil {
		return nil
	} // nothing to do
	r.mu.Lock()
	// Build active map (exclude expired)
	now := time.Now()
	active := make(map[string]time.Time, len(r.seen))
	for k, t := range r.seen {
		if now.Sub(t) <= r.ttl {
			active[k] = t
		}
	}
	r.mu.Unlock()
	// Write snapshot outside lock and measure duration
	var snapStart time.Time
	if r.metrics != nil {
		snapStart = time.Now()
	}
	if err := r.wal.Snapshot(active); err != nil {
		if r.metrics != nil {
			r.metrics.IncReplayStoreErrors()
		}
		return err
	}
	if r.metrics != nil {
		r.metrics.ObserveReplayWALSnapshotDuration(time.Since(snapStart))
	}
	// Rotate WAL (truncate)
	if err := r.wal.Rotate(); err != nil {
		if r.metrics != nil {
			r.metrics.IncReplayStoreErrors()
		}
		return err
	}
	// Re-append active entries
	start := time.Now()
	for k, ts := range active {
		_ = r.wal.AppendRecord(replay.WALRecord{Op: "Record", Key: []byte(k), TS: ts.Unix()})
	}
	if r.metrics != nil {
		r.metrics.ObserveReplayWALFlushLatency(time.Since(start))
		r.metrics.SetReplayWALPending(0)
	}
	return nil
}

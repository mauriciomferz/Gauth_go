package token

import (
	"container/list"
	"fmt"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/mauriciomferz/AgentAuth/internal/metrics"
	"github.com/mauriciomferz/AgentAuth/pkg/replay"
)

// CheckAndStore implements auth.ReplayStore to provide replay protection.
func (r *ReplayNonceStore) CheckAndStore(jti string) error {
	now := time.Now()
	if r.Seen(jti, now) {
		return fmt.Errorf("replay detected: JTI %s already seen", jti)
	}
	r.Record(jti, now)
	return nil
}

// ReplayNonceStore provides minimal in-memory nonce/JTI replay protection for token issuance.
// Not production-ready: single-process only, periodic TTL sweep on access.
type ReplayNonceStore struct {
	mu      sync.Mutex
	ttl     time.Duration
	seen    map[string]*list.Element
	lru     *list.List
	cap     int // optional capacity limit; evict oldest on insert when exceeded
	wal     *replay.WALStore
	metrics metrics.Metrics // optional; nil-safe
	stopCh  chan struct{}
}

type replayEntry struct {
	key string
	ts  time.Time
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
	if raw := os.Getenv("AGENTAUTH_REPLAY_CAP"); raw != "" {
		if v, err := strconv.Atoi(raw); err == nil && v > 0 {
			cap = v
		}
	}
	if raw := os.Getenv("AGENTAUTH_REPLAY_TTL"); raw != "" {
		if v, err := time.ParseDuration(raw); err == nil && v > 0 {
			ttl = v
		}
	}
	walPath := os.Getenv("AGENTAUTH_REPLAY_WAL")
	var wal *replay.WALStore
	if walPath != "" {
		w, err := replay.NewWALStore(walPath)
		if err == nil {
			wal = w
		} else if m != nil {
			m.IncReplayStoreErrors()
		}
	}
	store := &ReplayNonceStore{
		ttl:     ttl,
		seen:    make(map[string]*list.Element),
		lru:     list.New(),
		cap:     cap,
		wal:     wal,
		metrics: m,
		stopCh:  make(chan struct{}),
	}
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
				store.seen[string(rec.Key)] = store.lru.PushBack(&replayEntry{key: string(rec.Key), ts: ts})
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
	if os.Getenv("AGENTAUTH_DISABLE_BG_POLLS") != "1" {
		go store.backgroundCleanup()
	}
	return store
}

// NewReplayNonceStoreWithConfig creates a store with explicit capacity and WAL path instead of relying
// on environment variables. This enables isolated durable replay stores (e.g. attestation vs token issuance)
// without mutating global process environment. If walPath is empty durability is disabled.
func NewReplayNonceStoreWithConfig(ttl time.Duration, capacity int, walPath string, m metrics.Metrics) *ReplayNonceStore {
	store := &ReplayNonceStore{
		ttl:     ttl,
		seen:    make(map[string]*list.Element),
		lru:     list.New(),
		cap:     capacity,
		metrics: m,
		stopCh:  make(chan struct{}),
	}
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
					store.seen[string(rec.Key)] = store.lru.PushBack(&replayEntry{key: string(rec.Key), ts: ts})
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
	if os.Getenv("AGENTAUTH_DISABLE_BG_POLLS") != "1" {
		go store.backgroundCleanup()
	}
	return store
}

// backgroundCleanup periodically removes expired entries to prevent unbounded growth.
func (r *ReplayNonceStore) backgroundCleanup() {
	// Run cleanup at 1/10th of TTL or at least every 5s, max 1m
	interval := r.ttl / 10
	if interval < 5*time.Second {
		interval = 5 * time.Second
	} else if interval > 1*time.Minute {
		interval = 1 * time.Minute
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-r.stopCh:
			return
		case <-ticker.C:
			r.cleanup()
		}
	}
}

// cleanup removes expired entries.
func (r *ReplayNonceStore) cleanup() {
	r.mu.Lock()
	defer r.mu.Unlock()
	now := time.Now()
	for {
		el := r.lru.Front()
		if el == nil {
			break
		}
		ent := el.Value.(*replayEntry)
		if now.Sub(ent.ts) > r.ttl {
			delete(r.seen, ent.key)
			r.lru.Remove(el)
		} else {
			break // Front is the oldest; if it's not expired, nothing after it is
		}
	}
}

// Seen returns true if nonce already recorded (and not expired).
// Optimized to avoid O(N) scan.
func (r *ReplayNonceStore) Seen(n string, now time.Time) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	el, ok := r.seen[n]
	if !ok {
		return false
	}
	ent := el.Value.(*replayEntry)
	if now.Sub(ent.ts) > r.ttl {
		delete(r.seen, n) // Lazy delete on access
		r.lru.Remove(el)
		return false
	}
	return true
}

// Record stores nonce occurrence time, enforcing capacity logic.
func (r *ReplayNonceStore) Record(n string, now time.Time) {
	r.RecordWithEvict(n, now)
}

// RecordWithEvict records and applies capacity eviction if configured.
func (r *ReplayNonceStore) RecordWithEvict(n string, now time.Time) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// If already seen, update its position and time
	if el, ok := r.seen[n]; ok {
		ent := el.Value.(*replayEntry)
		ent.ts = now
		r.lru.MoveToBack(el)
	} else {
		// New entry
		r.seen[n] = r.lru.PushBack(&replayEntry{key: n, ts: now})
	}

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

	// Capacity enforcement - O(1) eviction
	if r.cap > 0 && len(r.seen) > r.cap {
		el := r.lru.Front()
		if el != nil {
			ent := el.Value.(*replayEntry)
			delete(r.seen, ent.key)
			r.lru.Remove(el)
			if r.metrics != nil {
				r.metrics.IncReplayStoreEvictions()
			}
		}
	}
}

// Size returns current number of active entries.
// Note: This might include some expired entries that haven't been swept yet.
// For accuracy, we could scan, but that returns to O(N).
// Given "Size()" is often for metrics, returning raw map size is O(1) and acceptable if we accept slight overcount between sweeps.
// However, original implementation did lazy cleanup. To be safe, let's keep it O(1) and rely on background.
func (r *ReplayNonceStore) Size() int {
	r.mu.Lock()
	defer r.mu.Unlock()
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
	// Build active map (exclude expired - one-time O(N) ok for implementation maintenance)
	now := time.Now()
	active := make(map[string]time.Time, len(r.seen))
	for k, el := range r.seen {
		ent := el.Value.(*replayEntry)
		if now.Sub(ent.ts) <= r.ttl {
			active[k] = ent.ts
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

// IsDurable returns true if the store is backed by a write-ahead log (WAL).
func (r *ReplayNonceStore) IsDurable() bool {
	return r.wal != nil
}

// Close closes the underlying WAL store if present and stops background cleanup.
func (r *ReplayNonceStore) Close() error {
	close(r.stopCh) // Stop background cleanup
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.wal != nil {
		return r.wal.Close()
	}
	return nil
}

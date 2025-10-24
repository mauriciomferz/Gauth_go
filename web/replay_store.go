package web

import (
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/replay"
)

// ReplayNonceStore provides minimal in-memory nonce/JTI replay protection for token issuance.
// Not production-ready: single-process only, periodic TTL sweep on access.
type ReplayNonceStore struct {
	 mu   sync.Mutex
	 ttl  time.Duration
	 seen map[string]time.Time
	 cap  int // optional capacity limit; evict oldest on insert when exceeded
	 wal  *replay.WALStore
}

func NewReplayNonceStore(ttl time.Duration) *ReplayNonceStore {
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
		 }
	 }
	 store := &ReplayNonceStore{ttl: ttl, seen: make(map[string]time.Time), cap: cap, wal: wal}
	 // Recover from WAL if available
	 if wal != nil {
		 wal.Recover(func(rec replay.WALRecord) error {
			 if rec.Op == "Record" {
				 store.seen[string(rec.Key)] = time.Unix(rec.TS, 0)
			 }
			 return nil
		 })
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
		 r.wal.AppendRecord(replay.WALRecord{
			 Op: "Record",
			 Key: []byte(n),
			 Value: nil,
			 TS: now.Unix(),
		 })
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
		 r.wal.AppendRecord(replay.WALRecord{
			 Op: "Record",
			 Key: []byte(n),
			 Value: nil,
			 TS: now.Unix(),
		 })
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

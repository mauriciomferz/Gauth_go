package capability

// RB13 capability snapshot retention & diff support.
// Provides a lightweight in-memory ring of capability registry snapshots keyed by
// deterministic hash. This enables computing diffs against historical states.
// Persistence and signed artifacts can be layered later without changing this surface.

import (
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"sync"
	"time"
)

// Snapshot holds a capability registry state at a specific point in time.
type Snapshot struct {
	Hash         string
	Timestamp    time.Time
	Capabilities []Capability
}

// SnapshotStore retains a bounded history of snapshots (unique by hash).
// When capacity is exceeded the oldest snapshot is evicted.
type SnapshotStore struct {
	mu       sync.RWMutex
	ring     []Snapshot
	index    map[string]int // hash -> slice index
	capacity int
}

// NewSnapshotStore constructs a store with provided capacity (<=0 disables retention).
func NewSnapshotStore(capacity int) *SnapshotStore {
	if capacity < 0 {
		capacity = 0
	}
	return &SnapshotStore{capacity: capacity, index: make(map[string]int)}
}

// RegistryHash computes deterministic hash of capability list (sorted by ID) using ID|version|stable lines.
func RegistryHash(list []Capability) string {
	// Defensive copy so sorting doesn't mutate caller slice.
	cp := make([]Capability, len(list))
	copy(cp, list)
	sort.Slice(cp, func(i, j int) bool { return cp[i].ID < cp[j].ID })
	h := sha256.New()
	for _, c := range cp {
		line := c.ID + "|" + c.Version + "|" + boolToStr(c.Stable) + "\n"
		h.Write([]byte(line))
	}
	return "sha256:" + hex.EncodeToString(h.Sum(nil))
}

func boolToStr(b bool) string {
	if b {
		return "1"
	}
	return "0"
}

// Add ingests a new snapshot if hash not already present. Returns existing snapshot if already stored.
func (s *SnapshotStore) Add(list []Capability, hash string) Snapshot {
	if s == nil {
		return Snapshot{Hash: hash, Timestamp: time.Now().UTC(), Capabilities: list}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if idx, exists := s.index[hash]; exists {
		return s.ring[idx]
	}
	snap := Snapshot{Hash: hash, Timestamp: time.Now().UTC(), Capabilities: append([]Capability{}, list...)}
	if s.capacity <= 0 {
		// retention disabled: replace ring with single latest snapshot
		s.ring = []Snapshot{snap}
		s.index = map[string]int{hash: 0}
		return snap
	}
	s.ring = append(s.ring, snap)
	s.index[hash] = len(s.ring) - 1
	// Evict oldest if over capacity
	if len(s.ring) > s.capacity {
		// Remove index 0
		evicted := s.ring[0]
		delete(s.index, evicted.Hash)
		s.ring = s.ring[1:]
		// Rebuild index (small cost; capacity is expected small <100)
		for i, sn := range s.ring {
			s.index[sn.Hash] = i
		}
	}
	return snap
}

// Get returns snapshot for hash.
func (s *SnapshotStore) Get(hash string) (Snapshot, bool) {
	if s == nil {
		return Snapshot{}, false
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	idx, ok := s.index[hash]
	if !ok {
		return Snapshot{}, false
	}
	return s.ring[idx], true
}

// Latest returns the most recently added snapshot.
func (s *SnapshotStore) Latest() (Snapshot, bool) {
	if s == nil {
		return Snapshot{}, false
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	if len(s.ring) == 0 {
		return Snapshot{}, false
	}
	return s.ring[len(s.ring)-1], true
}

// ListHashes returns hashes in insertion order (oldest first).
func (s *SnapshotStore) ListHashes() []string {
	if s == nil {
		return nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]string, 0, len(s.ring))
	for _, sn := range s.ring {
		out = append(out, sn.Hash)
	}
	return out
}

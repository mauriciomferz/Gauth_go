package limits

// Package limits provides a lightweight persistent numeric counters facility used to track
// system-wide limit consumption (e.g. tokens_issued_total, delegations_revoked_total, policy_revisions_total).
// It is intentionally decoupled from the metrics interface to avoid inflating in-memory atomic surfaces.
// Persistence format is a simple JSON map of string->uint64.

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
)

// Store maintains named uint64 counters with optional persistence.
type Store struct {
	mu       sync.RWMutex
	counters map[string]uint64
	path     string
}

// New creates a new Store; if path is non-empty and exists it is loaded.
func New(path string) (*Store, error) {
	s := &Store{counters: make(map[string]uint64), path: path}
	if path != "" {
		if _, err := os.Stat(path); err == nil {
			if err := s.load(); err != nil {
				return nil, err
			}
		}
	}
	return s, nil
}

// Inc increments the named counter by delta (default 1 when delta==0) returning new value.
func (s *Store) Inc(name string, delta uint64) uint64 {
	if name == "" {
		return 0
	}
	if delta == 0 {
		delta = 1
	}
	s.mu.Lock()
	s.counters[name] += delta
	v := s.counters[name]
	s.mu.Unlock()
	return v
}

// Get returns current value (0 if absent).
func (s *Store) Get(name string) uint64 {
	s.mu.RLock()
	v := s.counters[name]
	s.mu.RUnlock()
	return v
}

// Snapshot returns a shallow copy of current counters.
func (s *Store) Snapshot() map[string]uint64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make(map[string]uint64, len(s.counters))
	for k, v := range s.counters {
		out[k] = v
	}
	return out
}

// Persist saves counters to the configured path (error if path empty).
func (s *Store) Persist() error {
	if s.path == "" {
		return errors.New("limits: no path configured")
	}
	tmp := s.path + ".tmp"
	// Ensure directory exists
	if dir := filepath.Dir(s.path); dir != "" {
		if err := os.MkdirAll(dir, 0o750); err != nil {
			return err
		}
	}
	s.mu.RLock()
	data, err := json.MarshalIndent(s.counters, "", "  ")
	s.mu.RUnlock()
	if err != nil {
		return err
	}
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}

// load restores counters from path.
func (s *Store) load() error {
	b, err := os.ReadFile(s.path)
	if err != nil {
		return err
	}
	var m map[string]uint64
	if err := json.Unmarshal(b, &m); err != nil {
		return err
	}
	if m == nil {
		m = make(map[string]uint64)
	}
	s.mu.Lock()
	s.counters = m
	s.mu.Unlock()
	return nil
}

const SnapshotType = "limits_snapshot"

// LedgerEntry produces a structured object suitable for inclusion in an audit ledger.
// Callers JSON-encode the returned map under a ledger event type (e.g. limits_snapshot).
func (s *Store) LedgerEntry() map[string]any {
	snap := s.Snapshot()
	out := make(map[string]any, len(snap)+1)
	for k, v := range snap {
		out[k] = v
	}
	out["_type"] = SnapshotType
	return out
}

// DefaultPath returns a recommended path under working directory.
func DefaultPath() string { return filepath.Join("tmp", "limits_counters.json") }

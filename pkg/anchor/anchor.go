package anchor

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// AnchorClient defines minimal interface for external anchoring of integrity hashes
// (revocation chain aggregate, policy chain head, etc.).
// Implementations may push hashes to immutable ledgers or timestamping services.
// Future expansion: Anchor(hash string, meta map[string]string) (AnchorRecord, error)
// For now we keep it lean to support discovery wiring and tests.
//
// Error modes:
// - returns error on empty hash
// - returns error if underlying storage fails (memory impl cannot fail)
// - LatestAnchor returns ("", time.Time{}, nil) when no anchors yet
// Concurrency: methods must be safe for concurrent use.
//
// Success criteria for initial prototype:
// - Append anchors preserving chronological order
// - Expose latest anchor hash and timestamp
// - Provide snapshot of total count for metrics/gap reporting
// - Support idempotent duplicate anchoring (same hash again returns existing timestamp)
//
// Edge cases:
// - Empty hash input
// - Very frequent anchoring of identical hash (spam) -> memory impl deduplicates
// - Clock skew: we always use time.Now().UTC()
// - Overflow: unrealistic; slice growth only
//
// Future enhancements (not yet implemented):
// - Expiration / pruning
// - External provider plugin (e.g., blockchain, transparency log)
// - Verification of external receipt / inclusion proof
// - Batch anchoring
// - Asynchronous anchoring queue
// - Metrics instrumentation hooks
//
// AnchorRecord holds anchor event details.
// Timestamp intentionally uses RFC3339Nano when serialized by callers.
// Provider is reserved for external provider name (empty for memory).
// Optional fields (e.g., receipt) can be added later via struct extension.
// Backward compatibility: new fields added with omitempty.
type AnchorRecord struct {
	Hash       string    `json:"hash"`
	AnchoredAt time.Time `json:"anchored_at"`
	Provider   string    `json:"provider,omitempty"`
}

type AnchorClient interface {
	Anchor(hash string) (AnchorRecord, error)
	LatestAnchor() (AnchorRecord, error)
	TotalAnchors() int
}

// MemoryAnchor is an in-memory AnchorClient implementation.
// Not safe for multi-process durability; intended for prototype & testing only.
// Deduplicates by hash: subsequent Anchor(hash) returns existing record.
// Maintains chronological order of first-seen hashes.
type MemoryAnchor struct {
	mu          sync.RWMutex
	order       []string // chronological list of hashes
	records     map[string]AnchorRecord
	persistPath string // optional JSON file path
}

// NewMemoryAnchor constructs an empty memory anchor client.
func NewMemoryAnchor() *MemoryAnchor {
	return &MemoryAnchor{order: make([]string, 0), records: make(map[string]AnchorRecord)}
}

// EnablePersistence sets a file path for persistence and performs initial load (best-effort).
// File format: JSON object {"anchors":[{"hash":"...","anchored_at":"RFC3339Nano"},...]}
func (m *MemoryAnchor) EnablePersistence(path string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.persistPath = path
	// Attempt load if file exists
	if fi, err := os.Stat(path); err == nil && fi.Size() > 0 {
		b, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("anchor load read: %w", err)
		}
		var tmp struct {
			Anchors []AnchorRecord `json:"anchors"`
		}
		if err := json.Unmarshal(b, &tmp); err != nil {
			return fmt.Errorf("anchor load json: %w", err)
		}
		// Rebuild maps preserving chronological order
		m.order = make([]string, 0, len(tmp.Anchors))
		m.records = make(map[string]AnchorRecord, len(tmp.Anchors))
		for _, ar := range tmp.Anchors {
			m.records[ar.Hash] = ar
			m.order = append(m.order, ar.Hash)
		}
	}
	return nil
}

// save persists state if persistPath set (idempotent on error: logs to stderr, returns error for caller to decide).
func (m *MemoryAnchor) save() error {
	if m.persistPath == "" {
		return nil
	}
	// Build snapshot
	list := make([]AnchorRecord, 0, len(m.order))
	for _, h := range m.order {
		list = append(list, m.records[h])
	}
	tmp := struct {
		Anchors []AnchorRecord `json:"anchors"`
	}{Anchors: list}
	b, err := json.MarshalIndent(tmp, "", "  ")
	if err != nil {
		return err
	}
	// Ensure directory exists
	// #nosec G301
	if err := os.MkdirAll(filepath.Dir(m.persistPath), 0o750); err != nil {
		return err
	}
	tmpPath := m.persistPath + ".tmp"
	if err := os.WriteFile(tmpPath, b, 0o600); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, m.persistPath); err != nil {
		return err
	}
	return nil
}

// Anchor appends (or returns existing) anchor record for hash.
func (m *MemoryAnchor) Anchor(hash string) (AnchorRecord, error) {
	if hash == "" {
		return AnchorRecord{}, errors.New("hash required")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if rec, ok := m.records[hash]; ok {
		return rec, nil // idempotent
	}
	rec := AnchorRecord{Hash: hash, AnchoredAt: time.Now().UTC(), Provider: "memory"}
	m.records[hash] = rec
	m.order = append(m.order, hash)
	if err := m.save(); err != nil {
		fmt.Fprintf(os.Stderr, "[anchor] persistence save error: %v\n", err)
	}
	return rec, nil
}

// LatestAnchor returns latest anchored record or empty record when none.
func (m *MemoryAnchor) LatestAnchor() (AnchorRecord, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if len(m.order) == 0 {
		return AnchorRecord{}, nil
	}
	h := m.order[len(m.order)-1]
	return m.records[h], nil
}

// TotalAnchors returns count of unique anchored hashes.
func (m *MemoryAnchor) TotalAnchors() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.order)
}

// Compile-time interface assertion
var _ AnchorClient = (*MemoryAnchor)(nil)

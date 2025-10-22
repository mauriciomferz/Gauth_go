package anchor

// ExternalReceiptStore provides append-only hash-chained persistence for external capability anchoring receipts.
// Format (JSON):
// {
//   "entries": [ {"hash":"...","timestamp":"...","provider":"...","version":1,"latency_seconds":0.123,"prev_hash":"<prior_chain_hash>","chain_hash":"<current_payload_hash>"}, ...],
//   "chain_head":"<hash>",
//   "timestamp":"<file_write_time>"
// }
// ChainHash = sha256(prev_hash || canonical_base_receipt_json). PrevHash empty for first entry.
// Each append rewrites entire file (low volume expected).
// Incremental verification mirrors notary ReceiptStore.

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Verification status constants (mirrors notary receipt store & metrics integrity statuses)
const (
	externalReceiptStatusOK       = "ok"
	externalReceiptStatusMismatch = "mismatch"
	externalReceiptStatusEmpty    = "empty"
)

// ExternalAnchorReceipt is the base receipt emitted by external anchor providers.
// Success is implicit (we only persist successful attempts).
// Timestamp stored with nanosecond precision UTC.
// Version reserved for future schema evolution (start at 1).
// LatencySeconds records observed provider operation latency.
// Provider normalized label (e.g. tsa-stub) matching metrics.
// Hash is provider-returned anchored hash (could be identical to capability registry hash or provider-specific digest).
// NOTE: We do not persist failure attempts to keep audit small; failures recorded via metrics.
// Extensible in future (e.g. provider signature, certificate chain, anchor ledger reference).
// json field order stable via struct marshal for deterministic hashing.
type ExternalAnchorReceipt struct {
	Hash           string  `json:"hash"`
	Timestamp      string  `json:"timestamp"`
	Provider       string  `json:"provider"`
	Version        int     `json:"version"`
	LatencySeconds float64 `json:"latency_seconds"`
}

// StoredExternalAnchorReceipt extends ExternalAnchorReceipt with hash-chain linkage.
// PrevHash is previous entry's chain hash (empty for first).
// ChainHash is sha256(prev_hash || canonical_json(base_with_prev_hash)).
// We exclude chain_hash from the canonical payload used for hashing.
type StoredExternalAnchorReceipt struct {
	ExternalAnchorReceipt
	PrevHash  string `json:"prev_hash"`
	ChainHash string `json:"chain_hash"`
}

// ExternalReceiptStore concurrency-safe store.
type ExternalReceiptStore struct {
	mu               sync.RWMutex
	path             string
	entries          []StoredExternalAnchorReceipt
	headHash         string
	lastVerifiedLen  int    // number of entries verified up to headVerifiedHash
	headVerifiedHash string // chain hash at last verification point
}

// NewExternalReceiptStore creates store bound to path (may not yet exist).
func NewExternalReceiptStore(path string) *ExternalReceiptStore {
	return &ExternalReceiptStore{path: path}
}

// Load loads existing store file if present.
func (rs *ExternalReceiptStore) Load() error {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	if rs.path == "" {
		return errors.New("external receipt store path empty")
	}
	// Ensure parent directory exists (it may not yet if path configured but never written).
	if err := os.MkdirAll(filepath.Dir(rs.path), 0o755); err != nil {
		return err
	}
	b, err := os.ReadFile(rs.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	var raw struct {
		Entries   []StoredExternalAnchorReceipt `json:"entries"`
		ChainHead string                        `json:"chain_head"`
	}
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	rs.entries = raw.Entries
	rs.headHash = raw.ChainHead
	return nil
}

// Append persists a successful external anchor receipt with chain linkage.
func (rs *ExternalReceiptStore) Append(r ExternalAnchorReceipt) (StoredExternalAnchorReceipt, error) {
	if r.Hash == "" {
		return StoredExternalAnchorReceipt{}, errors.New("receipt hash empty")
	}
	rs.mu.Lock()
	defer rs.mu.Unlock()
	if rs.path == "" {
		return StoredExternalAnchorReceipt{}, errors.New("external receipt store path empty")
	}
	// Create parent directory if missing to avoid transient write failures under concurrent test temp dir creation.
	if err := os.MkdirAll(filepath.Dir(rs.path), 0o755); err != nil {
		return StoredExternalAnchorReceipt{}, err
	}
	sr := StoredExternalAnchorReceipt{ExternalAnchorReceipt: r, PrevHash: rs.headHash}
	base := struct {
		Hash           string  `json:"hash"`
		Timestamp      string  `json:"timestamp"`
		Provider       string  `json:"provider"`
		Version        int     `json:"version"`
		LatencySeconds float64 `json:"latency_seconds"`
		PrevHash       string  `json:"prev_hash"`
	}{Hash: r.Hash, Timestamp: r.Timestamp, Provider: r.Provider, Version: r.Version, LatencySeconds: r.LatencySeconds, PrevHash: sr.PrevHash}
	enc, err := json.Marshal(base)
	if err != nil {
		return StoredExternalAnchorReceipt{}, err
	}
	h := sha256.Sum256(append([]byte(sr.PrevHash), enc...))
	sr.ChainHash = fmt.Sprintf("%x", h[:])
	rs.entries = append(rs.entries, sr)
	rs.headHash = sr.ChainHash
	// Serialize entire file
	out := struct {
		Entries   []StoredExternalAnchorReceipt `json:"entries"`
		ChainHead string                        `json:"chain_head"`
		Timestamp string                        `json:"timestamp"`
	}{Entries: rs.entries, ChainHead: rs.headHash, Timestamp: time.Now().UTC().Format(time.RFC3339Nano)}
	buf, err := json.Marshal(out)
	if err != nil {
		return StoredExternalAnchorReceipt{}, err
	}
	tmpPath := rs.path + ".tmp"
	if err := os.WriteFile(tmpPath, buf, 0o600); err != nil {
		return StoredExternalAnchorReceipt{}, err
	}
	if err := os.Rename(tmpPath, rs.path); err != nil {
		return StoredExternalAnchorReceipt{}, err
	}
	return sr, nil
}

// Latest returns most recent stored external anchor receipt (zero Provider if none).
func (rs *ExternalReceiptStore) Latest() StoredExternalAnchorReceipt {
	rs.mu.RLock()
	defer rs.mu.RUnlock()
	if len(rs.entries) == 0 {
		return StoredExternalAnchorReceipt{}
	}
	return rs.entries[len(rs.entries)-1]
}

// Entries returns a copy of all stored receipts.
func (rs *ExternalReceiptStore) Entries() []StoredExternalAnchorReceipt {
	rs.mu.RLock()
	defer rs.mu.RUnlock()
	out := make([]StoredExternalAnchorReceipt, len(rs.entries))
	copy(out, rs.entries)
	return out
}

// VerifyIncremental performs incremental hash-chain verification similar to notarization store.
// Returns status (ok|mismatch|empty), mismatch index (-1 if none), and head hash of verified chain.
func (rs *ExternalReceiptStore) VerifyIncremental() (string, int, string) {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	n := len(rs.entries)
	if n == 0 {
		rs.lastVerifiedLen = 0
		rs.headVerifiedHash = ""
		return externalReceiptStatusEmpty, -1, ""
	}
	start := 0
	if rs.lastVerifiedLen > 0 && rs.lastVerifiedLen <= n {
		if rs.entries[rs.lastVerifiedLen-1].ChainHash == rs.headVerifiedHash {
			start = rs.lastVerifiedLen
		} else {
			start = 0 // fallback full scan
		}
	}
	prev := ""
	if start > 0 {
		prev = rs.entries[start-1].ChainHash
	}
	for i := start; i < n; i++ {
		e := rs.entries[i]
		base := struct {
			Hash           string  `json:"hash"`
			Timestamp      string  `json:"timestamp"`
			Provider       string  `json:"provider"`
			Version        int     `json:"version"`
			LatencySeconds float64 `json:"latency_seconds"`
			PrevHash       string  `json:"prev_hash"`
		}{Hash: e.Hash, Timestamp: e.Timestamp, Provider: e.Provider, Version: e.Version, LatencySeconds: e.LatencySeconds, PrevHash: e.PrevHash}
		enc, err := json.Marshal(base)
		if err != nil {
			return externalReceiptStatusMismatch, i, rs.headHash
		}
		h := sha256.Sum256(append([]byte(e.PrevHash), enc...))
		expected := fmt.Sprintf("%x", h[:])
		if expected != e.ChainHash || (i == 0 && e.PrevHash != "") || (i > 0 && e.PrevHash != prev) {
			return externalReceiptStatusMismatch, i, rs.headHash
		}
		prev = e.ChainHash
	}
	rs.lastVerifiedLen = n
	rs.headVerifiedHash = rs.entries[n-1].ChainHash
	return externalReceiptStatusOK, -1, rs.headVerifiedHash
}

package notary

// ReceiptStore provides append-only hash-chained persistence for notarization receipts.
// Format (JSON):
// {
//   "entries": [
//     {
//       "hash":"...",
//       "timestamp":"...",
//       "provider":"...",
//       "success":true,
//       "latency_seconds":0.123,
//       "prev_hash":"<prior_file_hash>",
//       "chain_hash":"<current_payload_hash>"
//     },
//     ...
//   ],
//   "chain_head":"<hash>"
// }
// Each append rewrites entire file (simple, small volume expected).
// File-level chain_hash computed as sha256(prev_file_chain_hash || payload_without_chain_hash_field).
// This allows external integrity verification while staying minimal.

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"
)

// Receipt verification status constants (shared across incremental verification logic).
const (
	receiptStatusOK       = "ok"
	receiptStatusMismatch = "mismatch"
	receiptStatusEmpty    = "empty"
)

// StoredReceipt extends Receipt with chain metadata for persistence.
type StoredReceipt struct {
	Receipt
	PrevHash  string `json:"prev_hash"`
	ChainHash string `json:"chain_hash"`
	// MerkleRoot is optional; populated only when AGENTAUTH_NOTARY_MERKLE_ENABLED=1.
	// Represents the Merkle root of all receipt leaf hashes up to and including this entry.
	// Leaf hash definition: sha256(json_base_without_chain_hash). For consistency we reuse the
	// same 'enc' bytes used during ChainHash computation (excluding PrevHash/ChainHash itself).
	// This allows external auditors to recompute tree independently if they have the entries array.
	MerkleRoot string `json:"merkle_root,omitempty"`
}

// ReceiptStore is safe for concurrent reads/appends.
type ReceiptStore struct {
	mu       sync.RWMutex
	path     string
	entries  []StoredReceipt
	headHash string
	// incremental verification state
	lastVerifiedLen  int    // number of entries verified up to headVerifiedHash
	headVerifiedHash string // chain hash at last verification point
	// incremental merkle state (demo optimization): stores leaf hashes so we avoid
	// re-serializing prior receipts on each append; we still recompute full root from
	// leaf hashes for now. Future work: maintain subtree partials for O(log n) updates.
	merkleLeafHashes  [][32]byte
	merkleInitialized bool
}

// NewReceiptStore creates a store bound to a path (may not exist yet).
func NewReceiptStore(path string) *ReceiptStore {
	return &ReceiptStore{path: path}
}

// Load loads existing file if present.
func (rs *ReceiptStore) Load() error {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	if rs.path == "" {
		return errors.New("receipt store path empty")
	}
	b, err := os.ReadFile(rs.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	var raw struct {
		Entries   []StoredReceipt `json:"entries"`
		ChainHead string          `json:"chain_head"`
	}
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	rs.entries = raw.Entries
	rs.headHash = raw.ChainHead
	return nil
}

// Append adds a new receipt (must be Success==true) computing hash chain and persisting.
func (rs *ReceiptStore) Append(r Receipt) (StoredReceipt, error) {
	if !r.Success {
		return StoredReceipt{}, errors.New("cannot append unsuccessful receipt")
	}
	rs.mu.Lock()
	defer rs.mu.Unlock()
	if rs.path == "" {
		return StoredReceipt{}, errors.New("receipt store path empty")
	}
	// Build stored receipt without chain fields to compute hash.
	sr := StoredReceipt{Receipt: r, PrevHash: rs.headHash}
	// Marshal base portion (excluding ChainHash initially)
	tmp := struct {
		Hash           string  `json:"hash"`
		Timestamp      string  `json:"timestamp"`
		Provider       string  `json:"provider"`
		Version        int     `json:"version"`
		Success        bool    `json:"success"`
		LatencySeconds float64 `json:"latency_seconds"`
		PrevHash       string  `json:"prev_hash"`
	}{
		Hash:           r.Hash,
		Timestamp:      r.Timestamp,
		Provider:       r.Provider,
		Version:        r.Version,
		Success:        r.Success,
		LatencySeconds: r.LatencySeconds,
		PrevHash:       sr.PrevHash,
	}
	enc, err := json.Marshal(tmp)
	if err != nil {
		return StoredReceipt{}, err
	}
	h := sha256.Sum256(append([]byte(sr.PrevHash), enc...))
	sr.ChainHash = fmt.Sprintf("%x", h[:])
	merkleEnabled := os.Getenv("AGENTAUTH_NOTARY_MERKLE_ENABLED") == "1"
	if merkleEnabled {
		// Initialize leaf hash cache if first time enabling after prior disabled appends.
		if !rs.merkleInitialized {
			for _, e := range rs.entries {
				tmp := struct {
					Hash           string  `json:"hash"`
					Timestamp      string  `json:"timestamp"`
					Provider       string  `json:"provider"`
					Version        int     `json:"version"`
					Success        bool    `json:"success"`
					LatencySeconds float64 `json:"latency_seconds"`
					PrevHash       string  `json:"prev_hash"`
				}{
					Hash:           e.Hash,
					Timestamp:      e.Timestamp,
					Provider:       e.Provider,
					Version:        e.Version,
					Success:        e.Success,
					LatencySeconds: e.LatencySeconds,
					PrevHash:       e.PrevHash,
				}
				baseBytes, marshalErr := json.Marshal(tmp)
				if marshalErr != nil {
					return StoredReceipt{}, marshalErr
				}
				rs.merkleLeafHashes = append(rs.merkleLeafHashes, sha256.Sum256(baseBytes))
			}
			rs.merkleInitialized = true
		}
		// Append new leaf hash using existing serialization 'enc'
		rs.merkleLeafHashes = append(rs.merkleLeafHashes, sha256.Sum256(enc))
		root := computeMerkleRoot(rs.merkleLeafHashes)
		sr.MerkleRoot = fmt.Sprintf("%x", root[:])
	}
	// Append and persist full set.
	rs.entries = append(rs.entries, sr)
	rs.headHash = sr.ChainHash
	// Serialize entire file.
	out := struct {
		Entries   []StoredReceipt `json:"entries"`
		ChainHead string          `json:"chain_head"`
		Timestamp string          `json:"timestamp"`
	}{
		Entries:   rs.entries,
		ChainHead: rs.headHash,
		Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
	}
	buf, err := json.Marshal(out)
	if err != nil {
		return StoredReceipt{}, err
	}
	tmpPath := rs.path + ".tmp"
	if err := os.WriteFile(tmpPath, buf, 0o600); err != nil {
		return StoredReceipt{}, err
	}
	if err := os.Rename(tmpPath, rs.path); err != nil {
		return StoredReceipt{}, err
	}
	return sr, nil
}

// Latest returns latest stored receipt (zero Provider if none).
func (rs *ReceiptStore) Latest() StoredReceipt {
	rs.mu.RLock()
	defer rs.mu.RUnlock()
	if len(rs.entries) == 0 {
		return StoredReceipt{}
	}
	return rs.entries[len(rs.entries)-1]
}

// Entries returns copy of all entries.
func (rs *ReceiptStore) Entries() []StoredReceipt {
	rs.mu.RLock()
	defer rs.mu.RUnlock()
	out := make([]StoredReceipt, len(rs.entries))
	copy(out, rs.entries)
	return out
}

// VerifyIncremental performs incremental hash-chain verification. It verifies only new entries
// since the last successful verification if the previously verified head hash still matches
// the current entries prefix. On mismatch or if state diverges it falls back to a full scan.
// Returns status (ok|mismatch|empty), mismatch index (-1 if none), and head hash of verified chain.
func (rs *ReceiptStore) VerifyIncremental() (string, int, string) {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	n := len(rs.entries)
	if n == 0 {
		rs.lastVerifiedLen = 0
		rs.headVerifiedHash = ""
		return receiptStatusEmpty, -1, ""
	}
	// Determine if we can incremental: lengths must be >= previous and the previous
	// head hash must match the chain hash at lastVerifiedLen-1.
	start := 0
	if rs.lastVerifiedLen > 0 && rs.lastVerifiedLen <= n {
		if rs.entries[rs.lastVerifiedLen-1].ChainHash == rs.headVerifiedHash {
			start = rs.lastVerifiedLen // safe to verify tail only
		} else {
			// previous state invalidated (maybe rewrite); force full scan
			start = 0
		}
	}
	prev := ""
	if start > 0 {
		// resume from prior verified tail
		prev = rs.entries[start-1].ChainHash
	}
	for i := start; i < n; i++ {
		e := rs.entries[i]
		// Reconstruct base portion used for hashing (mirroring Append logic)
		tmp := struct {
			Hash           string  `json:"hash"`
			Timestamp      string  `json:"timestamp"`
			Provider       string  `json:"provider"`
			Version        int     `json:"version"`
			Success        bool    `json:"success"`
			LatencySeconds float64 `json:"latency_seconds"`
			PrevHash       string  `json:"prev_hash"`
		}{
			Hash:           e.Hash,
			Timestamp:      e.Timestamp,
			Provider:       e.Provider,
			Version:        e.Version,
			Success:        e.Success,
			LatencySeconds: e.LatencySeconds,
			PrevHash:       e.PrevHash,
		}
		enc, err := json.Marshal(tmp)
		if err != nil {
			return receiptStatusMismatch, i, rs.headHash
		} // serialization failure implies data corruption
		h := sha256.Sum256(append([]byte(e.PrevHash), enc...))
		expected := fmt.Sprintf("%x", h[:])
		if expected != e.ChainHash || (i == 0 && e.PrevHash != "") || (i > 0 && e.PrevHash != prev) {
			// mismatch detected
			return receiptStatusMismatch, i, rs.headHash
		}
		prev = e.ChainHash
	}
	// successful verification; update incremental state
	rs.lastVerifiedLen = n
	rs.headVerifiedHash = rs.entries[n-1].ChainHash
	return receiptStatusOK, -1, rs.headVerifiedHash
}

// computeMerkleRoot computes a binary Merkle tree root from leaf hashes.
// For an odd number of nodes at a level, the last node is duplicated.
// Leaves are already 32-byte digests. Returns zero hash if no leaves.
func computeMerkleRoot(leaves [][32]byte) [32]byte {
	if len(leaves) == 0 {
		return sha256.Sum256(nil)
	}
	level := leaves
	for len(level) > 1 {
		nextCount := (len(level) + 1) / 2
		next := make([][32]byte, 0, nextCount)
		for i := 0; i < len(level); i += 2 {
			if i+1 < len(level) {
				// pair
				combined := append(level[i][:], level[i+1][:]...)
				next = append(next, sha256.Sum256(combined))
			} else {
				// duplicate last
				combined := append(level[i][:], level[i][:]...)
				next = append(next, sha256.Sum256(combined))
			}
		}
		level = next
	}
	return level[0]
}

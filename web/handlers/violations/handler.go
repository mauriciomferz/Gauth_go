package violations

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"
)

// ViolationProvider abstracts the service providing violation counts.
type ViolationProvider interface {
	ViolationSnapshot() map[string]uint64
}

// Metrics interface for recording violation-related metrics.
type Metrics interface {
	// Add violation specific metrics if needed in future
}

// Handler manages violation tracking, history, and persistence.
type Handler struct {
	mu              sync.Mutex
	history         []SnapshotEntry
	persistencePath string
	prevHash        string
	lastPersist     time.Time

	// External deps
	Service ViolationProvider
	Metrics Metrics
}

// SnapshotEntry represents a point-in-time snapshot of violations.
type SnapshotEntry struct {
	At       time.Time         `json:"at"`
	Snapshot map[string]uint64 `json:"snapshot"`
}

// NewHandler creates a new violation handler.
func NewHandler(service ViolationProvider, metrics Metrics, persistencePath string) *Handler {
	return &Handler{
		Service:         service,
		Metrics:         metrics,
		persistencePath: persistencePath,
	}
}

// copyMap helper
func copyMap(m map[string]uint64) map[string]uint64 {
	out := make(map[string]uint64, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

// Update retrieves snapshot from service, updates history.
func (h *Handler) Update() {
	if h.Service == nil {
		return
	}

	snap := h.Service.ViolationSnapshot()
	h.AddSnapshot(time.Now(), snap)
}

// AddSnapshot appends a snapshot to the history, handling pruning.
func (h *Handler) AddSnapshot(at time.Time, snap map[string]uint64) {
	h.mu.Lock()
	defer h.mu.Unlock()

	clone := copyMap(snap)

	// Don't append if too frequent (limit 1 per second usually handled by caller, but good safeguard)
	if len(h.history) > 0 {
		last := h.history[len(h.history)-1]
		if at.Sub(last.At) < time.Second {
			return
		}
	}

	h.history = append(h.history, SnapshotEntry{At: at, Snapshot: clone})

	// Prune
	if len(h.history) > 3600 { // 1 hour history at 1/sec
		h.history = h.history[1:]
	}
}

// ComputeRates calculates per-minute rates for a given window.
func (h *Handler) ComputeRates(window time.Duration) map[string]float64 {
	h.mu.Lock()
	defer h.mu.Unlock()

	out := make(map[string]float64)
	if len(h.history) < 2 {
		return out
	}
	latest := h.history[len(h.history)-1]
	cut := time.Now().Add(-window)
	baseIdx := -1
	for i := len(h.history) - 1; i >= 0; i-- {
		if h.history[i].At.Before(cut) {
			break
		}
		baseIdx = i
	}
	if baseIdx == -1 {
		baseIdx = 0
	}
	// Adaptation from semantic handler: if window is short, try to capture previous point
	if baseIdx == len(h.history)-1 && baseIdx > 0 {
		baseIdx--
	}

	base := h.history[baseIdx]
	elapsed := latest.At.Sub(base.At).Seconds()
	if elapsed <= 0 {
		return out
	}

	for k, cur := range latest.Snapshot {
		prev := base.Snapshot[k]
		delta := float64(0)
		if cur >= prev {
			delta = float64(cur - prev)
		}
		rate := (delta / elapsed) * 60.0
		if rate < 0 {
			rate = 0
		}
		out[k] = rate
	}
	return out
}

// Stats returns history length (for debugging).
func (h *Handler) Stats() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.history)
}

// Count returns the number of loaded/stored entries for persistence logging.
func (h *Handler) Count() int {
	return h.Stats()
}

// History returns a copy of history (for verification/UI).
func (h *Handler) History() []SnapshotEntry {
	h.mu.Lock()
	defer h.mu.Unlock()
	out := make([]SnapshotEntry, len(h.history))
	copy(out, h.history) // Shallow copy of slice struct, map ref copy (careful)
	// Deep copy maps
	for i := range out {
		out[i].Snapshot = copyMap(h.history[i].Snapshot)
	}
	return out
}

// PersistencePayload matches legacy format for compatibility.
type PersistencePayload struct {
	Counters  map[string]uint64 `json:"counters"`
	Timestamp string            `json:"timestamp"`
}

// PersistenceWrapper matches legacy format.
type PersistenceWrapper struct {
	Payload  json.RawMessage `json:"payload"`
	PrevHash string          `json:"prev_hash"`
	Hash     string          `json:"hash"`
}

// RestoreViolationsCallback interface for restoring state to service.
// The handler manages history, but the service manages the actual counters.
type RestoreViolationsCallback interface {
	RestoreViolations(counters map[string]uint64)
}

// Load restores violations from persistence file.
// Since counters reside in the Service (Repo/Limits), we need to callback to restore them.
// We can assert Service implements RestoreViolationsCallback.
func (h *Handler) Load() error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.persistencePath == "" {
		return nil
	}

	data, err := os.ReadFile(h.persistencePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	var w PersistenceWrapper
	if err := json.Unmarshal(data, &w); err != nil {
		return err
	}

	h.prevHash = w.Hash

	// Payload is JSON of PersistencePayload
	var p PersistencePayload
	if err := json.Unmarshal(w.Payload, &p); err != nil {
		return err
	}

	// Restore to service if possible
	if restorer, ok := h.Service.(RestoreViolationsCallback); ok {
		restorer.RestoreViolations(p.Counters)
	}

	return nil
}

// Save persists current snapshot with integrity hash.
func (h *Handler) Save() error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.persistencePath == "" || h.Service == nil {
		return nil
	}

	// Counters come from service
	counters := h.Service.ViolationSnapshot()

	payloadObj := PersistencePayload{
		Counters:  counters,
		Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
	}
	payloadBytes, err := json.Marshal(payloadObj)
	if err != nil {
		return err
	}

	// Calculate hash: SHA256(prevHash + payload)
	hashSum := sha256.Sum256(append([]byte(h.prevHash), payloadBytes...))
	curHash := fmt.Sprintf("%x", hashSum)

	w := PersistenceWrapper{
		Payload:  payloadBytes,
		PrevHash: h.prevHash,
		Hash:     curHash,
	}

	finalBytes, err := json.MarshalIndent(w, "", "  ")
	if err != nil {
		return err
	}

	tmp := h.persistencePath + ".tmp"
	if err := os.WriteFile(tmp, finalBytes, 0644); err != nil {
		return err
	}
	if err := os.Rename(tmp, h.persistencePath); err != nil {
		return err
	}

	// Update state
	h.prevHash = curHash
	h.lastPersist = time.Now()

	return nil
}

// VerifyPersistence checks integrity of the persistence file.
func (h *Handler) VerifyPersistence() (status string, details map[string]any) {
	h.mu.Lock()
	defer h.mu.Unlock()

	details = make(map[string]any)
	if h.persistencePath == "" {
		return "unconfigured", details
	}

	data, err := os.ReadFile(h.persistencePath)
	if err != nil {
		return "read_failed", map[string]any{"error": err.Error()}
	}

	var w PersistenceWrapper
	if err := json.Unmarshal(data, &w); err != nil {
		return "invalid_json", details
	}

	// Canonicalize payload (compact) before hashing
	dst := &bytes.Buffer{}
	if err := json.Compact(dst, w.Payload); err != nil {
		return "invalid_json_payload", details
	}
	canonicalPayload := dst.Bytes()

	recomputed := fmt.Sprintf("%x", sha256.Sum256(append([]byte(w.PrevHash), canonicalPayload...)))
	integrity := "ok"
	if recomputed != w.Hash {
		integrity = "mismatch"
	}
	details["expected"] = w.Hash
	details["recomputed"] = recomputed
	details["prev_hash"] = w.PrevHash

	// Decode payload for inspection
	var p PersistencePayload
	if err := json.Unmarshal(w.Payload, &p); err == nil {
		details["timestamp"] = p.Timestamp
	}

	return integrity, details
}

package semantic

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"sync"
	"time"

	"github.com/mauriciomferz/AgentAuth/internal/anchor"
	"github.com/mauriciomferz/AgentAuth/pkg/ledger"
)

// AAP001Service abstracts the external PoA service
type AAP001Service interface {
	SemanticSnapshot() map[string]uint64
}

// Metrics interface for recording semantic anomaly stats
type Metrics interface {
	RecordSemanticAnomaly(key string, score float64)
}

// anomalyPersist defines persistence format for EWMA semantic anomaly stats.
type anomalyPersist struct {
	Mean  float64 `json:"mean"`
	M2    float64 `json:"m2"`
	Count int     `json:"count"`
}

// Handler manages semantic anomaly detection state and calculations.
type Handler struct {
	mu      sync.Mutex
	ewma    map[string]*anomalyPersist
	scores  map[string]float64
	history []struct {
		At       time.Time
		Snapshot map[string]uint64
	}
	persistencePath string
	prevHash        string

	// Ledger storage for archival rotation
	Ledger ledger.Store
	// External anchoring provider
	AnchorProvider    anchor.Provider
	LastAnchorReceipt anchor.Receipt

	// Last archival/anchoring timestamps
	LastArchive     time.Time
	ArchiveInterval time.Duration
	LastAnchor      time.Time
	AnchorInterval  time.Duration

	// External deps
	Service AAP001Service
	Metrics Metrics

	// Config
	WarmupSamples int
}

// SnapshotEntry represents a point-in-time snapshot.
type SnapshotEntry struct {
	At       time.Time         `json:"at"`
	Snapshot map[string]uint64 `json:"snapshot"`
}

// NewHandler creates a new semantic anomaly handler.
func NewHandler(service AAP001Service, metrics Metrics, persistencePath string) *Handler {
	return &Handler{
		Service:         service,
		Metrics:         metrics,
		persistencePath: persistencePath,
		ewma:            make(map[string]*anomalyPersist),
		scores:          make(map[string]float64),
		WarmupSamples:   10,
		ArchiveInterval: 1 * time.Hour,
		AnchorInterval:  24 * time.Hour,
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

// Update retrieves snapshot from service, updates history and calculates anomalies.
func (h *Handler) Update() {
	var ss map[string]uint64
	if h.Service != nil {
		ss = h.Service.SemanticSnapshot()
	}

	h.mu.Lock()
	// Manual unlock later to avoid deadlock/panic with archival calls

	if ss != nil && os.Getenv("GAUTH_SEMANTIC_HISTORY_DISABLE") != "1" {
		now := time.Now()
		h.appendHistory(now, ss)
	}

	// Anomaly detection logic
	// Compute 60s rates
	rates := h.computeRatesLocked(60 * time.Second)

	for k, v := range rates {
		fVal := v
		e, exists := h.ewma[k]
		if !exists {
			e = &anomalyPersist{}
			h.ewma[k] = e
		}

		// Check for anomaly against accumulated history
		if e.Count > h.WarmupSamples {
			variance := e.M2 / float64(e.Count-1)
			stdDev := math.Sqrt(variance)
			if stdDev > 0 {
				zScore := (fVal - e.Mean) / stdDev
				if zScore > 3.0 || zScore < -3.0 { // 3-sigma anomaly
					// Track anomaly score
					h.scores[k] = zScore
					if h.Metrics != nil {
						h.Metrics.RecordSemanticAnomaly(k, zScore)
					}
				} else {
					delete(h.scores, k)
				}
			}
		}

		// Update Welford's algorithm
		e.Count++
		delta := fVal - e.Mean
		e.Mean += delta / float64(e.Count)
		delta2 := fVal - e.Mean
		e.M2 += delta * delta2
	}
	if err := h.saveLocked(); err != nil {
		fmt.Fprintf(os.Stderr, "[semantic] save failed: %v\n", err)
	}

	shouldArchive := h.Ledger != nil && (h.LastArchive.IsZero() || time.Since(h.LastArchive) >= h.ArchiveInterval)
	shouldAnchor := h.AnchorProvider != nil && h.Ledger != nil && (h.LastAnchor.IsZero() || time.Since(h.LastAnchor) >= h.AnchorInterval)
	h.mu.Unlock()

	if shouldArchive {
		h.archiveToLedger()
	}

	if shouldAnchor {
		h.anchorLedgerTip()
	}
}

// archiveToLedger appends current EWMA state to persistent ledger.
func (h *Handler) archiveToLedger() {
	h.mu.Lock()
	if h.Ledger == nil {
		h.mu.Unlock()
		return
	}
	// Take a snapshot of current EWMA state
	metadata := make(map[string]interface{})
	for k, v := range h.ewma {
		metadata[k] = v
	}
	h.mu.Unlock()

	entry := &ledger.Entry{
		ID:       fmt.Sprintf("semantic-%d", time.Now().Unix()),
		TS:       time.Now().UTC(),
		Type:     "semantic_snapshot",
		Subject:  "ewma_stats",
		Object:   "semantic_handler",
		Metadata: metadata,
	}

	if err := h.Ledger.Append(context.Background(), entry); err != nil {
		fmt.Fprintf(os.Stderr, "[semantic] archival failed: %v\n", err)
		return
	}

	h.mu.Lock()
	h.LastArchive = time.Now()
	h.mu.Unlock()
}

// anchorLedgerTip push latest ledger hash to external anchor provider.
func (h *Handler) anchorLedgerTip() {
	h.mu.Lock()
	if h.Ledger == nil || h.AnchorProvider == nil {
		h.mu.Unlock()
		return
	}
	h.mu.Unlock()

	tip := ledger.ChainTip(h.Ledger)
	if tip == "" {
		return
	}

	receipt, err := h.AnchorProvider.Anchor(tip)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[semantic] anchoring failed: %v\n", err)
		return
	}

	h.mu.Lock()
	h.LastAnchorReceipt = receipt
	h.LastAnchor = time.Now()
	h.mu.Unlock()
}

// appendHistory appends a snapshot to the history, handling pruning.
// Must be called with h.mu locked.
func (h *Handler) appendHistory(at time.Time, snap map[string]uint64) {
	appendAllowed := true
	if len(h.history) > 0 {
		last := h.history[len(h.history)-1]
		if at.Sub(last.At) < time.Second {
			appendAllowed = false
		}
	}
	if appendAllowed {
		clone := copyMap(snap)
		h.history = append(h.history, struct {
			At       time.Time
			Snapshot map[string]uint64
		}{At: at, Snapshot: clone})

		// Prune
		if len(h.history) > 3600 { // 1 hour history at 1/sec
			h.history = h.history[1:]
		}
	}
}

// ComputeRates calculates per-minute rates for a given window.
func (h *Handler) ComputeRates(window time.Duration) map[string]float64 {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.computeRatesLocked(window)
}

// AddSnapshot appends a snapshot with a specific timestamp (for testing/simulation).
func (h *Handler) AddSnapshot(at time.Time, snap map[string]uint64) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.appendHistory(at, snap)
}

// History returns a copy of the recorded semantic snapshot history.
func (h *Handler) History() []SnapshotEntry {
	h.mu.Lock()
	defer h.mu.Unlock()
	out := make([]SnapshotEntry, len(h.history))
	for i, v := range h.history {
		out[i] = SnapshotEntry{At: v.At, Snapshot: copyMap(v.Snapshot)}
	}
	return out
}

// computeRatesLocked internal helper
func (h *Handler) computeRatesLocked(window time.Duration) map[string]float64 {
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
	// If the window only contains the latest point (baseIdx == last), try to use the previous point
	// to establish a valid interval covering part of the window.
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

// Stats returns counts of internal anomaly tracking maps (for debugging).
func (h *Handler) Stats() (ewmaEntries int, scoreEntries int) {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.ewma), len(h.scores)
}

// Count returns the number of loaded/stored entries for persistence logging.
func (h *Handler) Count() int {
	ewma, _ := h.Stats()
	return ewma
}

// Scores returns a copy of current anomaly Z-scores.
func (h *Handler) Scores() map[string]float64 {
	h.mu.Lock()
	defer h.mu.Unlock()
	out := make(map[string]float64, len(h.scores))
	for k, v := range h.scores {
		out[k] = v
	}
	return out
}

// Load restores semantic counters from the persistence file.
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

	var loaded map[string]*anomalyPersist

	type wrapper struct {
		Payload  json.RawMessage `json:"payload"`
		PrevHash string          `json:"prev_hash"`
		Hash     string          `json:"hash"`
	}
	var w wrapper
	// Try to unmarshal as wrapper first
	if err := json.Unmarshal(data, &w); err == nil && len(w.Payload) > 0 {
		h.prevHash = w.Hash
		if err := json.Unmarshal(w.Payload, &loaded); err != nil {
			return err
		}
	} else {
		// Fallback: legacy format (just the map)
		if err := json.Unmarshal(data, &loaded); err != nil {
			return err
		}
	}

	if h.ewma == nil {
		h.ewma = make(map[string]*anomalyPersist)
	}
	for k, v := range loaded {
		h.ewma[k] = v
	}
	return nil
}

// Save writes semantic counters snapshot to the persistence file.
func (h *Handler) Save() error {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.saveLocked()
}

func (h *Handler) saveLocked() error {
	if h.persistencePath == "" || len(h.ewma) == 0 {
		return nil
	}

	payloadBytes, err := json.Marshal(h.ewma)
	if err != nil {
		return err
	}

	// Wrapper for hashing
	type wrapper struct {
		Payload  json.RawMessage `json:"payload"`
		PrevHash string          `json:"prev_hash"`
		Hash     string          `json:"hash"`
	}

	// Calculate hash (chaining previous hash + payload)
	hashSum := sha256.Sum256(append([]byte(h.prevHash), payloadBytes...))
	hash := fmt.Sprintf("%x", hashSum)

	// Optimization: skip write if unchanged
	if hash == h.prevHash {
		return nil
	}

	w := wrapper{
		Payload:  payloadBytes,
		PrevHash: h.prevHash,
		Hash:     hash,
	}

	finalBytes, err := json.MarshalIndent(w, "", "  ")
	if err != nil {
		return err
	}

	// Atomic write
	tmp := h.persistencePath + ".tmp"
	if err := os.WriteFile(tmp, finalBytes, 0600); err != nil {
		return err
	}
	if err := os.Rename(tmp, h.persistencePath); err != nil {
		return err
	}

	h.prevHash = hash
	return nil
}

// VerifyPersistence checks the integrity of the persistence file.
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

	type wrapper struct {
		Payload  json.RawMessage `json:"payload"`
		PrevHash string          `json:"prev_hash"`
		Hash     string          `json:"hash"`
	}
	var w wrapper
	if err := json.Unmarshal(data, &w); err != nil || len(w.Payload) == 0 || w.Hash == "" {
		return "legacy", details
	}

	// Canonicalize payload to compact JSON for verification (ignoring indentation from file)
	var compactPayload bytes.Buffer
	if err := json.Compact(&compactPayload, w.Payload); err != nil {
		return "parse_error", details
	}

	recomputed := fmt.Sprintf("%x", sha256.Sum256(append([]byte(w.PrevHash), compactPayload.Bytes()...)))

	details["expected"] = w.Hash
	details["recomputed"] = recomputed
	details["prev_hash"] = w.PrevHash

	if recomputed != w.Hash {
		return "mismatch", details
	}
	return "ok", details
}

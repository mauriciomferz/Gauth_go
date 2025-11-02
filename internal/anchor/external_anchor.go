package anchor

import (
	"sync"
	"time"
)

// CombinedAnchorReceipt is a prototype receipt for future combined capability+rotation anchoring.
// Kept distinct from ExternalAnchorReceipt to avoid conflicts with existing implementation.
type CombinedAnchorReceipt struct {
    Seq       uint64    `json:"seq"`
    HashHex   string    `json:"hash_hex"`
    Timestamp time.Time `json:"timestamp"`
    Provider  string    `json:"provider"`
}

// CombinedAnchorWriter defines submission for combined anchors (unused placeholder).
type CombinedAnchorWriter interface {
    Submit(hashHex string, ts time.Time) (CombinedAnchorReceipt, error)
    Chain() []CombinedAnchorReceipt
}

// InMemoryCombinedAnchorWriter is an unused stub retained for future expansion.
type InMemoryCombinedAnchorWriter struct {
    mu       sync.Mutex
    provider string
    chain    []CombinedAnchorReceipt
    seq      uint64
}

func NewInMemoryCombinedAnchorWriter(provider string) *InMemoryCombinedAnchorWriter {
    return &InMemoryCombinedAnchorWriter{provider: provider}
}

func (w *InMemoryCombinedAnchorWriter) Submit(hashHex string, ts time.Time) (CombinedAnchorReceipt, error) {
    w.mu.Lock()
    defer w.mu.Unlock()
    w.seq++
    r := CombinedAnchorReceipt{Seq: w.seq, HashHex: hashHex, Timestamp: ts, Provider: w.provider}
    w.chain = append(w.chain, r)
    return r, nil
}

func (w *InMemoryCombinedAnchorWriter) Chain() []CombinedAnchorReceipt {
    w.mu.Lock()
    defer w.mu.Unlock()
    out := make([]CombinedAnchorReceipt, len(w.chain))
    copy(out, w.chain)
    return out
}

package notary

// Notarization handlers - extracted from server_clean.go
// Provides endpoints for combined anchor and notarization receipts.

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	internalNotary "github.com/mauriciomferz/AgentAuth/internal/notary"
)

// CombinedAnchorEntry represents a combined capability+rotation digest entry.
type CombinedAnchorEntry struct {
	Digest       string    `json:"digest"`
	Capability   string    `json:"capability"`
	RotationHead string    `json:"rotation_head"`
	EmittedAt    time.Time `json:"emitted_at"`
}

// ReceiptStoreProvider abstracts the receipt storage operations using internal types.
type ReceiptStoreProvider interface {
	Latest() internalNotary.StoredReceipt
	Entries() []internalNotary.StoredReceipt
}

// Deps defines the dependency surface required for notary handlers.
type Deps interface {
	// Combined anchor dependencies
	GetCapabilityRegistryHash() string
	GetRotationLedgerHeadHash() string

	// Receipt store
	GetReceiptStore() ReceiptStoreProvider
}

// Metrics abstracts optional metrics recording.
type Metrics interface {
	IncCombinedAnchorEmitted()
	IncCombinedAnchorFailures()
	SetReceiptIntegrityStatus(status string)
}

// Handler manages notarization state and provides HTTP handlers.
type Handler struct {
	mu                  sync.Mutex
	combinedAnchorChain []CombinedAnchorEntry
	deps                Deps
	metrics             Metrics
}

// NewHandler creates a new notary handler.
func NewHandler(deps Deps, metrics Metrics) *Handler {
	return &Handler{
		combinedAnchorChain: make([]CombinedAnchorEntry, 0),
		deps:                deps,
		metrics:             metrics,
	}
}

// RegisterRoutes registers notary endpoints on the router.
func (h *Handler) RegisterRoutes(r *gin.Engine) {
	// Combined anchor endpoints
	r.POST("/api/v1/anchor/emitCombined", h.CombinedAnchorEmit)
	r.GET("/api/v1/anchor/chain", h.CombinedAnchorChain)
	r.GET("/api/v1/anchor/verifyChain", h.CombinedAnchorVerify)

	// Notarization receipts endpoints (beta)
	r.GET("/api/v1/beta/notarization/receipts/latest", h.ReceiptLatest)
	r.GET("/api/v1/beta/notarization/receipts", h.ReceiptsChain)
	r.GET("/api/v1/beta/notarization/receipts/verify", h.ReceiptsVerify)
}

// CombinedAnchorEmit emits a combined capability+rotation digest.
func (h *Handler) CombinedAnchorEmit(c *gin.Context) {
	capHash := h.deps.GetCapabilityRegistryHash()
	if capHash == "" {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "capability_hash_unset"})
		return
	}
	rotationHead := h.deps.GetRotationLedgerHeadHash()
	combo := capHash + ":" + rotationHead
	dig := sha256.Sum256([]byte(combo))
	hexDigest := hex.EncodeToString(dig[:])

	entry := CombinedAnchorEntry{
		Digest:       hexDigest,
		Capability:   capHash,
		RotationHead: rotationHead,
		EmittedAt:    time.Now().UTC(),
	}

	h.mu.Lock()
	h.combinedAnchorChain = append(h.combinedAnchorChain, entry)
	h.mu.Unlock()

	if h.metrics != nil {
		h.metrics.IncCombinedAnchorEmitted()
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "combined_hash": hexDigest, "rotation_head": rotationHead})
}

// CombinedAnchorChain returns the in-memory chain entries.
func (h *Handler) CombinedAnchorChain(c *gin.Context) {
	h.mu.Lock()
	out := make([]CombinedAnchorEntry, len(h.combinedAnchorChain))
	copy(out, h.combinedAnchorChain)
	h.mu.Unlock()

	c.JSON(http.StatusOK, gin.H{"success": true, "entries": out})
}

// CombinedAnchorVerify recomputes each digest and reports status.
func (h *Handler) CombinedAnchorVerify(c *gin.Context) {
	h.mu.Lock()
	chainCopy := make([]CombinedAnchorEntry, len(h.combinedAnchorChain))
	copy(chainCopy, h.combinedAnchorChain)
	h.mu.Unlock()

	for _, e := range chainCopy {
		combo := e.Capability + ":" + e.RotationHead
		dig := sha256.Sum256([]byte(combo))
		if hex.EncodeToString(dig[:]) != e.Digest {
			if h.metrics != nil {
				h.metrics.IncCombinedAnchorFailures()
			}
			c.JSON(http.StatusOK, gin.H{"success": true, "status": "mismatch"})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "status": "ok"})
}

// ReceiptLatest returns the latest persisted successful receipt.
func (h *Handler) ReceiptLatest(c *gin.Context) {
	store := h.deps.GetReceiptStore()
	if store == nil {
		c.JSON(http.StatusOK, gin.H{"success": true, "configured": false})
		return
	}
	latest := store.Latest()
	if latest.Timestamp == "" {
		c.JSON(http.StatusOK, gin.H{"success": true, "configured": true, "empty": true})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "configured": true, "empty": false, "receipt": latest})
}

// ReceiptsChain returns a lightweight summary of the receipt chain.
func (h *Handler) ReceiptsChain(c *gin.Context) {
	store := h.deps.GetReceiptStore()
	if store == nil {
		c.JSON(http.StatusOK, gin.H{"success": true, "configured": false})
		return
	}
	entries := store.Entries()
	chain := make([]gin.H, 0, len(entries))
	for _, e := range entries {
		chain = append(chain, gin.H{"hash": e.Hash, "timestamp": e.Timestamp, "provider": e.Provider, "chain_hash": e.ChainHash, "prev_hash": e.PrevHash})
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "configured": true, "total": len(chain), "entries": chain})
}

// ReceiptsVerify verifies the integrity of the receipt chain.
func (h *Handler) ReceiptsVerify(c *gin.Context) {
	store := h.deps.GetReceiptStore()
	if store == nil {
		c.JSON(http.StatusOK, gin.H{"success": true, "configured": false})
		return
	}
	entries := store.Entries()
	if len(entries) == 0 {
		c.JSON(http.StatusOK, gin.H{"success": true, "configured": true, "integrity": "empty", "total": 0})
		return
	}

	prev := ""
	for i, e := range entries {
		// Reconstruct base used for hashing
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
			c.JSON(http.StatusOK, gin.H{"success": true, "configured": true, "integrity": "mismatch", "total": len(entries), "details": gin.H{"mismatch_index": i, "reason": "marshal_error"}})
			return
		}
		expected := fmt.Sprintf("%x", sha256.Sum256(append([]byte(e.PrevHash), enc...)))
		if expected != e.ChainHash || e.PrevHash != prev {
			c.JSON(http.StatusOK, gin.H{"success": true, "configured": true, "integrity": "mismatch", "total": len(entries), "details": gin.H{"mismatch_index": i, "expected": expected, "got": e.ChainHash}})
			return
		}
		prev = expected
	}

	if h.metrics != nil {
		h.metrics.SetReceiptIntegrityStatus("ok")
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "configured": true, "integrity": "ok", "total": len(entries)})
}

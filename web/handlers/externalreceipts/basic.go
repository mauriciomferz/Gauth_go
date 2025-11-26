package externalreceipts

// RB8 External anchor receipt chain endpoints modular extraction.
// Provides GET /api/v1/beta/capabilities/anchor/external/receipts/latest
//          GET /api/v1/beta/capabilities/anchor/external/receipts
//          GET /api/v1/beta/capabilities/anchor/external/receipts/verify
// Mirrors original JSON shapes and integrity verification logic from monolithic server.
// Error taxonomy here is minimal (original endpoints returned 200 with configured flags or mismatch details).

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	anchorint "github.com/mauriciomferz/Gauth_go/internal/anchor"
	"github.com/gin-gonic/gin"
)

const (
	emptyValue = "empty" // matches original key indicating empty chain or receipt
)

// Deps defines dependency surface supplied by BetaServer via accessor methods.
type Deps interface {
	// ExternalReceiptStore returns raw store reference (nil when unconfigured).
	ExternalReceiptStore() any
	// Store accessor helpers (avoid exposing concrete type in interface for simpler test stubs)
	ExternalReceiptStoreLatest() anchorint.StoredExternalAnchorReceipt
	ExternalReceiptStoreEntries() []anchorint.StoredExternalAnchorReceipt
	ExternalReceiptIntegrityStatus() string
	ExternalReceiptLastVerify() time.Time
	SetExternalAnchorReceiptsIntegrity(string)
	SetExternalAnchorReceiptsLastVerifyAge(uint64)
}

// RegisterChain mounts all external receipt chain endpoints.
func RegisterChain(rg *gin.RouterGroup, d Deps) {
	rg.GET("/capabilities/anchor/external/receipts/latest", func(c *gin.Context) { latestHandler(c, d) })
	rg.GET("/capabilities/anchor/external/receipts", func(c *gin.Context) { chainHandler(c, d) })
	rg.GET("/capabilities/anchor/external/receipts/verify", func(c *gin.Context) { verifyHandler(c, d) })
}

// latestHandler returns latest persisted external receipt or empty indicator.
func latestHandler(c *gin.Context, d Deps) {
	if d.ExternalReceiptStore() == nil {
		c.JSON(http.StatusOK, gin.H{"success": true, "configured": false})
		return
	}
	latest := d.ExternalReceiptStoreLatest()
	if latest.Hash == "" {
		c.JSON(http.StatusOK, gin.H{"success": true, "configured": true, emptyValue: true})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "configured": true, emptyValue: false, "receipt": latest})
}

// chainHandler returns summary list of all receipt entries.
func chainHandler(c *gin.Context, d Deps) {
	if d.ExternalReceiptStore() == nil {
		c.JSON(http.StatusOK, gin.H{"success": true, "configured": false})
		return
	}
	entries := d.ExternalReceiptStoreEntries()
	chain := make([]gin.H, 0, len(entries))
	for _, e := range entries {
		chain = append(chain, gin.H{"hash": e.Hash, "timestamp": e.Timestamp, "provider": e.Provider, "chain_hash": e.ChainHash, "prev_hash": e.PrevHash})
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "configured": true, "total": len(chain), "entries": chain})
}

// verifyHandler performs integrity verification identical to original logic.
func verifyHandler(c *gin.Context, d Deps) {
	if d.ExternalReceiptStore() == nil {
		c.JSON(http.StatusOK, gin.H{"success": true, "configured": false})
		return
	}
	entries := d.ExternalReceiptStoreEntries()
	if len(entries) == 0 {
		d.SetExternalAnchorReceiptsIntegrity(emptyValue)
		c.JSON(http.StatusOK, gin.H{"success": true, "configured": true, "integrity": emptyValue, "total": 0})
		return
	}
	prev := ""
	for i, e := range entries {
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
			d.SetExternalAnchorReceiptsIntegrity("mismatch")
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "marshal_failed", "detail": err.Error()})
			return
		}
		expected := fmt.Sprintf("%x", sha256.Sum256(append([]byte(e.PrevHash), enc...)))
		if expected != e.ChainHash || e.PrevHash != prev {
			d.SetExternalAnchorReceiptsIntegrity("mismatch")
			c.JSON(http.StatusOK, gin.H{"success": true, "configured": true, "integrity": "mismatch", "total": len(entries), "details": gin.H{"mismatch_index": i, "expected": expected, "stored": e.ChainHash, "prev_expected": prev, "prev_stored": e.PrevHash}})
			return
		}
		prev = expected
	}
	d.SetExternalAnchorReceiptsIntegrity("ok")
	d.SetExternalAnchorReceiptsLastVerifyAge(0)
	c.JSON(http.StatusOK, gin.H{"success": true, "configured": true, "integrity": "ok", "total": len(entries), "chain_head": prev})
}

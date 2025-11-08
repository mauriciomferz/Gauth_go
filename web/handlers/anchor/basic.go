package anchor

// RB8 Anchor endpoints modular extraction.
// Provides POST /api/v1/beta/capabilities/anchor, GET /anchor/latest, /anchor/material, /anchor/status.
// Mirrors original JSON shapes and error taxonomy from monolithic server to preserve backward compatibility.
// NOTE: respondError duplicated locally to avoid import cycle (same shape as web.respondError).

import (
	"encoding/json"
	"net/http"
	"os"
	"time"

	anchorpkg "github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/anchor"
	"github.com/gin-gonic/gin"
)

// SignedAnchorWrapper duplicated from web package to avoid import cycle.
// Mode=="eddsa" && Signature != "" indicates Ed25519 signature present; Artifact raw bytes must be preserved for verification.
type SignedAnchorWrapper struct {
	Artifact  json.RawMessage `json:"artifact"`
	Kid       string          `json:"kid"`
	Signature string          `json:"signature"`
	Mode      string          `json:"mode"`
}

const sigModeEdDSA = "eddsa"

// Deps defines the server dependency surface required for all four anchor endpoints.
// BetaServer implements this via thin accessor methods.
// Methods kept narrow to reduce coupling; future external receipts extraction can extend.
type Deps interface {
	CapabilityAnchorEnabled() bool
	CapabilityRegistryHash() string
	CapabilityPrevRegistryHash() string
	CapabilityRegistryChangeAt() time.Time
	AnchorClient() interface {
		Anchor(string) (anchorpkg.AnchorRecord, error)
		LatestAnchor() (anchorpkg.AnchorRecord, error)
		TotalAnchors() int
	}
	CapAnchorFilePath() string
	CapAnchorLastWrite() time.Time
	CapAnchorWriteInterval() time.Duration
	CapAnchorAgeSeconds() uint64
	CapAnchorStaleThresholdSeconds() int
	CapAnchorStale() bool
	CapAnchorMetrics() (emitted, skipped, hashChanged, lastWriteUnix uint64, ok bool)
	NotarizationEnabled() bool
	LastNotarizationTime() time.Time
	LastNotarizationReceipt() (hash, timestamp, provider string, success bool)
	ExternalAnchorReceipt() (hash, timestamp, provider string, version int)
}

// RegisterAll mounts all capability anchor endpoints on provided router group.
func RegisterAll(r *gin.RouterGroup, deps Deps) {
	r.POST("/capabilities/anchor", func(c *gin.Context) { postAnchorHandler(c, deps) })
	r.GET("/capabilities/anchor/latest", func(c *gin.Context) { latestAnchorHandler(c, deps) })
	r.GET("/capabilities/anchor/material", func(c *gin.Context) { materialHandler(c, deps) })
	r.GET("/capabilities/anchor/status", func(c *gin.Context) { statusHandler(c, deps) })
}

// POST /capabilities/anchor
func postAnchorHandler(c *gin.Context, d Deps) {
	if !d.CapabilityAnchorEnabled() {
		c.JSON(http.StatusForbidden, gin.H{"success": false, "error": "capability_anchor_disabled", "code": "anchoring_disabled"})
		return
	}
	client := d.AnchorClient()
	if client == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "anchor_client_unavailable", "code": "anchor_client_unavailable"})
		return
	}
	hash := d.CapabilityRegistryHash()
	if hash == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "registry_hash_empty", "code": "registry_hash_empty"})
		return
	}
	rec, err := client.Anchor(hash)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "anchor_failure", "code": "anchor_failure", "detail": err.Error()})
		return
	}
	payload := gin.H{"success": true, "hash": rec.Hash, "anchored_at": rec.AnchoredAt.Format(time.RFC3339), "total": client.TotalAnchors()}
	prev := d.CapabilityPrevRegistryHash()
	if prev != "" {
		payload["previous_hash"] = prev
	}
	chAt := d.CapabilityRegistryChangeAt()
	if !chAt.IsZero() {
		payload["registry_last_changed_at"] = chAt.Format(time.RFC3339)
	}
	c.JSON(http.StatusOK, payload)
}

// GET /capabilities/anchor/latest
func latestAnchorHandler(c *gin.Context, d Deps) {
	client := d.AnchorClient()
	if client == nil {
		c.JSON(http.StatusOK, gin.H{"success": true, "anchored": false, "latest": nil})
		return
	}
	latest, _ := client.LatestAnchor()
	if latest.Hash == "" {
		c.JSON(http.StatusOK, gin.H{"success": true, "anchored": false, "latest": nil, "total": client.TotalAnchors()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "anchored": true, "latest": gin.H{"hash": latest.Hash, "anchored_at": latest.AnchoredAt.Format(time.RFC3339)}, "total": client.TotalAnchors(), "capability_registry_hash": d.CapabilityRegistryHash()})
}

// GET /capabilities/anchor/material
func materialHandler(c *gin.Context, d Deps) {
	path := d.CapAnchorFilePath()
	if path == "" {
		c.JSON(http.StatusOK, gin.H{"success": true, "configured": false, "emitted": false, "registry_hash": d.CapabilityRegistryHash()})
		return
	}
	b, err := os.ReadFile(path)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "read_failed", "detail": err.Error()})
		return
	}
	// Attempt signed wrapper decode first (preserve raw Artifact bytes) else generic decode.
	var wrapper SignedAnchorWrapper
	if err := json.Unmarshal(b, &wrapper); err == nil && wrapper.Mode == sigModeEdDSA && wrapper.Signature != "" && len(wrapper.Artifact) > 0 {
		c.JSON(http.StatusOK, gin.H{"success": true, "configured": true, "emitted": len(b) > 0, "path": path, "size": len(b), "artifact": wrapper, "registry_hash": d.CapabilityRegistryHash(), "last_write": d.CapAnchorLastWrite().UTC().Format(time.RFC3339Nano)})
		return
	}
	var js any
	_ = json.Unmarshal(b, &js)
	c.JSON(http.StatusOK, gin.H{"success": true, "configured": true, "emitted": len(b) > 0, "path": path, "size": len(b), "artifact": js, "registry_hash": d.CapabilityRegistryHash(), "last_write": d.CapAnchorLastWrite().UTC().Format(time.RFC3339Nano)})
}

// GET /capabilities/anchor/status
func statusHandler(c *gin.Context, d Deps) {
	configured := d.CapAnchorFilePath() != "" && d.CapAnchorWriteInterval() > 0
	lastWrite := ""
	if !d.CapAnchorLastWrite().IsZero() {
		lastWrite = d.CapAnchorLastWrite().UTC().Format(time.RFC3339Nano)
	}
	payload := gin.H{
		"success":                 true,
		"configured":              configured,
		"last_write":              lastWrite,
		"registry_hash":           d.CapabilityRegistryHash(),
		"age_seconds":             d.CapAnchorAgeSeconds(),
		"stale_threshold_seconds": d.CapAnchorStaleThresholdSeconds(),
		"stale":                   d.CapAnchorStale(),
	}
	if e, s, h, lw, ok := d.CapAnchorMetrics(); ok {
		payload["emitted_total"] = e
		payload["skipped_total"] = s
		payload["hash_changed_total"] = h
		if lw > 0 {
			payload["last_write_unix"] = lw
		}
	}
	// Notarization summary (prototype) mirrors original exposure.
	if d.NotarizationEnabled() {
		lt := d.LastNotarizationTime()
		if !lt.IsZero() {
			hash, ts, provider, success := d.LastNotarizationReceipt()
			payload["last_notarized_at"] = lt.UTC().Format(time.RFC3339Nano)
			payload["notarized_age_seconds"] = uint64(time.Since(lt).Seconds())
			payload["notarization_receipt"] = gin.H{"hash": hash, "timestamp": ts, "provider": provider, "success": success}
		} else {
			payload["notarized_age_seconds"] = 0
			hash, ts, provider, success := d.LastNotarizationReceipt()
			if provider != "" || hash != "" || ts != "" || success { // provider hint when absent
				payload["notarization_provider"] = provider
			}
		}
	}
	// External anchor receipt exposure
	if h, ts, provider, version := d.ExternalAnchorReceipt(); h != "" {
		payload["external_anchor_receipt"] = gin.H{"hash": h, "timestamp": ts, "provider": provider, "version": version}
	}
	c.JSON(http.StatusOK, payload)
}

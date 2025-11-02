package audit

// RB8 Capability audit chain endpoints modular extraction.
// Provides GET /api/v1/beta/capabilities/audit/verify and POST /api/v1/beta/capabilities/audit/anchor.
// Mirrors original JSON shapes and error taxonomy from monolithic server to preserve backward compatibility.
// NOTE: respondError duplicated locally (same shape as global) to avoid import cycles.

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	anchorpkg "github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/anchor"
	"github.com/gin-gonic/gin"
)

// Deps defines the narrow dependency surface required for capability audit handlers.
// BetaServer implements this via accessor methods added in server_clean.go.
// Kept intentionally small to reduce coupling and simplify future refactors.
type Deps interface {
    CapAuditPersistPath() string
    CapAuditPrevHash() string
    AnchorClient() interface {
        Anchor(string) (anchorpkg.AnchorRecord, error)
        LatestAnchor() (anchorpkg.AnchorRecord, error)
        TotalAnchors() int
    }
}

// RegisterBasic mounts capability audit verify + anchor endpoints on provided router group.
func RegisterBasic(rg *gin.RouterGroup, deps Deps) {
    rg.GET("/capabilities/audit/verify", func(c *gin.Context) { verifyHandler(c, deps) })
    rg.POST("/capabilities/audit/anchor", func(c *gin.Context) { anchorHandler(c, deps) })
}

// respondError mirrors global error taxonomy response shape (success=false payload).
func respondError(c *gin.Context, status int, errorCode, code, message, ref string, detail any) {
    payload := gin.H{"success": false, "error": errorCode, "code": code, "message": message, "ref": ref}
    if detail != nil { payload["detail"] = detail }
    c.JSON(status, payload)
}

// GET /capabilities/audit/verify
// Response: {success, configured:bool, latest:{hash,prev_hash,timestamp}?, chain_tip, integrity_ok:bool}
func verifyHandler(c *gin.Context, d Deps) {
    path := d.CapAuditPersistPath()
    if path == "" {
        c.JSON(http.StatusOK, gin.H{"success": true, "configured": false})
        return
    }
    b, err := os.ReadFile(path)
    if err != nil {
        respondError(c, http.StatusInternalServerError, "capabilities_audit_read_failed", "read_failed", "failed reading capability audit file", "rfc111:capabilities_audit_verify", nil)
        return
    }
    var wrapper struct {
        Payload   json.RawMessage `json:"payload"`
        PrevHash  string          `json:"prev_hash"`
        Hash      string          `json:"hash"`
        Timestamp string          `json:"timestamp"`
    }
    if jerr := json.Unmarshal(b, &wrapper); jerr != nil {
        respondError(c, http.StatusInternalServerError, "capabilities_audit_invalid_json", "invalid_json", "capability audit wrapper invalid JSON", "rfc111:capabilities_audit_verify", nil)
        return
    }
    h := sha256.Sum256(wrapper.Payload)
    recomputed := fmt.Sprintf("sha256:%x", h[:])
    integrity := recomputed == wrapper.Hash
    c.JSON(http.StatusOK, gin.H{"success": true, "configured": true, "latest": gin.H{"hash": wrapper.Hash, "prev_hash": wrapper.PrevHash, "timestamp": wrapper.Timestamp}, "chain_tip": d.CapAuditPrevHash(), "integrity_ok": integrity})
}

// POST /capabilities/audit/anchor
// Mirrors monolithic implementation; preserves error codes and fields.
func anchorHandler(c *gin.Context, d Deps) {
    if os.Getenv("GAUTH_CAPABILITY_ANCHOR_ENABLE") != "1" {
        respondError(c, http.StatusForbidden, "capability_anchor_disabled", "anchoring_disabled", "capability anchoring disabled", "rfc111:capability_anchor", nil)
        return
    }
    client := d.AnchorClient()
    if client == nil {
        respondError(c, http.StatusInternalServerError, "capability_anchor_client_unavailable", "anchor_client_unavailable", "anchor client unavailable", "rfc111:capability_anchor", nil)
        return
    }
    tip := d.CapAuditPrevHash()
    if tip == "" {
        respondError(c, http.StatusBadRequest, "capabilities_audit_chain_tip_empty", "chain_tip_empty", "capability audit chain tip empty", "rfc111:capabilities_audit_anchor", nil)
        return
    }
    rec, err := client.Anchor(tip)
    if err != nil {
        respondError(c, http.StatusInternalServerError, "capabilities_audit_anchor_failure", "anchor_failure", "failed anchoring capability audit chain tip", "rfc111:capabilities_audit_anchor", err.Error())
        return
    }
    c.JSON(http.StatusOK, gin.H{"success": true, "hash": rec.Hash, "anchored_at": rec.AnchoredAt.UTC().Format(time.RFC3339), "total": client.TotalAnchors(), "chain_tip": tip, "type": "capability_audit_chain_tip"})
}

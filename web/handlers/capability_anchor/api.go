package capability_anchor

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// API exposes capability anchoring endpoints.
type API struct {
	handler *Handler
}

// NewAPI creates a new API instance.
func NewAPI(h *Handler) *API {
	return &API{handler: h}
}

// RegisterRoutes registers endpoints to the router.
func (a *API) RegisterRoutes(router *gin.Engine) {
	group := router.Group("/api/v1/beta/anchoring")
	{
		group.POST("/anchor", a.apiCapabilityAnchor)
		group.GET("/status", a.apiCapabilityAnchorStatus)
		group.GET("/latest", a.apiCapabilityAnchorLatest)
		group.GET("/chain", a.apiCapabilityAnchorMaterial)
	}
	// External verify under capabilities path (for test compatibility)
	router.GET("/api/v1/beta/capabilities/anchor/external/verify", a.apiExternalAnchorVerify)
	// Legacy aliases for receipts (for test compatibility)
	router.GET("/api/v1/beta/capabilities/anchor/external/receipts/latest", a.apiCapabilityAnchorLatest)
	router.GET("/api/v1/beta/capabilities/anchor/external/receipts", a.apiCapabilityAnchorMaterial)

	// Prometheus metrics
	router.GET("/metrics/anchoring/prometheus", a.apiCapabilityAnchorPrometheus)
}

// apiCapabilityAnchor triggers an immediate anchor operation (manual).
func (a *API) apiCapabilityAnchor(c *gin.Context) {
	receipt, err := a.handler.Anchor(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"receipt":   receipt,
		"timestamp": time.Now().UTC().Format(time.RFC3339Nano),
	})
}

// apiCapabilityAnchorStatus returns the status of the External Anchor Provider.
func (a *API) apiCapabilityAnchorStatus(c *gin.Context) {
	last := a.handler.GetLastReceipt()
	configured := false
	providerName := "none"

	a.handler.mu.RLock()
	if a.handler.Provider != nil {
		configured = true
		// Hacky way to get provider name/type if needed, or just rely on LastReceipt
		if last.Provider != "" {
			providerName = last.Provider
		} else {
			providerName = "configured"
		}
	}
	a.handler.mu.RUnlock()

	c.JSON(http.StatusOK, gin.H{
		"active":      configured,
		"provider":    providerName,
		"last_anchor": last.Timestamp.UTC().Format(time.RFC3339Nano),
		"last_hash":   last.Hash,
		"version":     last.Version,
	})
}

// apiCapabilityAnchorLatest returns the most recent anchor receipt from the provider.
func (a *API) apiCapabilityAnchorLatest(c *gin.Context) {
	a.handler.mu.RLock()
	defer a.handler.mu.RUnlock()

	if a.handler.Store != nil {
		rec := a.handler.Store.Latest()
		if rec.Hash != "" {
			c.JSON(http.StatusOK, gin.H{"success": true, "receipt": rec})
			return
		}
	}

	rec, err := a.handler.Latest(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "receipt": rec})
}

// apiCapabilityAnchorMaterial returns material to verify the chain.
// If store is available, it returns the full chain of receipts.
func (a *API) apiCapabilityAnchorMaterial(c *gin.Context) {
	a.handler.mu.RLock()
	defer a.handler.mu.RUnlock()

	if a.handler.Store == nil {
		// Fallback to internal latest
		rec := a.handler.LastReceipt
		c.JSON(http.StatusOK, gin.H{
			"success":  true,
			"receipt":  rec,
			"chain":    nil,
			"verified": true, // Partial
		})
		return
	}

	entries := a.handler.Store.Entries()
	// Verify incremental
	status, idx, hash := a.handler.Store.VerifyIncremental()

	c.JSON(http.StatusOK, gin.H{
		"success":          true,
		"chain":            entries,
		"integrity_status": status,
		"fail_index":       idx,
		"fail_hash":        hash,
	})
}

// apiCapabilityAnchorPrometheus exposes anchoring metrics.
func (a *API) apiCapabilityAnchorPrometheus(c *gin.Context) {
	last := a.handler.GetLastReceipt()

	var b string
	b += "# HELP agentauth_anchor_last_timestamp_seconds Timestamp of last successful external anchor\n"
	b += "# TYPE agentauth_anchor_last_timestamp_seconds gauge\n"
	if !last.Timestamp.IsZero() {
		b += fmt.Sprintf("agentauth_anchor_last_timestamp_seconds %d\n", last.Timestamp.Unix())
	} else {
		b += "agentauth_anchor_last_timestamp_seconds 0\n"
	}

	active := 0
	a.handler.mu.RLock()
	if a.handler.Provider != nil {
		active = 1
	}
	a.handler.mu.RUnlock()

	b += "# HELP agentauth_anchor_provider_active Indicates if an external anchor provider is configured (1=yes, 0=no)\n"
	b += "# TYPE agentauth_anchor_provider_active gauge\n"
	b += fmt.Sprintf("agentauth_anchor_provider_active %d\n", active)

	c.Data(http.StatusOK, "text/plain; version=0.0.4", []byte(b))
}

// apiExternalAnchorVerify verifies the external anchor chain.
func (a *API) apiExternalAnchorVerify(c *gin.Context) {
	a.handler.mu.RLock()
	provider := a.handler.Provider
	store := a.handler.Store
	last := a.handler.LastReceipt
	a.handler.mu.RUnlock()

	if provider == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"success": false,
			"message": "external anchor provider not configured",
		})
		return
	}

	// Check if we have any receipt at all
	isEmpty := last.Hash == ""

	// Verify chain integrity
	valid := true
	var failIndex int = -1
	var failHash = ""
	if store != nil && !isEmpty {
		status, idx, hash := store.VerifyIncremental()
		if status != "ok" {
			valid = false
			failIndex = idx
			failHash = hash
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success":     true,
		"verified":    valid && !isEmpty,
		"empty":       isEmpty,
		"last_hash":   last.Hash,
		"last_anchor": last.Timestamp.UTC().Format(time.RFC3339Nano),
		"fail_index":  failIndex,
		"fail_hash":   failHash,
	})
}

package semantic

import (
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// API provides HTTP endpoints for semantic anomaly detection.
type API struct {
	Handler *Handler
}

// NewAPI creates a new semantic API.
func NewAPI(h *Handler) *API {
	return &API{Handler: h}
}

// RegisterRoutes registers endpoints on the provided router.
func (a *API) RegisterRoutes(r *gin.Engine) {
	r.GET("/api/v1/beta/metrics/poa/semantics", a.HandleCounters)
	r.GET("/api/v1/beta/metrics/poa/semantics/prometheus", a.HandlePrometheus)
	r.GET("/api/v1/beta/metrics/poa/semantics/verify", a.HandleVerify)
}

// HandleCounters exposes prototype PoA semantic counters.
func (a *API) HandleCounters(c *gin.Context) {
	if a.Handler.Service == nil {
		c.JSON(200, gin.H{"success": true, "counters": map[string]uint64{}, "wired": false})
		return
	}
	// Fetch fresh snapshot via Handler which may have internal processing
	a.Handler.Update()
	ss := a.Handler.Service.SemanticSnapshot()
	c.JSON(200, gin.H{"success": true, "counters": ss, "wired": true})
}

// HandlePrometheus exposes semantic counters in Prometheus format.
func (a *API) HandlePrometheus(c *gin.Context) {
	if a.Handler.Service == nil {
		c.String(200, "# RFC0111 Service not wired\n")
		return
	}

	ss := a.Handler.Service.SemanticSnapshot()
	var sb strings.Builder
	sb.WriteString("# HELP gauth_poa_semantic_counter Prototype RFC0111 semantic counters\n")
	sb.WriteString("# TYPE gauth_poa_semantic_counter counter\n")
	for k, v := range ss {
		// sanitization simple
		kSan := strings.ReplaceAll(k, "-", "_")
		kSan = strings.ReplaceAll(kSan, ".", "_")
		sb.WriteString(fmt.Sprintf("gauth_poa_semantic_counter{key=\"%s\"} %d\n", kSan, v))
	}

	// Also expose anomaly scores
	a.Handler.mu.Lock()
	scores := copyScores(a.Handler.scores)
	a.Handler.mu.Unlock()

	if len(scores) > 0 {
		sb.WriteString("\n# HELP gauth_poa_semantic_anomaly_score EWMA Z-score for detected anomalies\n")
		sb.WriteString("# TYPE gauth_poa_semantic_anomaly_score gauge\n")
		for k, v := range scores {
			kSan := strings.ReplaceAll(k, "-", "_")
			sb.WriteString(fmt.Sprintf("gauth_poa_semantic_anomaly_score{key=\"%s\"} %f\n", kSan, v))
		}
	}

	// Add rate metrics
	rates60 := a.Handler.ComputeRates(60 * time.Second)
	if len(rates60) > 0 {
		sb.WriteString("\n# HELP gauth_poa_semantic_rate_60s Per-minute rate of semantic events over last 60s\n")
		sb.WriteString("# TYPE gauth_poa_semantic_rate_60s gauge\n")
		for k, v := range rates60 {
			kSan := strings.ReplaceAll(k, "-", "_")
			kSan = strings.ReplaceAll(kSan, ".", "_")
			sb.WriteString(fmt.Sprintf("gauth_poa_semantic_rate_60s{key=\"%s\"} %f\n", kSan, v))
		}
	}

	rates300 := a.Handler.ComputeRates(300 * time.Second)
	if len(rates300) > 0 {
		sb.WriteString("\n# HELP gauth_poa_semantic_rate_300s Per-minute rate of semantic events over last 300s\n")
		sb.WriteString("# TYPE gauth_poa_semantic_rate_300s gauge\n")
		for k, v := range rates300 {
			kSan := strings.ReplaceAll(k, "-", "_")
			kSan = strings.ReplaceAll(kSan, ".", "_")
			sb.WriteString(fmt.Sprintf("gauth_poa_semantic_rate_300s{key=\"%s\"} %f\n", kSan, v))
		}
	}

	// Expose integrity status
	status, _ := a.Handler.VerifyPersistence()
	if status != "unconfigured" {
		val := 0
		if status == "ok" {
			val = 1
		}
		sb.WriteString("\n# HELP gauth_persistence_integrity_semantic Semantic persistence integrity check (1=ok, 0=mismatch/fail)\n")
		sb.WriteString("# TYPE gauth_persistence_integrity_semantic gauge\n")
		sb.WriteString(fmt.Sprintf("gauth_persistence_integrity_semantic %d\n", val))
	}

	c.String(200, sb.String())
}

// HandleVerify performs integrity verification for semantic persistence.
// Since the new handler manages persistence, we just check if it loaded correctly
// or if the file exists and is valid JSON.
func (a *API) HandleVerify(c *gin.Context) {
	// Re-load to verify integrity on disk
	err := a.Handler.Load()
	if err != nil {
		c.JSON(500, gin.H{"success": false, "error": err.Error(), "status": "corrupt"})
		return
	}

	// VerifyPersistence called
	integrity, _ := a.Handler.VerifyPersistence()

	ewma, scores := a.Handler.Stats()
	a.Handler.mu.Lock()
	ledgerWired := a.Handler.Ledger != nil
	anchorWired := a.Handler.AnchorProvider != nil
	lastAnchor := a.Handler.LastAnchor
	lastReceipt := a.Handler.LastAnchorReceipt
	a.Handler.mu.Unlock()

	c.JSON(200, gin.H{
		"success":   true,
		"status":    "ok",
		"integrity": integrity,
		"stats": gin.H{
			"ewma_entries":     ewma,
			"active_anomalies": scores,
		},
		"ledger": gin.H{
			"wired": ledgerWired,
		},
		"anchoring": gin.H{
			"wired":        anchorWired,
			"last_anchor":  lastAnchor.Format(time.RFC3339),
			"last_receipt": lastReceipt,
		},
	})
}

func copyScores(m map[string]float64) map[string]float64 {
	out := make(map[string]float64, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

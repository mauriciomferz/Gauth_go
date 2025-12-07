package violations

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// API exposes violation related endpoints.
type API struct {
	handler *Handler
}

// NewAPI creates a new API instance.
func NewAPI(h *Handler) *API {
	return &API{handler: h}
}

// RegisterRoutes registers endpoints to the router.
func (a *API) RegisterRoutes(router *gin.Engine) {
	// Violation metrics endpoints
	metricsGroup := router.Group("/api/v1/beta/metrics/violations")
	{
		metricsGroup.GET("", a.apiViolationMetrics)
		metricsGroup.GET("/prometheus", a.apiViolationMetricsPrometheus)
		metricsGroup.GET("/verify", a.apiViolationPersistenceVerify)
	}
}

// apiViolationMetrics returns violation counts in JSON format.
func (a *API) apiViolationMetrics(c *gin.Context) {
	// Get current counters from service
	counters := make(map[string]uint64)
	if a.handler.Service != nil {
		counters = a.handler.Service.ViolationSnapshot()
	}

	// Standard categories for violation types
	categories := []string{
		"sig_invalid", "expired", "not_yet_valid", "issuer_mismatch",
		"replay_detected", "audience_mismatch", "missing_claim", "unknown", "capability_denied",
	}

	// Ensure all categories are present in counters
	for _, cat := range categories {
		if _, ok := counters[cat]; !ok {
			counters[cat] = 0
		}
	}

	// Calculate total
	var total uint64
	for _, v := range counters {
		total += v
	}

	c.JSON(http.StatusOK, gin.H{
		"success":    true,
		"timestamp":  time.Now().UTC().Format(time.RFC3339Nano),
		"counters":   counters,
		"total":      total,
		"categories": categories,
	})
}

// apiViolationMetricsPrometheus returns violation metrics in Prometheus text format.
func (a *API) apiViolationMetricsPrometheus(c *gin.Context) {
	rates60 := a.handler.ComputeRates(60 * time.Second)
	rates300 := a.handler.ComputeRates(300 * time.Second)

	// Get current counters from service
	counters := make(map[string]uint64)
	if a.handler.Service != nil {
		counters = a.handler.Service.ViolationSnapshot()
	}

	status, _ := a.handler.VerifyPersistence()

	// Build Prometheus exposition format manually (simple enough for this use case)
	var out string

	// Validation total counters
	categories := []string{"sig_invalid", "expired", "not_yet_valid", "issuer_mismatch", "replay_detected", "audience_mismatch", "missing_claim", "unknown", "capability_denied"}
	var total uint64
	for _, cat := range categories {
		total += counters[cat]
	}
	out += "# HELP gauth_validation_total Total validation checks\n"
	out += "# TYPE gauth_validation_total counter\n"
	out += "gauth_validation_total " + fmtUint(total) + "\n"

	// Per-category counters
	for _, cat := range categories {
		out += "# HELP gauth_validation_" + cat + "_total Total " + cat + " violations\n"
		out += "# TYPE gauth_validation_" + cat + "_total counter\n"
		out += "gauth_validation_" + cat + "_total " + fmtUint(counters[cat]) + "\n"
	}

	out += "# HELP gauth_violations_rate_60s Per-second rate of violations over last 60s\n"
	out += "# TYPE gauth_violations_rate_60s gauge\n"
	for k, v := range rates60 {
		out += "# " + k + " " + fmtFloat(v) + "\n" // Comment line
		out += "gauth_violations_rate_60s{type=\"" + k + "\"} " + fmtFloat(v) + "\n"
	}
	out += "# HELP gauth_violations_rate_300s Per-second rate of violations over last 300s\n"
	out += "# TYPE gauth_violations_rate_300s gauge\n"
	for k, v := range rates300 {
		out += "gauth_violations_rate_300s{type=\"" + k + "\"} " + fmtFloat(v) + "\n"
	}

	// Integrity status as a gauge (1=ok, 0=bad)
	integrityVal := 0
	if status == "ok" {
		integrityVal = 1
	}
	out += "# HELP gauth_persistence_integrity_violation Violation persistence integrity check (1=ok, 0=mismatch/fail)\n"
	out += "# TYPE gauth_persistence_integrity_violation gauge\n"
	out += "gauth_persistence_integrity_violation " + fmtInt(integrityVal) + "\n"

	c.Data(http.StatusOK, "text/plain; version=0.0.4", []byte(out))
}

// apiViolationPersistenceVerify returns detailed verification status.
func (a *API) apiViolationPersistenceVerify(c *gin.Context) {
	status, details := a.handler.VerifyPersistence()
	c.JSON(http.StatusOK, gin.H{
		"integrity": status,
		"details":   details,
	})
}

func fmtFloat(f float64) string {
	b, _ := json.Marshal(f)
	return string(b)
}

func fmtInt(i int) string {
	b, _ := json.Marshal(i)
	return string(b)
}

func fmtUint(u uint64) string {
	b, _ := json.Marshal(u)
	return string(b)
}

package anchor

// RB8 Anchor metrics Prometheus exposition modular extraction.
// Provides GET /api/v1/beta/capabilities/anchor/metrics/prometheus via RegisterMetrics.
// Delegates to BetaServer wrapper method to preserve existing logic without duplication.

import "github.com/gin-gonic/gin"

// MetricsDeps defines dependency surface for metrics exposition handler.
// BetaServer implements this via CapabilityAnchorMetricsPrometheus wrapper.
// Narrow interface keeps package decoupled from full server.
type MetricsDeps interface {
	CapabilityAnchorMetricsPrometheus(*gin.Context)
}

// RegisterMetrics mounts Prometheus capability anchor metrics endpoint on router group.
func RegisterMetrics(r *gin.RouterGroup, deps MetricsDeps) {
	r.GET("/capabilities/anchor/metrics/prometheus", func(c *gin.Context) { deps.CapabilityAnchorMetricsPrometheus(c) })
}

package test

import (
	"strings"
	"testing"

	"github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/web"
	"github.com/gin-gonic/gin"
)

// TestCapabilityAnchorPrometheusIntegrityGauge ensures the custom Prometheus endpoint surfaces the receipt chain integrity gauge HELP/TYPE even when unconfigured.
func TestCapabilityAnchorPrometheusIntegrityGauge(t *testing.T) {
	gin.SetMode(gin.TestMode)
	srv := web.NewBetaServer(":0")
	w := web.PerformRequest(srv, "GET", "/api/v1/beta/capabilities/anchor/metrics/prometheus")
	if w.Code != 200 {
		t.Fatalf("unexpected status: %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "# HELP gauth_capability_anchor_notarization_receipts_integrity") {
		t.Fatalf("expected HELP line for integrity gauge")
	}
	if !strings.Contains(body, "# TYPE gauth_capability_anchor_notarization_receipts_integrity gauge") {
		t.Fatalf("expected TYPE line for integrity gauge")
	}
	if !strings.Contains(body, "gauth_capability_anchor_notarization_receipts_integrity") {
		t.Fatalf("expected gauge value line in body")
	}
}

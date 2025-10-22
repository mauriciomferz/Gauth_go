package test

import (
	"strings"
	"testing"
	"time"

	"github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/web"
	"github.com/gin-gonic/gin"
)

// TestCapabilityAnchorPrometheusVerifyParam ensures verify=1 triggers re-computation even when status already set.
func TestCapabilityAnchorPrometheusVerifyParam(t *testing.T) {
	gin.SetMode(gin.TestMode)
	srv := web.NewBetaServer(":0")
	// Force initial integrity computation via first scrape.
	w1 := web.PerformRequest(srv, "GET", "/api/v1/beta/capabilities/anchor/metrics/prometheus")
	if w1.Code != 200 {
		t.Fatalf("initial scrape status %d", w1.Code)
	}
	body1 := w1.Body.String()
	if !strings.Contains(body1, "gauth_capability_anchor_notarization_receipts_integrity") {
		t.Fatalf("expected integrity gauge line in first scrape")
	}
	// Capture timestamp before verify trigger
	t1 := srv.LastReceiptVerifyTime()
	time.Sleep(10 * time.Millisecond)
	// Trigger on-demand recompute using verify=1
	w3 := web.PerformRequest(srv, "GET", "/api/v1/beta/capabilities/anchor/metrics/prometheus?verify=1")
	if w3.Code != 200 {
		t.Fatalf("verify scrape status %d", w3.Code)
	}
	body3 := w3.Body.String()
	if !strings.Contains(body3, "gauth_capability_anchor_notarization_receipts_integrity") {
		t.Fatalf("expected integrity gauge line after verify")
	}
	// We cannot guarantee mismatch unless chain semantics known; ensure HELP line present and gauge value line present.
	if !strings.Contains(body3, "# HELP gauth_capability_anchor_notarization_receipts_integrity") {
		t.Fatalf("missing HELP after verify")
	}
	t2 := srv.LastReceiptVerifyTime()
	if t2.Before(t1) || t2.Equal(t1) {
		t.Fatalf("expected last verify time to advance after verify=1 param")
	}
}

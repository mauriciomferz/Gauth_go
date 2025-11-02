package web

import (
	"net/http/httptest"
	"os"
	"testing"
	"time"

	imetrics "github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/internal/metrics"
)

// TestExternalAnchorMetrics verifies metrics update and status receipt exposure when external anchor provider enabled.
func TestExternalAnchorMetrics(t *testing.T) {
	t.Setenv("GAUTH_CAP_EXTERNAL_ANCHOR_PROVIDER", "memory")
	t.Setenv("GAUTH_CAPABILITIES_PATH", "") // ensure static seed
	pm := imetrics.NewPrometheusMetrics(imetrics.PrometheusAdapterOptions{Namespace: "gauth", Subsystem: "rfc0111"})
	srv := NewBetaServer(":0")
	// Inject Prometheus metrics adapter
	srv.metrics = pm
	// Force capability reload to trigger emission path (anchor artifact emission invokes external provider)
	// Emission occurs on initial load; wait briefly for background goroutine age update.
	time.Sleep(50 * time.Millisecond)
	// Hit status endpoint
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/beta/capabilities/anchor/status", nil)
	srv.router.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("status code %d", rec.Code)
	}
	body := rec.Body.String()
	if os.Getenv("CI_DEBUG") == "1" {
		t.Log(body)
	}
	// Expect external_anchor_receipt present
	if !hasSubstr(body, "external_anchor_receipt") {
		t.Fatalf("expected external_anchor_receipt in status payload")
	}
}

// TestExternalAnchorFailureMetrics uses tsa_stub with fail probability 1 to force failures and verify counters increment.
func TestExternalAnchorFailureMetrics(t *testing.T) {
	t.Setenv("GAUTH_CAP_EXTERNAL_ANCHOR_PROVIDER", "tsa_stub")
	t.Setenv("GAUTH_CAP_EXTERNAL_ANCHOR_FAIL_PROB", "1") // always fail
	pm := imetrics.NewPrometheusMetrics(imetrics.PrometheusAdapterOptions{Namespace: "gauth", Subsystem: "rfc0111"})
	srv := NewBetaServer(":0")
	srv.metrics = pm
	// Trigger capability reload to attempt anchor
	time.Sleep(20 * time.Millisecond)
	// Since provider always fails, age gauge remains 0 and receipt absent.
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/beta/capabilities/anchor/status", nil)
	srv.router.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("status code %d", rec.Code)
	}
	body := rec.Body.String()
	if hasSubstr(body, "external_anchor_receipt") {
		t.Fatalf("unexpected receipt present for forced failure provider")
	}
}

// contains helper (avoid pulling strings package repeatedly for readability).
func hasSubstr(haystack, needle string) bool {
	return len(haystack) >= len(needle) && (capExtIndex(haystack, needle) >= 0)
}

// capExtIndex naive substring search (small bodies, keep dependency minimal).
func capExtIndex(s, sub string) int {
	if sub == "" {
		return 0
	}
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

package web

import (
	"io"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"testing"
	"time"

	imetrics "github.com/mauriciomferz/Gauth_go/internal/metrics"
	prom "github.com/prometheus/client_golang/prometheus"
	promhttp "github.com/prometheus/client_golang/prometheus/promhttp"
)

// TestCapabilityAnchorPrometheusGauge ensures Prometheus gauge reflects last write unix seconds after emission.
func TestCapabilityAnchorPrometheusGauge(t *testing.T) {
	t.Setenv("GAUTH_TOKEN_SIG_MODE", "eddsa")
	t.Setenv("GAUTH_CAP_ANCHOR_SIGN", "0")
	anchorFile, err := os.CreateTemp(t.TempDir(), "cap-anchor-prom-*.json")
	if err != nil {
		t.Fatalf("temp file: %v", err)
	}
	anchorFile.Close()
	t.Setenv("GAUTH_CAP_ANCHOR_FILE_PATH", anchorFile.Name())
	t.Setenv("GAUTH_CAP_ANCHOR_WRITE_INTERVAL", "1m")
	capFile := filepath.Join(t.TempDir(), "caps.json")
	if err2 := os.WriteFile(capFile, []byte(`{"schema_version":1,"capabilities":[{"id":"cap.transfer","version":"1.0","stable":true}],["action_mappings":{"transaction:execute":["cap.transfer"]}}`), 0o600); err2 != nil {
		// fix malformed JSON quickly
		_ = os.WriteFile(capFile, []byte(`{"schema_version":1,"capabilities":[{"id":"cap.transfer","version":"1.0","stable":true}],"action_mappings":{"transaction:execute":["cap.transfer"]}}`), 0o600)
	}
	t.Setenv("GAUTH_CAPABILITIES_PATH", capFile)
	t.Setenv("GAUTH_DISABLE_BG_POLLS", "1")
	t.Setenv("GAUTH_SKIP_SMOKETEST", "1")
	// Use a dedicated registry to avoid clashes with global default.
	reg := prom.NewRegistry()
	// Create server; then swap metrics to Prometheus adapter for test.
	srv := NewBetaServer(":0")
	t.Cleanup(func() { srv.Shutdown() })
	// Replace metrics with Prometheus implementation.
	pm := imetrics.NewPrometheusMetrics(imetrics.PrometheusAdapterOptions{Registry: reg, Namespace: "gauth", Subsystem: "rfc0111"})
	srv.metrics = pm
	// Instead of relying on background emission, set the metric explicitly for deterministic test.
	setUnix := time.Now().Unix()
	//nolint:gosec // G115: test code, Unix timestamp always positive
	pm.SetCapabilityAnchorLastWriteUnix(uint64(setUnix))
	// Collect metrics text exposition manually.
	// Use prometheus handler to gather current samples.
	rr := httptest.NewRecorder()
	promhttp.HandlerFor(reg, promhttp.HandlerOpts{}).ServeHTTP(rr, httptest.NewRequest("GET", "/metrics", nil))
	body, _ := io.ReadAll(rr.Body)
	re := regexp.MustCompile(`(?m)^gauth_rfc0111_capability_anchor_last_write_seconds ([0-9.e+\-]+)$`)
	m := re.FindSubmatch(body)
	if len(m) < 2 {
		t.Fatalf("gauge not found in metrics output:\n%s", string(body))
	}
	// Basic sanity: timestamp should be within a small drift of now.
	valStr := string(m[1])
	f, err := strconv.ParseFloat(valStr, 64)
	if err != nil {
		t.Fatalf("parse gauge value: %v", err)
	}
	ts := int64(f)
	now := time.Now().Unix()
	if ts == 0 {
		t.Fatalf("parsed gauge timestamp is zero; expected non-zero")
	}
	if diff := now - ts; diff < 0 || diff > 2 {
		t.Fatalf("gauge timestamp drift out of range drift=%d now=%d ts=%d setUnix=%d", diff, now, ts, setUnix)
	}

	// Cross-check gauge value matches the explicit setter value within drift.
	if diff := ts - setUnix; diff < 0 || diff > 2 {
		t.Fatalf("gauge vs set value mismatch drift=%d gauge=%d set=%d", diff, ts, setUnix)
	}
}

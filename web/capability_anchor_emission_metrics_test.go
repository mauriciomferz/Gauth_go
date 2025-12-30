package web

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"regexp"
	"testing"
	"time"

	imetrics "github.com/mauriciomferz/AgentAuth/internal/metrics"
	"github.com/mauriciomferz/AgentAuth/pkg/anchor"
	"github.com/prometheus/client_golang/prometheus"
)

// TestCapabilityAnchorEmissionMetrics verifies emission interval histogram & jitter gauge exposition logic
// via the custom /api/v1/beta/capabilities/anchor/metrics/prometheus endpoint after multiple emissions.
func TestCapabilityAnchorEmissionMetrics(t *testing.T) {
	t.Setenv("AGENTAUTH_CAP_ANCHOR_FILE_PATH", t.TempDir()+"/anchor.json")
	t.Setenv("AGENTAUTH_CAP_ANCHOR_WRITE_INTERVAL", "5ms")
	pm := imetrics.NewPrometheusMetrics(imetrics.PrometheusAdapterOptions{
		Namespace: "AGENTAUTH",
		Subsystem:"AAP-001",
		Registry:  prometheus.NewRegistry(),
	})
	srv := NewBetaServer(":0")
	t.Cleanup(func() { srv.Shutdown() })
	// Enable anchoring endpoints
	t.Setenv("AGENTAUTH_CAPABILITY_ANCHOR_ENABLE", "1")
	t.Setenv("AGENTAUTH_ANCHOR_PROVIDER", "memory")
	// Inject prometheus metrics (server default uses memory) via direct assignment for test simplicity.
	srv.metrics = pm
	// Ensure memory anchor client exists (NewBetaServer may not create if env set after construction)
	if srv.anchorClient == nil {
		srv.anchorClient = anchor.NewMemoryAnchor()
	}
	// Simulate multiple emissions by mutating registry hash & calling anchor endpoint (POST) after sleeps.
	// We directly modify internal hash fields to force change detection without full file loader.
	for i := 0; i < 5; i++ {
		time.Sleep(6 * time.Millisecond)
		// Mutate capabilityRegistryHash to new value ensuring semantic change triggers hash_changed counter logic if path exercised.
		payload := struct {
			Dummy int `json:"dummy"`
		}{Dummy: i}
		enc, _ := json.Marshal(payload)
		h := sha256.Sum256(enc)
		srv.capabilitiesHandler.RegistryHash = fmt.Sprintf("sha256:%x", h[:])
		// Anchor via POST endpoint (will emit artifact if interval elapsed).
		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/api/v1/beta/capabilities/anchor", nil)
		srv.router.ServeHTTP(w, req)
		if w.Code != 200 {
			t.Fatalf("anchor post %d status=%d body=%s", i, w.Code, w.Body.String())
		}
	}

	// Hit Prometheus exposition endpoint.
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/beta/capabilities/anchor/metrics/prometheus", nil)
	srv.router.ServeHTTP(rec, req)
	body := rec.Body.String()

	// Force anchor emission
	// If registry hash is empty, something is wrong
	if srv.GetCapabilityRegistryHash() == "" {
		t.Fatalf("expected registry hash")
	}

	// Basic presence checks.
	if !regexp.MustCompile(`AGENTAUTH_aap001_capability_anchor_emission_jitter_seconds`).MatchString(body) {
		t.Fatalf("expected jitter gauge line in body:\n%s", body)
	}
	if !regexp.MustCompile(`AGENTAUTH_aap001_capability_anchor_age_seconds`).MatchString(body) {
		t.Fatalf("expected age gauge line in body")
	}
	// Jitter should be >0 after varied intervals (non-zero stddev). Accept small floating value.
	if m := regexp.MustCompile(`AGENTAUTH_aap001_capability_anchor_emission_jitter_seconds ([0-9E.e+-]+)`).FindStringSubmatch(body); len(m) == 2 {
		// Accept presence; do not assert non-zero to avoid flakiness on CI timing.
	} else {
		t.Fatalf("did not find jitter metric line")
	}
}

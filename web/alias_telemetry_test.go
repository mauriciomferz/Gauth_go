package web

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mauriciomferz/AgentAuth/web/testutil"
)

// TestLegacyAliasTelemetry verifies legacy_alias_hits increments when invoking the deprecated alias.
func TestLegacyAliasTelemetry(t *testing.T) {
	testutil.WithEnv(t, "GAUTH_DISABLE_LEGACY_GOVERNANCE_ALIAS", "", func() {
		srv := NewBetaServer("")
		t.Cleanup(func() { srv.Shutdown() })
		// Baseline metrics
		m1 := httptest.NewRecorder()
		req1, _ := http.NewRequest("GET", "/api/v1/beta/metrics/lifecycle", nil)
		srv.router.ServeHTTP(m1, req1)
		if m1.Code != 200 {
			t.Fatalf("metrics baseline expected 200 got %d", m1.Code)
		}
		// Invoke legacy alias twice
		for i := 0; i < 2; i++ {
			rr := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/api/governance/lifecycle_timeline", nil)
			srv.router.ServeHTTP(rr, req)
			if rr.Code != 200 {
				t.Fatalf("legacy alias expected 200 got %d", rr.Code)
			}
		}
		// Check metrics again
		m2 := httptest.NewRecorder()
		req2, _ := http.NewRequest("GET", "/api/v1/beta/metrics/lifecycle", nil)
		srv.router.ServeHTTP(m2, req2)
		if m2.Code != 200 {
			t.Fatalf("metrics after hits expected 200 got %d", m2.Code)
		}
		body := m2.Body.String()
		if !containsTelemetry(body, 2) {
			t.Fatalf("expected legacy_alias_hits >=2 in metrics body: %s", body)
		}
	})
}

// containsTelemetry is a naive check to avoid importing JSON for this small test.
func containsTelemetry(body string, min int) bool {
	// Look for pattern "legacy_alias_hits": <number>
	// Accept any number >= min. Use strings.Index for simplicity.
	startPos := strings.Index(body, "legacy_alias_hits")
	if startPos == -1 {
		return false
	}
	colonPos := strings.Index(body[startPos:], ":")
	if colonPos == -1 {
		return false
	}
	colonPos += startPos
	j := colonPos + 1
	for j < len(body) && (body[j] == ' ' || body[j] == '\t') {
		j++
	}
	start := j
	for j < len(body) && body[j] >= '0' && body[j] <= '9' {
		j++
	}
	if start == j {
		return false
	}
	val := 0
	for k := start; k < j; k++ {
		val = val*10 + int(body[k]-'0')
	}
	return val >= min
}

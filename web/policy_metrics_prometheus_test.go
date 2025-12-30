package web

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestPolicyMetricsPrometheus ensures Prometheus exposition contains expected metric lines after evaluations.
func TestPolicyMetricsPrometheus(t *testing.T) {
	bs := NewBetaServer("")
	t.Cleanup(func() { bs.Shutdown() })
	// Run a few evaluations to populate metrics & latency histogram.
	payloads := []string{
		`{"subject":"alice@example.com","action":"read","resource":"report:finance","attrs":{}}`,
		`{"subject":"alice@example.com","action":"write","resource":"report:finance","attrs":{"classification":"secret"}}`,
		`{"subject":"alice@example.com","action":"read","resource":"report:finance","attrs":{}}`,
	}
	for i, body := range payloads {
		rec := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/policy/evaluate", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		bs.router.ServeHTTP(rec, req)
		if rec.Code != 200 {
			// Some requests may deny but should still be 200
			t.Fatalf("evaluation %d failed code=%d body=%s", i, rec.Code, rec.Body.String())
		}
	}
	// Fetch Prometheus metrics
	recM := httptest.NewRecorder()
	reqM, _ := http.NewRequest("GET", "/api/v1/policy/metrics/prometheus", nil)
	bs.router.ServeHTTP(recM, reqM)
	if recM.Code != 200 {
		t.Fatalf("prometheus metrics endpoint code=%d body=%s", recM.Code, recM.Body.String())
	}
	out := recM.Body.String()
	// Required lines / prefixes
	checks := []string{
		"AGENTAUTH_policy_evaluations_total ",
		"AGENTAUTH_policy_evaluations_allow_total ",
		"AGENTAUTH_policy_evaluations_deny_total ",
		"AGENTAUTH_policy_eval_latency_ns_bucket{le=\"+Inf\"}",
		"AGENTAUTH_policy_eval_latency_ns_count ",
		"AGENTAUTH_policy_eval_latency_ns_sum ",
		"AGENTAUTH_policy_eval_latency_ns_p99 ",
	}
	for _, ck := range checks {
		if !strings.Contains(out, ck) {
			// Provide a helpful diff context
			t.Fatalf("missing metrics line containing %q. Output:\n%s", ck, out)
		}
	}
}

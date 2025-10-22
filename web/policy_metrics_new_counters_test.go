package web

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestPolicyMetricsNewCounters validates that rollback and diff counters increment
// and are exposed via both JSON and Prometheus metrics endpoints.
func TestPolicyMetricsNewCounters(t *testing.T) {
	srv := NewTestServerNoSeed(t)
	// Append two bundles via existing endpoint (seed creation simulated by adding bundles)
	// Reuse demo bundle POST helper in existing tests via PerformRequest.
	// Use minimal bundles; payload not relevant for counter test.
	var versions []int
	for i := 0; i < 2; i++ {
		body := `{"id":"b` + fmt.Sprintf("%d", i+1) + `","policies":[{"id":"p` + fmt.Sprintf("%d", i+1) + `","subjects":["sub"],"rules":[{"actions":["act"],"resources":["res"],"effect":"allow"}]}]}`
		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/api/v1/beta/policy/bundles", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		srv.router.ServeHTTP(w, req)
		if w.Code != 201 {
			t.Fatalf("bundle append failed: status=%d body=%s", w.Code, w.Body.String())
		}
		var resp struct {
			Success       bool `json:"success"`
			PolicyVersion int  `json:"policy_version"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("unmarshal bundle resp: %v", err)
		}
		if !resp.Success {
			t.Fatalf("bundle success false")
		}
		versions = append(versions, resp.PolicyVersion)
	}
	// Perform a diff request (from active to head) - defaults handled by endpoint.
	// diff between version[0] and version[1]
	diffPath := fmt.Sprintf("/api/v1/beta/policy/diff?from=%d&to=%d", versions[0], versions[1])
	wDiff := PerformRequest(srv, "GET", diffPath)
	if wDiff.Code != 200 {
		t.Fatalf("diff request failed: status=%d body=%s", wDiff.Code, wDiff.Body.String())
	}
	// Perform a rollback to version 1 (requires admin token header).
	wRbRec := httptest.NewRecorder()
	rbReq := httptest.NewRequest("POST", "/api/v1/beta/policy/rollback?version=1", nil)
	rbReq.Header.Set("X-Admin-Token", "test-admin")
	srv.router.ServeHTTP(wRbRec, rbReq)
	wRb := wRbRec
	if wRb.Code != 200 {
		t.Fatalf("rollback failed: status=%d body=%s", wRb.Code, wRb.Body.String())
	}
	// Fetch JSON metrics and verify counters
	wMetrics := PerformRequest(srv, "GET", "/api/v1/beta/policy/metrics")
	if wMetrics.Code != 200 {
		t.Fatalf("metrics fetch failed: %d", wMetrics.Code)
	}
	var js struct {
		Success       bool   `json:"success"`
		RollbackCount uint64 `json:"rollback_count"`
		DiffRequests  uint64 `json:"diff_requests"`
	}
	if err := json.Unmarshal(wMetrics.Body.Bytes(), &js); err != nil {
		t.Fatalf("unmarshal metrics: %v", err)
	}
	if !js.Success {
		t.Fatalf("metrics success false")
	}
	if js.RollbackCount != 1 {
		t.Fatalf("expected rollback_count=1 got %d", js.RollbackCount)
	}
	if js.DiffRequests != 1 {
		t.Fatalf("expected diff_requests=1 got %d", js.DiffRequests)
	}
	// Fetch Prometheus metrics and check presence of lines
	wProm := PerformRequest(srv, "GET", "/api/v1/beta/policy/metrics/prometheus")
	if wProm.Code != 200 {
		t.Fatalf("prometheus metrics fetch failed: %d", wProm.Code)
	}
	promBody := wProm.Body.String()
	if !strings.Contains(promBody, "gauth_policy_rollback_total 1") {
		t.Fatalf("expected rollback prometheus metric not found or incorrect: %s", promBody)
	}
	if !strings.Contains(promBody, "gauth_policy_diff_requests_total 1") {
		t.Fatalf("expected diff requests prometheus metric not found or incorrect: %s", promBody)
	}
}

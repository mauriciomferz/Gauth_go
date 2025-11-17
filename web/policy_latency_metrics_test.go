package web

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestPolicyLatencyMetrics validates p99 presence and non-zero histogram counts after evaluations.
func TestPolicyLatencyMetrics(t *testing.T) {
	bs := NewBetaServer("")
	t.Cleanup(func() { bs.Shutdown() })
	// Fire several evaluations to populate buckets
	payloads := []string{
		`{"subject":"alice@example.com","action":"read","resource":"report:finance","attrs":{}}`,
		`{"subject":"alice@example.com","action":"write","resource":"report:finance","attrs":{"classification":"secret"}}`,
		`{"subject":"alice@example.com","action":"read","resource":"report:finance","attrs":{}}`,
		`{"subject":"alice@example.com","action":"read","resource":"report:finance","attrs":{}}`,
	}
	for i, body := range payloads {
		rec := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/beta/policy/evaluate", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		bs.router.ServeHTTP(rec, req)
		if rec.Code != 200 {
			t.Fatalf("evaluation %d failed code=%d body=%s", i, rec.Code, rec.Body.String())
		}
	}
	// Fetch metrics JSON
	recM := httptest.NewRecorder()
	reqM, _ := http.NewRequest("GET", "/api/v1/beta/policy/metrics", nil)
	bs.router.ServeHTTP(recM, reqM)
	if recM.Code != 200 {
		t.Fatalf("metrics endpoint code=%d body=%s", recM.Code, recM.Body.String())
	}
	var out struct {
		Success          bool              `json:"success"`
		P99              int64             `json:"p99_latency_ns"`
		LatencyHistogram map[string]uint64 `json:"latency_histogram"`
		Total            uint64            `json:"total"`
	}
	if err := json.Unmarshal(recM.Body.Bytes(), &out); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if !out.Success {
		t.Fatalf("success=false")
	}
	if out.Total < uint64(len(payloads)) {
		t.Fatalf("unexpected total %d", out.Total)
	}
	if out.P99 <= 0 {
		t.Fatalf("p99 should be > 0 got %d", out.P99)
	}
	// Ensure at least one bucket has non-zero count
	found := false
	for _, v := range out.LatencyHistogram {
		if v > 0 {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected non-zero histogram counts")
	}
}

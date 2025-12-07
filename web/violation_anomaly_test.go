package web

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// TestViolationAnomalyMetrics ensures anomaly section fields populate and rate increases after multiple invalid validations.
func TestViolationAnomalyMetrics(t *testing.T) {
	srv := NewBetaServer("8082")
	t.Cleanup(func() { srv.Shutdown() })
	// Generate several invalid validation attempts (empty token triggers missing_claim)
	// First, establish baseline
	w0 := httptest.NewRecorder()
	srv.router.ServeHTTP(w0, httptest.NewRequest("GET", "/api/v1/beta/metrics/violations", nil))

	for i := 0; i < 15; i++ {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/api/v1/token/validate", strings.NewReader(`{"token":""}`))
		req.Header.Set("Content-Type", "application/json")
		srv.router.ServeHTTP(w, req)
		if w.Code != 200 && w.Code != 400 {
			t.Fatalf("unexpected validate status %d", w.Code)
		}
		time.Sleep(10 * time.Millisecond) // spread out timestamps slightly
	}
	// Query violation metrics endpoint
	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest("GET", "/api/v1/beta/metrics/violations", nil)
	srv.router.ServeHTTP(w2, req2)
	if w2.Code != 200 {
		t.Fatalf("violation metrics status %d", w2.Code)
	}
	var resp struct {
		Success  bool              `json:"success"`
		Counters map[string]uint64 `json:"counters"`
		Total    uint64            `json:"total"`
		Anomaly  struct {
			Rate60    float64 `json:"rate_per_minute_60s"`
			Rate300   float64 `json:"rate_per_minute_300s"`
			Surge     bool    `json:"surge_60s"`
			Threshold float64 `json:"surge_threshold_per_minute"`
		} `json:"anomaly"`
	}
	if err := json.Unmarshal(w2.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !resp.Success {
		t.Fatalf("success=false")
	}
	if resp.Counters["missing_claim"] == 0 {
		t.Fatalf("expected missing_claim > 0")
	}
	if resp.Total == 0 {
		t.Fatalf("expected total > 0")
	}
	if resp.Anomaly.Rate60 <= 0 {
		t.Fatalf("expected positive 60s rate, got %f", resp.Anomaly.Rate60)
	}
	if resp.Anomaly.Rate300 <= 0 {
		t.Fatalf("expected positive 300s rate, got %f", resp.Anomaly.Rate300)
	}
	if resp.Anomaly.Threshold <= 0 {
		t.Fatalf("expected default threshold >0")
	}
}

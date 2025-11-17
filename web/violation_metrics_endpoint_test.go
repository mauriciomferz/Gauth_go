package web

import (
	"encoding/json"
	"net/http/httptest"
	"testing"
)

// TestViolationMetricsEndpoint verifies the JSON structure of /api/v1/beta/metrics/violations.
func TestViolationMetricsEndpoint(t *testing.T) {
	srv := NewBetaServer("")
	t.Cleanup(func() { srv.Shutdown() })
	// Exercise some token validation failure paths indirectly by calling /api/v1/token/validate with malformed payloads.
	// (Will increment missing_claim due to empty token value.)
	for i := 0; i < 3; i++ {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/api/v1/token/validate", nil)
		srv.router.ServeHTTP(w, req) // expect 400 but counters may increment
	}
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/beta/metrics/violations", nil)
	srv.router.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("expected 200 got %d", w.Code)
	}
	var payload struct {
		Success    bool              `json:"success"`
		Timestamp  string            `json:"timestamp"`
		Counters   map[string]uint64 `json:"counters"`
		Total      uint64            `json:"total"`
		Categories []string          `json:"categories"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if !payload.Success {
		t.Fatalf("success=false in response")
	}
	expectedCats := []string{"sig_invalid", "expired", "not_yet_valid", "issuer_mismatch", "replay_detected", "audience_mismatch", "missing_claim", "unknown", "capability_denied"}
	if len(payload.Categories) != len(expectedCats) {
		t.Fatalf("unexpected categories length %d", len(payload.Categories))
	}
	for i, cat := range expectedCats {
		if payload.Categories[i] != cat {
			t.Fatalf("category order mismatch index %d expected %s got %s", i, cat, payload.Categories[i])
		}
		if _, ok := payload.Counters[cat]; !ok {
			t.Fatalf("missing counter key %s", cat)
		}
	}
	// Total equals sum; allow both zero or positive.
	var sum uint64
	for _, v := range payload.Counters {
		sum += v
	}
	if sum != payload.Total {
		t.Fatalf("total mismatch sum=%d total=%d", sum, payload.Total)
	}
}

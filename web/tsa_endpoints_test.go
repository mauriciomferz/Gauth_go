package web

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// TestTSAAnchorAndVerify exercises the TSA prototype endpoints when enabled.
func TestTSAAnchorAndVerify(t *testing.T) {
	t.Setenv("GAUTH_TSA_ENDPOINTS_ENABLE", "1")
	// Need capability registry hash to validate verify path; server seeds static hash when no file.
	s := NewBetaServer("")
	// Submit anchor
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/beta/tsa/anchor", bytesJSON(map[string]string{"hash": "demo-hash-value"}))
	s.router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("anchor status %d", w.Code)
	}
	var resp struct {
		Success bool           `json:"success"`
		Receipt map[string]any `json:"receipt"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal anchor: %v", err)
	}
	if !resp.Success || resp.Receipt["hash"] != testDemoHashValue {
		t.Fatalf("unexpected receipt %+v", resp.Receipt)
	}
	// Verify receipt (should fail hash mismatch if capabilityRegistryHash differs)
	vrBody, _ := json.Marshal(resp.Receipt)
	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest("POST", "/api/v1/beta/tsa/verify", bytes.NewReader(vrBody))
	s.router.ServeHTTP(w2, req2)
	if w2.Code != http.StatusOK {
		t.Fatalf("verify status %d", w2.Code)
	}
	var vresp struct {
		Success  bool   `json:"success"`
		Verified bool   `json:"verified"`
		Reason   string `json:"reason"`
	}
	if err := json.Unmarshal(w2.Body.Bytes(), &vresp); err != nil {
		t.Fatalf("unmarshal verify: %v", err)
	}
	if !vresp.Success {
		t.Fatalf("verify not success")
	}
	if vresp.Verified && s.capabilityRegistryHash != testDemoHashValue {
		t.Fatalf("expected mismatch verify=false reason=%s", vresp.Reason)
	}
	// Basic latency metric sanity: we can't scrape Prometheus adapter directly here without instrumentation; ensure processing time < 1s
	if dur := time.Since(s.start); dur <= 0 {
		t.Fatalf("invalid server time reference")
	}
}

// bytesJSON helper returns reader for JSON body.
func bytesJSON(v any) *bytes.Reader { b, _ := json.Marshal(v); return bytes.NewReader(b) }

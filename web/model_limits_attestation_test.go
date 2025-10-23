package web

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

// TestModelLimitsAttestation exercises the consolidated attestation endpoint.
func TestModelLimitsAttestation(t *testing.T) {
	// Prepare temp paths
	auditFile, _ := os.CreateTemp(t.TempDir(), "audit_*.jsonl")
	anchorFile, _ := os.CreateTemp(t.TempDir(), "anchor_*.jsonl")
	limitsFile, _ := os.CreateTemp(t.TempDir(), "limits_*.json")
	_, _ = limitsFile.Write([]byte(`{"model_limits":{"demo":{"max_input_tokens":10}}}`))
	limitsFile.Close()
	os.Setenv("GAUTH_MODEL_LIMITS_PATH", limitsFile.Name())
	os.Setenv("GAUTH_MODEL_LIMIT_AUDIT_PATH", auditFile.Name())
	os.Setenv("GAUTH_MODEL_LIMIT_ANCHOR_PATH", anchorFile.Name())
	os.Setenv("GAUTH_MODEL_LIMIT_ANCHOR_INTERVAL", "1")
	os.Setenv("GAUTH_MODEL_LIMITS_STRICT_UNKNOWN", "1")
	bs := NewBetaServer("")
	// Trigger a couple exceed events to populate audit + anchor chains.
	for i := 0; i < 2; i++ {
		w := httptest.NewRecorder()
		body, _ := json.Marshal(map[string]any{"model_id": "demo", "input_tokens": 20})
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/model/validate", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		bs.router.ServeHTTP(w, req)
		if w.Code != 400 {
			t.Fatalf("expected exceed 400 got %d", w.Code)
		}
	}
	// Query attestation endpoint
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest(http.MethodGet, "/api/v1/model/limits/attestation", nil)
	bs.router.ServeHTTP(w2, req2)
	if w2.Code != 200 {
		t.Fatalf("attestation status=%d body=%s", w2.Code, w2.Body.String())
	}
	var resp struct {
		Success       bool           `json:"success"`
		Configured    bool           `json:"configured"`
		Snapshot      map[string]any `json:"snapshot"`
		Audit         map[string]any `json:"audit"`
		Anchor        map[string]any `json:"anchor"`
		StrictUnknown bool           `json:"strict_unknown"`
	}
	if err := json.Unmarshal(w2.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !resp.Success || !resp.Configured {
		t.Fatalf("expected success+configured got %+v", resp)
	}
	if resp.Snapshot["hash"] == "" {
		t.Fatalf("empty snapshot hash")
	}
	if resp.Audit["head_hash"] == "" {
		t.Fatalf("empty audit head hash")
	}
	if resp.Anchor["latest_hash"] == "" {
		t.Fatalf("empty anchor latest hash")
	}
	if resp.StrictUnknown != true {
		t.Fatalf("strict_unknown flag false")
	}
}

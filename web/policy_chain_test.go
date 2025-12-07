package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestPolicyChainEndpoint verifies the policy chain endpoint does not panic and returns success JSON.
func TestPolicyChainEndpoint(t *testing.T) {
	bs := NewBetaServer("")
	t.Cleanup(func() { bs.Shutdown() })
	rec := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/policy/chain?offset=0&limit=10", nil)
	bs.router.ServeHTTP(rec, req)
	if rec.Code != 200 {
		b := rec.Body.String()
		t.Fatalf("expected 200 OK, got %d body=%s", rec.Code, b)
	}
	var resp struct {
		Success  bool     `json:"success"`
		HeadHash string   `json:"head_hash"`
		Hashes   []string `json:"hashes"`
		Total    int      `json:"total"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid JSON: %v body=%s", err, rec.Body.String())
	}
	if !resp.Success {
		t.Fatalf("expected success true, got false body=%s", rec.Body.String())
	}
	if resp.Total != len(resp.Hashes) {
		t.Fatalf("expected total==len(hashes); total=%d hashes=%d", resp.Total, len(resp.Hashes))
	}
}

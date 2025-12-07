package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestPolicyHeadPolicies ensures the new endpoint returns the seeded policies.
func TestPolicyHeadPolicies(t *testing.T) {
	bs := NewBetaServer("")
	t.Cleanup(func() { bs.Shutdown() })
	rec := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/policy/head/policies", nil)
	bs.router.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	var out struct {
		Success  bool          `json:"success"`
		HeadHash string        `json:"head_hash"`
		Policies []interface{} `json:"policies"`
		Count    int           `json:"count"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("invalid JSON: %v body=%s", err, rec.Body.String())
	}
	if !out.Success {
		t.Fatalf("success=false body=%s", rec.Body.String())
	}
	if out.Count != len(out.Policies) {
		t.Fatalf("count mismatch: count=%d len=%d", out.Count, len(out.Policies))
	}
	// When seeding enabled, expect at least 2 policies.
	if out.HeadHash != "" && out.Count < 2 {
		t.Fatalf("expected seeded bundle to have >=2 policies, got %d", out.Count)
	}
}

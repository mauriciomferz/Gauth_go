package web

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mauriciomferz/AgentAuth/pkg/policy"
)

// TestPolicyVersionAndRollback covers auto-increment, rollback behavior and evaluation version tagging.
func TestPolicyVersionAndRollback(t *testing.T) {
	bs := NewTestServerNoSeed(t)

	// helper to append bundle
	append := func(id string, policies []policy.Policy) (hash string, version int) {
		payload := map[string]any{"id": id, "policies": policies}
		b, _ := json.Marshal(payload)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/policy/bundles", bytes.NewBuffer(b))
		req.Header.Set("X-Admin-Token", "test-admin")
		resp := httptest.NewRecorder()
		bs.router.ServeHTTP(resp, req)
		if resp.Code != 201 {
			t.Fatalf("append expected 201 got %d body=%s", resp.Code, resp.Body.String())
		}
		var out struct {
			Success       bool   `json:"success"`
			BundleHash    string `json:"bundle_hash"`
			PolicyVersion int    `json:"policy_version"`
		}
		if err := json.Unmarshal(resp.Body.Bytes(), &out); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if !out.Success {
			t.Fatalf("append failed")
		}
		return out.BundleHash, out.PolicyVersion
	}
	pol := []policy.Policy{{ID: "p1", Subjects: []string{"alice"}, Rules: []policy.Rule{{Actions: []string{"read"}, Resources: []string{"doc"}, Effect: policy.Allow}}}}
	_, v1 := append("b1", pol)
	if v1 != 1 {
		t.Fatalf("expected version 1 got %d", v1)
	}
	_, v2 := append("b2", pol)
	if v2 != 2 {
		t.Fatalf("expected version 2 got %d", v2)
	}
	_, v3 := append("b3", pol)
	if v3 != 3 {
		t.Fatalf("expected version 3 got %d", v3)
	}

	// Rollback to version 2 (requires admin token header)
	reqRB := httptest.NewRequest(http.MethodPost, "/api/v1/policy/rollback?version=2", nil)
	reqRB.Header.Set("X-Admin-Token", "test-admin")
	respRB := httptest.NewRecorder()
	bs.router.ServeHTTP(respRB, reqRB)
	if respRB.Code != 200 {
		t.Fatalf("rollback code=%d body=%s", respRB.Code, respRB.Body.String())
	}
	var rb struct {
		Success       bool `json:"success"`
		ActiveVersion int  `json:"active_version"`
	}
	if err := json.Unmarshal(respRB.Body.Bytes(), &rb); err != nil {
		t.Fatalf("rollback unmarshal: %v", err)
	}
	if rb.ActiveVersion != 2 {
		t.Fatalf("expected active_version 2 got %d", rb.ActiveVersion)
	}

	// Evaluate after rollback; expect policy_version=2
	evalPayload := []byte(`{"subject":"alice","action":"read","resource":"doc"}`)
	reqEval := httptest.NewRequest(http.MethodPost, "/api/v1/policy/evaluate", bytes.NewBuffer(evalPayload))
	respEval := httptest.NewRecorder()
	bs.router.ServeHTTP(respEval, reqEval)
	if respEval.Code != 200 {
		t.Fatalf("eval code=%d body=%s", respEval.Code, respEval.Body.String())
	}
	var evalResp struct {
		Success       bool `json:"success"`
		PolicyVersion int  `json:"policy_version"`
		Allow         bool `json:"allow"`
	}
	if err := json.Unmarshal(respEval.Body.Bytes(), &evalResp); err != nil {
		t.Fatalf("eval unmarshal: %v", err)
	}
	if !evalResp.Success || !evalResp.Allow {
		t.Fatalf("eval unexpected")
	}
	if evalResp.PolicyVersion != 2 {
		t.Fatalf("expected eval policy_version 2 got %d", evalResp.PolicyVersion)
	}

	// Append new bundle should clear rollback and set version 4 active
	_, v4 := append("b4", pol)
	if v4 != 4 {
		t.Fatalf("expected version 4 got %d", v4)
	}
	reqEval2 := httptest.NewRequest(http.MethodPost, "/api/v1/policy/evaluate", bytes.NewBuffer(evalPayload))
	respEval2 := httptest.NewRecorder()
	bs.router.ServeHTTP(respEval2, reqEval2)
	var evalResp2 struct {
		Success       bool `json:"success"`
		PolicyVersion int  `json:"policy_version"`
	}
	if err := json.Unmarshal(respEval2.Body.Bytes(), &evalResp2); err != nil {
		t.Fatalf("eval2 unmarshal: %v", err)
	}
	if evalResp2.PolicyVersion != 4 {
		t.Fatalf("expected eval policy_version 4 got %d", evalResp2.PolicyVersion)
	}
}

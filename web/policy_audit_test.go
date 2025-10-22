package web

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"
)

// TestPolicyRollbackAudit ensures audit entry emitted with expected metadata fields after rollback.
func TestPolicyRollbackAudit(t *testing.T) {
	tmpFile, err := os.CreateTemp(t.TempDir(), "chain_state_audit_*.json")
	if err != nil {
		t.Fatalf("temp file: %v", err)
	}
	path := tmpFile.Name()
	tmpFile.Close()
	os.Setenv("POLICY_CHAIN_STATE_PATH", path)
	defer os.Unsetenv("POLICY_CHAIN_STATE_PATH")

	s := newTestServer(t)
	// Append two bundles to have rollback target
	b1 := `{"id":"audit-b1","policies":[{"id":"p1","subjects":["a"],"rules":[{"actions":["x"],"resources":["r"],"effect":"allow"}]}]}`
	r1 := httptest.NewRequest(http.MethodPost, "/api/v1/beta/policy/bundles", bytes.NewBufferString(b1))
	r1.Header.Set("X-Admin-Token", "test-admin")
	w1 := httptest.NewRecorder()
	s.router.ServeHTTP(w1, r1)
	if w1.Code != 201 {
		t.Fatalf("append1 status %d body=%s", w1.Code, w1.Body.String())
	}

	b2 := `{"id":"audit-b2","policies":[{"id":"p2","subjects":["a"],"rules":[{"actions":["y"],"resources":["r"],"effect":"allow"}]}]}`
	r2 := httptest.NewRequest(http.MethodPost, "/api/v1/beta/policy/bundles", bytes.NewBufferString(b2))
	r2.Header.Set("X-Admin-Token", "test-admin")
	w2 := httptest.NewRecorder()
	s.router.ServeHTTP(w2, r2)
	if w2.Code != 201 {
		t.Fatalf("append2 status %d body=%s", w2.Code, w2.Body.String())
	}

	// Get active version
	tReq := httptest.NewRequest(http.MethodGet, "/api/v1/beta/policy/timeline", nil)
	tResp := httptest.NewRecorder()
	s.router.ServeHTTP(tResp, tReq)
	if tResp.Code != 200 {
		t.Fatalf("timeline status %d body=%s", tResp.Code, tResp.Body.String())
	}
	var tl struct {
		ActiveVersion int `json:"active_version"`
	}
	if err := json.Unmarshal(tResp.Body.Bytes(), &tl); err != nil {
		t.Fatalf("timeline unmarshal: %v", err)
	}
	if tl.ActiveVersion < 2 {
		t.Fatalf("need at least version 2, got %d", tl.ActiveVersion)
	}

	target := tl.ActiveVersion - 1
	// Perform rollback (with token)
	rbReq := httptest.NewRequest(http.MethodPost, "/api/v1/beta/policy/rollback?version="+strconv.Itoa(target), nil)
	rbReq.Header.Set("X-Admin-Token", "test-admin")
	rbResp := httptest.NewRecorder()
	s.router.ServeHTTP(rbResp, rbReq)
	if rbResp.Code != 200 {
		t.Fatalf("rollback status %d body=%s", rbResp.Code, rbResp.Body.String())
	}

	// Fetch audit log endpoint (assuming /api/v1/beta/audit exists); if not, test will fail prompting implementation.
	aReq := httptest.NewRequest(http.MethodGet, "/api/v1/beta/audit", nil)
	aResp := httptest.NewRecorder()
	s.router.ServeHTTP(aResp, aReq)
	if aResp.Code != 200 {
		t.Fatalf("audit endpoint status %d body=%s", aResp.Code, aResp.Body.String())
	}
	var audit struct {
		Entries []struct {
			Action   string            `json:"action"`
			Resource string            `json:"resource"`
			Outcome  string            `json:"outcome"`
			Meta     map[string]string `json:"meta"`
		} `json:"entries"`
	}
	if err := json.Unmarshal(aResp.Body.Bytes(), &audit); err != nil {
		t.Fatalf("audit unmarshal: %v", err)
	}
	found := false
	for _, e := range audit.Entries {
		if e.Action == "rollback" && e.Resource == "policy_chain" && e.Outcome == "success" {
			// Validate metadata keys
			if e.Meta["target_version"] == strconv.Itoa(target) && e.Meta["previous_active_version"] == strconv.Itoa(target+1) && e.Meta["head_hash"] != "" {
				found = true
				break
			}
		}
	}
	if !found {
		t.Fatalf("rollback audit entry with expected metadata not found entries=%v", audit.Entries)
	}
}

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

// TestPolicyRollbackRBAC verifies rollback requires admin token.
func TestPolicyRollbackRBAC(t *testing.T) {
	// Use temp persistence path to ensure clean registry state each phase
	tmpFile, err := os.CreateTemp(t.TempDir(), "chain_state_rbac_*.json")
	if err != nil {
		t.Fatalf("temp file: %v", err)
	}
	path := tmpFile.Name()
	tmpFile.Close()
	t.Setenv("POLICY_CHAIN_STATE_PATH", path)
	defer os.Unsetenv("POLICY_CHAIN_STATE_PATH")

	s := newTestServer(t)
	// Append two bundles to have rollback target > head
	b1 := `{"id":"rbac-b1","policies":[{"id":"p1","subjects":["a"],"rules":[{"actions":["x"],"resources":["r"],"effect":"allow"}]}]}`
	r1 := httptest.NewRequest(http.MethodPost, "/api/v1/policy/bundles", bytes.NewBufferString(b1))
	r1.Header.Set("X-Admin-Token", "test-admin")
	w1 := httptest.NewRecorder()
	s.router.ServeHTTP(w1, r1)
	if w1.Code != 201 {
		t.Fatalf("append1 status %d body=%s", w1.Code, w1.Body.String())
	}

	b2 := `{"id":"rbac-b2","policies":[{"id":"p2","subjects":["a"],"rules":[{"actions":["y"],"resources":["r"],"effect":"allow"}]}]}`
	r2 := httptest.NewRequest(http.MethodPost, "/api/v1/policy/bundles", bytes.NewBufferString(b2))
	r2.Header.Set("X-Admin-Token", "test-admin")
	w2 := httptest.NewRecorder()
	s.router.ServeHTTP(w2, r2)
	if w2.Code != 201 {
		t.Fatalf("append2 status %d body=%s", w2.Code, w2.Body.String())
	}

	// Determine current active version from timeline
	tReq := httptest.NewRequest(http.MethodGet, "/api/v1/policy/timeline", nil)
	tResp := httptest.NewRecorder()
	s.router.ServeHTTP(tResp, tReq)
	if tResp.Code != 200 {
		t.Fatalf("timeline status %d body=%s", tResp.Code, tResp.Body.String())
	}
	var tl struct {
		ActiveVersion int `json:"active_version"`
		Timeline      []struct {
			Version int `json:"version"`
		}
	}
	if err := json.Unmarshal(tResp.Body.Bytes(), &tl); err != nil {
		t.Fatalf("timeline unmarshal: %v", err)
	}
	if tl.ActiveVersion == 0 || len(tl.Timeline) < 2 {
		t.Fatalf("expected >=2 versions got active=%d timeline=%d", tl.ActiveVersion, len(tl.Timeline))
	}

	// Attempt rollback without token (target previous version)
	target := tl.ActiveVersion - 1
	rbReqNo := httptest.NewRequest(http.MethodPost, "/api/v1/policy/rollback?version="+intToStr(target), nil)
	rbRespNo := httptest.NewRecorder()
	s.router.ServeHTTP(rbRespNo, rbReqNo)
	if rbRespNo.Code != 403 {
		t.Fatalf("expected 403 without token got %d body=%s", rbRespNo.Code, rbRespNo.Body.String())
	}

	// Attempt rollback with admin token
	rbReq := httptest.NewRequest(http.MethodPost, "/api/v1/policy/rollback?version="+intToStr(target), nil)
	rbReq.Header.Set("X-Admin-Token", "test-admin")
	rbResp := httptest.NewRecorder()
	s.router.ServeHTTP(rbResp, rbReq)
	if rbResp.Code != 200 {
		t.Fatalf("expected 200 with token got %d body=%s", rbResp.Code, rbResp.Body.String())
	}

	// Verify active version decreased
	tReq2 := httptest.NewRequest(http.MethodGet, "/api/v1/policy/timeline", nil)
	tResp2 := httptest.NewRecorder()
	s.router.ServeHTTP(tResp2, tReq2)
	var tl2 struct {
		ActiveVersion int `json:"active_version"`
	}
	if err := json.Unmarshal(tResp2.Body.Bytes(), &tl2); err != nil {
		t.Fatalf("timeline2 unmarshal: %v", err)
	}
	if tl2.ActiveVersion != target {
		t.Fatalf("expected active version %d after rollback got %d", target, tl2.ActiveVersion)
	}
}

// intToStr small helper (avoid strconv import for minimal diff)
func intToStr(i int) string { return strconv.Itoa(i) }

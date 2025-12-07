package web

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestPolicyTimelineRollback verifies timeline reflects rollback state (rolled_back=true when active version != latest).
func TestPolicyTimelineRollback(t *testing.T) {
	srv := newTestServer(t)
	// Seed two bundles via append endpoint (requires admin token header when configured; tests use test server with token?)
	// Provide minimal valid bundles.
	b1 := `{"id":"b1","policies":[{"id":"p1","subjects":["a"],"rules":[{"actions":["x"],"resources":["r"],"effect":"allow"}]}]}`
	req1 := httptest.NewRequest(http.MethodPost, "/api/v1/policy/bundles", bytes.NewBufferString(b1))
	req1.Header.Set("X-Admin-Token", "test-admin")
	w1 := httptest.NewRecorder()
	srv.router.ServeHTTP(w1, req1)
	if w1.Code != 201 {
		t.Fatalf("expected 201 append1 got %d body=%s", w1.Code, w1.Body.String())
	}

	b2 := `{"id":"b2","policies":[{"id":"p2","subjects":["a"],"rules":[{"actions":["x"],"resources":["r"],"effect":"allow"}]}]}`
	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/policy/bundles", bytes.NewBufferString(b2))
	req2.Header.Set("X-Admin-Token", "test-admin")
	w2 := httptest.NewRecorder()
	srv.router.ServeHTTP(w2, req2)
	if w2.Code != 201 {
		t.Fatalf("expected 201 append2 got %d body=%s", w2.Code, w2.Body.String())
	}

	// Rollback to version 1
	rb := httptest.NewRequest(http.MethodPost, "/api/v1/policy/rollback?version=1", nil)
	rb.Header.Set("X-Admin-Token", "test-admin")
	wrb := httptest.NewRecorder()
	srv.router.ServeHTTP(wrb, rb)
	if wrb.Code != 200 {
		t.Fatalf("expected 200 rollback got %d body=%s", wrb.Code, wrb.Body.String())
	}

	// Fetch timeline
	reqT := httptest.NewRequest(http.MethodGet, "/api/v1/policy/timeline", nil)
	wT := httptest.NewRecorder()
	srv.router.ServeHTTP(wT, reqT)
	if wT.Code != 200 {
		t.Fatalf("expected 200 timeline got %d body=%s", wT.Code, wT.Body.String())
	}
	var resp struct {
		Success       bool `json:"success"`
		ActiveVersion int  `json:"active_version"`
		RolledBack    bool `json:"rolled_back"`
		Timeline      []struct {
			Version int  `json:"version"`
			Active  bool `json:"active"`
		} `json:"timeline"`
	}
	if err := json.Unmarshal(wT.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal timeline: %v", err)
	}
	if !resp.Success {
		t.Fatalf("timeline success=false")
	}
	if resp.ActiveVersion != 1 {
		t.Fatalf("expected active_version=1 got %d", resp.ActiveVersion)
	}
	if !resp.RolledBack {
		t.Fatalf("expected rolled_back=true")
	}
	// Ensure one of timeline entries has Active true (version 1)
	foundActive := false
	for _, e := range resp.Timeline {
		if e.Version == 1 && e.Active {
			foundActive = true
		}
	}
	if !foundActive {
		t.Fatalf("active marker for version 1 not found in timeline")
	}
}

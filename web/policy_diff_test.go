package web

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestPolicyDiffClassification ensures /policy/diff returns correct added/removed/changed sets.
func TestPolicyDiffClassification(t *testing.T) {
	s := newTestServer(t)
	// Append base bundle (policy p1 allow)
	b1 := `{"id":"b1","policies":[{"id":"p1","subjects":["alice"],"rules":[{"actions":["read"],"resources":["doc"],"effect":"allow"}]}]}`
	r1 := httptest.NewRequest(http.MethodPost, "/api/v1/policy/bundles", bytes.NewBufferString(b1))
	r1.Header.Set("X-Admin-Token", "test-admin")
	w1 := httptest.NewRecorder()
	s.router.ServeHTTP(w1, r1)
	if w1.Code != 201 {
		t.Fatalf("append b1 status=%d body=%s", w1.Code, w1.Body.String())
	}
	var out1 struct {
		PolicyVersion int `json:"policy_version"`
		Success       bool
	}
	if err := json.Unmarshal(w1.Body.Bytes(), &out1); err != nil {
		t.Fatalf("unmarshal b1: %v", err)
	}
	if !out1.Success {
		t.Fatalf("append b1 success=false")
	}

	// Append second bundle with modified p1 and added p2
	b2 := `{"id":"b2","policies":[{"id":"p1","subjects":["alice"],"rules":[{"actions":["read"],"resources":["doc2"],"effect":"allow"}]},{"id":"p2","subjects":["bob"],"rules":[{"actions":["write"],"resources":["doc"],"effect":"deny"}]}]}`
	r2 := httptest.NewRequest(http.MethodPost, "/api/v1/policy/bundles", bytes.NewBufferString(b2))
	r2.Header.Set("X-Admin-Token", "test-admin")
	w2 := httptest.NewRecorder()
	s.router.ServeHTTP(w2, r2)
	if w2.Code != 201 {
		t.Fatalf("append b2 status=%d body=%s", w2.Code, w2.Body.String())
	}
	var out2 struct {
		PolicyVersion int `json:"policy_version"`
		Success       bool
	}
	if err := json.Unmarshal(w2.Body.Bytes(), &out2); err != nil {
		t.Fatalf("unmarshal b2: %v", err)
	}
	if !out2.Success {
		t.Fatalf("append b2 success=false")
	}

	// Diff using actual recorded versions (robust whether seed bundle exists or not)
	fromVer := out1.PolicyVersion
	toVer := out2.PolicyVersion
	diffReq := httptest.NewRequest(http.MethodGet, "/api/v1/policy/diff?from="+itoa(fromVer)+"&to="+itoa(toVer), nil)
	wDiff := httptest.NewRecorder()
	s.router.ServeHTTP(wDiff, diffReq)
	if wDiff.Code != 200 {
		t.Fatalf("diff status=%d body=%s", wDiff.Code, wDiff.Body.String())
	}
	var resp struct {
		Success bool `json:"success"`
		Diff    struct {
			Added []struct {
				ID string `json:"id"`
			} `json:"added"`
			Removed []struct {
				ID string `json:"id"`
			} `json:"removed"`
			Changed []struct {
				ID string `json:"id"`
			} `json:"changed"`
			Unchanged []struct {
				ID string `json:"id"`
			} `json:"unchanged"`
			FromVersion int `json:"from_version"`
			ToVersion   int `json:"to_version"`
		} `json:"diff"`
	}
	if err := json.Unmarshal(wDiff.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal diff: %v", err)
	}
	if !resp.Success {
		t.Fatalf("diff success=false")
	}
	if resp.Diff.FromVersion != fromVer || resp.Diff.ToVersion != toVer {
		t.Fatalf("unexpected versions %+v expected from=%d to=%d", resp.Diff, fromVer, toVer)
	}
	// Validate classification
	if len(resp.Diff.Added) != 1 || resp.Diff.Added[0].ID != "p2" {
		t.Fatalf("expected p2 added got %+v", resp.Diff.Added)
	}
	if len(resp.Diff.Removed) != 0 {
		t.Fatalf("expected no removed policies got %+v", resp.Diff.Removed)
	}
	if len(resp.Diff.Changed) != 1 || resp.Diff.Changed[0].ID != "p1" {
		t.Fatalf("expected p1 changed got %+v", resp.Diff.Changed)
	}
	if len(resp.Diff.Unchanged) != 0 {
		t.Fatalf("expected 0 unchanged got %+v", resp.Diff.Unchanged)
	}

	// Negative identical version diff should error
	badReq := httptest.NewRequest(http.MethodGet, "/api/v1/policy/diff?from="+itoa(toVer)+"&to="+itoa(toVer), nil)
	wBad := httptest.NewRecorder()
	s.router.ServeHTTP(wBad, badReq)
	if wBad.Code == 200 {
		t.Fatalf("expected error for identical versions, got 200 body=%s", wBad.Body.String())
	}
}

// simple int to string helper (avoid importing strconv in test changes)
func itoa(i int) string { return fmt.Sprintf("%d", i) }

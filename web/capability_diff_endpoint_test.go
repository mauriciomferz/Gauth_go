package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mauriciomferz/Gauth_go/internal/capability"
)

// helper to perform GET request against server.
func perform(server *BetaServer, path string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", path, nil)
	server.router.ServeHTTP(w, req)
	return w
}

func TestCapabilityDiffAddedRemovedModified(t *testing.T) {
	srv := NewBetaServer("0")
	t.Cleanup(func() { srv.Shutdown() })
	// Baseline capabilities: A, B (overwrite static seed after server init)
	capability.Reset([]capability.Capability{
		{ID: "cap.A", Version: "1.0", Stable: true},
		{ID: "cap.B", Version: "1.0", Stable: true},
	})
	// Initial diff (no since) => empty diff
	w := perform(srv, "/api/v1/capabilities/diff")
	if w.Code != 200 {
		t.Fatalf("unexpected status baseline: %d body=%s", w.Code, w.Body.String())
	}
	var baselineResp struct {
		BaseHash    string `json:"base_hash"`
		CurrentHash string `json:"current_hash"`
		Added       []any  `json:"added"`
		Removed     []any  `json:"removed"`
		Modified    []any  `json:"modified"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &baselineResp); err != nil {
		t.Fatalf("unmarshal baseline: %v", err)
	}
	if len(baselineResp.Added) != 0 || len(baselineResp.Removed) != 0 || len(baselineResp.Modified) != 0 {
		t.Fatalf("expected empty diff arrays baseline; got added=%d removed=%d modified=%d", len(baselineResp.Added), len(baselineResp.Removed), len(baselineResp.Modified))
	}
	baseHash := baselineResp.CurrentHash
	// Mutate registry: modify A version, remove B, add C
	capability.Reset([]capability.Capability{
		{ID: "cap.A", Version: "2.0", Stable: true},
		{ID: "cap.C", Version: "1.0", Stable: true},
	})
	// Diff against baseline
	w2 := perform(srv, "/api/v1/capabilities/diff?since="+baseHash)
	if w2.Code != 200 {
		t.Fatalf("unexpected status mutated: %d body=%s", w2.Code, w2.Body.String())
	}
	var diffResp struct {
		Added []struct {
			ID string `json:"id"`
		} `json:"added"`
		Removed []struct {
			ID string `json:"id"`
		} `json:"removed"`
		Modified []struct {
			ID string `json:"id"`
		} `json:"modified"`
	}
	if err := json.Unmarshal(w2.Body.Bytes(), &diffResp); err != nil {
		t.Fatalf("unmarshal diff: %v", err)
	}
	// Assertions
	if len(diffResp.Added) != 1 || diffResp.Added[0].ID != "cap.C" {
		t.Fatalf("expected added cap.C; got %#v", diffResp.Added)
	}
	if len(diffResp.Removed) != 1 || diffResp.Removed[0].ID != "cap.B" {
		t.Fatalf("expected removed cap.B; got %#v", diffResp.Removed)
	}
	foundModified := false
	for _, m := range diffResp.Modified {
		if m.ID == "cap.A" {
			foundModified = true
		}
	}
	if !foundModified {
		t.Fatalf("expected modified cap.A; got %#v", diffResp.Modified)
	}
}

func TestCapabilityDiffUnknownBaseline(t *testing.T) {
	capability.Reset([]capability.Capability{{ID: "cap.X", Version: "1.0", Stable: true}})
	srv := NewBetaServer("0")
	t.Cleanup(func() { srv.Shutdown() })
	w := perform(srv, "/api/v1/capabilities/diff?since=sha256:deadbeef")
	if w.Code != 404 {
		t.Fatalf("expected 404 for unknown baseline; got %d body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "version_not_found") {
		t.Fatalf("expected error code in body; got %s", w.Body.String())
	}
}

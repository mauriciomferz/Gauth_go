package anchor_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	webpkg "github.com/mauriciomferz/Gauth_go/web"
)

// perform helper similar to capabilities tests
func perform(server *webpkg.BetaServer, method, path string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, nil)
	w := httptest.NewRecorder()
	server.Engine().ServeHTTP(w, req)
	return w
}

func TestAnchorEndpointsFullCycle(t *testing.T) {
	// Enable anchoring + file emission
	t.Setenv("GAUTH_CAPABILITY_ANCHOR_ENABLE", "1")
	t.Setenv("GAUTH_ANCHOR_PROVIDER", "memory")
	anchorFile := filepath.Join(t.TempDir(), "cap-anchor.json")
	t.Setenv("GAUTH_CAP_ANCHOR_FILE_PATH", anchorFile)
	t.Setenv("GAUTH_CAP_ANCHOR_WRITE_INTERVAL", "1ms")
	// Provide capabilities file for registry hash population
	capFile := filepath.Join(t.TempDir(), "caps.json")
	if err := os.WriteFile(capFile, []byte(`{"schema_version":1,"capabilities":[{"id":"demo.cap","version":"1.0","status":"active"}]}`), 0o600); err != nil {
		t.Fatalf("write capabilities file: %v", err)
	}
	t.Setenv("GAUTH_CAPABILITIES_PATH", capFile)
	srv := webpkg.NewBetaServer(":0")
	// Force reload to populate registry hash and trigger emission
	_ = perform(srv, http.MethodPost, "/api/v1/beta/capabilities/reload")
	// POST anchor
	wPost := perform(srv, http.MethodPost, "/api/v1/beta/capabilities/anchor")
	if wPost.Code != http.StatusOK {
		t.Fatalf("anchor POST status=%d body=%s", wPost.Code, wPost.Body.String())
	}
	var postResp struct {
		Success bool   `json:"success"`
		Hash    string `json:"hash"`
		Total   int    `json:"total"`
	}
	if err := json.Unmarshal(wPost.Body.Bytes(), &postResp); err != nil {
		t.Fatalf("unmarshal post: %v", err)
	}
	if !postResp.Success || postResp.Hash == "" || postResp.Total < 1 {
		t.Fatalf("unexpected post resp %+v", postResp)
	}
	// Latest
	wLatest := perform(srv, http.MethodGet, "/api/v1/beta/capabilities/anchor/latest")
	if wLatest.Code != http.StatusOK {
		t.Fatalf("latest status=%d", wLatest.Code)
	}
	// Material
	// Allow a tiny delay for write interval logic
	time.Sleep(2 * time.Millisecond)
	wMaterial := perform(srv, http.MethodGet, "/api/v1/beta/capabilities/anchor/material")
	if wMaterial.Code != http.StatusOK {
		t.Fatalf("material status=%d body=%s", wMaterial.Code, wMaterial.Body.String())
	}
	var matResp struct {
		Success    bool   `json:"success"`
		Configured bool   `json:"configured"`
		Emitted    bool   `json:"emitted"`
		Registry   string `json:"registry_hash"`
	}
	if err := json.Unmarshal(wMaterial.Body.Bytes(), &matResp); err != nil {
		t.Fatalf("unmarshal material: %v body=%s", err, wMaterial.Body.String())
	}
	if !matResp.Success || !matResp.Configured || !matResp.Emitted || matResp.Registry == "" {
		t.Fatalf("unexpected material resp %+v", matResp)
	}
	// Status
	wStatus := perform(srv, http.MethodGet, "/api/v1/beta/capabilities/anchor/status")
	if wStatus.Code != http.StatusOK {
		t.Fatalf("status endpoint code=%d body=%s", wStatus.Code, wStatus.Body.String())
	}
	var stResp struct {
		Success    bool   `json:"success"`
		Registry   string `json:"registry_hash"`
		Configured bool   `json:"configured"`
	}
	if err := json.Unmarshal(wStatus.Body.Bytes(), &stResp); err != nil {
		t.Fatalf("unmarshal status: %v", err)
	}
	if !stResp.Success || stResp.Registry == "" || !stResp.Configured {
		t.Fatalf("unexpected status resp %+v", stResp)
	}
}

func TestAnchorPostDisabled(t *testing.T) {
	// Ensure disabled env (unset) -> 403
	srv := webpkg.NewBetaServer(":0")
	w := perform(srv, http.MethodPost, "/api/v1/beta/capabilities/anchor")
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 got %d", w.Code)
	}
	var resp struct {
		Success bool   `json:"success"`
		Code    string `json:"code"`
		Error   string `json:"error"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Success || resp.Code != "anchoring_disabled" || resp.Error != "capability_anchor_disabled" {
		t.Fatalf("unexpected error payload %+v", resp)
	}
}

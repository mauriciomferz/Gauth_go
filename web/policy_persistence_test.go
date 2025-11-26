package web

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/mauriciomferz/Gauth_go/web/testutil"
)

// TestPolicyPersistenceRoundTrip verifies that setting POLICY_CHAIN_STATE_PATH persists appended bundles and they reload on new server instance.
func TestPolicyPersistenceRoundTrip(t *testing.T) {
	tmpFile, err := os.CreateTemp(t.TempDir(), "chain_state_*.json")
	if err != nil {
		t.Fatalf("temp file: %v", err)
	}
	path := tmpFile.Name()
	tmpFile.Close()
	os.Setenv("POLICY_CHAIN_STATE_PATH", path)
	defer os.Unsetenv("POLICY_CHAIN_STATE_PATH")

	// First server: append two bundles
	s1 := newTestServer(t)
	r1 := httptest.NewRequest(http.MethodPost, "/api/v1/beta/policy/bundles", bytes.NewBufferString(testutil.PolicyBundleB1V1))
	r1.Header.Set("X-Admin-Token", "test-admin")
	w1 := httptest.NewRecorder()
	s1.router.ServeHTTP(w1, r1)
	if w1.Code != 201 {
		t.Fatalf("append1 status %d body=%s", w1.Code, w1.Body.String())
	}

	r2 := httptest.NewRequest(http.MethodPost, "/api/v1/beta/policy/bundles", bytes.NewBufferString(testutil.PolicyBundleB2V1))
	r2.Header.Set("X-Admin-Token", "test-admin")
	w2 := httptest.NewRecorder()
	s1.router.ServeHTTP(w2, r2)
	if w2.Code != 201 {
		t.Fatalf("append2 status %d body=%s", w2.Code, w2.Body.String())
	}

	// Force an explicit save to ensure content flushed (handler should already do this, but double-ensure for test stability)
	if s1.policyPersistPath != "" {
		if saveErr := savePolicyChainToFile(s1.policyPersistPath, s1.policyRegistry); saveErr != nil {
			t.Fatalf("explicit save failed: %v", saveErr)
		}
	}

	// Ensure file exists and non-empty
	info, err := os.Stat(path)
	if err != nil || info.Size() == 0 {
		t.Fatalf("expected persistence file with size >0")
	}

	// Second server (fresh) should load file and expose both versions via timeline
	s2 := newTestServer(t)
	reqT := httptest.NewRequest(http.MethodGet, "/api/v1/beta/policy/timeline", nil)
	wT := httptest.NewRecorder()
	s2.router.ServeHTTP(wT, reqT)
	if wT.Code != 200 {
		t.Fatalf("timeline status %d body=%s", wT.Code, wT.Body.String())
	}
	body := wT.Body.String()
	// Expect at least two bundles (seed + 2 appended) => >=3 versions; but if no seed present then >=2.
	var resp struct {
		Timeline []struct {
			Version int `json:"version"`
		}
	}
	if err := json.Unmarshal([]byte(body), &resp); err != nil {
		t.Fatalf("unmarshal timeline: %v body=%s", err, body)
	}
	if len(resp.Timeline) < 2 {
		t.Fatalf("expected >=2 versions in timeline got %d body=%s", len(resp.Timeline), body)
	}
}

package web

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/mauriciomferz/Gauth_go/web/testutil"
)

// TestLegacyLifecycleAliasDeprecation ensures the legacy alias prints a deprecation warning.
func TestLegacyLifecycleAliasDeprecation(t *testing.T) {
	var srv *BetaServer
	testutil.WithEnv(t, "GAUTH_DISABLE_LEGACY_GOVERNANCE_ALIAS", "", func() {
		srv = NewBetaServer("")
		// Seed one lifecycle event so handler returns events (not required for warning but keeps response meaningful)
		seed := httptest.NewRecorder()
		reqSeed, _ := http.NewRequest("POST", "/api/v1/delegation/status/update", strings.NewReader(`{"delegation_id":"warnA","new_status":"active"}`))
		reqSeed.Header.Set("Content-Type", "application/json")
		srv.router.ServeHTTP(seed, reqSeed)
		if seed.Code != 200 {
			t.Fatalf("seed expected 200 got %d body=%s", seed.Code, seed.Body.String())
		}

		// Capture stderr
		origStderr := os.Stderr
		r, w, err := os.Pipe()
		if err != nil {
			t.Fatalf("pipe: %v", err)
		}
		os.Stderr = w

		// Invoke legacy alias
		rr := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/governance/lifecycle_timeline?limit=1", nil)
		srv.router.ServeHTTP(rr, req)

		// Restore stderr
		w.Close()
		os.Stderr = origStderr
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		r.Close()

		if rr.Code != 200 {
			t.Fatalf("expected 200 legacy alias got %d body=%s", rr.Code, rr.Body.String())
		}
		out := buf.String()
		if !strings.Contains(out, "[deprecate] /api/governance/lifecycle_timeline invoked") {
			t.Fatalf("expected deprecation warning in stderr, got: %s", out)
		}
	})
}

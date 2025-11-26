package web

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mauriciomferz/Gauth_go/web/testutil"
)

// TestLegacyAliasDisabled ensures setting GAUTH_DISABLE_LEGACY_GOVERNANCE_ALIAS=1 prevents route registration.
func TestLegacyAliasDisabled(t *testing.T) {
	testutil.WithEnv(t, "GAUTH_DISABLE_LEGACY_GOVERNANCE_ALIAS", "1", func() {
		srv := NewBetaServer("")
		t.Cleanup(func() { srv.Shutdown() })
		rr := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/governance/lifecycle_timeline", nil)
		srv.router.ServeHTTP(rr, req)
		if rr.Code != 404 {
			t.Fatalf("expected 404 for disabled legacy alias got %d body=%s", rr.Code, rr.Body.String())
		}
	})
}

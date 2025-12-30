package web

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mauriciomferz/AgentAuth/web/testutil"
)

// TestLegacyAliasDisabled ensures setting AGENTAUTH_DISABLE_LEGACY_GOVERNANCE_ALIAS=1 prevents route registration.
func TestLegacyAliasDisabled(t *testing.T) {
	testutil.WithEnv(t, "AGENTAUTH_DISABLE_LEGACY_GOVERNANCE_ALIAS", "1", func() {
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

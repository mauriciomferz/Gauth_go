package web

import (
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"

	imetrics "github.com/mauriciomferz/AgentAuth/internal/metrics"
	delegation "github.com/mauriciomferz/AgentAuth/pkg/delegation"
)

// TestRevocationAutoSignPrometheusMetrics ensures the Prometheus exposition includes the revocation auto-sign counters.
func TestRevocationAutoSignPrometheusMetrics(t *testing.T) {
	t.Setenv("GAUTH_OTEL_METRICS_ENABLE", "0") // keep deterministic & fast
	s := NewBetaServerWithMetrics(":0", imetrics.NewMemory())
	t.Cleanup(func() { s.Shutdown() })
	defer s.Shutdown()
	// Ensure revocation chain exists
	if s.revocationChain == nil {
		s.revocationChain = delegation.NewRevocationChain()
	}

	// 1. Empty chain rotation -> skipped_empty++
	triggerRevocationAutoSign(s)
	// 2. Add first event -> emitted++
	_, _ = s.revocationChain.Append(delegation.RevocationEvent{ID: "rev-1", DelegationID: "del-1", Reason: string(delegation.RevocationReasonUserRequest)})
	triggerRevocationAutoSign(s)
	// 3. Duplicate rotation -> skipped_duplicate++
	triggerRevocationAutoSign(s)
	// 4. Add second event -> emitted++ (emitted=2, skipped_empty=1, skipped_duplicate=1)
	_, _ = s.revocationChain.Append(delegation.RevocationEvent{ID: "rev-2", DelegationID: "del-2", Reason: string(delegation.RevocationReasonUserRequest)})
	triggerRevocationAutoSign(s)

	req := httptest.NewRequest("GET", "/api/v1/beta/metrics/revocation/auto-sign/prometheus", nil)
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("unexpected status %d", w.Code)
	}
	body := w.Body.String()
	// Basic presence checks with exact expected final values.
	expected := map[string]int64{
		"gauth_revocation_auto_sign_emitted":           2,
		"gauth_revocation_auto_sign_skipped_empty":     1,
		"gauth_revocation_auto_sign_skipped_duplicate": 1,
	}
	for name, val := range expected {
		needle := name + " " + intToString(val)
		if !strings.Contains(body, needle) {
			t.Fatalf("missing metric line %q in body:\n%s", needle, body)
		}
	}
}

// intToString avoids strconv import just for tests given style elsewhere; small helper.
func intToString(v int64) string { return fmt.Sprintf("%d", v) }

package web

import (
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

// TestSemanticRatesPrometheus ensures semantic rate metrics are exposed in Prometheus format.
func TestSemanticRatesPrometheus(t *testing.T) {
	os.Setenv("GAUTH_SEMANTIC_PERSIST_PATH", "") // disable persistence interference
	srv := NewBetaServer("8123")
	t.Cleanup(func() { srv.Shutdown() })
	if srv.rfc0111Service == nil {
		t.Fatalf("RFC0111 service not initialized; GAUTH_DISABLE_RFC0111_SERVICE should not be set")
	}
	// Inject Mock Service
	mockSvc := &mockRFC0111Service{
		snapshots: []map[string]uint64{
			{"scope_violation": 10},
			{"scope_violation": 20}, // Rate ~10/sec (if delay 1s)
			{"scope_violation": 30},
		},
	}
	srv.rfc0111Service = mockSvc
	srv.semanticHandler.Service = mockSvc

	// Update to establish baseline
	srv.semanticHandler.Update()

	// Sleep and Update again to establish rate
	time.Sleep(1100 * time.Millisecond)
	srv.semanticHandler.Update()
	// Prometheus endpoint output
	wp := httptest.NewRecorder()
	rp := httptest.NewRequest("GET", "/api/v1/beta/metrics/poa/semantics/prometheus", nil)
	srv.router.ServeHTTP(wp, rp)
	out := wp.Body.String()
	// Expect rate metric headers and at least one category rate line
	if !strings.Contains(out, "gauth_poa_semantic_rate_60s") {
		t.Fatalf("expected gauth_poa_semantic_rate_60s metrics in output")
	}
	if !strings.Contains(out, "gauth_poa_semantic_rate_300s") {
		t.Fatalf("expected gauth_poa_semantic_rate_300s metrics in output")
	}
}

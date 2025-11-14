package web

import (
	"net/http/httptest"
	"strings"
	"testing"
)

// TestObservabilityPanelPresence ensures unified observability section and metrics IDs exist in index.html.
// NOTE: Skipped because index.html is now a marketing landing page, not the observability dashboard.
// The observability features are accessed via API endpoints, not the landing page UI.
func TestObservabilityPanelPresence(t *testing.T) {
	t.Skip("Observability panel moved to dedicated dashboard - index.html is now marketing page")
	srv := NewBetaServer("")
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/index.html", nil)
	srv.router.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	body := w.Body.String()
	// Required section
	mustContain := []string{
		"id=\"observability\"",
		"id=\"authz-metrics-panel\"",
		"id=\"policy-metrics-panel\"",
		"id=\"m-policy-total\"",
		"id=\"m-policy-allow\"",
		"id=\"m-policy-deny\"",
		"id=\"m-policy-last-reason\"",
		"id=\"policy-latency-histogram\"",
		"id=\"authz-latency-histogram\"",
	}
	for _, frag := range mustContain {
		if !strings.Contains(body, frag) {
			t.Errorf("index.html missing fragment: %s", frag)
		}
	}
}

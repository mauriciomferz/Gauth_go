package web

import (
	"net/http/httptest"
	"strings"
	"testing"
)

// TestObservabilityPanelPresence ensures unified observability section and metrics IDs exist in index.html.
func TestObservabilityPanelPresence(t *testing.T) {
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

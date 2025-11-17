package web

import (
	"net/http/httptest"
	"strings"
	"testing"
)

// TestLatencyPercentilesEndpoint ensures the latency percentile endpoint responds with expected shape.
func TestLatencyPercentilesEndpoint(t *testing.T) {
	srv := NewBetaServer("0")
	t.Cleanup(func() { srv.Shutdown() })
	w := httptest.NewRecorder()
	srv.router.ServeHTTP(w, httptest.NewRequest("GET", "/api/v1/beta/metrics/latency", nil))
	if w.Code != 200 {
		t.Fatalf("status %d", w.Code)
	}
	body := w.Body.String()
	if body == "" || body[0] != '{' {
		t.Fatalf("unexpected body: %q", body)
	}
	// Basic keys presence checks (friendly names may appear even with empty observations)
	wantKeys := []string{"generated_at", "histograms"}
	for _, k := range wantKeys {
		if !strings.Contains(body, k) {
			t.Fatalf("missing key %s in response: %s", k, body)
		}
	}
}

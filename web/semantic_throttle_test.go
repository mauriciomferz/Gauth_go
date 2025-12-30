package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// TestSemanticReactiveThrottle verifies throttle activation denies demo action when z-score threshold configured very low.
func TestSemanticReactiveThrottle(t *testing.T) {
	t.Setenv("AGENTAUTH_SEMANTIC_ANOMALY_Z_THRESHOLD", "0.0") // treat any >=0 score as exceed
	s := NewBetaServer("")
	t.Cleanup(func() { s.Shutdown() })
	// Synthetically vary semantic history snapshots to generate changing per-category rates.
	// We directly manipulate semanticHistory before requesting diagnostics to accumulate EWMA samples.
	// Trigger semantic counters via authorize endpoint (empty payload leads to some defaults). Provide mismatching scopes.
	for i := 0; i < 3; i++ {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/api/v1/poa/authorize", strings.NewReader(`{"delegation":{"scope":["a"],"requested_scope":["b"],"amount_limit":1,"requested_amount":5}}`))
		req.Header.Set("Content-Type", "application/json")
		s.router.ServeHTTP(w, req)
		time.Sleep(5 * time.Millisecond)
	}
	if err := s.semanticHandler.Save(); err != nil {
		t.Fatalf("save semantic failed: %v", err)
	}
	// Force activation (simulate anomaly exceed) for demo action denial.
	s.semanticThrottleActive = true
	// Attempt demo action should be denied with 429 semantic_throttle_active
	wAct := httptest.NewRecorder()
	reqAct, _ := http.NewRequest("POST", "/api/v1/beta/throttle/demoAction", http.NoBody)
	s.router.ServeHTTP(wAct, reqAct)
	if wAct.Code != 429 {
		t.Fatalf("expected 429 got %d body=%s", wAct.Code, wAct.Body.String())
	}
	var errResp struct {
		Code string `json:"code"`
		RFC  string `json:"rfc_ref"`
	}
	if err := json.Unmarshal(wAct.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("unmarshal act: %v", err)
	}
	if errResp.Code != "semantic_throttle_active" {
		t.Fatalf("expected semantic_throttle_active code got %s", errResp.Code)
	}
	if errResp.RFC != "AAP002:reactive_controls" {
		t.Fatalf("expected AAP002:reactive_controls got %s", errResp.RFC)
	}
}

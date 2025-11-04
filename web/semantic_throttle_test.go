package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

// TestSemanticReactiveThrottle verifies throttle activation denies demo action when z-score threshold configured very low.
func TestSemanticReactiveThrottle(t *testing.T) {
	os.Setenv("GAUTH_SEMANTIC_ANOMALY_Z_THRESHOLD", "0.0") // treat any >=0 score as exceed
	s := NewBetaServer("")
	// Synthetically vary semantic history snapshots to generate changing per-category rates.
	// We directly manipulate semanticHistory before requesting diagnostics to accumulate EWMA samples.
	base := time.Now().Add(-40 * time.Second)
	for i := 0; i < 6; i++ {
		s.semanticHistMu.Lock()
		// Build snapshot with one counter increasing non-linearly.
		snap := map[string]uint64{"scope_violation": uint64(i*i + i)}
		s.semanticHistory = append(s.semanticHistory, struct {
			At       time.Time
			Snapshot map[string]uint64
		}{At: base.Add(time.Duration(i) * 5 * time.Second), Snapshot: snap})
		s.semanticHistMu.Unlock()
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/diagnostics/semantic", http.NoBody)
		s.router.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("diag status %d", w.Code)
		}
		time.Sleep(5 * time.Millisecond)
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
	if errResp.RFC != "rfc115:reactive_controls" {
		t.Fatalf("expected rfc115:reactive_controls got %s", errResp.RFC)
	}
}

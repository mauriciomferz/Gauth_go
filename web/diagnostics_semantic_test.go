package web

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mauriciomferz/AgentAuth/pkg/agentauth_aap_001"
)

// TestSemanticDiagnostics_Unwired verifies payload fields when no AAP001 service is wired (wired=false).
func TestSemanticDiagnostics_Unwired(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("AGENTAUTH_DISABLE_AAP001_SERVICE", "1")
	s := NewBetaServer("")
	t.Cleanup(func() { s.Shutdown() })
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/diagnostics/semantic", nil)
	s.router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("diagnostics status=%d body=%s", w.Code, w.Body.String())
	}
	var body struct {
		Wired            bool             `json:"wired"`
		Timestamp        string           `json:"timestamp"`
		History          []map[string]any `json:"history"`
		Anomaly          map[string]any   `json:"anomaly"`
		IntegrityStatus  string           `json:"integrity_status"`
		HistoryWindowCap int              `json:"history_window_cap"`
		PrevHash         string           `json:"prev_hash"`
		CurrentHash      string           `json:"current_hash"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if body.Wired {
		t.Fatalf("unexpected wired state: %+v", body)
	}
	if body.Timestamp == "" {
		t.Fatalf("timestamp missing")
	}
	if body.IntegrityStatus == "" || body.IntegrityStatus != "unconfigured" {
		t.Fatalf("unexpected integrity_status: %s", body.IntegrityStatus)
	}
	if body.Anomaly == nil || body.Anomaly["rate_per_minute_60s"] == nil || body.Anomaly["scores"] == nil {
		t.Fatalf("anomaly section missing expected keys: %+v", body.Anomaly)
	}
}

// TestSemanticDiagnostics_UnwiredStrictUnavailable verifies error response when strict wiring is required.
func TestSemanticDiagnostics_UnwiredStrictUnavailable(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("AGENTAUTH_DISABLE_AAP001_SERVICE", "1")
	t.Setenv("AGENTAUTH_SEMANTIC_DIAGNOSTICS_REQUIRE_WIRED", "1")
	s := NewBetaServer("")
	t.Cleanup(func() { s.Shutdown() })
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/diagnostics/semantic", nil)
	s.router.ServeHTTP(w, req)
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 got %d body=%s", w.Code, w.Body.String())
	}
	var body ErrorEnvelope
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if body.Code != "semantic_metrics_unavailable" {
		t.Fatalf("unexpected body: %+v", body)
	}
}

// mockAAP001Service provides deterministic counter snapshots for wired diagnostics testing.
type mockAAP001Service struct {
	snapshots []map[string]uint64
	idx       int
}

// Revocation workflow methods (no-op) to satisfy extended interface for tests.
// They return nil to simulate successful operations without affecting snapshots.
func (m *mockAAP001Service) InitiateRevocation(ctx context.Context, req agentauth_aap_001.RevocationRequest) error {
	return nil
}
func (m *mockAAP001Service) ApproveRevocation(ctx context.Context, poaID, approver string) error {
	return nil
}
func (m *mockAAP001Service) CancelRevocation(ctx context.Context, poaID, actor string) error {
	return nil
}

func (m *mockAAP001Service) SemanticSnapshot() map[string]uint64 {
	if len(m.snapshots) == 0 {
		return map[string]uint64{}
	}
	snap := m.snapshots[m.idx%len(m.snapshots)]
	m.idx++
	out := make(map[string]uint64, len(snap))
	for k, v := range snap {
		out[k] = v
	}
	return out
}

// TestSemanticDiagnostics_Wired verifies populated counters, history growth, non-empty anomaly score and hash chain fields.
func TestSemanticDiagnostics_Wired(t *testing.T) {
	gin.SetMode(gin.TestMode)
	persistFile := t.TempDir() + "/semantic_counters.json"
	t.Setenv("AGENTAUTH_SEMANTIC_PERSIST_PATH", persistFile)
	s := NewBetaServer("")
	t.Cleanup(func() { s.Shutdown() })
	// Inject mock service with stable history then a massive spike to trigger anomaly.
	// Needs enough history to establish low variance, then a spike.
	mockSvc := &mockAAP001Service{snapshots: []map[string]uint64{
		{"scope_violation": 100, "restriction_mismatch": 10},
		{"scope_violation": 200, "restriction_mismatch": 20}, // Rate ~10/0.1s in theory but here 1s sleep
		{"scope_violation": 305, "restriction_mismatch": 30}, // Slight jitter
		{"scope_violation": 400, "restriction_mismatch": 40},
		{"scope_violation": 505, "restriction_mismatch": 50},
		{"scope_violation": 150000, "restriction_mismatch": 6000}, // SPIKE
	}}
	s.aap001Service = mockSvc
	if s.semanticHandler != nil {
		s.semanticHandler.Service = mockSvc
	}
	// Set low warmup threshold for test
	if s.semanticHandler != nil {
		s.semanticHandler.WarmupSamples = 3
	}
	// Issue multiple requests to build history and hit the spike.
	// We iterate len-1 times, last one is the final verification.
	for i := 0; i < len(mockSvc.snapshots)-1; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/diagnostics/semantic", nil)
		s.router.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("iteration %d status=%d body=%s", i, w.Code, w.Body.String())
		}
		time.Sleep(1100 * time.Millisecond)
	}
	// Final snapshot for assertions
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/diagnostics/semantic", nil)
	s.router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("final status=%d body=%s", w.Code, w.Body.String())
	}
	var body struct {
		Wired           bool              `json:"wired"`
		Counters        map[string]uint64 `json:"counters"`
		History         []map[string]any  `json:"history"`
		Anomaly         map[string]any    `json:"anomaly"`
		IntegrityStatus string            `json:"integrity_status"`
		PrevHash        string            `json:"prev_hash"`
		CurrentHash     string            `json:"current_hash"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal wired: %v", err)
	}
	if !body.Wired {
		t.Fatalf("wired expected true: %+v", body)
	}
	if len(body.Counters) == 0 {
		t.Fatalf("expected counters populated")
	}
	if len(body.History) < 2 {
		t.Fatalf("expected history length >=2 got %d", len(body.History))
	}
	scores, _ := body.Anomaly["scores"].(map[string]any)
	if len(scores) == 0 {
		t.Fatalf("expected anomaly scores non-empty: %+v", body.Anomaly)
	}
	if body.IntegrityStatus != "ok" {
		t.Fatalf("expected integrity_status ok got %s", body.IntegrityStatus)
	}
	if body.PrevHash == "" || body.CurrentHash == "" || body.PrevHash == body.CurrentHash {
		t.Fatalf("hash chain not evolving: prev=%s current=%s", body.PrevHash, body.CurrentHash)
	}
}

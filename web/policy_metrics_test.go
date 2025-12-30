package web

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mauriciomferz/AgentAuth/pkg/policy"
)

// TestPolicyMetricsEndpoint verifies counters increment after evaluations.
func TestPolicyMetricsEndpoint(t *testing.T) {
	bs := NewBetaServer("")
	t.Cleanup(func() { bs.Shutdown() })

	// Inject a policy to allow the first request:
	// Subject: alice@example.com, Action: read, Resource: report:finance
	if bs.policyHandler != nil && bs.policyHandler.Registry != nil {
		p := policy.Policy{
			ID:       "allow-alice",
			Subjects: []string{"alice@example.com"},
			Rules: []policy.Rule{
				{
					Actions:   []string{"read"},
					Resources: []string{"report:finance"},
					Effect:    policy.Allow,
				},
			},
		}
		b := policy.Bundle{
			ID:       "test-bundle",
			Policies: []policy.Policy{p},
		}
		if _, err := bs.policyHandler.Registry.AddBundle(b); err != nil {
			t.Fatalf("failed to add test policy bundle: %v", err)
		}
	}
	// Perform two evaluations (one allow, one deny if possible)
	// Allow case: alice read report:finance
	allowReqBody := `{"subject":"alice@example.com","action":"read","resource":"report:finance","attrs":{}}`
	rec1 := httptest.NewRecorder()
	req1, _ := http.NewRequest("POST", "/api/v1/policy/evaluate", bytes.NewBufferString(allowReqBody))
	req1.Header.Set("Content-Type", "application/json")
	bs.router.ServeHTTP(rec1, req1)
	if rec1.Code != 200 {
		t.Fatalf("first evaluation failed code=%d body=%s", rec1.Code, rec1.Body.String())
	}
	// Deny case: classification secret triggers deny-secret-classification
	denyReqBody := `{"subject":"alice@example.com","action":"write","resource":"report:finance","attrs":{"classification":"secret"}}`
	rec2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("POST", "/api/v1/policy/evaluate", bytes.NewBufferString(denyReqBody))
	req2.Header.Set("Content-Type", "application/json")
	bs.router.ServeHTTP(rec2, req2)
	if rec2.Code != 200 {
		t.Fatalf("second evaluation failed code=%d body=%s", rec2.Code, rec2.Body.String())
	}
	// Fetch metrics
	recM := httptest.NewRecorder()
	reqM, _ := http.NewRequest("GET", "/api/v1/policy/metrics", nil)
	bs.router.ServeHTTP(recM, reqM)
	if recM.Code != 200 {
		t.Fatalf("metrics endpoint code=%d body=%s", recM.Code, recM.Body.String())
	}
	var out struct {
		Success      bool   `json:"success"`
		Total        uint64 `json:"total"`
		Allow        uint64 `json:"allow"`
		Deny         uint64 `json:"deny"`
		LastReason   string `json:"last_reason"`
		LastMatched  int    `json:"last_matched"`
		LastDeniedBy int    `json:"last_denied_by"`
	}
	if err := json.Unmarshal(recM.Body.Bytes(), &out); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if !out.Success {
		t.Fatalf("success=false")
	}
	if out.Total != 2 {
		t.Fatalf("expected total 2 got %d", out.Total)
	}
	if out.Allow < 1 || out.Deny < 1 {
		t.Fatalf("expected at least one allow and one deny; allow=%d deny=%d", out.Allow, out.Deny)
	}
	if out.LastReason == "" {
		t.Fatalf("last_reason empty")
	}
}

// (Removed custom reader; using bytes.NewBufferString for simplicity.)

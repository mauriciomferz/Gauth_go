package web

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mauriciomferz/AgentAuth/pkg/policy"
)

// TestAuthzMetricsEndpoint ensures metrics JSON shape is returned.
func TestAuthzMetricsEndpoint(t *testing.T) {
	srv := NewBetaServer("")
	t.Cleanup(func() { srv.Shutdown() })
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/beta/authz/metrics", nil)
	srv.router.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("expected 200 got %d body=%s", w.Code, w.Body.String())
	}
	var payload struct {
		Success bool `json:"success"`
		Metrics struct {
			Decisions uint64 `json:"decisions"`
		} `json:"metrics"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if !payload.Success {
		t.Fatalf("success=false payload=%+v", payload)
	}
}

// TestAuthzEvaluateEndpoint performs an evaluation and expects a decision object.
func TestAuthzEvaluateEndpoint(t *testing.T) {
	srv := NewBetaServer("")
	t.Cleanup(func() { srv.Shutdown() })

	// Inject test policy to allow alice@example.com to read report:finance
	if srv.policyHandler != nil && srv.policyHandler.Registry != nil {
		p := policy.Policy{
			ID:       "allow-alice-authz",
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
			ID:       "authz-test-bundle",
			Policies: []policy.Policy{p},
		}
		if _, err := srv.policyHandler.Registry.AddBundle(b); err != nil {
			t.Fatalf("failed to add test policy bundle: %v", err)
		}
	}

	body := map[string]any{"subject": "alice@example.com", "resource": "report:finance", "action": "read", "context": map[string]string{"department": "finance"}}
	b, _ := json.Marshal(body)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/beta/authz/evaluate", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	srv.router.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("expected 200 got %d body=%s", w.Code, w.Body.String())
	}
	var payload struct {
		Success  bool `json:"success"`
		Decision struct {
			Allow  bool   `json:"allow"`
			Reason string `json:"reason"`
		} `json:"decision"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if !payload.Success || !payload.Decision.Allow {
		t.Fatalf("expected allow success; payload=%+v", payload)
	}
	if payload.Decision.Reason == "" {
		t.Fatalf("missing decision reason")
	}
}

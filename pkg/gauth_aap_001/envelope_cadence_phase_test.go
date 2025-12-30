package gauth_aap_001

import (
	"os"
	"testing"
	"time"

	"github.com/mauriciomferz/AgentAuth/internal/metrics"
	"github.com/mauriciomferz/AgentAuth/pkg/audit"
	"github.com/mauriciomferz/AgentAuth/pkg/authz"
)

// TestEnvelopeIssuanceCadenceHistogram ensures cadence histogram observes intervals between consecutive issuances.
func TestEnvelopeIssuanceCadenceHistogram(t *testing.T) {
	mem := metrics.NewMemory()
	// Seed permissive policies (MemoryAuthorizer defaults to deny without policies)
	ma := authz.NewMemoryAuthorizer()
	ma.AddPolicy(authz.Policy{ID: "allow_create", Subject: "*", Resource: "poa", Actions: []string{"create_delegation"}, Effect: authz.Allow})
	svc := NewService(audit.NewMemoryLogger(nil), ma, WithMetrics(mem))
	_ = os.Unsetenv("AGENTAUTH_POA_ENVELOPE_V2")
	// First issuance – no cadence recorded (previous timestamp absent). Use same grantor for both issuances
	req1 := DelegationRequest{Grantor: "cadence@example.com", Grantee: "g@example.com", Scope: []string{"x"}, Duration: time.Minute}
	if _, err := svc.CreateDelegationCtx(tContext(), req1); err != nil {
		t.Fatalf("first issuance failed: %v", err)
	}
	// Sleep >1s to ensure second boundary crosses (cadence logic uses whole seconds)
	time.Sleep(1100 * time.Millisecond)
	req2 := DelegationRequest{Grantor: "cadence@example.com", Grantee: "g@example.com", Scope: []string{"x"}, Duration: time.Minute}
	if _, err := svc.CreateDelegationCtx(tContext(), req2); err != nil {
		t.Fatalf("second issuance failed: %v", err)
	}
	// Memory adapter stores aggregate cadence stats; ensure count >=1 and avg within expected bounds.
	avg := mem.EnvelopeIssuanceCadenceAvgSeconds()
	if avg <= 0 {
		t.Fatalf("expected positive cadence avg, got %f", avg)
	}
}

// TestSunsetPhaseGaugeMemory verifies setting sunset phase persists in memory metrics implementation.
func TestSunsetPhaseGaugeMemory(t *testing.T) {
	mem := metrics.NewMemory()
	svc := NewService(audit.NewMemoryLogger(nil), authz.NewMemoryAuthorizer(), WithMetrics(mem))
	// Initial phase should be zero (unset)
	if phase := mem.EnvelopeV1SunsetPhase(); phase != 0 {
		t.Fatalf("expected initial sunset phase 0 got %d", phase)
	}
	// Set phase to Pilot (1)
	svc.metrics.SetEnvelopeV1SunsetPhase(1)
	if phase := mem.EnvelopeV1SunsetPhase(); phase != 1 {
		t.Fatalf("expected sunset phase 1 got %d", phase)
	}
	// Transition to Stabilization (3)
	svc.metrics.SetEnvelopeV1SunsetPhase(3)
	if phase := mem.EnvelopeV1SunsetPhase(); phase != 3 {
		t.Fatalf("expected sunset phase 3 got %d", phase)
	}
}

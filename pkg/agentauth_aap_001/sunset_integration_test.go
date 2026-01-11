package agentauth_aap_001

import (
	"context"
	"testing"
	"time"

	"github.com/mauriciomferz/AgentAuth/internal/metrics"
	"github.com/mauriciomferz/AgentAuth/internal/sunset"
	"github.com/mauriciomferz/AgentAuth/pkg/audit"
	"github.com/mauriciomferz/AgentAuth/pkg/authz"
)

func TestServiceUseSunsetController(t *testing.T) {
	mem := metrics.NewMemory()
	ma := authz.NewMemoryAuthorizer()
	ma.AddPolicy(authz.Policy{ID: "allow_create", Subject: "*", Resource: "poa", Actions: []string{"create_delegation"}, Effect: authz.Allow})
	svc := NewService(audit.NewMemoryLogger(nil), ma, WithMetrics(mem))

	// Ensure sunset controller is initialized
	if svc.sunsetController == nil {
		t.Fatal("sunsetController not initialized in Service")
	}

	// Ensure unset env var to rely on controller logic
	t.Setenv("AGENTAUTH_POA_ENVELOPE_V2", "")

	// 1. Initial State: Pilot (1) -> Use V2? No, Pilot means V1 default.
	req := DelegationRequest{Grantor: "p@example.com", Grantee: "q@example.com", Scope: []string{"x"}, Duration: time.Minute}
	_, err := svc.CreateDelegationCtx(context.Background(), req)
	if err != nil {
		t.Fatalf("create delegation failed: %v", err)
	}
	// Verify V1
	if count := mem.EnvelopeV2IssuedCount(); count != 0 {
		t.Errorf("Expected 0 V2 tokens in Pilot phase, got %d", count)
	}
	if count := mem.EnvelopeV1IssuedCount(); count != 1 {
		t.Errorf("Expected 1 V1 token in Pilot phase, got %d", count)
	}

	// 2. Advance Phase to Stabilization (3)
	// We manually set the phase metric, which the controller reads via Phase() -> metrics.
	mem.SetEnvelopeV1SunsetPhase(sunset.PhaseStabilization)

	// Verify controller sees the change
	if phase := svc.sunsetController.Phase(); phase != uint64(sunset.PhaseStabilization) {
		t.Errorf("expected phase Stabilization (3), got %d", phase)
	}

	// Issue again
	_, err = svc.CreateDelegationCtx(context.Background(), req)
	if err != nil {
		t.Fatalf("create delegation 2 failed: %v", err)
	}

	// Verify V2
	if count := mem.EnvelopeV2IssuedCount(); count != 1 {
		t.Errorf("Expected 1 V2 token in Stabilization phase, got %d", count)
	}
	// V1 count should remain 1
	if count := mem.EnvelopeV1IssuedCount(); count != 1 {
		t.Errorf("Expected V1 token count to remain 1, got %d", count)
	}
}

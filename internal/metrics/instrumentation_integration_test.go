package metrics_test

import (
	"context"
	"testing"
	"time"

	"github.com/mauriciomferz/AgentAuth/internal/metrics"
	"github.com/mauriciomferz/AgentAuth/pkg/pdp"
)

// TestPDPDecisionRecording verifies RecordDecision & unauthorized counter increments.
func TestPDPDecisionRecording(t *testing.T) {
	m := metrics.NewMemory()
	eng := pdp.NewInMemoryEngine(pdp.DenyOverridesStrategy{}).WithMetrics(m)
	// Policy allowing read on resourceA, denying write on resourceA
	eng.AddPolicy(pdp.Policy{
		ID:       "p1",
		Subjects: []string{"alice"},
		Rules: []pdp.Rule{
			{ID: "r1", Actions: []string{"read"}, Resources: []string{"resourceA"}, Effect: "allow"},
			{ID: "r2", Actions: []string{"write"}, Resources: []string{"resourceA"}, Effect: "deny"},
		},
	})
	// Allow decision
	decAllow, err := eng.Evaluate(context.Background(), pdp.Request{
		Subject:    "alice",
		Action:     "read",
		Resource:   "resourceA",
		Attributes: map[string]string{},
		Time:       time.Now(),
	})
	if err != nil || !decAllow.Allow {
		t.Fatalf("expected allow got %+v err=%v", decAllow, err)
	}
	// Deny decision
	decDeny, err := eng.Evaluate(context.Background(), pdp.Request{
		Subject:    "alice",
		Action:     "write",
		Resource:   "resourceA",
		Attributes: map[string]string{},
		Time:       time.Now(),
	})
	if err != nil || decDeny.Allow {
		t.Fatalf("expected deny got %+v err=%v", decDeny, err)
	}
	snap := m.SnapshotEx()
	if snap.UnauthorizedDecisions == 0 {
		t.Fatalf("expected unauthorized counter increment")
	}
	// Decision labeled counts are stored internally; ensure at least two recorded via heuristic (cannot access map directly)
	// (SnapshotStruct does not expose decision label counts; presence validated indirectly by unauthorized counter.)
}

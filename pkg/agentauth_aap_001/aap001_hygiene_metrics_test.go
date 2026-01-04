package agentauth_aap_001

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/mauriciomferz/AgentAuth/internal/metrics"
	"github.com/mauriciomferz/AgentAuth/pkg/audit"
	"github.com/mauriciomferz/AgentAuth/pkg/authz"
	"github.com/mauriciomferz/AgentAuth/pkg/pdp"
)

// testMetricsMemory wraps metrics.Memory to expose decision & violation breakdown for assertions.
func newTestMetrics() *metrics.Memory { return metrics.NewMemory() }

// TestUTF8ScopeViolationCounter ensures scope_utf8_invalid increments via IncViolation hook indirectly (mapped to scope violations counter).
func TestUTF8ScopeViolationCounter(t *testing.T) {
	m := newTestMetrics()
	svc := NewService(audit.NewMemoryLogger(nil), authz.NewMemoryAuthorizer(), WithMetrics(m))
	// Inject an invalid UTF-8 scope entry (simulate by raw bytes with continuation error)
	bad := string([]byte{0xff, 0xfe, 'a'})
	_, err := svc.CreateDelegation(DelegationRequest{Grantor: "alice", Grantee: "bob", Scope: []string{bad}, Duration: time.Minute})
	if err == nil {
		t.Fatalf("expected error for invalid utf8 scope")
	}
	snap := m.SnapshotEx()
	if snap.ScopeViolations == 0 {
		t.Fatalf("expected scope violations >0 after invalid utf8, got %d", snap.ScopeViolations)
	}
}

// TestControlCharRestrictionViolation ensures restriction_control_char increments (mapped to restriction violations counter).
func TestControlCharRestrictionViolation(t *testing.T) {
	m := newTestMetrics()
	svc := NewService(audit.NewMemoryLogger(nil), authz.NewMemoryAuthorizer(), WithMetrics(m))
	// control character in restriction value (0x07 bell)
	rs := map[string]string{"currency": "USD" + string(rune(0x07))}
	_, err := svc.CreateDelegation(DelegationRequest{Grantor: "alice", Grantee: "bob", Scope: []string{"pay"}, Restrictions: rs, Duration: time.Minute})
	if err == nil {
		t.Fatalf("expected error for control character in restriction value")
	}
	snap := m.SnapshotEx()
	if snap.RestrictionViolations == 0 {
		t.Fatalf("expected restriction violations >0 after control char, got %d", snap.RestrictionViolations)
	}
}

// TestDecisionLabeledMetrics ensures RecordDecision creates labeled breakdown entries (action|resource|outcome) via PDP path.
func TestDecisionLabeledMetrics(t *testing.T) {
	m := newTestMetrics()
	// Set up minimal PDP engine with one allow and one deny rule to exercise both outcomes
	eng := pdp.NewInMemoryEngine(pdp.DenyOverridesStrategy{}).WithMetrics(m)
	eng.AddPolicy(pdp.Policy{ID: "p1", Subjects: []string{"alice"}, Rules: []pdp.Rule{{ID: "ra", Actions: []string{"read"}, Resources: []string{"doc"}, Effect: "allow"}, {ID: "rd", Actions: []string{"delete"}, Resources: []string{"doc"}, Effect: "deny"}}})
	// Allow
	decA, err := eng.Evaluate(context.Background(), pdp.Request{Subject: "alice", Action: "read", Resource: "doc", Attributes: map[string]string{}, Time: time.Now()})
	if err != nil || !decA.Allow {
		t.Fatalf("expected allow decision err=%v allow=%v", err, decA.Allow)
	}
	// Deny
	decD, err := eng.Evaluate(context.Background(), pdp.Request{Subject: "alice", Action: "delete", Resource: "doc", Attributes: map[string]string{}, Time: time.Now()})
	if err != nil || decD.Allow {
		t.Fatalf("expected deny decision err=%v allow=%v", err, decD.Allow)
	}
	snap := m.SnapshotEx()
	// DecisionBreakdown keys are action|resource|outcome
	foundAllow, foundDeny := false, false
	for k := range snap.DecisionBreakdown {
		if strings.Contains(k, "read|doc|allow") {
			foundAllow = true
		}
		if strings.Contains(k, "delete|doc|deny") {
			foundDeny = true
		}
	}
	if !foundAllow || !foundDeny {
		t.Fatalf("expected labeled decisions for allow & deny; allow=%v deny=%v keys=%v", foundAllow, foundDeny, snap.DecisionBreakdown)
	}
}

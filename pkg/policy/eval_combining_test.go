package policy

import (
	"context"
	"testing"
)

// TestPolicyEvaluationCombiningStub constructs a single bundle containing both an allow and deny policy.
// The ChainEngine only evaluates the head bundle (deny-overrides). We assert that:
//   - read is allowed (matches allow-read policy, no deny rule for read)
//   - write is denied (matches deny-write policy)
func TestPolicyEvaluationCombiningStub(t *testing.T) {
	reg := NewRegistry()
	b := Bundle{ID: "b1", Policies: []Policy{
		{ID: "allow-read", Subjects: []string{"alice"}, Rules: []Rule{{Actions: []string{"read"}, Resources: []string{"doc:1"}, Effect: Allow}}},
		{ID: "deny-write", Subjects: []string{"alice"}, Rules: []Rule{{Actions: []string{"write"}, Resources: []string{"doc:1"}, Effect: Deny}}},
	}}
	if _, err := reg.AddBundle(b); err != nil {
		t.Fatalf("add bundle: %v", err)
	}
	eng := NewChainEngine(reg)
	dec, err := eng.Evaluate(context.TODO(), EvalRequest{Subject: "alice", Action: "read", Resource: "doc:1"})
	if err != nil {
		t.Fatalf("eval read: %v", err)
	}
	if !dec.Allow || dec.Deny {
		t.Fatalf("expected allow for read; got allow=%v deny=%v", dec.Allow, dec.Deny)
	}
	if dec.Reason != "allowed" {
		t.Fatalf("expected reason 'allowed', got %q", dec.Reason)
	}
	dec2, err := eng.Evaluate(context.TODO(), EvalRequest{Subject: "alice", Action: "write", Resource: "doc:1"})
	if err != nil {
		t.Fatalf("eval write: %v", err)
	}
	if dec2.Allow || !dec2.Deny {
		t.Fatalf("expected deny for write; got allow=%v deny=%v", dec2.Allow, dec2.Deny)
	}
	if dec2.Reason != "denied by policy" {
		t.Fatalf("expected reason 'denied by policy', got %q", dec2.Reason)
	}
}

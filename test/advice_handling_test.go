package test

import (
	"context"
	"fmt"
	"testing"

	metrics "github.com/mauriciomferz/AgentAuth/internal/metrics"
	"github.com/mauriciomferz/AgentAuth/pkg/authz"
)

// fakeExecutor simulates failures for specific IDs.
type fakeExecutor struct{}

func (f *fakeExecutor) Execute(ob authz.Obligation, ctx map[string]interface{}) error {
	if ob.Type == "fail" {
		return fmt.Errorf("simulated failure")
	}
	return nil
}
func (f *fakeExecutor) PersistAudit(ob authz.Obligation, ctx map[string]interface{}, result error) error {
	return nil
}

// minimal metrics provider (Memory implements needed interface)

func TestAdviceFailureDoesNotChangeDecision(t *testing.T) {
	ma := authz.NewMemoryAuthorizer()
	mem := metrics.NewMemory()
	ma.SetMetricsProvider(mem)
	ma.SetObligationExecutor(&fakeExecutor{})

	policy := authz.Policy{ID: "p1", Subject: "bob", Resource: "doc:7", Actions: []string{"read"}, Effect: authz.Allow,
		Advice: []authz.Advice{{ID: "adv1", Type: "fail", Params: map[string]string{"x": "1"}}},
	}
	ma.AddPolicy(policy)
	ma.Snapshot()
	req := authz.Request{Subject: "bob", Resource: "doc:7", Action: "read"}
	dec, err := ma.Authorize(context.Background(), req)
	if err != nil {
		t.Fatalf("authorize err: %v", err)
	}
	if !dec.Allow {
		t.Fatalf("decision flipped unexpectedly on advice failure")
	}
	if dec.Metadata["advice_failure"] != "adv1" {
		t.Fatalf("expected advice_failure metadata adv1 got %v", dec.Metadata["advice_failure"])
	}
}

func TestMandatoryObligationFailureFlipsDecision(t *testing.T) {
	ma := authz.NewMemoryAuthorizer()
	mem := metrics.NewMemory()
	ma.SetMetricsProvider(mem)
	ma.SetObligationExecutor(&fakeExecutor{})
	policy := authz.Policy{ID: "p2", Subject: "alice", Resource: "doc:99", Actions: []string{"write"}, Effect: authz.Allow,
		Obligations: []authz.Obligation{{ID: "obl1", Type: "fail", Mandatory: true}},
	}
	ma.AddPolicy(policy)
	ma.Snapshot()
	req := authz.Request{Subject: "alice", Resource: "doc:99", Action: "write"}
	dec, err := ma.Authorize(context.Background(), req)
	if err != nil {
		t.Fatalf("authorize err: %v", err)
	}
	if dec.Allow {
		t.Fatalf("expected decision flipped to deny due to mandatory obligation failure")
	}
	if dec.Metadata["obligation_failure"] != "obl1" {
		t.Fatalf("expected obligation_failure metadata obl1 got %v", dec.Metadata["obligation_failure"])
	}
	if mem.MandatoryObligationFailures() == 0 {
		t.Fatalf("expected mandatory obligation failure metric increment")
	}
}

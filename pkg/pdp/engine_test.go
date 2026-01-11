package pdp

import (
	"context"
	"testing"
	"time"
)

func TestDenyOverridesStrategy(t *testing.T) {
	eng := NewInMemoryEngine(DenyOverridesStrategy{})
	eng.AddPolicy(Policy{
		ID:       "p1",
		Subjects: []string{"alice"},
		Rules: []Rule{
			{
				ID:        "r1",
				Actions:   []string{"read"},
				Resources: []string{"doc"},
				Effect:    "allow",
			},
		},
	})
	eng.AddPolicy(Policy{
		ID:       "p2",
		Subjects: []string{"alice"},
		Rules: []Rule{
			{
				ID:        "r2",
				Actions:   []string{"read"},
				Resources: []string{"doc"},
				Effect:    "deny",
			},
		},
	})
	dec, err := eng.Evaluate(context.Background(), Request{Subject: "alice", Action: "read", Resource: "doc", Time: time.Now()})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dec.Allow {
		t.Fatalf("expected deny due to deny_overrides, got allow")
	}
	if dec.Reason == "" {
		t.Errorf("missing reason")
	}
	if len(dec.DenyPolicies) == 0 {
		t.Errorf("expected denyPolicies non-empty")
	}
}

func TestPermitOverridesStrategy(t *testing.T) {
	eng := NewInMemoryEngine(PermitOverridesStrategy{})
	eng.AddPolicy(Policy{
		ID:       "p1",
		Subjects: []string{"bob"},
		Rules: []Rule{
			{
				ID:        "r1",
				Actions:   []string{"write"},
				Resources: []string{"doc"},
				Effect:    "deny",
			},
		},
	})
	eng.AddPolicy(Policy{
		ID:       "p2",
		Subjects: []string{"bob"},
		Rules: []Rule{
			{
				ID:        "r2",
				Actions:   []string{"write"},
				Resources: []string{"doc"},
				Effect:    "allow",
			},
		},
	})
	dec, err := eng.Evaluate(context.Background(), Request{Subject: "bob", Action: "write", Resource: "doc", Time: time.Now()})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !dec.Allow {
		t.Fatalf("expected allow due to permit_overrides, got deny")
	}
	if len(dec.Policies) == 0 {
		t.Errorf("expected contributing allow policies")
	}
}

func TestFirstApplicableStrategy(t *testing.T) {
	eng := NewInMemoryEngine(FirstApplicableStrategy{})
	// Order matters: first deny should short-circuit
	eng.AddPolicy(Policy{
		ID:       "pdeny",
		Subjects: []string{"carl"},
		Rules: []Rule{
			{
				ID:        "rdeny",
				Actions:   []string{"delete"},
				Resources: []string{"doc"},
				Effect:    "deny",
			},
		},
	})
	eng.AddPolicy(Policy{
		ID:       "pallow",
		Subjects: []string{"carl"},
		Rules: []Rule{
			{
				ID:        "rallow",
				Actions:   []string{"delete"},
				Resources: []string{"doc"},
				Effect:    "allow",
			},
		},
	})
	dec, err := eng.Evaluate(context.Background(), Request{Subject: "carl", Action: "delete", Resource: "doc", Time: time.Now()})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dec.Allow {
		t.Fatalf("expected deny due to first applicable deny")
	}
	if dec.Reason == "" || dec.Reason == defaultDenyReason {
		t.Fatalf("unexpected reason: %s", dec.Reason)
	}
}

func TestDefaultDenyNoMatch(t *testing.T) {
	eng := NewInMemoryEngine(DenyOverridesStrategy{})
	dec, err := eng.Evaluate(context.Background(), Request{Subject: "nobody", Action: "read", Resource: "doc", Time: time.Now()})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dec.Allow {
		t.Fatalf("expected deny (default) got allow")
	}
	if dec.Reason != "No matching policy - default deny" {
		t.Errorf("unexpected reason: %s", dec.Reason)
	}
}

func TestTracePopulation(t *testing.T) {
	eng := NewInMemoryEngine(DenyOverridesStrategy{})
	eng.AddPolicy(Policy{
		ID:       "ptrace",
		Subjects: []string{"dana"},
		Rules: []Rule{
			{
				ID:        "r1",
				Actions:   []string{"read"},
				Resources: []string{"doc"},
				Effect:    "allow",
			},
		},
	})
	dec, err := eng.Evaluate(context.Background(), Request{Subject: "dana", Action: "read", Resource: "doc", Time: time.Now()})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(dec.Trace) == 0 {
		t.Fatalf("expected trace entries")
	}
	if dec.Trace[0].PolicyID != "ptrace" {
		t.Errorf("unexpected trace policy id: %s", dec.Trace[0].PolicyID)
	}
}

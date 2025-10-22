package test

import (
	"context"
	"testing"
	"time"

	authz "github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/authz"
	"github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/policy"
)

// Benchmark comparing simple map-based RBAC (MemoryAuthorizer) vs ChainEngine evaluation.
func BenchmarkPolicyEvaluation(b *testing.B) {
	// Setup legacy memory authorizer
	mem := authz.NewMemoryAuthorizer()
	mem.AddPolicy(authz.Policy{ID: "p1", Subject: "alice", Resource: "doc", Actions: []string{"read"}, Effect: authz.Allow})

	// Setup chain engine with equivalent policy
	reg := policy.NewRegistry()
	if _, err := reg.AddBundle(policy.Bundle{ID: "b1", Policies: []policy.Policy{{ID: "rbac", Subjects: []string{"alice"}, Rules: []policy.Rule{{Actions: []string{"read"}, Resources: []string{"doc"}, Effect: policy.Allow}}}}}); err != nil {
		b.Fatalf("add bundle b1 failed: %v", err)
	}
	eng := policy.NewChainEngine(reg)

	ctx := context.Background()

	b.Run("memory_authorizer", func(b *testing.B) {
		req := authz.Request{Subject: "alice", Action: "read", Resource: "doc"}
		for i := 0; i < b.N; i++ {
			if _, err := mem.Authorize(ctx, req); err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("chain_engine_simple", func(b *testing.B) {
		req := policy.EvalRequest{Subject: "alice", Action: "read", Resource: "doc", Now: time.Now()}
		for i := 0; i < b.N; i++ {
			if _, err := eng.Evaluate(ctx, req); err != nil {
				b.Fatal(err)
			}
		}
	})

	// Add a bundle with attribute expression and time window plus deny override
	if _, err := reg.AddBundle(policy.Bundle{ID: "b2", Policies: []policy.Policy{
		{ID: "abac", Subjects: []string{"*"}, Rules: []policy.Rule{{Actions: []string{"view"}, Resources: []string{"report"}, Expr: "department == 'finance' && time_between(\"09:00\",\"17:00\")", Effect: policy.Allow}}},
		// Deny applies to a different resource to avoid overriding allow benchmark
		{ID: "deny-secret", Subjects: []string{"*"}, Rules: []policy.Rule{{Actions: []string{"view"}, Resources: []string{"secret"}, Expr: "department == 'finance'", Effect: policy.Deny}}},
	}}); err != nil {
		b.Fatalf("add bundle b2 failed: %v", err)
	}

	b.Run("chain_engine_expr_allow", func(b *testing.B) {
		req := policy.EvalRequest{Subject: "bob", Action: "view", Resource: "report", Attrs: map[string]string{"department": "finance"}, Now: mustParseClockBench("10:00")}
		for i := 0; i < b.N; i++ {
			if dec, err := eng.Evaluate(ctx, req); err != nil || !dec.Allow {
				b.Fatalf("expected allow: %+v err=%v", dec, err)
			}
		}
	})
	b.Run("chain_engine_expr_deny_override", func(b *testing.B) {
		req := policy.EvalRequest{Subject: "bob", Action: "view", Resource: "secret", Attrs: map[string]string{"department": "finance"}, Now: mustParseClockBench("10:00")}
		for i := 0; i < b.N; i++ {
			if dec, err := eng.Evaluate(ctx, req); err != nil {
				b.Fatal(err)
			} else if !dec.Deny {
				b.Fatalf("expected deny override: %+v", dec)
			}
		}
	})

	// Add bundle with OR chains and numeric comparisons
	if _, err := reg.AddBundle(policy.Bundle{ID: "b3", Policies: []policy.Policy{
		{ID: "or-policy", Subjects: []string{"*"}, Rules: []policy.Rule{{Actions: []string{"transfer"}, Resources: []string{"acct"}, Expr: "role == 'ops' || role == 'finance' || role == 'admin'", Effect: policy.Allow}}},
		{ID: "numeric-policy", Subjects: []string{"*"}, Rules: []policy.Rule{{Actions: []string{"purchase"}, Resources: []string{"item"}, Expr: "amount <= 1000 && amount >= 10", Effect: policy.Allow}}},
	}}); err != nil {
		b.Fatalf("add bundle b3 failed: %v", err)
	}

	b.Run("chain_engine_or_match", func(b *testing.B) {
		req := policy.EvalRequest{Subject: "carol", Action: "transfer", Resource: "acct", Attrs: map[string]string{"role": "finance"}}
		for i := 0; i < b.N; i++ {
			if dec, err := eng.Evaluate(ctx, req); err != nil || !dec.Allow {
				b.Fatalf("expected allow via OR match: %+v err=%v", dec, err)
			}
		}
	})
	b.Run("chain_engine_or_no_match", func(b *testing.B) {
		req := policy.EvalRequest{Subject: "dave", Action: "transfer", Resource: "acct", Attrs: map[string]string{"role": "legal"}}
		for i := 0; i < b.N; i++ {
			if dec, err := eng.Evaluate(ctx, req); err != nil {
				b.Fatal(err)
			} else if dec.Allow {
				b.Fatalf("expected not allow for non-matching OR: %+v", dec)
			}
		}
	})
	b.Run("chain_engine_numeric_match", func(b *testing.B) {
		req := policy.EvalRequest{Subject: "erin", Action: "purchase", Resource: "item", Attrs: map[string]string{"amount": "250"}}
		for i := 0; i < b.N; i++ {
			if dec, err := eng.Evaluate(ctx, req); err != nil || !dec.Allow {
				b.Fatalf("expected allow numeric within range: %+v err=%v", dec, err)
			}
		}
	})
	b.Run("chain_engine_numeric_outside", func(b *testing.B) {
		req := policy.EvalRequest{Subject: "frank", Action: "purchase", Resource: "item", Attrs: map[string]string{"amount": "5000"}}
		for i := 0; i < b.N; i++ {
			if dec, err := eng.Evaluate(ctx, req); err != nil {
				b.Fatal(err)
			} else if dec.Allow {
				b.Fatalf("expected deny outside numeric range: %+v", dec)
			}
		}
	})
}

func mustParseClockBench(clock string) time.Time {
	tm, _ := time.Parse("15:04", clock)
	return time.Date(2025, 1, 1, tm.Hour(), tm.Minute(), 0, 0, time.UTC)
}

package test

import (
	"context"
	"fmt"
	"testing"
	"time"

	authz "github.com/mauriciomferz/Gauth_go/pkg/authz"
	"github.com/mauriciomferz/Gauth_go/pkg/policy"
)

// helper to build a bundle
func buildBundle(id string, policies []policy.Policy) policy.Bundle {
	return policy.Bundle{ID: id, Policies: policies}
}

func TestPolicyEnginePatterns(t *testing.T) {
	reg := policy.NewRegistry()
	// Policies: RBAC allow, ABAC dept match, time window allow, explicit deny
	b, err := reg.AddBundle(buildBundle("b1", []policy.Policy{
		{ID: "rbac-admin", Subjects: []string{"alice"}, Rules: []policy.Rule{{Actions: []string{"create"}, Resources: []string{"doc"}, Effect: policy.Allow}}},
		{ID: "abac-dept", Subjects: []string{"*"}, Rules: []policy.Rule{{Actions: []string{"read"}, Resources: []string{"doc"}, Expr: "department == 'finance'", Effect: policy.Allow}}},
		{ID: "time-window", Subjects: []string{"*"}, Rules: []policy.Rule{{Actions: []string{"export"}, Resources: []string{"report"}, Expr: "time_between(\"09:00\",\"17:00\")", Effect: policy.Allow}}},
		{ID: "deny-secret", Subjects: []string{"*"}, Rules: []policy.Rule{{Actions: []string{"read"}, Resources: []string{"secret"}, Effect: policy.Deny}}},
	}))
	if err != nil {
		t.Fatalf("add bundle: %v", err)
	}
	if b.Hash == "" {
		t.Fatalf("expected bundle hash computed")
	}
	if verifyErr := reg.VerifyChain(); verifyErr != nil {
		t.Fatalf("verify chain failed: %v", verifyErr)
	}

	eng := policy.NewChainEngine(reg)
	adapter := policy.NewAuthorizerAdapter(eng)
	ctx := context.Background()

	// 1. RBAC allow
	dec, err := adapter.Authorize(ctx, authz.Request{Subject: "alice", Action: "create", Resource: "doc"})
	if err != nil || !dec.Allow {
		t.Fatalf("expected RBAC allow: %+v err=%v", dec, err)
	}

	// 2. ABAC department match
	dec, err = adapter.Authorize(ctx, authz.Request{Subject: "bob", Action: "read", Resource: "doc", Context: map[string]string{"department": "finance"}})
	if err != nil || !dec.Allow {
		t.Fatalf("expected ABAC allow: %+v err=%v", dec, err)
	}

	// 3. ABAC non-match
	dec, _ = adapter.Authorize(ctx, authz.Request{Subject: "bob", Action: "read", Resource: "doc", Context: map[string]string{"department": "hr"}})
	if dec.Allow {
		t.Fatalf("expected deny due to dept mismatch")
	}

	// 4. Time window allow (simulate 10:00 UTC)
	// Direct engine call to control time
	evalDec, err := eng.Evaluate(ctx, policy.EvalRequest{Subject: "carol", Action: "export", Resource: "report", Now: mustParseClock(t, "10:00"), Attrs: map[string]string{}})
	if err != nil || !evalDec.Allow {
		t.Fatalf("expected time window allow: %+v err=%v", evalDec, err)
	}
	// 5. Time window deny (simulate 20:00 UTC)
	evalDec, _ = eng.Evaluate(ctx, policy.EvalRequest{Subject: "carol", Action: "export", Resource: "report", Now: mustParseClock(t, "20:00")})
	if evalDec.Allow {
		t.Fatalf("expected time window deny")
	}

	// 6. Deny override
	dec, _ = adapter.Authorize(ctx, authz.Request{Subject: "any", Action: "read", Resource: "secret"})
	if dec.Allow {
		t.Fatalf("expected deny override for secret")
	}
}

func TestExpressionOrAndNumeric(t *testing.T) {
	reg := policy.NewRegistry()
	eng := policy.NewChainEngine(reg)
	bundle := policy.Bundle{ID: "expr-advanced", Policies: []policy.Policy{{
		ID: "p-or", Subjects: []string{"user"}, Rules: []policy.Rule{{Actions: []string{"read"}, Resources: []string{"obj"}, Effect: policy.Allow, Expr: "role == 'admin' || level >= 5"}},
	}, {
		ID: "p-deny-num", Subjects: []string{"user"}, Rules: []policy.Rule{{Actions: []string{"read"}, Resources: []string{"obj"}, Effect: policy.Deny, Expr: "sensitivity > 7"}},
	}}}
	if _, err := reg.AddBundle(bundle); err != nil {
		t.Fatalf("add bundle: %v", err)
	}

	// Should allow via OR first clause (role == admin) even if level low
	dec, _ := eng.Evaluate(context.Background(), policy.EvalRequest{Subject: "user", Action: "read", Resource: "obj", Attrs: map[string]string{"role": "admin", "level": "1", "sensitivity": "3"}})
	if !dec.Allow {
		t.Fatalf("expected allow via OR admin, got %+v", dec)
	}

	// Should allow via numeric level >=5 when role != admin
	dec2, _ := eng.Evaluate(context.Background(), policy.EvalRequest{Subject: "user", Action: "read", Resource: "obj", Attrs: map[string]string{"role": "viewer", "level": "5", "sensitivity": "3"}})
	if !dec2.Allow {
		t.Fatalf("expected allow via level>=5, got %+v", dec2)
	}

	// Should deny due to sensitivity > 7 (deny overrides allow) even though OR would allow
	dec3, _ := eng.Evaluate(context.Background(), policy.EvalRequest{Subject: "user", Action: "read", Resource: "obj", Attrs: map[string]string{"role": "admin", "level": "9", "sensitivity": "9"}})
	if !dec3.Deny {
		t.Fatalf("expected deny due to sensitivity, got %+v", dec3)
	}
}

func mustParseClock(t *testing.T, clock string) time.Time {
	tm, err := time.Parse("15:04", clock)
	if err != nil {
		t.Fatalf("parse clock: %v", err)
	}
	return time.Date(2025, 1, 1, tm.Hour(), tm.Minute(), 0, 0, time.UTC)
}

func TestPolicyBundleChainIntegrity(t *testing.T) {
	reg := policy.NewRegistry()
	for i := 0; i < 3; i++ {
		_, err := reg.AddBundle(buildBundle(fmt.Sprintf("b%d", i), []policy.Policy{}))
		if err != nil {
			t.Fatalf("add bundle: %v", err)
		}
	}
	if verifyErr := reg.VerifyChain(); verifyErr != nil {
		t.Fatalf("expected chain valid: %v", verifyErr)
	}
	// Tamper
	head := reg.Head()
	head.PrevHash = "deadbeef" // breaks link
	if err := reg.VerifyChain(); err == nil {
		t.Fatalf("expected verification failure after tamper")
	}
}

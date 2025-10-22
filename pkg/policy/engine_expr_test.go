package policy

import (
	"context"
	"testing"
	"time"
)

func TestEvalExprNotAndParens(t *testing.T) {
	now := time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC)
	attrs := map[string]string{"role": "finance", "amount": "150"}
	cases := []struct {
		expr string
		want bool
	}{
		{"role == 'finance'", true},
		{"!(role == 'finance')", false},
		{"!(role == 'ops')", true},
		{"(role == 'ops') || (role == 'finance')", true},
		{"!(role == 'finance' && amount > 100)", false},
		{"!(role == 'finance' && amount > 100) || role == 'finance'", true},
		{"role == 'finance' && (amount > 10 && amount < 200)", true},
		{"role == 'finance' && (amount > 10 && (amount < 100))", false},
		{"!(role == 'finance') || (amount < 10)", false},
	}
	for i, c := range cases {
		got, err := evalExpr(c.expr, attrs, now)
		if err != nil {
			t.Fatalf("case %d error: %v", i, err)
		}
		if got != c.want {
			t.Errorf("case %d expr=%q got %v want %v", i, c.expr, got, c.want)
		}
	}
}

func TestPolicyEvaluateWithNotAndParens(t *testing.T) {
	reg := NewRegistry()
	b, err := reg.AddBundle(Bundle{ID: "b1", Policies: []Policy{{
		ID:       "p1",
		Subjects: []string{"alice@example.com"},
		Rules: []Rule{{
			Actions: []string{"read"}, Resources: []string{"report:finance"}, Effect: Allow,
			Expr: "role == 'finance' && !(amount > 500) && (amount > 100)",
		}},
	}}})
	if err != nil || b.Hash == "" {
		t.Fatalf("add bundle: %v", err)
	}
	eng := NewChainEngine(reg)
	dec, err := eng.Evaluate(context.Background(), EvalRequest{Subject: "alice@example.com", Action: "read", Resource: "report:finance", Attrs: map[string]string{"role": "finance", "amount": "150"}, Now: time.Now()})
	if err != nil {
		t.Fatalf("eval: %v", err)
	}
	if !dec.Allow {
		t.Fatalf("expected allow, got decision %#v", dec)
	}
	// Fails when amount 600
	dec2, err := eng.Evaluate(context.Background(), EvalRequest{Subject: "alice@example.com", Action: "read", Resource: "report:finance", Attrs: map[string]string{"role": "finance", "amount": "600"}, Now: time.Now()})
	if err != nil {
		t.Fatalf("eval2: %v", err)
	}
	if dec2.Allow {
		t.Fatalf("expected deny due to amount > 500, got %#v", dec2)
	}
}

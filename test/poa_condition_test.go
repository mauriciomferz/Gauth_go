package test

import (
	"context"
	"testing"

	"github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/authz"
)

// TestPolicyExpressionBasic ensures simple subject + context expression works.
func TestPolicyExpressionBasic(t *testing.T) {
    ma := authz.NewMemoryAuthorizer()
    p := authz.Policy{ID: "expr1", Subject: "alice", Resource: "obj:42", Actions: []string{"read"}, Effect: authz.Allow, Expression: "subject == \"alice\" && ctx.env == \"prod\""}
    ma.AddPolicy(p)
    ma.Snapshot()
    req := authz.Request{Subject: "alice", Resource: "obj:42", Action: "read", Context: map[string]string{"env": "prod"}}
    dec, err := ma.Authorize(context.Background(), req)
    if err != nil { t.Fatalf("unexpected error: %v", err) }
    if !dec.Allow { t.Fatalf("expected allow, got deny: %v", dec) }
    // negative env mismatch
    req2 := authz.Request{Subject: "alice", Resource: "obj:42", Action: "read", Context: map[string]string{"env": "staging"}}
    dec2, _ := ma.Authorize(context.Background(), req2)
    if dec2.Allow { t.Fatalf("expected deny due to expression mismatch") }
}

// TestPolicyExpressionInList verifies membership operator.
func TestPolicyExpressionInList(t *testing.T) {
    ma := authz.NewMemoryAuthorizer()
    p := authz.Policy{ID: "expr2", Subject: "*", Resource: "data:*", Actions: []string{"write"}, Effect: authz.Allow, Expression: "ctx.tier in [\"gold\", \"platinum\"]"}
    ma.AddPolicy(p)
    ma.Snapshot()
    req := authz.Request{Subject: "bob", Resource: "data:55", Action: "write", Context: map[string]string{"tier": "gold"}}
    if dec, _ := ma.Authorize(context.Background(), req); !dec.Allow { t.Fatalf("expected allow for tier gold") }
    req2 := authz.Request{Subject: "bob", Resource: "data:55", Action: "write", Context: map[string]string{"tier": "bronze"}}
    if dec, _ := ma.Authorize(context.Background(), req2); dec.Allow { t.Fatalf("expected deny for tier bronze") }
}

// TestPolicyExpressionNumericComparison verifies numeric >= operator.
func TestPolicyExpressionNumericComparison(t *testing.T) {
    ma := authz.NewMemoryAuthorizer()
    p := authz.Policy{ID: "expr3", Subject: "carol", Resource: "account:1", Actions: []string{"update"}, Effect: authz.Allow, Expression: "ctx.age >= 21"}
    ma.AddPolicy(p)
    ma.Snapshot()
    req := authz.Request{Subject: "carol", Resource: "account:1", Action: "update", Context: map[string]string{"age": "25"}}
    if dec, _ := ma.Authorize(context.Background(), req); !dec.Allow { t.Fatalf("expected allow age 25") }
    req2 := authz.Request{Subject: "carol", Resource: "account:1", Action: "update", Context: map[string]string{"age": "19"}}
    if dec, _ := ma.Authorize(context.Background(), req2); dec.Allow { t.Fatalf("expected deny age 19") }
}

// TestPolicyExpressionParseError fail closed behavior.
func TestPolicyExpressionParseError(t *testing.T) {
    ma := authz.NewMemoryAuthorizer()
    // malformed expression (missing closing bracket)
    p := authz.Policy{ID: "expr4", Subject: "dave", Resource: "svc:1", Actions: []string{"deploy"}, Effect: authz.Allow, Expression: "ctx.env in [\"prod\", \"staging\""}
    ma.AddPolicy(p)
    ma.Snapshot()
    req := authz.Request{Subject: "dave", Resource: "svc:1", Action: "deploy", Context: map[string]string{"env": "prod"}}
    if dec, _ := ma.Authorize(context.Background(), req); dec.Allow { t.Fatalf("expected deny due to parse error fail-closed") }
}

// TestPolicyExpressionLimits ensures token limit triggers fail closed.
func TestPolicyExpressionLimits(t *testing.T) {
    // Use EvaluateExpression directly with custom limits.
    req := authz.Request{Subject: "eve", Resource: "r", Action: "act", Context: map[string]string{"k": "v"}}
    // Craft expression exceeding small token limit.
    expr := "subject == \"eve\" && action == \"act\" && resource == \"r\" && ctx.k == \"v\""
    lim := &authz.ExprLimits{MaxTokens: 5, MaxDepth: 10, MaxOps: 100, MaxIdentifierLength: 64, MaxLiteralLength: 64}
    ok, err := authz.EvaluateExpression(expr, req, lim)
    if err == nil || ok { t.Fatalf("expected error & false due to token limit, got ok=%v err=%v", ok, err) }
}

// Debug expression evaluation to ensure interpreter returns expected boolean.
func TestPolicyExpressionDebug(t *testing.T) {
    req := authz.Request{Subject: "alice", Resource: "obj:42", Action: "read", Context: map[string]string{"env": "prod"}}
    ok, err := authz.EvaluateExpression("subject == \"alice\" && ctx.env == \"prod\"", req, nil)
    if err != nil { t.Fatalf("unexpected error: %v", err) }
    if !ok { t.Fatalf("expected ok=true for expression, got false") }
}

func TestPolicyExpressionTokenDump(t *testing.T) {
    toks := authz.DebugLex("subject == \"alice\" && ctx.env == \"prod\"")
    if len(toks) == 0 { t.Fatalf("no tokens produced") }
    // Basic sanity: expect at least 9 tokens (subject,==,alice,&&,ctx.env,==,prod,EOF)
    if len(toks) < 8 { t.Fatalf("unexpected small token slice: %d", len(toks)) }
    for i, tk := range toks { t.Logf("%d: typ=%d lit=%s", i, tk.Type, tk.Literal) }
}

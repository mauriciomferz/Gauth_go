package test

import (
	"testing"

	"github.com/mauriciomferz/AgentAuth/pkg/authz"
)

// FuzzPolicyExpressionCompile provides fuzz seeds for expression parser; ensures no panics.
func FuzzPolicyExpressionCompile(f *testing.F) {
	// Seeds covering operators and structures.
	seeds := []string{
		"subject == \"alice\"",
		"subject == \"alice\" && ctx.env == \"prod\"",
		"ctx.tier in [\"gold\", \"silver\"]",
		"ctx.age >= 21 && ctx.country == \"US\"",
		"!(ctx.flag == \"x\") || subject != \"bob\"",
		"resource == \"r:1\" && action == \"read\"",
	}
	for _, s := range seeds {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, expr string) {
		req := authz.Request{
			Subject: "alice", Resource: "obj", Action: "read",
			Context: map[string]string{"env": "prod", "tier": "gold", "age": "30", "country": "US", "flag": "y"},
		}
		_, _ = authz.EvaluateExpression(expr, req, nil) // ignore result; ensure no panic
	})
}

package expr_test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/mauriciomferz/AgentAuth/pkg/authz"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestExprLimits verifies that expression complexity limits are enforced.
func TestExprLimits(t *testing.T) {
	// Deep nesting with operations to force AST depth (parens alone might be flattened)
	var sbDepth strings.Builder
	for i := 0; i < 50; i++ {
		sbDepth.WriteString("true && (")
	}
	sbDepth.WriteString("true")
	for i := 0; i < 50; i++ {
		sbDepth.WriteString(")")
	}
	deepExpr := sbDepth.String()
	limits := &authz.ExprLimits{MaxDepth: 20, MaxTokens: 1000, MaxOps: 1000, MaxIdentifierLength: 64, MaxLiteralLength: 256}

	_, err := authz.EvaluateExpression(deepExpr, authz.Request{}, limits)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "max depth exceeded")

	// Many operations
	manyOps := "true" + strings.Repeat(" && true", 100)
	limitsOps := &authz.ExprLimits{MaxDepth: 200, MaxOps: 50, MaxTokens: 1000, MaxIdentifierLength: 1000, MaxLiteralLength: 1000}
	_, err = authz.EvaluateExpression(manyOps, authz.Request{}, limitsOps)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "max ops exceeded")
}

// TestRegexMatchPerformance demonstrates the cost of repeated compilation.
func TestRegexMatchPerformance(t *testing.T) {
	// Pattern that takes a bit of time to compile?
	// Go's regexp is fast, but we can measure throughput.
	pattern := "^(a+)+$"
	input := "aaaaaaaaaaaaaaaaaaaaaaaa"

	// Create an expression that calls regex_match many times
	// Since we don't have loops, we simulate by just calling it once and measuring latency in a loop
	// or constructing a giant expression

	// construct "regex_match(...,...) && regex_match(...,...) ..."
	count := 1000
	var sb strings.Builder
	sb.WriteString("true")
	for i := 0; i < count; i++ {
		sb.WriteString(fmt.Sprintf(" && regex_match(\"%s\", \"%s\")", input, pattern))
	}
	expr := sb.String()

	start := time.Now()
	// Use high limits to allow execution
	limits := &authz.ExprLimits{MaxOps: 10000, MaxTokens: 100000, MaxDepth: 2000, MaxLiteralLength: 256, MaxIdentifierLength: 64}

	_, err := authz.EvaluateExpression(expr, authz.Request{}, limits)
	require.NoError(t, err)
	duration := time.Since(start)

	t.Logf("Execution of %d regex_matches took %v", count, duration)

	// If caching were implemented, this should be much faster.
	// Without caching, 1000 compilations + match might take e.g. 10-50ms or more.
}

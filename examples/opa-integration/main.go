package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/open-policy-agent/opa/rego"
)

// Note: Policy loading removed - use OPA server with ConfigMaps in production

// OPAScopeValidator validates scopes using OPA Rego policies
type OPAScopeValidator struct {
	query rego.PreparedEvalQuery
}

// NewOPAScopeValidator creates a validator loading policy from file
func NewOPAScopeValidator() (*OPAScopeValidator, error) {
	// Read policy from filesystem
	policyBytes, err := os.ReadFile("policies/agentauth_basic.rego")
	if err != nil {
		return nil, fmt.Errorf("failed to read policy: %w", err)
	}

	// Prepare query
	query, err := rego.New(
		rego.Query("data.agentauth.authz.allow"),
		rego.Module("agentauth_basic.rego", string(policyBytes)),
	).PrepareForEval(context.Background())

	if err != nil {
		return nil, fmt.Errorf("failed to prepare query: %w", err)
	}

	return &OPAScopeValidator{query: query}, nil
}

// ValidateScope checks if child scopes are covered by parent scopes
func (v *OPAScopeValidator) ValidateScope(ctx context.Context, parent, child []string) error {
	// Build input for OPA
	input := map[string]interface{}{
		"action":        "validate_scope",
		"parent_scopes": parent,
		"child_scopes":  child,
	}

	// Evaluate policy
	results, err := v.query.Eval(ctx, rego.EvalInput(input))
	if err != nil {
		return fmt.Errorf("OPA evaluation failed: %w", err)
	}

	if len(results) == 0 {
		return fmt.Errorf("OPA returned no results")
	}

	allowed, ok := results[0].Expressions[0].Value.(bool)
	if !ok {
		return fmt.Errorf("OPA returned unexpected result type")
	}

	if !allowed {
		return fmt.Errorf("scope validation failed: child scopes not covered by parent")
	}

	return nil
}

// ValidateWithDetails returns detailed decision information (Reconstructed)
func (v *OPAScopeValidator) ValidateWithDetails(ctx context.Context, parent, child []string) (bool, map[string]interface{}, error) {
	policyBytes, err := os.ReadFile("policies/scope_validation.rego")
	if err != nil {
		// Fallback if specific policy missing, or just error
		return false, nil, fmt.Errorf("failed to read scope_validation.rego: %w", err)
	}

	detailsQuery, err := rego.New(
		rego.Query("data.agentauth.authz.decision_details"),
		rego.Module("scope_validation.rego", string(policyBytes)),
	).PrepareForEval(ctx)

	if err != nil {
		// If query fails (e.g. unknown rule), return error
		return false, nil, fmt.Errorf("failed to prepare details query: %w", err)
	}

	input := map[string]interface{}{
		"action":        "validate_scope",
		"parent_scopes": parent,
		"child_scopes":  child,
	}

	results, err := detailsQuery.Eval(ctx, rego.EvalInput(input))
	if err != nil {
		return false, nil, err
	}

	if len(results) == 0 {
		return false, nil, fmt.Errorf("OPA returned no results")
	}

	details, ok := results[0].Expressions[0].Value.(map[string]interface{})
	if !ok {
		// Sometimes it might be nil or different type
		return false, nil, fmt.Errorf("unexpected result format")
	}

	// Assume allowed is part of details or separate?
	// The original blob had: allowed := details["allow"].(bool)
	allowed := false
	if val, ok := details["allow"]; ok {
		if b, ok := val.(bool); ok {
			allowed = b
		}
	}

	return allowed, details, nil
}

func main() {
	if len(os.Args) > 1 && os.Args[1] == "benchmark" {
		benchmark()
		return
	}

	fmt.Println("========================================")
	fmt.Println("AgentAuth OPA Integration Examples")
	fmt.Println("========================================")
	fmt.Println()

	example1()
	example2()
	example3()
	example4()

	fmt.Println("========================================")
	fmt.Println("✅ All examples completed")
	fmt.Println("========================================")
}

func example1() {
	fmt.Println("=== Example 1: Basic Scope Validation ===")
	validator, err := NewOPAScopeValidator()
	if err != nil {
		log.Fatal(err)
	}

	// Test case 1: Valid delegation
	parent1 := []string{"users:*"}
	child1 := []string{"users:read", "users:write"}
	fmt.Printf("Parent: %v\n", parent1)
	fmt.Printf("Child:  %v\n", child1)

	err = validator.ValidateScope(context.Background(), parent1, child1)
	if err != nil {
		fmt.Printf("❌ Validation failed: %v\n", err)
	} else {
		fmt.Println("✅ Validation successful")
	}
	fmt.Println()

	// Test case 2: Scope escalation (should fail)
	parent2 := []string{"users:read"}
	child2 := []string{"users:write"}
	fmt.Printf("Parent: %v\n", parent2)
	fmt.Printf("Child:  %v\n", child2)

	err = validator.ValidateScope(context.Background(), parent2, child2)
	if err == nil {
		fmt.Println("❌ ERROR: Should have failed for scope escalation")
	} else {
		fmt.Printf("✅ Validation failed as expected: %v\n", err)
	}
	fmt.Println()
}

func example2() {
	fmt.Println("=== Example 2: Multi-Tenant Isolation ===")
	validator, err := NewOPAScopeValidator()
	if err != nil {
		log.Fatal(err)
	}

	// Test tenant-scoped delegation
	parent := []string{"tenant:acme:*"}
	validChild := []string{"tenant:acme:users:read"}
	invalidChild := []string{"tenant:globex:users:read"}

	fmt.Printf("Parent: %v\n", parent)

	// Valid tenant access
	fmt.Printf("Child:  %v\n", validChild)
	err = validator.ValidateScope(context.Background(), parent, validChild)
	if err != nil {
		fmt.Printf("❌ Validation failed: %v\n", err)
	} else {
		fmt.Println("✅ Validation successful (same tenant)")
	}

	// Invalid cross-tenant access
	fmt.Printf("Child:  %v\n", invalidChild)
	err = validator.ValidateScope(context.Background(), parent, invalidChild)
	if err == nil {
		fmt.Println("❌ ERROR: Should have failed for cross-tenant access")
	} else {
		fmt.Printf("✅ Validation failed as expected: %v\n", err)
	}
	fmt.Println()
}

func example3() {
	fmt.Println("=== Example 3: API Versioning ===")
	validator, err := NewOPAScopeValidator()
	if err != nil {
		log.Fatal(err)
	}

	// Admin delegates v1 API access
	parent := []string{"api:v1:*"}
	validChild := []string{"api:v1:users:read", "api:v1:files:write"}
	invalidChild := []string{"api:v2:users:read"}

	fmt.Printf("Parent: %v\n", parent)

	// Valid v1 API access
	fmt.Printf("Child:  %v\n", validChild)
	err = validator.ValidateScope(context.Background(), parent, validChild)
	if err != nil {
		fmt.Printf("❌ Validation failed: %v\n", err)
	} else {
		fmt.Println("✅ Validation successful (v1 API)")
	}

	// Invalid v2 API escalation
	fmt.Printf("Child:  %v\n", invalidChild)
	err = validator.ValidateScope(context.Background(), parent, invalidChild)
	if err == nil {
		fmt.Println("❌ ERROR: Should have failed for version escalation")
	} else {
		fmt.Printf("✅ Validation failed as expected: %v\n", err)
	}
	fmt.Println()
}

func example4() {
	fmt.Println("=== Example 4: Detailed Decision Info ===")
	validator, err := NewOPAScopeValidator()
	if err != nil {
		log.Fatal(err)
	}

	parent := []string{"users:*", "files:read"}
	child := []string{"users:read", "files:read"}

	fmt.Printf("Parent: %v\n", parent)
	fmt.Printf("Child:  %v\n", child)

	allowed, details, err := validator.ValidateWithDetails(context.Background(), parent, child)
	if err != nil {
		// Example 4 depends on "scope_validation.rego" which might not exist in this context.
		// If it fails, we just print the error and move on, to avoid crashing the whole demo.
		fmt.Printf("⚠️  Detailed validation skipped (policy file missing?): %v\n", err)
		return
	}

	fmt.Printf("Allowed: %v\n", allowed)
	fmt.Println("Details:")
	prettyJSON, _ := json.MarshalIndent(details, "  ", "  ")
	fmt.Printf("  %s\n", prettyJSON)
	fmt.Println()
}

func benchmark() {
	fmt.Println("=== Performance Benchmark ===")
	validator, err := NewOPAScopeValidator()
	if err != nil {
		log.Fatal(err)
	}

	parent := []string{"users:*", "files:read"}
	child := []string{"users:read", "users:write"}
	iterations := 10000

	// Warmup
	for i := 0; i < 100; i++ {
		_ = validator.ValidateScope(context.Background(), parent, child)
	}

	start := time.Now()
	for i := 0; i < iterations; i++ {
		_ = validator.ValidateScope(context.Background(), parent, child)
	}
	duration := time.Since(start)

	fmt.Printf("OPA Embedded: %d iterations in %v\n", iterations, duration)
	fmt.Printf("Average:      %.2f μs per validation\n", float64(duration.Microseconds())/float64(iterations))
	fmt.Printf("Throughput:   %.0f validations/sec\n", float64(iterations)/duration.Seconds())
	fmt.Println()
}

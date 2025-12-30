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
	policyBytes, err := os.ReadFile("policies/gauth_basic.rego")
	if err != nil {
		return nil, fmt.Errorf("failed to read policy: %w", err)
	}

	// Prepare query
	query, err := rego.New(
		rego.Query("data.gauth.authz.allow"),
		rego.Module("gauth_basic.rego", string(policyBytes)),
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

	// Check result


































































































































































































































































}	fmt.Println("========================================")	fmt.Println("✅ All examples completed")	fmt.Println("========================================")	benchmark()	example4()	example3()	example2()	example1()	fmt.Println()	fmt.Println("========================================")	fmt.Println("AgentAuth OPA Integration Examples")	fmt.Println("========================================")	}		return		benchmark()	if len(os.Args) > 1 && os.Args[1] == "benchmark" {	// Check if running in example modefunc main() {}	fmt.Println()	fmt.Printf("Throughput:   %.0f validations/sec\n", float64(iterations)/duration.Seconds())	fmt.Printf("Average:      %.2f μs per validation\n", float64(duration.Microseconds())/float64(iterations))	fmt.Printf("OPA Embedded: %d iterations in %v\n", iterations, duration)	duration := time.Since(start)	}		_ = validator.ValidateScope(context.Background(), parent, child)	for i := 0; i < iterations; i++ {	start := time.Now()	iterations := 10000	// Benchmark OPA	}		_ = validator.ValidateScope(context.Background(), parent, child)	for i := 0; i < 100; i++ {	// Warmup	child := []string{"users:read", "users:write"}	parent := []string{"users:*", "files:read"}	}		log.Fatal(err)	if err != nil {	validator, err := NewOPAScopeValidator()	fmt.Println("=== Performance Benchmark ===")func benchmark() {// Benchmark: Compare OPA vs native Go}	fmt.Println()	fmt.Printf("  %s\n", prettyJSON)	prettyJSON, _ := json.MarshalIndent(details, "  ", "  ")	fmt.Println("Details:")	fmt.Printf("Allowed: %v\n", allowed)	}		log.Fatal(err)	if err != nil {	allowed, details, err := validator.ValidateWithDetails(context.Background(), parent, child)	fmt.Printf("Child:  %v\n", child)	fmt.Printf("Parent: %v\n", parent)	child := []string{"users:read", "files:read"}	parent := []string{"users:*", "files:read"}	}		log.Fatal(err)	if err != nil {	validator, err := NewOPAScopeValidator()	fmt.Println("=== Example 4: Detailed Decision Info ===")func example4() {// Example 4: Detailed decision information}	fmt.Println()	}		fmt.Println("❌ ERROR: Should have failed for version escalation")	} else {		fmt.Printf("❌ Validation failed: %v (expected - version escalation)\n", err)	if err != nil {	err = validator.ValidateScope(context.Background(), parent, invalidChild)	fmt.Printf("Child:  %v\n", invalidChild)	// Invalid v2 API escalation	}		fmt.Println("✅ Validation successful (v1 API)")	} else {		fmt.Printf("❌ Validation failed: %v\n", err)	if err != nil {	err = validator.ValidateScope(context.Background(), parent, validChild)	fmt.Printf("Child:  %v\n", validChild)	// Valid v1 API access	fmt.Printf("Parent: %v\n", parent)	invalidChild := []string{"api:v2:users:read"}	validChild := []string{"api:v1:users:read", "api:v1:files:write"}	parent := []string{"api:v1:*"}	// Admin delegates v1 API access	}		log.Fatal(err)	if err != nil {	validator, err := NewOPAScopeValidator()	fmt.Println("=== Example 3: API Versioning ===")func example3() {// Example 3: API versioning}	fmt.Println()	}		fmt.Println("❌ ERROR: Should have failed for cross-tenant access")	} else {		fmt.Printf("❌ Validation failed: %v (expected - cross-tenant)\n", err)	if err != nil {	err = validator.ValidateScope(context.Background(), parent, invalidChild)	fmt.Printf("Child:  %v\n", invalidChild)	// Invalid cross-tenant access	}		fmt.Println("✅ Validation successful (same tenant)")	} else {		fmt.Printf("❌ Validation failed: %v\n", err)	if err != nil {	err = validator.ValidateScope(context.Background(), parent, validChild)	fmt.Printf("Child:  %v\n", validChild)	// Valid tenant access	fmt.Printf("Parent: %v\n", parent)	invalidChild := []string{"tenant:globex:users:read"}	validChild := []string{"tenant:acme:users:read"}	parent := []string{"tenant:acme:*"}	// Test tenant-scoped delegation	}		log.Fatal(err)	if err != nil {	validator, err := NewOPAScopeValidator()	fmt.Println("=== Example 2: Multi-Tenant Isolation ===")func example2() {// Example 2: Multi-tenant isolation}	fmt.Println()	}		fmt.Println("✅ Validation successful")	} else {		fmt.Printf("❌ Validation failed: %v\n", err)	if err != nil {	err = validator.ValidateScope(context.Background(), parent2, child2)	fmt.Printf("Child:  %v\n", child2)	fmt.Printf("Parent: %v\n", parent2)	child2 := []string{"users:write"}	parent2 := []string{"users:read"}	// Test case 2: Scope escalation (should fail)	fmt.Println()	}		fmt.Println("✅ Validation successful")	} else {		fmt.Printf("❌ Validation failed: %v\n", err)	if err != nil {	err = validator.ValidateScope(context.Background(), parent1, child1)	fmt.Printf("Child:  %v\n", child1)	fmt.Printf("Parent: %v\n", parent1)	child1 := []string{"users:read", "users:write"}	parent1 := []string{"users:*"}	// Test case 1: Valid delegation	}		log.Fatal(err)	if err != nil {	validator, err := NewOPAScopeValidator()	fmt.Println("=== Example 1: Basic Scope Validation ===")func example1() {// Example 1: Basic scope validation}	return data	}		panic(err)	if err != nil {	data, err := policies.ReadFile("policies/scope_validation.rego")func mustReadPolicy() []byte {}	return allowed, details, nil	allowed := details["allow"].(bool)	}		return false, nil, fmt.Errorf("unexpected result format")	if !ok {	details, ok := results[0].Expressions[0].Value.(map[string]interface{})	}		return false, nil, fmt.Errorf("OPA returned no results")	if len(results) == 0 {	}		return false, nil, fmt.Errorf("OPA evaluation failed: %w", err)	if err != nil {	results, err := detailsQuery.Eval(ctx, rego.EvalInput(input))	}		return false, nil, fmt.Errorf("failed to prepare details query: %w", err)	if err != nil {	).PrepareForEval(context.Background())		rego.Module("scope_validation.rego", string(mustReadPolicy())),		rego.Query("data.gauth.authz.decision_details"),	detailsQuery, err := rego.New(	// Query both allow and decision_details	}		"child_scopes":  child,		"parent_scopes": parent,		"action":        "validate_scope",	input := map[string]interface{}{func (v *OPAScopeValidator) ValidateWithDetails(ctx context.Context, parent, child []string) (bool, map[string]interface{}, error) {// ValidateWithDetails returns detailed decision information}	return nil	}		return fmt.Errorf("scope validation failed: child scopes not covered by parent")	if !allowed {	}		return fmt.Errorf("OPA returned unexpected result type")	if !ok {	allowed, ok := results[0].Expressions[0].Value.(bool)	}		return fmt.Errorf("OPA returned no results")	if len(results) == 0 {
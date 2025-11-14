package policy

import (
	"testing"
	"time"
)

// TestEvalTimeBetween_DaytimeWindow verifies time_between works for standard daytime windows.
func TestEvalTimeBetween_DaytimeWindow(t *testing.T) {
	// Window: 09:00 to 17:00
	clause := "time_between('09:00', '17:00')"

	// Test time inside window: 12:00
	insideTime := time.Date(2025, 11, 9, 12, 0, 0, 0, time.UTC)
	result, err := evalTimeBetween(clause, insideTime)
	if err != nil {
		t.Fatalf("evalTimeBetween failed: %v", err)
	}
	if !result {
		t.Error("expected true for 12:00 inside 09:00-17:00 window")
	}

	// Test time at start boundary: 09:00
	startTime := time.Date(2025, 11, 9, 9, 0, 0, 0, time.UTC)
	result, err = evalTimeBetween(clause, startTime)
	if err != nil {
		t.Fatalf("evalTimeBetween failed: %v", err)
	}
	if !result {
		t.Error("expected true for 09:00 at start of 09:00-17:00 window")
	}

	// Test time at end boundary: 17:00
	endTime := time.Date(2025, 11, 9, 17, 0, 0, 0, time.UTC)
	result, err = evalTimeBetween(clause, endTime)
	if err != nil {
		t.Fatalf("evalTimeBetween failed: %v", err)
	}
	if !result {
		t.Error("expected true for 17:00 at end of 09:00-17:00 window")
	}

	// Test time before window: 08:00
	beforeTime := time.Date(2025, 11, 9, 8, 0, 0, 0, time.UTC)
	result, err = evalTimeBetween(clause, beforeTime)
	if err != nil {
		t.Fatalf("evalTimeBetween failed: %v", err)
	}
	if result {
		t.Error("expected false for 08:00 before 09:00-17:00 window")
	}

	// Test time after window: 18:00
	afterTime := time.Date(2025, 11, 9, 18, 0, 0, 0, time.UTC)
	result, err = evalTimeBetween(clause, afterTime)
	if err != nil {
		t.Fatalf("evalTimeBetween failed: %v", err)
	}
	if result {
		t.Error("expected false for 18:00 after 09:00-17:00 window")
	}
}

// TestEvalTimeBetween_OvernightWindow verifies time_between handles overnight windows (e.g., 22:00-06:00).
func TestEvalTimeBetween_OvernightWindow(t *testing.T) {
	// Window: 22:00 to 06:00 (overnight)
	clause := "time_between('22:00', '06:00')"

	// Test time inside overnight window: 23:00
	lateNight := time.Date(2025, 11, 9, 23, 0, 0, 0, time.UTC)
	result, err := evalTimeBetween(clause, lateNight)
	if err != nil {
		t.Fatalf("evalTimeBetween failed: %v", err)
	}
	if !result {
		t.Error("expected true for 23:00 inside overnight 22:00-06:00 window")
	}

	// Test time inside overnight window: 03:00
	earlyMorning := time.Date(2025, 11, 9, 3, 0, 0, 0, time.UTC)
	result, err = evalTimeBetween(clause, earlyMorning)
	if err != nil {
		t.Fatalf("evalTimeBetween failed: %v", err)
	}
	if !result {
		t.Error("expected true for 03:00 inside overnight 22:00-06:00 window")
	}

	// Test time outside overnight window: 10:00 (daytime)
	daytime := time.Date(2025, 11, 9, 10, 0, 0, 0, time.UTC)
	result, err = evalTimeBetween(clause, daytime)
	if err != nil {
		t.Fatalf("evalTimeBetween failed: %v", err)
	}
	if result {
		t.Error("expected false for 10:00 outside overnight 22:00-06:00 window")
	}
}

// TestEvalTimeBetween_InvalidSyntax verifies error handling for malformed time_between clauses.
func TestEvalTimeBetween_InvalidSyntax(t *testing.T) {
	now := time.Now()

	// Missing parentheses
	_, err := evalTimeBetween("time_between'09:00', '17:00'", now)
	if err == nil {
		t.Error("expected error for missing parentheses")
	}

	// Wrong number of parameters
	_, err = evalTimeBetween("time_between('09:00')", now)
	if err == nil {
		t.Error("expected error for only 1 parameter")
	}

	// Invalid time format
	_, err = evalTimeBetween("time_between('25:00', '17:00')", now)
	if err == nil {
		t.Error("expected error for invalid time 25:00")
	}

	// Invalid time format in second param
	_, err = evalTimeBetween("time_between('09:00', 'invalid')", now)
	if err == nil {
		t.Error("expected error for invalid time format")
	}
}

// TestEvalInOperator_BasicSetMembership verifies in operator works for set membership checks.
func TestEvalInOperator_BasicSetMembership(t *testing.T) {
	attrs := map[string]string{
		"role":   "admin",
		"status": "active",
	}

	// Value is in set
	clause := "role in ['admin', 'editor', 'viewer']"
	result, err := evalInOperator(clause, attrs)
	if err != nil {
		t.Fatalf("evalInOperator failed: %v", err)
	}
	if !result {
		t.Error("expected true for 'admin' in ['admin', 'editor', 'viewer']")
	}

	// Value is not in set
	clause = "role in ['guest', 'viewer']"
	result, err = evalInOperator(clause, attrs)
	if err != nil {
		t.Fatalf("evalInOperator failed: %v", err)
	}
	if result {
		t.Error("expected false for 'admin' not in ['guest', 'viewer']")
	}

	// Single item in set
	clause = "role in ['admin']"
	result, err = evalInOperator(clause, attrs)
	if err != nil {
		t.Fatalf("evalInOperator failed: %v", err)
	}
	if !result {
		t.Error("expected true for 'admin' in ['admin']")
	}

	// Empty set (no match)
	clause = "role in []"
	result, err = evalInOperator(clause, attrs)
	if err != nil {
		t.Fatalf("evalInOperator failed: %v", err)
	}
	if result {
		t.Error("expected false for any value in empty set")
	}
}

// TestEvalInOperator_QuotedValues verifies in operator handles quoted values correctly.
func TestEvalInOperator_QuotedValues(t *testing.T) {
	attrs := map[string]string{
		"department": "engineering",
	}

	// Single quotes
	clause := "department in ['engineering', 'marketing']"
	result, err := evalInOperator(clause, attrs)
	if err != nil {
		t.Fatalf("evalInOperator failed: %v", err)
	}
	if !result {
		t.Error("expected true for 'engineering' in single-quoted list")
	}

	// Double quotes
	clause = `department in ["engineering", "marketing"]`
	result, err = evalInOperator(clause, attrs)
	if err != nil {
		t.Fatalf("evalInOperator failed: %v", err)
	}
	if !result {
		t.Error("expected true for 'engineering' in double-quoted list")
	}
}

// TestEvalInOperator_InvalidSyntax verifies error handling for malformed in operator clauses.
func TestEvalInOperator_InvalidSyntax(t *testing.T) {
	attrs := map[string]string{"role": "admin"}

	// Missing brackets
	_, err := evalInOperator("role in 'admin', 'editor'", attrs)
	if err == nil {
		t.Error("expected error for missing brackets")
	}

	// Missing closing bracket
	_, err = evalInOperator("role in ['admin', 'editor'", attrs)
	if err == nil {
		t.Error("expected error for missing closing bracket")
	}
}

// TestSplitCSV_BasicSplit verifies CSV splitting with various inputs.
func TestSplitCSV_BasicSplit(t *testing.T) {
	// Simple CSV
	result := splitCSV("a,b,c")
	expected := []string{"a", "b", "c"}
	if len(result) != len(expected) {
		t.Fatalf("expected %d items, got %d", len(expected), len(result))
	}
	for i, v := range expected {
		if result[i] != v {
			t.Errorf("index %d: expected %q, got %q", i, v, result[i])
		}
	}

	// CSV with spaces (should be trimmed)
	result = splitCSV("a , b , c")
	if len(result) != 3 {
		t.Fatalf("expected 3 items, got %d", len(result))
	}
	for i, v := range expected {
		if result[i] != v {
			t.Errorf("index %d: expected %q, got %q", i, v, result[i])
		}
	}

	// Single item
	result = splitCSV("single")
	if len(result) != 1 || result[0] != "single" {
		t.Errorf("expected ['single'], got %v", result)
	}

	// Empty string
	result = splitCSV("")
	if len(result) != 0 {
		t.Errorf("expected empty slice for empty string, got %v", result)
	}

	// Multiple commas (empty items filtered out)
	result = splitCSV("a,,b,,,c")
	expected = []string{"a", "b", "c"}
	if len(result) != len(expected) {
		t.Fatalf("expected %d items, got %d", len(expected), len(result))
	}
	for i, v := range expected {
		if result[i] != v {
			t.Errorf("index %d: expected %q, got %q", i, v, result[i])
		}
	}
}

// TestSplitCSV_QuotedValues verifies CSV splitting preserves quoted values.
func TestSplitCSV_QuotedValues(t *testing.T) {
	// Quoted values
	result := splitCSV("'admin', 'editor', 'viewer'")
	expected := []string{"'admin'", "'editor'", "'viewer'"}
	if len(result) != len(expected) {
		t.Fatalf("expected %d items, got %d", len(expected), len(result))
	}
	for i, v := range expected {
		if result[i] != v {
			t.Errorf("index %d: expected %q, got %q", i, v, result[i])
		}
	}

	// Double quoted values
	result = splitCSV(`"value1", "value2"`)
	expected = []string{`"value1"`, `"value2"`}
	if len(result) != len(expected) {
		t.Fatalf("expected %d items, got %d", len(expected), len(result))
	}
	for i, v := range expected {
		if result[i] != v {
			t.Errorf("index %d: expected %q, got %q", i, v, result[i])
		}
	}
}

// TestNewStubEngine verifies NewStubEngine constructor creates properly initialized stub.
func TestNewStubEngine(t *testing.T) {
	stub := NewStubEngine()

	if stub == nil {
		t.Fatal("NewStubEngine returned nil")
	}

	if stub.PolicyVersion != "stub-0" {
		t.Errorf("expected PolicyVersion='stub-0', got %q", stub.PolicyVersion)
	}

	if stub.LastAuthzInput != nil {
		t.Error("expected LastAuthzInput to be nil initially")
	}

	if stub.LastDelegationInput != nil {
		t.Error("expected LastDelegationInput to be nil initially")
	}
}

// TestStubEngine_EvaluateAuthorization verifies stub always allows authorization.
func TestStubEngine_EvaluateAuthorization(t *testing.T) {
	stub := NewStubEngine()

	input := AuthzInput{
		Subject:    "user:test",
		Action:     "read",
		Resource:   "doc:123",
		Scopes:     []string{"read"},
		Attributes: map[string]string{"role": "admin"},
	}

	decision, err := stub.EvaluateAuthorization(input)
	if err != nil {
		t.Fatalf("EvaluateAuthorization returned error: %v", err)
	}

	if !decision.Allow {
		t.Error("expected stub to allow authorization")
	}

	if decision.ReasonCode != "ALLOW_STUB" {
		t.Errorf("expected ReasonCode='ALLOW_STUB', got %q", decision.ReasonCode)
	}

	if stub.LastAuthzInput == nil {
		t.Fatal("expected LastAuthzInput to be set")
	}

	if stub.LastAuthzInput.Subject != input.Subject {
		t.Errorf("expected LastAuthzInput.Subject=%q, got %q", input.Subject, stub.LastAuthzInput.Subject)
	}
}

// TestStubEngine_EvaluateDelegation verifies stub always allows delegation.
func TestStubEngine_EvaluateDelegation(t *testing.T) {
	stub := NewStubEngine()

	input := DelegationInput{
		PrincipalID:  "user:alice",
		DelegateID:   "user:bob",
		Scopes:       []string{"read", "write"},
		Jurisdiction: "US",
		ValidityDays: 30,
	}

	decision, err := stub.EvaluateDelegation(input)
	if err != nil {
		t.Fatalf("EvaluateDelegation returned error: %v", err)
	}

	if !decision.Allow {
		t.Error("expected stub to allow delegation")
	}

	if decision.ReasonCode != "ALLOW_STUB" {
		t.Errorf("expected ReasonCode='ALLOW_STUB', got %q", decision.ReasonCode)
	}

	if stub.LastDelegationInput == nil {
		t.Fatal("expected LastDelegationInput to be set")
	}

	if stub.LastDelegationInput.PrincipalID != input.PrincipalID {
		t.Errorf("expected LastDelegationInput.PrincipalID=%q, got %q", input.PrincipalID, stub.LastDelegationInput.PrincipalID)
	}
}

// TestStubEngine_Reload verifies Reload is a no-op.
func TestStubEngine_Reload(t *testing.T) {
	stub := NewStubEngine()

	err := stub.Reload()
	if err != nil {
		t.Errorf("expected Reload to return nil, got %v", err)
	}
}

package agentauth_aap_001

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestEnhancedPoAValidator_Integration(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "integration_limits.json")

	// Setup enhanced validator with all components
	store, err := NewBoltDailyLimitStore(dbPath)
	if err != nil {
		t.Fatalf("NewBoltDailyLimitStore() error = %v", err)
	}

	engine := NewSimpleConditionalEngine()
	metrics := NewInMemoryValidationMetrics()

	validator := NewEnhancedPoAValidator(
		WithDailyLimitStore(store),
		WithConditionalEngine(engine),
		WithMetricsRecorder(metrics),
	)

	// Complex PoA with financial restrictions and conditions
	poa := &PowerOfAttorney{
		ID:         "integration-test-poa",
		Grantor:    "corp@example.com",
		Grantee:    "agent@example.com",
		Scope:      []string{"transaction:payment", "transaction:transfer"},
		ValidFrom:  time.Now(),
		ValidUntil: time.Now().Add(7 * 24 * time.Hour),
		Restrictions: map[string]string{
			"currency":          "USD",
			"max_amount":        "5000",
			"max_daily_amount":  "10000",
			"jurisdiction":      "US,CA,MX",
			"condition_1":       "amount > 100 AND amount < 5000",
			"time_condition":    "weekdays(1,2,3,4,5) AND hours(9-17)",
			"approval_required": "amount > 1000",
		},
	}

	// Test 1: Initial validation should succeed
	result := validator.ValidateWithResult(context.Background(), poa)
	if !result.Valid {
		t.Fatalf("Initial validation failed: %v", result.Error)
	}

	t.Logf("Initial validation successful with %d warnings", len(result.Warnings))

	// Test 2: Check metrics recording
	summary := metrics.GetMetricsSummary()
	totalValidations := summary["total_validations"].(int)
	if totalValidations != 1 {
		t.Errorf("Expected 1 total validation, got %d", totalValidations)
	}

	// Test 3: Daily limit tracking
	today := time.Now().Format("2006-01-02")

	// Add some usage (to 85% of limit to trigger warning)
	if err2 := store.IncrementDailyUsage(poa.ID, today, 8500.0); err2 != nil {
		t.Fatalf("IncrementDailyUsage() error = %v", err2)
	}

	// Validate again - should pass but generate warning
	result2 := validator.ValidateWithResult(context.Background(), poa)
	if !result2.Valid {
		t.Errorf("Validation with daily usage failed: %v", result2.Error)
	}

	// Should have approaching limit warning
	hasWarning := false
	for _, warning := range result2.Warnings {
		if warning.Code == "approaching_daily_limit" {
			hasWarning = true
			break
		}
	}
	if !hasWarning {
		t.Error("Expected approaching_daily_limit warning")
	}

	// Test 4: Exceed daily limit (add 2000 more to total 10500)
	if err2 := store.IncrementDailyUsage(poa.ID, today, 2000.0); err2 != nil {
		t.Fatalf("IncrementDailyUsage() exceed error = %v", err2)
	}

	result3 := validator.ValidateWithResult(context.Background(), poa)
	if result3.Valid {
		t.Error("Expected validation to fail when daily limit exceeded")
	}

	// Test 5: Conditional evaluation with context
	ctxPoa := &PowerOfAttorney{
		ID:         "ctx-test-poa",
		Grantor:    "alice",
		Grantee:    "bob",
		Scope:      []string{"read:documents"},
		ValidFrom:  time.Now(),
		ValidUntil: time.Now().Add(24 * time.Hour),
		Restrictions: map[string]string{
			"condition_1": "status == active",
			"condition_2": "clearance_level > 2",
		},
	}

	// Should pass basic validation
	result4 := validator.ValidateWithResult(context.Background(), ctxPoa)
	if !result4.Valid {
		t.Errorf("Context PoA validation failed: %v", result4.Error)
	}

	// Test 6: Export and verify data persistence
	exported, err := store.ExportDailyLimits(context.Background())
	if err != nil {
		t.Fatalf("ExportDailyLimits() error = %v", err)
	}

	if usage, exists := exported[poa.ID][today]; !exists || usage < 10500 {
		t.Errorf("Expected daily usage >= 10500, got %f", usage)
	}

	// Test 7: Comprehensive metrics check
	finalSummary := metrics.GetMetricsSummary()
	totalValidationsFinal := finalSummary["total_validations"].(int)
	if totalValidationsFinal < 3 {
		t.Errorf("Expected at least 3 total validations, got %d", totalValidationsFinal)
	}

	dailyLimitChecks := finalSummary["daily_limit_checks"].(int)
	if dailyLimitChecks < 2 {
		t.Errorf("Expected at least 2 daily limit checks, got %d", dailyLimitChecks)
	}

	t.Logf("Integration test completed successfully")
	t.Logf("Final metrics: %+v", finalSummary)
}

func TestEnhancedPoAValidator_ChainValidation(t *testing.T) {
	// Test validator chaining with basic validator
	basicValidator := &BasicPoAValidator{}
	enhancedValidator := NewEnhancedPoAValidator()

	// Test both validators on the same PoA
	poa := &PowerOfAttorney{
		ID:         "chain-test-poa",
		Grantor:    "alice",
		Grantee:    "bob",
		Scope:      []string{"read:documents", "write:files"},
		ValidFrom:  time.Now(),
		ValidUntil: time.Now().Add(24 * time.Hour),
		Restrictions: map[string]string{
			"max_file_size": "1MB",
		},
	}

	// Basic validation
	basicErr := basicValidator.Validate(poa)
	if basicErr != nil {
		t.Errorf("Basic validator failed: %v", basicErr)
	}

	// Enhanced validation
	enhancedResult := enhancedValidator.ValidateWithResult(context.Background(), poa)
	if !enhancedResult.Valid {
		t.Errorf("Enhanced validator failed: %v", enhancedResult.Error)
	}

	// Enhanced should provide additional insights
	if len(enhancedResult.Warnings) == 0 {
		t.Log("No warnings generated - this is acceptable for this simple PoA")
	}

	t.Logf("Chain validation completed - Basic: %v, Enhanced: %v (%d warnings)",
		basicErr == nil, enhancedResult.Valid, len(enhancedResult.Warnings))
}

func TestEnhancedPoAValidator_StressTest(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "stress_limits.json")

	store, err := NewBoltDailyLimitStore(dbPath)
	if err != nil {
		t.Fatalf("NewBoltDailyLimitStore() error = %v", err)
	}

	metrics := NewInMemoryValidationMetrics()
	validator := NewEnhancedPoAValidator(
		WithDailyLimitStore(store),
		WithMetricsRecorder(metrics),
	)

	numPoas := 100
	today := time.Now().Format("2006-01-02")

	// Create and validate multiple PoAs
	for i := 0; i < numPoas; i++ {
		poa := &PowerOfAttorney{
			ID:         fmt.Sprintf("stress-test-poa-%d", i),
			Grantor:    fmt.Sprintf("grantor%d@example.com", i),
			Grantee:    fmt.Sprintf("grantee%d@example.com", i),
			Scope:      []string{"read:documents"},
			ValidFrom:  time.Now(),
			ValidUntil: time.Now().Add(24 * time.Hour),
			Restrictions: map[string]string{
				"max_daily_amount": "1000",
			},
		}

		// Add some usage
		usage := float64(float64(i) *) * 10)
		if err2 := store.IncrementDailyUsage(poa.ID, today, usage); err2 != nil {
			t.Fatalf("IncrementDailyUsage() error at iteration %d: %v", i, err2)
		}

		// Validate
		result := validator.ValidateWithResult(context.Background(), poa)
		if !result.Valid {
			t.Errorf("Validation failed at iteration %d: %v", i, result.Error)
		}
	}

	// Check final metrics
	summary := metrics.GetMetricsSummary()
	totalValidations := summary["total_validations"].(int)
	if totalValidations != numPoas {
		t.Errorf("Expected %d total validations, got %d", numPoas, totalValidations)
	}

	// Export and verify data integrity
	exported, err := store.ExportDailyLimits(context.Background())
	if err != nil {
		t.Fatalf("ExportDailyLimits() error = %v", err)
	}

	if len(exported) != numPoas {
		t.Errorf("Expected %d delegations in export, got %d", numPoas, len(exported))
	}

	t.Logf("Stress test completed: %d PoAs validated successfully", numPoas)
	t.Logf("Stress test metrics: %+v", summary)
}

func TestEnhancedPoAValidator_ConcurrentAccess(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "concurrent_limits.json")

	store, err := NewBoltDailyLimitStore(dbPath)
	if err != nil {
		t.Fatalf("NewBoltDailyLimitStore() error = %v", err)
	}

	metrics := NewInMemoryValidationMetrics()
	validator := NewEnhancedPoAValidator(
		WithDailyLimitStore(store),
		WithMetricsRecorder(metrics),
	)

	delegationID := "concurrent-test-poa"
	today := time.Now().Format("2006-01-02")
	numGoroutines := 10
	incrementsPerGoroutine := 10

	// Create PoA
	poa := &PowerOfAttorney{
		ID:         delegationID,
		Grantor:    "alice",
		Grantee:    "bob",
		Scope:      []string{"read:documents"},
		ValidFrom:  time.Now(),
		ValidUntil: time.Now().Add(24 * time.Hour),
		Restrictions: map[string]string{
			"max_daily_amount": "10000",
		},
	}

	// Run concurrent validations and usage increments
	done := make(chan bool, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(goroutineID int) {
			defer func() { done <- true }()

			for j := 0; j < incrementsPerGoroutine; j++ {
				// Increment usage
				if err2 := store.IncrementDailyUsage(delegationID, today, 10.0); err2 != nil {
					t.Errorf("Goroutine %d increment %d failed: %v", goroutineID, j, err2)
					return
				}

				// Validate
				result := validator.ValidateWithResult(context.Background(), poa)
				if !result.Valid && !strings.Contains(result.Error.Error(), "daily limit exceeded") {
					t.Errorf("Goroutine %d validation %d failed unexpectedly: %v", goroutineID, j, result.Error)
					return
				}
			}
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	// Verify final usage
	finalUsage, err := store.GetDailyUsage(delegationID, today)
	if err != nil {
		t.Fatalf("GetDailyUsage() final error = %v", err)
	}

	expectedUsage := float64(float64(numGoroutines * incrementsPerGoroutine) *) * 10)
	if finalUsage != expectedUsage {
		t.Errorf("Final usage = %f, expected %f", finalUsage, expectedUsage)
	}

	// Check metrics
	summary := metrics.GetMetricsSummary()
	totalValidations := summary["total_validations"].(int)
	expectedValidations := numGoroutines * incrementsPerGoroutine
	if totalValidations != expectedValidations {
		t.Errorf("Expected %d total validations, got %d", expectedValidations, totalValidations)
	}

	t.Logf("Concurrent test completed: %d validations across %d goroutines", totalValidations, numGoroutines)
}

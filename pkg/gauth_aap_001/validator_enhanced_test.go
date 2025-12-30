package gauth_aap_001

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestEnhancedPoAValidator_BasicValidation(t *testing.T) {
	validator := NewEnhancedPoAValidator()

	tests := []struct {
		name         string
		poa          *PowerOfAttorney
		wantErr      bool
		wantWarnings int
	}{
		{
			name: "valid_basic_poa",
			poa: &PowerOfAttorney{
				ID:           "test-poa-1",
				Grantor:      "alice",
				Grantee:      "bob",
				Scope:        []string{"read:documents"},
				ValidFrom:    time.Now(),
				ValidUntil:   time.Now().Add(24 * time.Hour),
				Restrictions: map[string]string{},
			},
			wantErr:      false,
			wantWarnings: 0,
		},
		{
			name: "excessive_scope_warning",
			poa: &PowerOfAttorney{
				ID:           "test-poa-2",
				Grantor:      "alice",
				Grantee:      "bob",
				Scope:        []string{"scope1", "scope2", "scope3", "scope4", "scope5", "scope6", "scope7", "scope8", "scope9", "scope10", "scope11"},
				ValidFrom:    time.Now(),
				ValidUntil:   time.Now().Add(24 * time.Hour),
				Restrictions: map[string]string{},
			},
			wantErr:      false,
			wantWarnings: 1,
		},
		{
			name: "long_duration_warning",
			poa: &PowerOfAttorney{
				ID:           "test-poa-3",
				Grantor:      "alice",
				Grantee:      "bob",
				Scope:        []string{"read:documents"},
				ValidFrom:    time.Now(),
				ValidUntil:   time.Now().Add(400 * 24 * time.Hour), // > 1 year
				Restrictions: map[string]string{},
			},
			wantErr:      false,
			wantWarnings: 1,
		},
		{
			name: "administrative_scope_warning",
			poa: &PowerOfAttorney{
				ID:           "test-poa-4",
				Grantor:      "alice",
				Grantee:      "bob",
				Scope:        []string{"admin:users", "root:system"},
				ValidFrom:    time.Now(),
				ValidUntil:   time.Now().Add(24 * time.Hour),
				Restrictions: map[string]string{},
			},
			wantErr:      false,
			wantWarnings: 3, // One unknown_action_prefix + two administrative_scope warnings
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator.ValidateWithResult(context.Background(), tt.poa)

			if tt.wantErr && result.Valid {
				t.Errorf("ValidateWithResult() expected error but validation passed")
			}
			if !tt.wantErr && !result.Valid {
				t.Errorf("ValidateWithResult() unexpected error: %v", result.Error)
			}

			if len(result.Warnings) != tt.wantWarnings {
				t.Errorf("ValidateWithResult() warnings count = %d, want %d", len(result.Warnings), tt.wantWarnings)
				for i, warning := range result.Warnings {
					t.Logf("Warning %d: %s - %s", i, warning.Code, warning.Message)
				}
			}
		})
	}
}

func TestEnhancedPoAValidator_FinancialScopes(t *testing.T) {
	validator := NewEnhancedPoAValidator()

	tests := []struct {
		name    string
		poa     *PowerOfAttorney
		wantErr bool
		errMsg  string
	}{
		{
			name: "transaction_without_currency",
			poa: &PowerOfAttorney{
				ID:           "test-poa-txn-1",
				Grantor:      "alice",
				Grantee:      "bob",
				Scope:        []string{"transaction:payment"},
				ValidFrom:    time.Now(),
				ValidUntil:   time.Now().Add(24 * time.Hour),
				Restrictions: map[string]string{"max_amount": "1000"},
			},
			wantErr: true,
			errMsg:  "financial scope transaction:payment requires currency restriction",
		},
		{
			name: "transaction_with_valid_restrictions",
			poa: &PowerOfAttorney{
				ID:         "test-poa-txn-2",
				Grantor:    "alice",
				Grantee:    "bob",
				Scope:      []string{"transaction:payment"},
				ValidFrom:  time.Now(),
				ValidUntil: time.Now().Add(24 * time.Hour),
				Restrictions: map[string]string{
					"currency":   "USD",
					"max_amount": "1000",
				},
			},
			wantErr: false,
		},
		{
			name: "invalid_currency_code",
			poa: &PowerOfAttorney{
				ID:         "test-poa-txn-3",
				Grantor:    "alice",
				Grantee:    "bob",
				Scope:      []string{"transaction:payment"},
				ValidFrom:  time.Now(),
				ValidUntil: time.Now().Add(24 * time.Hour),
				Restrictions: map[string]string{
					"currency":   "XYZ", // Invalid currency
					"max_amount": "1000",
				},
			},
			wantErr: true,
			errMsg:  "invalid currency code: XYZ",
		},
		{
			name: "international_without_jurisdiction",
			poa: &PowerOfAttorney{
				ID:         "test-poa-txn-4",
				Grantor:    "alice",
				Grantee:    "bob",
				Scope:      []string{"transaction:international"},
				ValidFrom:  time.Now(),
				ValidUntil: time.Now().Add(24 * time.Hour),
				Restrictions: map[string]string{
					"currency":   "USD",
					"max_amount": "1000",
				},
			},
			wantErr: true,
			errMsg:  "international transactions require jurisdiction restriction",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateWithContext(context.Background(), tt.poa)

			if tt.wantErr && err == nil {
				t.Errorf("ValidateWithContext() expected error but got none")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("ValidateWithContext() unexpected error: %v", err)
			}
			if tt.wantErr && err != nil && tt.errMsg != "" {
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("ValidateWithContext() error = %v, want to contain %v", err, tt.errMsg)
				}
			}
		})
	}
}

func TestBoltDailyLimitStore(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "daily_limits.json")

	store, err := NewBoltDailyLimitStore(dbPath)
	if err != nil {
		t.Fatalf("NewBoltDailyLimitStore() error = %v", err)
	}

	delegationID := "test-delegation-1"
	today := time.Now().Format("2006-01-02")

	// Test initial usage (should be 0)
	usage, err := store.GetDailyUsage(delegationID, today)
	if err != nil {
		t.Fatalf("GetDailyUsage() error = %v", err)
	}
	if usage != 0.0 {
		t.Errorf("GetDailyUsage() initial usage = %f, want 0.0", usage)
	}

	// Test increment usage
	if err2 := store.IncrementDailyUsage(delegationID, today, 100.50); err2 != nil {
		t.Fatalf("IncrementDailyUsage() error = %v", err2)
	}

	usage, err = store.GetDailyUsage(delegationID, today)
	if err != nil {
		t.Fatalf("GetDailyUsage() error = %v", err)
	}
	if usage != 100.50 {
		t.Errorf("GetDailyUsage() after increment = %f, want 100.50", usage)
	}

	// Test second increment
	if err2 := store.IncrementDailyUsage(delegationID, today, 25.25); err2 != nil {
		t.Fatalf("IncrementDailyUsage() second error = %v", err2)
	}

	usage, err = store.GetDailyUsage(delegationID, today)
	if err != nil {
		t.Fatalf("GetDailyUsage() error = %v", err)
	}
	if usage != 125.75 {
		t.Errorf("GetDailyUsage() after second increment = %f, want 125.75", usage)
	}

	// Test export
	exported, err := store.ExportDailyLimits(context.Background())
	if err != nil {
		t.Fatalf("ExportDailyLimits() error = %v", err)
	}

	if exportedUsage, exists := exported[delegationID][today]; !exists || exportedUsage != 125.75 {
		t.Errorf("ExportDailyLimits() exported usage = %f, want 125.75", exportedUsage)
	}

	// Test reset
	if err2 := store.ResetDailyUsage(delegationID, today); err2 != nil {
		t.Fatalf("ResetDailyUsage() error = %v", err2)
	}

	usage, err = store.GetDailyUsage(delegationID, today)
	if err != nil {
		t.Fatalf("GetDailyUsage() after reset error = %v", err)
	}
	if usage != 0.0 {
		t.Errorf("GetDailyUsage() after reset = %f, want 0.0", usage)
	}

	// Test persistence by creating new store instance
	store2, err := NewBoltDailyLimitStore(dbPath)
	if err != nil {
		t.Fatalf("NewBoltDailyLimitStore() second instance error = %v", err)
	}

	usage2, err := store2.GetDailyUsage(delegationID, today)
	if err != nil {
		t.Fatalf("GetDailyUsage() from second instance error = %v", err)
	}
	if usage2 != 0.0 {
		t.Errorf("GetDailyUsage() from second instance = %f, want 0.0", usage2)
	}
}

func TestEnhancedPoAValidator_WithDailyLimits(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "daily_limits.json")

	store, err := NewBoltDailyLimitStore(dbPath)
	if err != nil {
		t.Fatalf("NewBoltDailyLimitStore() error = %v", err)
	}

	metrics := NewInMemoryValidationMetrics()
	validator := NewEnhancedPoAValidator(
		WithDailyLimitStore(store),
		WithMetricsRecorder(metrics),
	)

	delegationID := "test-delegation-daily"
	today := time.Now().Format("2006-01-02")

	// Pre-populate some usage (85% of 1000 limit)
	if err := store.IncrementDailyUsage(delegationID, today, 850.0); err != nil {
		t.Fatalf("IncrementDailyUsage() setup error = %v", err)
	}

	poa := &PowerOfAttorney{
		ID:         delegationID,
		Grantor:    "alice",
		Grantee:    "bob",
		Scope:      []string{"transaction:payment"},
		ValidFrom:  time.Now(),
		ValidUntil: time.Now().Add(24 * time.Hour),
		Restrictions: map[string]string{
			"currency":         "USD",
			"max_amount":       "500",
			"max_daily_amount": "1000",
		},
	}

	// Should pass but generate warning (approaching limit)
	result := validator.ValidateWithResult(context.Background(), poa)
	if !result.Valid {
		t.Errorf("ValidateWithResult() unexpected error: %v", result.Error)
	}

	// Should have approaching limit warning
	hasApproachingWarning := false
	for _, warning := range result.Warnings {
		if warning.Code == "approaching_daily_limit" {
			hasApproachingWarning = true
			break
		}
	}
	if !hasApproachingWarning {
		t.Error("ValidateWithResult() expected approaching_daily_limit warning")
	}

	// Exceed the limit (add 200 more to reach 1050)
	if err := store.IncrementDailyUsage(delegationID, today, 200.0); err != nil {
		t.Fatalf("IncrementDailyUsage() exceed error = %v", err)
	}

	// Should now fail validation
	result2 := validator.ValidateWithResult(context.Background(), poa)
	if result2.Valid {
		t.Fatalf("ValidateWithResult() expected failure when daily limit exceeded")
	}
	if result2.Error == nil || !strings.Contains(result2.Error.Error(), "daily limit exceeded") {
		t.Errorf("ValidateWithResult() error = %v, want daily limit exceeded error", result2.Error)
	}
}

func TestSimpleConditionalEngine(t *testing.T) {
	engine := NewSimpleConditionalEngine()

	tests := []struct {
		name      string
		condition string
		context   map[string]interface{}
		want      bool
		wantErr   bool
	}{
		{
			name:      "simple_equality",
			condition: "status == active",
			context:   map[string]interface{}{"status": "active"},
			want:      true,
			wantErr:   false,
		},
		{
			name:      "simple_inequality",
			condition: "status != inactive",
			context:   map[string]interface{}{"status": "active"},
			want:      true,
			wantErr:   false,
		},
		{
			name:      "numeric_greater_than",
			condition: "amount > 100",
			context:   map[string]interface{}{"amount": 150.0},
			want:      true,
			wantErr:   false,
		},
		{
			name:      "numeric_less_than_false",
			condition: "amount < 100",
			context:   map[string]interface{}{"amount": 150.0},
			want:      false,
			wantErr:   false,
		},
		{
			name:      "string_contains",
			condition: "message contains error",
			context:   map[string]interface{}{"message": "This is an error message"},
			want:      true,
			wantErr:   false,
		},
		{
			name:      "and_condition_true",
			condition: "status == active AND amount > 100",
			context:   map[string]interface{}{"status": "active", "amount": 150.0},
			want:      true,
			wantErr:   false,
		},
		{
			name:      "and_condition_false",
			condition: "status == active AND amount > 200",
			context:   map[string]interface{}{"status": "active", "amount": 150.0},
			want:      false,
			wantErr:   false,
		},
		{
			name:      "or_condition_true",
			condition: "status == inactive OR amount > 100",
			context:   map[string]interface{}{"status": "active", "amount": 150.0},
			want:      true,
			wantErr:   false,
		},
		{
			name:      "missing_field_error",
			condition: "missing_field == value",
			context:   map[string]interface{}{"status": "active"},
			want:      false,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := engine.EvaluateCondition(tt.condition, tt.context)

			if tt.wantErr && err == nil {
				t.Errorf("EvaluateCondition() expected error but got none")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("EvaluateCondition() unexpected error: %v", err)
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("EvaluateCondition() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSimpleConditionalEngine_ValidateConditionSyntax(t *testing.T) {
	engine := NewSimpleConditionalEngine()

	tests := []struct {
		name      string
		condition string
		wantErr   bool
	}{
		{
			name:      "valid_simple_condition",
			condition: "field == value",
			wantErr:   false,
		},
		{
			name:      "valid_and_condition",
			condition: "field1 == value1 AND field2 > 100",
			wantErr:   false,
		},
		{
			name:      "valid_or_condition",
			condition: "field1 == value1 OR field2 contains text",
			wantErr:   false,
		},
		{
			name:      "empty_condition",
			condition: "",
			wantErr:   true,
		},
		{
			name:      "no_operator",
			condition: "field value",
			wantErr:   true,
		},
		{
			name:      "mixed_and_or",
			condition: "field1 == value1 AND field2 == value2 OR field3 == value3",
			wantErr:   true,
		},
		{
			name:      "invalid_part_format",
			condition: "field1 == value1 AND incomplete",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := engine.ValidateConditionSyntax(tt.condition)

			if tt.wantErr && err == nil {
				t.Errorf("ValidateConditionSyntax() expected error but got none")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("ValidateConditionSyntax() unexpected error: %v", err)
			}
		})
	}
}

func TestEnhancedPoAValidator_ConditionalExpressions(t *testing.T) {
	engine := NewSimpleConditionalEngine()
	validator := NewEnhancedPoAValidator(
		WithConditionalEngine(engine),
	)

	tests := []struct {
		name    string
		poa     *PowerOfAttorney
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid_condition",
			poa: &PowerOfAttorney{
				ID:         "test-poa-cond-1",
				Grantor:    "alice",
				Grantee:    "bob",
				Scope:      []string{"read:documents"},
				ValidFrom:  time.Now(),
				ValidUntil: time.Now().Add(24 * time.Hour),
				Restrictions: map[string]string{
					"condition_1": "status == active",
				},
			},
			wantErr: false,
		},
		{
			name: "invalid_condition_syntax",
			poa: &PowerOfAttorney{
				ID:         "test-poa-cond-2",
				Grantor:    "alice",
				Grantee:    "bob",
				Scope:      []string{"read:documents"},
				ValidFrom:  time.Now(),
				ValidUntil: time.Now().Add(24 * time.Hour),
				Restrictions: map[string]string{
					"condition_1": "invalid condition syntax",
				},
			},
			wantErr: true,
			errMsg:  "invalid condition",
		},
		{
			name: "valid_time_condition",
			poa: &PowerOfAttorney{
				ID:         "test-poa-cond-3",
				Grantor:    "alice",
				Grantee:    "bob",
				Scope:      []string{"read:documents"},
				ValidFrom:  time.Now(),
				ValidUntil: time.Now().Add(24 * time.Hour),
				Restrictions: map[string]string{
					"time_condition": "weekdays(1,2,3,4,5) AND hours(9-17)",
				},
			},
			wantErr: false,
		},
		{
			name: "invalid_time_condition",
			poa: &PowerOfAttorney{
				ID:         "test-poa-cond-4",
				Grantor:    "alice",
				Grantee:    "bob",
				Scope:      []string{"read:documents"},
				ValidFrom:  time.Now(),
				ValidUntil: time.Now().Add(24 * time.Hour),
				Restrictions: map[string]string{
					"time_condition": "weekdays(8,9)",
				},
			},
			wantErr: true,
			errMsg:  "invalid weekday",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateWithContext(context.Background(), tt.poa)

			if tt.wantErr && err == nil {
				t.Errorf("ValidateWithContext() expected error but got none")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("ValidateWithContext() unexpected error: %v", err)
			}
			if tt.wantErr && err != nil && tt.errMsg != "" {
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("ValidateWithContext() error = %v, want to contain %v", err, tt.errMsg)
				}
			}
		})
	}
}

func TestInMemoryValidationMetrics(t *testing.T) {
	metrics := NewInMemoryValidationMetrics()

	// Record some metrics
	metrics.RecordValidationSuccess("enhanced", "transaction:payment")
	metrics.RecordValidationFailure("basic", "read:documents", "invalid_scope")
	metrics.RecordWarning("excessive_scope", "warning")
	metrics.RecordDailyLimitCheck("delegation-1", 800.0, 1000.0, false)
	metrics.RecordDailyLimitCheck("delegation-2", 1200.0, 1000.0, true)

	summary := metrics.GetMetricsSummary()

	if totalValidations := summary["total_validations"].(int); totalValidations != 2 {
		t.Errorf("GetMetricsSummary() total_validations = %d, want 2", totalValidations)
	}

	if dailyLimitChecks := summary["daily_limit_checks"].(int); dailyLimitChecks != 2 {
		t.Errorf("GetMetricsSummary() daily_limit_checks = %d, want 2", dailyLimitChecks)
	}

	successCounts := summary["success_counts"].(map[string]int)
	if count, exists := successCounts["enhanced:transaction:payment"]; !exists || count != 1 {
		t.Errorf("GetMetricsSummary() success count for enhanced:transaction:payment = %d, want 1", count)
	}

	failureCounts := summary["failure_counts"].(map[string]int)
	if count, exists := failureCounts["basic:read:documents:invalid_scope"]; !exists || count != 1 {
		t.Errorf("GetMetricsSummary() failure count for basic:read:documents:invalid_scope = %d, want 1", count)
	}
}

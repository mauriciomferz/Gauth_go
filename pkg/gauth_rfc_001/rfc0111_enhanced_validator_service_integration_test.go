package gauth_rfc_001

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/mauriciomferz/Gauth_go/pkg/audit"
	"github.com/mauriciomferz/Gauth_go/pkg/authz"
	cr "github.com/mauriciomferz/Gauth_go/pkg/crypto"
)

// TestEnhancedValidatorServiceIntegration_WarningCollection tests that warnings are collected during delegation creation
func TestEnhancedValidatorServiceIntegration_WarningCollection(t *testing.T) {
	ctx := context.Background()

	// Setup enhanced validator with daily limit store
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "daily_limits.json")
	store, err := NewBoltDailyLimitStore(dbPath)
	if err != nil {
		t.Fatalf("Failed to create daily limit store: %v", err)
	}

	metrics := NewInMemoryValidationMetrics()
	enhancedValidator := NewEnhancedPoAValidator(
		WithDailyLimitStore(store),
		WithMetricsRecorder(metrics),
	)

	// Create service with enhanced validator
	kp, err := cr.NewInMemoryEd25519Provider()
	if err != nil {
		t.Fatalf("Failed to create key provider: %v", err)
	}

	auditLogger := audit.NewMemoryLogger(nil)
	authorizer := authz.NewMemoryAuthorizer()
	authorizer.AddPolicy(authz.Policy{
		ID:       "p1",
		Subject:  "alice",
		Resource: "*",
		Actions:  []string{"create_delegation"},
		Effect:   authz.Allow,
	})

	svc := NewService(auditLogger, authorizer,
		WithEnhancedValidator(enhancedValidator),
		WithSignerProvider(kp.ActiveSigner),
	)

	// Create delegation with high amount that triggers warning
	req := DelegationRequest{
		Grantor:  "alice",
		Grantee:  "bob",
		Scope:    []string{"transaction:withdraw"},
		Duration: 24 * time.Hour,
		Restrictions: map[string]string{
			"currency":   "USD",
			"max_amount": "1500000", // High amount should trigger warning
		},
	}

	resp, err := svc.CreateDelegationCtx(ctx, req)
	if err != nil {
		t.Fatalf("CreateDelegationCtx failed: %v", err)
	}

	// Verify warnings were collected
	if len(resp.Warnings) == 0 {
		t.Error("Expected warnings to be collected but got none")
	}

	// Check for high_amount_limit warning
	foundHighAmountWarning := false
	for _, warning := range resp.Warnings {
		if warning.Code == "high_amount_limit" {
			foundHighAmountWarning = true
			if warning.Severity != "warning" {
				t.Errorf("Expected severity='warning', got %s", warning.Severity)
			}
		}
	}

	if !foundHighAmountWarning {
		t.Error("Expected high_amount_limit warning but didn't find it")
	}

	// Verify metrics were recorded
	summary := metrics.GetMetricsSummary()
	totalValidations := summary["total_validations"].(int)
	if totalValidations != 1 {
		t.Errorf("Expected 1 validation, got %d", totalValidations)
	}
}

// TestEnhancedValidatorServiceIntegration_DailyLimitEnforcement tests daily limit enforcement
func TestEnhancedValidatorServiceIntegration_DailyLimitEnforcement(t *testing.T) {
	ctx := context.Background()

	// Setup enhanced validator with daily limit store
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "daily_limits.json")
	store, err := NewBoltDailyLimitStore(dbPath)
	if err != nil {
		t.Fatalf("Failed to create daily limit store: %v", err)
	}

	enhancedValidator := NewEnhancedPoAValidator(
		WithDailyLimitStore(store),
	)

	// Create service with enhanced validator
	kp, err := cr.NewInMemoryEd25519Provider()
	if err != nil {
		t.Fatalf("Failed to create key provider: %v", err)
	}

	auditLogger := audit.NewMemoryLogger(nil)
	authorizer := authz.NewMemoryAuthorizer()
	authorizer.AddPolicy(authz.Policy{
		ID:       "p1",
		Subject:  "alice",
		Resource: "*",
		Actions:  []string{"create_delegation"},
		Effect:   authz.Allow,
	})

	svc := NewService(auditLogger, authorizer,
		WithEnhancedValidator(enhancedValidator),
		WithSignerProvider(kp.ActiveSigner),
	)

	// Create delegation with daily limit
	req := DelegationRequest{
		Grantor:  "alice",
		Grantee:  "bob",
		Scope:    []string{"transaction:withdraw"},
		Duration: 24 * time.Hour,
		Restrictions: map[string]string{
			"currency":         "USD",
			"max_daily_amount": "10000",
			"max_amount":       "5000",
		},
	}

	// First delegation should succeed
	resp1, err := svc.CreateDelegationCtx(ctx, req)
	if err != nil {
		t.Fatalf("First CreateDelegationCtx failed: %v", err)
	}

	// Simulate usage to 85% of daily limit (should trigger warning)
	today := time.Now().Format("2006-01-02")
	if err := store.IncrementDailyUsage(resp1.POA.ID, today, 8500); err != nil {
		t.Fatalf("IncrementDailyUsage failed: %v", err)
	}

	// Create another delegation with same ID (should get approaching limit warning)
	// Note: For this test we're using the validator directly since CreateDelegationCtx creates new IDs
	poa := &PowerOfAttorney{
		ID:         resp1.POA.ID,
		Grantor:    "alice",
		Grantee:    "bob",
		Scope:      []string{"transaction:withdraw"},
		ValidFrom:  time.Now(),
		ValidUntil: time.Now().Add(24 * time.Hour),
		Restrictions: map[string]string{
			"currency":         "USD",
			"max_daily_amount": "10000",
			"max_amount":       "5000",
		},
	}

	result := enhancedValidator.ValidateWithResult(ctx, poa)
	if !result.Valid {
		t.Errorf("Validation should pass but got error: %v", result.Error)
	}

	// Should have approaching_daily_limit warning
	foundApproachingWarning := false
	for _, warning := range result.Warnings {
		if warning.Code == "approaching_daily_limit" {
			foundApproachingWarning = true
		}
	}
	if !foundApproachingWarning {
		t.Error("Expected approaching_daily_limit warning")
	}

	// Exceed daily limit
	if err := store.IncrementDailyUsage(resp1.POA.ID, today, 2000); err != nil {
		t.Fatalf("IncrementDailyUsage failed: %v", err)
	}

	// Validation should now fail
	result2 := enhancedValidator.ValidateWithResult(ctx, poa)
	if result2.Valid {
		t.Error("Validation should fail when daily limit exceeded")
	}
}

// TestEnhancedValidatorServiceIntegration_BackwardCompatibility tests that Service works without enhanced validator
func TestEnhancedValidatorServiceIntegration_BackwardCompatibility(t *testing.T) {
	ctx := context.Background()

	// Create service WITHOUT enhanced validator
	kp, err := cr.NewInMemoryEd25519Provider()
	if err != nil {
		t.Fatalf("Failed to create key provider: %v", err)
	}

	auditLogger := audit.NewMemoryLogger(nil)
	authorizer := authz.NewMemoryAuthorizer()
	authorizer.AddPolicy(authz.Policy{
		ID:       "p1",
		Subject:  "alice",
		Resource: "*",
		Actions:  []string{"create_delegation"},
		Effect:   authz.Allow,
	})

	svc := NewService(auditLogger, authorizer,
		WithSignerProvider(kp.ActiveSigner),
	)

	// Create delegation should work fine
	req := DelegationRequest{
		Grantor:  "alice",
		Grantee:  "bob",
		Scope:    []string{"read:documents"},
		Duration: 24 * time.Hour,
	}

	resp, err := svc.CreateDelegationCtx(ctx, req)
	if err != nil {
		t.Fatalf("CreateDelegationCtx failed: %v", err)
	}

	// Warnings should be empty or nil (backward compatible)
	if len(resp.Warnings) != 0 {
		t.Errorf("Expected no warnings when enhanced validator not configured, got %d", len(resp.Warnings))
	}

	// POA should be created successfully
	if resp.POA.ID == "" {
		t.Error("Expected POA to be created")
	}
}

// TestEnhancedValidatorServiceIntegration_MetricsRecording tests that metrics are properly recorded
func TestEnhancedValidatorServiceIntegration_MetricsRecording(t *testing.T) {
	ctx := context.Background()

	// Setup enhanced validator with metrics
	metrics := NewInMemoryValidationMetrics()
	enhancedValidator := NewEnhancedPoAValidator(
		WithMetricsRecorder(metrics),
	)

	// Create service
	kp, err := cr.NewInMemoryEd25519Provider()
	if err != nil {
		t.Fatalf("Failed to create key provider: %v", err)
	}

	auditLogger := audit.NewMemoryLogger(nil)
	authorizer := authz.NewMemoryAuthorizer()
	authorizer.AddPolicy(authz.Policy{
		ID:       "p1",
		Subject:  "alice",
		Resource: "*",
		Actions:  []string{"create_delegation"},
		Effect:   authz.Allow,
	})

	svc := NewService(auditLogger, authorizer,
		WithEnhancedValidator(enhancedValidator),
		WithSignerProvider(kp.ActiveSigner),
	)

	// Create successful delegation
	req := DelegationRequest{
		Grantor:  "alice",
		Grantee:  "bob",
		Scope:    []string{"read:documents"},
		Duration: 24 * time.Hour,
	}

	_, err = svc.CreateDelegationCtx(ctx, req)
	if err != nil {
		t.Fatalf("CreateDelegationCtx failed: %v", err)
	}

	// Verify metrics
	summary := metrics.GetMetricsSummary()
	totalValidations := summary["total_validations"].(int)
	if totalValidations != 1 {
		t.Errorf("Expected 1 total validation, got %d", totalValidations)
	}

	successCounts := summary["success_counts"].(map[string]int)
	foundEnhancedSuccess := false
	for key, count := range successCounts {
		if count > 0 && (key == "enhanced:transaction:withdraw" || key == "enhanced:read:documents") {
			foundEnhancedSuccess = true
			break
		}
	}
	if !foundEnhancedSuccess {
		t.Errorf("Expected enhanced validation success in metrics, got: %v", successCounts)
	}
}

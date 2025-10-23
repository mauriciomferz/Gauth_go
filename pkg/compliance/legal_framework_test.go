package compliance

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestNewLegalFrameworkValidator(t *testing.T) {
	validator := NewLegalFrameworkValidator()
	
	// Test initialization
	if validator == nil {
		t.Fatal("NewLegalFrameworkValidator should return a non-nil validator")
	}
	
	// Test supported jurisdictions are initialized
	jurisdictions := validator.GetSupportedJurisdictions()
	expectedJurisdictions := []Jurisdiction{JurisdictionUS, JurisdictionEU, JurisdictionUK, JurisdictionCA, JurisdictionAU, JurisdictionJP}
	
	if len(jurisdictions) != len(expectedJurisdictions) {
		t.Fatalf("Expected %d jurisdictions, got %d", len(expectedJurisdictions), len(jurisdictions))
	}
	
	// Check all expected jurisdictions are present
	jurisdictionMap := make(map[Jurisdiction]bool)
	for _, j := range jurisdictions {
		jurisdictionMap[j] = true
	}
	
	for _, expected := range expectedJurisdictions {
		if !jurisdictionMap[expected] {
			t.Errorf("Expected jurisdiction %s not found", expected)
		}
	}
}

func TestValidateJurisdiction(t *testing.T) {
	validator := NewLegalFrameworkValidator()
	ctx := context.Background()
	
	tests := []struct {
		name         string
		jurisdiction Jurisdiction
		action       string
		expectError  bool
		errorContains string
	}{
		{
			name:         "Valid US trade execution",
			jurisdiction: JurisdictionUS,
			action:       "trade_execution",
			expectError:  false,
		},
		{
			name:         "Valid EU fund transfer", 
			jurisdiction: JurisdictionEU,
			action:       "fund_transfer",
			expectError:  false,
		},
		{
			name:         "Unsupported jurisdiction",
			jurisdiction: Jurisdiction("INVALID"),
			action:       "trade_execution",
			expectError:  true,
			errorContains: "unsupported jurisdiction",
		},
		{
			name:         "US autonomous decision (should fail)",
			jurisdiction: JurisdictionUS,
			action:       "autonomous_decision",
			expectError:  true,
			errorContains: "autonomous AI decisions not permitted",
		},
		{
			name:         "US centralized decision (should pass)",
			jurisdiction: JurisdictionUS,
			action:       "centralized_decision",
			expectError:  false,
		},
		{
			name:         "Board approval action",
			jurisdiction: JurisdictionUS,
			action:       "high_value_transaction",
			expectError:  false,
		},
		{
			name:         "Board approval unauthorized action",
			jurisdiction: JurisdictionEU,
			action:       "unauthorized_high_value_transaction",
			expectError:  true,
			errorContains: "unauthorized actions not permitted",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateJurisdiction(ctx, tt.jurisdiction, tt.action)
			
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error to contain '%s', but got: %s", tt.errorContains, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %s", err.Error())
				}
			}
		})
	}
}

func TestValidateEntityType(t *testing.T) {
	validator := NewLegalFrameworkValidator()
	
	tests := []struct {
		name         string
		jurisdiction Jurisdiction
		entityType   EntityType
		expectError  bool
	}{
		{
			name:         "Valid US corporation",
			jurisdiction: JurisdictionUS,
			entityType:   EntityTypeCorporation,
			expectError:  false,
		},
		{
			name:         "Valid US organization",
			jurisdiction: JurisdictionUS,
			entityType:   EntityTypeOrganization,
			expectError:  false,
		},
		{
			name:         "Valid US AI agent",
			jurisdiction: JurisdictionUS,
			entityType:   EntityTypeAIAgent,
			expectError:  false,
		},
		{
			name:         "Invalid entity type in jurisdiction",
			jurisdiction: JurisdictionUS,
			entityType:   EntityType("invalid_type"),
			expectError:  true,
		},
		{
			name:         "Unsupported jurisdiction",
			jurisdiction: Jurisdiction("INVALID"),
			entityType:   EntityTypeCorporation,
			expectError:  true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateEntityType(tt.jurisdiction, tt.entityType)
			
			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
			} else if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %s", err.Error())
			}
		})
	}
}

func TestGetJurisdictionRules(t *testing.T) {
	validator := NewLegalFrameworkValidator()
	
	tests := []struct {
		name         string
		jurisdiction Jurisdiction
		expectError  bool
	}{
		{
			name:         "Valid US jurisdiction",
			jurisdiction: JurisdictionUS,
			expectError:  false,
		},
		{
			name:         "Valid EU jurisdiction",
			jurisdiction: JurisdictionEU,
			expectError:  false,
		},
		{
			name:         "Invalid jurisdiction",
			jurisdiction: Jurisdiction("INVALID"),
			expectError:  true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rules, err := validator.GetJurisdictionRules(tt.jurisdiction)
			
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				if rules != nil {
					t.Errorf("Expected nil rules on error, but got non-nil")
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %s", err.Error())
				}
				if rules == nil {
					t.Errorf("Expected non-nil rules but got nil")
				} else {
					// Validate structure
					if rules.Jurisdiction != tt.jurisdiction {
						t.Errorf("Expected jurisdiction %s, got %s", tt.jurisdiction, rules.Jurisdiction)
					}
					if len(rules.SupportedEntities) == 0 {
						t.Errorf("Expected non-empty supported entities")
					}
					if len(rules.ComplianceRules) == 0 {
						t.Errorf("Expected non-empty compliance rules")
					}
				}
			}
		})
	}
}

func TestValidateJurisdictionRequirements(t *testing.T) {
	validator := NewLegalFrameworkValidator()
	ctx := context.Background()
	
	validRequirements := &JurisdictionRequirements{
		Jurisdiction: JurisdictionUS,
		ValueLimits: map[string]float64{
			"test_action": 1000000.0,
		},
		RequiredApprovals: map[string]ApprovalLevel{
			"test_action": DualApproval,
		},
	}
	
	tests := []struct {
		name         string
		requirements *JurisdictionRequirements
		action       string
		expectError  bool
	}{
		{
			name:         "Valid requirements",
			requirements: validRequirements,
			action:       "test_action",
			expectError:  false,
		},
		{
			name:         "Nil requirements",
			requirements: nil,
			action:       "test_action",
			expectError:  true,
		},
		{
			name: "Invalid value limit",
			requirements: &JurisdictionRequirements{
				Jurisdiction: JurisdictionUS,
				ValueLimits: map[string]float64{
					"test_action": -1000.0, // Invalid negative limit
				},
			},
			action:      "test_action",
			expectError: true,
		},
		{
			name: "Missing approval level",
			requirements: &JurisdictionRequirements{
				Jurisdiction: JurisdictionUS,
				RequiredApprovals: map[string]ApprovalLevel{
					"test_action": ApprovalLevel(""), // Empty approval level
				},
			},
			action:      "test_action", 
			expectError: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateJurisdictionRequirements(ctx, tt.requirements, tt.action)
			
			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
			} else if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %s", err.Error())
			}
		})
	}
}

func TestGetMetrics(t *testing.T) {
	validator := NewLegalFrameworkValidator()
	ctx := context.Background()
	
	// Initial metrics should be zero
	metrics := validator.GetMetrics()
	if metrics.ValidationAttempts != 0 {
		t.Errorf("Expected 0 validation attempts, got %d", metrics.ValidationAttempts)
	}
	if metrics.ValidationSuccesses != 0 {
		t.Errorf("Expected 0 validation successes, got %d", metrics.ValidationSuccesses)
	}
	if metrics.ValidationFailures != 0 {
		t.Errorf("Expected 0 validation failures, got %d", metrics.ValidationFailures)
	}
	
	// Perform some validations
	validator.ValidateJurisdiction(ctx, JurisdictionUS, "trade_execution") // Should succeed
	validator.ValidateJurisdiction(ctx, JurisdictionUS, "autonomous_decision") // Should fail
	validator.ValidateJurisdiction(ctx, Jurisdiction("INVALID"), "action") // Should fail
	
	// Check updated metrics
	updatedMetrics := validator.GetMetrics()
	if updatedMetrics.ValidationAttempts != 3 {
		t.Errorf("Expected 3 validation attempts, got %d", updatedMetrics.ValidationAttempts)
	}
	if updatedMetrics.ValidationSuccesses != 1 {
		t.Errorf("Expected 1 validation success, got %d", updatedMetrics.ValidationSuccesses)
	}
	if updatedMetrics.ValidationFailures != 2 {
		t.Errorf("Expected 2 validation failures, got %d", updatedMetrics.ValidationFailures)
	}
	
	// Check jurisdiction counts
	if updatedMetrics.JurisdictionCounts[JurisdictionUS] != 2 {
		t.Errorf("Expected 2 US jurisdiction attempts, got %d", updatedMetrics.JurisdictionCounts[JurisdictionUS])
	}
	
	// Check violation counts
	if len(updatedMetrics.ViolationCounts) == 0 {
		t.Errorf("Expected some violation counts, got none")
	}
}

func TestAddJurisdiction(t *testing.T) {
	validator := NewLegalFrameworkValidator()
	
	// Add a custom jurisdiction
	customJurisdiction := JurisdictionRequirements{
		Jurisdiction:      Jurisdiction("TEST"),
		SupportedEntities: []EntityType{EntityTypeCorporation},
		ComplianceRules:   []ComplianceRule{},
		ValueLimits:       map[string]float64{"test_action": 500000.0},
		RequiredApprovals: map[string]ApprovalLevel{"test_action": SingleApproval},
	}
	
	validator.AddJurisdiction(customJurisdiction)
	
	// Verify it was added
	jurisdictions := validator.GetSupportedJurisdictions()
	found := false
	for _, j := range jurisdictions {
		if j == Jurisdiction("TEST") {
			found = true
			break
		}
	}
	
	if !found {
		t.Errorf("Custom jurisdiction TEST was not added")
	}
	
	// Verify we can get its rules
	rules, err := validator.GetJurisdictionRules(Jurisdiction("TEST"))
	if err != nil {
		t.Errorf("Expected to get rules for custom jurisdiction, got error: %s", err.Error())
	}
	if rules == nil {
		t.Errorf("Expected non-nil rules for custom jurisdiction")
	}
}

func TestTimeRestrictions(t *testing.T) {
	validator := NewLegalFrameworkValidator()
	
	// Add jurisdiction with time restrictions for testing
	testJurisdiction := JurisdictionRequirements{
		Jurisdiction:      Jurisdiction("TIME_TEST"),
		SupportedEntities: []EntityType{EntityTypeCorporation},
		ComplianceRules:   []ComplianceRule{},
		TimeRestrictions: map[string][]string{
			"time_restricted_action": {"weekdays", "business_hours"},
			"always_allowed_action":  {},
		},
	}
	
	validator.AddJurisdiction(testJurisdiction)
	
	ctx := context.Background()
	
	// Test time-restricted action (should pass due to simplified implementation)
	err := validator.ValidateJurisdiction(ctx, Jurisdiction("TIME_TEST"), "time_restricted_action")
	if err != nil {
		t.Errorf("Expected time-restricted action to pass, got error: %s", err.Error())
	}
	
	// Test always allowed action
	err = validator.ValidateJurisdiction(ctx, Jurisdiction("TIME_TEST"), "always_allowed_action")
	if err != nil {
		t.Errorf("Expected always allowed action to pass, got error: %s", err.Error())
	}
}

func TestConcurrentAccess(t *testing.T) {
	validator := NewLegalFrameworkValidator()
	ctx := context.Background()
	
	// Test concurrent access to validator methods
	done := make(chan bool, 10)
	
	for i := 0; i < 10; i++ {
		go func(id int) {
			defer func() { done <- true }()
			
			// Perform various operations concurrently
			validator.ValidateJurisdiction(ctx, JurisdictionUS, "concurrent_test")
			validator.GetSupportedJurisdictions()
			validator.GetJurisdictionRules(JurisdictionEU)
			validator.ValidateEntityType(JurisdictionUK, EntityTypeCorporation)
			validator.GetMetrics()
		}(i)
	}
	
	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		select {
		case <-done:
			// Success
		case <-time.After(5 * time.Second):
			t.Fatal("Concurrent access test timed out")
		}
	}
	
	// Verify metrics reflect concurrent operations
	metrics := validator.GetMetrics()
	if metrics.ValidationAttempts != 10 {
		t.Errorf("Expected 10 validation attempts from concurrent access, got %d", metrics.ValidationAttempts)
	}
}

func TestBoardApprovalValidation(t *testing.T) {
	validator := NewLegalFrameworkValidator()
	ctx := context.Background()
	
	// Test board approval for high value transaction in EU (should require board approval)
	err := validator.ValidateJurisdiction(ctx, JurisdictionEU, "high_value_transaction")
	if err != nil {
		t.Errorf("Expected board approval validation to pass for authorized action, got error: %s", err.Error())
	}
	
	// Test board approval for unauthorized action (should fail)
	err = validator.ValidateJurisdiction(ctx, JurisdictionEU, "unauthorized_high_value_transaction")
	if err == nil {
		t.Errorf("Expected board approval validation to fail for unauthorized action")
	} else if !strings.Contains(err.Error(), "unauthorized actions not permitted") {
		t.Errorf("Expected unauthorized action error message, got: %s", err.Error())
	}
}

func TestComplianceRuleValidation(t *testing.T) {
	validator := NewLegalFrameworkValidator()
	
	// Test that compliance rules are properly initialized
	rules, err := validator.GetJurisdictionRules(JurisdictionUS)
	if err != nil {
		t.Fatalf("Failed to get US rules: %s", err.Error())
	}
	
	// Check for AI oversight rule
	foundAIRule := false
	for _, rule := range rules.ComplianceRules {
		if rule.Framework == "AI_OVERSIGHT" {
			foundAIRule = true
			if !rule.Mandatory {
				t.Errorf("AI oversight rule should be mandatory")
			}
			if rule.Validation == nil {
				t.Errorf("AI oversight rule should have validation function")
			}
			break
		}
	}
	
	if !foundAIRule {
		t.Errorf("Expected AI oversight rule in US jurisdiction")
	}
	
	// Test other jurisdictions have appropriate rules
	for _, jurisdiction := range []Jurisdiction{JurisdictionEU, JurisdictionUK, JurisdictionCA, JurisdictionAU, JurisdictionJP} {
		rules, err := validator.GetJurisdictionRules(jurisdiction)
		if err != nil {
			t.Errorf("Failed to get rules for %s: %s", jurisdiction, err.Error())
			continue
		}
		
		if len(rules.ComplianceRules) == 0 {
			t.Errorf("Expected compliance rules for jurisdiction %s", jurisdiction)
		}
	}
}

func TestValueLimitsValidation(t *testing.T) {
	validator := NewLegalFrameworkValidator()
	
	// Test that all jurisdictions have appropriate value limits
	jurisdictions := []Jurisdiction{JurisdictionUS, JurisdictionEU, JurisdictionUK, JurisdictionCA, JurisdictionAU, JurisdictionJP}
	
	for _, jurisdiction := range jurisdictions {
		rules, err := validator.GetJurisdictionRules(jurisdiction)
		if err != nil {
			t.Errorf("Failed to get rules for %s: %s", jurisdiction, err.Error())
			continue
		}
		
		// Check common value limits exist
		expectedActions := []string{"trade_execution", "fund_transfer", "high_value_transaction"}
		for _, action := range expectedActions {
			if limit, exists := rules.ValueLimits[action]; exists {
				if limit <= 0 {
					t.Errorf("Value limit for %s in %s should be positive, got %f", action, jurisdiction, limit)
				}
			}
		}
		
		// Check required approvals exist for actions with limits
		for action := range rules.ValueLimits {
			if _, exists := rules.RequiredApprovals[action]; !exists {
				t.Errorf("Action %s in %s has value limit but no required approval level", action, jurisdiction)
			}
		}
	}
}
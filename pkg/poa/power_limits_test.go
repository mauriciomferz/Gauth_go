package poa

import (
	"strings"
	"testing"
)

func TestPowerLimitsValidation_RFC115_C5(t *testing.T) {
	// Test Numeric Enforcement
	t.Run("NumericEnforcement", func(t *testing.T) {
		validLimits := &PowerLimitSet{
			ModelLimits: &ModelLimits{
				MaxParameters:    175000000000,
				MaxContextWindow: 32000,
			},
			OutcomeLimitations: &OutcomeLimitations{
				MinAccuracyThreshold: 0.95,
				MaxUncertainty:       0.1,
			},
		}
		if err := validLimits.Validate(); err != nil {
			t.Errorf("Valid limits failed validation: %v", err)
		}

		invalidLimits := &PowerLimitSet{
			ModelLimits: &ModelLimits{
				MaxParameters: -1,
			},
		}
		if err := invalidLimits.Validate(); err == nil {
			t.Error("Expected error for negative MaxParameters")
		}

		invalidOutcome := &PowerLimitSet{
			OutcomeLimitations: &OutcomeLimitations{
				MinAccuracyThreshold: 1.5, // > 1.0
			},
		}
		if err := invalidOutcome.Validate(); err == nil {
			t.Error("Expected error for MinAccuracyThreshold > 1.0")
		}
	})

	// Test Logical Consistency (Conflict Detection)
	t.Run("LogicalConsistency", func(t *testing.T) {
		conflicting := &ModelLimits{
			AllowedMethods:    []string{"beam_search"},
			ProhibitedMethods: []string{"beam_search"}, // Conflict
		}
		if err := conflicting.Validate(); err == nil {
			t.Error("Expected error for conflicting allowed/prohibited methods")
		} else if !strings.Contains(err.Error(), "both allowed and prohibited") {
			t.Errorf("Unexpected error message: %v", err)
		}
	})

	// Test Enforcement Helper
	t.Run("EnforcementHelper", func(t *testing.T) {
		limits := &PowerLimitSet{
			BehavioralLimits: &BehavioralLimits{
				ProhibitedActions: []string{"transfer_funds", "delete_db"},
			},
		}

		// Prohibited action
		err := EnforcePowerLimits("transfer_funds:acct123", limits)
		if err == nil {
			t.Error("Expected enforcement error for prohibited action")
		}

		// Allowed action
		err = EnforcePowerLimits("read_logs", limits)
		if err != nil {
			t.Errorf("Unexpected enforcement error for allowed action: %v", err)
		}

		// Nil limits check
		err = EnforcePowerLimits("anything", nil)
		if err != nil {
			t.Error("Expected strict nil limits to allow (or default behavior, creating validation logic check)")
		}
	})
	// Test Structural Limits
	t.Run("StructuralLimits", func(t *testing.T) {
		validLimits := &PowerLimitSet{
			ResourceLimits: &ResourceLimits{
				MaxNodes: 100,
				MaxDepth: 5,
				MaxWidth: 10,
			},
		}

		if err := validLimits.Validate(); err != nil {
			t.Errorf("Valid structural limits failed validation: %v", err)
		}

		negativeNodes := &PowerLimitSet{
			ResourceLimits: &ResourceLimits{
				MaxNodes: -1,
			},
		}

		if err := negativeNodes.Validate(); err == nil {
			t.Error("Expected error for negative MaxNodes")
		} else if !strings.Contains(err.Error(), "max nodes cannot be negative") {
			t.Errorf("Unexpected error message: %v", err)
		}

		negativeDepth := &PowerLimitSet{
			ResourceLimits: &ResourceLimits{
				MaxDepth: -5,
			},
		}

		if err := negativeDepth.Validate(); err == nil {
			t.Error("Expected error for negative MaxDepth")
		} else if !strings.Contains(err.Error(), "max depth cannot be negative") {
			t.Errorf("Unexpected error message: %v", err)
		}

		negativeWidth := &PowerLimitSet{
			ResourceLimits: &ResourceLimits{
				MaxWidth: -10,
			},
		}

		if err := negativeWidth.Validate(); err == nil {
			t.Error("Expected error for negative MaxWidth")
		} else if !strings.Contains(err.Error(), "max width cannot be negative") {
			t.Errorf("Unexpected error message: %v", err)
		}
	})
}

package poa

import (
	"testing"
	"time"
)

func TestRightsObligationsExecution_RFC115_C6(t *testing.T) {
	// Test Reporting Duty Validation
	t.Run("ReportingDutyValidation", func(t *testing.T) {
		validDuty := ReportingDuty{
			Type:       ReportTypeActivity,
			Frequency:  FrequencyDaily,
			Recipients: []string{"audit@example.com"},
			Content:    []string{"logs"},
			Mandatory:  true,
		}
		if err := validDuty.Validate(); err != nil {
			t.Errorf("Valid duty failed validation: %v", err)
		}

		invalid := validDuty
		invalid.Type = ""
		if err := invalid.Validate(); err == nil {
			t.Error("Expected error for missing ReportType")
		}
	})

	// Test Reporting Compliance Enforcement
	t.Run("ReportingComplianceEnforcement", func(t *testing.T) {
		duty := ReportingDuty{
			Type:      ReportTypePerformance,
			Frequency: FrequencyDaily,
			Mandatory: true,
		}

		now := time.Now()

		// Case 1: Just reported (within 24h)
		lastReport := now.Add(-1 * time.Hour)
		if err := EnforceReportingCompliance(lastReport, duty); err != nil {
			t.Errorf("Expected compliant (1h ago for daily): %v", err)
		}

		// Case 2: Overdue (reported 25h ago)
		lastReport = now.Add(-25 * time.Hour)
		if err := EnforceReportingCompliance(lastReport, duty); err == nil {
			t.Error("Expected compliance violation (25h ago for daily)")
		}

		// Case 3: Not mandatory
		duty.Mandatory = false
		if err := EnforceReportingCompliance(lastReport, duty); err != nil {
			t.Error("Expected no error for non-mandatory overdue report")
		}
	})

	// Test Liability Rule Structure
	t.Run("LiabilityRuleStructure", func(t *testing.T) {
		rule := LiabilityRule{
			RuleID:        "L1",
			PrimaryParty:  LiabilityOperator,
			LiabilityType: LiabilityStrict,
		}
		if err := rule.Validate(); err != nil {
			t.Errorf("Valid liability rule failed: %v", err)
		}

		// Insurance check
		rule.InsuranceRequired = true
		if err := rule.Validate(); err == nil {
			t.Error("Expected error when insurance required but coverage missing")
		}

		amt := MonetaryAmount{Amount: 1000000, Currency: "USD"}
		rule.MinInsuranceCoverage = &amt
		if err := rule.Validate(); err != nil {
			t.Errorf("Should be valid with insurance coverage provided: %v", err)
		}
	})
}

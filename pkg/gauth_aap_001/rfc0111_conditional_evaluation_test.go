package gauth_aap_001

import (
	"context"
	"testing"
	"time"

	"github.com/mauriciomferz/Gauth_go/pkg/rfc"
)

func TestConditionalSyntaxValidation(t *testing.T) {
	engine := NewSimpleConditionalEngine()
	// Create collector to capture warnings
	collector := NewWarningCollector()
	validator := NewEnhancedPoAValidator(
		WithConditionalEngine(engine),
		WithWarningCollector(collector),
	)

	// Case 1: Invalid Syntax (Missing operator)
	pInvalid := &PowerOfAttorney{
		Grantor:    "alice",
		Grantee:    "bob",
		Scope:      []string{"read"},
		ValidUntil: time.Now().Add(1 * time.Hour),
		Restrictions: map[string]string{
			"condition_bad": "field_value_no_op",
		},
	}

	_ = validator.Validate(pInvalid)
	warnings := collector.GetWarnings()

	// Debug print
	t.Logf("Warnings collected: %d", len(warnings))
	for i, w := range warnings {
		t.Logf("Warning %d: Code=%s Msg=%s Field=%s", i, w.Code, w.Message, w.Field)
	}

	found := false
	for _, w := range warnings {
		if w.Code == "invalid_condition_syntax" && w.Field == "restriction" && w.Value == "condition_bad" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected warning for invalid syntax, got none")
	}

	// Case 2: Valid Syntax (Quotes)
	pValid := &PowerOfAttorney{
		Grantor:    "alice",
		Grantee:    "bob",
		Scope:      []string{"read"},
		ValidUntil: time.Now().Add(1 * time.Hour),
		Restrictions: map[string]string{
			"condition_good": "status == \"active user\"",
		},
	}
	collector.ClearWarnings()
	_ = validator.Validate(pValid)
	warnings = collector.GetWarnings()
	for _, w := range warnings {
		if w.Code == "invalid_condition_syntax" {
			t.Errorf("Unexpected syntax warning for valid condition: %v", w)
		}
	}
}

func TestConditionalRuntimeEnforcement(t *testing.T) {
	engine := NewSimpleConditionalEngine()
	validator := NewEnhancedPoAValidator(WithConditionalEngine(engine))
	ctx := context.Background()

	p := &PowerOfAttorney{
		Restrictions: map[string]string{
			// Numeric check
			"condition_min_balance": "balance >= 100",
			// String check with quotes
			"condition_status": "status == \"gold member\"",
			// Existence check (implicit)
			"condition_verified": "is_verified == true",
		},
		Scope: []string{"transfer"},
	}

	// Case 1: Failure (Balance too low)
	ctxDataFail := map[string]interface{}{
		"balance":     50,
		"status":      "gold member",
		"is_verified": true,
	}
	err := validator.EvaluatePoAConditions(ctx, p, ctxDataFail)
	if err == nil {
		t.Error("Expected error for low balance, got nil")
	} else if rfcErr, ok := err.(rfc.RFCError); !ok || rfcErr.Code != rfc.ErrRestrictionExceeded {
		t.Errorf("Expected ErrRestrictionExceeded, got %v", err)
	}

	// Case 2: Failure (Wrong status)
	ctxDataStatusFail := map[string]interface{}{
		"balance":     150,
		"status":      "silver member",
		"is_verified": true,
	}
	err = validator.EvaluatePoAConditions(ctx, p, ctxDataStatusFail)
	if err == nil {
		t.Error("Expected error for wrong status, got nil")
	}

	// Case 3: Success
	ctxDataSuccess := map[string]interface{}{
		"balance":     100, // Edge case exact match
		"status":      "gold member",
		"is_verified": true,
	}
	err = validator.EvaluatePoAConditions(ctx, p, ctxDataSuccess)
	if err != nil {
		t.Errorf("Expected success, got error: %v", err)
	}

	// Case 4: Missing field
	ctxDataMissing := map[string]interface{}{
		"balance": 200,
		// status missing
	}
	err = validator.EvaluatePoAConditions(ctx, p, ctxDataMissing)
	if err == nil {
		t.Error("Expected error for missing field, got nil")
	}
}

func TestConditionalQuotedParsing(t *testing.T) {
	// Direct test of engine splitting logic (since it's internal/private, we test via public interface)
	// We already tested quoted string "gold member" above.
	// Let's test unbalanced quotes via syntax validation.

	engine := NewSimpleConditionalEngine()
	validator := NewEnhancedPoAValidator(WithConditionalEngine(engine), WithWarningCollector(NewWarningCollector()))

	pUnbalanced := &PowerOfAttorney{
		Grantor:    "alice",
		Grantee:    "bob",
		Scope:      []string{"read"},
		ValidUntil: time.Now().Add(1 * time.Hour),
		Restrictions: map[string]string{
			"condition_oops": "name == \"unbalanced",
		},
	}
	_ = validator.Validate(pUnbalanced)
	warnings := validator.GetWarnings()
	found := false
	for _, w := range warnings {
		if w.Code == "invalid_condition_syntax" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected warning for unbalanced quotes, got none")
	}
}

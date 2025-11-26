package gauth_rfc_001

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"testing"
	"time"
)

// TestPropertyScopeSyntaxStability validates that scope syntax validation is consistent
// Property: validateScopeSyntax(s) returns same result for same input
func TestPropertyScopeSyntaxStability(t *testing.T) {
	validator := NewEnhancedPoAValidator()
	testCases := generateRandomScopes(500)

	for _, scope := range testCases {
		// Validate twice to check stability
		err1 := validator.validateScopeSyntax(scope)
		err2 := validator.validateScopeSyntax(scope)

		// Property: Same input produces same result
		if (err1 == nil) != (err2 == nil) {
			t.Errorf("Scope syntax validation unstable for %q: err1=%v, err2=%v", scope, err1, err2)
		}
	}
}

// TestPropertyScopeSyntaxValidInputs validates that syntactically valid scopes pass
// Property: All well-formed scopes pass syntax validation
func TestPropertyScopeSyntaxValidInputs(t *testing.T) {
	validator := NewEnhancedPoAValidator()

	validScopes := []string{
		"*",
		"read:documents",
		"write:files",
		"admin:users",
		"transaction:withdraw",
		"read:*",
		"namespace_123:action_456",
		"ns-123:act-456",
		"UPPERCASE:lowercase",
		"MixedCase:MixedCase",
	}

	for _, scope := range validScopes {
		err := validator.validateScopeSyntax(scope)
		if err != nil {
			t.Errorf("Valid scope %q rejected: %v", scope, err)
		}
	}
}

// TestPropertyScopeSyntaxInvalidInputs validates that syntactically invalid scopes fail
// Property: All malformed scopes fail syntax validation
func TestPropertyScopeSyntaxInvalidInputs(t *testing.T) {
	validator := NewEnhancedPoAValidator()

	invalidScopes := []string{
		"",                // Empty scope
		"read\x00:file",   // Control character (NULL)
		"read\x1f:file",   // Control character (Unit separator)
		":action",         // Missing namespace
		"namespace:",      // Missing action
		"bad-ns@:action",  // Invalid namespace character (@)
		"ns!:action",      // Invalid namespace character (!)
		"ns space:action", // Space in namespace
		"read\x7f:file",   // DEL character
		"ns#:action",      // Invalid character (#)
	}

	for _, scope := range invalidScopes {
		err := validator.validateScopeSyntax(scope)
		if err == nil {
			t.Errorf("Invalid scope %q accepted", scope)
		}
	}
}

// TestPropertyScopeSemanticsWildcardExclusive validates wildcard exclusivity
// Property: Wildcard scope must be alone in scope array
func TestPropertyScopeSemanticsWildcardExclusive(t *testing.T) {
	validator := NewEnhancedPoAValidator()

	// Property: Single wildcard is valid
	err := validator.validateScopeSemantics([]string{"*"})
	if err != nil {
		t.Errorf("Single wildcard rejected: %v", err)
	}

	// Property: Wildcard with other scopes is invalid
	err = validator.validateScopeSemantics([]string{"*", "read:documents"})
	if err == nil {
		t.Error("Wildcard with other scopes accepted")
	}

	err = validator.validateScopeSemantics([]string{"read:documents", "*"})
	if err == nil {
		t.Error("Wildcard with other scopes accepted (reversed order)")
	}

	err = validator.validateScopeSemantics([]string{"*", "read:documents", "write:files"})
	if err == nil {
		t.Error("Wildcard with multiple other scopes accepted")
	}
}

// TestPropertyScopeSemanticsNonEmpty validates scope array non-emptiness
// Property: Scope array must never be empty
func TestPropertyScopeSemanticsNonEmpty(t *testing.T) {
	validator := NewEnhancedPoAValidator()

	// Property: Empty scope array is invalid
	err := validator.validateScopeSemantics([]string{})
	if err == nil {
		t.Error("Empty scope array accepted")
	}

	// Property: Nil scope array is invalid
	err = validator.validateScopeSemantics(nil)
	if err == nil {
		t.Error("Nil scope array accepted")
	}
}

// TestPropertyActionTaxonomyConsistency validates action taxonomy validation stability
// Property: Action taxonomy validation is consistent for same input
func TestPropertyActionTaxonomyConsistency(t *testing.T) {
	validator := NewEnhancedPoAValidator()

	testCases := []struct {
		name        string
		actionClass string
		scopes      []string
	}{
		{"ValidActionClass", "read", []string{"read:documents"}},
		{"InvalidActionClass", "unknown_action", []string{"read:documents"}},
		{"EmptyActionClass", "", []string{"read:documents"}},
		{"ValidScopePrefixes", "read", []string{"read:documents", "write:files", "admin:users"}},
		{"UnknownScopePrefixes", "", []string{"unknown:documents", "invalid:files"}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			poa := &PowerOfAttorney{
				ActionClass: tc.actionClass,
				Scope:       tc.scopes,
			}

			// Clear warnings before each test
			validator.warningCollector.ClearWarnings()

			// Validate twice to check consistency
			err1 := validator.validateActionTaxonomy(poa)
			warnings1Count := len(validator.warningCollector.GetWarnings())

			validator.warningCollector.ClearWarnings()

			err2 := validator.validateActionTaxonomy(poa)
			warnings2Count := len(validator.warningCollector.GetWarnings())

			// Property: Same input produces same result
			if (err1 == nil) != (err2 == nil) {
				t.Errorf("Action taxonomy validation unstable: err1=%v, err2=%v", err1, err2)
			}

			// Check warning consistency
			if warnings1Count != warnings2Count {
				t.Errorf("Warning count inconsistent: %d vs %d", warnings1Count, warnings2Count)
			}
		})
	}
}

// TestPropertyTemporalConstraintsMonotonicity validates temporal constraint logic
// Property: Temporal constraints generate appropriate warnings for edge cases
func TestPropertyTemporalConstraintsMonotonicity(t *testing.T) {
	validator := NewEnhancedPoAValidator()
	ctx := context.Background()
	now := time.Now()

	testCases := []struct {
		name                string
		validFrom           time.Time
		validUntil          time.Time
		expectedWarningCode string
	}{
		{"FutureValidPeriod", now.Add(time.Hour), now.Add(2 * time.Hour), ""},
		{"CurrentValidPeriod", now, now.Add(time.Hour), ""},
		{"ShortDuration", now, now.Add(30 * time.Minute), "very_short_duration"},
		{"PastValidFrom", now.Add(-48 * time.Hour), now.Add(time.Hour), "past_valid_from"},
		{"LongDuration", now, now.Add(400 * 24 * time.Hour), "long_duration"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			poa := &PowerOfAttorney{
				Grantor:    "alice@example.com",
				Grantee:    "bob@example.com",
				ValidFrom:  tc.validFrom,
				ValidUntil: tc.validUntil,
				Scope:      []string{"read:documents"},
			}

			validator.warningCollector.ClearWarnings()

			// Use validateEnhancedSemantics for long_duration check
			err := validator.validateEnhancedSemantics(ctx, poa)
			if err != nil {
				t.Errorf("Enhanced semantic validation error: %v", err)
			}

			warnings := validator.warningCollector.GetWarnings()

			if tc.expectedWarningCode == "" {
				// No specific warnings expected (may have other warnings)
				// Just check no errors occurred
			} else {
				// Check for expected warning
				foundExpected := false
				for _, warning := range warnings {
					if warning.Code == tc.expectedWarningCode {
						foundExpected = true
						break
					}
				}
				if !foundExpected {
					t.Errorf("Expected warning %q not found, got %d warnings", tc.expectedWarningCode, len(warnings))
				}
			}
		})
	}
}

// TestPropertyRestrictionSemanticsConsistency validates restriction validation
// Property: Restriction validation is consistent and idempotent
func TestPropertyRestrictionSemanticsConsistency(t *testing.T) {
	validator := NewEnhancedPoAValidator()

	testCases := []struct {
		name         string
		restrictions map[string]string
	}{
		{"EmptyRestrictions", map[string]string{}},
		{"MaxAmountRestriction", map[string]string{"max_amount": "1000.0"}},
		{"IPWhitelistRestriction", map[string]string{"ip_whitelist": "192.168.1.1"}},
		{"MultipleRestrictions", map[string]string{
			"max_amount":   "500.0",
			"daily_limit":  "5000.0",
			"ip_whitelist": "10.0.0.1,10.0.0.2",
		}},
		{"InvalidAmountType", map[string]string{"max_amount": "not_a_number"}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			poa := &PowerOfAttorney{
				Restrictions: tc.restrictions,
				Scope:        []string{"transaction:withdraw"},
				ValidFrom:    time.Now(),
				ValidUntil:   time.Now().Add(time.Hour),
			}

			// Validate twice to check consistency
			err1 := validator.validateRestrictionSemantics(poa)
			err2 := validator.validateRestrictionSemantics(poa)

			// Property: Same input produces same result
			if (err1 == nil) != (err2 == nil) {
				t.Errorf("Restriction validation unstable: err1=%v, err2=%v", err1, err2)
			}
		})
	}
}

// TestPropertyValidationIdempotence validates that validation is idempotent
// Property: Validating same PoA multiple times produces same result
func TestPropertyValidationIdempotence(t *testing.T) {
	validator := NewEnhancedPoAValidator()
	ctx := context.Background()

	poa := &PowerOfAttorney{
		Grantor:     "alice@example.com",
		Grantee:     "bob@example.com",
		Scope:       []string{"read:documents", "write:files"},
		ValidFrom:   time.Now(),
		ValidUntil:  time.Now().Add(24 * time.Hour),
		ActionClass: "read",
	}

	// Validate 10 times
	var results []bool
	for i := 0; i < 10; i++ {
		err := validator.ValidateWithContext(ctx, poa)
		results = append(results, err == nil)
	}

	// Property: All results should be identical
	firstResult := results[0]
	for i, result := range results {
		if result != firstResult {
			t.Errorf("Validation result changed at iteration %d: expected %v, got %v", i, firstResult, result)
		}
	}
}

// TestPropertyEnhancedSemanticsComposability validates validation composability
// Property: Individual semantic checks can be composed without side effects
func TestPropertyEnhancedSemanticsComposability(t *testing.T) {
	validator := NewEnhancedPoAValidator()
	ctx := context.Background()

	poa := &PowerOfAttorney{
		Grantor:      "alice@example.com",
		Grantee:      "bob@example.com",
		Scope:        []string{"read:documents"},
		ValidFrom:    time.Now(),
		ValidUntil:   time.Now().Add(time.Hour),
		ActionClass:  "read",
		Restrictions: map[string]string{"max_amount": "100.0"},
	}

	// Run individual validations
	err1 := validator.validateRFC0115Semantics(ctx, poa)
	err2 := validator.validateCrossFieldConsistency(ctx, poa)
	err3 := validator.validateRestrictionSemantics(poa)

	// Run composed validation
	errComposed := validator.validateEnhancedSemantics(ctx, poa)

	// Property: Composed validation should catch all individual errors
	if err1 != nil && errComposed == nil {
		t.Error("Composed validation missed RFC0115 semantic error")
	}
	if err2 != nil && errComposed == nil {
		t.Error("Composed validation missed cross-field consistency error")
	}
	if err3 != nil && errComposed == nil {
		t.Error("Composed validation missed restriction semantic error")
	}
}

// TestPropertyWarningCollectionNonBlockingProperty validates warning collection doesn't block validation
// Property: Warnings are collected without blocking valid PoA validation
func TestPropertyWarningCollectionNonBlockingProperty(t *testing.T) {
	validator := NewEnhancedPoAValidator()
	ctx := context.Background()

	// PoA with conditions that generate warnings but should still be valid
	poa := &PowerOfAttorney{
		Grantor:     "alice@example.com",
		Grantee:     "bob@example.com",
		Scope:       []string{"read:documents", "read:documents"}, // Duplicate scope (warning)
		ValidFrom:   time.Now().Add(-48 * time.Hour),              // Past valid_from (warning)
		ValidUntil:  time.Now().Add(400 * 24 * time.Hour),         // Long duration (warning)
		ActionClass: "read",
	}

	result := validator.ValidateWithResult(ctx, poa)

	// Property: PoA should be valid despite warnings
	if !result.Valid {
		t.Errorf("PoA with warnings rejected: %v", result.Error)
	}

	// Property: Warnings should be collected
	if len(result.Warnings) == 0 {
		t.Error("Expected warnings but got none")
	}

	// Check for expected warning codes
	expectedWarnings := map[string]bool{
		"duplicate_scope": false,
		"past_valid_from": false,
		"long_duration":   false,
	}

	for _, warning := range result.Warnings {
		if _, exists := expectedWarnings[warning.Code]; exists {
			expectedWarnings[warning.Code] = true
		}
	}

	for code, found := range expectedWarnings {
		if !found {
			t.Errorf("Expected warning %q not found", code)
		}
	}
}

// TestPropertyScopeSubsumptionDetection validates scope subsumption warning
// Property: Wildcard scopes should trigger subsumption warnings for specific scopes
func TestPropertyScopeSubsumptionDetection(t *testing.T) {
	validator := NewEnhancedPoAValidator()

	testCases := []struct {
		name                string
		scopes              []string
		expectedSubsumption bool
	}{
		{"NoSubsumption", []string{"read:documents", "write:files"}, false},
		{"WildcardSubsumes", []string{"read:*", "read:documents"}, true},
		{"WildcardSubsumesMultiple", []string{"read:*", "read:documents", "read:files"}, true},
		{"NoWildcard", []string{"read:documents", "read:files"}, false},
		{"OnlyWildcard", []string{"read:*"}, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			validator.warningCollector.ClearWarnings()
			err := validator.validateScopeSemantics(tc.scopes)
			if err != nil {
				t.Fatalf("Unexpected validation error: %v", err)
			}

			warnings := validator.warningCollector.GetWarnings()
			foundSubsumption := false
			for _, warning := range warnings {
				if warning.Code == "scope_subsumption" {
					foundSubsumption = true
					break
				}
			}

			if tc.expectedSubsumption != foundSubsumption {
				t.Errorf("Subsumption detection mismatch: expected=%v, got=%v", tc.expectedSubsumption, foundSubsumption)
			}
		})
	}
}

// TestPropertyAdministrativeScopeDetection validates admin scope warnings
// Property: Administrative scopes trigger elevated approval warnings
func TestPropertyAdministrativeScopeDetection(t *testing.T) {
	validator := NewEnhancedPoAValidator()
	ctx := context.Background()

	testCases := []struct {
		name               string
		scopes             []string
		expectAdminWarning bool
	}{
		{"NoAdminScope", []string{"read:documents"}, false},
		{"AdminScope", []string{"admin:users"}, true},
		{"RootScope", []string{"root:system"}, true},
		{"MixedScopes", []string{"read:documents", "admin:users"}, true},
		{"AdminInNamespace", []string{"admin_read:documents"}, true}, // Contains "admin"
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			poa := &PowerOfAttorney{
				Grantor:    "alice@example.com",
				Grantee:    "bob@example.com",
				Scope:      tc.scopes,
				ValidFrom:  time.Now(),
				ValidUntil: time.Now().Add(time.Hour),
			}

			validator.warningCollector.ClearWarnings()
			_ = validator.validateEnhancedSemantics(ctx, poa)

			warnings := validator.warningCollector.GetWarnings()
			foundAdminWarning := false
			for _, warning := range warnings {
				if warning.Code == "administrative_scope" {
					foundAdminWarning = true
					break
				}
			}

			if tc.expectAdminWarning != foundAdminWarning {
				t.Errorf("Admin scope detection mismatch: expected=%v, got=%v", tc.expectAdminWarning, foundAdminWarning)
			}
		})
	}
}

// Helper functions

// generateRandomScopes generates random scope strings for property testing
func generateRandomScopes(count int) []string {
	scopes := make([]string, 0, count)

	// Valid scopes
	validPrefixes := []string{"read", "write", "admin", "transaction", "delete"}
	validResources := []string{"documents", "files", "users", "accounts", "data"}

	for i := 0; i < count/2; i++ {
		//nolint:gosec // G404: weak random acceptable for property-based test generation
		prefix := validPrefixes[rand.Intn(len(validPrefixes))]
		//nolint:gosec // G404: weak random acceptable for property-based test generation
		resource := validResources[rand.Intn(len(validResources))]
		scopes = append(scopes, fmt.Sprintf("%s:%s", prefix, resource))
	}

	// Edge cases
	scopes = append(scopes,
		"*",
		"",
		"no_colon",
		":missing_namespace",
		"missing_action:",
		strings.Repeat("a", 100)+":"+strings.Repeat("b", 100), // Long scope
		"valid-namespace:valid-action",
		"valid_namespace:valid_action",
		"CAPS:LOCK",
	)

	// Invalid scopes with control characters
	for i := 0; i < count/4; i++ {
		//nolint:gosec // G404: weak random acceptable for property-based test generation
		invalidChar := byte(rand.Intn(32)) // Control characters
		scopes = append(scopes, fmt.Sprintf("read%cfile", invalidChar))
	}

	return scopes
}

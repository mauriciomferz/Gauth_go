package gauth_rfc_001

import (
	"testing"
	"time"
)

// TestEnhancedJTIValidation tests the enhanced JTI validation function.
func TestEnhancedJTIValidation(t *testing.T) {
	nowFn := func() time.Time { return time.Now() }

	testCases := []struct {
		name           string
		jti            string
		expectErr      bool
		expectedReason string
	}{
		{
			name:      "valid_uuid_v4",
			jti:       "550e8400-e29b-41d4-a716-446655440000",
			expectErr: false,
		},
		{
			name:           "too_short",
			jti:            "550e8400-e29b-41d4-a716",
			expectErr:      true,
			expectedReason: "jti_length_invalid",
		},
		{
			name:           "too_long",
			jti:            "550e8400-e29b-41d4-a716-446655440000-extra",
			expectErr:      true,
			expectedReason: "jti_length_invalid",
		},
		{
			name:           "empty_string",
			jti:            "",
			expectErr:      true,
			expectedReason: "jti_length_invalid",
		},
		{
			name:           "wrong_version",
			jti:            "550e8400-e29b-31d4-a716-446655440000", // UUID v3
			expectErr:      true,
			expectedReason: "jti_format_invalid",
		},
		{
			name:           "wrong_variant",
			jti:            "550e8400-e29b-41d4-c716-446655440000", // variant 'c' instead of 8/9/a/b
			expectErr:      true,
			expectedReason: "jti_format_invalid",
		},
		{
			name:           "invalid_chars",
			jti:            "550e8400-e29b-41d4-a716-44665544000z", // 'z' is not hex
			expectErr:      true,
			expectedReason: "jti_format_invalid",
		},
		{
			name:           "missing_hyphens",
			jti:            "550e8400e29b41d4a716446655440000", // 32 chars (no hyphens)
			expectErr:      true,
			expectedReason: "jti_length_invalid", // Fails length check (32 != 36) before format check
		},
		{
			name:      "uppercase_hex",
			jti:       "550E8400-E29B-41D4-A716-446655440000",
			expectErr: false, // Should accept uppercase hex
		},
		{
			name:      "mixed_case_hex",
			jti:       "550e8400-E29B-41d4-A716-446655440000",
			expectErr: false, // Should accept mixed case
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err, reason := validateJTIEnhanced(tc.jti, nowFn)

			if tc.expectErr {
				if err == nil {
					t.Fatalf("Expected error but got nil")
				}
				if reason != tc.expectedReason {
					t.Fatalf("Expected reason %q but got %q", tc.expectedReason, reason)
				}
			} else {
				if err != nil {
					t.Fatalf("Expected no error but got: %v (reason: %s)", err, reason)
				}
				if reason != "" {
					t.Fatalf("Expected empty reason but got: %s", reason)
				}
			}
		})
	}
}

// TestJTIValidationError tests the error type formatting.
func TestJTIValidationError(t *testing.T) {
	err := &JTIValidationError{
		Reason: "format_invalid",
		JTI:    "bad-jti",
	}

	expected := "JTI validation failed: format_invalid (jti=bad-jti)"
	if err.Error() != expected {
		t.Fatalf("Expected error message %q but got %q", expected, err.Error())
	}
}

// TestIsUUIDv4 tests the existing UUID v4 validation function.
func TestIsUUIDv4(t *testing.T) {
	validCases := []string{
		"550e8400-e29b-41d4-a716-446655440000",
		"550E8400-E29B-41D4-A716-446655440000",
		"00000000-0000-4000-8000-000000000000",
		"ffffffff-ffff-4fff-bfff-ffffffffffff",
	}

	for _, jti := range validCases {
		if !isUUIDv4(jti) {
			t.Errorf("Expected %q to be valid UUID v4", jti)
		}
	}

	invalidCases := []string{
		"",
		"not-a-uuid",
		"550e8400-e29b-31d4-a716-446655440000", // version 3
		"550e8400-e29b-51d4-a716-446655440000", // version 5
		"550e8400e29b41d4a716446655440000",     // no hyphens
		"550e8400-e29b-41d4-c716-446655440000", // wrong variant
	}

	for _, jti := range invalidCases {
		if isUUIDv4(jti) {
			t.Errorf("Expected %q to be invalid UUID v4", jti)
		}
	}
}

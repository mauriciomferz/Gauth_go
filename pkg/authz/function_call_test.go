package authz

import (
	"testing"
)

// TestFunctionCallsInExpressions tests that function calls work in policy expressions.
func TestFunctionCallsInExpressions(t *testing.T) {
	testCases := []struct {
		name      string
		expr      string
		ctx       Request
		expected  bool
		expectErr bool
	}{
		{
			name:     "len_function",
			expr:     "len(subject) > 3",
			ctx:      Request{Subject: "alice"},
			expected: true,
		},
		{
			name:     "upper_function",
			expr:     `upper(subject) == "ADMIN"`,
			ctx:      Request{Subject: "admin"},
			expected: true,
		},
		{
			name:     "lower_function",
			expr:     `lower(subject) == "admin"`,
			ctx:      Request{Subject: "ADMIN"},
			expected: true,
		},
		{
			name:     "startsWith_function",
			expr:     `startsWith(subject, "ad")`,
			ctx:      Request{Subject: "admin"},
			expected: true,
		},
		{
			name:     "endsWith_function",
			expr:     `endsWith(resource, ".txt")`,
			ctx:      Request{Resource: "file.txt"},
			expected: true,
		},
		{
			name:     "contains_function",
			expr:     `contains(resource, "document")`,
			ctx:      Request{Resource: "/path/to/document.pdf"},
			expected: true,
		},
		{
			name:     "regex_match_function",
			expr:     `regex_match(subject, "^[a-z]+$")`,
			ctx:      Request{Subject: "alice"},
			expected: true,
		},
		{
			name:     "multiple_functions",
			expr:     `upper(subject) == "ALICE" && len(resource) > 5`,
			ctx:      Request{Subject: "alice", Resource: "/files"},
			expected: true,
		},
		{
			name:     "nested_functions",
			expr:     `len(upper(subject)) == 5`,
			ctx:      Request{Subject: "alice"},
			expected: true,
		},
		{
			name:     "function_with_comparison",
			expr:     `len(subject) >= 3 && len(subject) <= 10`,
			ctx:      Request{Subject: "bob"},
			expected: true,
		},
		{
			name:      "unknown_function",
			expr:      `unknown("test")`,
			ctx:       Request{Subject: "alice"},
			expectErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := EvaluateExpression(tc.expr, tc.ctx, nil)

			if tc.expectErr {
				if err == nil {
					t.Fatal("Expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if result != tc.expected {
				t.Fatalf("Expected %v, got %v", tc.expected, result)
			}
		})
	}
}

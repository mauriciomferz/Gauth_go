package authz

import (
	"testing"
)

// TestEvaluateExpression_Basic tests basic expression evaluation
func TestEvaluateExpression_Basic(t *testing.T) {
	tests := []struct {
		name    string
		expr    string
		req     Request
		want    bool
		wantErr bool
	}{
		{
			name: "simple_boolean_true",
			expr: "true",
			req:  Request{},
			want: true,
		},
		{
			name: "simple_boolean_false",
			expr: "false",
			req:  Request{},
			want: false,
		},
		{
			name: "subject_equality",
			expr: `subject == "alice"`,
			req:  Request{Subject: "alice"},
			want: true,
		},
		{
			name: "subject_inequality",
			expr: `subject != "alice"`,
			req:  Request{Subject: "bob"},
			want: true,
		},
		{
			name: "resource_equality",
			expr: `resource == "vault"`,
			req:  Request{Resource: "vault"},
			want: true,
		},
		{
			name: "action_equality",
			expr: `action == "read"`,
			req:  Request{Action: "read"},
			want: true,
		},
		{
			name: "context_access",
			expr: `role == "admin"`,
			req:  Request{Context: map[string]string{"role": "admin"}},
			want: true,
		},
		{
			name: "numeric_greater_than",
			expr: `age > 18`,
			req:  Request{Context: map[string]string{"age": "25"}},
			want: true,
		},
		{
			name: "numeric_less_than",
			expr: `age < 18`,
			req:  Request{Context: map[string]string{"age": "15"}},
			want: true,
		},
		{
			name: "numeric_greater_equal",
			expr: `age >= 18`,
			req:  Request{Context: map[string]string{"age": "18"}},
			want: true,
		},
		{
			name: "numeric_less_equal",
			expr: `age <= 65`,
			req:  Request{Context: map[string]string{"age": "65"}},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := EvaluateExpression(tt.expr, tt.req, nil)
			if (err != nil) != tt.wantErr {
				t.Errorf("EvaluateExpression() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("EvaluateExpression() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestEvaluateExpression_Logical tests logical operators
func TestEvaluateExpression_Logical(t *testing.T) {
	tests := []struct {
		name    string
		expr    string
		req     Request
		want    bool
		wantErr bool
	}{
		{
			name: "and_both_true",
			expr: "true && true",
			req:  Request{},
			want: true,
		},
		{
			name: "and_first_false",
			expr: "false && true",
			req:  Request{},
			want: false,
		},
		{
			name: "and_second_false",
			expr: "true && false",
			req:  Request{},
			want: false,
		},
		{
			name: "or_both_false",
			expr: "false || false",
			req:  Request{},
			want: false,
		},
		{
			name: "or_first_true",
			expr: "true || false",
			req:  Request{},
			want: true,
		},
		{
			name: "or_second_true",
			expr: "false || true",
			req:  Request{},
			want: true,
		},
		{
			name: "not_true",
			expr: "!true",
			req:  Request{},
			want: false,
		},
		{
			name: "not_false",
			expr: "!false",
			req:  Request{},
			want: true,
		},
		{
			name: "complex_and_or",
			expr: `subject == "alice" && (resource == "vault" || action == "read")`,
			req:  Request{Subject: "alice", Resource: "vault", Action: "write"},
			want: true,
		},
		{
			name: "complex_or_and",
			expr: `subject == "alice" || resource == "vault" && action == "read"`,
			req:  Request{Subject: "bob", Resource: "vault", Action: "read"},
			want: true,
		},
		{
			name: "not_with_parentheses",
			expr: `!(subject == "alice")`,
			req:  Request{Subject: "bob"},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := EvaluateExpression(tt.expr, tt.req, nil)
			if (err != nil) != tt.wantErr {
				t.Errorf("EvaluateExpression() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("EvaluateExpression() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestEvaluateExpression_In tests 'in' operator
func TestEvaluateExpression_In(t *testing.T) {
	tests := []struct {
		name    string
		expr    string
		req     Request
		want    bool
		wantErr bool
	}{
		{
			name: "in_list_string",
			expr: `subject in ["alice", "bob", "charlie"]`,
			req:  Request{Subject: "alice"},
			want: true,
		},
		{
			name: "not_in_list",
			expr: `subject in ["alice", "bob"]`,
			req:  Request{Subject: "dave"},
			want: false,
		},
		{
			name: "in_list_number",
			expr: `age in [18, 21, 25]`,
			req:  Request{Context: map[string]string{"age": "21"}},
			want: true,
		},
		{
			name: "in_empty_list",
			expr: `subject in []`,
			req:  Request{Subject: "alice"},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := EvaluateExpression(tt.expr, tt.req, nil)
			if (err != nil) != tt.wantErr {
				t.Errorf("EvaluateExpression() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("EvaluateExpression() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestEvaluateExpression_Errors tests error cases
func TestEvaluateExpression_Errors(t *testing.T) {
	tests := []struct {
		name    string
		expr    string
		req     Request
		wantErr bool
	}{
		{
			name:    "empty_expression",
			expr:    "",
			req:     Request{},
			wantErr: true, // Empty expression errors
		},
		{
			name:    "invalid_syntax_unclosed_paren",
			expr:    "(subject == \"alice\"",
			req:     Request{Subject: "alice"},
			wantErr: true,
		},
		{
			name:    "invalid_syntax_missing_operand",
			expr:    "subject ==",
			req:     Request{Subject: "alice"},
			wantErr: true,
		},
		{
			name:    "undefined_identifier",
			expr:    "undefined_var == 123",
			req:     Request{},
			wantErr: false, // undefined vars return empty/nil, not error
		},
		{
			name:    "invalid_comparison",
			expr:    `"string" > 123`,
			req:     Request{},
			wantErr: true, // type mismatch
		},
		{
			name:    "unclosed_string",
			expr:    `"unclosed string`,
			req:     Request{},
			wantErr: true,
		},
		{
			name:    "unclosed_bracket",
			expr:    `subject in ["alice", "bob"`,
			req:     Request{Subject: "alice"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := EvaluateExpression(tt.expr, tt.req, nil)
			if (err != nil) != tt.wantErr {
				t.Errorf("EvaluateExpression() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestEvaluateExpression_Limits tests resource limits
func TestEvaluateExpression_Limits(t *testing.T) {
	t.Run("max_tokens_exceeded", func(t *testing.T) {
		// Very long expression
		expr := "true"
		for i := 0; i < 100; i++ {
			expr += " && true"
		}
		limits := &ExprLimits{MaxTokens: 10, MaxDepth: 32, MaxOps: 1024, MaxIdentifierLength: 64, MaxLiteralLength: 256}
		_, err := EvaluateExpression(expr, Request{}, limits)
		if err == nil {
			t.Error("expected error for exceeding max tokens")
		}
	})

	t.Run("max_depth_exceeded", func(t *testing.T) {
		// Deeply nested expression with AND/OR operators
		expr := "true && true"
		for i := 0; i < 15; i++ {
			expr = "(" + expr + ") && (" + expr + ")"
		}
		limits := &ExprLimits{MaxTokens: 100000, MaxDepth: 10, MaxOps: 10000, MaxIdentifierLength: 64, MaxLiteralLength: 256}
		_, err := EvaluateExpression(expr, Request{}, limits)
		if err == nil {
			t.Error("expected error for exceeding max depth")
		}
	})

	t.Run("max_ops_exceeded", func(t *testing.T) {
		// Many operations
		expr := "true"
		for i := 0; i < 600; i++ {
			expr += " && true"
		}
		limits := &ExprLimits{MaxTokens: 2000, MaxDepth: 100, MaxOps: 10, MaxIdentifierLength: 64, MaxLiteralLength: 256}
		_, err := EvaluateExpression(expr, Request{}, limits)
		if err == nil {
			t.Error("expected error for exceeding max ops")
		}
	})

	t.Run("max_identifier_length", func(t *testing.T) {
		// Very long identifier
		longIdent := ""
		for i := 0; i < 100; i++ {
			longIdent += "a"
		}
		expr := longIdent + " == 1"
		limits := &ExprLimits{MaxTokens: 256, MaxDepth: 32, MaxOps: 1024, MaxIdentifierLength: 10, MaxLiteralLength: 256}
		_, err := EvaluateExpression(expr, Request{Context: map[string]string{longIdent: "1"}}, limits)
		if err == nil {
			t.Error("expected error for exceeding max identifier length")
		}
	})

	t.Run("max_literal_length", func(t *testing.T) {
		// Very long string literal
		longString := "\""
		for i := 0; i < 300; i++ {
			longString += "a"
		}
		longString += "\""
		expr := "subject == " + longString
		limits := &ExprLimits{MaxTokens: 256, MaxDepth: 32, MaxOps: 1024, MaxIdentifierLength: 64, MaxLiteralLength: 100}
		_, err := EvaluateExpression(expr, Request{Subject: "test"}, limits)
		if err == nil {
			t.Error("expected error for exceeding max literal length")
		}
	})
}

// TestEvaluateExpression_Parentheses tests parentheses handling
func TestEvaluateExpression_Parentheses(t *testing.T) {
	tests := []struct {
		name string
		expr string
		req  Request
		want bool
	}{
		{
			name: "simple_parentheses",
			expr: "(true)",
			req:  Request{},
			want: true,
		},
		{
			name: "nested_parentheses",
			expr: "((true))",
			req:  Request{},
			want: true,
		},
		{
			name: "precedence_with_parentheses",
			expr: "(true || false) && false",
			req:  Request{},
			want: false,
		},
		{
			name: "precedence_without_parentheses",
			expr: "true || false && false",
			req:  Request{},
			want: true, // OR has lower precedence, so true || (false && false) = true || false = true
		},
		{
			name: "complex_precedence",
			expr: "(false || true) && (true || false)",
			req:  Request{},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := EvaluateExpression(tt.expr, tt.req, nil)
			if err != nil {
				t.Errorf("EvaluateExpression() error = %v", err)
				return
			}
			if got != tt.want {
				t.Errorf("EvaluateExpression() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestEvaluateExpression_ContextAccess tests context variable access
func TestEvaluateExpression_ContextAccess(t *testing.T) {
	tests := []struct {
		name string
		expr string
		req  Request
		want bool
	}{
		{
			name: "direct_context_key",
			expr: `role == "admin"`,
			req:  Request{Context: map[string]string{"role": "admin"}},
			want: true,
		},
		{
			name: "department_context",
			expr: `department == "engineering"`,
			req:  Request{Context: map[string]string{"department": "engineering"}},
			want: true,
		},
		{
			name: "level_context",
			expr: `level > 5`,
			req:  Request{Context: map[string]string{"level": "7"}},
			want: true,
		},
		{
			name: "string_context",
			expr: `status == "active"`,
			req:  Request{Context: map[string]string{"status": "active"}},
			want: true,
		},
		{
			name: "boolean_context",
			expr: `enabled == true`,
			req:  Request{Context: map[string]string{"enabled": "true"}},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := EvaluateExpression(tt.expr, tt.req, nil)
			if err != nil {
				t.Errorf("EvaluateExpression() error = %v", err)
				return
			}
			if got != tt.want {
				t.Errorf("EvaluateExpression() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestDebugLex tests the debug lexer function
func TestDebugLex(t *testing.T) {
	tests := []struct {
		name          string
		expr          string
		wantTokens    int
		wantFirstType int
	}{
		{
			name:          "simple_boolean",
			expr:          "true",
			wantTokens:    2, // true, EOF
			wantFirstType: tokBool,
		},
		{
			name:          "identifier",
			expr:          "subject",
			wantTokens:    2, // subject, EOF
			wantFirstType: tokIdent,
		},
		{
			name:          "string_literal",
			expr:          `"alice"`,
			wantTokens:    2, // "alice", EOF
			wantFirstType: tokString,
		},
		{
			name:          "number_literal",
			expr:          "123",
			wantTokens:    2, // 123, EOF
			wantFirstType: tokNumber,
		},
		{
			name:       "comparison",
			expr:       "subject == \"alice\"",
			wantTokens: 4, // subject, ==, "alice", EOF
		},
		{
			name:       "logical_and",
			expr:       "true && false",
			wantTokens: 4, // true, &&, false, EOF
		},
		{
			name:       "logical_or",
			expr:       "true || false",
			wantTokens: 4, // true, ||, false, EOF
		},
		{
			name:       "not_operator",
			expr:       "!true",
			wantTokens: 3, // !, true, EOF
		},
		{
			name:       "parentheses",
			expr:       "(true)",
			wantTokens: 4, // (, true, ), EOF
		},
		{
			name:       "in_operator",
			expr:       "subject in [\"alice\"]",
			wantTokens: 6, // subject, in, [, "alice", ], EOF
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens := DebugLex(tt.expr)
			if len(tokens) != tt.wantTokens {
				t.Errorf("DebugLex() returned %d tokens, want %d", len(tokens), tt.wantTokens)
			}
			if tt.wantFirstType != 0 && len(tokens) > 0 {
				if tokens[0].Type != tt.wantFirstType {
					t.Errorf("DebugLex() first token type = %d, want %d", tokens[0].Type, tt.wantFirstType)
				}
			}
		})
	}
}

// TestEvaluateExpression_EdgeCases tests edge cases
func TestEvaluateExpression_EdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		expr    string
		req     Request
		want    bool
		wantErr bool
	}{
		{
			name:    "whitespace_only",
			expr:    "   \n\t  ",
			req:     Request{},
			want:    false,
			wantErr: true, // Empty expression should error
		},
		{
			name: "multiple_comparisons",
			expr: `subject == "alice" && resource == "vault" && action == "read"`,
			req:  Request{Subject: "alice", Resource: "vault", Action: "read"},
			want: true,
		},
		{
			name: "chained_or",
			expr: `action == "read" || action == "write" || action == "delete"`,
			req:  Request{Action: "write"},
			want: true,
		},
		{
			name: "mixed_types_string_number",
			expr: `value == "123"`,
			req:  Request{Context: map[string]string{"value": "123"}},
			want: true,
		},
		{
			name: "zero_number",
			expr: `count == 0`,
			req:  Request{Context: map[string]string{"count": "0"}},
			want: true,
		},
		{
			name: "empty_string",
			expr: `subject == ""`,
			req:  Request{Subject: ""},
			want: true,
		},
		{
			name: "truthy_non_empty_string",
			expr: `"non-empty"`,
			req:  Request{},
			want: true, // Non-empty string is truthy
		},
		{
			name: "falsy_empty_string",
			expr: `""`,
			req:  Request{},
			want: false, // Empty string is falsy
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := EvaluateExpression(tt.expr, tt.req, nil)
			if (err != nil) != tt.wantErr {
				t.Errorf("EvaluateExpression() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("EvaluateExpression() = %v, want %v", got, tt.want)
			}
		})
	}
}

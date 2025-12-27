package expr

import (
	"testing"
)

// TestFunctionRegistry tests the basic registry functionality.
func TestFunctionRegistry(t *testing.T) {
	t.Run("Register_and_Get", func(t *testing.T) {
		reg := NewRegistry()

		// Register a simple function
		testFn := func(args []interface{}) (interface{}, error) {
			return "test", nil
		}

		err := reg.Register("test_func", testFn)
		if err != nil {
			t.Fatalf("Failed to register function: %v", err)
		}

		// Retrieve the function
		fn, ok := reg.Get("test_func")
		if !ok {
			t.Fatal("Function not found")
		}
		if fn == nil {
			t.Fatal("Retrieved nil function")
		}
	})

	t.Run("Register_duplicate", func(t *testing.T) {
		reg := NewRegistry()

		testFn := func(args []interface{}) (interface{}, error) {
			return "test", nil
		}

		_ = reg.Register("dup", testFn)
		err := reg.Register("dup", testFn)
		if err == nil {
			t.Fatal("Expected error when registering duplicate function")
		}
	})

	t.Run("Register_invalid", func(t *testing.T) {
		reg := NewRegistry()

		// Empty name
		err := reg.Register("", nil)
		if err == nil {
			t.Fatal("Expected error for empty function name")
		}

		// Nil function
		err = reg.Register("valid_name", nil)
		if err == nil {
			t.Fatal("Expected error for nil function")
		}
	})

	t.Run("Get_nonexistent", func(t *testing.T) {
		reg := NewRegistry()

		_, ok := reg.Get("nonexistent")
		if ok {
			t.Fatal("Expected function not to be found")
		}
	})

	t.Run("List", func(t *testing.T) {
		reg := NewRegistry()

		testFn := func(args []interface{}) (interface{}, error) {
			return "test", nil
		}

		_ = reg.Register("func1", testFn)
		_ = reg.Register("func2", testFn)
		_ = reg.Register("func3", testFn)

		names := reg.List()
		if len(names) != 3 {
			t.Fatalf("Expected 3 functions, got %d", len(names))
		}
	})
}

// TestBuiltinLen tests the len() function.
func TestBuiltinLen(t *testing.T) {
	testCases := []struct {
		name      string
		args      []interface{}
		expected  float64
		expectErr bool
	}{
		{
			name:     "empty_string",
			args:     []interface{}{""},
			expected: 0,
		},
		{
			name:     "simple_string",
			args:     []interface{}{"hello"},
			expected: 5,
		},
		{
			name:     "unicode_string",
			args:     []interface{}{"héllo"}, // 'é' is 2 bytes in UTF-8
			expected: 6,                      // byte count, not rune count
		},
		{
			name:      "wrong_arg_count",
			args:      []interface{}{},
			expectErr: true,
		},
		{
			name:      "wrong_type",
			args:      []interface{}{123},
			expectErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := builtinLen(tc.args)

			if tc.expectErr {
				if err == nil {
					t.Fatal("Expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if result.(float64) != tc.expected {
				t.Fatalf("Expected %v, got %v", tc.expected, result)
			}
		})
	}
}

// TestBuiltinUpper tests the upper() function.
func TestBuiltinUpper(t *testing.T) {
	testCases := []struct {
		name      string
		args      []interface{}
		expected  string
		expectErr bool
	}{
		{
			name:     "lowercase",
			args:     []interface{}{"hello"},
			expected: "HELLO",
		},
		{
			name:     "mixed_case",
			args:     []interface{}{"Hello World"},
			expected: "HELLO WORLD",
		},
		{
			name:     "already_uppercase",
			args:     []interface{}{"HELLO"},
			expected: "HELLO",
		},
		{
			name:     "empty_string",
			args:     []interface{}{""},
			expected: "",
		},
		{
			name:      "wrong_arg_count",
			args:      []interface{}{},
			expectErr: true,
		},
		{
			name:      "wrong_type",
			args:      []interface{}{123},
			expectErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := builtinUpper(tc.args)

			if tc.expectErr {
				if err == nil {
					t.Fatal("Expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if result.(string) != tc.expected {
				t.Fatalf("Expected %v, got %v", tc.expected, result)
			}
		})
	}
}

// TestBuiltinLower tests the lower() function.
func TestBuiltinLower(t *testing.T) {
	testCases := []struct {
		name      string
		args      []interface{}
		expected  string
		expectErr bool
	}{
		{
			name:     "uppercase",
			args:     []interface{}{"HELLO"},
			expected: "hello",
		},
		{
			name:     "mixed_case",
			args:     []interface{}{"Hello World"},
			expected: "hello world",
		},
		{
			name:     "already_lowercase",
			args:     []interface{}{"hello"},
			expected: "hello",
		},
		{
			name:      "wrong_type",
			args:      []interface{}{123},
			expectErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := builtinLower(tc.args)

			if tc.expectErr {
				if err == nil {
					t.Fatal("Expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if result.(string) != tc.expected {
				t.Fatalf("Expected %v, got %v", tc.expected, result)
			}
		})
	}
}

// TestBuiltinStartsWith tests the startsWith() function.
func TestBuiltinStartsWith(t *testing.T) {
	testCases := []struct {
		name      string
		args      []interface{}
		expected  bool
		expectErr bool
	}{
		{
			name:     "match",
			args:     []interface{}{"hello world", "hello"},
			expected: true,
		},
		{
			name:     "no_match",
			args:     []interface{}{"hello world", "world"},
			expected: false,
		},
		{
			name:     "empty_prefix",
			args:     []interface{}{"hello", ""},
			expected: true,
		},
		{
			name:      "wrong_arg_count",
			args:      []interface{}{"hello"},
			expectErr: true,
		},
		{
			name:      "wrong_type_first",
			args:      []interface{}{123, "hello"},
			expectErr: true,
		},
		{
			name:      "wrong_type_second",
			args:      []interface{}{"hello", 123},
			expectErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := builtinStartsWith(tc.args)

			if tc.expectErr {
				if err == nil {
					t.Fatal("Expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if result.(bool) != tc.expected {
				t.Fatalf("Expected %v, got %v", tc.expected, result)
			}
		})
	}
}

// TestBuiltinEndsWith tests the endsWith() function.
func TestBuiltinEndsWith(t *testing.T) {
	testCases := []struct {
		name      string
		args      []interface{}
		expected  bool
		expectErr bool
	}{
		{
			name:     "match",
			args:     []interface{}{"hello world", "world"},
			expected: true,
		},
		{
			name:     "no_match",
			args:     []interface{}{"hello world", "hello"},
			expected: false,
		},
		{
			name:     "empty_suffix",
			args:     []interface{}{"hello", ""},
			expected: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := builtinEndsWith(tc.args)

			if tc.expectErr {
				if err == nil {
					t.Fatal("Expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if result.(bool) != tc.expected {
				t.Fatalf("Expected %v, got %v", tc.expected, result)
			}
		})
	}
}

// TestBuiltinContains tests the contains() function.
func TestBuiltinContains(t *testing.T) {
	testCases := []struct {
		name      string
		args      []interface{}
		expected  bool
		expectErr bool
	}{
		{
			name:     "match",
			args:     []interface{}{"hello world", "lo wo"},
			expected: true,
		},
		{
			name:     "no_match",
			args:     []interface{}{"hello world", "xyz"},
			expected: false,
		},
		{
			name:     "empty_substr",
			args:     []interface{}{"hello", ""},
			expected: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := builtinContains(tc.args)

			if tc.expectErr {
				if err == nil {
					t.Fatal("Expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if result.(bool) != tc.expected {
				t.Fatalf("Expected %v, got %v", tc.expected, result)
			}
		})
	}
}

// TestBuiltinRegexMatch tests the regex_match() function.
func TestBuiltinRegexMatch(t *testing.T) {
	testCases := []struct {
		name      string
		args      []interface{}
		expected  bool
		expectErr bool
	}{
		{
			name:     "simple_match",
			args:     []interface{}{"hello", "^hel"},
			expected: true,
		},
		{
			name:     "no_match",
			args:     []interface{}{"hello", "^world"},
			expected: false,
		},
		{
			name:     "complex_pattern",
			args:     []interface{}{"test123", `^test\d+$`},
			expected: true,
		},
		{
			name:      "invalid_pattern",
			args:      []interface{}{"hello", "["},
			expectErr: true,
		},
		{
			name:      "pattern_too_long",
			args:      []interface{}{"hello", string(make([]byte, 300))},
			expectErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := builtinRegexMatch(tc.args)

			if tc.expectErr {
				if err == nil {
					t.Fatal("Expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if result.(bool) != tc.expected {
				t.Fatalf("Expected %v, got %v", tc.expected, result)
			}
		})
	}
}

// TestDefaultRegistry verifies that all built-in functions are registered.
func TestDefaultRegistry(t *testing.T) {
	expectedFunctions := []string{
		"len",
		"upper",
		"lower",
		"startsWith",
		"endsWith",
		"contains",
		"regex_match",
	}

	for _, name := range expectedFunctions {
		t.Run(name, func(t *testing.T) {
			fn, ok := DefaultRegistry.Get(name)
			if !ok {
				t.Fatalf("Function %q not found in DefaultRegistry", name)
			}
			if fn == nil {
				t.Fatalf("Function %q is nil", name)
			}
		})
	}

	// Verify we have exactly the expected number of functions
	allFuncs := DefaultRegistry.List()
	if len(allFuncs) != len(expectedFunctions) {
		t.Fatalf("Expected %d functions, got %d", len(expectedFunctions), len(allFuncs))
	}
}

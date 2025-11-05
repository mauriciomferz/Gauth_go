package expr

import (
	"strings"
	"testing"
	"time"
)

// TestFunctionRegistry_Register tests basic registration
func TestFunctionRegistry_Register(t *testing.T) {
	registry := NewFunctionRegistry()
	
	metadata := FunctionMetadata{
		Name:        "test_func",
		Description: "Test function",
		MinArgs:     1,
		MaxArgs:     2,
		ReturnType:  ResultTypeBool,
		Category:    "test",
	}
	
	testFn := func(attrs map[string]string, now time.Time, args []FunctionArg) (FunctionResult, error) {
		return BoolResult(true), nil
	}
	
	err := registry.Register(metadata, testFn)
	if err != nil {
		t.Fatalf("Failed to register function: %v", err)
	}
	
	// Verify function was registered
	_, _, err = registry.Lookup("test_func")
	if err != nil {
		t.Errorf("Lookup failed after registration: %v", err)
	}
}

// TestFunctionRegistry_DuplicateRegistration tests duplicate prevention
func TestFunctionRegistry_DuplicateRegistration(t *testing.T) {
	registry := NewFunctionRegistry()
	
	metadata := FunctionMetadata{
		Name:       "dup_func",
		MinArgs:    0,
		MaxArgs:    0,
		ReturnType: ResultTypeBool,
	}
	
	testFn := func(attrs map[string]string, now time.Time, args []FunctionArg) (FunctionResult, error) {
		return BoolResult(true), nil
	}
	
	err := registry.Register(metadata, testFn)
	if err != nil {
		t.Fatalf("First registration failed: %v", err)
	}
	
	err = registry.Register(metadata, testFn)
	if err == nil {
		t.Error("Expected error for duplicate registration, got nil")
	}
	if !strings.Contains(err.Error(), "already registered") {
		t.Errorf("Expected 'already registered' error, got: %v", err)
	}
}

// TestFunctionRegistry_ValidationErrors tests metadata validation
func TestFunctionRegistry_ValidationErrors(t *testing.T) {
	registry := NewFunctionRegistry()
	testFn := func(attrs map[string]string, now time.Time, args []FunctionArg) (FunctionResult, error) {
		return BoolResult(true), nil
	}
	
	tests := []struct {
		name        string
		metadata    FunctionMetadata
		expectedErr string
	}{
		{
			name: "empty name",
			metadata: FunctionMetadata{
				Name:    "",
				MinArgs: 0,
				MaxArgs: 0,
			},
			expectedErr: "name cannot be empty",
		},
		{
			name: "negative MinArgs",
			metadata: FunctionMetadata{
				Name:    "test",
				MinArgs: -1,
				MaxArgs: 0,
			},
			expectedErr: "cannot be negative",
		},
		{
			name: "MaxArgs < MinArgs",
			metadata: FunctionMetadata{
				Name:    "test2",
				MinArgs: 5,
				MaxArgs: 3,
			},
			expectedErr: "must be >= MinArgs",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := registry.Register(tt.metadata, testFn)
			if err == nil {
				t.Errorf("Expected error containing '%s', got nil", tt.expectedErr)
			} else if !strings.Contains(err.Error(), tt.expectedErr) {
				t.Errorf("Expected error containing '%s', got: %v", tt.expectedErr, err)
			}
		})
	}
}

// TestFunctionRegistry_Call tests function invocation
func TestFunctionRegistry_Call(t *testing.T) {
	registry := NewFunctionRegistry()
	
	// Register a simple function
	registry.Register(FunctionMetadata{
		Name:       "add",
		MinArgs:    2,
		MaxArgs:    2,
		ArgTypes:   []ArgType{ArgTypeNumeric, ArgTypeNumeric},
		ReturnType: ResultTypeNumeric,
		Category:   "numeric",
	}, func(attrs map[string]string, now time.Time, args []FunctionArg) (FunctionResult, error) {
		return NumericResult(args[0].NumericValue + args[1].NumericValue), nil
	})
	
	result, err := registry.Call("add", nil, time.Now(), []FunctionArg{
		NumericArg(5),
		NumericArg(3),
	})
	
	if err != nil {
		t.Fatalf("Call failed: %v", err)
	}
	
	if result.NumericValue != 8 {
		t.Errorf("Expected result 8, got %f", result.NumericValue)
	}
}

// TestFunctionRegistry_ArgumentValidation tests argument count validation
func TestFunctionRegistry_ArgumentValidation(t *testing.T) {
	registry := NewFunctionRegistry()
	
	registry.Register(FunctionMetadata{
		Name:       "strict_func",
		MinArgs:    2,
		MaxArgs:    3,
		ReturnType: ResultTypeBool,
	}, func(attrs map[string]string, now time.Time, args []FunctionArg) (FunctionResult, error) {
		return BoolResult(true), nil
	})
	
	tests := []struct {
		name      string
		args      []FunctionArg
		expectErr bool
	}{
		{"too few args", []FunctionArg{StringArg("a")}, true},
		{"min args", []FunctionArg{StringArg("a"), StringArg("b")}, false},
		{"max args", []FunctionArg{StringArg("a"), StringArg("b"), StringArg("c")}, false},
		{"too many args", []FunctionArg{StringArg("a"), StringArg("b"), StringArg("c"), StringArg("d")}, true},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := registry.Call("strict_func", nil, time.Now(), tt.args)
			if tt.expectErr && err == nil {
				t.Error("Expected error, got nil")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("Expected no error, got: %v", err)
			}
		})
	}
}

// TestBuiltInFunction_Contains tests the contains function
func TestBuiltInFunction_Contains(t *testing.T) {
	registry := NewFunctionRegistry()
	
	tests := []struct {
		haystack string
		needle   string
		expected bool
	}{
		{"hello world", "world", true},
		{"hello world", "foo", false},
		{"", "test", false},
		{"test", "", true},
	}
	
	for _, tt := range tests {
		result, err := registry.Call("contains", nil, time.Now(), []FunctionArg{
			StringArg(tt.haystack),
			StringArg(tt.needle),
		})
		
		if err != nil {
			t.Errorf("contains(%q, %q) failed: %v", tt.haystack, tt.needle, err)
			continue
		}
		
		if result.BoolValue != tt.expected {
			t.Errorf("contains(%q, %q) = %v, expected %v", tt.haystack, tt.needle, result.BoolValue, tt.expected)
		}
	}
}

// TestBuiltInFunction_StartsEndsWith tests prefix/suffix functions
func TestBuiltInFunction_StartsEndsWith(t *testing.T) {
	registry := NewFunctionRegistry()
	
	tests := []struct {
		fn       string
		str      string
		pattern  string
		expected bool
	}{
		{"starts_with", "hello world", "hello", true},
		{"starts_with", "hello world", "world", false},
		{"ends_with", "hello world", "world", true},
		{"ends_with", "hello world", "hello", false},
	}
	
	for _, tt := range tests {
		result, err := registry.Call(tt.fn, nil, time.Now(), []FunctionArg{
			StringArg(tt.str),
			StringArg(tt.pattern),
		})
		
		if err != nil {
			t.Errorf("%s(%q, %q) failed: %v", tt.fn, tt.str, tt.pattern, err)
			continue
		}
		
		if result.BoolValue != tt.expected {
			t.Errorf("%s(%q, %q) = %v, expected %v", tt.fn, tt.str, tt.pattern, result.BoolValue, tt.expected)
		}
	}
}

// TestBuiltInFunction_RegexMatch tests regex matching
func TestBuiltInFunction_RegexMatch(t *testing.T) {
	registry := NewFunctionRegistry()
	
	tests := []struct {
		str      string
		pattern  string
		expected bool
		expectErr bool
	}{
		{"test123", `^test\d+$`, true, false},
		{"test", `^test\d+$`, false, false},
		{"hello@example.com", `^[\w.-]+@[\w.-]+\.\w+$`, true, false},
		{"invalid", `(unclosed`, false, true}, // invalid regex
	}
	
	for _, tt := range tests {
		result, err := registry.Call("regex_match", nil, time.Now(), []FunctionArg{
			StringArg(tt.str),
			StringArg(tt.pattern),
		})
		
		if tt.expectErr {
			if err == nil {
				t.Errorf("regex_match(%q, %q) expected error, got nil", tt.str, tt.pattern)
			}
			continue
		}
		
		if err != nil {
			t.Errorf("regex_match(%q, %q) failed: %v", tt.str, tt.pattern, err)
			continue
		}
		
		if result.BoolValue != tt.expected {
			t.Errorf("regex_match(%q, %q) = %v, expected %v", tt.str, tt.pattern, result.BoolValue, tt.expected)
		}
	}
}

// TestBuiltInFunction_StringTransforms tests string transformation functions
func TestBuiltInFunction_StringTransforms(t *testing.T) {
	registry := NewFunctionRegistry()
	
	tests := []struct {
		fn       string
		input    string
		expected string
	}{
		{"to_upper", "hello", "HELLO"},
		{"to_lower", "WORLD", "world"},
		{"trim", "  test  ", "test"},
	}
	
	for _, tt := range tests {
		result, err := registry.Call(tt.fn, nil, time.Now(), []FunctionArg{StringArg(tt.input)})
		
		if err != nil {
			t.Errorf("%s(%q) failed: %v", tt.fn, tt.input, err)
			continue
		}
		
		if result.StringValue != tt.expected {
			t.Errorf("%s(%q) = %q, expected %q", tt.fn, tt.input, result.StringValue, tt.expected)
		}
	}
}

// TestBuiltInFunction_StrLength tests string length
func TestBuiltInFunction_StrLength(t *testing.T) {
	registry := NewFunctionRegistry()
	
	tests := []struct {
		input    string
		expected float64
	}{
		{"hello", 5},
		{"", 0},
		{"世界", 6}, // UTF-8 bytes
	}
	
	for _, tt := range tests {
		result, err := registry.Call("str_length", nil, time.Now(), []FunctionArg{StringArg(tt.input)})
		
		if err != nil {
			t.Errorf("str_length(%q) failed: %v", tt.input, err)
			continue
		}
		
		if result.NumericValue != tt.expected {
			t.Errorf("str_length(%q) = %f, expected %f", tt.input, result.NumericValue, tt.expected)
		}
	}
}

// TestBuiltInFunction_Numeric tests numeric functions
func TestBuiltInFunction_Numeric(t *testing.T) {
	registry := NewFunctionRegistry()
	
	tests := []struct {
		fn       string
		args     []FunctionArg
		expected float64
	}{
		{"abs", []FunctionArg{NumericArg(-5)}, 5},
		{"abs", []FunctionArg{NumericArg(5)}, 5},
		{"min", []FunctionArg{NumericArg(3), NumericArg(7)}, 3},
		{"max", []FunctionArg{NumericArg(3), NumericArg(7)}, 7},
	}
	
	for _, tt := range tests {
		result, err := registry.Call(tt.fn, nil, time.Now(), tt.args)
		
		if err != nil {
			t.Errorf("%s failed: %v", tt.fn, err)
			continue
		}
		
		if result.NumericValue != tt.expected {
			t.Errorf("%s = %f, expected %f", tt.fn, result.NumericValue, tt.expected)
		}
	}
}

// TestBuiltInFunction_TimeBetween tests time range checking
func TestBuiltInFunction_TimeBetween(t *testing.T) {
	registry := NewFunctionRegistry()
	
	tests := []struct {
		now      time.Time
		start    string
		end      string
		expected bool
	}{
		{time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC), "09:00", "17:00", true},
		{time.Date(2025, 1, 1, 8, 0, 0, 0, time.UTC), "09:00", "17:00", false},
		{time.Date(2025, 1, 1, 23, 0, 0, 0, time.UTC), "22:00", "06:00", true}, // overnight
		{time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC), "22:00", "06:00", false}, // overnight
	}
	
	for _, tt := range tests {
		result, err := registry.Call("time_between", nil, tt.now, []FunctionArg{
			StringArg(tt.start),
			StringArg(tt.end),
		})
		
		if err != nil {
			t.Errorf("time_between(%q, %q) at %v failed: %v", tt.start, tt.end, tt.now.Format("15:04"), err)
			continue
		}
		
		if result.BoolValue != tt.expected {
			t.Errorf("time_between(%q, %q) at %v = %v, expected %v", tt.start, tt.end, tt.now.Format("15:04"), result.BoolValue, tt.expected)
		}
	}
}

// TestBuiltInFunction_TimeQueries tests time query functions
func TestBuiltInFunction_TimeQueries(t *testing.T) {
	registry := NewFunctionRegistry()
	
	// Saturday
	saturday := time.Date(2025, 1, 18, 15, 30, 0, 0, time.UTC)
	// Monday
	monday := time.Date(2025, 1, 20, 9, 0, 0, 0, time.UTC)
	
	tests := []struct {
		fn       string
		now      time.Time
		expected interface{}
	}{
		{"is_weekend", saturday, true},
		{"is_weekend", monday, false},
		{"weekday", saturday, float64(6)}, // Saturday = 6
		{"weekday", monday, float64(1)},   // Monday = 1
		{"hour", time.Date(2025, 1, 1, 15, 30, 0, 0, time.UTC), float64(15)},
	}
	
	for _, tt := range tests {
		result, err := registry.Call(tt.fn, nil, tt.now, []FunctionArg{})
		
		if err != nil {
			t.Errorf("%s at %v failed: %v", tt.fn, tt.now, err)
			continue
		}
		
		if boolExp, ok := tt.expected.(bool); ok {
			if result.BoolValue != boolExp {
				t.Errorf("%s at %v = %v, expected %v", tt.fn, tt.now, result.BoolValue, boolExp)
			}
		} else if numExp, ok := tt.expected.(float64); ok {
			if result.NumericValue != numExp {
				t.Errorf("%s at %v = %f, expected %f", tt.fn, tt.now, result.NumericValue, numExp)
			}
		}
	}
}

// TestBuiltInFunction_In tests collection membership
func TestBuiltInFunction_In(t *testing.T) {
	registry := NewFunctionRegistry()
	
	tests := []struct {
		value    string
		list     []string
		expected bool
	}{
		{"apple", []string{"apple", "banana", "cherry"}, true},
		{"grape", []string{"apple", "banana", "cherry"}, false},
		{"", []string{"", "test"}, true},
	}
	
	for _, tt := range tests {
		args := []FunctionArg{StringArg(tt.value)}
		for _, item := range tt.list {
			args = append(args, StringArg(item))
		}
		
		result, err := registry.Call("in", nil, time.Now(), args)
		
		if err != nil {
			t.Errorf("in(%q, %v) failed: %v", tt.value, tt.list, err)
			continue
		}
		
		if result.BoolValue != tt.expected {
			t.Errorf("in(%q, %v) = %v, expected %v", tt.value, tt.list, result.BoolValue, tt.expected)
		}
	}
}

// TestFunctionRegistry_Metrics tests metrics tracking
func TestFunctionRegistry_Metrics(t *testing.T) {
	registry := NewFunctionRegistry()
	
	// Make some calls
	registry.Call("contains", nil, time.Now(), []FunctionArg{StringArg("hello"), StringArg("world")})
	registry.Call("contains", nil, time.Now(), []FunctionArg{StringArg("test"), StringArg("x")})
	
	metrics := registry.GetMetrics()
	
	if totalCalls, ok := metrics["total_calls"].(uint64); !ok || totalCalls < 2 {
		t.Errorf("Expected at least 2 total calls, got %v", metrics["total_calls"])
	}
}

// TestFunctionRegistry_ListByCategory tests category filtering
func TestFunctionRegistry_ListByCategory(t *testing.T) {
	registry := NewFunctionRegistry()
	
	stringFuncs := registry.ListByCategory("string")
	if len(stringFuncs) == 0 {
		t.Error("Expected string functions, got none")
	}
	
	numericFuncs := registry.ListByCategory("numeric")
	if len(numericFuncs) == 0 {
		t.Error("Expected numeric functions, got none")
	}
	
	for _, fn := range stringFuncs {
		if fn.Category != "string" {
			t.Errorf("Expected category 'string', got %s", fn.Category)
		}
	}
}

// TestFunctionRegistry_Unregister tests function removal
func TestFunctionRegistry_Unregister(t *testing.T) {
	registry := NewFunctionRegistry()
	
	metadata := FunctionMetadata{
		Name:       "temp_func",
		MinArgs:    0,
		MaxArgs:    0,
		ReturnType: ResultTypeBool,
	}
	
	registry.Register(metadata, func(attrs map[string]string, now time.Time, args []FunctionArg) (FunctionResult, error) {
		return BoolResult(true), nil
	})
	
	// Verify it exists
	_, _, err := registry.Lookup("temp_func")
	if err != nil {
		t.Fatal("Function should exist after registration")
	}
	
	// Unregister
	err = registry.Unregister("temp_func")
	if err != nil {
		t.Fatalf("Unregister failed: %v", err)
	}
	
	// Verify it's gone
	_, _, err = registry.Lookup("temp_func")
	if err == nil {
		t.Error("Function should not exist after unregistration")
	}
}

// Copyright 2025 AgentAuth Contributors
// SPDX-License-Identifier: MIT

package agentauth

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSecureJSONParser_BasicParsing tests normal JSON parsing functionality
func TestSecureJSONParser_BasicParsing(t *testing.T) {
	parser := DefaultSecureParser()

	validJSON := `{"name": "alice", "age": 30, "roles": ["admin", "user"]}`
	var result map[string]interface{}

	err := parser.ParseSecure([]byte(validJSON), &result)
	require.NoError(t, err)
	assert.Equal(t, "alice", result["name"])
	assert.Equal(t, float64(30), result["age"])
	assert.Len(t, result["roles"], 2)
}

// TestSecureJSONParser_DepthLimit tests max nesting depth enforcement
func TestSecureJSONParser_DepthLimit(t *testing.T) {
	parser := DefaultSecureParser()
	parser.MaxDepth = 5

	tests := []struct {
		name    string
		json    string
		wantErr bool
	}{
		{
			name:    "depth_3_allowed",
			json:    `{"a":{"b":{"c":123}}}`,
			wantErr: false,
		},
		{
			name:    "depth_5_allowed_exact_limit",
			json:    `{"a":{"b":{"c":{"d":{"e":123}}}}}`,
			wantErr: false,
		},
		{
			name:    "depth_6_rejected",
			json:    `{"a":{"b":{"c":{"d":{"e":{"f":123}}}}}}`,
			wantErr: true,
		},
		{
			name:    "array_depth_6_rejected",
			json:    `[[[[[[123]]]]]]`,
			wantErr: true,
		},
		{
			name:    "mixed_depth_6_rejected",
			json:    `{"a":[{"b":{"c":[{"d":123}]}}]}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result interface{}
			err := parser.ParseSecure([]byte(tt.json), &result)
			if tt.wantErr {
				assert.Error(t, err, "Expected depth limit error")
				assert.Contains(t, err.Error(), "nesting depth exceeds limit")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestSecureJSONParser_SizeLimit tests max payload size enforcement
func TestSecureJSONParser_SizeLimit(t *testing.T) {
	parser := DefaultSecureParser()
	parser.MaxSize = 1024 // 1KB limit for test

	// Generate JSON just under limit (should pass)
	smallJSON := `{"data":"` + strings.Repeat("x", 900) + `"}`
	var result map[string]interface{}
	err := parser.ParseSecure([]byte(smallJSON), &result)
	assert.NoError(t, err, "Small JSON should pass")

	// Generate JSON over limit (should fail)
	largeJSON := `{"data":"` + strings.Repeat("x", 2000) + `"}`
	err = parser.ParseSecure([]byte(largeJSON), &result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds max size")
}

// TestSecureJSONParser_UTF8Validation tests UTF-8 validation
func TestSecureJSONParser_UTF8Validation(t *testing.T) {
	parser := DefaultSecureParser()
	parser.ValidateUTF8 = true

	tests := []struct {
		name    string
		json    []byte
		wantErr bool
	}{
		{
			name:    "valid_utf8_ascii",
			json:    []byte(`{"name":"alice"}`),
			wantErr: false,
		},
		{
			name:    "valid_utf8_unicode",
			json:    []byte(`{"name":"日本語"}`),
			wantErr: false,
		},
		{
			name:    "invalid_utf8_sequence",
			json:    []byte{'{', '"', 'n', 'a', 'm', 'e', '"', ':', '"', 0xFF, 0xFE, '"', '}'},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result map[string]interface{}
			err := parser.ParseSecure(tt.json, &result)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "invalid UTF-8")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestSecureJSONParser_StrictUnknownFields tests strict unknown field rejection
func TestSecureJSONParser_StrictUnknownFields(t *testing.T) {
	type User struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}

	strictParser := DefaultSecureParser()
	strictParser.StrictUnknownFields = true

	lenientParser := DefaultSecureParser()
	lenientParser.StrictUnknownFields = false

	jsonWithUnknown := `{"name":"alice","age":30,"unknown_field":"value"}`

	// Strict mode should reject
	var strictUser User
	err := strictParser.ParseSecure([]byte(jsonWithUnknown), &strictUser)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown field")

	// Lenient mode should accept (ignore unknown fields)
	var lenientUser User
	err = lenientParser.ParseSecure([]byte(jsonWithUnknown), &lenientUser)
	assert.NoError(t, err)
	assert.Equal(t, "alice", lenientUser.Name)
	assert.Equal(t, 30, lenientUser.Age)
}

// TestSecureJSONParser_MalformedJSON tests error handling for malformed JSON
func TestSecureJSONParser_MalformedJSON(t *testing.T) {
	parser := DefaultSecureParser()

	malformedTests := []string{
		`{`,                // Incomplete object
		`{"key":}`,         // Missing value
		`{"key":"value"`,   // Missing closing brace
		`{"key":value}`,    // Unquoted string value
		`{'key':'value'}`,  // Single quotes instead of double
		`{"key":"value",}`, // Trailing comma
		`{key:"value"}`,    // Unquoted key
	}

	for i, malformed := range malformedTests {
		t.Run(fmt.Sprintf("malformed_%d", i), func(t *testing.T) {
			var result map[string]interface{}
			err := parser.ParseSecure([]byte(malformed), &result)
			assert.Error(t, err, "Malformed JSON should be rejected")
		})
	}
}

// TestSecureJSONParser_NumericPrecision tests numeric value handling
func TestSecureJSONParser_NumericPrecision(t *testing.T) {
	parser := DefaultSecureParser()

	tests := []struct {
		name     string
		json     string
		expected interface{}
	}{
		{
			name:     "small_integer",
			json:     `{"value":42}`,
			expected: float64(42),
		},
		{
			name:     "large_integer",
			json:     `{"value":9007199254740991}`, // Max safe integer in float64
			expected: float64(9007199254740991),
		},
		{
			name:     "float_value",
			json:     `{"value":3.14159}`,
			expected: float64(3.14159),
		},
		{
			name:     "scientific_notation",
			json:     `{"value":1.23e10}`,
			expected: float64(1.23e10),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result map[string]interface{}
			err := parser.ParseSecure([]byte(tt.json), &result)
			require.NoError(t, err)
			assert.InDelta(t, tt.expected, result["value"], 0.000001)
		})
	}
}

// TestSecureJSONParser_EdgeCases tests edge case handling
func TestSecureJSONParser_EdgeCases(t *testing.T) {
	parser := DefaultSecureParser()

	tests := []struct {
		name    string
		json    string
		wantErr bool
	}{
		{
			name:    "empty_object",
			json:    `{}`,
			wantErr: false,
		},
		{
			name:    "empty_array",
			json:    `[]`,
			wantErr: false,
		},
		{
			name:    "null_value",
			json:    `{"key":null}`,
			wantErr: false,
		},
		{
			name:    "boolean_values",
			json:    `{"t":true,"f":false}`,
			wantErr: false,
		},
		{
			name:    "nested_empty_objects",
			json:    `{"a":{},"b":{"c":{}}}`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result interface{}
			err := parser.ParseSecure([]byte(tt.json), &result)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestComputeMaxDepth tests the depth calculation function directly
func TestComputeMaxDepth(t *testing.T) {
	tests := []struct {
		name          string
		json          string
		expectedDepth int
	}{
		{
			name:          "flat_object",
			json:          `{"a":1,"b":2}`,
			expectedDepth: 1,
		},
		{
			name:          "nested_2_levels",
			json:          `{"a":{"b":1}}`,
			expectedDepth: 2,
		},
		{
			name:          "nested_5_levels",
			json:          `{"a":{"b":{"c":{"d":{"e":1}}}}}`,
			expectedDepth: 5,
		},
		{
			name:          "array_depth_3",
			json:          `[[[1]]]`,
			expectedDepth: 3,
		},
		{
			name:          "mixed_nesting",
			json:          `{"a":[{"b":{"c":1}}]}`,
			expectedDepth: 4,
		},
		{
			name:          "string_with_brackets",
			json:          `{"a":"value with { and [ brackets }]"}`,
			expectedDepth: 1, // Brackets in strings don't count
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			depth := computeMaxDepth([]byte(tt.json))
			assert.Equal(t, tt.expectedDepth, depth)
		})
	}
}

// TestValidateJSONSecurity tests the fast pre-validation function
func TestValidateJSONSecurity(t *testing.T) {
	tests := []struct {
		name     string
		json     string
		maxSize  int
		maxDepth int
		wantErr  bool
	}{
		{
			name:     "valid_json",
			json:     `{"a":{"b":1}}`,
			maxSize:  1024,
			maxDepth: 5,
			wantErr:  false,
		},
		{
			name:     "exceeds_size",
			json:     `{"data":"` + strings.Repeat("x", 2000) + `"}`,
			maxSize:  1024,
			maxDepth: 10,
			wantErr:  true,
		},
		{
			name:     "exceeds_depth",
			json:     `[[[[[[1]]]]]]`, // Depth 6
			maxSize:  1024,
			maxDepth: 5,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateJSONSecurity([]byte(tt.json), tt.maxDepth, tt.maxSize)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestSecureJSONParser_BackwardCompatibility verifies that default settings
// maintain backward compatibility with standard json.Unmarshal behavior
func TestSecureJSONParser_BackwardCompatibility(t *testing.T) {
	parser := DefaultSecureParser()

	// Typical JWT payload (should parse identically to json.Unmarshal)
	jwtPayload := `{
		"sub": "user123",
		"scope": "read write",
		"exp": 1735689600,
		"iat": 1735603200,
		"iss": "https://auth.example.com",
		"aud": "https://api.example.com",
		"jti": "550e8400-e29b-41d4-a716-446655440000"
	}`

	// Parse with SecureJSONParser
	var secureResult map[string]interface{}
	err := parser.ParseSecure([]byte(jwtPayload), &secureResult)
	require.NoError(t, err)

	// Parse with standard json.Unmarshal
	var standardResult map[string]interface{}
	err = json.Unmarshal([]byte(jwtPayload), &standardResult)
	require.NoError(t, err)

	// Verify identical results
	assert.Equal(t, standardResult["sub"], secureResult["sub"])
	assert.Equal(t, standardResult["scope"], secureResult["scope"])
	assert.Equal(t, standardResult["exp"], secureResult["exp"])
	assert.Equal(t, standardResult["iat"], secureResult["iat"])
	assert.Equal(t, standardResult["iss"], secureResult["iss"])
	assert.Equal(t, standardResult["aud"], secureResult["aud"])
	assert.Equal(t, standardResult["jti"], secureResult["jti"])
}

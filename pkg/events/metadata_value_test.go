package events

import (
	"encoding/json"
	"testing"
	"time"
)

// This test file verifies functionality of the MetadataValue type
// which provides strong typing for event metadata.

func TestMetadataValue(t *testing.T) {
	t.Run("Value Creators", func(t *testing.T) {
		// Test string creation
		sv := NewStringValue("test")
		if sv.Type != "string" {
			t.Errorf("Expected type string, got %s", sv.Type)
		}
		if sv.Value.(string) != "test" {
			t.Errorf("Expected value 'test', got %v", sv.Value)
		}

		// Test int creation
		iv := NewIntValue(42)
		if iv.Type != "int" {
			t.Errorf("Expected type int, got %s", iv.Type)
		}
		if iv.Value.(int) != 42 {
			t.Errorf("Expected value 42, got %v", iv.Value)
		}

		// Test bool creation
		bv := NewBoolValue(true)
		if bv.Type != "bool" {
			t.Errorf("Expected type bool, got %s", bv.Type)
		}
		if !bv.Value.(bool) {
			t.Errorf("Expected value true, got %v", bv.Value)
		}

		// Test float creation
		fv := NewFloatValue(3.14)
		if fv.Type != "float" {
			t.Errorf("Expected type float, got %s", fv.Type)
		}
		if fv.Value.(float64) != 3.14 {
			t.Errorf("Expected value 3.14, got %v", fv.Value)
		}

		// Test time creation
		now := time.Now().UTC().Truncate(time.Second)
		tv := NewTimeValue(now)
		if tv.Type != "time" {
			t.Errorf("Expected type time, got %s", tv.Type)
		}
	})

	t.Run("ToString Conversion", func(t *testing.T) {
		tests := []struct {
			name     string
			value    MetadataValue
			expected string
		}{
			{
				name:     "string value",
				value:    NewStringValue("test"),
				expected: "test",
			},
			{
				name:     "int value",
				value:    NewIntValue(42),
				expected: "42",
			},
			{
				name:     "bool true value",
				value:    NewBoolValue(true),
				expected: "true",
			},
			{
				name:     "bool false value",
				value:    NewBoolValue(false),
				expected: "false",
			},
			{
				name:     "float value",
				value:    NewFloatValue(3.14),
				expected: "3.14",
			},
		}

		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				result := test.value.ToString()
				if result != test.expected {
					t.Errorf("Expected %q but got %q", test.expected, result)
				}
			})
		}
	})

	t.Run("ReadOnly Flag", func(t *testing.T) {
		// Create a value
		sv := NewStringValue("protected")

		// Make it read-only
		roSv := NewReadOnlyValue(sv)

		// Check that it's read-only
		if !roSv.ReadOnly {
			t.Error("Expected ReadOnly to be true")
		}

		// Check that the value is still accessible
		if roSv.Value.(string) != "protected" {
			t.Errorf("Expected value 'protected', got %v", roSv.Value)
		}
	})

	t.Run("Type Conversion - ToInt", func(t *testing.T) {
		// Direct int
		iv := NewIntValue(42)
		i, err := iv.ToInt()
		if err != nil {
			t.Errorf("Failed to convert int: %v", err)
		}
		if i != 42 {
			t.Errorf("Expected 42, got %d", i)
		}

		// From string
		sv := NewStringValue("42")
		i, err = sv.ToInt()
		if err != nil {
			t.Errorf("Failed to convert string to int: %v", err)
		}
		if i != 42 {
			t.Errorf("Expected 42, got %d", i)
		}

		// Invalid conversion
		bv := NewBoolValue(true)
		_, err = bv.ToInt()
		if err == nil {
			t.Error("Expected error when converting bool to int")
		}
	})

	t.Run("Type Conversion - ToBool", func(t *testing.T) {
		// Direct bool
		bv := NewBoolValue(true)
		b, err := bv.ToBool()
		if err != nil {
			t.Errorf("Failed to convert bool: %v", err)
		}
		if !b {
			t.Error("Expected true, got false")
		}

		// From string
		sv := NewStringValue("true")
		b, err = sv.ToBool()
		if err != nil {
			t.Errorf("Failed to convert string to bool: %v", err)
		}
		if !b {
			t.Error("Expected true, got false")
		}

		// "1" should be true
		sv = NewStringValue("1")
		b, err = sv.ToBool()
		if err != nil {
			t.Errorf("Failed to convert string '1' to bool: %v", err)
		}
		if !b {
			t.Error("Expected '1' to convert to true")
		}

		// Invalid conversion
		iv := NewIntValue(1)
		_, err = iv.ToBool()
		if err == nil {
			t.Error("Expected error when converting int to bool")
		}
	})

	t.Run("JSON Marshaling", func(t *testing.T) {
		values := []MetadataValue{
			NewStringValue("test string"),
			NewIntValue(42),
			NewBoolValue(true),
			NewFloatValue(3.14),
			NewReadOnlyValue(NewStringValue("protected")),
		}

		for _, value := range values {
			// Marshal to JSON
			data, err := json.Marshal(value)
			if err != nil {
				t.Fatalf("JSON marshaling failed: %v", err)
			}

			// Unmarshal back
			var result MetadataValue
			err = json.Unmarshal(data, &result)
			if err != nil {
				t.Fatalf("JSON unmarshaling failed: %v", err)
			}

			// Check type
			if result.Type != value.Type {
				t.Errorf("Expected type %q but got %q", value.Type, result.Type)
			}

			// Check ReadOnly flag
			if result.ReadOnly != value.ReadOnly {
				t.Errorf("Expected ReadOnly %v but got %v", value.ReadOnly, result.ReadOnly)
			}
		}
	})
}

package events

import (
	"encoding/json"
	"testing"
	"time"
)

// This test file verifies functionality of the MetadataValue type
// which provides strong typing for event metadata.

func TestMetadataValue(t *testing.T) {
	t.Run("Value Creators", testMetadataValueCreators)
	t.Run("ToString Conversion", testMetadataValueToString)
	t.Run("ReadOnly Flag", testMetadataValueReadOnly)
	t.Run("Type Conversion - ToInt", testMetadataValueToInt)
	t.Run("Type Conversion - ToBool", testMetadataValueToBool)
	t.Run("JSON Marshaling", testMetadataValueJSONMarshaling)
}

func testMetadataValueCreators(t *testing.T) {
	testStringValueCreation(t)
	testIntValueCreation(t)
	testBoolValueCreation(t)
	testFloatValueCreation(t)
	testTimeValueCreation(t)
}

func testStringValueCreation(t *testing.T) {
	sv := NewStringValue("test")
	if sv.Type != "string" {
		t.Errorf("Expected type string, got %s", sv.Type)
	}
	if sv.Value.(string) != "test" {
		t.Errorf("Expected value 'test', got %v", sv.Value)
	}
}

func testIntValueCreation(t *testing.T) {
	iv := NewIntValue(42)
	if iv.Type != "int" {
		t.Errorf("Expected type int, got %s", iv.Type)
	}
	if iv.Value.(int) != 42 {
		t.Errorf("Expected value 42, got %v", iv.Value)
	}
}

func testBoolValueCreation(t *testing.T) {
	bv := NewBoolValue(true)
	if bv.Type != "bool" {
		t.Errorf("Expected type bool, got %s", bv.Type)
	}
	if !bv.Value.(bool) {
		t.Errorf("Expected value true, got %v", bv.Value)
	}
}

func testFloatValueCreation(t *testing.T) {
	fv := NewFloatValue(3.14)
	if fv.Type != "float" {
		t.Errorf("Expected type float, got %s", fv.Type)
	}
	if fv.Value.(float64) != 3.14 {
		t.Errorf("Expected value 3.14, got %v", fv.Value)
	}
}

func testTimeValueCreation(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	tv := NewTimeValue(now)
	if tv.Type != "time" {
		t.Errorf("Expected type time, got %s", tv.Type)
	}
}

func testMetadataValueToString(t *testing.T) {
	tests := []struct {
		name     string
		value    MetadataValue
		expected string
	}{
		{"string value", NewStringValue("test"), "test"},
		{"int value", NewIntValue(42), "42"},
		{"bool true value", NewBoolValue(true), "true"},
		{"bool false value", NewBoolValue(false), "false"},
		{"float value", NewFloatValue(3.14), "3.14"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := test.value.ToString()
			if result != test.expected {
				t.Errorf("Expected %q but got %q", test.expected, result)
			}
		})
	}
}

func testMetadataValueReadOnly(t *testing.T) {
	sv := NewStringValue("protected")
	roSv := NewReadOnlyValue(sv)

	if !roSv.ReadOnly {
		t.Error("Expected ReadOnly to be true")
	}

	if roSv.Value.(string) != "protected" {
		t.Errorf("Expected value 'protected', got %v", roSv.Value)
	}
}

func testMetadataValueToInt(t *testing.T) {
	// Direct int conversion
	iv := NewIntValue(42)
	i, err := iv.ToInt()
	if err != nil {
		t.Errorf("Failed to convert int: %v", err)
	}
	if i != 42 {
		t.Errorf("Expected 42, got %d", i)
	}

	// String to int conversion
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
}

func testMetadataValueToBool(t *testing.T) {
	// Direct bool conversion
	bv := NewBoolValue(true)
	b, err := bv.ToBool()
	if err != nil {
		t.Errorf("Failed to convert bool: %v", err)
	}
	if !b {
		t.Error("Expected true, got false")
	}

	// String to bool conversions
	testStringToBoolConversions(t)

	// Invalid conversion
	iv := NewIntValue(1)
	_, err = iv.ToBool()
	if err == nil {
		t.Error("Expected error when converting int to bool")
	}
}

func testStringToBoolConversions(t *testing.T) {
	// "true" to bool
	sv := NewStringValue("true")
	b, err := sv.ToBool()
	if err != nil {
		t.Errorf("Failed to convert string to bool: %v", err)
	}
	if !b {
		t.Error("Expected true, got false")
	}

	// "1" to bool
	sv = NewStringValue("1")
	b, err = sv.ToBool()
	if err != nil {
		t.Errorf("Failed to convert string '1' to bool: %v", err)
	}
	if !b {
		t.Error("Expected '1' to convert to true")
	}
}

func testMetadataValueJSONMarshaling(t *testing.T) {
	values := []MetadataValue{
		NewStringValue("test string"),
		NewIntValue(42),
		NewBoolValue(true),
		NewFloatValue(3.14),
		NewReadOnlyValue(NewStringValue("protected")),
	}

	for _, value := range values {
		data, err := json.Marshal(value)
		if err != nil {
			t.Fatalf("JSON marshaling failed: %v", err)
		}

		var result MetadataValue
		err = json.Unmarshal(data, &result)
		if err != nil {
			t.Fatalf("JSON unmarshaling failed: %v", err)
		}

		if result.Type != value.Type {
			t.Errorf("Expected type %q but got %q", value.Type, result.Type)
		}

		if result.ReadOnly != value.ReadOnly {
			t.Errorf("Expected ReadOnly %v but got %v", value.ReadOnly, result.ReadOnly)
		}
	}
}

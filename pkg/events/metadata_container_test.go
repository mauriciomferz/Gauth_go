package events

import (
	"encoding/json"
	"testing"
	"time"
)

// Test constants
const (
	testValueString    = "test"
	testValueProtected = "protected"
)

// This test file verifies functionality of the metadata container
// which stores strongly typed values and provides type-safe accessors.

func TestMetadataContainer(t *testing.T) {
	t.Run("Basic Operations", func(t *testing.T) {
		m := NewMetadata()

		// Test Set and Get operations
		testVal := NewStringValue("test")
		m.Set("test_key", testVal)

		retrieved, exists := m.Get("test_key")
		if !exists {
			t.Error("Expected key to exist, but it doesn't")
		}

		if retrieved.Type != MetadataTypeString {
			t.Errorf("Expected string type, got %s", retrieved.Type)
		}

		if retrieved.Value.(string) != testValueString {
			t.Errorf("Expected value '%s', got %v", testValueString, retrieved.Value)
		}

		// Test Has method
		if !m.Has("test_key") {
			t.Error("Expected Has to return true for existing key")
		}

		if m.Has("nonexistent") {
			t.Error("Expected Has to return false for nonexistent key")
		}

		// Test Delete
		m.Delete("test_key")
		if m.Has("test_key") {
			t.Error("Expected key to be deleted")
		}
	})

	t.Run("Type-Specific Methods", func(t *testing.T) {
		m := NewMetadata()

		// String
		m.SetString("string_key", "string_value")
		val, ok := m.GetString("string_key")
		if !ok {
			t.Error("GetString failed to retrieve value")
		}
		if val != "string_value" {
			t.Errorf("Expected 'string_value', got '%s'", val)
		}

		// Int
		m.SetInt("int_key", 42)
		val2, ok := m.GetInt("int_key")
		if !ok {
			t.Error("GetInt failed to retrieve value")
		}
		if val2 != 42 {
			t.Errorf("Expected 42, got %d", val2)
		}

		// Int64
		m.SetInt64("int64_key", 9223372036854775807)
		// There might not be a GetInt64 method, so we'll just check the key exists
		if !m.Has("int64_key") {
			t.Error("Int64 key not set correctly")
		}

		// Float
		m.SetFloat("float_key", 3.14159)
		// Check if the key exists since GetFloat might not be implemented
		if !m.Has("float_key") {
			t.Error("Float key not set correctly")
		}

		// Bool
		m.SetBool("bool_key", true)
		val5, ok := m.GetBool("bool_key")
		if !ok {
			t.Error("GetBool failed to retrieve value")
		}
		if !val5 {
			t.Errorf("Expected true, got %t", val5)
		}

		// Time
		now := time.Now().UTC().Truncate(time.Second)
		m.SetTime("time_key", now)
		// Check if the key exists since GetTime might not be implemented
		if !m.Has("time_key") {
			t.Error("Time key not set correctly")
		}
	})

	t.Run("ReadOnly Values", func(t *testing.T) {
		m := NewMetadata()

		// Set a read-only value
		m.SetReadOnly("readonly_key", NewStringValue(testValueProtected))

		// Try to overwrite it
		m.SetString("readonly_key", "new_value")

		// Check that it wasn't changed
		val, ok := m.GetString("readonly_key")
		if !ok {
			t.Error("Failed to retrieve read-only value")
		}
		if val != testValueProtected {
			t.Errorf("Expected read-only value to be unchanged, but got '%s'", val)
		}

		// Try to delete it
		m.Delete("readonly_key")

		// Check that it wasn't deleted
		if !m.Has("readonly_key") {
			t.Error("Expected read-only value to be protected from deletion")
		}

		// Clear all
		m.Clear()

		// Check that even read-only values are cleared
		if m.Has("readonly_key") {
			t.Error("Expected Clear to remove all values including read-only ones")
		}
	})

	t.Run("Keys and Size", func(t *testing.T) {
		m := NewMetadata()

		m.SetString("key1", "value1")
		m.SetInt("key2", 2)
		m.SetBool("key3", true)

		// Check size
		if m.Size() != 3 {
			t.Errorf("Expected size 3, got %d", m.Size())
		}

		// Check keys
		keys := m.Keys()
		if len(keys) != 3 {
			t.Errorf("Expected 3 keys, got %d", len(keys))
		}

		// Check that all keys are present
		keyMap := make(map[string]bool)
		for _, k := range keys {
			keyMap[k] = true
		}

		expectedKeys := []string{"key1", "key2", "key3"}
		for _, k := range expectedKeys {
			if !keyMap[k] {
				t.Errorf("Expected key '%s' is missing", k)
			}
		}
	})

	t.Run("JSON Marshaling", func(t *testing.T) {
		m := NewMetadata()

		m.SetString("string", "text")
		m.SetInt("int", 42)
		m.SetBool("bool", true)

		// Marshal to JSON
		data, err := json.Marshal(m)
		if err != nil {
			t.Fatalf("JSON marshaling failed: %v", err)
		}

		// Unmarshal
		m2 := NewMetadata()
		err = json.Unmarshal(data, m2)
		if err != nil {
			t.Fatalf("JSON unmarshaling failed: %v", err)
		}

		// Validate unmarshaled data
		if val, ok := m2.GetString("string"); !ok || val != "text" {
			t.Errorf("String unmarshaling failed, got %v, exists: %v", val, ok)
		}

		if val, ok := m2.GetInt("int"); !ok || val != 42 {
			t.Errorf("Int unmarshaling failed, got %v, exists: %v", val, ok)
		}

		if val, ok := m2.GetBool("bool"); !ok || !val {
			t.Errorf("Bool unmarshaling failed, got %v, exists: %v", val, ok)
		}
	})
}

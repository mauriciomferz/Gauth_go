package resource

import (
	"testing"
)

func TestResourceConfig(t *testing.T) {
	config := NewResourceConfig()

	// Test string values
	config.SetString("name", "test-resource")
	if val, ok := config.GetString("name"); !ok || val != "test-resource" {
		t.Errorf("String value not set correctly: got %v, want %s", val, "test-resource")
	}

	// Test int values
	config.SetInt("count", 42)
	if val, ok := config.GetInt("count"); !ok || val != 42 {
		t.Errorf("Int value not set correctly: got %v, want %d", val, 42)
	}

	// Test float values
	config.SetFloat("ratio", 3.14)
	if val, ok := config.GetFloat("ratio"); !ok || val != 3.14 {
		t.Errorf("Float value not set correctly: got %v, want %f", val, 3.14)
	}

	// Test bool values
	config.SetBool("enabled", true)
	if val, ok := config.GetBool("enabled"); !ok || val != true {
		t.Errorf("Bool value not set correctly: got %v, want %t", val, true)
	}

	// Test map values
	mapVal := map[string]interface{}{
		"key1": "value1",
		"key2": 123,
	}
	config.SetMap("settings", mapVal)
	if val, ok := config.GetMap("settings"); !ok {
		t.Error("Map value not set correctly")
	} else {
		if val["key1"] != "value1" || val["key2"] != 123 {
			t.Errorf("Map value content not correct: got %v", val)
		}
	}

	// Test slice values
	sliceVal := []interface{}{"one", 2, true}
	config.SetSlice("list", sliceVal)
	if val, ok := config.GetSlice("list"); !ok {
		t.Error("Slice value not set correctly")
	} else {
		if len(val) != 3 || val[0] != "one" || val[1] != 2 || val[2] != true {
			t.Errorf("Slice value content not correct: got %v", val)
		}
	}

	// Test Has and Remove
	if !config.Has("name") {
		t.Error("Has should return true for existing key")
	}
	config.Remove("name")
	if config.Has("name") {
		t.Error("Remove should delete the key")
	}
}

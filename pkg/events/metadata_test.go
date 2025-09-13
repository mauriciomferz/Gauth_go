package events

import (
	"encoding/json"
	"testing"
	"time"
)

func TestMetadataTypes(t *testing.T) {
	// Create a new empty metadata
	meta := NewMetadata()

	// Test setting and retrieving string values
	meta.SetString("username", "testuser")
	if val, exists := meta.GetString("username"); !exists || val != "testuser" {
		t.Errorf("String metadata not stored correctly: got %v, exists: %v, want 'testuser'", val, exists)
	}

	// Test setting and retrieving int values
	meta.SetInt("count", 42)
	if val, exists := meta.GetInt("count"); !exists || val != 42 {
		t.Errorf("Int metadata not stored correctly: got %d, exists: %v, want 42", val, exists)
	}

	// Test setting and retrieving bool values
	meta.SetBool("verified", true)
	if val, exists := meta.GetBool("verified"); !exists || !val {
		t.Errorf("Bool metadata not stored correctly: got %v, exists: %v, want true", val, exists)
	}

	// Test read-only values
	meta.SetReadOnly("readonly", NewStringValue("protected"))
	if val, exists := meta.Get("readonly"); !exists || !val.ReadOnly {
		t.Errorf("Read-only flag not set correctly: got %v, exists: %v, expected ReadOnly=true", val, exists)
	}

	// Try to update a read-only value (should not change)
	meta.SetString("readonly", "changed")
	if val, exists := meta.GetString("readonly"); !exists || val != "protected" {
		t.Errorf("Read-only value incorrectly changed: got %v, expected 'protected'", val)
	}
}

func TestMetadataJSON(t *testing.T) {
	// Create metadata with different types
	meta := NewMetadata()
	meta.SetString("name", "John")
	meta.SetInt("age", 30)
	meta.SetBool("active", true)
	meta.SetFloat("score", 95.5)

	now := time.Now()
	meta.SetTime("created", now)

	// Marshal to JSON
	data, err := json.Marshal(meta)
	if err != nil {
		t.Fatalf("Failed to marshal metadata to JSON: %v", err)
	}

	// Unmarshal back
	var newMeta Metadata
	err = json.Unmarshal(data, &newMeta)
	if err != nil {
		t.Fatalf("Failed to unmarshal metadata from JSON: %v", err)
	}

	// Verify values were preserved
	if name, exists := newMeta.GetString("name"); !exists || name != "John" {
		t.Errorf("String value not preserved in JSON round-trip: got %v, want 'John'", name)
	}

	if age, exists := newMeta.GetInt("age"); !exists || age != 30 {
		t.Errorf("Int value not preserved in JSON round-trip: got %v, want 30", age)
	}

	if active, exists := newMeta.GetBool("active"); !exists || !active {
		t.Errorf("Bool value not preserved in JSON round-trip: got %v, want true", active)
	}
}

func TestEventWithTypedMetadata(t *testing.T) {
	// Create a new event
	event := NewEvent()
	event.Type = EventTypeAuth
	event.Action = string(ActionLogin) // Using a known auth action
	event.Status = string(StatusSuccess)
	event.Subject = "user123"

	// Add metadata using the fluent API
	event = event.WithStringMetadata("source_ip", "192.168.1.1")
	event = event.WithIntMetadata("attempt", 1)
	event = event.WithBoolMetadata("remember_me", true)

	// Verify metadata was correctly added
	if event.Metadata == nil {
		t.Fatal("Event metadata is nil")
	}

	if sourceIP, exists := event.Metadata.GetString("source_ip"); !exists || sourceIP != "192.168.1.1" {
		t.Errorf("String metadata not set correctly: got %v, want '192.168.1.1'", sourceIP)
	}

	if attempt, exists := event.Metadata.GetInt("attempt"); !exists || attempt != 1 {
		t.Errorf("Int metadata not set correctly: got %v, want 1", attempt)
	}

	if rememberMe, exists := event.Metadata.GetBool("remember_me"); !exists || !rememberMe {
		t.Errorf("Bool metadata not set correctly: got %v, want true", rememberMe)
	}

}

package poa

import (
	"encoding/json"
	"strings"
	"testing"
)

// MockInterpreter demonstrates how external logic would evaluate SpecialConditions
type MockInterpreter struct{}

// Evaluate checks if conditions are met. Returns true if allowed (or no conditions), false if blocked.
func (m *MockInterpreter) Evaluate(conditions []string, ctx map[string]interface{}) (bool, error) {
	if len(conditions) == 0 {
		return true, nil
	}
	for _, cond := range conditions {
		// Simple syntax: "key operator value"
		// Supported: "time < 2025-01-01", "location == 'US'"
		parts := strings.Split(cond, " ")
		if len(parts) != 3 {
			continue // skip complex/unknown formats for this mock
		}
		key, op, val := parts[0], parts[1], parts[2]
		val = strings.Trim(val, "'") // Strip quotes

		switch key {
		case "env":
			if ctxVal, ok := ctx["env"].(string); ok {
				if op == "==" && ctxVal != val {
					return false, nil // Condition failed
				}
				if op == "!=" && ctxVal == val {
					return false, nil
				}
			}
		}
	}
	return true, nil
}

func TestSpecialConditions_RFC115_C7(t *testing.T) {
	t.Run("Structure_Serialization", func(t *testing.T) {
		sc := SpecialConditions{
			ConditionalEffectiveness: []string{"env == 'prod'", "time < 2030"},
			ImmediateNotification:    []string{"on_access", "on_error"},
		}

		// Wrap in Definition to test persistence context
		def := PoADefinition{
			Requirements: Requirements{
				SpecialConditions: sc,
			},
		}

		data, err := json.Marshal(def)
		if err != nil {
			t.Fatalf("Marshal failed: %v", err)
		}

		var decoded PoADefinition
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("Unmarshal failed: %v", err)
		}

		decodedSC := decoded.Requirements.SpecialConditions
		if len(decodedSC.ConditionalEffectiveness) != 2 {
			t.Errorf("Expected 2 effectiveness conditions, got %d", len(decodedSC.ConditionalEffectiveness))
		}
		if len(decodedSC.ImmediateNotification) != 2 {
			t.Errorf("Expected 2 notification triggers, got %d", len(decodedSC.ImmediateNotification))
		}
		if decodedSC.ConditionalEffectiveness[0] != "env == 'prod'" {
			t.Errorf("Mismatch condition data")
		}
	})

	t.Run("MockInterpreter_Evaluation", func(t *testing.T) {
		interpreter := &MockInterpreter{}

		conditions := []string{"env == 'prod'"}

		// Case 1: Context matches
		ctxPass := map[string]interface{}{"env": "prod"}
		allowed, _ := interpreter.Evaluate(conditions, ctxPass)
		if !allowed {
			t.Error("Expected condition to pass for env=prod")
		}

		// Case 2: Context fails
		ctxFail := map[string]interface{}{"env": "dev"}
		allowed, _ = interpreter.Evaluate(conditions, ctxFail)
		if allowed {
			t.Error("Expected condition to fail for env=dev")
		}

		// Case 3: Empty conditions (default allow)
		allowed, _ = interpreter.Evaluate(nil, ctxFail)
		if !allowed {
			t.Error("Expected empty conditions to allow")
		}
	})
}

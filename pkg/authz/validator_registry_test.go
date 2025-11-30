package authz

import (
	"errors"
	"strings"
	"testing"
	"time"
)

// TestValidatorRegistry_Register tests registering validators
func TestValidatorRegistry_Register(t *testing.T) {
	vr := NewValidatorRegistry()

	t.Run("register valid validator", func(t *testing.T) {
		err := vr.Register("test-validator", func(req Request, policy Policy) error {
			return nil
		})
		if err != nil {
			t.Errorf("Register() error = %v, expected nil", err)
		}
	})

	t.Run("register with description", func(t *testing.T) {
		err := vr.Register("validator-with-desc", func(req Request, policy Policy) error {
			return nil
		}, WithDescription("Test validator"))
		if err != nil {
			t.Errorf("Register() error = %v, expected nil", err)
		}
	})

	t.Run("register with version", func(t *testing.T) {
		err := vr.Register("validator-with-version", func(req Request, policy Policy) error {
			return nil
		}, WithVersion("1.0.0"))
		if err != nil {
			t.Errorf("Register() error = %v, expected nil", err)
		}
	})

	t.Run("register with tags", func(t *testing.T) {
		err := vr.Register("validator-with-tags", func(req Request, policy Policy) error {
			return nil
		}, WithTags("security", "compliance"))
		if err != nil {
			t.Errorf("Register() error = %v, expected nil", err)
		}
	})

	t.Run("register with timeout", func(t *testing.T) {
		err := vr.Register("validator-with-timeout", func(req Request, policy Policy) error {
			return nil
		}, WithTimeout(100*time.Millisecond))
		if err != nil {
			t.Errorf("Register() error = %v, expected nil", err)
		}
	})

	t.Run("register with all options", func(t *testing.T) {
		err := vr.Register("validator-full", func(req Request, policy Policy) error {
			return nil
		}, WithDescription("Full validator"), WithVersion("2.0.0"), WithTags("tag1", "tag2"), WithTimeout(200*time.Millisecond))
		if err != nil {
			t.Errorf("Register() error = %v, expected nil", err)
		}
	})

	t.Run("register overwrites existing ID", func(t *testing.T) {
		// First registration
		err1 := vr.Register("overwrite-test", func(req Request, policy Policy) error {
			return errors.New("first")
		})
		if err1 != nil {
			t.Fatalf("First Register() error = %v", err1)
		}

		// Second registration (overwrite)
		err2 := vr.Register("overwrite-test", func(req Request, policy Policy) error {
			return errors.New("second")
		})
		if err2 != nil {
			t.Fatalf("Second Register() error = %v", err2)
		}

		// Test that the second function is used
		err := vr.Invoke("overwrite-test", Request{}, Policy{})
		if err == nil || !strings.Contains(err.Error(), "second") {
			t.Errorf("Expected second validator, got error: %v", err)
		}
	})

	t.Run("register with empty ID returns error", func(t *testing.T) {
		err := vr.Register("", func(req Request, policy Policy) error {
			return nil
		})
		if err == nil {
			t.Error("Expected error for empty ID, got nil")
		}
		if !strings.Contains(err.Error(), "id and function required") {
			t.Errorf("Expected 'id and function required' error, got: %v", err)
		}
	})

	t.Run("register with nil function returns error", func(t *testing.T) {
		err := vr.Register("nil-func", nil)
		if err == nil {
			t.Error("Expected error for nil function, got nil")
		}
		if !strings.Contains(err.Error(), "id and function required") {
			t.Errorf("Expected 'id and function required' error, got: %v", err)
		}
	})
}

// TestValidatorRegistry_Invoke tests invoking validators
func TestValidatorRegistry_Invoke(t *testing.T) {
	t.Run("invoke successful validator", func(t *testing.T) {
		vr := NewValidatorRegistry()
		called := false
		_ = vr.Register("success-validator", func(req Request, policy Policy) error {
			called = true
			return nil
		})

		err := vr.Invoke("success-validator", Request{}, Policy{})
		if err != nil {
			t.Errorf("Invoke() error = %v, expected nil", err)
		}
		if !called {
			t.Error("Validator function was not called")
		}
	})

	t.Run("invoke failing validator", func(t *testing.T) {
		vr := NewValidatorRegistry()
		_ = vr.Register("fail-validator", func(req Request, policy Policy) error {
			return errors.New("validation failed")
		})

		err := vr.Invoke("fail-validator", Request{}, Policy{})
		if err == nil {
			t.Error("Expected error from failing validator, got nil")
		}
		if !strings.Contains(err.Error(), "validation failed") {
			t.Errorf("Expected 'validation failed' error, got: %v", err)
		}
	})

	t.Run("invoke non-existent validator", func(t *testing.T) {
		vr := NewValidatorRegistry()
		err := vr.Invoke("non-existent", Request{}, Policy{})
		if err == nil {
			t.Error("Expected error for non-existent validator, got nil")
		}
		if !strings.Contains(err.Error(), "not found") {
			t.Errorf("Expected 'not found' error, got: %v", err)
		}
	})

	t.Run("invoke validator with request and policy data", func(t *testing.T) {
		vr := NewValidatorRegistry()
		var capturedReq Request
		var capturedPolicy Policy

		_ = vr.Register("data-validator", func(req Request, policy Policy) error { // Test setup
			capturedReq = req
			capturedPolicy = policy
			return nil
		})

		testReq := Request{
			Subject:  "test-user",
			Action:   "read",
			Resource: "test-resource",
			Context:  map[string]string{"key": "value"},
		}
		testPolicy := Policy{
			ID:      "test-policy",
			Subject: "test-user",
		}

		err := vr.Invoke("data-validator", testReq, testPolicy)
		if err != nil {
			t.Errorf("Invoke() error = %v", err)
		}
		if capturedReq.Subject != "test-user" || capturedPolicy.ID != "test-policy" {
			t.Error("Validator did not receive correct request and policy data")
		}
	})

	t.Run("invoke validator with timeout - success", func(t *testing.T) {
		vr := NewValidatorRegistry()
		_ = vr.Register("fast-validator", func(req Request, policy Policy) error { // Test setup
			time.Sleep(10 * time.Millisecond)
			return nil
		}, WithTimeout(100*time.Millisecond))

		err := vr.Invoke("fast-validator", Request{}, Policy{})
		if err != nil {
			t.Errorf("Invoke() error = %v, expected nil for fast validator", err)
		}
	})

	t.Run("invoke validator with timeout - timeout occurs", func(t *testing.T) {
		vr := NewValidatorRegistry()
		_ = vr.Register("slow-validator", func(req Request, policy Policy) error { // Test setup
			time.Sleep(200 * time.Millisecond)
			return nil
		}, WithTimeout(50*time.Millisecond))

		err := vr.Invoke("slow-validator", Request{}, Policy{})
		if err == nil {
			t.Error("Expected timeout error, got nil")
		}
		if !strings.Contains(err.Error(), "timeout") {
			t.Errorf("Expected 'timeout' error, got: %v", err)
		}
	})

	t.Run("invoke validator without timeout", func(t *testing.T) {
		vr := NewValidatorRegistry()
		_ = vr.Register("no-timeout-validator", func(req Request, policy Policy) error {
			time.Sleep(10 * time.Millisecond)
			return nil
		}) // No timeout set

		err := vr.Invoke("no-timeout-validator", Request{}, Policy{})
		if err != nil {
			t.Errorf("Invoke() error = %v, expected nil", err)
		}
	})
}

// TestValidatorRegistry_Snapshot tests metrics snapshot
func TestValidatorRegistry_Snapshot(t *testing.T) {
	t.Run("snapshot empty registry", func(t *testing.T) {
		vr := NewValidatorRegistry()
		snapshot := vr.Snapshot()
		if len(snapshot) != 0 {
			t.Errorf("Expected empty snapshot, got %d entries", len(snapshot))
		}
	})

	t.Run("snapshot with one validator - no invocations", func(t *testing.T) {
		vr := NewValidatorRegistry()
		_ = vr.Register("test-validator", func(req Request, policy Policy) error {
			return nil
		}, WithDescription("Test validator"), WithVersion("1.0.0"), WithTags("tag1", "tag2"))

		snapshot := vr.Snapshot()
		if len(snapshot) != 1 {
			t.Errorf("Expected 1 entry in snapshot, got %d", len(snapshot))
		}
		if snapshot[0].ID != "test-validator" {
			t.Errorf("Expected ID 'test-validator', got %s", snapshot[0].ID)
		}
		if snapshot[0].Description != "Test validator" {
			t.Errorf("Expected description 'Test validator', got %s", snapshot[0].Description)
		}
		if snapshot[0].Version != "1.0.0" {
			t.Errorf("Expected version '1.0.0', got %s", snapshot[0].Version)
		}
		if len(snapshot[0].Tags) != 2 {
			t.Errorf("Expected 2 tags, got %d", len(snapshot[0].Tags))
		}
		if snapshot[0].Invocations != 0 || snapshot[0].Failures != 0 {
			t.Errorf("Expected 0 invocations and failures, got %d and %d", snapshot[0].Invocations, snapshot[0].Failures)
		}
	})

	t.Run("snapshot with invocations and failures", func(t *testing.T) {
		vr := NewValidatorRegistry()
		callCount := 0
		_ = vr.Register("success-fail-validator", func(req Request, policy Policy) error {
			callCount++
			if callCount%2 == 0 {
				return errors.New("fail")
			}
			return nil
		})

		// Invoke 5 times: 3 successes, 2 failures
		for i := 0; i < 5; i++ {
			_ = vr.Invoke("success-fail-validator", Request{}, Policy{})
		}

		snapshot := vr.Snapshot()
		if len(snapshot) != 1 {
			t.Fatalf("Expected 1 entry in snapshot, got %d", len(snapshot))
		}
		if snapshot[0].Invocations != 5 {
			t.Errorf("Expected 5 invocations, got %d", snapshot[0].Invocations)
		}
		if snapshot[0].Failures != 2 {
			t.Errorf("Expected 2 failures, got %d", snapshot[0].Failures)
		}
	})

	t.Run("snapshot with multiple validators", func(t *testing.T) {
		vr := NewValidatorRegistry()
		_ = vr.Register("validator1", func(req Request, policy Policy) error {
			return nil
		})
		_ = vr.Register("validator2", func(req Request, policy Policy) error {
			return errors.New("fail")
		})
		_ = vr.Register("validator3", func(req Request, policy Policy) error {
			return nil
		})

		// Invoke each once
		_ = vr.Invoke("validator1", Request{}, Policy{})
		_ = vr.Invoke("validator2", Request{}, Policy{})
		_ = vr.Invoke("validator3", Request{}, Policy{})

		snapshot := vr.Snapshot()
		if len(snapshot) != 3 {
			t.Errorf("Expected 3 entries in snapshot, got %d", len(snapshot))
		}

		// Check that all validators are in the snapshot
		ids := make(map[string]bool)
		for _, vm := range snapshot {
			ids[vm.ID] = true
		}
		if !ids["validator1"] || !ids["validator2"] || !ids["validator3"] {
			t.Error("Not all validators present in snapshot")
		}
	})

	t.Run("snapshot with latency histogram", func(t *testing.T) {
		vr := NewValidatorRegistry()
		_ = vr.Register("latency-validator", func(req Request, policy Policy) error {
			time.Sleep(1 * time.Millisecond)
			return nil
		})

		// Invoke a few times to populate histogram
		for i := 0; i < 3; i++ {
			_ = vr.Invoke("latency-validator", Request{}, Policy{})
		}

		snapshot := vr.Snapshot()
		if len(snapshot) != 1 {
			t.Fatalf("Expected 1 entry in snapshot, got %d", len(snapshot))
		}
		if len(snapshot[0].LatencyHistogram) == 0 {
			t.Error("Expected non-empty latency histogram")
		}
	})
}

// TestValidatorRegistry_Metadata tests metadata options
func TestValidatorRegistry_Metadata(t *testing.T) {
	vr := NewValidatorRegistry()

	// Register validator with all metadata
	_ = vr.Register("full-metadata", func(req Request, policy Policy) error {
		return nil
	}, WithDescription("Full metadata validator"), WithVersion("3.1.4"), WithTags("tag-a", "tag-b", "tag-c"))

	snapshot := vr.Snapshot()
	if len(snapshot) != 1 {
		t.Fatalf("Expected 1 validator, got %d", len(snapshot))
	}

	vm := snapshot[0]
	if vm.Description != "Full metadata validator" {
		t.Errorf("Description = %q, expected 'Full metadata validator'", vm.Description)
	}
	if vm.Version != "3.1.4" {
		t.Errorf("Version = %q, expected '3.1.4'", vm.Version)
	}
	if len(vm.Tags) != 3 {
		t.Errorf("Expected 3 tags, got %d", len(vm.Tags))
	}
	expectedTags := map[string]bool{"tag-a": true, "tag-b": true, "tag-c": true}
	for _, tag := range vm.Tags {
		if !expectedTags[tag] {
			t.Errorf("Unexpected tag: %s", tag)
		}
	}
}

// TestValidatorRegistry_Concurrency tests concurrent access
func TestValidatorRegistry_Concurrency(t *testing.T) {
	vr := NewValidatorRegistry()
	_ = vr.Register("concurrent-validator", func(req Request, policy Policy) error {
		time.Sleep(1 * time.Millisecond)
		return nil
	})

	// Run concurrent invocations
	const goroutines = 10
	const invocationsPerGoroutine = 10
	done := make(chan bool, goroutines)

	for i := 0; i < goroutines; i++ {
		go func() {
			for j := 0; j < invocationsPerGoroutine; j++ {
				_ = vr.Invoke("concurrent-validator", Request{}, Policy{})
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < goroutines; i++ {
		<-done
	}

	snapshot := vr.Snapshot()
	if len(snapshot) != 1 {
		t.Fatalf("Expected 1 validator, got %d", len(snapshot))
	}
	expectedInvocations := uint64(goroutines * invocationsPerGoroutine)
	if snapshot[0].Invocations != expectedInvocations {
		t.Errorf("Expected %d invocations, got %d", expectedInvocations, snapshot[0].Invocations)
	}
}

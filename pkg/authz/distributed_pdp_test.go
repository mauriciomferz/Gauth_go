package authz

import (
	"testing"

	"github.com/mauriciomferz/AgentAuth/pkg/policy"
)

// MockEngine implements policy.Engine for testing
type MockEngine struct {
	CalledCount int
	LastInput   policy.AuthzInput
	Result      policy.Decision
	Err         error
}

func (m *MockEngine) EvaluateAuthorization(input policy.AuthzInput) (policy.Decision, error) {
	m.CalledCount++
	m.LastInput = input
	return m.Result, m.Err
}

func (m *MockEngine) EvaluateDelegation(input policy.DelegationInput) (policy.Decision, error) {
	return policy.Decision{}, nil
}

func (m *MockEngine) Reload() error {
	return nil
}

func TestDistributedPDP_Decide(t *testing.T) {
	mockEngine := &MockEngine{
		Result: policy.Decision{
			Allow:      true,
			ReasonCode: "test-allow",
			Metadata:   map[string]string{"version": "1"},
		},
	}

	pdp := NewDefaultDistributedPDP(mockEngine)
	req := map[string]interface{}{
		"subject":  "user:alice",
		"action":   "read",
		"resource": "doc:1",
		"scopes":   []string{"read:all"},
		"context": map[string]interface{}{
			"ip": "127.0.0.1",
		},
	}

	// First call - should hit engine
	dec, err := pdp.Decide(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !dec.Allow {
		t.Error("expected allow")
	}
	if mockEngine.CalledCount != 1 {
		t.Errorf("expected engine called 1 time, got %d", mockEngine.CalledCount)
	}

	// Verify input mapping
	if mockEngine.LastInput.Subject != "user:alice" {
		t.Errorf("expected subject user:alice, got %s", mockEngine.LastInput.Subject)
	}
	if mockEngine.LastInput.Attributes["ip"] != "127.0.0.1" {
		t.Errorf("expected ip attribute 127.0.0.1, got %v", mockEngine.LastInput.Attributes["ip"])
	}

	// Second call - should hit cache
	dec, err = pdp.Decide(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !dec.Allow {
		t.Error("expected allow from cache")
	}
	if mockEngine.CalledCount != 1 {
		t.Errorf("expected engine called 1 time (cache hit), got %d", mockEngine.CalledCount)
	}

	// Different request - should hit engine
	req2 := map[string]interface{}{
		"subject":  "user:bob",
		"action":   "read",
		"resource": "doc:1",
	}
	pdp.Decide(req2)
	if mockEngine.CalledCount != 2 {
		t.Errorf("expected engine called 2 times, got %d", mockEngine.CalledCount)
	}
}

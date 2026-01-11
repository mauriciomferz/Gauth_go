package pdp

import (
	"context"
	"testing"
	"time"
)

// TestEngine_AdviceEmissionForNonMandatoryObligations verifies that non-mandatory obligations
// trigger advice events during policy evaluation (P2.1 / sec2.item3).
func TestEngine_AdviceEmissionForNonMandatoryObligations(t *testing.T) {
	// Setup: Create advice channel
	adviceChannel := NewBufferedAdviceChannel(10)
	defer func() { _ = adviceChannel.Close() }()

	// Setup: Create engine with advice channel and obligation executor
	executor := NewExtendedObligationExecutor(WithAdviceChannel(adviceChannel))
	engine := NewInMemoryEngine(DenyOverridesStrategy{}).
		WithAdviceChannel(adviceChannel).
		WithObligations(executor, "")

	// Setup: Add policy with non-mandatory obligations
	policy := Policy{
		ID:       "policy_advice_test",
		Subjects: []string{"alice"},
		Rules: []Rule{
			{
				ID:        "rule_advice_test",
				Actions:   []string{"read"},
				Resources: []string{"doc:123"},
				Effect:    "allow",
			},
		},
		Obligations: []Obligation{
			{
				ID:        "log:user_read_action",
				Mandatory: false, // Non-mandatory obligation should emit advice
				Attributes: map[string]string{
					"type": "audit_log",
				},
			},
			{
				ID:        "notify:admin_alert",
				Mandatory: false,
				Attributes: map[string]string{
					"type": "notification",
				},
			},
		},
	}
	engine.AddPolicy(policy)

	// Execute: Evaluate request that matches policy
	req := Request{
		Subject:  "alice",
		Action:   "read",
		Resource: "doc:123",
		Time:     time.Now(),
	}

	decision, err := engine.Evaluate(context.Background(), req)
	if err != nil {
		t.Fatalf("evaluation failed: %v", err)
	}

	// Verify: Decision should be Allow
	if !decision.Allow {
		t.Error("expected Allow decision for matching policy")
	}

	// Verify: Should have 2 obligations
	if len(decision.Obligations) != 2 {
		t.Fatalf("expected 2 obligations, got %d", len(decision.Obligations))
	}

	// Verify: Should receive 2 advice events (one per non-mandatory obligation)
	receivedAdvice := make([]AdviceEvent, 0, 2)
	timeout := time.After(2 * time.Second)

	for i := 0; i < 2; i++ {
		select {
		case advice := <-adviceChannel.AdviceEvents():
			receivedAdvice = append(receivedAdvice, advice)
		case <-timeout:
			t.Fatalf("timeout waiting for advice event %d", i+1)
		}
	}

	// Verify: Advice events have correct context
	if len(receivedAdvice) != 2 {
		t.Fatalf("expected 2 advice events, got %d", len(receivedAdvice))
	}

	for _, advice := range receivedAdvice {
		if advice.Subject != "alice" {
			t.Errorf("expected subject 'alice', got '%s'", advice.Subject)
		}
		if advice.Action != "read" {
			t.Errorf("expected action 'read', got '%s'", advice.Action)
		}
		if advice.Resource != "doc:123" {
			t.Errorf("expected resource 'doc:123', got '%s'", advice.Resource)
		}
		if advice.AdviceID == "" {
			t.Error("advice ID should not be empty")
		}
		if advice.Message == "" {
			t.Error("advice message should not be empty")
		}
		// Verify metadata
		if advice.Metadata["success"] == "" {
			t.Error("advice metadata should contain 'success' field")
		}
		if advice.Metadata["duration_ms"] == "" {
			t.Error("advice metadata should contain 'duration_ms' field")
		}
	}
}

// TestEngine_NoAdviceEmissionForMandatoryObligations verifies that mandatory obligations
// do NOT trigger advice events (only non-mandatory obligations emit advice).
func TestEngine_NoAdviceEmissionForMandatoryObligations(t *testing.T) {
	// Setup: Create advice channel
	adviceChannel := NewBufferedAdviceChannel(10)
	t.Cleanup(func() { _ = adviceChannel.Close() })

	// Setup: Create engine with advice channel and obligation executor
	executor := NewExtendedObligationExecutor(WithAdviceChannel(adviceChannel))
	engine := NewInMemoryEngine(DenyOverridesStrategy{}).
		WithAdviceChannel(adviceChannel).
		WithObligations(executor, "")

	// Setup: Add policy with MANDATORY obligations
	policy := Policy{
		ID:       "policy_mandatory_test",
		Subjects: []string{"bob"},
		Rules: []Rule{
			{
				ID:        "rule_mandatory_test",
				Actions:   []string{"write"},
				Resources: []string{"doc:456"},
				Effect:    "allow",
			},
		},
		Obligations: []Obligation{
			{
				ID:        "log:mandatory_write",
				Mandatory: true, // Mandatory obligation should NOT emit advice
				Attributes: map[string]string{
					"type": "audit_log",
				},
			},
		},
	}
	engine.AddPolicy(policy)

	// Execute: Evaluate request that matches policy
	req := Request{
		Subject:  "bob",
		Action:   "write",
		Resource: "doc:456",
		Time:     time.Now(),
	}

	decision, err := engine.Evaluate(context.Background(), req)
	if err != nil {
		t.Fatalf("evaluation failed: %v", err)
	}

	// Verify: Decision should be Allow
	if !decision.Allow {
		t.Error("expected Allow decision for matching policy")
	}

	// Verify: Should have 1 mandatory obligation
	if len(decision.Obligations) != 1 {
		t.Fatalf("expected 1 obligation, got %d", len(decision.Obligations))
	}

	// Verify: Should NOT receive any advice events (mandatory obligations don't emit advice)
	select {
	case advice := <-adviceChannel.AdviceEvents():
		t.Errorf("unexpected advice event for mandatory obligation: %+v", advice)
	case <-time.After(500 * time.Millisecond):
		// Expected: no advice events
	}
}

// TestEngine_MixedMandatoryAndNonMandatoryObligations verifies advice emission
// only for non-mandatory obligations when both types are present.
func TestEngine_MixedMandatoryAndNonMandatoryObligations(t *testing.T) {
	// Setup: Create advice channel
	adviceChannel := NewBufferedAdviceChannel(10)
	t.Cleanup(func() { _ = adviceChannel.Close() })

	// Setup: Create engine with advice channel and obligation executor
	executor := NewExtendedObligationExecutor(WithAdviceChannel(adviceChannel))
	engine := NewInMemoryEngine(DenyOverridesStrategy{}).
		WithAdviceChannel(adviceChannel).
		WithObligations(executor, "")

	// Setup: Add policy with mixed mandatory and non-mandatory obligations
	policy := Policy{
		ID:       "policy_mixed_test",
		Subjects: []string{"charlie"},
		Rules: []Rule{
			{
				ID:        "rule_mixed_test",
				Actions:   []string{"delete"},
				Resources: []string{"doc:789"},
				Effect:    "allow",
			},
		},
		Obligations: []Obligation{
			{
				ID:        "log:mandatory_delete",
				Mandatory: true, // Mandatory - no advice
				Attributes: map[string]string{
					"type": "audit_log",
				},
			},
			{
				ID:        "notify:optional_alert",
				Mandatory: false, // Non-mandatory - should emit advice
				Attributes: map[string]string{
					"type": "notification",
				},
			},
			{
				ID:        "rate_limit:optional_check",
				Mandatory: false, // Non-mandatory - should emit advice
				Attributes: map[string]string{
					"type": "rate_limit",
				},
			},
		},
	}
	engine.AddPolicy(policy)

	// Execute: Evaluate request that matches policy
	req := Request{
		Subject:  "charlie",
		Action:   "delete",
		Resource: "doc:789",
		Time:     time.Now(),
	}

	decision, err := engine.Evaluate(context.Background(), req)
	if err != nil {
		t.Fatalf("evaluation failed: %v", err)
	}

	// Verify: Decision should be Allow
	if !decision.Allow {
		t.Error("expected Allow decision for matching policy")
	}

	// Verify: Should have 3 obligations
	if len(decision.Obligations) != 3 {
		t.Fatalf("expected 3 obligations, got %d", len(decision.Obligations))
	}

	// Verify: Should receive exactly 2 advice events (only non-mandatory obligations)
	receivedAdvice := make([]AdviceEvent, 0, 2)
	timeout := time.After(2 * time.Second)

	for i := 0; i < 2; i++ {
		select {
		case advice := <-adviceChannel.AdviceEvents():
			receivedAdvice = append(receivedAdvice, advice)
		case <-timeout:
			t.Fatalf("timeout waiting for advice event %d", i+1)
		}
	}

	// Verify: Exactly 2 advice events (not 3)
	if len(receivedAdvice) != 2 {
		t.Fatalf("expected 2 advice events (non-mandatory only), got %d", len(receivedAdvice))
	}

	// Verify: No more advice events
	select {
	case advice := <-adviceChannel.AdviceEvents():
		t.Errorf("unexpected extra advice event: %+v", advice)
	case <-time.After(500 * time.Millisecond):
		// Expected: no more advice events
	}
}

// TestEngine_AdviceEmissionWithoutAdviceChannel verifies that the engine
// continues to work correctly when no advice channel is configured.
func TestEngine_AdviceEmissionWithoutAdviceChannel(t *testing.T) {
	// Setup: Create engine WITHOUT advice channel
	executor := NewExtendedObligationExecutor() // No advice channel option
	engine := NewInMemoryEngine(DenyOverridesStrategy{}).
		WithObligations(executor, "")
	// Note: No WithAdviceChannel call

	// Setup: Add policy with non-mandatory obligations
	policy := Policy{
		ID:       "policy_no_channel_test",
		Subjects: []string{"dave"},
		Rules: []Rule{
			{
				ID:        "rule_no_channel_test",
				Actions:   []string{"read"},
				Resources: []string{"doc:999"},
				Effect:    "allow",
			},
		},
		Obligations: []Obligation{
			{
				ID:        "log:no_channel_test",
				Mandatory: false,
			},
		},
	}
	engine.AddPolicy(policy)

	// Execute: Evaluate request
	req := Request{
		Subject:  "dave",
		Action:   "read",
		Resource: "doc:999",
		Time:     time.Now(),
	}

	decision, err := engine.Evaluate(context.Background(), req)
	if err != nil {
		t.Fatalf("evaluation failed: %v", err)
	}

	// Verify: Decision should be Allow (no error despite missing advice channel)
	if !decision.Allow {
		t.Error("expected Allow decision for matching policy")
	}

	// Verify: Should have 1 obligation
	if len(decision.Obligations) != 1 {
		t.Fatalf("expected 1 obligation, got %d", len(decision.Obligations))
	}
}

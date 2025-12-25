package authz

import (
	"context"
	"testing"

	"github.com/mauriciomferz/Gauth_go/pkg/policy"
)

// TestNewAuthorizerAdapter verifies AuthorizerAdapter constructor correctly initializes with provided engine.
func TestNewAuthorizerAdapter(t *testing.T) {
	reg := policy.NewRegistry()
	engine := policy.NewChainEngine(reg)
	adapter := NewAuthorizerAdapter(engine)

	if adapter == nil {
		t.Fatal("NewAuthorizerAdapter returned nil")
	}
	if adapter.engine != engine {
		t.Errorf("expected engine to be set, got %v", adapter.engine)
	}
}

// TestAuthorizerAdapterAuthorize_Allow verifies Authorize method correctly translates and evaluates allowing requests.
func TestAuthorizerAdapterAuthorize_Allow(t *testing.T) {
	// Create a simple bundle that allows "read" on "document:123" for user:alice
	bundle := policy.Bundle{
		ID:      "test-bundle-1",
		Version: 1,
		Policies: []policy.Policy{
			{
				ID:       "policy-allow-alice-read",
				Subjects: []string{"user:alice"},
				Rules: []policy.Rule{
					{
						Actions:   []string{"read"},
						Resources: []string{"document:123"}, // Exact match
						Expr:      "",                       // Always true
						Effect:    policy.Allow,
					},
				},
			},
		},
	}
	reg := policy.NewRegistry()
	_, err := reg.AddBundle(bundle)
	if err != nil {
		t.Fatalf("failed to add bundle: %v", err)
	}

	engine := policy.NewChainEngine(reg)
	adapter := NewAuthorizerAdapter(engine)

	req := Request{
		Subject:  "user:alice",
		Action:   "read",
		Resource: "document:123",
	}

	decision, err := adapter.Authorize(context.Background(), req)
	if err != nil {
		t.Fatalf("Authorize returned error: %v", err)
	}
	if !decision.Allow {
		t.Errorf("expected Allow=true, got Allow=%v", decision.Allow)
	}
	if decision.Reason == "" {
		t.Error("expected non-empty Reason")
	}
}

// TestAuthorizerAdapterAuthorize_Deny verifies Authorize method correctly handles deny decisions.
func TestAuthorizerAdapterAuthorize_Deny(t *testing.T) {
	// Create a bundle with deny policy
	bundle := policy.Bundle{
		ID:      "test-bundle-deny",
		Version: 1,
		Policies: []policy.Policy{
			{
				ID:       "policy-deny-bob",
				Subjects: []string{"user:bob"},
				Rules: []policy.Rule{
					{
						Actions:   []string{"delete"},
						Resources: []string{"document:sensitive"}, // Exact match
						Expr:      "",
						Effect:    policy.Deny,
					},
				},
			},
		},
	}
	reg := policy.NewRegistry()
	_, err := reg.AddBundle(bundle)
	if err != nil {
		t.Fatalf("failed to add bundle: %v", err)
	}

	engine := policy.NewChainEngine(reg)
	adapter := NewAuthorizerAdapter(engine)

	req := Request{
		Subject:  "user:bob",
		Action:   "delete",
		Resource: "document:sensitive",
	}

	decision, err := adapter.Authorize(context.Background(), req)
	if err != nil {
		t.Fatalf("Authorize returned error: %v", err)
	}
	if decision.Allow {
		t.Errorf("expected Allow=false, got Allow=%v", decision.Allow)
	}
	if decision.Reason == "" {
		t.Error("expected non-empty Reason")
	}
}

// TestAuthorizerAdapterAuthorize_NotApplicable verifies Authorize handles non-matching policies.
func TestAuthorizerAdapterAuthorize_NotApplicable(t *testing.T) {
	// Create bundle that doesn't match the request
	bundle := policy.Bundle{
		ID:      "test-bundle-mismatch",
		Version: 1,
		Policies: []policy.Policy{
			{
				ID:       "policy-charlie",
				Subjects: []string{"user:charlie"},
				Rules: []policy.Rule{
					{
						Actions:   []string{"write"},
						Resources: []string{"file:report.txt"}, // Exact match
						Expr:      "",
						Effect:    policy.Allow,
					},
				},
			},
		},
	}
	reg := policy.NewRegistry()
	_, err := reg.AddBundle(bundle)
	if err != nil {
		t.Fatalf("failed to add bundle: %v", err)
	}

	engine := policy.NewChainEngine(reg)
	adapter := NewAuthorizerAdapter(engine)

	// Request from different user/action
	req := Request{
		Subject:  "user:david",
		Action:   "read",
		Resource: "document:public",
	}

	decision, err := adapter.Authorize(context.Background(), req)
	if err != nil {
		t.Fatalf("Authorize returned error: %v", err)
	}
	if decision.Allow {
		t.Errorf("expected Allow=false for non-applicable policy, got Allow=%v", decision.Allow)
	}
}

// TestAuthorizerAdapterAuthorize_EmptyRegistry verifies Authorize handles empty registry correctly.
func TestAuthorizerAdapterAuthorize_EmptyRegistry(t *testing.T) {
	reg := policy.NewRegistry()
	engine := policy.NewChainEngine(reg)
	adapter := NewAuthorizerAdapter(engine)

	req := Request{
		Subject:  "user:anyone",
		Action:   "read",
		Resource: "document:any",
	}

	decision, err := adapter.Authorize(context.Background(), req)
	if err != nil {
		t.Fatalf("Authorize returned error: %v", err)
	}
	if decision.Allow {
		t.Errorf("expected Allow=false for empty registry, got Allow=%v", decision.Allow)
	}
}

// TestAuthorizerAdapterAuthorize_ContextAttributes verifies context attributes are passed to evaluation.
func TestAuthorizerAdapterAuthorize_ContextAttributes(t *testing.T) {
	// Create bundle with conditional rule based on context
	bundle := policy.Bundle{
		ID:      "test-bundle-context",
		Version: 1,
		Policies: []policy.Policy{
			{
				ID:       "policy-context-aware",
				Subjects: []string{"user:eve"},
				Rules: []policy.Rule{
					{
						Actions:   []string{"read"},
						Resources: []string{"document:secret"}, // Exact match
						Expr:      "ip_address == '192.168.1.1'",
						Effect:    policy.Allow,
					},
				},
			},
		},
	}
	reg := policy.NewRegistry()
	_, err := reg.AddBundle(bundle)
	if err != nil {
		t.Fatalf("failed to add bundle: %v", err)
	}

	engine := policy.NewChainEngine(reg)
	adapter := NewAuthorizerAdapter(engine)

	// Request with matching context
	req := Request{
		Subject:  "user:eve",
		Action:   "read",
		Resource: "document:secret",
		Context: map[string]string{
			"ip_address": "192.168.1.1",
		},
	}

	decision, err := adapter.Authorize(context.Background(), req)
	if err != nil {
		t.Fatalf("Authorize returned error: %v", err)
	}
	if !decision.Allow {
		t.Errorf("expected Allow=true with matching context, got Allow=%v", decision.Allow)
	}

	// Request with non-matching context
	req.Context = map[string]string{
		"ip_address": "10.0.0.1",
	}
	decision, err = adapter.Authorize(context.Background(), req)
	if err != nil {
		t.Fatalf("Authorize returned error: %v", err)
	}
	if decision.Allow {
		t.Errorf("expected Allow=false with non-matching context, got Allow=%v", decision.Allow)
	}
}

// TestAuthorizerAdapterAuthorize_DenyOverrides verifies deny-overrides semantics work correctly.
func TestAuthorizerAdapterAuthorize_DenyOverrides(t *testing.T) {
	// Create bundle with both allow and deny policies - deny should win
	bundle := policy.Bundle{
		ID:      "test-bundle-deny-wins",
		Version: 1,
		Policies: []policy.Policy{
			{
				ID:       "policy-allow-frank",
				Subjects: []string{"user:frank"},
				Rules: []policy.Rule{
					{
						Actions:   []string{"read"},
						Resources: []string{"*"}, // Wildcard
						Expr:      "",
						Effect:    policy.Allow,
					},
				},
			},
			{
				ID:       "policy-deny-frank-sensitive",
				Subjects: []string{"user:frank"},
				Rules: []policy.Rule{
					{
						Actions:   []string{"read"},
						Resources: []string{"document:sensitive"}, // Exact match
						Expr:      "",
						Effect:    policy.Deny,
					},
				},
			},
		},
	}
	reg := policy.NewRegistry()
	_, err := reg.AddBundle(bundle)
	if err != nil {
		t.Fatalf("failed to add bundle: %v", err)
	}

	engine := policy.NewChainEngine(reg)
	adapter := NewAuthorizerAdapter(engine)

	req := Request{
		Subject:  "user:frank",
		Action:   "read",
		Resource: "document:sensitive",
	}

	decision, err := adapter.Authorize(context.Background(), req)
	if err != nil {
		t.Fatalf("Authorize returned error: %v", err)
	}
	if decision.Allow {
		t.Errorf("expected Allow=false (deny overrides), got Allow=%v", decision.Allow)
	}
}

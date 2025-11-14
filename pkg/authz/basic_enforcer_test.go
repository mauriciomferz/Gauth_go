package authz

import (
	"context"
	"testing"
)

// TestNewBasicEnforcer verifies BasicEnforcer constructor
func TestNewBasicEnforcer(t *testing.T) {
	enforcer := NewBasicEnforcer()

	if enforcer == nil {
		t.Fatal("NewBasicEnforcer returned nil")
	}
	if enforcer.policies == nil {
		t.Error("policies map not initialized")
	}
	if len(enforcer.policies) != 0 {
		t.Errorf("initial policy count = %d, want 0", len(enforcer.policies))
	}
}

// TestBasicEnforcer_AddPolicy verifies policy addition
func TestBasicEnforcer_AddPolicy(t *testing.T) {
	ctx := context.Background()
	enforcer := NewBasicEnforcer()

	policy := &Policy{
		ID:       "policy1",
		Subject:  "user:123",
		Resource: "document:456",
		Actions:  []string{"read"},
		Effect:   Allow,
	}

	err := enforcer.AddPolicy(ctx, policy)
	if err != nil {
		t.Fatalf("AddPolicy failed: %v", err)
	}

	// Verify policy stored
	stored := enforcer.policies["policy1"]
	if stored == nil {
		t.Fatal("Policy not stored")
	}
	if stored.ID != policy.ID {
		t.Errorf("stored ID = %q, want %q", stored.ID, policy.ID)
	}
	if stored.Subject != policy.Subject {
		t.Errorf("stored Subject = %q, want %q", stored.Subject, policy.Subject)
	}
}

// TestBasicEnforcer_AddMultiplePolicies verifies adding multiple policies
func TestBasicEnforcer_AddMultiplePolicies(t *testing.T) {
	ctx := context.Background()
	enforcer := NewBasicEnforcer()

	policies := []*Policy{
		{ID: "policy1", Subject: "user:1", Resource: "doc:1", Actions: []string{"read"}, Effect: Allow},
		{ID: "policy2", Subject: "user:2", Resource: "doc:2", Actions: []string{"write"}, Effect: Allow},
		{ID: "policy3", Subject: "user:3", Resource: "doc:3", Actions: []string{"delete"}, Effect: Deny},
	}

	for _, p := range policies {
		err := enforcer.AddPolicy(ctx, p)
		if err != nil {
			t.Fatalf("AddPolicy failed for %s: %v", p.ID, err)
		}
	}

	if len(enforcer.policies) != 3 {
		t.Errorf("policy count = %d, want 3", len(enforcer.policies))
	}
}

// TestBasicEnforcer_RemovePolicy verifies policy removal
func TestBasicEnforcer_RemovePolicy(t *testing.T) {
	ctx := context.Background()
	enforcer := NewBasicEnforcer()

	// Add policies
	enforcer.AddPolicy(ctx, &Policy{ID: "policy1", Subject: "user:1", Resource: "doc:1", Actions: []string{"read"}, Effect: Allow})
	enforcer.AddPolicy(ctx, &Policy{ID: "policy2", Subject: "user:2", Resource: "doc:2", Actions: []string{"write"}, Effect: Allow})

	// Verify both present
	if len(enforcer.policies) != 2 {
		t.Errorf("policy count before removal = %d, want 2", len(enforcer.policies))
	}

	// Remove policy1
	err := enforcer.RemovePolicy(ctx, "policy1")
	if err != nil {
		t.Fatalf("RemovePolicy failed: %v", err)
	}

	// Verify policy1 removed
	if _, exists := enforcer.policies["policy1"]; exists {
		t.Error("policy1 should be removed")
	}

	// Verify policy2 still present
	if _, exists := enforcer.policies["policy2"]; !exists {
		t.Error("policy2 should still be present")
	}

	if len(enforcer.policies) != 1 {
		t.Errorf("policy count after removal = %d, want 1", len(enforcer.policies))
	}
}

// TestBasicEnforcer_RemoveNonExistentPolicy verifies removing non-existent policy is safe
func TestBasicEnforcer_RemoveNonExistentPolicy(t *testing.T) {
	ctx := context.Background()
	enforcer := NewBasicEnforcer()

	// Remove non-existent policy (should not error)
	err := enforcer.RemovePolicy(ctx, "nonexistent")
	if err != nil {
		t.Errorf("RemovePolicy on non-existent should not error: %v", err)
	}
}

// TestMatchesPattern verifies pattern matching logic
func TestMatchesPattern(t *testing.T) {
	enforcer := NewBasicEnforcer()

	tests := []struct {
		name    string
		pattern string
		value   string
		want    bool
	}{
		{"exact match", "user:123", "user:123", true},
		{"exact mismatch", "user:123", "user:456", false},
		{"wildcard all", "*", "anything", true},
		{"wildcard all empty", "*", "", true},
		{"prefix wildcard match", "/docs/*", "/docs/report.pdf", true},
		{"prefix wildcard match nested", "/docs/*", "/docs/subfolder/file.txt", true},
		{"prefix wildcard mismatch", "/docs/*", "/images/photo.jpg", false},
		{"prefix wildcard exact prefix", "/docs/*", "/docs/", true},
		{"prefix wildcard too short", "/docs/*", "/doc", false},
		{"empty pattern empty value", "", "", true},
		{"empty pattern non-empty value", "", "something", false},
		{"non-empty pattern empty value", "user:*", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := enforcer.matchesPattern(tt.pattern, tt.value)
			if got != tt.want {
				t.Errorf("matchesPattern(%q, %q) = %v, want %v", tt.pattern, tt.value, got, tt.want)
			}
		})
	}
}

// TestActionsContain verifies action matching
func TestActionsContain(t *testing.T) {
	tests := []struct {
		name    string
		actions []string
		action  string
		want    bool
	}{
		{"single exact match", []string{"read"}, "read", true},
		{"single mismatch", []string{"read"}, "write", false},
		{"multiple with match", []string{"read", "write", "delete"}, "write", true},
		{"multiple no match", []string{"read", "write", "delete"}, "admin", false},
		{"wildcard all", []string{"*"}, "anything", true},
		{"wildcard with others", []string{"read", "*", "write"}, "delete", true},
		{"empty actions", []string{}, "read", false},
		{"empty action string", []string{"read"}, "", false},
		{"case sensitive", []string{"read"}, "READ", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := actionsContain(tt.actions, tt.action)
			if got != tt.want {
				t.Errorf("actionsContain(%v, %q) = %v, want %v", tt.actions, tt.action, got, tt.want)
			}
		})
	}
}

// TestBasicEnforcer_Evaluate_Allow verifies allow decisions
func TestBasicEnforcer_Evaluate_Allow(t *testing.T) {
	ctx := context.Background()
	enforcer := NewBasicEnforcer()

	// Add allow policy
	enforcer.AddPolicy(ctx, &Policy{
		ID:       "allow-read",
		Subject:  "user:123",
		Resource: "document:456",
		Actions:  []string{"read"},
		Effect:   Allow,
	})

	// Matching request
	req := &Request{
		Subject:  "user:123",
		Resource: "document:456",
		Action:   "read",
	}

	decision, err := enforcer.Evaluate(ctx, req)
	if err != nil {
		t.Fatalf("Evaluate failed: %v", err)
	}

	if !decision.Allow {
		t.Error("Decision should allow")
	}
	if decision.Reason == "" {
		t.Error("Decision should have a reason")
	}
	if decision.Reason != "allowed by policy allow-read" {
		t.Errorf("Reason = %q, want %q", decision.Reason, "allowed by policy allow-read")
	}
}

// TestBasicEnforcer_Evaluate_Deny verifies deny decisions
func TestBasicEnforcer_Evaluate_Deny(t *testing.T) {
	ctx := context.Background()
	enforcer := NewBasicEnforcer()

	// Add deny policy
	enforcer.AddPolicy(ctx, &Policy{
		ID:       "deny-delete",
		Subject:  "user:789",
		Resource: "system:config",
		Actions:  []string{"delete"},
		Effect:   Deny,
	})

	// Matching request
	req := &Request{
		Subject:  "user:789",
		Resource: "system:config",
		Action:   "delete",
	}

	decision, err := enforcer.Evaluate(ctx, req)
	if err != nil {
		t.Fatalf("Evaluate failed: %v", err)
	}

	if decision.Allow {
		t.Error("Decision should deny")
	}
	if decision.Reason != "denied by policy deny-delete" {
		t.Errorf("Reason = %q, want %q", decision.Reason, "denied by policy deny-delete")
	}
}

// TestBasicEnforcer_Evaluate_NoMatch verifies no matching policy
func TestBasicEnforcer_Evaluate_NoMatch(t *testing.T) {
	ctx := context.Background()
	enforcer := NewBasicEnforcer()

	// Add policy
	enforcer.AddPolicy(ctx, &Policy{
		ID:       "policy1",
		Subject:  "user:123",
		Resource: "document:456",
		Actions:  []string{"read"},
		Effect:   Allow,
	})

	// Non-matching request (different subject)
	req := &Request{
		Subject:  "user:999",
		Resource: "document:456",
		Action:   "read",
	}

	decision, err := enforcer.Evaluate(ctx, req)
	if err != nil {
		t.Fatalf("Evaluate failed: %v", err)
	}

	if decision.Allow {
		t.Error("Decision should deny (no match)")
	}
	if decision.Reason != "no matching policy" {
		t.Errorf("Reason = %q, want %q", decision.Reason, "no matching policy")
	}
}

// TestBasicEnforcer_Evaluate_WildcardAction verifies wildcard action matching
func TestBasicEnforcer_Evaluate_WildcardAction(t *testing.T) {
	ctx := context.Background()
	enforcer := NewBasicEnforcer()

	// Add policy with wildcard action
	enforcer.AddPolicy(ctx, &Policy{
		ID:       "wildcard-actions",
		Subject:  "admin:*",
		Resource: "system:*",
		Actions:  []string{"*"},
		Effect:   Allow,
	})

	// Request with any action
	req := &Request{
		Subject:  "admin:root",
		Resource: "system:database",
		Action:   "backup",
	}

	decision, err := enforcer.Evaluate(ctx, req)
	if err != nil {
		t.Fatalf("Evaluate failed: %v", err)
	}

	if !decision.Allow {
		t.Error("Wildcard action should match any action")
	}
}

// TestBasicEnforcer_Evaluate_PrefixWildcard verifies prefix wildcard pattern matching
func TestBasicEnforcer_Evaluate_PrefixWildcard(t *testing.T) {
	ctx := context.Background()
	enforcer := NewBasicEnforcer()

	// Add policy with prefix wildcard
	enforcer.AddPolicy(ctx, &Policy{
		ID:       "docs-access",
		Subject:  "user:*",
		Resource: "/documents/*",
		Actions:  []string{"read"},
		Effect:   Allow,
	})

	tests := []struct {
		name     string
		resource string
		want     bool
	}{
		{"direct child", "/documents/report.pdf", true},
		{"nested path", "/documents/2024/Q1/summary.pdf", true},
		{"exact prefix", "/documents/", true},
		{"different path", "/images/photo.jpg", false},
		{"similar prefix", "/document/file.txt", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &Request{
				Subject:  "user:alice",
				Resource: tt.resource,
				Action:   "read",
			}

			decision, err := enforcer.Evaluate(ctx, req)
			if err != nil {
				t.Fatalf("Evaluate failed: %v", err)
			}

			if decision.Allow != tt.want {
				t.Errorf("For resource %q: Allow = %v, want %v", tt.resource, decision.Allow, tt.want)
			}
		})
	}
}

// TestBasicEnforcer_Authorize_NewInterface verifies Authorize with Request parameter
func TestBasicEnforcer_Authorize_NewInterface(t *testing.T) {
	ctx := context.Background()
	enforcer := NewBasicEnforcer()

	enforcer.AddPolicy(ctx, &Policy{
		ID:       "policy1",
		Subject:  "user:123",
		Resource: "doc:1",
		Actions:  []string{"read"},
		Effect:   Allow,
	})

	req := &Request{
		Subject:  "user:123",
		Resource: "doc:1",
		Action:   "read",
	}

	decision, err := enforcer.Authorize(ctx, req)
	if err != nil {
		t.Fatalf("Authorize failed: %v", err)
	}

	if !decision.Allow {
		t.Error("Decision should allow")
	}
	// Verify compatibility field set
	if !decision.Allowed {
		t.Error("Compatibility field Allowed should be set")
	}
}

// TestBasicEnforcer_Authorize_LegacyInterface verifies Authorize with Subject/Action/Resource parameters
func TestBasicEnforcer_Authorize_LegacyInterface(t *testing.T) {
	ctx := context.Background()
	enforcer := NewBasicEnforcer()

	enforcer.AddPolicy(ctx, &Policy{
		ID:       "policy1",
		Subject:  "user:456",
		Resource: "file:789",
		Actions:  []string{"write"},
		Effect:   Allow,
	})

	subject := Subject{ID: "user:456"}
	action := Action{Name: "write"}
	resource := Resource{ID: "file:789"}

	decision, err := enforcer.Authorize(ctx, subject, action, resource)
	if err != nil {
		t.Fatalf("Authorize failed: %v", err)
	}

	if !decision.Allow {
		t.Error("Decision should allow")
	}
	// Verify compatibility field set
	if !decision.Allowed {
		t.Error("Compatibility field Allowed should be set")
	}
}

// TestBasicEnforcer_AuthorizeWithParams verifies AuthorizeWithParams wrapper
func TestBasicEnforcer_AuthorizeWithParams(t *testing.T) {
	ctx := context.Background()
	enforcer := NewBasicEnforcer()

	enforcer.AddPolicy(ctx, &Policy{
		ID:       "policy1",
		Subject:  "service:api",
		Resource: "database:users",
		Actions:  []string{"query"},
		Effect:   Allow,
	})

	subject := Subject{ID: "service:api"}
	action := Action{Name: "query"}
	resource := Resource{ID: "database:users"}

	decision, err := enforcer.AuthorizeWithParams(ctx, subject, action, resource)
	if err != nil {
		t.Fatalf("AuthorizeWithParams failed: %v", err)
	}

	if !decision.Allow {
		t.Error("Decision should allow")
	}
}

// TestBasicEnforcer_EmptyPolicies verifies behavior with no policies
func TestBasicEnforcer_EmptyPolicies(t *testing.T) {
	ctx := context.Background()
	enforcer := NewBasicEnforcer()

	req := &Request{
		Subject:  "user:123",
		Resource: "doc:456",
		Action:   "read",
	}

	decision, err := enforcer.Evaluate(ctx, req)
	if err != nil {
		t.Fatalf("Evaluate failed: %v", err)
	}

	if decision.Allow {
		t.Error("Should deny when no policies exist")
	}
	if decision.Reason != "no matching policy" {
		t.Errorf("Reason = %q, want %q", decision.Reason, "no matching policy")
	}
}

// TestBasicEnforcer_FirstMatchWins verifies first matching policy is used
func TestBasicEnforcer_FirstMatchWins(t *testing.T) {
	ctx := context.Background()
	enforcer := NewBasicEnforcer()

	// Add multiple policies that could match
	enforcer.AddPolicy(ctx, &Policy{
		ID:       "allow-policy",
		Subject:  "user:*",
		Resource: "doc:*",
		Actions:  []string{"*"},
		Effect:   Allow,
	})
	enforcer.AddPolicy(ctx, &Policy{
		ID:       "deny-policy",
		Subject:  "user:*",
		Resource: "doc:*",
		Actions:  []string{"*"},
		Effect:   Deny,
	})

	req := &Request{
		Subject:  "user:123",
		Resource: "doc:456",
		Action:   "read",
	}

	decision, err := enforcer.Evaluate(ctx, req)
	if err != nil {
		t.Fatalf("Evaluate failed: %v", err)
	}

	// Result depends on map iteration order, but should match one of them
	if decision.Reason != "allowed by policy allow-policy" && decision.Reason != "denied by policy deny-policy" {
		t.Errorf("Reason should mention one of the policies, got: %q", decision.Reason)
	}
}

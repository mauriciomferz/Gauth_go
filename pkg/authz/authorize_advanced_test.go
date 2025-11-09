package authz

import (
	"context"
	"fmt"
	"testing"
	"time"
)

// TestMatchesPolicy tests the matchesPolicy function comprehensively
func TestMatchesPolicy(t *testing.T) {
	ma := NewMemoryAuthorizer()

	tests := []struct {
		name     string
		request  Request
		policy   Policy
		expected bool
	}{
		{
			name: "simple match - all fields match",
			request: Request{
				Subject:  "user1",
				Action:   "read",
				Resource: "file.txt",
				Context:  map[string]string{},
			},
			policy: Policy{
				ID:       "policy1",
				Subject:  "user1",
				Actions:  []string{"read"},
				Resource: "file.txt",
				Effect:   Allow,
			},
			expected: true,
		},
		{
			name: "no match - subject mismatch",
			request: Request{
				Subject:  "user2",
				Action:   "read",
				Resource: "file.txt",
				Context:  map[string]string{},
			},
			policy: Policy{
				ID:       "policy1",
				Subject:  "user1",
				Actions:  []string{"read"},
				Resource: "file.txt",
				Effect:   Allow,
			},
			expected: false,
		},
		{
			name: "no match - action mismatch",
			request: Request{
				Subject:  "user1",
				Action:   "write",
				Resource: "file.txt",
				Context:  map[string]string{},
			},
			policy: Policy{
				ID:       "policy1",
				Subject:  "user1",
				Actions:  []string{"read"},
				Resource: "file.txt",
				Effect:   Allow,
			},
			expected: false,
		},
		{
			name: "no match - resource mismatch",
			request: Request{
				Subject:  "user1",
				Action:   "read",
				Resource: "other.txt",
				Context:  map[string]string{},
			},
			policy: Policy{
				ID:       "policy1",
				Subject:  "user1",
				Actions:  []string{"read"},
				Resource: "file.txt",
				Effect:   Allow,
			},
			expected: false,
		},
		{
			name: "wildcard subject matches all",
			request: Request{
				Subject:  "anyone",
				Action:   "read",
				Resource: "file.txt",
				Context:  map[string]string{},
			},
			policy: Policy{
				ID:       "policy1",
				Subject:  "*",
				Actions:  []string{"read"},
				Resource: "file.txt",
				Effect:   Allow,
			},
			expected: true,
		},
		{
			name: "wildcard action matches all",
			request: Request{
				Subject:  "user1",
				Action:   "delete",
				Resource: "file.txt",
				Context:  map[string]string{},
			},
			policy: Policy{
				ID:       "policy1",
				Subject:  "user1",
				Actions:  []string{"*"},
				Resource: "file.txt",
				Effect:   Allow,
			},
			expected: true,
		},
		{
			name: "wildcard resource matches all",
			request: Request{
				Subject:  "user1",
				Action:   "read",
				Resource: "any/path/file.txt",
				Context:  map[string]string{},
			},
			policy: Policy{
				ID:       "policy1",
				Subject:  "user1",
				Actions:  []string{"read"},
				Resource: "*",
				Effect:   Allow,
			},
			expected: true,
		},
		{
			name: "role-based match - single role",
			request: Request{
				Subject:  "user1",
				Action:   "admin",
				Resource: "system",
				Context:  map[string]string{"roles": "admin"},
			},
			policy: Policy{
				ID:       "policy1",
				Subject:  "", // empty subject with roles
				Roles:    []string{"admin"},
				Actions:  []string{"admin"},
				Resource: "system",
				Effect:   Allow,
			},
			expected: true,
		},
		{
			name: "role-based match - multiple roles in context",
			request: Request{
				Subject:  "user1",
				Action:   "write",
				Resource: "data",
				Context:  map[string]string{"roles": "reader,writer,admin"},
			},
			policy: Policy{
				ID:       "policy1",
				Subject:  "",
				Roles:    []string{"writer"},
				Actions:  []string{"write"},
				Resource: "data",
				Effect:   Allow,
			},
			expected: true,
		},
		{
			name: "role-based no match - required role not in context",
			request: Request{
				Subject:  "user1",
				Action:   "delete",
				Resource: "data",
				Context:  map[string]string{"roles": "reader,writer"},
			},
			policy: Policy{
				ID:       "policy1",
				Subject:  "",
				Roles:    []string{"admin"},
				Actions:  []string{"delete"},
				Resource: "data",
				Effect:   Allow,
			},
			expected: false,
		},
		{
			name: "required scopes match",
			request: Request{
				Subject:  "user1",
				Action:   "read",
				Resource: "api",
				Context:  map[string]string{"scopes": "read write admin"},
			},
			policy: Policy{
				ID:             "policy1",
				Subject:        "user1",
				Actions:        []string{"read"},
				Resource:       "api",
				RequiredScopes: []string{"read", "write"},
				Effect:         Allow,
			},
			expected: true,
		},
		{
			name: "required scopes mismatch",
			request: Request{
				Subject:  "user1",
				Action:   "read",
				Resource: "api",
				Context:  map[string]string{"scopes": "read"},
			},
			policy: Policy{
				ID:             "policy1",
				Subject:        "user1",
				Actions:        []string{"read"},
				Resource:       "api",
				RequiredScopes: []string{"read", "write"},
				Effect:         Allow,
			},
			expected: false,
		},
		{
			name: "conditions match - equals",
			request: Request{
				Subject:  "user1",
				Action:   "read",
				Resource: "file.txt",
				Context:  map[string]string{"department": "engineering"},
			},
			policy: Policy{
				ID:       "policy1",
				Subject:  "user1",
				Actions:  []string{"read"},
				Resource: "file.txt",
				Conditions: []Condition{
					{Key: "department", Operator: "equals", Values: []string{"engineering"}},
				},
				Effect: Allow,
			},
			expected: true,
		},
		{
			name: "conditions mismatch - equals",
			request: Request{
				Subject:  "user1",
				Action:   "read",
				Resource: "file.txt",
				Context:  map[string]string{"department": "sales"},
			},
			policy: Policy{
				ID:       "policy1",
				Subject:  "user1",
				Actions:  []string{"read"},
				Resource: "file.txt",
				Conditions: []Condition{
					{Key: "department", Operator: "equals", Values: []string{"engineering"}},
				},
				Effect: Allow,
			},
			expected: false,
		},
		{
			name: "conditions key missing",
			request: Request{
				Subject:  "user1",
				Action:   "read",
				Resource: "file.txt",
				Context:  map[string]string{},
			},
			policy: Policy{
				ID:       "policy1",
				Subject:  "user1",
				Actions:  []string{"read"},
				Resource: "file.txt",
				Conditions: []Condition{
					{Key: "department", Operator: "equals", Values: []string{"engineering"}},
				},
				Effect: Allow,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ma.matchesPolicy(tt.request, tt.policy)
			if result != tt.expected {
				t.Errorf("matchesPolicy() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

// TestAuthorize_CombiningAlgorithms tests different policy combining algorithms
func TestAuthorize_CombiningAlgorithms(t *testing.T) {
	tests := []struct {
		name            string
		combining       CombiningStrategy
		policies        []Policy
		request         Request
		expectedAllow   bool
		expectedReason  string
	}{
		{
			name:      "DenyOverrides - deny wins over allow",
			combining: DenyOverrides,
			policies: []Policy{
				{
					ID:       "allow-policy",
					Subject:  "user1",
					Actions:  []string{"read"},
					Resource: "*",
					Effect:   Allow,
				},
				{
					ID:       "deny-policy",
					Subject:  "user1",
					Actions:  []string{"read"},
					Resource: "*",
					Effect:   Deny,
				},
			},
			request: Request{
				Subject:  "user1",
				Action:   "read",
				Resource: "file.txt",
				Context:  map[string]string{},
			},
			expectedAllow:  false,
			expectedReason: "Deny by policy deny-policy",
		},
		{
			name:      "DenyOverrides - allow when no deny",
			combining: DenyOverrides,
			policies: []Policy{
				{
					ID:       "allow-policy",
					Subject:  "user1",
					Actions:  []string{"read"},
					Resource: "*",
					Effect:   Allow,
				},
			},
			request: Request{
				Subject:  "user1",
				Action:   "read",
				Resource: "file.txt",
				Context:  map[string]string{},
			},
			expectedAllow:  true,
			expectedReason: "Allow by policy allow-policy",
		},
		{
			name:      "PermitOverrides - allow wins over deny",
			combining: PermitOverrides,
			policies: []Policy{
				{
					ID:       "deny-policy",
					Subject:  "user1",
					Actions:  []string{"read"},
					Resource: "*",
					Effect:   Deny,
				},
				{
					ID:       "allow-policy",
					Subject:  "user1",
					Actions:  []string{"read"},
					Resource: "*",
					Effect:   Allow,
				},
			},
			request: Request{
				Subject:  "user1",
				Action:   "read",
				Resource: "file.txt",
				Context:  map[string]string{},
			},
			expectedAllow:  true,
			expectedReason: "Allow by policy allow-policy",
		},
		{
			name:      "PermitOverrides - deny when no allow",
			combining: PermitOverrides,
			policies: []Policy{
				{
					ID:       "deny-policy",
					Subject:  "user1",
					Actions:  []string{"read"},
					Resource: "*",
					Effect:   Deny,
				},
			},
			request: Request{
				Subject:  "user1",
				Action:   "read",
				Resource: "file.txt",
				Context:  map[string]string{},
			},
			expectedAllow:  false,
			expectedReason: "Deny by policy deny-policy",
		},
		{
			name:      "FirstApplicable - first matching policy wins",
			combining: FirstApplicable,
			policies: []Policy{
				{
					ID:       "first-deny",
					Subject:  "user1",
					Actions:  []string{"read"},
					Resource: "*",
					Effect:   Deny,
				},
				{
					ID:       "second-allow",
					Subject:  "user1",
					Actions:  []string{"read"},
					Resource: "*",
					Effect:   Allow,
				},
			},
			request: Request{
				Subject:  "user1",
				Action:   "read",
				Resource: "file.txt",
				Context:  map[string]string{},
			},
			expectedAllow:  false,
			expectedReason: "Deny by policy first-deny",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ma := NewMemoryAuthorizer()
			ma.SetCombiningStrategy(tt.combining)
			for _, policy := range tt.policies {
				ma.AddPolicy(policy)
			}

			ctx := context.Background()
			decision, err := ma.Authorize(ctx, tt.request)
			if err != nil {
				t.Fatalf("Authorize() error = %v", err)
			}
			if decision.Allow != tt.expectedAllow {
				t.Errorf("decision.Allow = %v, expected %v", decision.Allow, tt.expectedAllow)
			}
			if decision.Reason != tt.expectedReason {
				t.Errorf("decision.Reason = %q, expected %q", decision.Reason, tt.expectedReason)
			}
		})
	}
}

// TestAuthorize_NoMatchingPolicy tests default deny when no policy matches
func TestAuthorize_NoMatchingPolicy(t *testing.T) {
	ma := NewMemoryAuthorizer()
	ma.AddPolicy(Policy{
		ID:       "policy1",
		Subject:  "admin",
		Actions:  []string{"delete"},
		Resource: "system",
		Effect:   Allow,
	})

	ctx := context.Background()
	request := Request{
		Subject:  "user1",
		Action:   "read",
		Resource: "file.txt",
		Context:  map[string]string{},
	}

	decision, err := ma.Authorize(ctx, request)
	if err != nil {
		t.Fatalf("Authorize() error = %v", err)
	}
	if decision.Allow {
		t.Error("Expected default deny when no policy matches")
	}
	if decision.Reason != "No matching policy found - default deny" {
		t.Errorf("decision.Reason = %q, expected default deny message", decision.Reason)
	}
}

// MockObligationExecutorForPost implements ObligationExecutor for testing executePostDecision
type MockObligationExecutorForPost struct {
	executedObligations []string
	failOnID            string // if set, fail when this obligation ID is encountered
}

func (m *MockObligationExecutorForPost) Execute(obligation Obligation, context map[string]interface{}) error {
	m.executedObligations = append(m.executedObligations, obligation.ID)
	if obligation.ID == m.failOnID {
		return fmt.Errorf("obligation %s failed", obligation.ID)
	}
	return nil
}

func (m *MockObligationExecutorForPost) PersistAudit(obligation Obligation, context map[string]interface{}, result error) error {
	return nil
}

// TestAuthorize_WithObligations tests authorization with obligations
func TestAuthorize_WithObligations(t *testing.T) {
	ma := NewMemoryAuthorizer()
	mockExec := &MockObligationExecutorForPost{}
	ma.SetObligationExecutor(mockExec)

	policy := Policy{
		ID:       "policy-with-obligation",
		Subject:  "user1",
		Actions:  []string{"write"},
		Resource: "data",
		Effect:   Allow,
		Obligations: []Obligation{
			{ID: "log-access", Type: "audit", Mandatory: false},
			{ID: "notify-admin", Type: "notification", Mandatory: false},
		},
	}
	ma.AddPolicy(policy)

	ctx := context.Background()
	request := Request{
		Subject:  "user1",
		Action:   "write",
		Resource: "data",
		Context:  map[string]string{},
	}

	decision, err := ma.Authorize(ctx, request)
	if err != nil {
		t.Fatalf("Authorize() error = %v", err)
	}
	if !decision.Allow {
		t.Error("Expected decision to be allowed")
	}
	if len(mockExec.executedObligations) != 2 {
		t.Errorf("Expected 2 obligations executed, got %d", len(mockExec.executedObligations))
	}
}

// TestAuthorize_WithMandatoryObligationFailure tests mandatory obligation failure
func TestAuthorize_WithMandatoryObligationFailure(t *testing.T) {
	ma := NewMemoryAuthorizer()
	mockExec := &MockObligationExecutorForPost{failOnID: "mandatory-check"}
	ma.SetObligationExecutor(mockExec)

	policy := Policy{
		ID:       "policy-with-mandatory-obligation",
		Subject:  "user1",
		Actions:  []string{"delete"},
		Resource: "critical-data",
		Effect:   Allow,
		Obligations: []Obligation{
			{ID: "mandatory-check", Type: "validation", Mandatory: true},
		},
	}
	ma.AddPolicy(policy)

	ctx := context.Background()
	request := Request{
		Subject:  "user1",
		Action:   "delete",
		Resource: "critical-data",
		Context:  map[string]string{},
	}

	decision, err := ma.Authorize(ctx, request)
	if err != nil {
		t.Fatalf("Authorize() error = %v", err)
	}
	if decision.Allow {
		t.Error("Expected decision to be denied due to mandatory obligation failure")
	}
	if decision.Metadata["obligation_failure"] != "mandatory-check" {
		t.Errorf("Expected obligation_failure metadata, got %v", decision.Metadata)
	}
}

// TestAuthorize_WithAdvice tests authorization with advice (non-mandatory obligations)
func TestAuthorize_WithAdvice(t *testing.T) {
	ma := NewMemoryAuthorizer()
	mockExec := &MockObligationExecutorForPost{failOnID: "optional-advice"}
	ma.SetObligationExecutor(mockExec)

	policy := Policy{
		ID:       "policy-with-advice",
		Subject:  "user1",
		Actions:  []string{"read"},
		Resource: "data",
		Effect:   Allow,
		Advice: []Advice{
			{ID: "optional-advice", Type: "suggestion"},
		},
	}
	ma.AddPolicy(policy)

	ctx := context.Background()
	request := Request{
		Subject:  "user1",
		Action:   "read",
		Resource: "data",
		Context:  map[string]string{},
	}

	decision, err := ma.Authorize(ctx, request)
	if err != nil {
		t.Fatalf("Authorize() error = %v", err)
	}
	// Advice failure should NOT prevent authorization
	if !decision.Allow {
		t.Error("Expected decision to be allowed despite advice failure")
	}
	if decision.Metadata["advice_failure"] != "optional-advice" {
		t.Errorf("Expected advice_failure metadata, got %v", decision.Metadata)
	}
}

// TestAuthorize_LegacyCacheEnabled tests legacy cache (non-LRU) behavior
func TestAuthorize_LegacyCacheEnabled(t *testing.T) {
	ma := NewMemoryAuthorizer()
	ma.EnableCaching(100 * time.Millisecond)

	policy := Policy{
		ID:       "policy1",
		Subject:  "user1",
		Actions:  []string{"read"},
		Resource: "file.txt",
		Effect:   Allow,
	}
	ma.AddPolicy(policy)

	ctx := context.Background()
	request := Request{
		Subject:  "user1",
		Action:   "read",
		Resource: "file.txt",
		Context:  map[string]string{},
	}

	// First call - cache miss
	decision1, err := ma.Authorize(ctx, request)
	if err != nil {
		t.Fatalf("Authorize() error = %v", err)
	}
	if !decision1.Allow {
		t.Error("Expected first decision to be allowed")
	}
	if decision1.Metadata["cache_hit"] != "false" {
		t.Errorf("Expected cache_hit=false on first call, got %s", decision1.Metadata["cache_hit"])
	}

	// Second call - cache hit
	decision2, err := ma.Authorize(ctx, request)
	if err != nil {
		t.Fatalf("Authorize() error = %v", err)
	}
	if !decision2.Allow {
		t.Error("Expected second decision to be allowed")
	}
	if decision2.Metadata["cache_hit"] != "true" {
		t.Errorf("Expected cache_hit=true on second call, got %s", decision2.Metadata["cache_hit"])
	}

	// Wait for cache to expire
	time.Sleep(150 * time.Millisecond)

	// Third call - cache miss after expiration
	decision3, err := ma.Authorize(ctx, request)
	if err != nil {
		t.Fatalf("Authorize() error = %v", err)
	}
	if !decision3.Allow {
		t.Error("Expected third decision to be allowed")
	}
	if decision3.Metadata["cache_hit"] != "false" {
		t.Errorf("Expected cache_hit=false after expiration, got %s", decision3.Metadata["cache_hit"])
	}
}

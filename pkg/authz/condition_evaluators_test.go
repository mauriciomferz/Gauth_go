package authz

import (
	"context"
	"testing"
)

// TestEvaluateCondition_NumericGt tests numeric greater than conditions
func TestEvaluateCondition_NumericGt(t *testing.T) {
	ma := NewMemoryAuthorizer()

	tests := []struct {
		name     string
		request  Request
		policy   Policy
		expected bool
	}{
		{
			name: "numeric_gt matches when value greater",
			request: Request{
				Subject:  "user1",
				Action:   "read",
				Resource: "data",
				Context:  map[string]string{"score": "95"},
			},
			policy: Policy{
				ID:       "high-score-policy",
				Subject:  "user1",
				Actions:  []string{"read"},
				Resource: "data",
				Conditions: []Condition{
					{Key: "score", Operator: "numeric_gt", Values: []string{"90"}},
				},
				Effect: Allow,
			},
			expected: true,
		},
		{
			name: "numeric_gt no match when value equal",
			request: Request{
				Subject:  "user1",
				Action:   "read",
				Resource: "data",
				Context:  map[string]string{"score": "90"},
			},
			policy: Policy{
				ID:       "high-score-policy",
				Subject:  "user1",
				Actions:  []string{"read"},
				Resource: "data",
				Conditions: []Condition{
					{Key: "score", Operator: "numeric_gt", Values: []string{"90"}},
				},
				Effect: Allow,
			},
			expected: false,
		},
		{
			name: "numeric_gt no match when value less",
			request: Request{
				Subject:  "user1",
				Action:   "read",
				Resource: "data",
				Context:  map[string]string{"score": "85"},
			},
			policy: Policy{
				ID:       "high-score-policy",
				Subject:  "user1",
				Actions:  []string{"read"},
				Resource: "data",
				Conditions: []Condition{
					{Key: "score", Operator: "numeric_gt", Values: []string{"90"}},
				},
				Effect: Allow,
			},
			expected: false,
		},
		{
			name: "numeric_gt with invalid context value",
			request: Request{
				Subject:  "user1",
				Action:   "read",
				Resource: "data",
				Context:  map[string]string{"score": "not-a-number"},
			},
			policy: Policy{
				ID:       "high-score-policy",
				Subject:  "user1",
				Actions:  []string{"read"},
				Resource: "data",
				Conditions: []Condition{
					{Key: "score", Operator: "numeric_gt", Values: []string{"90"}},
				},
				Effect: Allow,
			},
			expected: false,
		},
		{
			name: "numeric_gt with multiple thresholds - matches first",
			request: Request{
				Subject:  "user1",
				Action:   "read",
				Resource: "data",
				Context:  map[string]string{"age": "35"},
			},
			policy: Policy{
				ID:       "age-policy",
				Subject:  "user1",
				Actions:  []string{"read"},
				Resource: "data",
				Conditions: []Condition{
					{Key: "age", Operator: "numeric_gt", Values: []string{"30", "40", "50"}},
				},
				Effect: Allow,
			},
			expected: true,
		},
		{
			name: "numeric_gt with invalid threshold value",
			request: Request{
				Subject:  "user1",
				Action:   "read",
				Resource: "data",
				Context:  map[string]string{"score": "95"},
			},
			policy: Policy{
				ID:       "policy1",
				Subject:  "user1",
				Actions:  []string{"read"},
				Resource: "data",
				Conditions: []Condition{
					{Key: "score", Operator: "numeric_gt", Values: []string{"invalid", "90"}},
				},
				Effect: Allow,
			},
			expected: true, // Should skip invalid threshold and check "90"
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

// TestEvaluateCondition_NumericLt tests numeric less than conditions
func TestEvaluateCondition_NumericLt(t *testing.T) {
	ma := NewMemoryAuthorizer()

	tests := []struct {
		name     string
		request  Request
		policy   Policy
		expected bool
	}{
		{
			name: "numeric_lt matches when value less",
			request: Request{
				Subject:  "user1",
				Action:   "write",
				Resource: "file",
				Context:  map[string]string{"size": "50"},
			},
			policy: Policy{
				ID:       "small-file-policy",
				Subject:  "user1",
				Actions:  []string{"write"},
				Resource: "file",
				Conditions: []Condition{
					{Key: "size", Operator: "numeric_lt", Values: []string{"100"}},
				},
				Effect: Allow,
			},
			expected: true,
		},
		{
			name: "numeric_lt no match when value equal",
			request: Request{
				Subject:  "user1",
				Action:   "write",
				Resource: "file",
				Context:  map[string]string{"size": "100"},
			},
			policy: Policy{
				ID:       "small-file-policy",
				Subject:  "user1",
				Actions:  []string{"write"},
				Resource: "file",
				Conditions: []Condition{
					{Key: "size", Operator: "numeric_lt", Values: []string{"100"}},
				},
				Effect: Allow,
			},
			expected: false,
		},
		{
			name: "numeric_lt no match when value greater",
			request: Request{
				Subject:  "user1",
				Action:   "write",
				Resource: "file",
				Context:  map[string]string{"size": "150"},
			},
			policy: Policy{
				ID:       "small-file-policy",
				Subject:  "user1",
				Actions:  []string{"write"},
				Resource: "file",
				Conditions: []Condition{
					{Key: "size", Operator: "numeric_lt", Values: []string{"100"}},
				},
				Effect: Allow,
			},
			expected: false,
		},
		{
			name: "numeric_lt with invalid context value",
			request: Request{
				Subject:  "user1",
				Action:   "write",
				Resource: "file",
				Context:  map[string]string{"size": "large"},
			},
			policy: Policy{
				ID:       "small-file-policy",
				Subject:  "user1",
				Actions:  []string{"write"},
				Resource: "file",
				Conditions: []Condition{
					{Key: "size", Operator: "numeric_lt", Values: []string{"100"}},
				},
				Effect: Allow,
			},
			expected: false,
		},
		{
			name: "numeric_lt with float values",
			request: Request{
				Subject:  "user1",
				Action:   "read",
				Resource: "data",
				Context:  map[string]string{"price": "19.99"},
			},
			policy: Policy{
				ID:       "cheap-items",
				Subject:  "user1",
				Actions:  []string{"read"},
				Resource: "data",
				Conditions: []Condition{
					{Key: "price", Operator: "numeric_lt", Values: []string{"20.0"}},
				},
				Effect: Allow,
			},
			expected: true,
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

// TestEvaluateCondition_TimeBefore tests time before conditions
func TestEvaluateCondition_TimeBefore(t *testing.T) {
	ma := NewMemoryAuthorizer()

	tests := []struct {
		name     string
		request  Request
		policy   Policy
		expected bool
	}{
		{
			name: "time_before matches when time is before",
			request: Request{
				Subject:  "user1",
				Action:   "access",
				Resource: "system",
				Context:  map[string]string{"timestamp": "2024-01-01T10:00:00Z"},
			},
			policy: Policy{
				ID:       "deadline-policy",
				Subject:  "user1",
				Actions:  []string{"access"},
				Resource: "system",
				Conditions: []Condition{
					{Key: "timestamp", Operator: "time_before", Values: []string{"2024-01-01T12:00:00Z"}},
				},
				Effect: Allow,
			},
			expected: true,
		},
		{
			name: "time_before no match when time is equal",
			request: Request{
				Subject:  "user1",
				Action:   "access",
				Resource: "system",
				Context:  map[string]string{"timestamp": "2024-01-01T12:00:00Z"},
			},
			policy: Policy{
				ID:       "deadline-policy",
				Subject:  "user1",
				Actions:  []string{"access"},
				Resource: "system",
				Conditions: []Condition{
					{Key: "timestamp", Operator: "time_before", Values: []string{"2024-01-01T12:00:00Z"}},
				},
				Effect: Allow,
			},
			expected: false,
		},
		{
			name: "time_before no match when time is after",
			request: Request{
				Subject:  "user1",
				Action:   "access",
				Resource: "system",
				Context:  map[string]string{"timestamp": "2024-01-01T14:00:00Z"},
			},
			policy: Policy{
				ID:       "deadline-policy",
				Subject:  "user1",
				Actions:  []string{"access"},
				Resource: "system",
				Conditions: []Condition{
					{Key: "timestamp", Operator: "time_before", Values: []string{"2024-01-01T12:00:00Z"}},
				},
				Effect: Allow,
			},
			expected: false,
		},
		{
			name: "time_before with invalid context time",
			request: Request{
				Subject:  "user1",
				Action:   "access",
				Resource: "system",
				Context:  map[string]string{"timestamp": "invalid-time"},
			},
			policy: Policy{
				ID:       "deadline-policy",
				Subject:  "user1",
				Actions:  []string{"access"},
				Resource: "system",
				Conditions: []Condition{
					{Key: "timestamp", Operator: "time_before", Values: []string{"2024-01-01T12:00:00Z"}},
				},
				Effect: Allow,
			},
			expected: false,
		},
		{
			name: "time_before with invalid policy time",
			request: Request{
				Subject:  "user1",
				Action:   "access",
				Resource: "system",
				Context:  map[string]string{"timestamp": "2024-01-01T10:00:00Z"},
			},
			policy: Policy{
				ID:       "deadline-policy",
				Subject:  "user1",
				Actions:  []string{"access"},
				Resource: "system",
				Conditions: []Condition{
					{Key: "timestamp", Operator: "time_before", Values: []string{"invalid-time", "2024-01-01T12:00:00Z"}},
				},
				Effect: Allow,
			},
			expected: true, // Should skip invalid and check valid time
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

// TestEvaluateCondition_TimeAfter tests time after conditions
func TestEvaluateCondition_TimeAfter(t *testing.T) {
	ma := NewMemoryAuthorizer()

	tests := []struct {
		name     string
		request  Request
		policy   Policy
		expected bool
	}{
		{
			name: "time_after matches when time is after",
			request: Request{
				Subject:  "user1",
				Action:   "access",
				Resource: "system",
				Context:  map[string]string{"timestamp": "2024-01-01T14:00:00Z"},
			},
			policy: Policy{
				ID:       "start-time-policy",
				Subject:  "user1",
				Actions:  []string{"access"},
				Resource: "system",
				Conditions: []Condition{
					{Key: "timestamp", Operator: "time_after", Values: []string{"2024-01-01T12:00:00Z"}},
				},
				Effect: Allow,
			},
			expected: true,
		},
		{
			name: "time_after no match when time is equal",
			request: Request{
				Subject:  "user1",
				Action:   "access",
				Resource: "system",
				Context:  map[string]string{"timestamp": "2024-01-01T12:00:00Z"},
			},
			policy: Policy{
				ID:       "start-time-policy",
				Subject:  "user1",
				Actions:  []string{"access"},
				Resource: "system",
				Conditions: []Condition{
					{Key: "timestamp", Operator: "time_after", Values: []string{"2024-01-01T12:00:00Z"}},
				},
				Effect: Allow,
			},
			expected: false,
		},
		{
			name: "time_after no match when time is before",
			request: Request{
				Subject:  "user1",
				Action:   "access",
				Resource: "system",
				Context:  map[string]string{"timestamp": "2024-01-01T10:00:00Z"},
			},
			policy: Policy{
				ID:       "start-time-policy",
				Subject:  "user1",
				Actions:  []string{"access"},
				Resource: "system",
				Conditions: []Condition{
					{Key: "timestamp", Operator: "time_after", Values: []string{"2024-01-01T12:00:00Z"}},
				},
				Effect: Allow,
			},
			expected: false,
		},
		{
			name: "time_after with invalid context time",
			request: Request{
				Subject:  "user1",
				Action:   "access",
				Resource: "system",
				Context:  map[string]string{"timestamp": "not-a-date"},
			},
			policy: Policy{
				ID:       "start-time-policy",
				Subject:  "user1",
				Actions:  []string{"access"},
				Resource: "system",
				Conditions: []Condition{
					{Key: "timestamp", Operator: "time_after", Values: []string{"2024-01-01T12:00:00Z"}},
				},
				Effect: Allow,
			},
			expected: false,
		},
		{
			name: "time_after with multiple values",
			request: Request{
				Subject:  "user1",
				Action:   "access",
				Resource: "system",
				Context:  map[string]string{"timestamp": "2024-06-15T10:00:00Z"},
			},
			policy: Policy{
				ID:       "seasonal-policy",
				Subject:  "user1",
				Actions:  []string{"access"},
				Resource: "system",
				Conditions: []Condition{
					{Key: "timestamp", Operator: "time_after", Values: []string{"2024-01-01T00:00:00Z", "2024-12-31T23:59:59Z"}},
				},
				Effect: Allow,
			},
			expected: true, // After first value
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

// TestAuthorize_WithTimeConditions tests full authorization with time conditions
func TestAuthorize_WithTimeConditions(t *testing.T) {
	ma := NewMemoryAuthorizer()

	// Policy with time_before condition
	policy := Policy{
		ID:       "time-restricted",
		Subject:  "*",
		Actions:  []string{"read"},
		Resource: "document",
		Conditions: []Condition{
			{Key: "access_time", Operator: "time_before", Values: []string{"2024-12-31T23:59:59Z"}},
		},
		Effect: Allow,
	}
	ma.AddPolicy(policy)

	ctx := context.Background()

	// Test with time before deadline
	request1 := Request{
		Subject:  "user1",
		Action:   "read",
		Resource: "document",
		Context:  map[string]string{"access_time": "2024-06-01T10:00:00Z"},
	}
	decision1, err := ma.Authorize(ctx, request1)
	if err != nil {
		t.Fatalf("Authorize() error = %v", err)
	}
	if !decision1.Allow {
		t.Error("Expected authorization to be allowed before deadline")
	}

	// Test with time after deadline
	request2 := Request{
		Subject:  "user1",
		Action:   "read",
		Resource: "document",
		Context:  map[string]string{"access_time": "2025-01-01T00:00:00Z"},
	}
	decision2, err := ma.Authorize(ctx, request2)
	if err != nil {
		t.Fatalf("Authorize() error = %v", err)
	}
	if decision2.Allow {
		t.Error("Expected authorization to be denied after deadline")
	}
}

// TestAuthorize_WithNumericConditions tests full authorization with numeric conditions
func TestAuthorize_WithNumericConditions(t *testing.T) {
	ma := NewMemoryAuthorizer()

	// Policy with numeric_gt condition
	policy := Policy{
		ID:       "premium-access",
		Subject:  "*",
		Actions:  []string{"access"},
		Resource: "premium-content",
		Conditions: []Condition{
			{Key: "subscription_level", Operator: "numeric_gt", Values: []string{"5"}},
		},
		Effect: Allow,
	}
	ma.AddPolicy(policy)

	ctx := context.Background()

	// Test with level above threshold
	request1 := Request{
		Subject:  "user1",
		Action:   "access",
		Resource: "premium-content",
		Context:  map[string]string{"subscription_level": "10"},
	}
	decision1, err := ma.Authorize(ctx, request1)
	if err != nil {
		t.Fatalf("Authorize() error = %v", err)
	}
	if !decision1.Allow {
		t.Error("Expected authorization for high subscription level")
	}

	// Test with level below threshold
	request2 := Request{
		Subject:  "user1",
		Action:   "access",
		Resource: "premium-content",
		Context:  map[string]string{"subscription_level": "3"},
	}
	decision2, err := ma.Authorize(ctx, request2)
	if err != nil {
		t.Fatalf("Authorize() error = %v", err)
	}
	if decision2.Allow {
		t.Error("Expected authorization to be denied for low subscription level")
	}
}

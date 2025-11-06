package ai

import (
	"testing"
)

func TestModelMetadata_TokenLimitPerCall(t *testing.T) {
	matrix := NewAICapabilityMatrix()
	matrix.SetEnforcementActive(true)

	// Register agent rule (no required claims for testing)
	matrix.UpdateEntityRule(AIEntityAgent, AICapabilityRule{
		EntityType:       AIEntityAgent,
		AllowedActions:   []string{"generate_text", "analyze"},
		RequiredClaims:   []string{}, // Empty required claims for testing
		ForbiddenActions: []string{},
		AuditLevel:       "basic",
	})

	profile := AISystemProfile{
		EntityType: AIEntityAgent,
		SystemID:   "agent-001",
		ModelName:  "gpt-4",
		ModelMetadata: &ModelMetadata{
			ModelName:         "gpt-4",
			ModelVersion:      "0613",
			TokenLimitPerCall: 4096,
			CostLimitPerCall:  0.05,
			ContextWindow:     8192,
		},
		Jurisdiction: "US",
		RiskLevel:    "medium",
	}

	tests := []struct {
		name           string
		action         string
		claims         map[string]any
		expectAllow    bool
		expectReason   string
		expectViolated []string
	}{
		{
			name:   "within token limit",
			action: "generate_text",
			claims: map[string]any{
				"requested_tokens":            2048,
				"nist_ai_compliance":          "verified",
				"algorithmic_accountability":  "enabled",
			},
			expectAllow: true,
		},
		{
			name:   "exceed token limit int",
			action: "generate_text",
			claims: map[string]any{
				"requested_tokens":            5000,
				"nist_ai_compliance":          "verified",
				"algorithmic_accountability":  "enabled",
			},
			expectAllow:    false,
			expectReason:   "requested tokens 5000 exceeds model limit 4096 per call",
			expectViolated: []string{"model_limit_exceeded"},
		},
		{
			name:   "exceed token limit float",
			action: "generate_text",
			claims: map[string]any{
				"requested_tokens":            5000.0,
				"nist_ai_compliance":          "verified",
				"algorithmic_accountability":  "enabled",
			},
			expectAllow:    false,
			expectReason:   "requested tokens 5000 exceeds model limit 4096 per call",
			expectViolated: []string{"model_limit_exceeded"},
		},
		{
			name:   "exceed cost limit",
			action: "generate_text",
			claims: map[string]any{
				"estimated_cost":              0.10,
				"nist_ai_compliance":          "verified",
				"algorithmic_accountability":  "enabled",
			},
			expectAllow:    false,
			expectReason:   "estimated cost 0.1000 USD exceeds model limit 0.0500 USD per call",
			expectViolated: []string{"model_limit_exceeded"},
		},
		{
			name:   "within cost limit",
			action: "generate_text",
			claims: map[string]any{
				"estimated_cost":              0.02,
				"nist_ai_compliance":          "verified",
				"algorithmic_accountability":  "enabled",
			},
			expectAllow: true,
		},
		{
			name:   "exceed context window",
			action: "analyze",
			claims: map[string]any{
				"context_size":                10000,
				"nist_ai_compliance":          "verified",
				"algorithmic_accountability":  "enabled",
			},
			expectAllow:    false,
			expectReason:   "context size 10000 exceeds model context window 8192",
			expectViolated: []string{"model_limit_exceeded"},
		},
		{
			name:   "within context window",
			action: "analyze",
			claims: map[string]any{
				"context_size":                7000,
				"nist_ai_compliance":          "verified",
				"algorithmic_accountability":  "enabled",
			},
			expectAllow: true,
		},
		{
			name:   "no limits claimed passes",
			action: "generate_text",
			claims: map[string]any{
				"other_metadata":              "value",
				"nist_ai_compliance":          "verified",
				"algorithmic_accountability":  "enabled",
			},
			expectAllow: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decision := matrix.EnforceAICapabilities(profile, tt.action, tt.claims)
			gotAllow := decision.Decision == DecisionAllow
			if gotAllow != tt.expectAllow {
				t.Errorf("EnforceAICapabilities() allow = %v, want %v", gotAllow, tt.expectAllow)
				t.Logf("Decision: %s, Reason: %s", decision.Decision, decision.Reason)
			}
			if !tt.expectAllow {
				if decision.Reason != tt.expectReason {
					t.Errorf("EnforceAICapabilities() reason = %q, want %q", decision.Reason, tt.expectReason)
				}
				if len(decision.ViolatedRules) != len(tt.expectViolated) {
					t.Errorf("EnforceAICapabilities() violated rules count = %d, want %d", len(decision.ViolatedRules), len(tt.expectViolated))
				} else {
					for i, rule := range tt.expectViolated {
						if decision.ViolatedRules[i] != rule {
							t.Errorf("EnforceAICapabilities() violated rule[%d] = %q, want %q", i, decision.ViolatedRules[i], rule)
						}
					}
				}
			}
		})
	}
}

func TestModelMetadata_BatchSizeLimit(t *testing.T) {
	matrix := NewAICapabilityMatrix()
	matrix.SetEnforcementActive(true)

	matrix.UpdateEntityRule(AIEntityModel, AICapabilityRule{
		EntityType:       AIEntityModel,
		AllowedActions:   []string{"embed"},
		RequiredClaims:   []string{}, // Empty for testing
		ForbiddenActions: []string{},
		AuditLevel:       "detailed",
	})

	profile := AISystemProfile{
		EntityType: AIEntityModel,
		SystemID:   "embedding-model-001",
		ModelName:  "text-embedding-ada-002",
		ModelMetadata: &ModelMetadata{
			ModelName:    "text-embedding-ada-002",
			ModelVersion: "v2",
			MaxBatchSize: 100,
		},
		Jurisdiction: "EU",
		RiskLevel:    "low",
	}

	// Within limit
	decision := matrix.EnforceAICapabilities(profile, "embed", map[string]any{
		"batch_size":            50,
		"eu_ai_conformity":      "compliant",
		"human_oversight":       "enabled",
		"ce_marking":            "verified",
		"transparency_report":   "published",
	})
	if decision.Decision != DecisionAllow {
		t.Errorf("Expected allow for batch_size=50, got %s: %s", decision.Decision, decision.Reason)
	}

	// Exceed limit (int)
	decision = matrix.EnforceAICapabilities(profile, "embed", map[string]any{
		"batch_size":            150,
		"eu_ai_conformity":      "compliant",
		"human_oversight":       "enabled",
		"ce_marking":            "verified",
		"transparency_report":   "published",
	})
	if decision.Decision != DecisionDeny {
		t.Errorf("Expected deny for batch_size=150, got %s", decision.Decision)
	}
	expectedReason := "batch size 150 exceeds model limit 100"
	if decision.Reason != expectedReason {
		t.Errorf("Expected reason %q, got %q", expectedReason, decision.Reason)
	}

	// Exceed limit (float)
	decision = matrix.EnforceAICapabilities(profile, "embed", map[string]any{
		"batch_size":            150.0,
		"eu_ai_conformity":      "compliant",
		"human_oversight":       "enabled",
		"ce_marking":            "verified",
		"transparency_report":   "published",
	})
	if decision.Decision != DecisionDeny {
		t.Errorf("Expected deny for batch_size=150.0, got %s", decision.Decision)
	}
}

func TestModelMetadata_NilMetadata(t *testing.T) {
	matrix := NewAICapabilityMatrix()
	matrix.SetEnforcementActive(true)

	matrix.UpdateEntityRule(AIEntityAgent, AICapabilityRule{
		EntityType:       AIEntityAgent,
		AllowedActions:   []string{"query"},
		RequiredClaims:   []string{}, // Empty for testing
		ForbiddenActions: []string{},
		AuditLevel:       "basic",
	})

	profile := AISystemProfile{
		EntityType:    AIEntityAgent,
		SystemID:      "agent-no-limits",
		ModelMetadata: nil, // No limits
		Jurisdiction:  "US",
		RiskLevel:     "low",
	}

	// Should allow when no metadata present
	decision := matrix.EnforceAICapabilities(profile, "query", map[string]any{
		"requested_tokens":            999999, // Huge value but no limit set
		"nist_ai_compliance":          "verified",
		"algorithmic_accountability":  "enabled",
	})
	if decision.Decision != DecisionAllow {
		t.Errorf("Expected allow when ModelMetadata is nil, got %s: %s", decision.Decision, decision.Reason)
	}
}

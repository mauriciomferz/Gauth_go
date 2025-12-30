// Copyright 2025 AgentAuth Contributors
// SPDX-License-Identifier: MIT

package ai

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCapabilityEnforcer_RegisterAndEnforce(t *testing.T) {
	enforcer := NewCapabilityEnforcer()

	limits := &CapabilityLimits{
		CapabilityID:  "text-generation",
		MaxRequests:   100,
		MaxTokens:     4096,
		AllowedModels: []string{"gpt-4", "gpt-3.5-turbo"},
	}

	err := enforcer.RegisterCapability(limits)
	require.NoError(t, err)

	// Test allowed usage
	result, err := enforcer.Enforce(&UsageContext{
		CapabilityID: "text-generation",
		ModelName:    "gpt-4",
		TokenCount:   1000,
		RequestCount: 50,
	})
	require.NoError(t, err)
	assert.True(t, result.Allowed)
	assert.Empty(t, result.Violations)

	// Test token limit violation
	result, err = enforcer.Enforce(&UsageContext{
		CapabilityID: "text-generation",
		ModelName:    "gpt-4",
		TokenCount:   5000, // exceeds limit
		RequestCount: 50,
	})
	require.NoError(t, err)
	assert.False(t, result.Allowed)
	assert.Contains(t, result.Violations[0], "token count")

	// Test model not allowed
	result, err = enforcer.Enforce(&UsageContext{
		CapabilityID: "text-generation",
		ModelName:    "claude-2",
		TokenCount:   1000,
		RequestCount: 50,
	})
	require.NoError(t, err)
	assert.False(t, result.Allowed)
	assert.Contains(t, result.Violations[0], "not in allowed list")
}

func TestCapabilityEnforcer_ForbiddenActions(t *testing.T) {
	enforcer := NewCapabilityEnforcer()

	limits := &CapabilityLimits{
		CapabilityID:     "code-execution",
		MaxRequests:      50,
		ForbiddenActions: []string{"delete_files", "system_call"},
	}

	err := enforcer.RegisterCapability(limits)
	require.NoError(t, err)

	// Allowed action
	result, err := enforcer.Enforce(&UsageContext{
		CapabilityID: "code-execution",
		Action:       "run_python",
		RequestCount: 10,
	})
	require.NoError(t, err)
	assert.True(t, result.Allowed)

	// Forbidden action
	result, err = enforcer.Enforce(&UsageContext{
		CapabilityID: "code-execution",
		Action:       "delete_files",
		RequestCount: 10,
	})
	require.NoError(t, err)
	assert.False(t, result.Allowed)
	assert.Contains(t, result.Violations[0], "forbidden")
}

func TestCapabilityEnforcer_RequireApproval(t *testing.T) {
	enforcer := NewCapabilityEnforcer()

	limits := &CapabilityLimits{
		CapabilityID:    "sensitive-data-access",
		RequireApproval: true,
	}

	err := enforcer.RegisterCapability(limits)
	require.NoError(t, err)

	result, err := enforcer.Enforce(&UsageContext{
		CapabilityID: "sensitive-data-access",
	})
	require.NoError(t, err)
	assert.True(t, result.RequireAuth)
	assert.NotEmpty(t, result.ApprovalHint)
}

func TestCapabilityEnforcer_UpdateAndRemove(t *testing.T) {
	enforcer := NewCapabilityEnforcer()

	limits := &CapabilityLimits{
		CapabilityID: "test-cap",
		MaxRequests:  100,
	}

	err := enforcer.RegisterCapability(limits)
	require.NoError(t, err)

	// Update
	limits.MaxRequests = 200
	err = enforcer.UpdateCapability(limits)
	require.NoError(t, err)

	updated, exists := enforcer.GetCapability("test-cap")
	require.True(t, exists)
	assert.Equal(t, int64(200), updated.MaxRequests)

	// Remove
	err = enforcer.RemoveCapability("test-cap")
	require.NoError(t, err)

	_, exists = enforcer.GetCapability("test-cap")
	assert.False(t, exists)
}

func TestCapabilityEnforcer_ExportImport(t *testing.T) {
	enforcer := NewCapabilityEnforcer()

	limits := &CapabilityLimits{
		CapabilityID: "export-test",
		MaxRequests:  500,
		MaxTokens:    2048,
	}

	err := enforcer.RegisterCapability(limits)
	require.NoError(t, err)

	// Export
	data, err := enforcer.ExportMatrix()
	require.NoError(t, err)
	assert.NotEmpty(t, data)

	// Import into new enforcer
	enforcer2 := NewCapabilityEnforcer()
	err = enforcer2.ImportMatrix(data)
	require.NoError(t, err)

	imported, exists := enforcer2.GetCapability("export-test")
	require.True(t, exists)
	assert.Equal(t, limits.MaxRequests, imported.MaxRequests)
	assert.Equal(t, limits.MaxTokens, imported.MaxTokens)
}

func TestCapabilityEnforcer_UnregisteredCapability(t *testing.T) {
	enforcer := NewCapabilityEnforcer()

	result, err := enforcer.Enforce(&UsageContext{
		CapabilityID: "unknown-capability",
	})
	require.NoError(t, err)
	assert.False(t, result.Allowed)
	assert.Contains(t, result.Reason, "not registered")
}

func TestCapabilityEnforcer_ModelMetadataLimits(t *testing.T) {
	enforcer := NewCapabilityEnforcer()

	limits := &CapabilityLimits{
		CapabilityID: "text-gen-with-limits",
		ModelMetadata: map[string]ModelLimits{
			"gpt-4": {
				MaxContextTokens:  8192,
				MaxOutputTokens:   4096,
				CostPerToken:      0.00003,
				MaxCostPerRequest: 0.5,
				Deprecated:        false,
			},
			"gpt-3-old": {
				MaxContextTokens: 2048,
				Deprecated:       true,
			},
		},
	}

	err := enforcer.RegisterCapability(limits)
	require.NoError(t, err)

	// Test within model limits
	result, err := enforcer.Enforce(&UsageContext{
		CapabilityID: "text-gen-with-limits",
		ModelName:    "gpt-4",
		TokenCount:   5000,
	})
	require.NoError(t, err)
	assert.True(t, result.Allowed)

	// Test exceeds context limit
	result, err = enforcer.Enforce(&UsageContext{
		CapabilityID: "text-gen-with-limits",
		ModelName:    "gpt-4",
		TokenCount:   10000, // exceeds 8192
	})
	require.NoError(t, err)
	assert.False(t, result.Allowed)
	assert.Contains(t, result.Violations[0], "context limit")

	// Test cost limit (use token count that doesn't exceed context limit)
	result, err = enforcer.Enforce(&UsageContext{
		CapabilityID: "text-gen-with-limits",
		ModelName:    "gpt-4",
		TokenCount:   8000, // 8000 * 0.00003 = 0.24, within context but let's make it exceed
	})
	require.NoError(t, err)
	// This should pass all checks since 8000 < 8192 context and cost is 0.24 < 0.5
	assert.True(t, result.Allowed)

	// Now test actual cost limit violation with high token count that still fits context
	gpt4Limits := limits.ModelMetadata["gpt-4"]
	gpt4Limits.MaxCostPerRequest = 0.1 // Lower the cost limit
	limits.ModelMetadata["gpt-4"] = gpt4Limits
	err = enforcer.UpdateCapability(limits)
	require.NoError(t, err)

	result, err = enforcer.Enforce(&UsageContext{
		CapabilityID: "text-gen-with-limits",
		ModelName:    "gpt-4",
		TokenCount:   5000, // 5000 * 0.00003 = 0.15 > 0.1 max cost
	})
	require.NoError(t, err)
	assert.False(t, result.Allowed)
	assert.Contains(t, result.Violations[0], "cost")

	// Test deprecated model
	result, err = enforcer.Enforce(&UsageContext{
		CapabilityID: "text-gen-with-limits",
		ModelName:    "gpt-3-old",
		TokenCount:   1000,
	})
	require.NoError(t, err)
	assert.False(t, result.Allowed)
	assert.Contains(t, result.Violations[0], "deprecated")
}

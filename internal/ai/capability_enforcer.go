// Copyright 2025 Gimel Foundation
// SPDX-License-Identifier: Apache-2.0

package ai

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync"
)

// CapabilityEnforcer provides runtime enforcement of AI capability limits.
// Addresses gap sec11.item1 (P1): AI capability matrix enforcement.
type CapabilityEnforcer struct {
	matrix map[string]*CapabilityLimits
	mu     sync.RWMutex
}

// CapabilityLimits defines runtime limits for a specific AI capability.
type CapabilityLimits struct {
	CapabilityID     string                 `json:"capability_id"`
	MaxRequests      int64                  `json:"max_requests"`       // Max requests per time period
	MaxTokens        int64                  `json:"max_tokens"`         // Max tokens per request
	AllowedModels    []string               `json:"allowed_models"`     // Whitelist of allowed models
	ForbiddenActions []string               `json:"forbidden_actions"`  // Blacklist of forbidden actions
	RequireApproval  bool                   `json:"require_approval"`   // Whether approval is required
	ModelMetadata    map[string]ModelLimits `json:"model_metadata"`     // Per-model limits (addresses sec11.item2)
	Metadata         map[string]interface{} `json:"metadata,omitempty"` // Additional metadata
}

// ModelLimits defines per-model constraints and metadata.
// Addresses gap sec11.item2 (P2): Model limit checks with metadata evaluation.
type ModelLimits struct {
	MaxContextTokens   int64   `json:"max_context_tokens"`   // Maximum context window
	MaxOutputTokens    int64   `json:"max_output_tokens"`    // Maximum output length
	CostPerToken       float64 `json:"cost_per_token"`       // Cost per token
	MaxCostPerRequest  float64 `json:"max_cost_per_request"` // Maximum cost per request
	RequiresFineTuning bool    `json:"requires_fine_tuning"` // Whether fine-tuning is required
	Deprecated         bool    `json:"deprecated"`           // Whether model is deprecated
}

// UsageContext provides context for capability enforcement checks.
type UsageContext struct {
	CapabilityID string
	ModelName    string
	Action       string
	TokenCount   int64
	RequestCount int64
}

// EnforcementResult contains the result of a capability enforcement check.
type EnforcementResult struct {
	Allowed      bool     `json:"allowed"`
	Reason       string   `json:"reason,omitempty"`
	Violations   []string `json:"violations,omitempty"`
	RequireAuth  bool     `json:"require_auth"`
	ApprovalHint string   `json:"approval_hint,omitempty"`
}

// NewCapabilityEnforcer creates a new capability enforcer.
func NewCapabilityEnforcer() *CapabilityEnforcer {
	return &CapabilityEnforcer{
		matrix: make(map[string]*CapabilityLimits),
	}
}

// RegisterCapability registers capability limits for enforcement.
func (e *CapabilityEnforcer) RegisterCapability(limits *CapabilityLimits) error {
	if limits == nil {
		return errors.New("capability_enforcer: nil limits")
	}
	if limits.CapabilityID == "" {
		return errors.New("capability_enforcer: empty capability_id")
	}

	e.mu.Lock()
	defer e.mu.Unlock()
	
	e.matrix[limits.CapabilityID] = limits
	return nil
}

// Enforce checks whether the given usage context is allowed.
func (e *CapabilityEnforcer) Enforce(ctx *UsageContext) (*EnforcementResult, error) {
	if ctx == nil {
		return nil, errors.New("capability_enforcer: nil context")
	}

	e.mu.RLock()
	limits, exists := e.matrix[ctx.CapabilityID]
	e.mu.RUnlock()

	if !exists {
		return &EnforcementResult{
			Allowed: false,
			Reason:  fmt.Sprintf("capability %s not registered", ctx.CapabilityID),
		}, nil
	}

	result := &EnforcementResult{
		Allowed:     true,
		RequireAuth: limits.RequireApproval,
	}

	violations := []string{}

	// Check token limit
	if limits.MaxTokens > 0 && ctx.TokenCount > limits.MaxTokens {
		violations = append(violations, fmt.Sprintf("token count %d exceeds limit %d", ctx.TokenCount, limits.MaxTokens))
		result.Allowed = false
	}

	// Check request limit
	if limits.MaxRequests > 0 && ctx.RequestCount > limits.MaxRequests {
		violations = append(violations, fmt.Sprintf("request count %d exceeds limit %d", ctx.RequestCount, limits.MaxRequests))
		result.Allowed = false
	}

	// Check model whitelist
	if len(limits.AllowedModels) > 0 {
		modelAllowed := false
		for _, allowed := range limits.AllowedModels {
			if allowed == ctx.ModelName {
				modelAllowed = true
				break
			}
		}
		if !modelAllowed {
			violations = append(violations, fmt.Sprintf("model %s not in allowed list", ctx.ModelName))
			result.Allowed = false
		}
	}

	// Check model-specific metadata limits (sec11.item2)
	if modelLimits, ok := limits.ModelMetadata[ctx.ModelName]; ok {
		// Check if model is deprecated
		if modelLimits.Deprecated {
			violations = append(violations, fmt.Sprintf("model %s is deprecated", ctx.ModelName))
			result.Allowed = false
		}

		// Check context token limit
		if modelLimits.MaxContextTokens > 0 && ctx.TokenCount > modelLimits.MaxContextTokens {
			violations = append(violations, fmt.Sprintf("token count %d exceeds model context limit %d", ctx.TokenCount, modelLimits.MaxContextTokens))
			result.Allowed = false
		}

		// Calculate and check cost
		if modelLimits.CostPerToken > 0 {
			estimatedCost := float64(ctx.TokenCount) * modelLimits.CostPerToken
			if modelLimits.MaxCostPerRequest > 0 && estimatedCost > modelLimits.MaxCostPerRequest {
				violations = append(violations, fmt.Sprintf("estimated cost %.4f exceeds max cost %.4f", estimatedCost, modelLimits.MaxCostPerRequest))
				result.Allowed = false
			}
		}
	}

	// Check forbidden actions
	for _, forbidden := range limits.ForbiddenActions {
		if forbidden == ctx.Action {
			violations = append(violations, fmt.Sprintf("action %s is forbidden", ctx.Action))
			result.Allowed = false
			break
		}
	}

	if len(violations) > 0 {
		result.Violations = violations
		result.Reason = fmt.Sprintf("%d violations detected", len(violations))
	}

	if limits.RequireApproval {
		result.ApprovalHint = fmt.Sprintf("Capability %s requires manual approval", ctx.CapabilityID)
	}

	return result, nil
}

// GetCapability retrieves registered capability limits.
func (e *CapabilityEnforcer) GetCapability(capabilityID string) (*CapabilityLimits, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	
	limits, exists := e.matrix[capabilityID]
	return limits, exists
}

// ListCapabilities returns all registered capability IDs.
func (e *CapabilityEnforcer) ListCapabilities() []string {
	e.mu.RLock()
	defer e.mu.RUnlock()
	
	ids := make([]string, 0, len(e.matrix))
	for id := range e.matrix {
		ids = append(ids, id)
	}
	return ids
}

// UpdateCapability updates existing capability limits.
func (e *CapabilityEnforcer) UpdateCapability(limits *CapabilityLimits) error {
	if limits == nil || limits.CapabilityID == "" {
		return errors.New("capability_enforcer: invalid limits")
	}

	e.mu.Lock()
	defer e.mu.Unlock()
	
	if _, exists := e.matrix[limits.CapabilityID]; !exists {
		return fmt.Errorf("capability_enforcer: capability %s not found", limits.CapabilityID)
	}
	
	e.matrix[limits.CapabilityID] = limits
	return nil
}

// RemoveCapability removes a capability from enforcement.
func (e *CapabilityEnforcer) RemoveCapability(capabilityID string) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	
	if _, exists := e.matrix[capabilityID]; !exists {
		return fmt.Errorf("capability_enforcer: capability %s not found", capabilityID)
	}
	
	delete(e.matrix, capabilityID)
	return nil
}

// ExportMatrix exports the entire capability matrix as JSON.
func (e *CapabilityEnforcer) ExportMatrix() ([]byte, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	
	return json.MarshalIndent(e.matrix, "", "  ")
}

// ImportMatrix imports a capability matrix from JSON.
func (e *CapabilityEnforcer) ImportMatrix(data []byte) error {
	var matrix map[string]*CapabilityLimits
	if err := json.Unmarshal(data, &matrix); err != nil {
		return fmt.Errorf("capability_enforcer: unmarshal matrix: %w", err)
	}

	e.mu.Lock()
	defer e.mu.Unlock()
	
	e.matrix = matrix
	return nil
}

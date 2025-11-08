// Package ai provides integration between AI capability matrix and the main server
// This file implements the integration layer for AI-specific capability enforcement
package ai

import (
	"fmt"
	"strings"
	"time"

	"github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/internal/capability"
)

// ServerIntegration provides integration between AICapabilityMatrix and BetaServer
type ServerIntegration struct {
	matrix          *AICapabilityMatrix
	auditCallback   func(string, map[string]any) // action, metadata
	metricsCallback func(string)                 // metric name to increment
}

// NewServerIntegration creates a new integration layer
func NewServerIntegration() *ServerIntegration {
	integration := &ServerIntegration{
		matrix: NewAICapabilityMatrix(),
	}

	// Set up audit callback for AI decisions
	integration.matrix.SetAuditCallback(integration.handleAIDecisionAudit)

	return integration
}

// SetAuditCallback sets the callback for audit events
func (si *ServerIntegration) SetAuditCallback(callback func(string, map[string]any)) {
	si.auditCallback = callback
}

// SetMetricsCallback sets the callback for metrics events
func (si *ServerIntegration) SetMetricsCallback(callback func(string)) {
	si.metricsCallback = callback
}

// GetMetricsCallback returns the metrics callback if set (nil-safe for callers).
func (si *ServerIntegration) GetMetricsCallback() func(string) {
	return si.metricsCallback
}

// EnableEnforcement enables AI capability enforcement
func (si *ServerIntegration) EnableEnforcement(enabled bool) {
	si.matrix.SetEnforcementActive(enabled)
}

// IsEnforcementEnabled returns whether AI enforcement is enabled
func (si *ServerIntegration) IsEnforcementEnabled() bool {
	return si.matrix.IsEnforcementActive()
}

// EnforceAICapabilities validates AI entity access - main integration point
func (si *ServerIntegration) EnforceAICapabilities(action string, claims map[string]any) (bool, []string, map[string]any) {
	// Extract AI system profile from claims
	profile := si.extractAIProfile(claims)

	// If no AI profile detected, treat as human user (no AI restrictions)
	if profile.EntityType == AIEntityHuman {
		return true, nil, map[string]any{
			"ai_enforcement": "skipped",
			"entity_type":    "human",
		}
	}

	// Perform AI capability enforcement
	decision := si.matrix.EnforceAICapabilities(profile, action, claims)

	// Prepare metadata for audit/metrics
	metadata := map[string]any{
		"ai_enforcement":      "enforced",
		"entity_type":         string(profile.EntityType),
		"system_id":           profile.SystemID,
		"decision":            decision.Decision,
		"reason":              decision.Reason,
		"required_human_auth": decision.RequiredHumanAuth,
		"audit_level":         decision.AuditLevel,
		"applied_policies":    decision.AppliedPolicies,
		"decision_id":         decision.DecisionID,
		"jurisdiction":        profile.Jurisdiction,
		"industry_context":    profile.IndustryContext,
		"risk_level":          profile.RiskLevel,
	}

	// Increment metrics
	if si.metricsCallback != nil {
		if decision.Decision == DecisionAllow {
			si.metricsCallback("ai_capability_enforce_allowed")
		} else {
			si.metricsCallback("ai_capability_enforce_denied")
		}

		// Entity-type specific metrics
		si.metricsCallback(fmt.Sprintf("ai_%s_requests", strings.ToLower(string(profile.EntityType))))
		if decision.Decision == DecisionDeny {
			si.metricsCallback(fmt.Sprintf("ai_%s_denied", strings.ToLower(string(profile.EntityType))))
		}
	}

	// Handle decision
	if decision.Decision == DecisionAllow {
		return true, nil, metadata
	} else {
		// Return missing capabilities for standard capability system integration
		missing := append(decision.MissingCapabilities, decision.ViolatedRules...)
		return false, missing, metadata
	}
}

// extractAIProfile extracts AI system profile from claims
//
//nolint:gocyclo // AI profile extraction with extensive claim validation
func (si *ServerIntegration) extractAIProfile(claims map[string]any) AISystemProfile {
	profile := AISystemProfile{
		EntityType:   AIEntityHuman, // Default to human
		SystemID:     "unknown",
		Jurisdiction: "US",     // Default jurisdiction
		RiskLevel:    "medium", // Default risk level
	}

	// Check for AI entity type indicator
	if entityType, exists := claims["ai_entity_type"]; exists {
		if entityStr, ok := entityType.(string); ok {
			profile.EntityType = AIEntityType(entityStr)
		}
	}

	// If no explicit AI entity type, try to infer from other claims
	if profile.EntityType == AIEntityHuman {
		// Check for AI-related claims that indicate non-human entity
		aiIndicators := []string{
			"ai_entity_verified", "ai_agent_registered", "ai_model_certified",
			"ai_system_registered", "ai_robot_certified", "ai_analytics_approved",
			"ai_automation_certified",
		}

		for _, indicator := range aiIndicators {
			if si.hasClaimValue(claims, indicator) {
				// Infer entity type from specific claims
				switch {
				case si.hasClaimValue(claims, "ai_agent_registered"):
					profile.EntityType = AIEntityAgent
				case si.hasClaimValue(claims, "ai_model_certified"):
					profile.EntityType = AIEntityModel
				case si.hasClaimValue(claims, "ai_system_registered"):
					profile.EntityType = AIEntitySystem
				case si.hasClaimValue(claims, "ai_robot_certified"):
					profile.EntityType = AIEntityRobot
				case si.hasClaimValue(claims, "ai_analytics_approved"):
					profile.EntityType = AIEntityAnalytics
				case si.hasClaimValue(claims, "ai_automation_certified"):
					profile.EntityType = AIEntityAutomation
				default:
					profile.EntityType = AIEntityAssistant // Default AI type
				}
				break
			}
		}
	}

	// Extract other profile fields
	if systemID, exists := claims["system_id"]; exists {
		if systemStr, ok := systemID.(string); ok {
			profile.SystemID = systemStr
		}
	}

	if jurisdiction, exists := claims["jurisdiction"]; exists {
		if jurisdictionStr, ok := jurisdiction.(string); ok {
			profile.Jurisdiction = jurisdictionStr
		}
	}

	if riskLevel, exists := claims["risk_level"]; exists {
		if riskStr, ok := riskLevel.(string); ok {
			profile.RiskLevel = riskStr
		}
	}

	if industry, exists := claims["industry_context"]; exists {
		if industryStr, ok := industry.(string); ok {
			profile.IndustryContext = industryStr
		}
	}

	if modelName, exists := claims["model_name"]; exists {
		if modelStr, ok := modelName.(string); ok {
			profile.ModelName = modelStr
		}
	}

	if modelVersion, exists := claims["model_version"]; exists {
		if versionStr, ok := modelVersion.(string); ok {
			profile.ModelVersion = versionStr
		}
	}

	// Extract compliance flags
	if complianceRaw, exists := claims["compliance_flags"]; exists {
		switch v := complianceRaw.(type) {
		case []string:
			profile.ComplianceFlags = v
		case []any:
			for _, item := range v {
				if str, ok := item.(string); ok {
					profile.ComplianceFlags = append(profile.ComplianceFlags, str)
				}
			}
		case string:
			profile.ComplianceFlags = []string{v}
		}
	}

	// Extract certification authorities
	if certRaw, exists := claims["certified_by"]; exists {
		switch v := certRaw.(type) {
		case []string:
			profile.CertifiedBy = v
		case []any:
			for _, item := range v {
				if str, ok := item.(string); ok {
					profile.CertifiedBy = append(profile.CertifiedBy, str)
				}
			}
		case string:
			profile.CertifiedBy = []string{v}
		}
	}

	return profile
}

// hasClaimValue checks if a claim exists and has a truthy value
func (si *ServerIntegration) hasClaimValue(claims map[string]any, claimName string) bool {
	value, exists := claims[claimName]
	if !exists {
		return false
	}

	switch v := value.(type) {
	case bool:
		return v
	case string:
		return v != "" && v != "false" && v != "0"
	case []string:
		return len(v) > 0
	case []any:
		return len(v) > 0
	default:
		return value != nil
	}
}

// handleAIDecisionAudit handles audit logging for AI enforcement decisions
func (si *ServerIntegration) handleAIDecisionAudit(decision AIEnforcementDecision) {
	if si.auditCallback == nil {
		return
	}

	// Create audit metadata
	auditMetadata := map[string]any{
		"event_type":           "ai_capability_enforcement",
		"decision":             decision.Decision,
		"reason":               decision.Reason,
		"entity_type":          string(decision.SystemProfile.EntityType),
		"system_id":            decision.SystemProfile.SystemID,
		"requested_action":     decision.RequestedAction,
		"applied_policies":     decision.AppliedPolicies,
		"violated_rules":       decision.ViolatedRules,
		"missing_capabilities": decision.MissingCapabilities,
		"required_human_auth":  decision.RequiredHumanAuth,
		"audit_level":          decision.AuditLevel,
		"decision_id":          decision.DecisionID,
		"jurisdiction":         decision.SystemProfile.Jurisdiction,
		"industry_context":     decision.SystemProfile.IndustryContext,
		"risk_level":           decision.SystemProfile.RiskLevel,
		"model_name":           decision.SystemProfile.ModelName,
		"model_version":        decision.SystemProfile.ModelVersion,
		"compliance_flags":     decision.SystemProfile.ComplianceFlags,
		"certified_by":         decision.SystemProfile.CertifiedBy,
		"timestamp":            decision.Timestamp.Format(time.RFC3339),
	}

	// Send to audit system
	si.auditCallback("ai_capability_enforcement", auditMetadata)
}

// GetAICapabilityMatrix returns the underlying capability matrix for direct access
func (si *ServerIntegration) GetAICapabilityMatrix() *AICapabilityMatrix {
	return si.matrix
}

// GetSupportedEntityTypes returns all supported AI entity types
func (si *ServerIntegration) GetSupportedEntityTypes() []string {
	types := si.matrix.GetEntityTypes()
	result := make([]string, len(types))
	for i, t := range types {
		result[i] = string(t)
	}
	return result
}

// GetGovernancePolicies returns all governance policies
func (si *ServerIntegration) GetGovernancePolicies() []AIGovernancePolicy {
	return si.matrix.GetGovernancePolicies()
}

// ValidateAIProfile validates that an AI profile has required fields
func (si *ServerIntegration) ValidateAIProfile(profile AISystemProfile) []string {
	var errors []string

	if profile.SystemID == "" || profile.SystemID == "unknown" {
		errors = append(errors, "system_id is required for AI entities")
	}

	if profile.EntityType == AIEntityHuman {
		return errors // No additional validation for human users
	}

	if profile.Jurisdiction == "" {
		errors = append(errors, "jurisdiction is required for AI entities")
	}

	if profile.RiskLevel == "" {
		errors = append(errors, "risk_level is required for AI entities")
	}

	// Entity-specific validations
	switch profile.EntityType {
	case AIEntityModel:
		if profile.ModelName == "" {
			errors = append(errors, "model_name is required for AI model entities")
		}
	case AIEntityRobot:
		if len(profile.CertifiedBy) == 0 {
			errors = append(errors, "certified_by is required for AI robot entities")
		}
	case AIEntityAgent:
		if profile.RiskLevel == "critical" && len(profile.ComplianceFlags) == 0 {
			errors = append(errors, "compliance_flags required for critical risk AI agents")
		}
	}

	return errors
}

// CreateTestProfile creates a test AI profile for testing/demo purposes
func (si *ServerIntegration) CreateTestProfile(entityType string, jurisdiction string) AISystemProfile {
	return AISystemProfile{
		EntityType:      AIEntityType(entityType),
		SystemID:        fmt.Sprintf("test-%s-%d", entityType, time.Now().Unix()),
		ModelName:       "test-model",
		ModelVersion:    "1.0",
		TrainingDate:    "2025-01-01T00:00:00Z",
		RiskLevel:       "medium",
		IndustryContext: "general",
		Jurisdiction:    jurisdiction,
		CertifiedBy:     []string{"test-authority"},
		ComplianceFlags: []string{"test-compliant"},
	}
}

// ExtendStandardCapabilityEnforcement extends existing capability.ValidateCapabilities
// to include AI-specific governance rules
func (si *ServerIntegration) ExtendStandardCapabilityEnforcement(action string, required []string, provided map[string]bool, claims map[string]any) ([]string, map[string]any) {
	// First run standard capability validation
	missing := capability.ValidateCapabilities(required, provided)

	// Then run AI-specific enforcement
	allowed, aiMissing, aiMetadata := si.EnforceAICapabilities(action, claims)

	// Combine results
	if !allowed {
		missing = append(missing, aiMissing...)
	}

	// Merge metadata
	if aiMetadata != nil {
		if len(missing) == 0 {
			// If standard caps passed but AI enforcement failed, ensure we report denial
			if !allowed {
				missing = aiMissing
			}
		}
	}

	return missing, aiMetadata
}

// GetAICapabilityStatus returns status information about AI capability enforcement
func (si *ServerIntegration) GetAICapabilityStatus() map[string]any {
	policies := si.matrix.GetGovernancePolicies()
	entityTypes := si.matrix.GetEntityTypes()

	policyInfo := make([]map[string]any, len(policies))
	for i, policy := range policies {
		policyInfo[i] = map[string]any{
			"policy_id":            policy.PolicyID,
			"jurisdiction":         policy.Jurisdiction,
			"industry_context":     policy.IndustryContext,
			"compliance_framework": policy.ComplianceFramework,
			"effective_date":       policy.EffectiveDate,
			"last_updated":         policy.LastUpdated,
		}
	}

	entityTypeStrings := make([]string, len(entityTypes))
	for i, t := range entityTypes {
		entityTypeStrings[i] = string(t)
	}

	return map[string]any{
		"enforcement_active":     si.matrix.IsEnforcementActive(),
		"supported_entity_types": entityTypeStrings,
		"loaded_policies":        len(policies),
		"policies":               policyInfo,
		"last_check":             time.Now().Format(time.RFC3339),
	}
}

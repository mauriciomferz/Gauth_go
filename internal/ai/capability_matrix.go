// Package ai provides AI-specific capability governance and enforcement mechanisms.
// This implements the AI Capability & Governance matrix enforcement (sec11.item1)
// with runtime validation of AI entity permissions, system type restrictions,
// and jurisdiction-specific AI governance policies.
package ai

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

// AIEntityType defines the category of AI system making requests
type AIEntityType string

const (
	AIEntityHuman      AIEntityType = "human"      // Human user (no AI restrictions)
	AIEntityAssistant  AIEntityType = "assistant"  // AI assistant/chatbot
	AIEntityAgent      AIEntityType = "agent"      // Autonomous AI agent
	AIEntityModel      AIEntityType = "model"      // Direct model access
	AIEntitySystem     AIEntityType = "system"     // AI system integration
	AIEntityRobot      AIEntityType = "robot"      // Physical AI/robotics
	AIEntityAnalytics  AIEntityType = "analytics"  // AI analytics/ML pipeline
	AIEntityAutomation AIEntityType = "automation" // Process automation AI
)

// Decision constants for capability evaluation
const (
	DecisionAllow = "allow"
	DecisionDeny  = "deny"
)

// ModelMetadata defines operational limits for AI models (sec11.item2 P2).
// These limits are enforced during capability evaluation to prevent resource abuse.
type ModelMetadata struct {
	ModelName         string  `json:"model_name"`
	ModelVersion      string  `json:"model_version"`
	TokenLimitPerCall int     `json:"token_limit_per_call,omitempty"` // Max tokens per request
	TokenLimitDaily   int     `json:"token_limit_daily,omitempty"`    // Daily token budget
	CostLimitPerCall  float64 `json:"cost_limit_per_call,omitempty"`  // Max cost in USD per request
	CostLimitDaily    float64 `json:"cost_limit_daily,omitempty"`     // Daily cost budget in USD
	RateLimitRPM      int     `json:"rate_limit_rpm,omitempty"`       // Requests per minute
	RateLimitRPH      int     `json:"rate_limit_rph,omitempty"`       // Requests per hour
	ContextWindow     int     `json:"context_window,omitempty"`       // Total context window size
	MaxBatchSize      int     `json:"max_batch_size,omitempty"`       // Max batch size for embeddings/etc.
}

// AISystemProfile contains metadata about an AI system for governance decisions
type AISystemProfile struct {
	EntityType      AIEntityType   `json:"entity_type"`
	SystemID        string         `json:"system_id"`
	ModelName       string         `json:"model_name,omitempty"`
	ModelVersion    string         `json:"model_version,omitempty"`
	ModelMetadata   *ModelMetadata `json:"model_metadata,omitempty"`   // Optional model limits (sec11.item2)
	TrainingDate    string         `json:"training_date,omitempty"`    // RFC3339 format
	RiskLevel       string         `json:"risk_level"`                 // low, medium, high, critical
	IndustryContext string         `json:"industry_context,omitempty"` // healthcare, finance, etc.
	Jurisdiction    string         `json:"jurisdiction"`               // US, EU, UK, etc.
	CertifiedBy     []string       `json:"certified_by,omitempty"`     // certification authorities
	ComplianceFlags []string       `json:"compliance_flags,omitempty"` // GDPR, HIPAA, SOX, etc.
}

// AICapabilityRule defines what actions an AI entity type can perform
type AICapabilityRule struct {
	EntityType        AIEntityType `json:"entity_type"`
	AllowedActions    []string     `json:"allowed_actions"`
	ForbiddenActions  []string     `json:"forbidden_actions"`
	RequiredClaims    []string     `json:"required_claims"`     // Additional claims beyond capabilities
	MaxTransactionVal string       `json:"max_transaction_val"` // Dollar amount limit
	TimeWindows       []string     `json:"time_windows"`        // Allowed operating hours
	RequireHumanAuth  bool         `json:"require_human_auth"`  // Require human approval
	AuditLevel        string       `json:"audit_level"`         // none, basic, detailed, realtime
}

// AIGovernancePolicy contains jurisdiction and industry-specific AI rules
type AIGovernancePolicy struct {
	PolicyID            string                            `json:"policy_id"`
	Jurisdiction        string                            `json:"jurisdiction"`
	IndustryContext     string                            `json:"industry_context,omitempty"`
	ComplianceFramework string                            `json:"compliance_framework"` // GDPR, CCPA, AI_ACT, etc.
	EntityRestrictions  map[AIEntityType]AICapabilityRule `json:"entity_restrictions"`
	ProhibitedActions   []string                          `json:"prohibited_actions"`
	MandatoryClaims     []string                          `json:"mandatory_claims"`
	AuditRequirements   []string                          `json:"audit_requirements"`
	EffectiveDate       string                            `json:"effective_date"`            // RFC3339
	ExpirationDate      string                            `json:"expiration_date,omitempty"` // RFC3339
	LastUpdated         string                            `json:"last_updated"`              // RFC3339
}

// AICapabilityMatrix manages AI-specific capability enforcement
type AICapabilityMatrix struct {
	mu                 sync.RWMutex
	entityRules        map[AIEntityType]AICapabilityRule
	governancePolicies map[string]AIGovernancePolicy // keyed by policy_id
	policyIndex        map[string][]string           // jurisdiction -> policy_ids
	industryIndex      map[string][]string           // industry -> policy_ids
	defaultPolicy      *AIGovernancePolicy
	enforcementActive  bool
	auditCallback      func(decision AIEnforcementDecision)
}

// AIEnforcementDecision records the result of AI capability enforcement
type AIEnforcementDecision struct {
	SystemProfile       AISystemProfile `json:"system_profile"`
	RequestedAction     string          `json:"requested_action"`
	ProvidedClaims      map[string]any  `json:"provided_claims"`
	Decision            string          `json:"decision"` // allow, deny
	Reason              string          `json:"reason"`
	AppliedPolicies     []string        `json:"applied_policies"`
	MissingCapabilities []string        `json:"missing_capabilities,omitempty"`
	ViolatedRules       []string        `json:"violated_rules,omitempty"`
	RequiredHumanAuth   bool            `json:"required_human_auth"`
	AuditLevel          string          `json:"audit_level"`
	Timestamp           time.Time       `json:"timestamp"`
	DecisionID          string          `json:"decision_id"`
}

// NewAICapabilityMatrix creates a new AI capability enforcement system
func NewAICapabilityMatrix() *AICapabilityMatrix {
	matrix := &AICapabilityMatrix{
		entityRules:        make(map[AIEntityType]AICapabilityRule),
		governancePolicies: make(map[string]AIGovernancePolicy),
		policyIndex:        make(map[string][]string),
		industryIndex:      make(map[string][]string),
		enforcementActive:  false,
	}

	// Load default AI entity rules
	matrix.loadDefaultEntityRules()
	matrix.loadDefaultGovernancePolicies()

	return matrix
}

// loadDefaultEntityRules sets up baseline AI entity capability restrictions
func (m *AICapabilityMatrix) loadDefaultEntityRules() {
	// Human users have full access
	m.entityRules[AIEntityHuman] = AICapabilityRule{
		EntityType:       AIEntityHuman,
		AllowedActions:   []string{"*"}, // All actions allowed
		ForbiddenActions: []string{},
		RequiredClaims:   []string{},
		RequireHumanAuth: false,
		AuditLevel:       "basic",
	}

	// AI Assistants have restricted access
	m.entityRules[AIEntityAssistant] = AICapabilityRule{
		EntityType: AIEntityAssistant,
		AllowedActions: []string{
			"transaction:read", "transaction:query", "delegation:read",
			"info:read", "status:check", "audit:read",
		},
		ForbiddenActions: []string{
			"transaction:execute", "transaction:pay", "transaction:issue",
			"delegation:create", "delegation:revoke", "admin:*",
		},
		RequiredClaims:   []string{"ai_entity_verified"},
		RequireHumanAuth: false,
		AuditLevel:       "detailed",
	}

	// AI Agents have moderate access but require human auth for sensitive ops
	m.entityRules[AIEntityAgent] = AICapabilityRule{
		EntityType: AIEntityAgent,
		AllowedActions: []string{
			"transaction:read", "transaction:query", "transaction:execute",
			"delegation:read", "info:read", "status:check", "audit:read",
		},
		ForbiddenActions: []string{
			"transaction:pay", "transaction:issue",
			"delegation:create", "delegation:revoke", "admin:*",
		},
		RequiredClaims:    []string{"ai_entity_verified", "ai_agent_registered"},
		RequireHumanAuth:  true,      // Require human approval for transactions
		MaxTransactionVal: "1000.00", // $1000 limit
		AuditLevel:        "realtime",
	}

	// Direct Model access is highly restricted
	m.entityRules[AIEntityModel] = AICapabilityRule{
		EntityType:     AIEntityModel,
		AllowedActions: []string{"info:read", "status:check"},
		ForbiddenActions: []string{
			"transaction:*", "delegation:*", "admin:*",
		},
		RequiredClaims:   []string{"ai_model_certified", "ai_entity_verified"},
		RequireHumanAuth: true,
		AuditLevel:       "realtime",
	}

	// AI Systems (integration) have API-level access
	m.entityRules[AIEntitySystem] = AICapabilityRule{
		EntityType: AIEntitySystem,
		AllowedActions: []string{
			"transaction:read", "transaction:query",
			"delegation:read", "info:read", "status:check",
		},
		ForbiddenActions: []string{
			"transaction:execute", "transaction:pay", "transaction:issue",
			"delegation:create", "delegation:revoke", "admin:*",
		},
		RequiredClaims:   []string{"ai_system_registered", "ai_entity_verified"},
		RequireHumanAuth: false,
		AuditLevel:       "detailed",
	}

	// Robotics AI has physical-world restrictions
	m.entityRules[AIEntityRobot] = AICapabilityRule{
		EntityType: AIEntityRobot,
		AllowedActions: []string{
			"transaction:read", "transaction:query", "status:check",
		},
		ForbiddenActions: []string{
			"transaction:execute", "transaction:pay", "transaction:issue",
			"delegation:*", "admin:*",
		},
		RequiredClaims:    []string{"ai_robot_certified", "ai_entity_verified", "physical_safety_cert"},
		RequireHumanAuth:  true,
		MaxTransactionVal: "100.00", // Very low limits for safety
		AuditLevel:        "realtime",
	}

	// Analytics AI for data processing
	m.entityRules[AIEntityAnalytics] = AICapabilityRule{
		EntityType: AIEntityAnalytics,
		AllowedActions: []string{
			"transaction:read", "transaction:query", "audit:read",
			"info:read", "status:check",
		},
		ForbiddenActions: []string{
			"transaction:execute", "transaction:pay", "transaction:issue",
			"delegation:*", "admin:*",
		},
		RequiredClaims:   []string{"ai_analytics_approved", "ai_entity_verified"},
		RequireHumanAuth: false,
		AuditLevel:       "detailed",
	}

	// Process Automation AI
	m.entityRules[AIEntityAutomation] = AICapabilityRule{
		EntityType: AIEntityAutomation,
		AllowedActions: []string{
			"transaction:read", "transaction:query", "transaction:execute",
			"delegation:read", "status:check",
		},
		ForbiddenActions: []string{
			"transaction:pay", "transaction:issue",
			"delegation:create", "delegation:revoke", "admin:*",
		},
		RequiredClaims:    []string{"ai_automation_certified", "ai_entity_verified"},
		RequireHumanAuth:  true, // Require approval for execution
		MaxTransactionVal: "500.00",
		TimeWindows:       []string{"09:00-17:00"}, // Business hours only
		AuditLevel:        "realtime",
	}
}

// loadDefaultGovernancePolicies sets up jurisdiction-specific AI governance rules
func (m *AICapabilityMatrix) loadDefaultGovernancePolicies() {
	// EU AI Act compliance
	euPolicy := AIGovernancePolicy{
		PolicyID:            "eu_ai_act_2025",
		Jurisdiction:        "EU",
		ComplianceFramework: "EU_AI_ACT",
		EntityRestrictions: map[AIEntityType]AICapabilityRule{
			AIEntityAgent: {
				EntityType:       AIEntityAgent,
				AllowedActions:   []string{"transaction:read", "info:read", "status:check"},
				ForbiddenActions: []string{"transaction:execute", "transaction:pay", "delegation:*"},
				RequiredClaims:   []string{"eu_ai_conformity", "ai_risk_assessment", "human_oversight"},
				RequireHumanAuth: true,
				AuditLevel:       "realtime",
			},
			AIEntityModel: {
				EntityType:       AIEntityModel,
				AllowedActions:   []string{"info:read"},
				ForbiddenActions: []string{"transaction:*", "delegation:*"},
				RequiredClaims:   []string{"eu_ai_conformity", "ce_marking", "transparency_report"},
				RequireHumanAuth: true,
				AuditLevel:       "realtime",
			},
		},
		ProhibitedActions: []string{"transaction:pay", "delegation:create"}, // High-risk systems
		MandatoryClaims:   []string{"eu_ai_conformity", "human_oversight"},
		AuditRequirements: []string{"realtime_monitoring", "bias_testing", "transparency_log"},
		EffectiveDate:     "2025-08-01T00:00:00Z",
		LastUpdated:       time.Now().Format(time.RFC3339),
	}

	// US AI governance (sectoral approach)
	usPolicy := AIGovernancePolicy{
		PolicyID:            "us_ai_governance_2025",
		Jurisdiction:        "US",
		ComplianceFramework: "NIST_AI_RMF",
		EntityRestrictions: map[AIEntityType]AICapabilityRule{
			AIEntityAgent: {
				EntityType:        AIEntityAgent,
				AllowedActions:    []string{"transaction:read", "transaction:query", "transaction:execute"},
				RequiredClaims:    []string{"nist_ai_compliance", "algorithmic_accountability"},
				RequireHumanAuth:  false, // More permissive than EU
				MaxTransactionVal: "5000.00",
				AuditLevel:        "detailed",
			},
		},
		MandatoryClaims:   []string{"algorithmic_accountability"},
		AuditRequirements: []string{"impact_assessment", "bias_monitoring"},
		EffectiveDate:     "2025-01-01T00:00:00Z",
		LastUpdated:       time.Now().Format(time.RFC3339),
	}

	// UK AI governance principles
	ukPolicy := AIGovernancePolicy{
		PolicyID:            "uk_ai_principles_2025",
		Jurisdiction:        "UK",
		ComplianceFramework: "UK_AI_PRINCIPLES",
		EntityRestrictions: map[AIEntityType]AICapabilityRule{
			AIEntityAgent: {
				EntityType:        AIEntityAgent,
				AllowedActions:    []string{"transaction:read", "transaction:query", "transaction:execute"},
				RequiredClaims:    []string{"uk_ai_principles", "explainability", "fairness_assessment"},
				RequireHumanAuth:  false,
				MaxTransactionVal: "2500.00",
				AuditLevel:        "detailed",
			},
		},
		MandatoryClaims:   []string{"explainability", "fairness_assessment"},
		AuditRequirements: []string{"explainability_log", "fairness_monitoring"},
		EffectiveDate:     "2025-03-01T00:00:00Z",
		LastUpdated:       time.Now().Format(time.RFC3339),
	}

	// Healthcare industry-specific rules (HIPAA compliance for AI)
	healthcarePolicy := AIGovernancePolicy{
		PolicyID:            "healthcare_ai_hipaa_2025",
		IndustryContext:     "healthcare",
		ComplianceFramework: "HIPAA_AI",
		EntityRestrictions: map[AIEntityType]AICapabilityRule{
			AIEntityAssistant: {
				EntityType:       AIEntityAssistant,
				AllowedActions:   []string{"info:read", "status:check"},
				ForbiddenActions: []string{"transaction:*", "delegation:*"},
				RequiredClaims:   []string{"hipaa_compliance", "phi_protection", "healthcare_cert"},
				RequireHumanAuth: true,
				AuditLevel:       "realtime",
			},
			AIEntityAnalytics: {
				EntityType:       AIEntityAnalytics,
				AllowedActions:   []string{"transaction:read", "audit:read"},
				RequiredClaims:   []string{"hipaa_compliance", "de_identification", "healthcare_cert"},
				RequireHumanAuth: true,
				AuditLevel:       "realtime",
			},
		},
		ProhibitedActions: []string{"transaction:pay", "delegation:create"},
		MandatoryClaims:   []string{"hipaa_compliance", "phi_protection"},
		AuditRequirements: []string{"phi_access_log", "de_identification_audit", "breach_monitoring"},
		EffectiveDate:     "2025-01-01T00:00:00Z",
		LastUpdated:       time.Now().Format(time.RFC3339),
	}

	// Financial services AI governance (SOX, banking regulations)
	financePolicy := AIGovernancePolicy{
		PolicyID:            "finance_ai_compliance_2025",
		IndustryContext:     "finance",
		ComplianceFramework: "SOX_AI_BANKING",
		EntityRestrictions: map[AIEntityType]AICapabilityRule{
			AIEntityAgent: {
				EntityType:        AIEntityAgent,
				AllowedActions:    []string{"transaction:read", "transaction:query"},
				ForbiddenActions:  []string{"transaction:execute", "transaction:pay", "delegation:*"},
				RequiredClaims:    []string{"sox_compliance", "financial_cert", "model_validation"},
				RequireHumanAuth:  true,
				MaxTransactionVal: "100.00", // Very conservative for AI
				AuditLevel:        "realtime",
			},
			AIEntityAutomation: {
				EntityType:       AIEntityAutomation,
				AllowedActions:   []string{"transaction:read", "status:check"},
				RequiredClaims:   []string{"sox_compliance", "financial_cert", "operational_risk_approval"},
				RequireHumanAuth: true,
				TimeWindows:      []string{"09:00-16:00"}, // Market hours only
				AuditLevel:       "realtime",
			},
		},
		ProhibitedActions: []string{"transaction:pay", "delegation:create"},
		MandatoryClaims:   []string{"sox_compliance", "financial_cert"},
		AuditRequirements: []string{"transaction_log", "model_validation_audit", "operational_risk_monitoring"},
		EffectiveDate:     "2025-01-01T00:00:00Z",
		LastUpdated:       time.Now().Format(time.RFC3339),
	}

	// Store policies
	policies := []AIGovernancePolicy{euPolicy, usPolicy, ukPolicy, healthcarePolicy, financePolicy}
	for _, policy := range policies {
		m.governancePolicies[policy.PolicyID] = policy

		// Index by jurisdiction
		if policy.Jurisdiction != "" {
			m.policyIndex[policy.Jurisdiction] = append(m.policyIndex[policy.Jurisdiction], policy.PolicyID)
		}

		// Index by industry
		if policy.IndustryContext != "" {
			m.industryIndex[policy.IndustryContext] = append(m.industryIndex[policy.IndustryContext], policy.PolicyID)
		}
	}

	// Set US policy as default for backwards compatibility
	m.defaultPolicy = &usPolicy
}

// EnforceAICapabilities validates AI entity access to requested actions
func (m *AICapabilityMatrix) EnforceAICapabilities(
	profile AISystemProfile,
	action string,
	claims map[string]any,
) AIEnforcementDecision {
	m.mu.RLock()
	defer m.mu.RUnlock()

	decision := AIEnforcementDecision{
		SystemProfile:   profile,
		RequestedAction: action,
		ProvidedClaims:  claims,
		Decision:        DecisionDeny, // Default to deny
		Timestamp:       time.Now(),
		DecisionID:      fmt.Sprintf("ai-decision-%d", time.Now().UnixNano()),
	}

	// Ensure audit callback is always triggered at the end
	defer func() {
		if m.auditCallback != nil {
			go m.auditCallback(decision)
		}
	}()

	// If enforcement is disabled, allow with audit
	if !m.enforcementActive {
		decision.Decision = DecisionAllow
		decision.Reason = "AI capability enforcement disabled"
		decision.AuditLevel = "basic"
		return decision
	}

	// Get applicable policies for this AI system
	applicablePolicies := m.getApplicablePolicies(profile)
	decision.AppliedPolicies = make([]string, len(applicablePolicies))
	for i, policy := range applicablePolicies {
		decision.AppliedPolicies[i] = policy.PolicyID
	}

	// Check base entity type rules
	entityRule, exists := m.entityRules[profile.EntityType]
	if !exists {
		decision.Reason = fmt.Sprintf("Unknown AI entity type: %s", profile.EntityType)
		decision.ViolatedRules = []string{"unknown_entity_type"}
		return decision
	}

	// Check if action is explicitly forbidden
	if m.isActionForbidden(action, entityRule, applicablePolicies) {
		decision.Reason = fmt.Sprintf("Action %s is forbidden for AI entity type %s", action, profile.EntityType)
		decision.ViolatedRules = []string{"forbidden_action"}
		return decision
	}

	// Check if action is allowed by entity type
	if !m.isActionAllowed(action, entityRule) {
		decision.Reason = fmt.Sprintf("Action %s is not allowed for AI entity type %s", action, profile.EntityType)
		decision.ViolatedRules = []string{"action_not_allowed"}
		return decision
	}

	// Validate required claims
	missingClaims := m.validateRequiredClaims(entityRule, applicablePolicies, claims)
	if len(missingClaims) > 0 {
		decision.Reason = fmt.Sprintf("Missing required claims: %s", strings.Join(missingClaims, ", "))
		decision.MissingCapabilities = missingClaims
		decision.ViolatedRules = []string{"missing_required_claims"}
		return decision
	}

	// Check time window restrictions
	if !m.isWithinAllowedTimeWindow(entityRule) {
		decision.Reason = "Action attempted outside of allowed time window"
		decision.ViolatedRules = []string{"time_window_violation"}
		return decision
	}

	// Determine human authorization requirement
	decision.RequiredHumanAuth = entityRule.RequireHumanAuth
	for _, policy := range applicablePolicies {
		if restriction, exists := policy.EntityRestrictions[profile.EntityType]; exists {
			if restriction.RequireHumanAuth {
				decision.RequiredHumanAuth = true
			}
		}
	}

	// Determine audit level (highest level required)
	decision.AuditLevel = entityRule.AuditLevel
	for _, policy := range applicablePolicies {
		if restriction, exists := policy.EntityRestrictions[profile.EntityType]; exists {
			if m.getAuditLevelPriority(restriction.AuditLevel) > m.getAuditLevelPriority(decision.AuditLevel) {
				decision.AuditLevel = restriction.AuditLevel
			}
		}
	}

	// Check model metadata limits (sec11.item2 P2)
	if profile.ModelMetadata != nil {
		if limitViolation := m.checkModelLimits(profile.ModelMetadata, claims); limitViolation != "" {
			decision.Reason = limitViolation
			decision.ViolatedRules = []string{"model_limit_exceeded"}
			return decision
		}
	}

	// All checks passed - allow the action
	decision.Decision = DecisionAllow
	decision.Reason = "AI capability enforcement checks passed"

	return decision
}

// getApplicablePolicies finds governance policies that apply to the AI system
func (m *AICapabilityMatrix) getApplicablePolicies(profile AISystemProfile) []AIGovernancePolicy {
	var applicable []AIGovernancePolicy

	// Human entities are not subject to AI governance policies
	if profile.EntityType == AIEntityHuman {
		return applicable // Return empty slice
	}

	// Add jurisdiction-specific policies
	if policyIDs, exists := m.policyIndex[profile.Jurisdiction]; exists {
		for _, policyID := range policyIDs {
			if policy, exists := m.governancePolicies[policyID]; exists {
				applicable = append(applicable, policy)
			}
		}
	}

	// Add industry-specific policies
	if profile.IndustryContext != "" {
		if policyIDs, exists := m.industryIndex[profile.IndustryContext]; exists {
			for _, policyID := range policyIDs {
				if policy, exists := m.governancePolicies[policyID]; exists {
					// Avoid duplicates
					found := false
					for _, existing := range applicable {
						if existing.PolicyID == policy.PolicyID {
							found = true
							break
						}
					}
					if !found {
						applicable = append(applicable, policy)
					}
				}
			}
		}
	}

	// If no specific policies found, use default (but not for human entities)
	if len(applicable) == 0 && m.defaultPolicy != nil && profile.EntityType != AIEntityHuman {
		applicable = append(applicable, *m.defaultPolicy)
	}

	return applicable
}

// isActionForbidden checks if an action is explicitly forbidden
func (m *AICapabilityMatrix) isActionForbidden(action string, entityRule AICapabilityRule, policies []AIGovernancePolicy) bool {
	// Check entity-level forbidden actions
	for _, forbidden := range entityRule.ForbiddenActions {
		if m.matchesActionPattern(action, forbidden) {
			return true
		}
	}

	// Check policy-level forbidden actions
	for _, policy := range policies {
		for _, forbidden := range policy.ProhibitedActions {
			if m.matchesActionPattern(action, forbidden) {
				return true
			}
		}

		// Check entity-specific restrictions in policy
		if restriction, exists := policy.EntityRestrictions[entityRule.EntityType]; exists {
			for _, forbidden := range restriction.ForbiddenActions {
				if m.matchesActionPattern(action, forbidden) {
					return true
				}
			}
		}
	}

	return false
}

// isActionAllowed checks if an action is explicitly allowed
func (m *AICapabilityMatrix) isActionAllowed(action string, entityRule AICapabilityRule) bool {
	for _, allowed := range entityRule.AllowedActions {
		if m.matchesActionPattern(action, allowed) {
			return true
		}
	}
	return false
}

// matchesActionPattern checks if an action matches a pattern (supports wildcards)
func (m *AICapabilityMatrix) matchesActionPattern(action, pattern string) bool {
	if pattern == "*" {
		return true
	}
	if pattern == action {
		return true
	}
	if strings.HasSuffix(pattern, ":*") {
		prefix := strings.TrimSuffix(pattern, ":*")
		return strings.HasPrefix(action, prefix+":")
	}
	if strings.HasSuffix(pattern, "*") {
		prefix := strings.TrimSuffix(pattern, "*")
		return strings.HasPrefix(action, prefix)
	}
	return false
}

// validateRequiredClaims checks if all required claims are present
func (m *AICapabilityMatrix) validateRequiredClaims(
	entityRule AICapabilityRule,
	policies []AIGovernancePolicy,
	claims map[string]any,
) []string {
	requiredClaims := make(map[string]bool)

	// Add entity-level required claims
	for _, claim := range entityRule.RequiredClaims {
		requiredClaims[claim] = true
	}

	// Add policy-level mandatory claims
	for _, policy := range policies {
		for _, claim := range policy.MandatoryClaims {
			requiredClaims[claim] = true
		}

		// Add entity-specific required claims from policy
		if restriction, exists := policy.EntityRestrictions[entityRule.EntityType]; exists {
			for _, claim := range restriction.RequiredClaims {
				requiredClaims[claim] = true
			}
		}
	} // Check which claims are missing
	var missing []string
	for claim := range requiredClaims {
		if !m.hasClaimValue(claims, claim) {
			missing = append(missing, claim)
		}
	}

	return missing
}

// hasClaimValue checks if a claim exists and has a truthy value
func (m *AICapabilityMatrix) hasClaimValue(claims map[string]any, claimName string) bool {
	value, exists := claims[claimName]
	if !exists {
		return false
	}

	// Check various truthy values
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

// isWithinAllowedTimeWindow checks if current time is within allowed windows
func (m *AICapabilityMatrix) isWithinAllowedTimeWindow(entityRule AICapabilityRule) bool {
	if len(entityRule.TimeWindows) == 0 {
		return true // No restrictions
	}

	now := time.Now()
	currentTime := now.Format("15:04")

	for _, window := range entityRule.TimeWindows {
		if strings.Contains(window, "-") {
			parts := strings.Split(window, "-")
			if len(parts) == 2 {
				start := strings.TrimSpace(parts[0])
				end := strings.TrimSpace(parts[1])

				if currentTime >= start && currentTime <= end {
					return true
				}
			}
		}
	}

	return false
}

// getAuditLevelPriority returns numeric priority for audit levels
func (m *AICapabilityMatrix) getAuditLevelPriority(level string) int {
	switch level {
	case "none":
		return 0
	case "basic":
		return 1
	case "detailed":
		return 2
	case "realtime":
		return 3
	default:
		return 1 // default to basic
	}
}

// checkModelLimits validates model operational limits against request claims (sec11.item2 P2).
// Returns empty string if all limits are satisfied, or violation message if any limit is exceeded.
func (m *AICapabilityMatrix) checkModelLimits(metadata *ModelMetadata, claims map[string]any) string {
	if metadata == nil {
		return "" // No limits configured
	}

	// Check token limit per call
	if metadata.TokenLimitPerCall > 0 {
		if tokens, ok := claims["requested_tokens"].(int); ok && tokens > metadata.TokenLimitPerCall {
			return fmt.Sprintf("requested tokens %d exceeds model limit %d per call", tokens, metadata.TokenLimitPerCall)
		}
		if tokensFloat, ok := claims["requested_tokens"].(float64); ok && int(tokensFloat) > metadata.TokenLimitPerCall {
			return fmt.Sprintf("requested tokens %d exceeds model limit %d per call", int(tokensFloat), metadata.TokenLimitPerCall)
		}
	}

	// Check cost limit per call
	if metadata.CostLimitPerCall > 0 {
		if cost, ok := claims["estimated_cost"].(float64); ok && cost > metadata.CostLimitPerCall {
			return fmt.Sprintf("estimated cost %.4f USD exceeds model limit %.4f USD per call", cost, metadata.CostLimitPerCall)
		}
	}

	// Check context window size
	if metadata.ContextWindow > 0 {
		if contextSize, ok := claims["context_size"].(int); ok && contextSize > metadata.ContextWindow {
			return fmt.Sprintf("context size %d exceeds model context window %d", contextSize, metadata.ContextWindow)
		}
		if contextFloat, ok := claims["context_size"].(float64); ok && int(contextFloat) > metadata.ContextWindow {
			return fmt.Sprintf("context size %d exceeds model context window %d", int(contextFloat), metadata.ContextWindow)
		}
	}

	// Check batch size limit
	if metadata.MaxBatchSize > 0 {
		if batchSize, ok := claims["batch_size"].(int); ok && batchSize > metadata.MaxBatchSize {
			return fmt.Sprintf("batch size %d exceeds model limit %d", batchSize, metadata.MaxBatchSize)
		}
		if batchFloat, ok := claims["batch_size"].(float64); ok && int(batchFloat) > metadata.MaxBatchSize {
			return fmt.Sprintf("batch size %d exceeds model limit %d", int(batchFloat), metadata.MaxBatchSize)
		}
	}

	// Note: Daily limits (TokenLimitDaily, CostLimitDaily) and rate limits (RateLimitRPM, RateLimitRPH)
	// require stateful tracking across requests. Consider implementing with external rate limiter
	// or time-series database for production use.

	return "" // All checks passed
}

// SetEnforcementActive enables or disables AI capability enforcement
func (m *AICapabilityMatrix) SetEnforcementActive(active bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.enforcementActive = active
}

// IsEnforcementActive returns whether AI capability enforcement is active
func (m *AICapabilityMatrix) IsEnforcementActive() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.enforcementActive
}

// SetAuditCallback sets a callback function for enforcement decisions
func (m *AICapabilityMatrix) SetAuditCallback(callback func(AIEnforcementDecision)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.auditCallback = callback
}

// GetEntityTypes returns all supported AI entity types
func (m *AICapabilityMatrix) GetEntityTypes() []AIEntityType {
	m.mu.RLock()
	defer m.mu.RUnlock()

	types := make([]AIEntityType, 0, len(m.entityRules))
	for entityType := range m.entityRules {
		types = append(types, entityType)
	}
	return types
}

// GetGovernancePolicies returns all loaded governance policies
func (m *AICapabilityMatrix) GetGovernancePolicies() []AIGovernancePolicy {
	m.mu.RLock()
	defer m.mu.RUnlock()

	policies := make([]AIGovernancePolicy, 0, len(m.governancePolicies))
	for _, policy := range m.governancePolicies {
		policies = append(policies, policy)
	}
	return policies
}

// AddGovernancePolicy adds a new governance policy to the matrix
func (m *AICapabilityMatrix) AddGovernancePolicy(policy AIGovernancePolicy) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.governancePolicies[policy.PolicyID] = policy

	// Update indexes
	if policy.Jurisdiction != "" {
		m.policyIndex[policy.Jurisdiction] = append(m.policyIndex[policy.Jurisdiction], policy.PolicyID)
	}
	if policy.IndustryContext != "" {
		m.industryIndex[policy.IndustryContext] = append(m.industryIndex[policy.IndustryContext], policy.PolicyID)
	}
}

// UpdateEntityRule updates or adds an entity type rule
func (m *AICapabilityMatrix) UpdateEntityRule(entityType AIEntityType, rule AICapabilityRule) {
	m.mu.Lock()
	defer m.mu.Unlock()
	rule.EntityType = entityType
	m.entityRules[entityType] = rule
}

// GetEntityRule returns the rule for a specific entity type
func (m *AICapabilityMatrix) GetEntityRule(entityType AIEntityType) (AICapabilityRule, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	rule, exists := m.entityRules[entityType]
	return rule, exists
}

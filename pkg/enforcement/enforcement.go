// Package enforcement provides rule-based and disclosure-based enforcement mechanisms
// for the GAuth authorization framework with AI integration points.
package enforcement

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// EnforcementType defines the type of enforcement mechanism
type EnforcementType string

const (
	EnforcementRuleBased  EnforcementType = "rule-based"
	EnforcementDisclosure EnforcementType = "disclosure"
	EnforcementAIAssisted EnforcementType = "ai-assisted"
	EnforcementHybrid     EnforcementType = "hybrid"

	// Decision constants
	DecisionAllow  = "allow"
	DecisionDeny   = "deny"
	DecisionReview = "review"
)

// EnforcementDecision represents the result of an enforcement evaluation
type EnforcementDecision struct {
	Decision         string                  `json:"decision"` // "allow", "deny", "conditional"
	Reason           string                  `json:"reason"`
	AppliedRules     []string                `json:"applied_rules"`
	Disclosures      []DisclosureRequirement `json:"disclosures"`
	AIRecommendation *AIRecommendation       `json:"ai_recommendation,omitempty"`
	Timestamp        time.Time               `json:"timestamp"`
	EnforcementType  EnforcementType         `json:"enforcement_type"`
	Subject          string                  `json:"subject"`
	Resource         string                  `json:"resource"`
	Action           string                  `json:"action"`
	Metadata         map[string]interface{}  `json:"metadata"`
}

// DisclosureRequirement represents a transparency disclosure requirement
type DisclosureRequirement struct {
	Type         string `json:"type"` // "data-usage", "ai-involvement", "third-party-sharing"
	Description  string `json:"description"`
	Required     bool   `json:"required"`
	Acknowledged bool   `json:"acknowledged"`
}

// AIRecommendation represents AI-assisted enforcement recommendation
type AIRecommendation struct {
	Confidence float64 `json:"confidence"` // 0.0 - 1.0
	Suggestion string  `json:"suggestion"` // "allow", "deny", "review"
	Reasoning  string  `json:"reasoning"`
	AgentID    string  `json:"agent_id"`
}

// Rule represents an enforcement rule
type Rule struct {
	ID          string
	Name        string
	Description string
	Condition   func(ctx context.Context, req *EnforcementRequest) bool
	Action      string // "allow", "deny"
	Priority    int
	Enabled     bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// EnforcementRequest represents a request for enforcement evaluation.
//
// Field Semantics:
//
// Subject: Identifier of the user or AI agent performing the action.
//   Examples: "user:alice@example.com", "agent:diagnostic-ai-v2", "service:billing-processor"
//
// Resource: Semantic resource identifier representing the target of the action.
//   This should express INTENT, not implementation details.
//   Examples:
//     - Simple: "patient-record:12345", "financial-report:Q4-2024"
//     - Hierarchical: "medical:patient:12345:medications", "cloud:storage:bucket:documents"
//     - Complex operations: "treatment-plan:12345:recommend", "diagnosis:generate:patient:12345"
//   Avoid low-level paths like "POST /api/v1/patients/12345/medications?action=add"
//
// Action: Semantic action type describing the operation intent.
//   Beyond simple read/write - use domain-specific verbs:
//   Examples:
//     - Data operations: "read", "write", "update", "delete", "create"
//     - Medical domain: "diagnose", "prescribe", "analyze", "recommend"
//     - Financial domain: "transfer", "approve", "audit", "reconcile"
//     - AI operations: "train", "infer", "explain", "delegate"
//     - Hierarchical: "medical.diagnose", "financial.transaction.execute"
//
// Context: Additional contextual information for enforcement decisions.
//   Examples: time-of-day, geographic location, authentication method, risk indicators
//
// Disclosures: Required transparency disclosures (e.g., AI involvement, data usage)
type EnforcementRequest struct {
	Subject     string                  `json:"subject"`
	Resource    string                  `json:"resource"`
	Action      string                  `json:"action"`
	Context     map[string]interface{}  `json:"context"`
	Disclosures []DisclosureRequirement `json:"disclosures"`
}

// Enforcer is the main enforcement engine
type Enforcer struct {
	rules             map[string]*Rule
	mu                sync.RWMutex
	disclosureManager *DisclosureManager
	aiIntegration     AIIntegrationInterface
	auditCallback     func(EnforcementDecision)
}

// NewEnforcer creates a new enforcement engine
func NewEnforcer() *Enforcer {
	return &Enforcer{
		rules:             make(map[string]*Rule),
		disclosureManager: NewDisclosureManager(),
	}
}

// AddRule adds an enforcement rule
func (e *Enforcer) AddRule(rule *Rule) error {
	if rule.ID == "" {
		return fmt.Errorf("rule ID cannot be empty")
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	rule.CreatedAt = time.Now()
	rule.UpdatedAt = time.Now()
	e.rules[rule.ID] = rule
	return nil
}

// RemoveRule removes an enforcement rule
func (e *Enforcer) RemoveRule(ruleID string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if _, exists := e.rules[ruleID]; !exists {
		return fmt.Errorf("rule %s not found", ruleID)
	}

	delete(e.rules, ruleID)
	return nil
}

// Evaluate evaluates an enforcement request
func (e *Enforcer) Evaluate(ctx context.Context, req *EnforcementRequest) (EnforcementDecision, error) {
	decision := EnforcementDecision{
		Decision:        "allow",
		Reason:          "No rules triggered",
		AppliedRules:    []string{},
		Disclosures:     []DisclosureRequirement{},
		Timestamp:       time.Now(),
		EnforcementType: EnforcementRuleBased,
		Subject:         req.Subject,
		Resource:        req.Resource,
		Action:          req.Action,
		Metadata:        make(map[string]interface{}),
	}

	// Evaluate rule-based enforcement
	ruleDecision, appliedRules := e.evaluateRules(ctx, req)
	if ruleDecision != "allow" {
		decision.Decision = ruleDecision
		decision.Reason = "Rule-based enforcement triggered"
		decision.AppliedRules = appliedRules
	}

	// Evaluate disclosure requirements
	disclosures, disclosureOk := e.disclosureManager.EvaluateDisclosures(req)
	if !disclosureOk {
		decision.Decision = DecisionDeny
		decision.Reason = "Required disclosures not acknowledged"
		decision.Disclosures = disclosures
		decision.EnforcementType = EnforcementDisclosure
	}

	// Evaluate AI-assisted enforcement if available
	if e.aiIntegration != nil {
		aiRec, err := e.aiIntegration.EvaluateEnforcement(ctx, req)
		if err == nil && aiRec != nil {
			decision.AIRecommendation = aiRec
			decision.EnforcementType = EnforcementHybrid
			decision.Metadata["ai_agent_id"] = aiRec.AgentID
		}
	}

	// Audit logging
	if e.auditCallback != nil {
		go e.auditCallback(decision)
	}

	return decision, nil
}

// evaluateRules evaluates all rules and returns decision and applied rule IDs
func (e *Enforcer) evaluateRules(ctx context.Context, req *EnforcementRequest) (string, []string) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	appliedRules := []string{}
	decision := "allow"

	// Sort rules by priority (lower number = higher priority)
	for _, rule := range e.rules {
		if !rule.Enabled {
			continue
		}

		if rule.Condition(ctx, req) {
			appliedRules = append(appliedRules, rule.ID)
			if rule.Action == DecisionDeny {
				decision = DecisionDeny
				break // Deny takes precedence
			}
		}
	}

	return decision, appliedRules
}

// SetAIIntegration sets the AI integration for enforcement
func (e *Enforcer) SetAIIntegration(ai AIIntegrationInterface) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.aiIntegration = ai
}

// SetAuditCallback sets the audit callback for enforcement decisions
func (e *Enforcer) SetAuditCallback(callback func(EnforcementDecision)) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.auditCallback = callback
}

// GetMetrics returns enforcement engine metrics
func (e *Enforcer) GetMetrics() map[string]interface{} {
	e.mu.RLock()
	defer e.mu.RUnlock()

	totalRules := len(e.rules)
	enabledRules := 0
	for _, rule := range e.rules {
		if rule.Enabled {
			enabledRules++
		}
	}

	return map[string]interface{}{
		"total_rules":   totalRules,
		"enabled_rules": enabledRules,
		"ai_enabled":    e.aiIntegration != nil,
		"disclosures":   e.disclosureManager.GetDisclosures(),
	}
}

// DisclosureManager manages disclosure requirements
type DisclosureManager struct {
	disclosures []DisclosureRequirement
	mu          sync.RWMutex
}

// NewDisclosureManager creates a new disclosure manager
func NewDisclosureManager() *DisclosureManager {
	return &DisclosureManager{
		disclosures: []DisclosureRequirement{},
	}
}

// AddDisclosure adds a disclosure requirement
func (dm *DisclosureManager) AddDisclosure(disclosure DisclosureRequirement) {
	dm.mu.Lock()
	defer dm.mu.Unlock()
	dm.disclosures = append(dm.disclosures, disclosure)
}

// EvaluateDisclosures checks if all required disclosures are acknowledged
func (dm *DisclosureManager) EvaluateDisclosures(req *EnforcementRequest) ([]DisclosureRequirement, bool) {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	acknowledgedMap := make(map[string]bool)
	for _, d := range req.Disclosures {
		if d.Acknowledged {
			acknowledgedMap[d.Type] = true
		}
	}

	unacknowledged := []DisclosureRequirement{}
	allAcknowledged := true

	for _, disclosure := range dm.disclosures {
		if disclosure.Required && !acknowledgedMap[disclosure.Type] {
			unacknowledged = append(unacknowledged, disclosure)
			allAcknowledged = false
		}
	}

	return unacknowledged, allAcknowledged
}

// GetDisclosures returns all disclosure requirements
func (dm *DisclosureManager) GetDisclosures() []DisclosureRequirement {
	dm.mu.RLock()
	defer dm.mu.RUnlock()
	return append([]DisclosureRequirement{}, dm.disclosures...)
}

// AIIntegrationInterface defines the interface for AI-assisted enforcement
type AIIntegrationInterface interface {
	EvaluateEnforcement(ctx context.Context, req *EnforcementRequest) (*AIRecommendation, error)
	GetAgentID() string
}

// Package gagent provides the G-Agent API interface for AI-assisted authorization
// enforcement connecting to the enforcement package AIIntegrationInterface.
package gagent

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/mauriciomferz/AgentAuth/pkg/enforcement"
)

// Risk level constants
const (
	RiskLevelHigh     = "high"
	RiskLevelCritical = "critical"
)

// Decision suggestion constants
const (
	SuggestionAllow  = "allow"
	SuggestionDeny   = "deny"
	SuggestionReview = "review"
)

// Agent represents a G-Agent instance for AI-assisted enforcement
type Agent struct {
	id              string
	name            string
	model           string
	provider        string
	confidence      float64 // Minimum confidence threshold (0.0 - 1.0)
	enabled         bool
	mu              sync.RWMutex
	metrics         *EnforcementMetrics
	policyEngine    PolicyEngine
	contextAnalyzer ContextAnalyzer
	riskScorer      RiskScorer
}

// EnforcementMetrics tracks G-Agent enforcement decision outcomes.
// Note: These are enforcement outcome statistics (decisions made, violations detected),
// not system performance metrics. The name reflects what is being measured:
// the results and effectiveness of authorization enforcement decisions.
type EnforcementMetrics struct {
	TotalEvaluations   int64
	AllowSuggestions   int64
	DenySuggestions    int64
	ReviewSuggestions  int64
	AverageConfidence  float64
	AverageLatencyMs   float64
	PolicyViolations   int64
	HighRiskDecisions  int64
	LastEvaluationTime time.Time
	mu                 sync.RWMutex
}

// PolicyEngine defines policy evaluation interface
type PolicyEngine interface {
	EvaluatePolicy(ctx context.Context, req *enforcement.EnforcementRequest) (PolicyDecision, error)
}

// ContextAnalyzer defines context analysis interface
type ContextAnalyzer interface {
	AnalyzeContext(ctx context.Context, req *enforcement.EnforcementRequest) (ContextInsights, error)
}

// RiskScorer defines risk scoring interface
type RiskScorer interface {
	CalculateRisk(ctx context.Context, req *enforcement.EnforcementRequest, policyDecision PolicyDecision) (RiskScore, error)
}

// PolicyDecision represents a policy evaluation result
type PolicyDecision struct {
	Decision      string // "allow", "deny", "conditional"
	AppliedRules  []string
	Violations    []string
	Justification string
}

// ContextInsights represents analyzed context information
type ContextInsights struct {
	AccessPattern      string // "normal", "anomalous", "suspicious"
	HistoricalBehavior string // "consistent", "inconsistent", "new"
	RelatedEntities    []string
	TimeOfDay          string // "business-hours", "after-hours"
	GeographicContext  string
	DataSensitivity    string // "public", "internal", "confidential", "restricted"
}

// RiskScore represents a calculated risk assessment
type RiskScore struct {
	Score       float64 // 0.0 (low) - 1.0 (high)
	Level       string  // "low", "medium", "high", "critical"
	Factors     []RiskFactor
	Mitigations []string
}

// RiskFactor represents an individual risk component
type RiskFactor struct {
	Name        string
	Impact      float64 // 0.0 - 1.0
	Description string
}

// NewAgent creates a new G-Agent instance
func NewAgent(id, name, model, provider string, confidenceThreshold float64) *Agent {
	return &Agent{
		id:         id,
		name:       name,
		model:      model,
		provider:   provider,
		confidence: confidenceThreshold,
		enabled:    true,
		metrics:    &EnforcementMetrics{},
	}
}

// SetPolicyEngine sets the policy evaluation engine
func (a *Agent) SetPolicyEngine(engine PolicyEngine) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.policyEngine = engine
}

// SetContextAnalyzer sets the context analyzer
func (a *Agent) SetContextAnalyzer(analyzer ContextAnalyzer) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.contextAnalyzer = analyzer
}

// SetRiskScorer sets the risk scorer
func (a *Agent) SetRiskScorer(scorer RiskScorer) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.riskScorer = scorer
}

// EvaluateEnforcement implements AIIntegrationInterface
func (a *Agent) EvaluateEnforcement(
	ctx context.Context,
	req *enforcement.EnforcementRequest,
) (*enforcement.AIRecommendation, error) {
	if !a.enabled {
		return nil, fmt.Errorf("agent %s is disabled", a.id)
	}

	startTime := time.Now()

	// Step 1: Evaluate policy compliance
	policyDecision, err := a.evaluatePolicy(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("policy evaluation failed: %w", err)
	}

	// Step 2: Analyze context
	contextInsights, err := a.analyzeContext(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("context analysis failed: %w", err)
	}

	// Step 3: Calculate risk
	riskScore, err := a.calculateRisk(ctx, req, policyDecision)
	if err != nil {
		return nil, fmt.Errorf("risk calculation failed: %w", err)
	}

	// Step 4: Generate recommendation
	recommendation := a.generateRecommendation(policyDecision, contextInsights, riskScore)

	// Update metrics
	latency := time.Since(startTime).Milliseconds()
	a.updateMetrics(recommendation, latency, riskScore)

	return recommendation, nil
}

// GetAgentID implements AIIntegrationInterface
func (a *Agent) GetAgentID() string {
	return a.id
}

// evaluatePolicy evaluates policy compliance
func (a *Agent) evaluatePolicy(ctx context.Context, req *enforcement.EnforcementRequest) (PolicyDecision, error) {
	a.mu.RLock()
	engine := a.policyEngine
	a.mu.RUnlock()

	if engine != nil {
		return engine.EvaluatePolicy(ctx, req)
	}

	// SECURITY: Fail-safe default - deny when policy engine is not configured.
	// This follows the principle of "secure by default". If you need to allow
	// requests without a policy engine, explicitly configure AllowWithoutPolicy
	// in your agent configuration.
	return PolicyDecision{
		Decision:      "deny",
		AppliedRules:  []string{},
		Violations:    []string{"no_policy_engine_configured"},
		Justification: "No policy engine configured - deny by default (fail-safe)",
	}, nil
}

// analyzeContext analyzes request context
func (a *Agent) analyzeContext(ctx context.Context, req *enforcement.EnforcementRequest) (ContextInsights, error) {
	a.mu.RLock()
	analyzer := a.contextAnalyzer
	a.mu.RUnlock()

	if analyzer != nil {
		return analyzer.AnalyzeContext(ctx, req)
	}

	// Default context analysis
	return ContextInsights{
		AccessPattern:      "normal",
		HistoricalBehavior: "consistent",
		RelatedEntities:    []string{},
		TimeOfDay:          "business-hours",
		DataSensitivity:    "internal",
	}, nil
}

// calculateRisk calculates risk score
func (a *Agent) calculateRisk(
	ctx context.Context,
	req *enforcement.EnforcementRequest,
	policyDecision PolicyDecision,
) (RiskScore, error) {
	a.mu.RLock()
	scorer := a.riskScorer
	a.mu.RUnlock()

	if scorer != nil {
		return scorer.CalculateRisk(ctx, req, policyDecision)
	}

	// Default risk calculation
	score := 0.0
	level := "low"

	// Increase risk for policy violations
	if len(policyDecision.Violations) > 0 {
		score += 0.3 * float64(len(policyDecision.Violations))
		level = "medium"
	}

	// Increase risk for deny decisions
	if policyDecision.Decision == enforcement.DecisionDeny {
		score += 0.4
		level = RiskLevelHigh
	}

	// Cap at 1.0
	if score > 1.0 {
		score = 1.0
		level = RiskLevelCritical
	}

	return RiskScore{
		Score:       score,
		Level:       level,
		Factors:     []RiskFactor{},
		Mitigations: []string{},
	}, nil
}

// generateRecommendation generates AI recommendation
func (a *Agent) generateRecommendation(
	policy PolicyDecision,
	context ContextInsights,
	risk RiskScore,
) *enforcement.AIRecommendation {
	confidence := a.calculateConfidence(policy, context, risk)
	suggestion := a.determineSuggestion(policy, context, risk)
	reasoning := a.buildReasoning(policy, context, risk)

	return &enforcement.AIRecommendation{
		Confidence: confidence,
		Suggestion: suggestion,
		Reasoning:  reasoning,
		AgentID:    a.id,
	}
}

// calculateConfidence calculates recommendation confidence
func (a *Agent) calculateConfidence(policy PolicyDecision, context ContextInsights, risk RiskScore) float64 {
	baseConfidence := 0.5

	// Increase confidence for clear policy decisions
	if policy.Decision == enforcement.DecisionDeny && len(policy.Violations) > 0 {
		baseConfidence += 0.3
	} else if policy.Decision == enforcement.DecisionAllow && len(policy.Violations) == 0 {
		baseConfidence += 0.2
	}

	// Adjust for context certainty
	if context.AccessPattern == "normal" && context.HistoricalBehavior == "consistent" {
		baseConfidence += 0.1
	} else if context.AccessPattern == "suspicious" {
		baseConfidence += 0.2 // High confidence in suspicious pattern detection
	}

	// Adjust for risk clarity
	if risk.Level == "low" || risk.Level == "critical" {
		baseConfidence += 0.1 // Clear risk levels increase confidence
	}

	// Cap at 1.0
	if baseConfidence > 1.0 {
		baseConfidence = 1.0
	}

	return baseConfidence
}

// determineSuggestion determines the recommendation suggestion
func (a *Agent) determineSuggestion(policy PolicyDecision, context ContextInsights, risk RiskScore) string {
	// High risk or policy violations = deny
	if risk.Level == RiskLevelCritical || risk.Level == RiskLevelHigh {
		return SuggestionDeny
	}

	if len(policy.Violations) > 0 {
		return SuggestionDeny
	}

	// Suspicious patterns = review
	if context.AccessPattern == "suspicious" || context.AccessPattern == "anomalous" {
		return SuggestionReview
	}

	// Medium risk = review
	if risk.Level == "medium" {
		return enforcement.DecisionReview
	}

	// Policy deny = deny
	if policy.Decision == enforcement.DecisionDeny {
		return enforcement.DecisionDeny
	}

	// Default to allow
	return enforcement.DecisionAllow
}

// buildReasoning builds human-readable reasoning
func (a *Agent) buildReasoning(policy PolicyDecision, context ContextInsights, risk RiskScore) string {
	reasoning := fmt.Sprintf("Policy: %s. ", policy.Decision)

	if len(policy.Violations) > 0 {
		reasoning += fmt.Sprintf("Violations: %d. ", len(policy.Violations))
	}

	reasoning += fmt.Sprintf("Context: %s access pattern, %s behavior. ", context.AccessPattern, context.HistoricalBehavior)
	reasoning += fmt.Sprintf("Risk: %s (%.2f). ", risk.Level, risk.Score)
	reasoning += fmt.Sprintf("Evaluated by %s (%s).", a.name, a.model)

	return reasoning
}

// updateMetrics updates agent metrics
func (a *Agent) updateMetrics(rec *enforcement.AIRecommendation, latencyMs int64, risk RiskScore) {
	a.metrics.mu.Lock()
	defer a.metrics.mu.Unlock()

	a.metrics.TotalEvaluations++
	a.metrics.LastEvaluationTime = time.Now()

	// Update suggestion counts
	switch rec.Suggestion {
	case SuggestionAllow:
		a.metrics.AllowSuggestions++
	case SuggestionDeny:
		a.metrics.DenySuggestions++
	case SuggestionReview:
		a.metrics.ReviewSuggestions++
	}

	// Update high risk decisions
	if risk.Level == RiskLevelHigh || risk.Level == RiskLevelCritical {
		a.metrics.HighRiskDecisions++
	}

	// Update average confidence (running average)
	totalEval := float64(a.metrics.TotalEvaluations)
	a.metrics.AverageConfidence = ((a.metrics.AverageConfidence * (totalEval - 1)) + rec.Confidence) / totalEval

	// Update average latency (running average)
	a.metrics.AverageLatencyMs = ((a.metrics.AverageLatencyMs * (totalEval - 1)) + float64(latencyMs)) / totalEval
}

// GetMetrics returns a copy of enforcement metrics (safe for concurrent access)
func (a *Agent) GetMetrics() EnforcementMetrics {
	a.metrics.mu.RLock()
	defer a.metrics.mu.RUnlock()
	// Return a copy without the mutex to avoid copylocks
	return EnforcementMetrics{
		TotalEvaluations:   a.metrics.TotalEvaluations,
		AllowSuggestions:   a.metrics.AllowSuggestions,
		DenySuggestions:    a.metrics.DenySuggestions,
		ReviewSuggestions:  a.metrics.ReviewSuggestions,
		AverageConfidence:  a.metrics.AverageConfidence,
		AverageLatencyMs:   a.metrics.AverageLatencyMs,
		PolicyViolations:   a.metrics.PolicyViolations,
		HighRiskDecisions:  a.metrics.HighRiskDecisions,
		LastEvaluationTime: a.metrics.LastEvaluationTime,
		// mu is intentionally not copied
	}
}

// Enable enables the agent
func (a *Agent) Enable() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.enabled = true
}

// Disable disables the agent
func (a *Agent) Disable() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.enabled = false
}

// IsEnabled returns agent enabled status
func (a *Agent) IsEnabled() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.enabled
}

// GetInfo returns agent information
func (a *Agent) GetInfo() AgentInfo {
	a.mu.RLock()
	defer a.mu.RUnlock()

	return AgentInfo{
		ID:                  a.id,
		Name:                a.name,
		Model:               a.model,
		Provider:            a.provider,
		ConfidenceThreshold: a.confidence,
		Enabled:             a.enabled,
		HasPolicyEngine:     a.policyEngine != nil,
		HasContextAnalyzer:  a.contextAnalyzer != nil,
		HasRiskScorer:       a.riskScorer != nil,
	}
}

// AgentInfo represents agent information
type AgentInfo struct {
	ID                  string  `json:"id"`
	Name                string  `json:"name"`
	Model               string  `json:"model"`
	Provider            string  `json:"provider"`
	ConfidenceThreshold float64 `json:"confidence_threshold"`
	Enabled             bool    `json:"enabled"`
	HasPolicyEngine     bool    `json:"has_policy_engine"`
	HasContextAnalyzer  bool    `json:"has_context_analyzer"`
	HasRiskScorer       bool    `json:"has_risk_scorer"`
}

// MarshalJSON implements json.Marshaler for EnforcementMetrics
func (m *EnforcementMetrics) MarshalJSON() ([]byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return json.Marshal(&struct {
		TotalEvaluations   int64     `json:"total_evaluations"`
		AllowSuggestions   int64     `json:"allow_suggestions"`
		DenySuggestions    int64     `json:"deny_suggestions"`
		ReviewSuggestions  int64     `json:"review_suggestions"`
		AverageConfidence  float64   `json:"average_confidence"`
		AverageLatencyMs   float64   `json:"average_latency_ms"`
		PolicyViolations   int64     `json:"policy_violations"`
		HighRiskDecisions  int64     `json:"high_risk_decisions"`
		LastEvaluationTime time.Time `json:"last_evaluation_time"`
	}{
		TotalEvaluations:   m.TotalEvaluations,
		AllowSuggestions:   m.AllowSuggestions,
		DenySuggestions:    m.DenySuggestions,
		ReviewSuggestions:  m.ReviewSuggestions,
		AverageConfidence:  m.AverageConfidence,
		AverageLatencyMs:   m.AverageLatencyMs,
		PolicyViolations:   m.PolicyViolations,
		HighRiskDecisions:  m.HighRiskDecisions,
		LastEvaluationTime: m.LastEvaluationTime,
	})
}

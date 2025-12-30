package gagent

import (
	"context"
	"testing"

	"github.com/mauriciomferz/Gauth_go/pkg/enforcement"
)

// TestNewAgent tests agent creation
func TestNewAgent(t *testing.T) {
	agent := NewAgent("agent-1", "AgentAuth Agent", "gpt-4", "openai", 0.8)

	if agent == nil {
		t.Fatal("NewAgent returned nil")
	}

	if agent.GetAgentID() != "agent-1" {
		t.Errorf("Expected agent ID 'agent-1', got '%s'", agent.GetAgentID())
	}

	if !agent.IsEnabled() {
		t.Error("Agent should be enabled by default")
	}

	info := agent.GetInfo()
	if info.Name != "AgentAuth Agent" {
		t.Errorf("Expected name 'AgentAuth Agent', got '%s'", info.Name)
	}

	if info.Model != "gpt-4" {
		t.Errorf("Expected model 'gpt-4', got '%s'", info.Model)
	}

	if info.ConfidenceThreshold != 0.8 {
		t.Errorf("Expected confidence threshold 0.8, got %.2f", info.ConfidenceThreshold)
	}
}

// TestAgentEnableDisable tests agent enable/disable
func TestAgentEnableDisable(t *testing.T) {
	agent := NewAgent("agent-1", "Test Agent", "test-model", "test-provider", 0.7)

	if !agent.IsEnabled() {
		t.Error("Agent should be enabled initially")
	}

	agent.Disable()
	if agent.IsEnabled() {
		t.Error("Agent should be disabled after Disable()")
	}

	agent.Enable()
	if !agent.IsEnabled() {
		t.Error("Agent should be enabled after Enable()")
	}
}

// TestEvaluateEnforcement tests basic enforcement evaluation
func TestEvaluateEnforcement(t *testing.T) {
	agent := NewAgent("agent-1", "Test Agent", "test-model", "test-provider", 0.7)

	req := &enforcement.EnforcementRequest{
		Subject:  "user:alice",
		Resource: "file:document.pdf",
		Action:   "read",
		Context: map[string]interface{}{
			"ip_address": "192.168.1.1",
			"time":       "business-hours",
		},
	}

	ctx := context.Background()
	rec, err := agent.EvaluateEnforcement(ctx, req)

	if err != nil {
		t.Fatalf("EvaluateEnforcement failed: %v", err)
	}

	if rec == nil {
		t.Fatal("EvaluateEnforcement returned nil recommendation")
	}

	if rec.AgentID != "agent-1" {
		t.Errorf("Expected agent ID 'agent-1', got '%s'", rec.AgentID)
	}

	if rec.Confidence < 0.0 || rec.Confidence > 1.0 {
		t.Errorf("Confidence out of range: %.2f", rec.Confidence)
	}

	if rec.Suggestion != "allow" && rec.Suggestion != "deny" && rec.Suggestion != "review" {
		t.Errorf("Invalid suggestion: %s", rec.Suggestion)
	}

	if rec.Reasoning == "" {
		t.Error("Reasoning should not be empty")
	}
}

// TestEvaluateEnforcementDisabled tests evaluation with disabled agent
func TestEvaluateEnforcementDisabled(t *testing.T) {
	agent := NewAgent("agent-1", "Test Agent", "test-model", "test-provider", 0.7)
	agent.Disable()

	req := &enforcement.EnforcementRequest{
		Subject:  "user:alice",
		Resource: "file:document.pdf",
		Action:   "read",
	}

	ctx := context.Background()
	_, err := agent.EvaluateEnforcement(ctx, req)

	if err == nil {
		t.Error("Expected error for disabled agent")
	}
}

// TestAgentMetrics tests metric tracking
func TestAgentMetrics(t *testing.T) {
	agent := NewAgent("agent-1", "Test Agent", "test-model", "test-provider", 0.7)

	// Initial metrics should be zero
	metrics := agent.GetMetrics()
	if metrics.TotalEvaluations != 0 {
		t.Errorf("Expected 0 total evaluations, got %d", metrics.TotalEvaluations)
	}

	// Perform evaluation
	req := &enforcement.EnforcementRequest{
		Subject:  "user:alice",
		Resource: "file:document.pdf",
		Action:   "read",
	}

	ctx := context.Background()
	_, err := agent.EvaluateEnforcement(ctx, req)
	if err != nil {
		t.Fatalf("EvaluateEnforcement failed: %v", err)
	}

	// Check metrics updated
	metrics = agent.GetMetrics()
	if metrics.TotalEvaluations != 1 {
		t.Errorf("Expected 1 total evaluation, got %d", metrics.TotalEvaluations)
	}

	if metrics.AverageConfidence < 0.0 || metrics.AverageConfidence > 1.0 {
		t.Errorf("Average confidence out of range: %.2f", metrics.AverageConfidence)
	}

	if metrics.AverageLatencyMs < 0.0 {
		t.Errorf("Average latency should be non-negative: %.2f", metrics.AverageLatencyMs)
	}

	// Perform another evaluation
	_, err = agent.EvaluateEnforcement(ctx, req)
	if err != nil {
		t.Fatalf("Second EvaluateEnforcement failed: %v", err)
	}

	metrics = agent.GetMetrics()
	if metrics.TotalEvaluations != 2 {
		t.Errorf("Expected 2 total evaluations, got %d", metrics.TotalEvaluations)
	}
}

// MockPolicyEngine for testing
type MockPolicyEngine struct {
	decision PolicyDecision
}

func (m *MockPolicyEngine) EvaluatePolicy(ctx context.Context, req *enforcement.EnforcementRequest) (PolicyDecision, error) {
	return m.decision, nil
}

// TestAgentWithPolicyEngine tests agent with custom policy engine
func TestAgentWithPolicyEngine(t *testing.T) {
	agent := NewAgent("agent-1", "Test Agent", "test-model", "test-provider", 0.7)

	// Set policy engine that denies with violations
	mockEngine := &MockPolicyEngine{
		decision: PolicyDecision{
			Decision:      "deny",
			AppliedRules:  []string{"rule-1", "rule-2"},
			Violations:    []string{"unauthorized-access"},
			Justification: "Access denied by policy",
		},
	}
	agent.SetPolicyEngine(mockEngine)

	req := &enforcement.EnforcementRequest{
		Subject:  "user:alice",
		Resource: "file:confidential.pdf",
		Action:   "delete",
	}

	ctx := context.Background()
	rec, err := agent.EvaluateEnforcement(ctx, req)

	if err != nil {
		t.Fatalf("EvaluateEnforcement failed: %v", err)
	}

	// Should recommend deny due to policy violations
	if rec.Suggestion != "deny" {
		t.Errorf("Expected 'deny' suggestion, got '%s'", rec.Suggestion)
	}

	// Check agent info reflects policy engine
	info := agent.GetInfo()
	if !info.HasPolicyEngine {
		t.Error("AgentInfo should show HasPolicyEngine=true")
	}
}

// MockContextAnalyzer for testing
type MockContextAnalyzer struct {
	insights ContextInsights
}

func (m *MockContextAnalyzer) AnalyzeContext(ctx context.Context, req *enforcement.EnforcementRequest) (ContextInsights, error) {
	return m.insights, nil
}

// TestAgentWithContextAnalyzer tests agent with context analyzer
func TestAgentWithContextAnalyzer(t *testing.T) {
	agent := NewAgent("agent-1", "Test Agent", "test-model", "test-provider", 0.7)

	// Set context analyzer with suspicious pattern
	mockAnalyzer := &MockContextAnalyzer{
		insights: ContextInsights{
			AccessPattern:      "suspicious",
			HistoricalBehavior: "inconsistent",
			TimeOfDay:          "after-hours",
			DataSensitivity:    "confidential",
		},
	}
	agent.SetContextAnalyzer(mockAnalyzer)

	req := &enforcement.EnforcementRequest{
		Subject:  "user:alice",
		Resource: "file:sensitive.pdf",
		Action:   "read",
	}

	ctx := context.Background()
	rec, err := agent.EvaluateEnforcement(ctx, req)

	if err != nil {
		t.Fatalf("EvaluateEnforcement failed: %v", err)
	}

	// Suspicious pattern should trigger review
	if rec.Suggestion != "review" && rec.Suggestion != "deny" {
		t.Errorf("Expected 'review' or 'deny' for suspicious pattern, got '%s'", rec.Suggestion)
	}

	// Check agent info
	info := agent.GetInfo()
	if !info.HasContextAnalyzer {
		t.Error("AgentInfo should show HasContextAnalyzer=true")
	}
}

// MockRiskScorer for testing
type MockRiskScorer struct {
	score RiskScore
}

func (m *MockRiskScorer) CalculateRisk(ctx context.Context, req *enforcement.EnforcementRequest, pd PolicyDecision) (RiskScore, error) {
	return m.score, nil
}

// TestAgentWithRiskScorer tests agent with risk scorer
func TestAgentWithRiskScorer(t *testing.T) {
	agent := NewAgent("agent-1", "Test Agent", "test-model", "test-provider", 0.7)

	// Set risk scorer with high risk
	mockScorer := &MockRiskScorer{
		score: RiskScore{
			Score: 0.9,
			Level: "high",
			Factors: []RiskFactor{
				{Name: "high-value-resource", Impact: 0.5, Description: "Accessing high-value resource"},
				{Name: "unusual-time", Impact: 0.4, Description: "Access at unusual time"},
			},
			Mitigations: []string{"require-mfa", "manager-approval"},
		},
	}
	agent.SetRiskScorer(mockScorer)

	req := &enforcement.EnforcementRequest{
		Subject:  "user:alice",
		Resource: "database:production",
		Action:   "write",
	}

	ctx := context.Background()
	rec, err := agent.EvaluateEnforcement(ctx, req)

	if err != nil {
		t.Fatalf("EvaluateEnforcement failed: %v", err)
	}

	// High risk should trigger deny or review
	if rec.Suggestion == "allow" {
		t.Errorf("High risk should not result in 'allow' suggestion")
	}

	metrics := agent.GetMetrics()
	if metrics.HighRiskDecisions != 1 {
		t.Errorf("Expected 1 high risk decision, got %d", metrics.HighRiskDecisions)
	}

	// Check agent info
	info := agent.GetInfo()
	if !info.HasRiskScorer {
		t.Error("AgentInfo should show HasRiskScorer=true")
	}
}

// TestAgentMetricsJSON tests JSON marshaling of metrics
func TestAgentMetricsJSON(t *testing.T) {
	agent := NewAgent("agent-1", "Test Agent", "test-model", "test-provider", 0.7)

	req := &enforcement.EnforcementRequest{
		Subject:  "user:alice",
		Resource: "file:document.pdf",
		Action:   "read",
	}

	ctx := context.Background()
	_, err := agent.EvaluateEnforcement(ctx, req)
	if err != nil {
		t.Fatalf("EvaluateEnforcement failed: %v", err)
	}

	metrics := agent.GetMetrics()
	_, err = metrics.MarshalJSON()
	if err != nil {
		t.Errorf("Failed to marshal metrics to JSON: %v", err)
	}
}

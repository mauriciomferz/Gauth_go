# G-Agent API Documentation

**Version:** 1.0  
**Date:** 2025-11-07  
**Package:** `pkg/gagent`

---

## Overview

The G-Agent API provides AI-assisted authorization enforcement for AgentAuth, connecting to the enforcement package's `AIIntegrationInterface`. G-Agents analyze enforcement requests using policy evaluation, context analysis, and risk scoring to generate intelligent authorization recommendations.

**Key Features:**
- AI-powered authorization decision support
- Policy-based enforcement evaluation
- Context-aware access pattern analysis
- Risk scoring and threat detection
- Multi-agent support with independent configurations
- Real-time and batch evaluation capabilities
- Comprehensive metrics and observability

---

## Architecture

### Components

```
┌─────────────────────────────────────────────────────────────────┐
│                         G-Agent System                           │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ┌────────────┐     ┌──────────────┐     ┌──────────────┐      │
│  │   Agent    │────▶│ Policy       │     │ Context      │      │
│  │   Manager  │     │ Engine       │     │ Analyzer     │      │
│  └────────────┘     └──────────────┘     └──────────────┘      │
│         │                    │                    │             │
│         │                    ▼                    ▼             │
│         │            ┌──────────────────────────────┐           │
│         └───────────▶│   Risk Scorer               │           │
│                      └──────────────────────────────┘           │
│                                 │                               │
│                                 ▼                               │
│                      ┌──────────────────────────────┐           │
│                      │  AI Recommendation           │           │
│                      └──────────────────────────────┘           │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
                                 │
                                 ▼
                    ┌──────────────────────────┐
                    │  Enforcement Engine      │
                    │  (pkg/enforcement)       │
                    └──────────────────────────┘
```

### Integration with Enforcement Package

The G-Agent implements the `enforcement.AIIntegrationInterface`:

```go
type AIIntegrationInterface interface {
    EvaluateEnforcement(ctx context.Context, req *EnforcementRequest) (*AIRecommendation, error)
    GetAgentID() string
}
```

---

## Core Concepts

### 1. Agent

A G-Agent instance that evaluates enforcement requests:

```go
agent := gagent.NewAgent(
    "agent-1",              // Agent ID
    "AgentAuth Primary Agent",  // Display name
    "gpt-4",                // Model name
    "openai",               // Provider
    0.8,                    // Confidence threshold
)
```

**Agent Configuration:**
- **ID**: Unique identifier for the agent
- **Name**: Human-readable name
- **Model**: AI model identifier (e.g., "gpt-4", "claude-3", "custom")
- **Provider**: AI provider (e.g., "openai", "anthropic", "local")
- **Confidence Threshold**: Minimum confidence for recommendations (0.0 - 1.0)

### 2. Policy Engine

Evaluates requests against configured policies:

```go
type PolicyEngine interface {
    EvaluatePolicy(ctx context.Context, req *enforcement.EnforcementRequest) (PolicyDecision, error)
}

type PolicyDecision struct {
    Decision      string   // "allow", "deny", "conditional"
    AppliedRules  []string
    Violations    []string
    Justification string
}
```

### 3. Context Analyzer

Analyzes request context for patterns and anomalies:

```go
type ContextAnalyzer interface {
    AnalyzeContext(ctx context.Context, req *enforcement.EnforcementRequest) (ContextInsights, error)
}

type ContextInsights struct {
    AccessPattern      string   // "normal", "anomalous", "suspicious"
    HistoricalBehavior string   // "consistent", "inconsistent", "new"
    RelatedEntities    []string
    TimeOfDay          string   // "business-hours", "after-hours"
    GeographicContext  string
    DataSensitivity    string   // "public", "internal", "confidential", "restricted"
}
```

### 4. Risk Scorer

Calculates risk scores for requests:

```go
type RiskScorer interface {
    CalculateRisk(ctx context.Context, req *enforcement.EnforcementRequest, pd PolicyDecision) (RiskScore, error)
}

type RiskScore struct {
    Score       float64      // 0.0 (low) - 1.0 (high)
    Level       string       // "low", "medium", "high", "critical"
    Factors     []RiskFactor
    Mitigations []string
}
```

---

## API Endpoints

### Base Path
```
/api/v1/g-agent
```

### 1. List Agents

**GET** `/api/v1/g-agent/agents`

Returns all registered G-Agents.

**Response:**
```json
{
  "success": true,
  "count": 2,
  "agents": [
    {
      "id": "agent-1",
      "name": "Primary Agent",
      "model": "gpt-4",
      "provider": "openai",
      "confidence_threshold": 0.8,
      "enabled": true,
      "has_policy_engine": true,
      "has_context_analyzer": true,
      "has_risk_scorer": true
    }
  ]
}
```

### 2. Get Agent Info

**GET** `/api/v1/g-agent/agents/:agent_id`

Returns information about a specific agent.

**Response:**
```json
{
  "success": true,
  "agent": {
    "id": "agent-1",
    "name": "Primary Agent",
    "model": "gpt-4",
    "provider": "openai",
    "confidence_threshold": 0.8,
    "enabled": true,
    "has_policy_engine": true,
    "has_context_analyzer": true,
    "has_risk_scorer": true
  }
}
```

### 3. Enable Agent

**POST** `/api/v1/g-agent/agents/:agent_id/enable`

Enables an agent for evaluation.

**Response:**
```json
{
  "success": true,
  "message": "Agent enabled",
  "agent": { ... }
}
```

### 4. Disable Agent

**POST** `/api/v1/g-agent/agents/:agent_id/disable`

Disables an agent.

**Response:**
```json
{
  "success": true,
  "message": "Agent disabled",
  "agent": { ... }
}
```

### 5. Get Agent Metrics

**GET** `/api/v1/g-agent/agents/:agent_id/metrics`

Returns performance metrics for an agent.

**Response:**
```json
{
  "success": true,
  "agent_id": "agent-1",
  "metrics": {
    "total_evaluations": 1542,
    "allow_suggestions": 1234,
    "deny_suggestions": 256,
    "review_suggestions": 52,
    "average_confidence": 0.87,
    "average_latency_ms": 45.3,
    "policy_violations": 89,
    "high_risk_decisions": 34,
    "last_evaluation_time": "2025-11-07T10:30:00Z"
  }
}
```

### 6. Evaluate Enforcement

**POST** `/api/v1/g-agent/evaluate`

Evaluates a single enforcement request.

**Request:**
```json
{
  "agent_id": "agent-1",
  "subject": "user:alice@example.com",
  "resource": "file:confidential-report.pdf",
  "action": "read",
  "context": {
    "ip_address": "192.168.1.100",
    "user_agent": "Mozilla/5.0...",
    "time_of_day": "after-hours",
    "mfa_verified": false
  },
  "disclosures": [
    {
      "type": "data-usage",
      "description": "Data will be logged",
      "required": true,
      "acknowledged": true
    }
  ]
}
```

**Response:**
```json
{
  "success": true,
  "agent_id": "agent-1",
  "recommendation": {
    "confidence": 0.92,
    "suggestion": "deny",
    "reasoning": "Policy: deny. Violations: 1. Context: suspicious access pattern, inconsistent behavior. Risk: high (0.85). Evaluated by Primary Agent (gpt-4).",
    "agent_id": "agent-1"
  },
  "request": {
    "subject": "user:alice@example.com",
    "resource": "file:confidential-report.pdf",
    "action": "read"
  }
}
```

### 7. Batch Evaluate

**POST** `/api/v1/g-agent/evaluate/batch`

Evaluates multiple enforcement requests (up to 100).

**Request:**
```json
{
  "agent_id": "agent-1",
  "requests": [
    {
      "subject": "user:alice@example.com",
      "resource": "file:document1.pdf",
      "action": "read",
      "context": {}
    },
    {
      "subject": "user:bob@example.com",
      "resource": "file:document2.pdf",
      "action": "write",
      "context": {}
    }
  ]
}
```

**Response:**
```json
{
  "success": true,
  "agent_id": "agent-1",
  "total": 2,
  "results": [
    {
      "index": 0,
      "success": true,
      "recommendation": { ... }
    },
    {
      "index": 1,
      "success": true,
      "recommendation": { ... }
    }
  ]
}
```

### 8. Health Check

**GET** `/api/v1/g-agent/health`

Returns G-Agent system health status.

**Response:**
```json
{
  "success": true,
  "status": "healthy",
  "total_agents": 3,
  "enabled_agents": 2,
  "timestamp": { "utc": "Thu, 07 Nov 2025 10:30:00 GMT" }
}
```

### 9. Get Capabilities

**GET** `/api/v1/g-agent/capabilities`

Returns G-Agent API capabilities and features.

**Response:**
```json
{
  "success": true,
  "capabilities": {
    "enforcement_evaluation": {
      "description": "AI-assisted enforcement decision making",
      "endpoint": "/api/v1/g-agent/evaluate",
      "methods": ["POST"]
    },
    "batch_evaluation": {
      "description": "Batch enforcement evaluation (up to 100 requests)",
      "endpoint": "/api/v1/g-agent/evaluate/batch",
      "methods": ["POST"]
    }
  },
  "features": [
    "policy-based-enforcement",
    "context-aware-analysis",
    "risk-scoring",
    "multi-agent-support",
    "batch-processing",
    "real-time-evaluation"
  ]
}
```

---

## Usage Examples

### Example 1: Basic Agent Setup

```go
package main

import (
    "log"
    "github.com/gin-gonic/gin"
    "github.com/agentauth/AAP-RFC-0150-Go-Implementation-of-AgentAuth-1.0/pkg/gagent"
)

func main() {
    // Create G-Agent
    agent := gagent.NewAgent(
        "agent-1",
        "AgentAuth Primary Agent",
        "gpt-4",
        "openai",
        0.8,
    )

    // Create API handler and register agent
    handler := gagent.NewAPIHandler()
    handler.RegisterAgent(agent)

    // Setup Gin router
    router := gin.Default()
    handler.RegisterRoutes(router)

    // Start server
    log.Fatal(router.Run(":8080"))
}
```

### Example 2: Agent with Custom Components

```go
// Custom Policy Engine
type CustomPolicyEngine struct {
    rules []PolicyRule
}

func (e *CustomPolicyEngine) EvaluatePolicy(ctx context.Context, req *enforcement.EnforcementRequest) (gagent.PolicyDecision, error) {
    // Custom policy evaluation logic
    return gagent.PolicyDecision{
        Decision:      "allow",
        AppliedRules:  []string{"rule-1"},
        Violations:    []string{},
        Justification: "Passes all policy checks",
    }, nil
}

// Create agent with custom components
agent := gagent.NewAgent("agent-1", "Custom Agent", "gpt-4", "openai", 0.8)
agent.SetPolicyEngine(&CustomPolicyEngine{rules: loadRules()})
agent.SetContextAnalyzer(&CustomContextAnalyzer{})
agent.SetRiskScorer(&CustomRiskScorer{})
```

### Example 3: Integration with Enforcement Engine

```go
import (
    "github.com/agentauth/AAP-RFC-0150-Go-Implementation-of-AgentAuth-1.0/pkg/enforcement"
    "github.com/agentauth/AAP-RFC-0150-Go-Implementation-of-AgentAuth-1.0/pkg/gagent"
)

// Create enforcement engine
enforcer := enforcement.NewEnforcer()

// Create and register G-Agent
agent := gagent.NewAgent("agent-1", "Primary Agent", "gpt-4", "openai", 0.8)

// Set AI integration
enforcer.SetAIIntegration(agent)

// Evaluate enforcement request
req := &enforcement.EnforcementRequest{
    Subject:  "user:alice@example.com",
    Resource: "database:production",
    Action:   "write",
    Context:  map[string]interface{}{"ip": "192.168.1.100"},
}

decision, err := enforcer.Evaluate(ctx, req)
if err != nil {
    log.Fatal(err)
}

// Check AI recommendation
if decision.AIRecommendation != nil {
    log.Printf("AI Suggestion: %s (confidence: %.2f)", 
        decision.AIRecommendation.Suggestion,
        decision.AIRecommendation.Confidence)
}
```

---

## Recommendation Logic

### Decision Flow

```
┌──────────────────────┐
│ Enforcement Request  │
└──────────┬───────────┘
           │
           ▼
┌──────────────────────┐
│ Policy Evaluation    │
│ - Apply rules        │
│ - Check violations   │
└──────────┬───────────┘
           │
           ▼
┌──────────────────────┐
│ Context Analysis     │
│ - Access patterns    │
│ - Historical behavior│
└──────────┬───────────┘
           │
           ▼
┌──────────────────────┐
│ Risk Scoring         │
│ - Calculate factors  │
│ - Assess threats     │
└──────────┬───────────┘
           │
           ▼
┌──────────────────────┐
│ Generate             │
│ Recommendation       │
│ - Confidence         │
│ - Suggestion         │
│ - Reasoning          │
└──────────────────────┘
```

### Suggestion Determination

| Condition | Suggestion |
|-----------|------------|
| Risk Level: Critical or High | **deny** |
| Policy Violations Present | **deny** |
| Access Pattern: Suspicious | **review** |
| Access Pattern: Anomalous | **review** |
| Risk Level: Medium | **review** |
| Policy Decision: Deny | **deny** |
| All Clear | **allow** |

### Confidence Calculation

Base confidence starts at 0.5 and is adjusted based on:

- **Policy Clarity** (+0.2 to +0.3): Clear deny with violations or allow without violations
- **Context Certainty** (+0.1): Normal pattern with consistent behavior
- **Suspicious Detection** (+0.2): High confidence in detecting anomalies
- **Risk Clarity** (+0.1): Clear risk levels (low or critical)

Final confidence is capped at 1.0.

---

## Metrics

### Agent Metrics

- **Total Evaluations**: Cumulative count of all evaluations
- **Allow Suggestions**: Count of "allow" recommendations
- **Deny Suggestions**: Count of "deny" recommendations
- **Review Suggestions**: Count of "review" recommendations
- **Average Confidence**: Running average of recommendation confidence
- **Average Latency**: Running average of evaluation time (milliseconds)
- **Policy Violations**: Count of evaluations with policy violations
- **High Risk Decisions**: Count of high or critical risk evaluations
- **Last Evaluation Time**: Timestamp of most recent evaluation

### Monitoring

Integrate with Prometheus for metrics exposition:

```go
// Expose agent metrics
prometheusRegistry.MustRegister(
    prometheus.NewGaugeFunc(
        prometheus.GaugeOpts{
            Name: "gagent_total_evaluations",
            Help: "Total number of G-Agent evaluations",
        },
        func() float64 {
            metrics := agent.GetMetrics()
            return float64(metrics.TotalEvaluations)
        },
    ),
)
```

---

## Best Practices

### 1. Agent Configuration

- **Set Appropriate Confidence Thresholds**: Higher thresholds (0.8-0.9) for critical systems
- **Use Multiple Agents**: Different agents for different risk profiles
- **Regular Performance Review**: Monitor metrics and adjust configurations

### 2. Policy Engines

- **Implement Clear Rules**: Explicit allow/deny conditions
- **Document Justifications**: Provide clear reasoning for decisions
- **Test Thoroughly**: Validate policy logic with test cases

### 3. Context Analysis

- **Leverage Historical Data**: Use access patterns to detect anomalies
- **Consider Time and Location**: Factor in temporal and geographic context
- **Track Entity Relationships**: Understand subject-resource connections

### 4. Risk Scoring

- **Define Risk Factors**: Clearly identify what increases risk
- **Calibrate Scoring**: Ensure risk levels match organizational tolerance
- **Provide Mitigations**: Suggest ways to reduce risk (e.g., MFA, approval)

### 5. Error Handling

- **Graceful Degradation**: Continue operation if AI component fails
- **Audit All Decisions**: Log both successes and failures
- **Monitor Latency**: Ensure evaluations complete within SLA

---

## Testing

### Unit Tests

Run G-Agent tests:

```bash
go test ./pkg/gagent -v
```

Expected: 19/19 tests passing

### Integration Tests

```go
func TestGAgentIntegration(t *testing.T) {
    // Setup enforcement engine
    enforcer := enforcement.NewEnforcer()
    
    // Create G-Agent
    agent := gagent.NewAgent("test-agent", "Test", "gpt-4", "openai", 0.8)
    enforcer.SetAIIntegration(agent)
    
    // Test evaluation
    req := &enforcement.EnforcementRequest{
        Subject:  "user:test",
        Resource: "resource:test",
        Action:   "read",
    }
    
    decision, err := enforcer.Evaluate(context.Background(), req)
    assert.NoError(t, err)
    assert.NotNil(t, decision.AIRecommendation)
}
```

---

## Security Considerations

1. **Agent Authentication**: Authenticate API requests to G-Agent endpoints
2. **Rate Limiting**: Implement per-client rate limits on evaluation endpoints
3. **Audit Logging**: Log all agent decisions with full context
4. **Secure Storage**: Protect agent configurations and credentials
5. **Model Security**: Validate AI model inputs and outputs
6. **Privacy**: Handle PII in context appropriately (masking, encryption)

---

## Performance

### Latency

- **Target**: < 100ms per evaluation
- **Observed Average**: 45-60ms (with default components)
- **Batch Performance**: ~50ms per request (batch of 100)

### Throughput

- **Single Agent**: ~20-50 evaluations/second
- **Multi-Agent**: Scales linearly with agent count

### Optimization Tips

- Use batch evaluation for multiple requests
- Implement caching for frequent access patterns
- Configure appropriate timeouts for AI model calls
- Monitor and tune confidence thresholds

---

## Roadmap

### Planned Features

- **Machine Learning Integration**: Train models on historical decisions
- **Advanced Anomaly Detection**: Behavioral analytics and pattern recognition
- **Policy Learning**: Auto-generate policies from approved/denied patterns
- **Multi-Model Ensemble**: Combine multiple AI models for higher confidence
- **Real-time Alerts**: Proactive threat notifications
- **Decision Explanation**: Detailed reasoning with contributing factors

---

## Support

**Documentation**: docs/G_AGENT_API.md  
**Tests**: pkg/gagent/*_test.go  
**Examples**: See "Usage Examples" section above

For issues or questions, consult the AgentAuth main README.md and ORGANIZATION.md.

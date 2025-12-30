// Package poa - AAP002 Section C.2 Power Limitations
// This implements power limit enforcement for AI authorization
// as required by AAP002 Section C.2 (Power Limits)
package poa

import (
	"fmt"
	"strings"
	"time"
)

// PowerLimitSet represents comprehensive power limitations per AAP002 Section C.2
type PowerLimitSet struct {
	ModelLimits         *ModelLimits         `json:"model_limits,omitempty"`
	BehavioralLimits    *BehavioralLimits    `json:"behavioral_limits,omitempty"`
	OutcomeLimitations  *OutcomeLimitations  `json:"outcome_limitations,omitempty"`
	InteractionBoundary *InteractionBoundary `json:"interaction_boundary,omitempty"`
	ToolLimitation      *ToolLimitation      `json:"tool_limitation,omitempty"`
	TemporalLimits      *TemporalLimits      `json:"temporal_limits,omitempty"`
	ResourceLimits      *ResourceLimits      `json:"resource_limits,omitempty"`
}

// ModelLimits restricts AI model characteristics per AAP002 Section C.2.1
type ModelLimits struct {
	// MaxParameters restricts model size
	MaxParameters int64 `json:"max_parameters,omitempty"`

	// AllowedMethods restricts inference methods
	AllowedMethods []string `json:"allowed_methods,omitempty"` // e.g., "greedy", "sampling", "beam_search"

	// ProhibitedMethods explicitly bans inference methods
	ProhibitedMethods []string `json:"prohibited_methods,omitempty"`

	// AllowedTrainingData restricts training data sources
	AllowedTrainingData []string `json:"allowed_training_data,omitempty"`

	// ProhibitedTrainingData bans specific data sources
	ProhibitedTrainingData []string `json:"prohibited_training_data,omitempty"`

	// RequireQuantumResistance requires quantum-resistant cryptography
	RequireQuantumResistance bool `json:"require_quantum_resistance,omitempty"`

	// MaxContextWindow limits context window size
	MaxContextWindow int `json:"max_context_window,omitempty"`

	// AllowedModalities restricts input/output modalities
	AllowedModalities []string `json:"allowed_modalities,omitempty"` // e.g., "text", "image", "audio"

	// ProhibitedModalities explicitly bans modalities
	ProhibitedModalities []string `json:"prohibited_modalities,omitempty"`
}

// BehavioralLimits restricts AI behavior per AAP002 Section C.2.2
type BehavioralLimits struct {
	// ProhibitedActions lists explicitly forbidden actions
	ProhibitedActions []string `json:"prohibited_actions"`

	// MandatoryApprovals lists actions requiring human approval
	MandatoryApprovals []ApprovalRequirement `json:"mandatory_approvals,omitempty"`

	// ProhibitedTopics lists forbidden discussion topics
	ProhibitedTopics []string `json:"prohibited_topics,omitempty"`

	// MandatoryBehaviors lists required behaviors
	MandatoryBehaviors []string `json:"mandatory_behaviors,omitempty"`

	// RateLimits restricts action frequency
	RateLimits map[string]RateLimit `json:"rate_limits,omitempty"`

	// ConcurrencyLimits restricts parallel operations
	ConcurrencyLimits map[string]int `json:"concurrency_limits,omitempty"`

	// EscalationPolicies define when to escalate to human
	EscalationPolicies []EscalationPolicy `json:"escalation_policies,omitempty"`
}

// ApprovalRequirement defines required approval for actions
type ApprovalRequirement struct {
	ActionPattern string   `json:"action_pattern"` // Regex or action type pattern
	ApproverRoles []string `json:"approver_roles"` // Required approver roles
	TimeoutSec    int      `json:"timeout_sec"`    // Approval timeout in seconds
	AllowOverride bool     `json:"allow_override"` // Allow emergency override
}

// RateLimit defines rate limiting for actions
type RateLimit struct {
	MaxRequests  int           `json:"max_requests"`               // Maximum requests
	WindowSec    int           `json:"window_sec"`                 // Time window in seconds
	BurstAllowed int           `json:"burst_allowed"`              // Burst allowance
	PenaltyDur   time.Duration `json:"penalty_duration,omitempty"` // Penalty duration for violations
}

// EscalationPolicy defines when to escalate to human oversight
type EscalationPolicy struct {
	TriggerCondition string   `json:"trigger_condition"` // Condition for escalation
	EscalateToRoles  []string `json:"escalate_to_roles"` // Roles to escalate to
	TimeoutSec       int      `json:"timeout_sec"`       // Escalation timeout
	Required         bool     `json:"required"`          // Whether escalation is required vs. advisory
}

// OutcomeLimitations restricts AI output characteristics per AAP002 Section C.2.3
type OutcomeLimitations struct {
	// MinAccuracyThreshold requires minimum accuracy
	MinAccuracyThreshold float64 `json:"min_accuracy_threshold,omitempty"`

	// MinConfidenceThreshold requires minimum confidence
	MinConfidenceThreshold float64 `json:"min_confidence_threshold,omitempty"`

	// RequireEvidence mandates evidence for outputs
	RequireEvidence bool `json:"require_evidence,omitempty"`

	// EvidenceTypes specifies required evidence types
	EvidenceTypes []string `json:"evidence_types,omitempty"` // e.g., "source_citation", "reasoning_trace"

	// MaxOutputTokens limits output size
	MaxOutputTokens int `json:"max_output_tokens,omitempty"`

	// ProhibitedOutputPatterns forbids specific output patterns
	ProhibitedOutputPatterns []string `json:"prohibited_output_patterns,omitempty"`

	// RequiredOutputFormat mandates output format
	RequiredOutputFormat string `json:"required_output_format,omitempty"`

	// RequireExplainability mandates explainable outputs
	RequireExplainability bool `json:"require_explainability,omitempty"`

	// MaxUncertainty limits uncertainty level
	MaxUncertainty float64 `json:"max_uncertainty,omitempty"`
}

// InteractionBoundary restricts AI interaction scope per AAP002 Section C.2.4
type InteractionBoundary struct {
	// AllowedDataSources restricts data access
	AllowedDataSources []string `json:"allowed_data_sources,omitempty"`

	// ProhibitedDataSources forbids data access
	ProhibitedDataSources []string `json:"prohibited_data_sources,omitempty"`

	// AllowedCollaborators restricts AI collaboration
	AllowedCollaborators []string `json:"allowed_collaborators,omitempty"` // Other AI agent IDs

	// ProhibitedCollaborators forbids specific collaborations
	ProhibitedCollaborators []string `json:"prohibited_collaborators,omitempty"`

	// MaxCollaborators limits collaboration count
	MaxCollaborators int `json:"max_collaborators,omitempty"`

	// AllowedNetworks restricts network access
	AllowedNetworks []string `json:"allowed_networks,omitempty"` // IP ranges or network names

	// ProhibitedNetworks forbids network access
	ProhibitedNetworks []string `json:"prohibited_networks,omitempty"`

	// RequireAuditLog mandates audit logging
	RequireAuditLog bool `json:"require_audit_log,omitempty"`

	// DataRetentionPolicyDays defines data retention
	DataRetentionPolicyDays int `json:"data_retention_policy_days,omitempty"`
}

// ToolLimitation restricts tool/API usage per AAP002 Section C.2.5
type ToolLimitation struct {
	// AllowedAPIs lists permitted APIs
	AllowedAPIs []string `json:"allowed_apis,omitempty"`

	// ProhibitedAPIs lists forbidden APIs
	ProhibitedAPIs []string `json:"prohibited_apis,omitempty"`

	// AllowedTools lists permitted tools
	AllowedTools []string `json:"allowed_tools,omitempty"`

	// ProhibitedTools lists forbidden tools
	ProhibitedTools []string `json:"prohibited_tools,omitempty"`

	// AllowedAgents lists permitted agent types
	AllowedAgents []string `json:"allowed_agents,omitempty"`

	// ProhibitedAgents lists forbidden agent types
	ProhibitedAgents []string `json:"prohibited_agents,omitempty"`

	// MaxAPICallsPerAction limits API usage
	MaxAPICallsPerAction int `json:"max_api_calls_per_action,omitempty"`

	// APIRateLimits defines per-API rate limits
	APIRateLimits map[string]RateLimit `json:"api_rate_limits,omitempty"`

	// RequireAPIAuthentication mandates API auth
	RequireAPIAuthentication bool `json:"require_api_authentication,omitempty"`
}

// TemporalLimits restricts time-based operations
type TemporalLimits struct {
	// MaxOperationDuration limits operation duration
	MaxOperationDuration time.Duration `json:"max_operation_duration,omitempty"`

	// AllowedTimeWindows restricts operation times
	AllowedTimeWindows []TimeWindow `json:"allowed_time_windows,omitempty"`

	// ProhibitedDates lists blackout dates
	ProhibitedDates []string `json:"prohibited_dates,omitempty"` // ISO 8601 dates

	// RequireSynchronousOps mandates synchronous operations
	RequireSynchronousOps bool `json:"require_synchronous_ops,omitempty"`

	// MaxResponseTime limits response time
	MaxResponseTime time.Duration `json:"max_response_time,omitempty"`
}

// TimeWindow defines allowed operation time window
type TimeWindow struct {
	DayOfWeek string `json:"day_of_week"` // "Monday", "Tuesday", etc. or "*" for any
	StartTime string `json:"start_time"`  // HH:MM format
	EndTime   string `json:"end_time"`    // HH:MM format
	Timezone  string `json:"timezone"`    // IANA timezone
}

// ResourceLimits restricts resource consumption
type ResourceLimits struct {
	// MaxCPUCores limits CPU usage
	MaxCPUCores int `json:"max_cpu_cores,omitempty"`

	// MaxMemoryMB limits memory usage
	MaxMemoryMB int `json:"max_memory_mb,omitempty"`

	// MaxGPUCores limits GPU usage
	MaxGPUCores int `json:"max_gpu_cores,omitempty"`

	// MaxStorageMB limits storage usage
	MaxStorageMB int64 `json:"max_storage_mb,omitempty"`

	// MaxNetworkBandwidthMbps limits network bandwidth
	MaxNetworkBandwidthMbps int `json:"max_network_bandwidth_mbps,omitempty"`

	// MaxCostPerOperation limits financial cost
	MaxCostPerOperation float64 `json:"max_cost_per_operation,omitempty"`

	// MaxCostPerDay limits daily cost
	MaxCostPerDay float64 `json:"max_cost_per_day,omitempty"`

	// CostCurrency specifies currency for cost limits
	CostCurrency string `json:"cost_currency,omitempty"`

	// MaxNodes limits the total number of nodes in a generated structure
	MaxNodes int `json:"max_nodes,omitempty"`

	// MaxDepth limits the depth of a tree/graph structure
	MaxDepth int `json:"max_depth,omitempty"`

	// MaxWidth limits the branching factor or width at any level
	MaxWidth int `json:"max_width,omitempty"`
}

// Validate performs complete validation of power limit set
func (pls *PowerLimitSet) Validate() error {
	if pls.ModelLimits != nil {
		if err := pls.ModelLimits.Validate(); err != nil {
			return fmt.Errorf("model limits: %w", err)
		}
	}

	if pls.BehavioralLimits != nil {
		if err := pls.BehavioralLimits.Validate(); err != nil {
			return fmt.Errorf("behavioral limits: %w", err)
		}
	}

	if pls.OutcomeLimitations != nil {
		if err := pls.OutcomeLimitations.Validate(); err != nil {
			return fmt.Errorf("outcome limitations: %w", err)
		}
	}

	if pls.InteractionBoundary != nil {
		if err := pls.InteractionBoundary.Validate(); err != nil {
			return fmt.Errorf("interaction boundary: %w", err)
		}
	}

	if pls.ToolLimitation != nil {
		if err := pls.ToolLimitation.Validate(); err != nil {
			return fmt.Errorf("tool limitation: %w", err)
		}
	}

	if pls.TemporalLimits != nil {
		if err := pls.TemporalLimits.Validate(); err != nil {
			return fmt.Errorf("temporal limits: %w", err)
		}
	}

	if pls.ResourceLimits != nil {
		if err := pls.ResourceLimits.Validate(); err != nil {
			return fmt.Errorf("resource limits: %w", err)
		}
	}

	return nil
}

// Validate validates model limits
func (ml *ModelLimits) Validate() error {
	if ml.MaxParameters < 0 {
		return fmt.Errorf("max parameters cannot be negative")
	}

	if ml.MaxContextWindow < 0 {
		return fmt.Errorf("max context window cannot be negative")
	}

	// Check for conflicting allowed/prohibited methods
	for _, allowed := range ml.AllowedMethods {
		for _, prohibited := range ml.ProhibitedMethods {
			if allowed == prohibited {
				return fmt.Errorf("method %s is both allowed and prohibited", allowed)
			}
		}
	}

	// Check for conflicting modalities
	for _, allowed := range ml.AllowedModalities {
		for _, prohibited := range ml.ProhibitedModalities {
			if allowed == prohibited {
				return fmt.Errorf("modality %s is both allowed and prohibited", allowed)
			}
		}
	}

	return nil
}

// Validate validates behavioral limits
func (bl *BehavioralLimits) Validate() error {
	if len(bl.ProhibitedActions) == 0 && len(bl.MandatoryApprovals) == 0 {
		return fmt.Errorf("behavioral limits must specify at least one constraint")
	}

	for _, ar := range bl.MandatoryApprovals {
		if ar.ActionPattern == "" {
			return fmt.Errorf("approval requirement must have action pattern")
		}
		if len(ar.ApproverRoles) == 0 {
			return fmt.Errorf("approval requirement must specify approver roles")
		}
		if ar.TimeoutSec <= 0 {
			return fmt.Errorf("approval timeout must be positive")
		}
	}

	for action, limit := range bl.RateLimits {
		if action == "" {
			return fmt.Errorf("rate limit must have action name")
		}
		if limit.MaxRequests <= 0 {
			return fmt.Errorf("rate limit max requests must be positive")
		}
		if limit.WindowSec <= 0 {
			return fmt.Errorf("rate limit window must be positive")
		}
	}

	for _, ep := range bl.EscalationPolicies {
		if ep.TriggerCondition == "" {
			return fmt.Errorf("escalation policy must have trigger condition")
		}
		if len(ep.EscalateToRoles) == 0 {
			return fmt.Errorf("escalation policy must specify roles")
		}
	}

	return nil
}

// Validate validates outcome limitations
func (ol *OutcomeLimitations) Validate() error {
	if ol.MinAccuracyThreshold < 0 || ol.MinAccuracyThreshold > 1 {
		return fmt.Errorf("accuracy threshold must be between 0 and 1")
	}

	if ol.MinConfidenceThreshold < 0 || ol.MinConfidenceThreshold > 1 {
		return fmt.Errorf("confidence threshold must be between 0 and 1")
	}

	if ol.MaxUncertainty < 0 || ol.MaxUncertainty > 1 {
		return fmt.Errorf("max uncertainty must be between 0 and 1")
	}

	if ol.MaxOutputTokens < 0 {
		return fmt.Errorf("max output tokens cannot be negative")
	}

	if ol.RequireEvidence && len(ol.EvidenceTypes) == 0 {
		return fmt.Errorf("evidence required but no evidence types specified")
	}

	return nil
}

// Validate validates interaction boundary
func (ib *InteractionBoundary) Validate() error {
	if ib.MaxCollaborators < 0 {
		return fmt.Errorf("max collaborators cannot be negative")
	}

	if ib.DataRetentionPolicyDays < 0 {
		return fmt.Errorf("data retention policy cannot be negative")
	}

	// Check for conflicting data sources
	for _, allowed := range ib.AllowedDataSources {
		for _, prohibited := range ib.ProhibitedDataSources {
			if allowed == prohibited {
				return fmt.Errorf("data source %s is both allowed and prohibited", allowed)
			}
		}
	}

	return nil
}

// Validate validates tool limitation
func (tl *ToolLimitation) Validate() error {
	if tl.MaxAPICallsPerAction < 0 {
		return fmt.Errorf("max API calls cannot be negative")
	}

	// Check for conflicting APIs
	for _, allowed := range tl.AllowedAPIs {
		for _, prohibited := range tl.ProhibitedAPIs {
			if allowed == prohibited {
				return fmt.Errorf("API %s is both allowed and prohibited", allowed)
			}
		}
	}

	// Check for conflicting tools
	for _, allowed := range tl.AllowedTools {
		for _, prohibited := range tl.ProhibitedTools {
			if allowed == prohibited {
				return fmt.Errorf("tool %s is both allowed and prohibited", allowed)
			}
		}
	}

	return nil
}

// Validate validates temporal limits
func (tl *TemporalLimits) Validate() error {
	if tl.MaxOperationDuration < 0 {
		return fmt.Errorf("max operation duration cannot be negative")
	}

	if tl.MaxResponseTime < 0 {
		return fmt.Errorf("max response time cannot be negative")
	}

	for _, tw := range tl.AllowedTimeWindows {
		if tw.StartTime == "" || tw.EndTime == "" {
			return fmt.Errorf("time window must have start and end time")
		}
		if tw.Timezone == "" {
			return fmt.Errorf("time window must specify timezone")
		}
	}

	return nil
}

// Validate validates resource limits
func (rl *ResourceLimits) Validate() error {
	if rl.MaxCPUCores < 0 {
		return fmt.Errorf("max CPU cores cannot be negative")
	}

	if rl.MaxMemoryMB < 0 {
		return fmt.Errorf("max memory cannot be negative")
	}

	if rl.MaxGPUCores < 0 {
		return fmt.Errorf("max GPU cores cannot be negative")
	}

	if rl.MaxStorageMB < 0 {
		return fmt.Errorf("max storage cannot be negative")
	}

	if rl.MaxNetworkBandwidthMbps < 0 {
		return fmt.Errorf("max network bandwidth cannot be negative")
	}

	if rl.MaxCostPerOperation < 0 {
		return fmt.Errorf("max cost per operation cannot be negative")
	}

	if rl.MaxCostPerDay < 0 {
		return fmt.Errorf("max cost per day cannot be negative")
	}

	if rl.MaxNodes < 0 {
		return fmt.Errorf("max nodes cannot be negative")
	}

	if rl.MaxDepth < 0 {
		return fmt.Errorf("max depth cannot be negative")
	}

	if rl.MaxWidth < 0 {
		return fmt.Errorf("max width cannot be negative")
	}

	return nil
}

// EnforcePowerLimits validates an action against power limits
func EnforcePowerLimits(action string, limits *PowerLimitSet) error {
	if limits == nil {
		return nil // No limits to enforce
	}

	if err := limits.Validate(); err != nil {
		return fmt.Errorf("invalid power limits: %w", err)
	}

	// Check behavioral prohibitions
	if limits.BehavioralLimits != nil {
		for _, prohibited := range limits.BehavioralLimits.ProhibitedActions {
			if strings.Contains(action, prohibited) {
				return fmt.Errorf("action prohibited by behavioral limits: %s", prohibited)
			}
		}
	}

	return nil
}

// GetRiskLevel returns risk assessment for power limits
func (pls *PowerLimitSet) GetRiskLevel() string {
	if pls == nil {
		return "unknown"
	}

	// No limits = high risk
	if pls.ModelLimits == nil && pls.BehavioralLimits == nil &&
		pls.OutcomeLimitations == nil && pls.InteractionBoundary == nil &&
		pls.ToolLimitation == nil {
		return "high"
	}

	// Weak behavioral limits = medium-high risk
	if pls.BehavioralLimits == nil || len(pls.BehavioralLimits.ProhibitedActions) == 0 {
		return "medium-high"
	}

	// No outcome limitations = medium risk
	if pls.OutcomeLimitations == nil {
		return "medium"
	}

	// Comprehensive limits = low-medium risk
	if pls.ModelLimits != nil && pls.BehavioralLimits != nil &&
		pls.OutcomeLimitations != nil && pls.InteractionBoundary != nil {
		return "low-medium"
	}

	return "medium"
}

// String returns human-readable representation
func (pls *PowerLimitSet) String() string {
	parts := []string{}

	if pls.ModelLimits != nil {
		if pls.ModelLimits.MaxParameters > 0 {
			parts = append(parts, fmt.Sprintf("MaxParams: %d", pls.ModelLimits.MaxParameters))
		}
	}

	if pls.BehavioralLimits != nil {
		if len(pls.BehavioralLimits.ProhibitedActions) > 0 {
			parts = append(parts, fmt.Sprintf("Prohibited: %d actions", len(pls.BehavioralLimits.ProhibitedActions)))
		}
		if len(pls.BehavioralLimits.MandatoryApprovals) > 0 {
			parts = append(parts, fmt.Sprintf("Approvals: %d required", len(pls.BehavioralLimits.MandatoryApprovals)))
		}
	}

	if pls.OutcomeLimitations != nil {
		if pls.OutcomeLimitations.MinAccuracyThreshold > 0 {
			parts = append(parts, fmt.Sprintf("MinAccuracy: %.2f", pls.OutcomeLimitations.MinAccuracyThreshold))
		}
	}

	if pls.ToolLimitation != nil {
		if len(pls.ToolLimitation.AllowedAPIs) > 0 {
			parts = append(parts, fmt.Sprintf("AllowedAPIs: %d", len(pls.ToolLimitation.AllowedAPIs)))
		}
	}

	if pls.ResourceLimits != nil {
		if pls.ResourceLimits.MaxCostPerDay > 0 {
			parts = append(parts, fmt.Sprintf("MaxCost/day: %.2f %s",
				pls.ResourceLimits.MaxCostPerDay, pls.ResourceLimits.CostCurrency))
		}
	}

	if len(parts) == 0 {
		return "No limits specified"
	}

	return strings.Join(parts, " | ")
}

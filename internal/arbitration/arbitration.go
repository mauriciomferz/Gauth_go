// Package arbitration provides hooks for dispute resolution and policy conflict arbitration (RFC 0111 sec4.item3).
package arbitration

import (
	"context"
	"fmt"
	"time"
)

// Dispute represents a policy conflict or authorization dispute requiring arbitration.
type Dispute struct {
	ID         string                 `json:"id"`
	Type       string                 `json:"type"` // "policy_conflict", "delegation_dispute", "jurisdiction_conflict"
	Subject    string                 `json:"subject"`
	Resource   string                 `json:"resource"`
	Action     string                 `json:"action"`
	Policies   []string               `json:"policies"`  // Conflicting policy IDs
	Decisions  []string               `json:"decisions"` // Conflicting decisions
	Context    map[string]interface{} `json:"context"`   // Additional context
	CreatedAt  time.Time              `json:"created_at"`
	Status     string                 `json:"status"` // "pending", "resolved", "escalated"
	Resolution string                 `json:"resolution,omitempty"`
	ResolvedBy string                 `json:"resolved_by,omitempty"`
	ResolvedAt *time.Time             `json:"resolved_at,omitempty"`
}

// ArbitrationResult represents the outcome of arbitration.
type ArbitrationResult struct {
	DisputeID          string                 `json:"dispute_id"`
	Decision           string                 `json:"decision"` // "permit", "deny", "escalate"
	Reasoning          string                 `json:"reasoning"`
	AppliedRule        string                 `json:"applied_rule,omitempty"`
	Metadata           map[string]interface{} `json:"metadata,omitempty"`
	ResolvedAt         time.Time              `json:"resolved_at"`
	RequiresEscalation bool                   `json:"requires_escalation"`
}

// Arbiter defines the interface for dispute arbitration.
type Arbiter interface {
	// Arbitrate resolves a dispute and returns the arbitration result.
	Arbitrate(ctx context.Context, dispute *Dispute) (*ArbitrationResult, error)

	// RegisterRule registers a custom arbitration rule.
	RegisterRule(name string, priority int, handler RuleHandler) error

	// GetDispute retrieves a dispute by ID.
	GetDispute(ctx context.Context, disputeID string) (*Dispute, error)

	// ListDisputes lists disputes with optional filtering.
	ListDisputes(ctx context.Context, filter DisputeFilter) ([]*Dispute, error)
}

// RuleHandler processes a dispute according to specific arbitration logic.
type RuleHandler func(ctx context.Context, dispute *Dispute) (*ArbitrationResult, error)

// DisputeFilter specifies filtering criteria for disputes.
type DisputeFilter struct {
	Status   string
	Type     string
	Subject  string
	Resource string
	Since    *time.Time
	Until    *time.Time
}

// DefaultArbiter provides a basic arbitration implementation.
type DefaultArbiter struct {
	rules map[string]ruleEntry
}

type ruleEntry struct {
	priority int
	handler  RuleHandler
}

// NewDefaultArbiter creates a new default arbiter with standard rules.
func NewDefaultArbiter() *DefaultArbiter {
	arbiter := &DefaultArbiter{
		rules: make(map[string]ruleEntry),
	}

	// Register default rules
	if err := arbiter.RegisterRule("deny_overrides", 100, denyOverridesRule); err != nil {
		panic(fmt.Sprintf("failed to register deny_overrides rule: %v", err))
	}
	if err := arbiter.RegisterRule("permit_overrides", 90, permitOverridesRule); err != nil {
		panic(fmt.Sprintf("failed to register permit_overrides rule: %v", err))
	}
	if err := arbiter.RegisterRule("first_applicable", 80, firstApplicableRule); err != nil {
		panic(fmt.Sprintf("failed to register first_applicable rule: %v", err))
	}

	return arbiter
}

// Arbitrate resolves disputes using registered rules.
func (a *DefaultArbiter) Arbitrate(ctx context.Context, dispute *Dispute) (*ArbitrationResult, error) {
	// Apply rules in priority order (highest first)
	var bestResult *ArbitrationResult
	highestPriority := -1

	for name, entry := range a.rules {
		if entry.priority > highestPriority {
			result, err := entry.handler(ctx, dispute)
			if err == nil && result != nil {
				bestResult = result
				highestPriority = entry.priority
				bestResult.AppliedRule = name
			}
		}
	}

	if bestResult == nil {
		// Default to deny if no rule could resolve
		bestResult = &ArbitrationResult{
			DisputeID:  dispute.ID,
			Decision:   "deny",
			Reasoning:  "No applicable arbitration rule found",
			ResolvedAt: time.Now(),
		}
	}

	return bestResult, nil
}

// RegisterRule adds or updates an arbitration rule.
func (a *DefaultArbiter) RegisterRule(name string, priority int, handler RuleHandler) error {
	a.rules[name] = ruleEntry{
		priority: priority,
		handler:  handler,
	}
	return nil
}

// GetDispute retrieves a dispute (stub implementation).
func (a *DefaultArbiter) GetDispute(ctx context.Context, disputeID string) (*Dispute, error) {
	// Stub: In production, this would query a dispute store
	return nil, nil
}

// ListDisputes lists disputes (stub implementation).
func (a *DefaultArbiter) ListDisputes(ctx context.Context, filter DisputeFilter) ([]*Dispute, error) {
	// Stub: In production, this would query a dispute store
	return []*Dispute{}, nil
}

// Default arbitration rules

func denyOverridesRule(ctx context.Context, dispute *Dispute) (*ArbitrationResult, error) {
	// If any decision is "deny", the result is deny
	for _, decision := range dispute.Decisions {
		if decision == "deny" {
			return &ArbitrationResult{
				DisputeID:  dispute.ID,
				Decision:   "deny",
				Reasoning:  "Deny-overrides: At least one policy denied access",
				ResolvedAt: time.Now(),
			}, nil
		}
	}
	return nil, nil
}

func permitOverridesRule(ctx context.Context, dispute *Dispute) (*ArbitrationResult, error) {
	// If any decision is "permit", the result is permit
	for _, decision := range dispute.Decisions {
		if decision == "permit" {
			return &ArbitrationResult{
				DisputeID:  dispute.ID,
				Decision:   "permit",
				Reasoning:  "Permit-overrides: At least one policy permitted access",
				ResolvedAt: time.Now(),
			}, nil
		}
	}
	return nil, nil
}

func firstApplicableRule(ctx context.Context, dispute *Dispute) (*ArbitrationResult, error) {
	// Use the first decision in the list
	if len(dispute.Decisions) > 0 {
		return &ArbitrationResult{
			DisputeID:  dispute.ID,
			Decision:   dispute.Decisions[0],
			Reasoning:  "First-applicable: Using first decision in evaluation order",
			ResolvedAt: time.Now(),
		}, nil
	}
	return nil, nil
}

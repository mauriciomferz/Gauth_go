// Package agentauth - AgentAuth+ Integration for Authorization Chain Validation
// Integrates AgentAuth+ features (successor management, delegation policies, dual control,
// fiduciary duties, capability assessment) into AAP001 authorization flow
package agentauth

import (
	"context"
	"fmt"
	"time"

	"github.com/mauriciomferz/AgentAuth/pkg/agentauthplus"
	"github.com/mauriciomferz/AgentAuth/pkg/metrics"
	"github.com/mauriciomferz/AgentAuth/pkg/poa"
)

// AgentAuthPlusValidator validates AgentAuth+ policies during authorization
type AgentAuthPlusValidator struct {
	successorService    *agentauthplus.PostgreSQLSuccessorService
	delegationService   agentauthplus.DelegationService // Use interface for caching support
	dualControlService  *agentauthplus.PostgreSQLDualControlService
	fiduciaryService    *agentauthplus.PostgreSQLFiduciaryDutyService
	capabilityService   agentauthplus.CapabilityAssessmentService // Use interface for caching support
	enforceCapabilities bool
	enforceDualControl  bool
	enforceFiduciary    bool
}

// NewAgentAuthPlusValidator creates a new AgentAuth+ validator
// Accepts interface types for delegation and capability services to support caching
func NewAgentAuthPlusValidator(
	successorService *agentauthplus.PostgreSQLSuccessorService,
	delegationService agentauthplus.DelegationService,
	dualControlService *agentauthplus.PostgreSQLDualControlService,
	fiduciaryService *agentauthplus.PostgreSQLFiduciaryDutyService,
	capabilityService agentauthplus.CapabilityAssessmentService,
) *AgentAuthPlusValidator {
	return &AgentAuthPlusValidator{
		successorService:    successorService,
		delegationService:   delegationService,
		dualControlService:  dualControlService,
		fiduciaryService:    fiduciaryService,
		capabilityService:   capabilityService,
		enforceCapabilities: true,
		enforceDualControl:  true,
		enforceFiduciary:    true,
	}
}

// AgentAuthPlusValidationResult contains results from AgentAuth+ validation
type AgentAuthPlusValidationResult struct {
	Valid            bool                    `json:"valid"`
	SuccessorCheck   *SuccessorCheckResult   `json:"successor_check,omitempty"`
	DelegationCheck  *DelegationCheckResult  `json:"delegation_check,omitempty"`
	DualControlCheck *DualControlCheckResult `json:"dual_control_check,omitempty"`
	CapabilityCheck  *CapabilityCheckResult  `json:"capability_check,omitempty"`
	FiduciaryCheck   *FiduciaryCheckResult   `json:"fiduciary_check,omitempty"`
	Warnings         []string                `json:"warnings,omitempty"`
	FailureReason    string                  `json:"failure_reason,omitempty"`
}

// SuccessorCheckResult contains successor validation results
type SuccessorCheckResult struct {
	CheckPerformed   bool                               `json:"check_performed"`
	SuccessorActive  bool                               `json:"successor_active"`
	ActiveSuccessor  *agentauthplus.SuccessorActivation `json:"active_successor,omitempty"`
	EffectiveAgentID string                             `json:"effective_agent_id"` // primary or successor
}

// DelegationCheckResult contains delegation validation results
type DelegationCheckResult struct {
	CheckPerformed  bool                          `json:"check_performed"`
	DelegationValid bool                          `json:"delegation_valid"`
	CurrentDepth    int                           `json:"current_depth"`
	MaxAllowedDepth int                           `json:"max_allowed_depth"`
	DelegationChain []*agentauthplus.AIDelegation `json:"delegation_chain,omitempty"`
	DepthExceeded   bool                          `json:"depth_exceeded"`
	Warnings        []string                      `json:"warnings,omitempty"`
}

// DualControlCheckResult contains dual control validation results
type DualControlCheckResult struct {
	CheckPerformed    bool                               `json:"check_performed"`
	RequiresApproval  bool                               `json:"requires_approval"`
	ApprovalObtained  bool                               `json:"approval_obtained"`
	PendingApproval   *agentauthplus.DualControlApproval `json:"pending_approval,omitempty"`
	ApprovedAction    *agentauthplus.DualControlApproval `json:"approved_action,omitempty"`
	RequiredApprovers int                                `json:"required_approvers"`
	CurrentApprovers  int                                `json:"current_approvers"`
}

// CapabilityCheckResult contains capability assessment results
type CapabilityCheckResult struct {
	CheckPerformed    bool                                  `json:"check_performed"`
	CapabilityMet     bool                                  `json:"capability_met"`
	LatestAssessment  *agentauthplus.AICapabilityAssessment `json:"latest_assessment,omitempty"`
	RequiredLevel     string                                `json:"required_level,omitempty"`
	ActualLevel       string                                `json:"actual_level,omitempty"`
	DomainMatches     map[string]bool                       `json:"domain_matches,omitempty"`
	AssessmentExpired bool                                  `json:"assessment_expired"`
}

// FiduciaryCheckResult contains fiduciary duty validation results
type FiduciaryCheckResult struct {
	CheckPerformed       bool                                    `json:"check_performed"`
	HasViolations        bool                                    `json:"has_violations"`
	UnresolvedViolations []*agentauthplus.FiduciaryDutyViolation `json:"unresolved_violations,omitempty"`
	CriticalViolations   int                                     `json:"critical_violations"`
	BlockingAction       bool                                    `json:"blocking_action"` // whether violations block authorization
}

// ValidatePoAWithAgentAuthPlus performs comprehensive AgentAuth+ validation for a PoA
// This is called during AAP001 authorization chain validation to enforce
// AgentAuth+ policies (successor management, delegation, dual control, capabilities, fiduciary duties)
func (v *AgentAuthPlusValidator) ValidatePoAWithAgentAuthPlus(
	ctx context.Context,
	poaID string,
	poaDef *poa.PoADefinition,
	agentID string,
	actionType string,
) (*AgentAuthPlusValidationResult, error) {
	result := &AgentAuthPlusValidationResult{
		Valid:    true,
		Warnings: []string{},
	}

	// Step 1: Check for successor activation
	successorResult, err := v.checkSuccessorStatus(ctx, poaID, agentID)
	if err != nil {
		return nil, fmt.Errorf("successor check failed: %w", err)
	}
	result.SuccessorCheck = successorResult

	// If successor is active, use successor agent ID for subsequent checks
	effectiveAgentID := agentID
	if successorResult.SuccessorActive {
		effectiveAgentID = successorResult.ActiveSuccessor.SuccessorAgentID
		result.Warnings = append(result.Warnings,
			fmt.Sprintf("Successor AI %s is active, taking over from primary AI %s",
				effectiveAgentID, agentID))
	}

	// Step 2: Validate delegation chain
	delegationResult, err := v.checkDelegationChain(ctx, poaID, effectiveAgentID)
	if err != nil {
		return nil, fmt.Errorf("delegation check failed: %w", err)
	}
	result.DelegationCheck = delegationResult

	if delegationResult.DepthExceeded {
		result.Valid = false
		result.FailureReason = fmt.Sprintf("delegation depth %d exceeds maximum %d",
			delegationResult.CurrentDepth, delegationResult.MaxAllowedDepth)
		return result, nil
	}

	// Step 3: Check dual control requirements
	if v.enforceDualControl {
		dualControlResult, err := v.checkDualControlRequirements(ctx, poaID, effectiveAgentID, actionType)
		if err != nil {
			return nil, fmt.Errorf("dual control check failed: %w", err)
		}
		result.DualControlCheck = dualControlResult

		if dualControlResult.RequiresApproval && !dualControlResult.ApprovalObtained {
			result.Valid = false
			result.FailureReason = fmt.Sprintf("action %s requires dual control approval but none obtained", actionType)
			return result, nil
		}
	}

	// Step 4: Check capability requirements
	if v.enforceCapabilities {
		capabilityResult, err := v.checkCapabilityRequirements(ctx, effectiveAgentID, poaDef)
		if err != nil {
			return nil, fmt.Errorf("capability check failed: %w", err)
		}
		result.CapabilityCheck = capabilityResult

		if !capabilityResult.CapabilityMet {
			result.Valid = false
			result.FailureReason = fmt.Sprintf("agent %s does not meet capability requirements", effectiveAgentID)
			return result, nil
		}

		if capabilityResult.AssessmentExpired {
			result.Warnings = append(result.Warnings, "capability assessment is expired, re-assessment recommended")
		}
	}

	// Step 5: Check fiduciary duty violations
	if v.enforceFiduciary {
		fiduciaryResult, err := v.checkFiduciaryDuties(ctx, effectiveAgentID, poaID)
		if err != nil {
			return nil, fmt.Errorf("fiduciary check failed: %w", err)
		}
		result.FiduciaryCheck = fiduciaryResult

		if fiduciaryResult.BlockingAction {
			result.Valid = false
			result.FailureReason = fmt.Sprintf("agent has %d critical unresolved fiduciary violations",
				fiduciaryResult.CriticalViolations)
			return result, nil
		}

		if fiduciaryResult.HasViolations {
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("agent has %d unresolved fiduciary violations",
					len(fiduciaryResult.UnresolvedViolations)))
		}
	}

	return result, nil
}

// checkSuccessorStatus checks if a successor AI is active
func (v *AgentAuthPlusValidator) checkSuccessorStatus(
	ctx context.Context,
	poaID string,
	primaryAgentID string,
) (*SuccessorCheckResult, error) {
	start := time.Now()
	defer func() {
		metrics.RecordAgentAuthPlusValidation("successor", "checked", time.Since(start).Seconds())
	}()

	result := &SuccessorCheckResult{
		CheckPerformed:   true,
		SuccessorActive:  false,
		EffectiveAgentID: primaryAgentID,
	}

	if v.successorService == nil {
		result.CheckPerformed = false
		return result, nil
	}

	activeSuccessor, err := v.successorService.GetActiveSuccessor(ctx, poaID)
	if err != nil {
		return nil, err
	}

	if activeSuccessor != nil {
		result.SuccessorActive = true
		result.ActiveSuccessor = activeSuccessor
		result.EffectiveAgentID = activeSuccessor.SuccessorAgentID
		metrics.RecordAgentAuthPlusSuccessorActivation()
	}

	return result, nil
}

// checkDelegationChain validates the AI-to-AI delegation chain
func (v *AgentAuthPlusValidator) checkDelegationChain(
	ctx context.Context,
	poaID string,
	agentID string,
) (*DelegationCheckResult, error) {
	start := time.Now()
	defer func() {
		metrics.RecordAgentAuthPlusValidation("delegation", "checked", time.Since(start).Seconds())
	}()

	result := &DelegationCheckResult{
		CheckPerformed:  true,
		DelegationValid: true,
		DelegationChain: []*agentauthplus.AIDelegation{},
		Warnings:        []string{},
	}

	if v.delegationService == nil {
		result.CheckPerformed = false
		return result, nil
	}

	// Get delegation chain - service takes only agentID
	chain, err := v.delegationService.GetDelegationChain(ctx, agentID)
	if err != nil {
		return nil, err
	}

	result.DelegationChain = chain
	result.CurrentDepth = len(chain)

	// Record delegation depth metric
	if len(chain) > 0 {
		metrics.RecordAgentAuthPlusDelegationDepth(len(chain))
	}

	// Check max depth from first delegation (if any)
	if len(chain) > 0 {
		result.MaxAllowedDepth = chain[0].MaxAllowedDepth

		// Validate depth - CheckMaxDepthExceeded takes sourceAgentID and currentDepth
		depthExceeded, err := v.delegationService.CheckMaxDepthExceeded(ctx, agentID, len(chain))
		if err != nil {
			return nil, err
		}

		result.DepthExceeded = depthExceeded
		if depthExceeded {
			result.DelegationValid = false
		}

		// Validate each delegation in chain
		// ValidateDelegation needs sourceAgentID, targetAgentID, scope, depth
		for i, delegation := range chain {
			err := v.delegationService.ValidateDelegation(
				ctx,
				delegation.SourceAgentID,
				delegation.TargetAgentID,
				delegation.DelegatedScope,
				delegation.DelegationDepth,
			)
			if err != nil {
				result.DelegationValid = false
				result.Warnings = append(result.Warnings,
					fmt.Sprintf("delegation %d invalid: %s", i, err.Error()))
			}
		}
	}

	return result, nil
}

// checkDualControlRequirements validates dual control approval requirements
func (v *AgentAuthPlusValidator) checkDualControlRequirements(
	ctx context.Context,
	poaID string,
	agentID string,
	actionType string,
) (*DualControlCheckResult, error) {
	result := &DualControlCheckResult{
		CheckPerformed:   true,
		RequiresApproval: false,
		ApprovalObtained: false,
	}

	if v.dualControlService == nil {
		result.CheckPerformed = false
		return result, nil
	}

	// Check if action type requires dual control
	// Query for approval records matching this PoA and action type
	approvals, err := v.dualControlService.FindApprovalsByPoAAndAction(ctx, poaID, actionType)
	if err != nil {
		return nil, fmt.Errorf("failed to query dual control approvals: %w", err)
	}

	// Analyze approval status
	result.RequiresApproval = false
	result.ApprovalObtained = false

	if len(approvals) > 0 {
		result.RequiresApproval = true

		// Check if we have any approved, non-expired approvals
		now := time.Now().UTC()
		for _, approval := range approvals {
			if approval.Status == "approved" {
				// Check if approval is still valid (not expired)
				if approval.ExpiresAt == nil || approval.ExpiresAt.After(now) {
					result.ApprovalObtained = true
					result.ApprovedAction = approval
					result.CurrentApprovers = len(approval.ApprovedBy)
					result.RequiredApprovers = approval.RequiredApprovers
					break
				}
			} else if approval.Status == "pending" {
				result.PendingApproval = approval
				result.CurrentApprovers = len(approval.ApprovedBy)
				result.RequiredApprovers = approval.RequiredApprovers
			}
		}
	}

	return result, nil
}

// checkCapabilityRequirements validates AI capability against requirements
func (v *AgentAuthPlusValidator) checkCapabilityRequirements(
	ctx context.Context,
	agentID string,
	poaDef *poa.PoADefinition,
) (*CapabilityCheckResult, error) {
	result := &CapabilityCheckResult{
		CheckPerformed:    true,
		CapabilityMet:     true,
		DomainMatches:     make(map[string]bool),
		AssessmentExpired: false,
	}

	if v.capabilityService == nil {
		result.CheckPerformed = false
		return result, nil
	}

	// Get latest capability assessment
	assessment, err := v.capabilityService.GetLatestAssessment(ctx, agentID)
	if err != nil {
		return nil, err
	}

	if assessment == nil {
		result.CapabilityMet = false
		return result, nil
	}

	result.LatestAssessment = assessment
	result.ActualLevel = assessment.OverallLevel

	// Check if assessment is expired (ValidUntil is time.Time not pointer)
	if time.Now().After(assessment.ValidUntil) {
		result.AssessmentExpired = true
	}

	// Extract capability requirements from PoA
	// This would come from PoA definition's authorization constraints
	// For demonstration, we check if assessment exists and is not expired
	requiredLevel := "L2" // Default minimum for authorized AI agents
	result.RequiredLevel = requiredLevel

	// Compare capability levels (L0 < L1 < L2 < L3 < L4 < L5)
	if !v.compareCapabilityLevels(assessment.OverallLevel, requiredLevel) {
		result.CapabilityMet = false
	}

	// Check domain-specific requirements if present
	// This would be enhanced with actual domain requirements from PoA
	for domain := range assessment.DomainScores {
		result.DomainMatches[domain] = true
	}

	return result, nil
}

// checkFiduciaryDuties checks for fiduciary duty violations
func (v *AgentAuthPlusValidator) checkFiduciaryDuties(
	ctx context.Context,
	agentID string,
	poaID string,
) (*FiduciaryCheckResult, error) {
	result := &FiduciaryCheckResult{
		CheckPerformed:       true,
		HasViolations:        false,
		UnresolvedViolations: []*agentauthplus.FiduciaryDutyViolation{},
		CriticalViolations:   0,
		BlockingAction:       false,
	}

	if v.fiduciaryService == nil {
		result.CheckPerformed = false
		return result, nil
	}

	// Get violations for this agent and PoA (GetViolations takes poaID, agentID)
	allViolations, err := v.fiduciaryService.GetViolations(ctx, poaID, agentID)
	if err != nil {
		return nil, err
	}

	// Filter to only unresolved violations
	for _, violation := range allViolations {
		if violation.ResolutionStatus == "open" || violation.ResolutionStatus == "investigating" {
			result.UnresolvedViolations = append(result.UnresolvedViolations, violation)
		}
	}

	result.HasViolations = len(result.UnresolvedViolations) > 0

	// Count critical violations
	for _, violation := range result.UnresolvedViolations {
		if violation.Severity == "critical" {
			result.CriticalViolations++
		}
	}

	// Block authorization if there are critical violations
	if result.CriticalViolations > 0 {
		result.BlockingAction = true
	}

	return result, nil
}

// compareCapabilityLevels compares two capability levels
// Returns true if actual >= required
func (v *AgentAuthPlusValidator) compareCapabilityLevels(actual, required string) bool {
	levels := map[string]int{
		"L0": 0,
		"L1": 1,
		"L2": 2,
		"L3": 3,
		"L4": 4,
		"L5": 5,
	}

	actualLevel, okActual := levels[actual]
	requiredLevel, okRequired := levels[required]

	if !okActual || !okRequired {
		return false
	}

	return actualLevel >= requiredLevel
}

// SetEnforceCapabilities enables/disables capability enforcement
func (v *AgentAuthPlusValidator) SetEnforceCapabilities(enforce bool) {
	v.enforceCapabilities = enforce
}

// SetEnforceDualControl enables/disables dual control enforcement
func (v *AgentAuthPlusValidator) SetEnforceDualControl(enforce bool) {
	v.enforceDualControl = enforce
}

// SetEnforceFiduciary enables/disables fiduciary duty enforcement
func (v *AgentAuthPlusValidator) SetEnforceFiduciary(enforce bool) {
	v.enforceFiduciary = enforce
}

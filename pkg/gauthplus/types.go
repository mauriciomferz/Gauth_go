package gauthplus

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// ObligationType defines the nature of AI authorization
type ObligationType string

const (
	// ObligationPermissive - "do-unless" - AI can act unless explicitly prohibited
	ObligationPermissive ObligationType = "permissive"
	// ObligationMandatory - "need-to-do" - AI must perform certain actions
	ObligationMandatory ObligationType = "mandatory"
	// ObligationProhibitive - AI is explicitly forbidden from certain actions
	ObligationProhibitive ObligationType = "prohibitive"
)

// DelegationPolicy defines rules for AI-to-AI delegation
type DelegationPolicy struct {
	CanDelegate      bool     `json:"can_delegate"`
	MaxDepth         int      `json:"max_depth"`          // Maximum delegation chain depth
	AllowedDelegates []string `json:"allowed_delegates"`  // Whitelist of allowed delegate agent IDs
	RequiresApproval bool     `json:"requires_approval"`  // Whether delegation needs human approval
	ScopeRestriction string   `json:"scope_restriction"`  // "maintain", "reduce_only", "none"
	TimeRestriction  string   `json:"time_restriction"`   // "maintain", "reduce_only", "extend_allowed"
}

// FiduciaryDuties represents duties the AI agent must uphold
type FiduciaryDuties struct {
	DutyOfCare         bool     `json:"duty_of_care"`           // Act with reasonable care and diligence
	DutyOfLoyalty      bool     `json:"duty_of_loyalty"`        // Act in principal's best interest
	DutyOfGoodFaith    bool     `json:"duty_of_good_faith"`     // Act honestly and in good faith
	DutyOfDisclosure   bool     `json:"duty_of_disclosure"`     // Disclose conflicts of interest
	DutyOfConfidential bool     `json:"duty_of_confidential"`   // Maintain confidentiality
	SpecificDuties     []string `json:"specific_duties"`        // Industry/jurisdiction specific duties
	ViolationConseq    string   `json:"violation_consequences"` // "warn", "suspend", "revoke"
}

// CapabilityRequirements specifies required AI capability levels
type CapabilityRequirements struct {
	MinimumLevel            string             `json:"minimum_level"`      // "L0" through "L5"
	DomainScores            map[string]float64 `json:"domain_scores"`      // domain -> min_score (0.0-1.0) (matches service usage)
	RiskThresholds          map[string]float64 `json:"risk_thresholds"`    // risk category -> threshold (matches service usage)
	RequiresCert            bool               `json:"requires_cert"`      // Requires formal certification
	RequiredCertifications  []string           `json:"required_certifications"` // Required certification IDs (matches service usage)
}

// SuccessorActivation represents a successor AI taking over
type SuccessorActivation struct {
	ID               string                 `json:"id"`
	POAID            string                 `json:"poa_id"`
	PrimaryAgentID   string                 `json:"primary_agent_id"`
	SuccessorAgentID string                 `json:"successor_agent_id"`
	ActivationReason string                 `json:"activation_reason"` // unavailable, failure, manual, timeout
	ActivatedAt      time.Time              `json:"activated_at"`
	ActivatedBy      string                 `json:"activated_by"`
	DeactivatedAt    *time.Time             `json:"deactivated_at,omitempty"`
	DeactivatedBy    string                 `json:"deactivated_by,omitempty"`
	Status           string                 `json:"status"` // active, deactivated, superseded
	Metadata         map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt        time.Time              `json:"created_at"`
	UpdatedAt        time.Time              `json:"updated_at"`
}

// AIDelegation represents an AI-to-AI delegation
type AIDelegation struct {
	ID               string                 `json:"id"`
	SourcePOAID      string                 `json:"source_poa_id"`
	SourceAgentID    string                 `json:"source_agent_id"`
	TargetAgentID    string                 `json:"target_agent_id"`
	DelegatedScope   []string               `json:"delegated_scope"`
	DelegationDepth  int                    `json:"delegation_depth"`
	MaxAllowedDepth  int                    `json:"max_allowed_depth"`
	ValidFrom        time.Time              `json:"valid_from"`
	ValidUntil       time.Time              `json:"valid_until"`
	Status           string                 `json:"status"` // active, revoked, expired
	DelegationPolicy *DelegationPolicy      `json:"delegation_policy,omitempty"`
	RevokedAt        *time.Time             `json:"revoked_at,omitempty"`
	RevokedBy        string                 `json:"revoked_by,omitempty"`
	RevocationReason string                 `json:"revocation_reason,omitempty"`
	CreatedAt        time.Time              `json:"created_at"`
	UpdatedAt        time.Time              `json:"updated_at"`
}

// DualControlApproval represents a second-level approval workflow
type DualControlApproval struct {
	ID                  string                 `json:"id"`
	POAID               string                 `json:"poa_id"`
	ActionType          string                 `json:"action_type"`
	ActionDescription   string                 `json:"action_description"`
	RequestedBy         string                 `json:"requested_by"`
	RequestedAt         time.Time              `json:"requested_at"`
	RequiredApprovers   int                    `json:"required_approvers"`
	ApprovalThreshold   string                 `json:"approval_threshold"` // all, majority, quorum, weighted
	Status              string                 `json:"status"`             // pending, approved, rejected, expired
	ApprovedBy          []ApprovalRecord       `json:"approved_by"`
	RejectedBy          []ApprovalRecord       `json:"rejected_by"`
	DecisionFinalizedAt *time.Time             `json:"decision_finalized_at,omitempty"`
	ExpiresAt           *time.Time             `json:"expires_at,omitempty"`
	Metadata            map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt           time.Time              `json:"created_at"`
	UpdatedAt           time.Time              `json:"updated_at"`
}

// ApprovalRecord tracks individual approver decisions
type ApprovalRecord struct {
	ApproverID string    `json:"approver_id"`
	ApprovedAt time.Time `json:"approved_at"`
	Comments   string    `json:"comments,omitempty"`
	Weight     int       `json:"weight,omitempty"` // For weighted voting
}

// FiduciaryDutyViolation tracks duty breaches
type FiduciaryDutyViolation struct {
	ID                   string                 `json:"id"`
	POAID                string                 `json:"poa_id"`
	AgentID              string                 `json:"agent_id"`
	DutyType             string                 `json:"duty_type"` // care, loyalty, good_faith, disclosure, confidentiality
	ViolationDescription string                 `json:"violation_description"`
	Severity             string                 `json:"severity"` // minor, moderate, major, critical
	DetectedAt           time.Time              `json:"detected_at"`
	DetectedBy           string                 `json:"detected_by"`
	ReviewedBy           string                 `json:"reviewed_by,omitempty"`
	ReviewedAt           *time.Time             `json:"reviewed_at,omitempty"`
	ResolutionStatus     string                 `json:"resolution_status"` // open, investigating, resolved, dismissed
	ResolutionNotes      string                 `json:"resolution_notes,omitempty"`
	Consequences         map[string]interface{} `json:"consequences,omitempty"`
	Evidence             map[string]interface{} `json:"evidence,omitempty"`
	CreatedAt            time.Time              `json:"created_at"`
	UpdatedAt            time.Time              `json:"updated_at"`
}

// AICapabilityAssessment represents periodic AI capability evaluation
type AICapabilityAssessment struct {
	ID                      string             `json:"id"`
	AgentID                 string             `json:"agent_id"`
	AssessmentDate          time.Time          `json:"assessment_date"`
	OverallLevel            string             `json:"overall_level"` // L0 through L5 (matches DB column)
	DomainScores            map[string]float64 `json:"domain_scores"` // domain -> score (0.0-1.0) (matches DB column)
	RiskProfile             map[string]interface{} `json:"risk_profile"` // Risk assessment details (matches DB column)
	CertificationStatus     string             `json:"certification_status"` // uncertified, pending, certified, expired
	Certifications          []string           `json:"certifications"`
	Limitations             []string           `json:"limitations,omitempty"`
	RecommendedRestrictions []string           `json:"recommended_restrictions,omitempty"`
	AssessedBy              string             `json:"assessed_by"`
	ValidUntil              time.Time          `json:"valid_until"`
	Notes                   string             `json:"notes,omitempty"` // Assessment notes (matches DB column)
	SupersededBy            string             `json:"superseded_by,omitempty"`
	CreatedAt               time.Time          `json:"created_at"`
	UpdatedAt               time.Time          `json:"updated_at"`
}

// SuccessorManagementService handles successor AI activation and management
type SuccessorManagementService interface {
	// ActivateSuccessor activates the successor AI when primary fails
	ActivateSuccessor(ctx context.Context, poaID, primaryAgentID, successorAgentID, reason, activatedBy string) (*SuccessorActivation, error)
	// DeactivateSuccessor returns control to primary AI
	DeactivateSuccessor(ctx context.Context, activationID, deactivatedBy string) error
	// GetActiveSuccessor returns current successor activation if any
	GetActiveSuccessor(ctx context.Context, poaID string) (*SuccessorActivation, error)
	// ListSuccessorHistory returns activation history
	ListSuccessorHistory(ctx context.Context, poaID string) ([]*SuccessorActivation, error)
}

// DelegationService handles AI-to-AI delegations
type DelegationService interface {
	// CreateDelegation creates new AI-to-AI delegation
	CreateDelegation(ctx context.Context, delegation *AIDelegation) error
	// ValidateDelegation checks if delegation is allowed per policy
	ValidateDelegation(ctx context.Context, sourceAgentID, targetAgentID string, scope []string, depth int) error
	// RevokeDelegation revokes an active delegation
	RevokeDelegation(ctx context.Context, delegationID, revokedBy, reason string) error
	// GetDelegationChain returns full delegation chain
	GetDelegationChain(ctx context.Context, agentID string) ([]*AIDelegation, error)
	// CheckMaxDepthExceeded checks if adding delegation would exceed max depth
	CheckMaxDepthExceeded(ctx context.Context, sourceAgentID string, currentDepth int) (bool, error)
}

// DualControlService handles second-level approvals
type DualControlService interface {
	// RequestApproval initiates approval workflow
	RequestApproval(ctx context.Context, approval *DualControlApproval) (string, error)
	// ApproveAction records approver's approval
	ApproveAction(ctx context.Context, approvalID, approverID, comments string) error
	// RejectAction records approver's rejection
	RejectAction(ctx context.Context, approvalID, approverID, comments string) error
	// CheckApprovalStatus checks if threshold met
	CheckApprovalStatus(ctx context.Context, approvalID string) (string, error) // approved, rejected, pending
	// GetPendingApprovals returns approvals awaiting decision
	GetPendingApprovals(ctx context.Context, approverID string) ([]*DualControlApproval, error)
	// FindApprovalsByPoAAndAction queries approvals matching PoA and action type
	FindApprovalsByPoAAndAction(ctx context.Context, poaID, actionType string) ([]*DualControlApproval, error)
}

// FiduciaryDutyService tracks and manages fiduciary duties
type FiduciaryDutyService interface {
	// RecordViolation records a fiduciary duty breach
	RecordViolation(ctx context.Context, violation *FiduciaryDutyViolation) error
	// GetViolations returns violations for agent or PoA
	GetViolations(ctx context.Context, poaID, agentID string) ([]*FiduciaryDutyViolation, error)
	// ResolveViolation marks violation as resolved
	ResolveViolation(ctx context.Context, violationID, reviewedBy, notes string) error
	// GetViolationsBySeverity returns violations above severity threshold
	GetViolationsBySeverity(ctx context.Context, minSeverity string) ([]*FiduciaryDutyViolation, error)
}

// CapabilityAssessmentService handles AI capability evaluations
type CapabilityAssessmentService interface {
	// CreateAssessment creates new capability assessment
	CreateAssessment(ctx context.Context, assessment *AICapabilityAssessment) error
	// GetLatestAssessment returns most recent assessment for agent
	GetLatestAssessment(ctx context.Context, agentID string) (*AICapabilityAssessment, error)
	// CheckCapabilityMatch verifies agent meets PoA requirements
	CheckCapabilityMatch(ctx context.Context, agentID string, requirements *CapabilityRequirements) (bool, []string, error)
	// GetExpiringAssessments returns assessments expiring soon
	GetExpiringAssessments(ctx context.Context, daysUntilExpiry int) ([]*AICapabilityAssessment, error)
}

// Helper functions for policy validation

// ValidateDelegationPolicy checks if delegation policy is valid
func ValidateDelegationPolicy(policy *DelegationPolicy) error {
	if policy == nil {
		return nil // Policy is optional
	}

	if policy.MaxDepth < 0 {
		return fmt.Errorf("max_depth cannot be negative")
	}

	if policy.MaxDepth > 10 {
		return fmt.Errorf("max_depth cannot exceed 10 to prevent excessive delegation chains")
	}

	validScopeRestrictions := map[string]bool{
		"maintain":    true,
		"reduce_only": true,
		"none":        true,
	}
	if policy.ScopeRestriction != "" && !validScopeRestrictions[policy.ScopeRestriction] {
		return fmt.Errorf("invalid scope_restriction: %s", policy.ScopeRestriction)
	}

	validTimeRestrictions := map[string]bool{
		"maintain":       true,
		"reduce_only":    true,
		"extend_allowed": true,
	}
	if policy.TimeRestriction != "" && !validTimeRestrictions[policy.TimeRestriction] {
		return fmt.Errorf("invalid time_restriction: %s", policy.TimeRestriction)
	}

	return nil
}

// ValidateFiduciaryDuties checks if fiduciary duties are properly configured
func ValidateFiduciaryDuties(duties *FiduciaryDuties) error {
	if duties == nil {
		return nil // Duties are optional
	}

	if !duties.DutyOfCare && !duties.DutyOfLoyalty && !duties.DutyOfGoodFaith &&
		!duties.DutyOfDisclosure && !duties.DutyOfConfidential && len(duties.SpecificDuties) == 0 {
		return fmt.Errorf("at least one fiduciary duty must be specified")
	}

	validConsequences := map[string]bool{
		"warn":    true,
		"suspend": true,
		"revoke":  true,
	}
	if duties.ViolationConseq != "" && !validConsequences[duties.ViolationConseq] {
		return fmt.Errorf("invalid violation_consequences: %s", duties.ViolationConseq)
	}

	return nil
}

// ValidateCapabilityRequirements checks if capability requirements are valid
func ValidateCapabilityRequirements(reqs *CapabilityRequirements) error {
	if reqs == nil {
		return nil // Requirements are optional
	}

	validLevels := map[string]bool{
		"L0": true, "L1": true, "L2": true, "L3": true, "L4": true, "L5": true,
	}
	if reqs.MinimumLevel != "" && !validLevels[reqs.MinimumLevel] {
		return fmt.Errorf("invalid minimum_level: %s", reqs.MinimumLevel)
	}

	for domain, score := range reqs.DomainScores {
		if score < 0.0 || score > 1.0 {
			return fmt.Errorf("domain %s score must be between 0.0 and 1.0", domain)
		}
	}

	for category, threshold := range reqs.RiskThresholds {
		if threshold < 0.0 || threshold > 1.0 {
			return fmt.Errorf("risk threshold %s must be between 0.0 and 1.0", category)
		}
	}

	return nil
}

// MarshalDelegationPolicy converts policy to JSON for database storage
func MarshalDelegationPolicy(policy *DelegationPolicy) ([]byte, error) {
	if policy == nil {
		return nil, nil
	}
	return json.Marshal(policy)
}

// UnmarshalDelegationPolicy converts JSON to policy struct
func UnmarshalDelegationPolicy(data []byte) (*DelegationPolicy, error) {
	if data == nil {
		return nil, nil
	}
	var policy DelegationPolicy
	if err := json.Unmarshal(data, &policy); err != nil {
		return nil, err
	}
	return &policy, nil
}

// MarshalFiduciaryDuties converts duties to JSON for database storage
func MarshalFiduciaryDuties(duties *FiduciaryDuties) ([]byte, error) {
	if duties == nil {
		return nil, nil
	}
	return json.Marshal(duties)
}

// UnmarshalFiduciaryDuties converts JSON to duties struct
func UnmarshalFiduciaryDuties(data []byte) (*FiduciaryDuties, error) {
	if data == nil {
		return nil, nil
	}
	var duties FiduciaryDuties
	if err := json.Unmarshal(data, &duties); err != nil {
		return nil, err
	}
	return &duties, nil
}

// MarshalCapabilityRequirements converts requirements to JSON
func MarshalCapabilityRequirements(reqs *CapabilityRequirements) ([]byte, error) {
	if reqs == nil {
		return nil, nil
	}
	return json.Marshal(reqs)
}

// UnmarshalCapabilityRequirements converts JSON to requirements struct
func UnmarshalCapabilityRequirements(data []byte) (*CapabilityRequirements, error) {
	if data == nil {
		return nil, nil
	}
	var reqs CapabilityRequirements
	if err := json.Unmarshal(data, &reqs); err != nil {
		return nil, err
	}
	return &reqs, nil
}

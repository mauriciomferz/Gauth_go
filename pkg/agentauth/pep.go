// Package agentauth - AAP001 Power Enforcement Point (PEP)
// This implements the PEP component of the P*P architecture (AAP001 Section 3.1)
package agentauth

import (
	"context"
	"fmt"
	"time"

	"github.com/mauriciomferz/AgentAuth/pkg/poa"
)

// PowerEnforcementPoint (PEP) enforces authorization decisions at runtime
// AAP001 Section 3.1: "Power Enforcement Point (PEP) – usually the application, AI system or an
// intermediary that asks the PDP for a decision and enforces its result. AgentAuth differentiates between
// supply-side and demand-side PEP. The client itself Must make sure it decides and acts in line with
// its authorization, thus enforces compliance from the supply-side. The resource owner and/or resource
// server Must check authorization compliance of the transactions, actions and decisions of the client
// and its owner as demand-side."
type PowerEnforcementPoint struct {
	// Token validator for verifying extended tokens
	tokenValidator TokenValidator

	// PDP for authorization decisions
	pdp PowerDecisionPoint

	// Audit logger for tracking enforcement actions
	auditLogger PEPAuditLogger

	// Compliance tracker for monitoring behavior
	complianceTracker ComplianceTracker

	// Enforcement mode: "strict" or "advisory"
	enforcementMode string
}

// PowerDecisionPoint makes authorization decisions
type PowerDecisionPoint interface {
	MakeDecision(ctx context.Context, request *AuthorizationDecisionRequest) (*AuthorizationDecision, error)
}

// PEPAuditLogger logs enforcement actions
type PEPAuditLogger interface {
	LogEnforcement(ctx context.Context, entry *EnforcementAuditEntry) error
	LogViolation(ctx context.Context, entry *ViolationAuditEntry) error
}

// NewPowerEnforcementPoint creates a new PEP instance
func NewPowerEnforcementPoint(
	tokenValidator TokenValidator,
	pdp PowerDecisionPoint,
	auditLogger PEPAuditLogger,
	complianceTracker ComplianceTracker,
	enforcementMode string,
) *PowerEnforcementPoint {
	if enforcementMode == "" {
		enforcementMode = "strict" // Default to strict enforcement
	}

	return &PowerEnforcementPoint{
		tokenValidator:    tokenValidator,
		pdp:               pdp,
		auditLogger:       auditLogger,
		complianceTracker: complianceTracker,
		enforcementMode:   enforcementMode,
	}
}

// EnforcementRequest represents a request for authorization enforcement
type EnforcementRequest struct {
	// Extended token for authentication
	ExtendedToken string

	// Action details
	ActionType        string // "transaction", "decision", "action"
	ActionDescription string

	// Resource information
	ResourceID      string
	ResourceType    string
	ResourceOwnerID string

	// Transaction details (if applicable)
	TransactionType     poa.TransactionType
	TransactionAmount   float64
	TransactionCurrency string

	// Decision details (if applicable)
	DecisionType    poa.DecisionType
	DecisionSubject string
	DecisionImpact  string

	// Action details (if applicable)
	PhysicalAction bool
	ActionLocation string

	// Context
	Context   map[string]interface{}
	Timestamp time.Time
}

// EnforcementResult represents the result of enforcement
type EnforcementResult struct {
	// Enforcement decision
	Allowed  bool
	Decision *AuthorizationDecision

	// Validation results
	TokenValid        bool
	TokenExpired      bool
	ScopeValid        bool
	RestrictionsValid bool
	ComplianceValid   bool

	// Violations (if any)
	Violations []EnforcementViolation

	// Audit information
	EnforcementID string
	EnforcedAt    time.Time

	// Reason for decision
	AllowReason string
	DenyReason  string
}

// EnforcementViolation represents a detected violation
type EnforcementViolation struct {
	ViolationType string // "scope", "restriction", "compliance", "temporal", "geographic"
	Severity      string // "critical", "high", "medium", "low"
	Description   string
	DetectedAt    time.Time
}

// AuthorizationDecisionRequest for PDP
type AuthorizationDecisionRequest struct {
	ClientID           string
	ResourceID         string
	ActionType         string
	ActionDetails      map[string]interface{}
	PowerOfAttorney    *poa.PoADefinition
	AuthorizationChain *AuthorizationChain
	Context            map[string]interface{}
}

// AuthorizationDecision from PDP
type AuthorizationDecision struct {
	DecisionID     string
	Authorized     bool
	Reason         string
	Conditions     []string
	ValidUntil     time.Time
	RequiresReview bool
}

// EnforceAuthorization is the main enforcement method (Supply-side PEP)
// This enforces authorization before the client performs an action
func (pep *PowerEnforcementPoint) EnforceAuthorization(
	ctx context.Context,
	request *EnforcementRequest,
) (*EnforcementResult, error) {

	enforcementID := generateEnforcementID()
	startTime := time.Now()

	result := &EnforcementResult{
		Allowed:       false,
		Violations:    []EnforcementViolation{},
		EnforcementID: enforcementID,
		EnforcedAt:    startTime,
	}

	// Step 1: Validate extended token
	extendedToken, err := pep.tokenValidator.ValidateExtendedToken(ctx, request.ExtendedToken)
	if err != nil {
		result.TokenValid = false
		result.DenyReason = fmt.Sprintf("Token validation failed: %v", err)
		pep.logEnforcement(ctx, request, result, "token_invalid")
		return result, nil
	}
	result.TokenValid = true

	// Step 2: Check token expiration
	expiryTime := extendedToken.IssuedAt.Add(time.Duration(extendedToken.ExpiresIn) * time.Second)
	if time.Now().After(expiryTime) {
		result.TokenExpired = true
		result.DenyReason = "Token has expired"
		result.Violations = append(result.Violations, EnforcementViolation{
			ViolationType: "temporal",
			Severity:      "critical",
			Description:   "Extended token has expired",
			DetectedAt:    time.Now(),
		})
		pep.logViolation(ctx, request, result, "token_expired")
		return result, nil
	}

	// Step 3: Validate scope
	scopeValid, scopeViolations := pep.validateScope(request, extendedToken)
	result.ScopeValid = scopeValid
	if !scopeValid {
		result.Violations = append(result.Violations, scopeViolations...)
		result.DenyReason = "Action outside authorized scope"
		pep.logViolation(ctx, request, result, "scope_violation")

		if pep.enforcementMode == "strict" {
			return result, nil
		}
	}

	// Step 4: Check power restrictions
	restrictionsValid, restrictionViolations := pep.validateRestrictions(request, extendedToken)
	result.RestrictionsValid = restrictionsValid
	if !restrictionsValid {
		result.Violations = append(result.Violations, restrictionViolations...)
		result.DenyReason = "Action violates power restrictions"
		pep.logViolation(ctx, request, result, "restriction_violation")

		if pep.enforcementMode == "strict" {
			return result, nil
		}
	}

	// Step 5: Check compliance status
	if pep.complianceTracker != nil {
		complianceStatus, err := pep.complianceTracker.CheckCompliance(ctx, extendedToken.AccessToken)
		if err == nil && complianceStatus != nil {
			result.ComplianceValid = complianceStatus.Compliant
			if !complianceStatus.Compliant {
				for _, violation := range complianceStatus.Violations {
					result.Violations = append(result.Violations, EnforcementViolation{
						ViolationType: "compliance",
						Severity:      "high",
						Description:   violation,
						DetectedAt:    time.Now(),
					})
				}
				result.DenyReason = "Compliance violations detected"
				pep.logViolation(ctx, request, result, "compliance_violation")

				if pep.enforcementMode == "strict" {
					return result, nil
				}
			}
		}
	}

	// Step 6: Get PDP decision
	if pep.pdp != nil {
		pdpRequest := &AuthorizationDecisionRequest{
			ClientID:           extendedToken.ClientOwner.OwnerID,
			ResourceID:         request.ResourceID,
			ActionType:         request.ActionType,
			ActionDetails:      request.Context,
			PowerOfAttorney:    extendedToken.PowerOfAttorney,
			AuthorizationChain: extendedToken.AuthorizationChain,
			Context:            request.Context,
		}

		decision, err := pep.pdp.MakeDecision(ctx, pdpRequest)
		if err != nil {
			result.DenyReason = fmt.Sprintf("PDP decision failed: %v", err)
			pep.logEnforcement(ctx, request, result, "pdp_error")
			return result, nil
		}

		result.Decision = decision
		if !decision.Authorized {
			result.DenyReason = decision.Reason
			result.Violations = append(result.Violations, EnforcementViolation{
				ViolationType: "authorization",
				Severity:      "critical",
				Description:   fmt.Sprintf("PDP denied: %s", decision.Reason),
				DetectedAt:    time.Now(),
			})
			pep.logViolation(ctx, request, result, "pdp_denied")
			return result, nil
		}
	}

	// All checks passed - ALLOW
	result.Allowed = true
	result.AllowReason = "All authorization checks passed"
	pep.logEnforcement(ctx, request, result, "allowed")

	return result, nil
}

// ValidateDemandSide performs demand-side validation (Resource Server PEP)
// This validates that the client's action is authorized by the resource owner
func (pep *PowerEnforcementPoint) ValidateDemandSide(
	ctx context.Context,
	request *EnforcementRequest,
) (*EnforcementResult, error) {

	// Demand-side validation focuses on resource owner authorization
	// This is similar to EnforceAuthorization but from the resource server perspective
	result, err := pep.EnforceAuthorization(ctx, request)
	if err != nil {
		return nil, err
	}

	// Additional demand-side checks would go here
	// e.g., checking resource owner specific policies

	return result, nil
}

// validateScope checks if the action is within the authorized scope
func (pep *PowerEnforcementPoint) validateScope(
	request *EnforcementRequest,
	token *ExtendedToken,
) (bool, []EnforcementViolation) {

	violations := []EnforcementViolation{}

	if token.PowerOfAttorney == nil {
		violations = append(violations, EnforcementViolation{
			ViolationType: "scope",
			Severity:      "critical",
			Description:   "No power of attorney defined in token",
			DetectedAt:    time.Now(),
		})
		return false, violations
	}

	authorizedActions := token.PowerOfAttorney.Authorization.AuthorizedActions

	// Check based on action type
	switch request.ActionType {
	case actionTypeTransaction:
		// Check if transaction type is authorized
		found := false
		for _, tx := range authorizedActions.Transactions {
			if tx == request.TransactionType {
				found = true
				break
			}
		}
		if !found {
			violations = append(violations, EnforcementViolation{
				ViolationType: "scope",
				Severity:      "critical",
				Description:   fmt.Sprintf("Transaction type %s not authorized", request.TransactionType),
				DetectedAt:    time.Now(),
			})
			return false, violations
		}

	case actionTypeDecision:
		// Check if decision type is authorized
		found := false
		for _, dec := range authorizedActions.Decisions {
			if dec == request.DecisionType {
				found = true
				break
			}
		}
		if !found {
			violations = append(violations, EnforcementViolation{
				ViolationType: "scope",
				Severity:      "high",
				Description:   fmt.Sprintf("Decision type %s not authorized", request.DecisionType),
				DetectedAt:    time.Now(),
			})
			return false, violations
		}

	case actionTypeAction:
		// Check if action is authorized
		// Physical actions require special authorization
		if request.PhysicalAction {
			if len(authorizedActions.PhysicalActions) == 0 {
				violations = append(violations, EnforcementViolation{
					ViolationType: "scope",
					Severity:      "critical",
					Description:   "Physical actions not authorized",
					DetectedAt:    time.Now(),
				})
				return false, violations
			}
		}
	}

	return true, violations
}

// validateRestrictions checks power restrictions
func (pep *PowerEnforcementPoint) validateRestrictions(
	request *EnforcementRequest,
	token *ExtendedToken,
) (bool, []EnforcementViolation) {

	violations := []EnforcementViolation{}

	// Check token-level restrictions
	for _, restriction := range token.Restrictions {
		switch restriction.RestrictionType {
		case "value_limit":
			if request.ActionType == "transaction" && restriction.Value != nil {
				if limitValue, ok := restriction.Value.(float64); ok {
					if request.TransactionAmount > limitValue {
						violations = append(violations, EnforcementViolation{
							ViolationType: "restriction",
							Severity:      "critical",
							Description:   fmt.Sprintf("Transaction amount %.2f exceeds limit %.2f", request.TransactionAmount, limitValue),
							DetectedAt:    time.Now(),
						})
					}
				}
			}

		case "time_limit":
			// Temporal restrictions checked in main enforcement

		case "geographic_limit":
			if request.PhysicalAction && request.ActionLocation != "" {
				// Would validate location against allowed regions
				// Implementation depends on how geographic restrictions are structured
			}

		case "scope_limit":
			// Scope limitations are checked separately
		}
	}

	if len(violations) > 0 {
		return false, violations
	}

	return true, violations
}

// logEnforcement logs successful enforcement
func (pep *PowerEnforcementPoint) logEnforcement(
	ctx context.Context,
	request *EnforcementRequest,
	result *EnforcementResult,
	outcome string,
) {
	if pep.auditLogger == nil {
		return
	}

	entry := &EnforcementAuditEntry{
		EnforcementID:  result.EnforcementID,
		Timestamp:      time.Now(),
		ActionType:     request.ActionType,
		ResourceID:     request.ResourceID,
		Outcome:        outcome,
		Allowed:        result.Allowed,
		Reason:         result.AllowReason,
		ViolationCount: len(result.Violations),
	}

	_ = pep.auditLogger.LogEnforcement(ctx, entry)
}

// logViolation logs enforcement violations
func (pep *PowerEnforcementPoint) logViolation(
	ctx context.Context,
	request *EnforcementRequest,
	result *EnforcementResult,
	violationType string,
) {
	if pep.auditLogger == nil {
		return
	}

	for _, violation := range result.Violations {
		entry := &ViolationAuditEntry{
			EnforcementID: result.EnforcementID,
			Timestamp:     time.Now(),
			ViolationType: violation.ViolationType,
			Severity:      violation.Severity,
			Description:   violation.Description,
			ActionType:    request.ActionType,
			ResourceID:    request.ResourceID,
		}

		_ = pep.auditLogger.LogViolation(ctx, entry)
	}
}

// generateEnforcementID creates a unique enforcement identifier
func generateEnforcementID() string {
	return fmt.Sprintf("enf_%d", time.Now().UnixNano())
}

// EnforcementAuditEntry for audit logging
type EnforcementAuditEntry struct {
	EnforcementID  string
	Timestamp      time.Time
	ActionType     string
	ResourceID     string
	Outcome        string
	Allowed        bool
	Reason         string
	ViolationCount int
}

// ViolationAuditEntry for violation logging
type ViolationAuditEntry struct {
	EnforcementID string
	Timestamp     time.Time
	ViolationType string
	Severity      string
	Description   string
	ActionType    string
	ResourceID    string
}

// MemoryPEPAuditLogger is a simple in-memory implementation
type MemoryPEPAuditLogger struct {
	enforcements []EnforcementAuditEntry
	violations   []ViolationAuditEntry
}

// NewMemoryPEPAuditLogger creates a new in-memory audit logger
func NewMemoryPEPAuditLogger() *MemoryPEPAuditLogger {
	return &MemoryPEPAuditLogger{
		enforcements: make([]EnforcementAuditEntry, 0),
		violations:   make([]ViolationAuditEntry, 0),
	}
}

// LogEnforcement logs an enforcement action
func (l *MemoryPEPAuditLogger) LogEnforcement(ctx context.Context, entry *EnforcementAuditEntry) error {
	l.enforcements = append(l.enforcements, *entry)
	return nil
}

// LogViolation logs a violation
func (l *MemoryPEPAuditLogger) LogViolation(ctx context.Context, entry *ViolationAuditEntry) error {
	l.violations = append(l.violations, *entry)
	return nil
}

// GetEnforcements returns all enforcement logs
func (l *MemoryPEPAuditLogger) GetEnforcements() []EnforcementAuditEntry {
	return l.enforcements
}

// GetViolations returns all violation logs
func (l *MemoryPEPAuditLogger) GetViolations() []ViolationAuditEntry {
	return l.violations
}

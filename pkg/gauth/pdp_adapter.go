// Package gauth - Power Decision Point (PDP) Adapter
// This adapter connects the gauth.PowerDecisionPoint interface to policy-based decisions
package gauth

import (
	"context"
	"fmt"
	"time"

	"github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/poa"
)

// noopPEPAuditLogger is a simple no-op audit logger for PEP
type noopPEPAuditLogger struct{}

func (n *noopPEPAuditLogger) LogEnforcement(ctx context.Context, entry *EnforcementAuditEntry) error {
	// TODO: Implement audit logging to observability system
	return nil
}

func (n *noopPEPAuditLogger) LogViolation(ctx context.Context, entry *ViolationAuditEntry) error {
	// TODO: Implement violation logging to observability system
	return nil
}

// simpleTokenValidator adapts ExtendedTokenService to TokenValidator interface
type simpleTokenValidator struct {
	extTokenService *ExtendedTokenService
}

func (v *simpleTokenValidator) ValidateExtendedToken(ctx context.Context, token string) (*ExtendedToken, error) {
	result, err := v.extTokenService.ValidateExtendedToken(ctx, token)
	if err != nil {
		return nil, err
	}
	if !result.Valid {
		return nil, fmt.Errorf("token validation failed")
	}
	return result.ExtendedToken, nil
}

// SimplePDP is a basic PDP implementation for RFC-0111 compliance
// It provides policy-based authorization decisions
type SimplePDP struct {
	// Future: could add policy storage/engine here
}

// NewSimplePDP creates a new SimplePDP instance
func NewSimplePDP() *SimplePDP {
	return &SimplePDP{}
}

// MakeDecision implements the PowerDecisionPoint interface
func (pdp *SimplePDP) MakeDecision(
	ctx context.Context,
	request *AuthorizationDecisionRequest,
) (*AuthorizationDecision, error) {
	if request == nil {
		return &AuthorizationDecision{
			Authorized: false,
			Reason:     "nil request",
		}, nil
	}

	// Extract authorization scope from PoA
	authorized, reason := pdp.evaluateRequest(request)

	return &AuthorizationDecision{
		DecisionID:     fmt.Sprintf("pdp-decision-%d", time.Now().UnixNano()),
		Authorized:     authorized,
		Reason:         reason,
		Conditions:     []string{},
		ValidUntil:     time.Now().Add(5 * time.Minute), // Decision valid for 5 minutes
		RequiresReview: false,
	}, nil
}

// evaluateRequest performs the actual authorization logic
func (pdp *SimplePDP) evaluateRequest(request *AuthorizationDecisionRequest) (bool, string) {
	// Step 1: Validate Power of Attorney exists
	if request.PowerOfAttorney == nil {
		return false, "missing power of attorney credential"
	}

	// Step 2: Validate Authorization Chain
	if request.AuthorizationChain == nil {
		return false, "missing authorization chain"
	}

	if !request.AuthorizationChain.ChainValidated {
		return false, "authorization chain not validated"
	}

	// Step 3: Check action type against authorized scope
	if !pdp.isActionAuthorized(request.ActionType, request.PowerOfAttorney) {
		return false, fmt.Sprintf("action type '%s' not authorized in PoA", request.ActionType)
	}

	// Step 4: Check resource access
	if request.ResourceID != "" && !pdp.isResourceAuthorized(request.ResourceID, request.PowerOfAttorney) {
		return false, fmt.Sprintf("resource '%s' not authorized in PoA scope", request.ResourceID)
	}

	// All checks passed
	return true, "authorization granted per PoA and chain validation"
}

// isActionAuthorized checks if the action type is allowed in the PoA
func (pdp *SimplePDP) isActionAuthorized(actionType string, poaDef *poa.PoADefinition) bool {
	if poaDef == nil {
		// If no specific restrictions, allow (default permissive for demo)
		return true
	}

	authActions := poaDef.Authorization.AuthorizedActions

	// Check transaction types
	if actionType == "transaction" {
		return len(authActions.Transactions) > 0
	}

	// Check decision types
	if actionType == "decision" {
		return len(authActions.Decisions) > 0
	}

	// Check action types
	if actionType == "action" {
		physicalCount := len(authActions.PhysicalActions)
		nonPhysicalCount := len(authActions.NonPhysicalActions)
		return physicalCount > 0 || nonPhysicalCount > 0
	}

	// Unknown action type - deny by default
	return false
}

// isResourceAuthorized checks if the resource is within PoA scope
func (pdp *SimplePDP) isResourceAuthorized(resourceID string, poaDef *poa.PoADefinition) bool {
	// For now, use simple logic:
	// If geographic scope includes global, allow all resources
	if len(poaDef.Authorization.ApplicableRegions) > 0 {
		for _, region := range poaDef.Authorization.ApplicableRegions {
			if region.Type == poa.GeoTypeGlobal {
				return true
			}
		}
	}

	// If sectors defined, assume resource is authorized
	// (In production, this would check resource sector against authorized sectors)
	if len(poaDef.Authorization.ApplicableSectors) > 0 {
		return true
	}

	// Default: allow if no specific restrictions
	return true
}

// AddPolicy adds a policy to the PDP (future enhancement)
func (pdp *SimplePDP) AddPolicy(policyID string, policy interface{}) error {
	// TODO: Implement policy storage when policy engine is added
	return fmt.Errorf("policy management not yet implemented")
}

// RemovePolicy removes a policy from the PDP (future enhancement)
func (pdp *SimplePDP) RemovePolicy(policyID string) error {
	// TODO: Implement policy removal when policy engine is added
	return fmt.Errorf("policy management not yet implemented")
}

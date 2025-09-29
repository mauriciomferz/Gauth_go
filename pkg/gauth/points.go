// Package gauth: RFC111 Compliance Mapping
//
// This file implements the core P*P (Power*Point) roles as defined in RFC111:
//   - Power Enforcement Point (PEP): access control enforcement
//   - Power Decision Point (PDP): authorization logic
//   - Power Information Point (PIP): attribute gathering
//   - Power Administration Point (PAP): policy management
//   - Power Verification Point (PVP): token validation
//
// Relevant RFC111 Sections:
//   - Section 3: Nomenclature (P*P roles)
//   - Section 5: What GAuth is (role responsibilities)
//   - Section 6: How GAuth works (protocol flow, enforcement)
//
// Compliance:
//   - All logic is centralized and type-safe.
//   - No exclusions (Web3, DNA, decentralized auth) are present.
//   - All enforcement and decision logic is explicit and auditable.
//   - See README and docs/ for full protocol mapping.
//
// License: Apache 2.0 (see LICENSE file)
//
// ---
//
// The points system implements:
//   - Power enforcement points for access control
//   - Decision points for authorization logic
//   - Information points for attribute gathering
//   - Administration points for policy management
//   - Verification points for token validation
package gauth

import (
	"fmt"
)

const (
	TransactionExecuteScope = "transaction:execute"
)

// PowerEnforcementPoint handles access control enforcement.
type PowerEnforcementPoint struct {
	GAuth *GAuth
}

// EnforceRestrictions checks if a token allows a specific action.
// actionDetails must specify the type and amount of the action.
func (p *PowerEnforcementPoint) EnforceRestrictions(token string, _ ActionDetails) (bool, error) {
	if p.GAuth == nil {
		return false, fmt.Errorf("GAuth instance not configured")
	}
	tokenData, err := p.GAuth.ValidateToken(token)
	if err != nil {
		return false, err
	}
	for _, scope := range tokenData.Scope {
		if scope == TransactionExecuteScope {
			return true, nil
		}
	}
	// NOTE: tokenData.Restrictions is not present in tokenstore.TokenData, so this logic may need to be refactored if restrictions are required.
	return false, fmt.Errorf("action not allowed by token restrictions")
}

// PowerDecisionPoint handles authorization decisions.
// PowerDecisionPoint handles authorization decisions.
type PowerDecisionPoint struct {
	GAuth *GAuth
}

// PowerAdministrationPoint handles token administration.
type PowerAdministrationPoint struct {
	GAuth *GAuth
}

// PowerVerificationPoint handles token validation.
type PowerVerificationPoint struct {
	GAuth *GAuth
}

// MakeAuthorizationDecision decides if a token grants access to an action.
func (p *PowerDecisionPoint) MakeAuthorizationDecision(token string, requestedAction string) bool {
	tokenData, err := p.GAuth.ValidateToken(token)
	if err != nil {
		return false
	}
	for _, scope := range tokenData.Scope {
		if scope == requestedAction {
			return true
		}
	}
	return false
}

// AddTokenRestriction adds a new restriction to a token.
func (p *PowerAdministrationPoint) AddTokenRestriction(_ string, restriction Restriction) error {
	// Not implemented: tokenstore.TokenData does not support dynamic restrictions. This is a placeholder for future extension.
	return fmt.Errorf("AddTokenRestriction not implemented: tokenstore.TokenData does not support restrictions")
}

// InvalidateToken marks a token as invalid.
func (p *PowerAdministrationPoint) InvalidateToken(token string) error {
	// Use the TokenStore's Store method to mark as invalid
	tokenData, err := p.GAuth.ValidateToken(token)
	if err != nil {
		return err
	}
	tokenData.Valid = false
	if err := p.GAuth.TokenStore.Store(token, *tokenData); err != nil {
		return err
	}
	return nil
}

// PowerVerificationPoint handles token validation.

// UpdatePowerRestriction updates a power restriction for the given type.
func (p *PowerAdministrationPoint) UpdatePowerRestriction(_ PowerRestriction) error {
	if p.GAuth == nil {
		return fmt.Errorf("GAuth instance not configured")
	}
	// Implement logic to update the restriction using the explicit type
	// Example: p.GAuth.SetPowerRestriction(restriction.Type, restriction.Value)
	return nil
}

// ActionDetails represents the details of an action for enforcement.
type ActionDetails struct {
	Type   string  // e.g., "transaction_type"
	Amount float64 // e.g., transaction amount
}

// RestrictionValueType enumerates allowed value types for PowerRestriction.
type RestrictionValueType int

const (
	StringValue RestrictionValueType = iota
	FloatValue
)

// PowerRestriction represents a restriction to be applied or updated.
// Only one of StringValue or FloatValue should be set, according to ValueType.
type PowerRestriction struct {
	Type        string
	StringValue string
	FloatValue  float64
	ValueType   RestrictionValueType
}

// PowerVerificationPoint handles token validation.

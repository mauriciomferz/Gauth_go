// Package gauth provides embedded points for the GAuth protocol.
// The points system implements:
//   - Power enforcement points for access control
//   - Decision points for authorization logic
//   - Information points for attribute gathering
//   - Administration points for policy management
//   - Verification points for token validation
package gauth

import (
	"fmt"
	"time"

	"github.com/Gimel-Foundation/gauth/pkg/gauth"
)

// PowerEnforcementPoint handles access control enforcement.
type PowerEnforcementPoint struct {
	GAuth *GAuth
}

// EnforceRestrictions checks if a token allows a specific action.
// actionDetails must specify the type and amount of the action.
func (p *PowerEnforcementPoint) EnforceRestrictions(token string, actionDetails ActionDetails) (bool, error) {
	if p.GAuth == nil {
		return false, fmt.Errorf("GAuth instance not configured")
	}
	tokenResp, exists := p.GAuth.GetToken(token)
	if !exists {
		return false, fmt.Errorf("invalid token")
	}
	if time.Now().After(tokenResp.ValidUntil) {
		return false, fmt.Errorf("token expired")
	}
	for _, scope := range tokenResp.Scope {
		if scope == "transaction:execute" {
			return true, nil
		}
	}
	for _, restriction := range tokenResp.Restrictions {
		switch restriction.Type {
		case "transaction_type":
			if restriction.Value == actionDetails.Type {
				return true, nil
			}
		case "amount_limit":
			if maxAmount, ok := restriction.Value.(float64); ok {
				if actionDetails.Amount <= maxAmount {
					return true, nil
				}
			}
		}
	}
	return false, fmt.Errorf("action not allowed by token restrictions")
}

// PowerDecisionPoint handles authorization decisions.
// PowerDecisionPoint handles authorization decisions.
type PowerDecisionPoint struct {
	GAuth *gauth.Service
}

// PowerAdministrationPoint handles token administration.
type PowerAdministrationPoint struct {
	GAuth *gauth.Service
}

// PowerVerificationPoint handles token validation.
type PowerVerificationPoint struct {
	GAuth *gauth.Service
}

// MakeAuthorizationDecision decides if a token grants access to an action.
func (p *PowerDecisionPoint) MakeAuthorizationDecision(token string, requestedAction string) bool {
	tokenResp, exists := p.GAuth.GetToken(token)
	if !exists {
		return false
	}
	if time.Now().After(tokenResp.ValidUntil) {
		return false
	}
	for _, scope := range tokenResp.Scope {
		if scope == requestedAction {
			return true
		}
	}
	return false
}

// AddTokenRestriction adds a new restriction to a token.
func (p *PowerAdministrationPoint) AddTokenRestriction(token string, restriction gauth.Restriction) error {
	tokenResp, exists := p.GAuth.GetToken(token)
	if !exists {
		return fmt.Errorf("token not found")
	}
	tokenResp.Restrictions = append(tokenResp.Restrictions, restriction)
	p.GAuth.StoreToken(tokenResp)
	return nil
}

// InvalidateToken marks a token as invalid.
func (p *PowerAdministrationPoint) InvalidateToken(token string) error {
	p.GAuth.TokenStoreMutex.Lock()
	defer p.GAuth.TokenStoreMutex.Unlock()
	if td, exists := p.GAuth.TokenStore[token]; exists {
		td.Valid = false
		return nil
	}
	return fmt.Errorf("token not found")
}

// PowerVerificationPoint handles token validation.

// UpdatePowerRestriction updates a power restriction for the given type.
func (p *PowerAdministrationPoint) UpdatePowerRestriction(restriction PowerRestriction) error {
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

// Package enforcement - PEP (Power Enforcement Point) Implementation
// This file implements the AAP001 PEP architecture with supply-side and demand-side enforcement.
package enforcement

import (
	"context"
	"fmt"
)

// PEPSide defines which side of the PEP architecture is being enforced
type PEPSide string

const (
	// PEPSupplySide - Client-side enforcement: AI client enforces compliance with its own authorization
	// RFC Requirement: "The client itself Must make sure it decides and acts in line with its authorization,
	// thus enforces compliance from the supply-side."
	PEPSupplySide PEPSide = "supply-side"

	// PEPDemandSide - Resource server-side enforcement: Resource owner/server validates client compliance
	// RFC Requirement: "The resource owner and/or resource server Must check authorization compliance
	// of the transactions, actions and decisions of the client and its owner as demand-side."
	PEPDemandSide PEPSide = "demand-side"
)

// PDPClient defines the interface for integrating with a Policy Decision Point
// RFC: PEP asks the PDP for a decision and enforces its result
type PDPClient interface {
	// Decide asks the PDP for an authorization decision
	Decide(ctx context.Context, req *EnforcementRequest) (*PDPDecision, error)
}

// PDPDecision represents a decision from the Policy Decision Point
type PDPDecision struct {
	Decision   string                 `json:"decision"` // "permit", "deny", "indeterminate"
	Policies   []string               `json:"policies"` // Applied policies
	Attributes map[string]interface{} `json:"attributes"`
	Reason     string                 `json:"reason"`
}

// SupplySidePEP implements supply-side (client) enforcement
// The client MUST ensure it acts in line with its authorization
type SupplySidePEP struct {
	*Enforcer
	pdpClient PDPClient
	clientID  string
}

// NewSupplySidePEP creates a PEP configured for supply-side (client) enforcement
func NewSupplySidePEP(clientID string, pdp PDPClient) *SupplySidePEP {
	return &SupplySidePEP{
		Enforcer:  NewEnforcer(),
		pdpClient: pdp,
		clientID:  clientID,
	}
}

// EnforceClientAction enforces client-side authorization compliance before performing action
// This is called by the AI client before it performs any transaction, decision, or action
func (s *SupplySidePEP) EnforceClientAction(ctx context.Context, resource, action string, context map[string]interface{}) error {
	req := &EnforcementRequest{
		Subject:  s.clientID,
		Resource: resource,
		Action:   action,
		Context:  context,
	}

	// Step 1: Ask PDP for decision (if configured)
	if s.pdpClient != nil {
		pdpDecision, err := s.pdpClient.Decide(ctx, req)
		if err != nil {
			return fmt.Errorf("supply-side PEP: PDP decision failed: %w", err)
		}

		if pdpDecision.Decision == DecisionDeny {
			return fmt.Errorf("supply-side PEP: action denied by PDP: %s", pdpDecision.Reason)
		}
	}

	// Step 2: Evaluate local enforcement rules
	decision, err := s.Evaluate(ctx, req)
	if err != nil {
		return fmt.Errorf("supply-side PEP: enforcement evaluation failed: %w", err)
	}

	if decision.Decision == DecisionDeny {
		return fmt.Errorf("supply-side PEP: action denied: %s", decision.Reason)
	} // Client is authorized to proceed
	return nil
}

// DemandSidePEP implements demand-side (resource server) enforcement
// The resource owner/server MUST validate client authorization compliance
type DemandSidePEP struct {
	*Enforcer
	pdpClient PDPClient
	serverID  string
	ownerID   string
}

// NewDemandSidePEP creates a PEP configured for demand-side (resource server) enforcement
func NewDemandSidePEP(serverID, ownerID string, pdp PDPClient) *DemandSidePEP {
	return &DemandSidePEP{
		Enforcer:  NewEnforcer(),
		pdpClient: pdp,
		serverID:  serverID,
		ownerID:   ownerID,
	}
}

// ValidateClientCompliance validates client's authorization compliance from resource server side.
// This is called by the resource server/owner to check if the client's transaction/action/decision is
// authorized.
func (d *DemandSidePEP) ValidateClientCompliance(
	ctx context.Context,
	clientID string,
	resource string,
	action string,
	clientToken string,
	context map[string]interface{},
) error {
	if context == nil {
		context = make(map[string]interface{})
	}
	context["client_token"] = clientToken
	context["server_id"] = d.serverID
	context["owner_id"] = d.ownerID

	req := &EnforcementRequest{
		Subject:  clientID,
		Resource: resource,
		Action:   action,
		Context:  context,
	}

	// Step 1: Ask PDP for decision (if configured)
	if d.pdpClient != nil {
		pdpDecision, err := d.pdpClient.Decide(ctx, req)
		if err != nil {
			return fmt.Errorf("demand-side PEP: PDP decision failed: %w", err)
		}

		if pdpDecision.Decision == DecisionDeny {
			return fmt.Errorf("demand-side PEP: client action denied by PDP: %s", pdpDecision.Reason)
		}
	}

	// Step 2: Evaluate local enforcement rules
	decision, err := d.Evaluate(ctx, req)
	if err != nil {
		return fmt.Errorf("demand-side PEP: enforcement evaluation failed: %w", err)
	}

	if decision.Decision == DecisionDeny {
		return fmt.Errorf("demand-side PEP: client action denied: %s", decision.Reason)
	}

	// Client transaction/action/decision is authorized
	return nil
}

// EnforcementMetadata adds PEP-specific metadata to enforcement decisions
type EnforcementMetadata struct {
	PEPSide    PEPSide `json:"pep_side"`    // Which side is enforcing
	PDPQueried bool    `json:"pdp_queried"` // Whether PDP was consulted
	ClientID   string  `json:"client_id"`   // Client being enforced/validated
	ServerID   string  `json:"server_id"`   // Server enforcing (demand-side only)
	OwnerID    string  `json:"owner_id"`    // Resource owner (demand-side only)
}

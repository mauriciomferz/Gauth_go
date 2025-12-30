// Package agentauth - PDP Bridge Implementation
// Bridges the pkg/pdp Engine to the PDPClient interface used by ComplianceValidator
package agentauth

import (
	"context"
	"fmt"
	"time"

	"github.com/mauriciomferz/AgentAuth/pkg/pdp"
)

// PDPBridge implements PDPClient interface by wrapping pkg/pdp.Engine
// This bridges the existing PDP engine with the AAP-001 compliance validation
type PDPBridge struct {
	engine pdp.Engine
}

// NewPDPBridge creates a new PDP bridge with the given engine
func NewPDPBridge(engine pdp.Engine) *PDPBridge {
	return &PDPBridge{
		engine: engine,
	}
}

// EvaluatePolicy evaluates a policy decision request
// Implements PDPClient interface for AAP-001 compliance validation
func (b *PDPBridge) EvaluatePolicy(ctx context.Context, request interface{}) (bool, error) {
	// Extract policy request details from the interface{}
	pdpRequest, err := b.convertRequest(request)
	if err != nil {
		return false, fmt.Errorf("failed to convert policy request: %w", err)
	}

	// Evaluate using the PDP engine
	decision, err := b.engine.Evaluate(ctx, pdpRequest)
	if err != nil {
		return false, fmt.Errorf("PDP evaluation failed: %w", err)
	}

	// Return the decision result
	return decision.Allow, nil
}

// convertRequest converts various request types to pdp.Request
func (b *PDPBridge) convertRequest(request interface{}) (pdp.Request, error) {
	// Handle different request types
	switch req := request.(type) {
	case *ExtendedTokenRequest:
		return b.convertTokenRequest(req), nil
	case *ExtendedAuthorizationRequest:
		return b.convertAuthRequest(req), nil
	case *ExtendedAuthorizationGrant:
		return b.convertGrantRequest(req), nil
	case map[string]interface{}:
		return b.convertMapRequest(req), nil
	case pdp.Request:
		return req, nil
	default:
		return pdp.Request{}, fmt.Errorf("unsupported request type: %T", request)
	}
}

// convertTokenRequest converts ExtendedTokenRequest to pdp.Request
func (b *PDPBridge) convertTokenRequest(req *ExtendedTokenRequest) pdp.Request {
	attrs := make(map[string]string)

	// Add basic attributes
	if req.GrantID != "" {
		attrs["grant_id"] = req.GrantID
	}
	if len(req.Scope) > 0 {
		attrs["scope"] = fmt.Sprintf("%v", req.Scope)
	}
	if req.JurisdictionCode != "" {
		attrs["jurisdiction"] = req.JurisdictionCode
	}

	// Add entity information
	if req.ClientOwnerInfo != nil {
		attrs["client_owner_id"] = req.ClientOwnerInfo.OwnerID
		attrs["client_owner_type"] = req.ClientOwnerInfo.OwnerType
	}
	if req.ResourceOwnerInfo != nil {
		attrs["resource_owner_id"] = req.ResourceOwnerInfo.OwnerID
	}

	// Add legal framework
	if req.LegalFramework != nil && len(req.LegalFramework.ApplicableLaws) > 0 {
		attrs["legal_framework"] = req.LegalFramework.ApplicableLaws[0]
	}

	// Determine subject and resource
	subject := "unknown"
	if req.ClientOwnerInfo != nil {
		subject = req.ClientOwnerInfo.OwnerID
	}

	resource := "unknown"
	if req.ResourceOwnerInfo != nil {
		resource = req.ResourceOwnerInfo.OwnerID
	}

	// Determine action from requested actions
	action := "access"
	if len(req.RequestedActions) > 0 {
		action = req.RequestedActions[0]
	}

	return pdp.Request{
		Subject:    subject,
		Action:     action,
		Resource:   resource,
		Attributes: attrs,
		Time:       time.Now(),
	}
}

// convertAuthRequest converts ExtendedAuthorizationRequest to pdp.Request
func (b *PDPBridge) convertAuthRequest(req *ExtendedAuthorizationRequest) pdp.Request {
	attrs := make(map[string]string)

	// Add client information
	if req.AuthorizationRequest != nil {
		attrs["client_id"] = req.ClientID
	}

	// Add legal framework
	if req.LegalFramework != nil && len(req.LegalFramework.ApplicableLaws) > 0 {
		attrs["legal_framework"] = req.LegalFramework.ApplicableLaws[0]
	}

	// Add restrictions
	if len(req.Restrictions) > 0 {
		attrs["restrictions_count"] = fmt.Sprintf("%d", len(req.Restrictions))
	}

	// Add transaction context
	if len(req.TransactionContext) > 0 {
		attrs["has_transaction_context"] = "true"
	}

	// Determine subject, action, resource
	subject := req.ClientID
	action := "authorize"
	if len(req.RequestedActions) > 0 {
		action = req.RequestedActions[0]
	}
	resource := "authorization_grant"

	return pdp.Request{
		Subject:    subject,
		Action:     action,
		Resource:   resource,
		Attributes: attrs,
		Time:       req.RequestTime,
	}
}

// convertGrantRequest converts ExtendedAuthorizationGrant to pdp.Request
func (b *PDPBridge) convertGrantRequest(grant *ExtendedAuthorizationGrant) pdp.Request {
	attrs := make(map[string]string)

	// Add grant information
	if grant.AuthorizationGrant != nil {
		attrs["grant_id"] = grant.GrantID
		attrs["client_id"] = grant.ClientID
	}
	attrs["resource_owner_id"] = grant.ResourceOwnerID
	attrs["issuer_id"] = grant.IssuerID

	// Add legal framework
	if grant.LegalFramework != nil && len(grant.LegalFramework.ApplicableLaws) > 0 {
		attrs["legal_framework"] = grant.LegalFramework.ApplicableLaws[0]
	}

	// Add restrictions
	if len(grant.Restrictions) > 0 {
		attrs["restrictions_count"] = fmt.Sprintf("%d", len(grant.Restrictions))
	}

	return pdp.Request{
		Subject:    grant.ClientID,
		Action:     "use_grant",
		Resource:   grant.ResourceOwnerID,
		Attributes: attrs,
		Time:       grant.IssuedAt,
	}
}

// convertMapRequest converts map[string]interface{} to pdp.Request
func (b *PDPBridge) convertMapRequest(req map[string]interface{}) pdp.Request {
	attrs := make(map[string]string)

	// Extract common fields
	subject := getStringFromMap(req, "subject", "unknown")
	action := getStringFromMap(req, "action", "access")
	resource := getStringFromMap(req, "resource", "unknown")

	// Convert all other fields to attributes
	for k, v := range req {
		if k != "subject" && k != "action" && k != "resource" {
			attrs[k] = fmt.Sprintf("%v", v)
		}
	}

	return pdp.Request{
		Subject:    subject,
		Action:     action,
		Resource:   resource,
		Attributes: attrs,
		Time:       time.Now(),
	}
}

// getStringFromMap safely extracts string value from map
func getStringFromMap(m map[string]interface{}, key string, defaultVal string) string {
	if val, ok := m[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return defaultVal
}

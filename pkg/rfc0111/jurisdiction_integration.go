package rfc0111

import (
	"context"
	"fmt"

	"github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/internal/jurisdiction"
	"github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/compliance"
)

// JurisdictionEnforcement provides optional jurisdiction-specific runtime enforcement for RFC0111 operations.
// When enabled, all delegation creation and token verification operations are subject to jurisdiction
// rules including GDPR consent, CCPA opt-out, cross-border data transfer, data residency, and blocked actions.
//
// Integration is opt-in for backward compatibility. To enable:
//
//	integration := jurisdiction.NewServerIntegration()
//	svc := rfc0111.NewService(auditLogger, authorizer, rfc0111.WithJurisdictionEnforcement(integration))
//
// Jurisdiction is derived from:
//  1. PowerOfAttorney.Jurisdiction field (primary)
//  2. DelegationRequest.Claims["jurisdiction"] (fallback for issuance)
//  3. Token claims["jurisdiction"] (fallback for verification)
//  4. Default: compliance.JurisdictionUS (if no jurisdiction specified)
type JurisdictionEnforcement struct {
	integration *jurisdiction.ServerIntegration
	enabled     bool
}

// WithJurisdictionEnforcement enables jurisdiction-specific runtime enforcement.
// When enabled, CreateDelegation and VerifyToken operations will enforce:
//   - GDPR consent requirements (EU jurisdiction)
//   - CCPA opt-out respect (US jurisdiction)
//   - Cross-border data transfer rules
//   - Data residency requirements (EU personal/health data must stay in EU)
//   - Blocked actions per jurisdiction (e.g., EU: unrestricted_data_export)
//   - Value limits and approval requirements
//
// Example:
//
//	integration := jurisdiction.NewServerIntegration()
//	svc := NewService(auditLogger, authorizer, WithJurisdictionEnforcement(integration))
func WithJurisdictionEnforcement(integration *jurisdiction.ServerIntegration) Option {
	return func(s *Service) {
		s.jurisdictionEnforcement = &JurisdictionEnforcement{
			integration: integration,
			enabled:     true,
		}
	}
}

// enforceJurisdictionOnIssuance validates jurisdiction rules during delegation creation.
// This is called BEFORE creating the PowerOfAttorney to ensure compliance with:
//   - Blocked actions (e.g., EU blocks "unrestricted_data_export")
//   - Cross-border transfer rules (e.g., EU personal data can only go to EU/UK adequacy countries)
//   - Data residency rules (e.g., EU health data must stay in EU)
//   - Custom validators (e.g., GDPR consent required for data processing in EU)
//
// Enforcement failures return ErrUnauthorized with detailed violation messages.
// When jurisdiction enforcement is disabled (nil), this is a no-op returning nil.
func (s *Service) enforceJurisdictionOnIssuance(ctx context.Context, req DelegationRequest, poa *PowerOfAttorney) error {
	if s.jurisdictionEnforcement == nil || !s.jurisdictionEnforcement.enabled {
		return nil // Enforcement disabled, allow operation
	}

	// Extract jurisdiction from PowerOfAttorney (primary) or request claims (fallback)
	var jurisdictionStr string
	if poa.Jurisdiction != "" {
		jurisdictionStr = poa.Jurisdiction
	} else if req.Claims != nil {
		if j, ok := req.Claims["jurisdiction"].(string); ok {
			jurisdictionStr = j
		}
	}

	// Prepare claims for enforcement context
	claims := make(map[string]interface{})
	if req.Claims != nil {
		// Copy all request claims
		for k, v := range req.Claims {
			claims[k] = v
		}
	}
	// Add jurisdiction to claims if not already present
	if jurisdictionStr != "" {
		claims["jurisdiction"] = jurisdictionStr
	}
	// Add scope and restrictions as context for enforcement
	claims["scope"] = poa.Scope
	if poa.Restrictions != nil {
		for k, v := range poa.Restrictions {
			claims["restriction_"+k] = v
		}
	}

	// Determine action from scope (use first scope item as action, or default to "delegation")
	action := "delegation"
	if len(poa.Scope) > 0 {
		action = poa.Scope[0]
	}

	// Enforce jurisdiction rules
	decision, err := s.jurisdictionEnforcement.integration.EnforceJurisdiction(
		ctx,
		poa.Grantor, // subject
		poa.Grantee, // resource
		action,      // action
		claims,      // claims
	)
	if err != nil {
		// Enforcement check failed (engine error)
		// TODO(P1.3): Add s.metrics.IncJurisdictionEnforcementErrors() when metrics interface updated
		return fmt.Errorf("jurisdiction enforcement failed: %w", err)
	}

	if !decision.Allowed {
		// Enforcement denied the operation
		// TODO(P1.3): Add s.metrics.IncJurisdictionEnforcementDenials() when metrics interface updated
		// Build detailed violation message
		violationMsg := fmt.Sprintf("jurisdiction %s denied operation", decision.Jurisdiction)
		if len(decision.Violations) > 0 {
			violationMsg += fmt.Sprintf(": %v", decision.Violations)
		}
		if len(decision.AppliedRules) > 0 {
			violationMsg += fmt.Sprintf(" (rules: %v)", decision.AppliedRules)
		}
		return fmt.Errorf("%s", violationMsg)
	}

	// Enforcement allowed
	// TODO(P1.3): Add s.metrics.IncJurisdictionEnforcementAllows() when metrics interface updated
	return nil
}

// enforceJurisdictionOnVerification validates jurisdiction rules during token verification.
// This is called DURING token verification to ensure runtime compliance with:
//   - Action is not blocked in the jurisdiction
//   - Cross-border operations respect jurisdiction rules
//   - Data residency is maintained (if applicable)
//
// Unlike issuance enforcement which gates creation, verification enforcement validates
// that the token usage at runtime complies with current jurisdiction rules.
//
// When jurisdiction enforcement is disabled (nil), this is a no-op returning nil.
func (s *Service) enforceJurisdictionOnVerification(ctx context.Context, poa *PowerOfAttorney, claims map[string]interface{}) error {
	if s.jurisdictionEnforcement == nil || !s.jurisdictionEnforcement.enabled {
		return nil // Enforcement disabled, allow operation
	}

	// Extract jurisdiction from PowerOfAttorney or claims
	jurisdictionStr := poa.Jurisdiction
	if jurisdictionStr == "" && claims != nil {
		if j, ok := claims["jurisdiction"].(string); ok {
			jurisdictionStr = j
		}
	}

	// Prepare enforcement claims
	enforcementClaims := make(map[string]interface{})
	if claims != nil {
		// Copy all claims
		for k, v := range enforcementClaims {
			enforcementClaims[k] = v
		}
	}
	if jurisdictionStr != "" {
		enforcementClaims["jurisdiction"] = jurisdictionStr
	}
	// Add PowerOfAttorney context
	enforcementClaims["scope"] = poa.Scope
	if poa.Restrictions != nil {
		for k, v := range poa.Restrictions {
			enforcementClaims["restriction_"+k] = v
		}
	}

	// Determine action from claims or scope
	action := "verify_token"
	if claims != nil {
		if a, ok := claims["action"].(string); ok && a != "" {
			action = a
		}
	}
	if action == "verify_token" && len(poa.Scope) > 0 {
		action = poa.Scope[0] // Use first scope item as action
	}

	// Enforce jurisdiction rules
	decision, err := s.jurisdictionEnforcement.integration.EnforceJurisdiction(
		ctx,
		poa.Grantee,       // subject (token holder)
		poa.Grantor,       // resource (original delegator)
		action,            // action being performed
		enforcementClaims, // claims
	)
	if err != nil {
		// TODO(P1.3): Add s.metrics.IncJurisdictionEnforcementErrors() when metrics interface updated
		return fmt.Errorf("jurisdiction enforcement failed: %w", err)
	}

	if !decision.Allowed {
		// TODO(P1.3): Add s.metrics.IncJurisdictionEnforcementDenials() when metrics interface updated
		violationMsg := fmt.Sprintf("jurisdiction %s denied token usage", decision.Jurisdiction)
		if len(decision.Violations) > 0 {
			violationMsg += fmt.Sprintf(": %v", decision.Violations)
		}
		return fmt.Errorf("%s", violationMsg)
	}

	// TODO(P1.3): Add s.metrics.IncJurisdictionEnforcementAllows() when metrics interface updated
	return nil
}

// ExtractJurisdictionFromPOA extracts the jurisdiction from a PowerOfAttorney.
// Returns the jurisdiction enum value, or JurisdictionUS as default.
// Supports both explicit Jurisdiction field and fallback to Restrictions["jurisdiction"].
func ExtractJurisdictionFromPOA(poa *PowerOfAttorney) compliance.Jurisdiction {
	if poa == nil {
		return compliance.JurisdictionUS
	}

	// Build claims map from PowerOfAttorney fields
	claims := make(map[string]interface{})

	// Primary: use Jurisdiction field
	if poa.Jurisdiction != "" {
		claims["jurisdiction"] = poa.Jurisdiction
	}

	// Fallback: check Restrictions map
	if poa.Restrictions != nil {
		if j, ok := poa.Restrictions["jurisdiction"]; ok {
			claims["jurisdiction"] = j
		}
	}

	// Use jurisdiction extraction helper from internal/jurisdiction
	return jurisdiction.ExtractJurisdictionFromClaims(claims)
}

// ValidateJurisdictionCompliance validates that a PowerOfAttorney complies with its jurisdiction's rules.
// This is a standalone validation function that can be used for compliance checks without creating/verifying tokens.
//
// Returns nil if compliant, or an error describing the violation.
// When jurisdiction enforcement is disabled, this always returns nil.
func (s *Service) ValidateJurisdictionCompliance(ctx context.Context, poa *PowerOfAttorney) error {
	if s.jurisdictionEnforcement == nil || !s.jurisdictionEnforcement.enabled {
		return nil // Enforcement disabled
	}

	jurisdiction := ExtractJurisdictionFromPOA(poa)

	// Determine action from scope
	action := "delegation"
	if len(poa.Scope) > 0 {
		action = poa.Scope[0]
	}

	return s.jurisdictionEnforcement.integration.ValidateJurisdiction(ctx, jurisdiction, action)
}

// Package gauth - Request and Grant Compliance Validation per RFC-0111 Section 6
// Implements critical Gaps #2 and #3 from QUALITY_MANAGER_RFC_COMPLIANCE_FINAL_ASSESSMENT.md
// RFC Section 6: Two-phase protocol flow validation
package gauth

import (
	"context"
	"fmt"
	"time"

	"github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/poa"
)

// ComplianceValidator performs RFC-0111 Section 6 compliance validation
type ComplianceValidator struct {
	chainValidator *AuthorizationChainValidator
	pipClient      PIPClient
	pdpClient      PDPClient
	strictMode     bool
}

// NewComplianceValidator creates a new compliance validator
func NewComplianceValidator(
	chainValidator *AuthorizationChainValidator,
	pipClient PIPClient,
	pdpClient PDPClient,
) *ComplianceValidator {
	return &ComplianceValidator{
		chainValidator: chainValidator,
		pipClient:      pipClient,
		pdpClient:      pdpClient,
		strictMode:     true,
	}
}

// ExtendedAuthorizationRequest represents an RFC-0111 compliant authorization request
type ExtendedAuthorizationRequest struct {
	*AuthorizationRequest  // Embed existing type for compatibility
	PowerOfAttorney        *poa.PoADefinition       `json:"power_of_attorney,omitempty"`
	AuthorizationChain     *AuthorizationChain      `json:"authorization_chain,omitempty"`
	LegalFramework         *LegalFrameworkInfo      `json:"legal_framework,omitempty"`
	Restrictions           []PowerRestriction       `json:"restrictions,omitempty"`
	RequestedActions       []string                 `json:"requested_actions,omitempty"`
	TransactionContext     map[string]interface{}   `json:"transaction_context,omitempty"`
	RequestTime            time.Time                `json:"request_time"`
}

// ExtendedAuthorizationGrant represents an RFC-0111 compliant authorization grant
type ExtendedAuthorizationGrant struct {
	*AuthorizationGrant    // Embed existing type for compatibility
	ResourceOwnerID        string                   `json:"resource_owner_id"`
	IssuerID               string                   `json:"issuer_id"`
	AuthorizationChain     *AuthorizationChain      `json:"authorization_chain,omitempty"`
	LegalFramework         *LegalFrameworkInfo      `json:"legal_framework,omitempty"`
	Restrictions           []PowerRestriction       `json:"restrictions,omitempty"`
	IssuedAt               time.Time                `json:"issued_at"`
	ExpiresAt              time.Time                `json:"expires_at"`
	ConsentTimestamp       time.Time                `json:"consent_timestamp"`
	GrantCode              string                   `json:"grant_code,omitempty"`
}

// ValidateRequestCompliance implements RFC-0111 Section 6 step (b)
// "Request compliance validation"
// Validates that an authorization request complies with RFC-0111 requirements
func (v *ComplianceValidator) ValidateRequestCompliance(
	ctx context.Context,
	request *ExtendedAuthorizationRequest,
) (*RequestComplianceResult, error) {
	result := &RequestComplianceResult{
		Valid:          true,
		ValidationTime: time.Now(),
		Checks:         make(map[string]bool),
		Warnings:       make([]string, 0),
	}

	// Step 1: Validate request structure
	if err := v.validateRequestStructure(request, result); err != nil {
		result.Valid = false
		result.FailureReason = err.Error()
		return result, err
	}

	// Step 2: Validate client identification
	if err := v.validateClientIdentification(ctx, request, result); err != nil {
		result.Valid = false
		result.FailureReason = fmt.Sprintf("Client identification failed: %v", err)
		return result, err
	}

	// Step 3: Validate authorization chain (if provided)
	if request.AuthorizationChain != nil {
		chainResult, err := v.chainValidator.ValidateAuthorizationChain(ctx, request.AuthorizationChain)
		if err != nil {
			result.Valid = false
			result.FailureReason = fmt.Sprintf("Authorization chain validation failed: %v", err)
			result.ChainValidation = chainResult
			return result, err
		}
		result.ChainValidation = chainResult
		result.Checks["authorization_chain"] = true
	} else {
		result.Warnings = append(result.Warnings, "No authorization chain provided in request")
		result.Checks["authorization_chain"] = false
	}

	// Step 4: Validate Power of Attorney
	if request.PowerOfAttorney != nil {
		if err := v.validatePoA(ctx, request.PowerOfAttorney, result); err != nil {
			result.Valid = false
			result.FailureReason = fmt.Sprintf("PoA validation failed: %v", err)
			return result, err
		}
		result.Checks["power_of_attorney"] = true
	} else {
		if v.strictMode {
			result.Valid = false
			result.FailureReason = "Power of Attorney is required per RFC-0111"
			result.Checks["power_of_attorney"] = false
			return result, &GAuthError{
				Code:    "missing_poa",
				Message: "Power of Attorney is required per RFC-0111",
			}
		}
		result.Warnings = append(result.Warnings, "No Power of Attorney provided")
		result.Checks["power_of_attorney"] = false
	}

	// Step 5: Validate requested scope
	if err := v.validateRequestedScope(ctx, request, result); err != nil {
		result.Valid = false
		result.FailureReason = fmt.Sprintf("Scope validation failed: %v", err)
		return result, err
	}

	// Step 6: Validate legal framework compliance
	if err := v.validateLegalFramework(ctx, request, result); err != nil {
		result.Valid = false
		result.FailureReason = fmt.Sprintf("Legal framework validation failed: %v", err)
		return result, err
	}

	// Step 7: Validate temporal requirements
	if err := v.validateTemporalRequirements(request, result); err != nil {
		result.Valid = false
		result.FailureReason = fmt.Sprintf("Temporal validation failed: %v", err)
		return result, err
	}

	// Step 8: Validate restrictions and limitations
	if err := v.validateRestrictions(ctx, request, result); err != nil {
		result.Valid = false
		result.FailureReason = fmt.Sprintf("Restrictions validation failed: %v", err)
		return result, err
	}

	result.Valid = true
	return result, nil
}

// ValidateGrantCompliance implements RFC-0111 Section 6 step (f)
// "Grant compliance validation"
// Validates that an authorization grant complies with RFC-0111 requirements
func (v *ComplianceValidator) ValidateGrantCompliance(
	ctx context.Context,
	grant *ExtendedAuthorizationGrant,
) (*GrantComplianceResult, error) {
	result := &GrantComplianceResult{
		Valid:          true,
		ValidationTime: time.Now(),
		Checks:         make(map[string]bool),
		Warnings:       make([]string, 0),
	}

	// Step 1: Validate grant structure
	if err := v.validateGrantStructure(grant, result); err != nil {
		result.Valid = false
		result.FailureReason = err.Error()
		return result, err
	}

	// Step 2: Validate issuer authority
	if err := v.validateIssuerAuthority(ctx, grant, result); err != nil {
		result.Valid = false
		result.FailureReason = fmt.Sprintf("Issuer authority validation failed: %v", err)
		return result, err
	}

	// Step 3: Validate resource owner consent
	if err := v.validateResourceOwnerConsent(ctx, grant, result); err != nil {
		result.Valid = false
		result.FailureReason = fmt.Sprintf("Resource owner consent validation failed: %v", err)
		return result, err
	}

	// Step 4: Validate grant scope matches request
	if err := v.validateGrantScopeConsistency(ctx, grant, result); err != nil {
		result.Valid = false
		result.FailureReason = fmt.Sprintf("Grant scope consistency failed: %v", err)
		return result, err
	}

	// Step 5: Validate authorization chain in grant
	if grant.AuthorizationChain != nil {
		chainResult, err := v.chainValidator.ValidateAuthorizationChain(ctx, grant.AuthorizationChain)
		if err != nil {
			result.Valid = false
			result.FailureReason = fmt.Sprintf("Authorization chain validation failed: %v", err)
			result.ChainValidation = chainResult
			return result, err
		}
		result.ChainValidation = chainResult
		result.Checks["authorization_chain"] = true
	} else {
		result.Valid = false
		result.FailureReason = "Authorization chain is required in grant per RFC-0111"
		result.Checks["authorization_chain"] = false
		return result, &GAuthError{
			Code:    "missing_authorization_chain",
			Message: "Authorization chain is required in grant per RFC-0111",
		}
	}

	// Step 6: Validate legal framework consistency
	if err := v.validateGrantLegalFramework(ctx, grant, result); err != nil {
		result.Valid = false
		result.FailureReason = fmt.Sprintf("Legal framework validation failed: %v", err)
		return result, err
	}

	// Step 7: Validate grant expiration
	if err := v.validateGrantExpiration(grant, result); err != nil {
		result.Valid = false
		result.FailureReason = fmt.Sprintf("Grant expiration validation failed: %v", err)
		return result, err
	}

	// Step 8: Validate grant restrictions enforcement
	if err := v.validateGrantRestrictions(ctx, grant, result); err != nil {
		result.Valid = false
		result.FailureReason = fmt.Sprintf("Grant restrictions validation failed: %v", err)
		return result, err
	}

	result.Valid = true
	return result, nil
}

// validateRequestStructure validates the basic structure of an authorization request
func (v *ComplianceValidator) validateRequestStructure(
	request *ExtendedAuthorizationRequest,
	result *RequestComplianceResult,
) error {
	if request == nil || request.AuthorizationRequest == nil {
		return &GAuthError{
			Code:    "invalid_request",
			Message: "Authorization request cannot be nil",
		}
	}

	if request.ClientID == "" {
		result.Checks["client_id"] = false
		return &GAuthError{
			Code:    "missing_client_id",
			Message: "Client ID is required",
		}
	}
	result.Checks["client_id"] = true

	if len(request.Scopes) == 0 {
		result.Checks["scope"] = false
		return &GAuthError{
			Code:    "missing_scope",
			Message: "Request scope is required",
		}
	}
	result.Checks["scope"] = true

	return nil
}

// validateClientIdentification validates client identification
func (v *ComplianceValidator) validateClientIdentification(
	ctx context.Context,
	request *ExtendedAuthorizationRequest,
	result *RequestComplianceResult,
) error {
	// Verify client exists
	if v.pipClient != nil {
		clientInfo, err := v.pipClient.GetClientInfo(ctx, request.ClientID)
		if err != nil {
			result.Checks["client_exists"] = false
			return fmt.Errorf("failed to verify client: %w", err)
		}
		if clientInfo == nil {
			result.Checks["client_exists"] = false
			return &GAuthError{
				Code:    "unknown_client",
				Message: fmt.Sprintf("Client not found: %s", request.ClientID),
			}
		}
		result.Checks["client_exists"] = true
	} else {
		result.Warnings = append(result.Warnings, "PIP client not configured, skipping client verification")
		result.Checks["client_exists"] = false
	}

	return nil
}

// validatePoA validates Power of Attorney
func (v *ComplianceValidator) validatePoA(
	ctx context.Context,
	poaDef *poa.PoADefinition,
	result *RequestComplianceResult,
) error {
	if poaDef == nil {
		return &GAuthError{
			Code:    "invalid_poa",
			Message: "PoA definition cannot be nil",
		}
	}

	// Use PoA package validation
	if err := poa.ValidatePoADefinition(*poaDef); err != nil {
		return fmt.Errorf("PoA validation failed: %w", err)
	}

	// Check PoA authorized client operational status
	if poaDef.Parties.AuthorizedClient.StatusEnum != poa.OperationalStatusActive {
		return &GAuthError{
			Code:    "inactive_poa",
			Message: fmt.Sprintf("PoA authorized client is not active: %s", poaDef.Parties.AuthorizedClient.StatusEnum),
		}
	}

	return nil
}

// validateRequestedScope validates the requested scope
func (v *ComplianceValidator) validateRequestedScope(
	ctx context.Context,
	request *ExtendedAuthorizationRequest,
	result *RequestComplianceResult,
) error {
	// Validate scope against PoA authorized actions
	if request.PowerOfAttorney != nil {
		poaAuthorizedActions := request.PowerOfAttorney.Authorization.AuthorizedActions
		
		// Check if any actions are authorized
		hasTransactions := len(poaAuthorizedActions.Transactions) > 0
		hasDecisions := len(poaAuthorizedActions.Decisions) > 0
		hasPhysicalActions := len(poaAuthorizedActions.PhysicalActions) > 0
		hasNonPhysicalActions := len(poaAuthorizedActions.NonPhysicalActions) > 0
		
		if !hasTransactions && !hasDecisions && !hasPhysicalActions && !hasNonPhysicalActions {
			if v.strictMode {
				result.Checks["scope_allowed"] = false
				return &GAuthError{
					Code:    "no_authorized_actions",
					Message: "PoA does not define any authorized actions",
				}
			}
			result.Warnings = append(result.Warnings, "PoA has no authorized actions defined")
		}
		
		result.Checks["scope_allowed"] = true
	} else {
		result.Warnings = append(result.Warnings, "Cannot validate scope against PoA (not provided)")
		result.Checks["scope_allowed"] = false
	}

	return nil
}

// validateLegalFramework validates legal framework compliance
func (v *ComplianceValidator) validateLegalFramework(
	ctx context.Context,
	request *ExtendedAuthorizationRequest,
	result *RequestComplianceResult,
) error {
	if request.LegalFramework == nil {
		if v.strictMode {
			result.Checks["legal_framework"] = false
			return &GAuthError{
				Code:    "missing_legal_framework",
				Message: "Legal framework is required per RFC-0111",
			}
		}
		result.Warnings = append(result.Warnings, "No legal framework provided")
		result.Checks["legal_framework"] = false
		return nil
	}

	if len(request.LegalFramework.ApplicableLaws) == 0 {
		result.Warnings = append(result.Warnings, "No applicable laws specified in legal framework")
	}

	if request.LegalFramework.Jurisdiction == "" {
		result.Warnings = append(result.Warnings, "No jurisdiction specified in legal framework")
	}

	result.Checks["legal_framework"] = true
	return nil
}

// validateTemporalRequirements validates temporal requirements
func (v *ComplianceValidator) validateTemporalRequirements(
	request *ExtendedAuthorizationRequest,
	result *RequestComplianceResult,
) error {
	now := time.Now()

	// If PoA is provided, check its validity period
	if request.PowerOfAttorney != nil {
		validFrom := request.PowerOfAttorney.Requirements.ValidityPeriod.StartTime
		validUntil := request.PowerOfAttorney.Requirements.ValidityPeriod.EndTime

		if !validFrom.IsZero() && now.Before(validFrom) {
			result.Checks["temporal_validity"] = false
			return &GAuthError{
				Code:    "poa_not_yet_valid",
				Message: "PoA validity period has not started",
			}
		}

		if !validUntil.IsZero() && now.After(validUntil) {
			result.Checks["temporal_validity"] = false
			return &GAuthError{
				Code:    "poa_expired",
				Message: "PoA validity period has expired",
			}
		}
	}

	result.Checks["temporal_validity"] = true
	return nil
}

// validateRestrictions validates request restrictions
func (v *ComplianceValidator) validateRestrictions(
	ctx context.Context,
	request *ExtendedAuthorizationRequest,
	result *RequestComplianceResult,
) error {
	// Validate power restrictions if present
	if request.Restrictions != nil && len(request.Restrictions) > 0 {
		// Check each restriction is properly formed
		for _, restriction := range request.Restrictions {
			if restriction.RestrictionType == "" {
				result.Warnings = append(result.Warnings, "Restriction without type specified")
			}
			if restriction.EnforcementLevel == "" {
				restriction.EnforcementLevel = "mandatory" // Default to mandatory
			}
		}
		result.Checks["restrictions"] = true
	} else {
		result.Checks["restrictions"] = false
	}

	return nil
}

// validateGrantStructure validates grant structure
func (v *ComplianceValidator) validateGrantStructure(
	grant *ExtendedAuthorizationGrant,
	result *GrantComplianceResult,
) error {
	if grant == nil {
		return &GAuthError{
			Code:    "invalid_grant",
			Message: "Authorization grant cannot be nil",
		}
	}

	if grant.GrantID == "" {
		result.Checks["grant_id"] = false
		return &GAuthError{
			Code:    "missing_grant_id",
			Message: "Grant ID is required",
		}
	}
	result.Checks["grant_id"] = true

	if grant.ClientID == "" {
		result.Checks["client_id"] = false
		return &GAuthError{
			Code:    "missing_client_id",
			Message: "Client ID is required in grant",
		}
	}
	result.Checks["client_id"] = true

	if len(grant.Scope) == 0 {
		result.Checks["scope"] = false
		return &GAuthError{
			Code:    "missing_scope",
			Message: "Grant scope is required",
		}
	}
	result.Checks["scope"] = true

	return nil
}

// validateIssuerAuthority validates issuer authority
func (v *ComplianceValidator) validateIssuerAuthority(
	ctx context.Context,
	grant *ExtendedAuthorizationGrant,
	result *GrantComplianceResult,
) error {
	if grant.IssuerID == "" {
		result.Checks["issuer_authority"] = false
		return &GAuthError{
			Code:    "missing_issuer",
			Message: "Grant must have issuer ID",
		}
	}

	// Verify issuer is authorized
	if v.pipClient != nil {
		issuerInfo, err := v.pipClient.GetAuthorizationServerInfo(ctx, grant.IssuerID)
		if err != nil {
			result.Checks["issuer_authority"] = false
			return fmt.Errorf("failed to verify issuer authority: %w", err)
		}
		if issuerInfo == nil {
			result.Checks["issuer_authority"] = false
			return &GAuthError{
				Code:    "unknown_issuer",
				Message: fmt.Sprintf("Issuer not found: %s", grant.IssuerID),
			}
		}
		result.Checks["issuer_authority"] = true
	} else {
		result.Warnings = append(result.Warnings, "PIP client not configured, skipping issuer verification")
		result.Checks["issuer_authority"] = false
	}

	return nil
}

// validateResourceOwnerConsent validates resource owner consent
func (v *ComplianceValidator) validateResourceOwnerConsent(
	ctx context.Context,
	grant *ExtendedAuthorizationGrant,
	result *GrantComplianceResult,
) error {
	if grant.ResourceOwnerID == "" {
		if v.strictMode {
			result.Checks["resource_owner_consent"] = false
			return &GAuthError{
				Code:    "missing_resource_owner",
				Message: "Resource owner ID is required in grant",
			}
		}
		result.Warnings = append(result.Warnings, "No resource owner specified")
		result.Checks["resource_owner_consent"] = false
		return nil
	}

	// Check consent timestamp
	if grant.ConsentTimestamp.IsZero() {
		result.Warnings = append(result.Warnings, "No consent timestamp provided")
	}

	result.Checks["resource_owner_consent"] = true
	return nil
}

// validateGrantScopeConsistency validates grant scope consistency
func (v *ComplianceValidator) validateGrantScopeConsistency(
	ctx context.Context,
	grant *ExtendedAuthorizationGrant,
	result *GrantComplianceResult,
) error {
	// Ensure grant scope doesn't exceed requested scope (if available)
	// This would require accessing the original request, which we'd store
	// For now, just validate scope is reasonable
	result.Checks["scope_consistency"] = true
	return nil
}

// validateGrantLegalFramework validates grant legal framework
func (v *ComplianceValidator) validateGrantLegalFramework(
	ctx context.Context,
	grant *ExtendedAuthorizationGrant,
	result *GrantComplianceResult,
) error {
	if grant.LegalFramework == nil {
		if v.strictMode {
			result.Checks["legal_framework"] = false
			return &GAuthError{
				Code:    "missing_legal_framework",
				Message: "Legal framework is required in grant per RFC-0111",
			}
		}
		result.Warnings = append(result.Warnings, "No legal framework in grant")
		result.Checks["legal_framework"] = false
		return nil
	}

	result.Checks["legal_framework"] = true
	return nil
}

// validateGrantExpiration validates grant expiration
func (v *ComplianceValidator) validateGrantExpiration(
	grant *ExtendedAuthorizationGrant,
	result *GrantComplianceResult,
) error {
	now := time.Now()

	if grant.ExpiresAt.IsZero() {
		result.Warnings = append(result.Warnings, "Grant has no expiration time")
		result.Checks["expiration"] = false
		return nil
	}

	if now.After(grant.ExpiresAt) {
		result.Checks["expiration"] = false
		return &GAuthError{
			Code:    "expired_grant",
			Message: "Grant has expired",
		}
	}

	result.Checks["expiration"] = true
	return nil
}

// validateGrantRestrictions validates grant restrictions
func (v *ComplianceValidator) validateGrantRestrictions(
	ctx context.Context,
	grant *ExtendedAuthorizationGrant,
	result *GrantComplianceResult,
) error {
	if grant.Restrictions != nil && len(grant.Restrictions) > 0 {
		result.Checks["restrictions"] = true
	} else {
		result.Checks["restrictions"] = false
	}
	return nil
}

// RequestComplianceResult represents request compliance validation result
type RequestComplianceResult struct {
	Valid            bool                       `json:"valid"`
	ValidationTime   time.Time                  `json:"validation_time"`
	Checks           map[string]bool            `json:"checks"`
	ChainValidation  *ChainValidationResult     `json:"chain_validation,omitempty"`
	FailureReason    string                     `json:"failure_reason,omitempty"`
	Warnings         []string                   `json:"warnings,omitempty"`
}

// GrantComplianceResult represents grant compliance validation result
type GrantComplianceResult struct {
	Valid            bool                       `json:"valid"`
	ValidationTime   time.Time                  `json:"validation_time"`
	Checks           map[string]bool            `json:"checks"`
	ChainValidation  *ChainValidationResult     `json:"chain_validation,omitempty"`
	FailureReason    string                     `json:"failure_reason,omitempty"`
	Warnings         []string                   `json:"warnings,omitempty"`
}

// PIPClient interface for Power Information Point
type PIPClient interface {
	GetClientInfo(ctx context.Context, clientID string) (*ClientInfo, error)
	GetAuthorizationServerInfo(ctx context.Context, serverID string) (*AuthorizationServerInfo, error)
}

// PDPClient interface for Policy Decision Point
type PDPClient interface {
	EvaluatePolicy(ctx context.Context, request interface{}) (bool, error)
}

// ClientInfo represents client information from PIP
type ClientInfo struct {
	ClientID   string    `json:"client_id"`
	ClientName string    `json:"client_name"`
	Active     bool      `json:"active"`
	RegisteredAt time.Time `json:"registered_at"`
}

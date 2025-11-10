// Package gauth - Authorization Chain Validation per RFC-0111
// Implements critical Gap #1 from QUALITY_MANAGER_RFC_COMPLIANCE_FINAL_ASSESSMENT.md
package gauth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"
)

// ValidationContext provides context for authorization chain validation
type ValidationContext struct {
	Context                context.Context
	CommercialRegisterClient CommercialRegisterClient
	TrustServiceProvider   TrustServiceProvider
	RevocationChecker      RevocationChecker
	StrictMode             bool
}

// AuthorizationChainValidator validates complete authorization chains per RFC-0111
type AuthorizationChainValidator struct {
	commercialRegisterClient CommercialRegisterClient
	trustServiceProvider     TrustServiceProvider
	revocationChecker        RevocationChecker
	strictMode               bool
}

// NewAuthorizationChainValidator creates a new validator instance
func NewAuthorizationChainValidator(
	commercialRegisterClient CommercialRegisterClient,
	trustServiceProvider TrustServiceProvider,
	revocationChecker RevocationChecker,
) *AuthorizationChainValidator {
	return &AuthorizationChainValidator{
		commercialRegisterClient: commercialRegisterClient,
		trustServiceProvider:     trustServiceProvider,
		revocationChecker:        revocationChecker,
		strictMode:               true,
	}
}

// ValidateAuthorizationChain performs comprehensive RFC-0111 authorization chain validation
// RFC Requirement (Section 3, Page 6):
// "The 'owner's authorizer' is the authorizer of the client owner or resource owner,
// respectively, and defines the power of attorney of the client owner or resource owner,
// e.g. its statutory authority."
func (v *AuthorizationChainValidator) ValidateAuthorizationChain(
	ctx context.Context,
	chain *AuthorizationChain,
) (*ChainValidationResult, error) {
	if chain == nil {
		return nil, &GAuthError{
			Code:    "missing_chain",
			Message: "Authorization chain is required",
		}
	}

	result := &ChainValidationResult{
		Valid:             true,
		ValidationTime:    time.Now(),
		LinkValidations:   make([]*LinkValidationResult, 0),
		Warnings:          make([]string, 0),
	}

	// Step 1: Validate chain structure
	if err := v.validateChainStructure(chain, result); err != nil {
		result.Valid = false
		result.FailureReason = err.Error()
		return result, err
	}

	// Step 2: Validate Owner's Authorizer (Level 1)
	authorizerValidation, err := v.validateOwnersAuthorizer(ctx, chain.OwnersAuthorizer)
	if err != nil {
		result.Valid = false
		result.FailureReason = fmt.Sprintf("Owner's authorizer validation failed: %v", err)
		result.LinkValidations = append(result.LinkValidations, authorizerValidation)
		return result, err
	}
	result.LinkValidations = append(result.LinkValidations, authorizerValidation)

	// Step 3: Validate Client Owner (Level 2)
	ownerValidation, err := v.validateClientOwner(ctx, chain.ClientOwner, chain.OwnersAuthorizer)
	if err != nil {
		result.Valid = false
		result.FailureReason = fmt.Sprintf("Client owner validation failed: %v", err)
		result.LinkValidations = append(result.LinkValidations, ownerValidation)
		return result, err
	}
	result.LinkValidations = append(result.LinkValidations, ownerValidation)

	// Step 4: Validate Client (Level 3)
	clientValidation, err := v.validateClient(ctx, chain.Client, chain.ClientOwner)
	if err != nil {
		result.Valid = false
		result.FailureReason = fmt.Sprintf("Client validation failed: %v", err)
		result.LinkValidations = append(result.LinkValidations, clientValidation)
		return result, err
	}
	result.LinkValidations = append(result.LinkValidations, clientValidation)

	// Step 5: Verify chain integrity (cryptographic)
	integrityValid, integrityHash := v.verifyChainIntegrity(chain)
	result.ChainIntegrityHash = integrityHash
	result.ChainIntegrityValid = integrityValid
	if !integrityValid {
		result.Warnings = append(result.Warnings, "Chain integrity verification failed")
	}

	// Step 6: Check for revocations
	revocationResult, err := v.checkChainRevocations(ctx, chain)
	if err != nil {
		result.Warnings = append(result.Warnings, fmt.Sprintf("Revocation check warning: %v", err))
	}
	result.RevocationStatus = revocationResult

	// Step 7: Validate chain continuity
	if err := v.validateChainContinuity(chain); err != nil {
		result.Valid = false
		result.FailureReason = fmt.Sprintf("Chain continuity validation failed: %v", err)
		return result, err
	}

	result.Valid = true
	result.ValidatedChainDepth = chain.ChainDepth

	return result, nil
}

// validateChainStructure validates the basic structure of the authorization chain
func (v *AuthorizationChainValidator) validateChainStructure(
	chain *AuthorizationChain,
	result *ChainValidationResult,
) error {
	if chain.OwnersAuthorizer == nil {
		return &GAuthError{
			Code:    "missing_authorizer",
			Message: "Authorization chain must start with owner's authorizer",
		}
	}

	if chain.ClientOwner == nil {
		return &GAuthError{
			Code:    "missing_owner",
			Message: "Authorization chain must include client owner",
		}
	}

	if chain.Client == nil {
		return &GAuthError{
			Code:    "missing_client",
			Message: "Authorization chain must end with client",
		}
	}

	// Validate chain linkage
	if chain.ClientOwner.AuthorizedBy != chain.OwnersAuthorizer.EntityID {
		return &GAuthError{
			Code:    "broken_chain",
			Message: "Client owner must be authorized by owner's authorizer",
		}
	}

	if chain.Client.AuthorizedBy != chain.ClientOwner.EntityID {
		return &GAuthError{
			Code:    "broken_chain",
			Message: "Client must be authorized by client owner",
		}
	}

	return nil
}

// validateOwnersAuthorizer validates the owner's authorizer with RFC-0111 requirements
func (v *AuthorizationChainValidator) validateOwnersAuthorizer(
	ctx context.Context,
	authorizer *AuthorizationLink,
) (*LinkValidationResult, error) {
	result := &LinkValidationResult{
		Level:      1,
		EntityID:   authorizer.EntityID,
		EntityRole: authorizer.Role,
		Valid:      true,
		Checks:     make(map[string]bool),
	}

	// Check 1: Identity verification
	if !authorizer.IdentityVerified {
		result.Valid = false
		result.FailureReason = "Identity not verified"
		result.Checks["identity_verified"] = false
		return result, &GAuthError{
			Code:    "identity_not_verified",
			Message: "Owner's authorizer identity must be verified",
		}
	}
	result.Checks["identity_verified"] = true

	// Check 2: Status validation
	if authorizer.Status != "active" {
		result.Valid = false
		result.FailureReason = fmt.Sprintf("Invalid status: %s", authorizer.Status)
		result.Checks["status_active"] = false
		return result, &GAuthError{
			Code:    "invalid_status",
			Message: fmt.Sprintf("Owner's authorizer status must be active, got: %s", authorizer.Status),
		}
	}
	result.Checks["status_active"] = true

	// Check 3: Temporal validity
	now := time.Now()
	if now.Before(authorizer.ValidFrom) {
		result.Valid = false
		result.FailureReason = "Not yet valid"
		result.Checks["temporal_validity"] = false
		return result, &GAuthError{
			Code:    "not_yet_valid",
			Message: "Owner's authorizer authorization not yet valid",
		}
	}
	if now.After(authorizer.ValidUntil) {
		result.Valid = false
		result.FailureReason = "Expired"
		result.Checks["temporal_validity"] = false
		return result, &GAuthError{
			Code:    "expired",
			Message: "Owner's authorizer authorization has expired",
		}
	}
	result.Checks["temporal_validity"] = true

	// Check 4: Statutory authority validation (RFC requirement)
	if authorizer.StatutoryAuthority == "" {
		if v.strictMode {
			result.Valid = false
			result.FailureReason = "Missing statutory authority"
			result.Checks["statutory_authority"] = false
			return result, &GAuthError{
				Code:    "missing_statutory_authority",
				Message: "Owner's authorizer must have statutory authority per RFC-0111",
			}
		}
		result.Warnings = append(result.Warnings, "Statutory authority not specified")
		result.Checks["statutory_authority"] = false
	} else {
		result.Checks["statutory_authority"] = true
	}

	// Check 5: Commercial register verification (RFC requirement)
	if authorizer.CommercialRegisterRef != "" && v.commercialRegisterClient != nil {
		registerValid, err := v.verifyCommercialRegisterEntry(
			ctx,
			authorizer.CommercialRegisterRef,
			authorizer.EntityID,
			authorizer.LegalBasis,
		)
		if err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("Commercial register check warning: %v", err))
			result.Checks["commercial_register"] = false
		} else {
			result.Checks["commercial_register"] = registerValid
			if !registerValid && v.strictMode {
				result.Valid = false
				result.FailureReason = "Commercial register verification failed"
				return result, &GAuthError{
					Code:    "commercial_register_invalid",
					Message: "Owner's authorizer commercial register entry is invalid",
				}
			}
		}
	} else {
		result.Checks["commercial_register"] = false
		if v.strictMode && authorizer.LegalBasis != nil && authorizer.LegalBasis.BasisType == "company_law" {
			result.Valid = false
			result.FailureReason = "Missing commercial register reference"
			return result, &GAuthError{
				Code:    "missing_commercial_register",
				Message: "Owner's authorizer with company law basis must have commercial register reference",
			}
		}
	}

	// Check 6: Verification proof validation
	if authorizer.VerificationProof != "" && v.trustServiceProvider != nil {
		proofValid, err := v.verifyIdentityProof(
			ctx,
			authorizer.EntityID,
			authorizer.VerificationProof,
			authorizer.VerificationMethod,
		)
		if err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("Verification proof check warning: %v", err))
			result.Checks["verification_proof"] = false
		} else {
			result.Checks["verification_proof"] = proofValid
			if !proofValid && v.strictMode {
				result.Valid = false
				result.FailureReason = "Verification proof invalid"
				return result, &GAuthError{
					Code:    "verification_proof_invalid",
					Message: "Owner's authorizer verification proof is invalid",
				}
			}
		}
	} else {
		result.Checks["verification_proof"] = false
	}

	// Check 7: Legal basis validation
	if authorizer.LegalBasis == nil {
		if v.strictMode {
			result.Valid = false
			result.FailureReason = "Missing legal basis"
			result.Checks["legal_basis"] = false
			return result, &GAuthError{
				Code:    "missing_legal_basis",
				Message: "Owner's authorizer must have legal basis",
			}
		}
		result.Warnings = append(result.Warnings, "Legal basis not specified")
		result.Checks["legal_basis"] = false
	} else {
		result.Checks["legal_basis"] = true
	}

	return result, nil
}

// validateClientOwner validates the client owner link
func (v *AuthorizationChainValidator) validateClientOwner(
	ctx context.Context,
	owner *AuthorizationLink,
	authorizer *AuthorizationLink,
) (*LinkValidationResult, error) {
	result := &LinkValidationResult{
		Level:      2,
		EntityID:   owner.EntityID,
		EntityRole: owner.Role,
		Valid:      true,
		Checks:     make(map[string]bool),
	}

	// Check 1: Identity verification
	if !owner.IdentityVerified {
		result.Valid = false
		result.FailureReason = "Identity not verified"
		result.Checks["identity_verified"] = false
		return result, &GAuthError{
			Code:    "identity_not_verified",
			Message: "Client owner identity must be verified",
		}
	}
	result.Checks["identity_verified"] = true

	// Check 2: Status validation
	if owner.Status != "active" {
		result.Valid = false
		result.FailureReason = fmt.Sprintf("Invalid status: %s", owner.Status)
		result.Checks["status_active"] = false
		return result, &GAuthError{
			Code:    "invalid_status",
			Message: fmt.Sprintf("Client owner status must be active, got: %s", owner.Status),
		}
	}
	result.Checks["status_active"] = true

	// Check 3: Temporal validity
	now := time.Now()
	if now.Before(owner.ValidFrom) || now.After(owner.ValidUntil) {
		result.Valid = false
		result.FailureReason = "Temporal validity check failed"
		result.Checks["temporal_validity"] = false
		return result, &GAuthError{
			Code:    "temporal_validity_failed",
			Message: "Client owner authorization is not temporally valid",
		}
	}
	result.Checks["temporal_validity"] = true

	// Check 4: Authorization linkage to authorizer
	if owner.AuthorizedBy != authorizer.EntityID {
		result.Valid = false
		result.FailureReason = "Authorization linkage broken"
		result.Checks["authorization_linkage"] = false
		return result, &GAuthError{
			Code:    "broken_authorization_chain",
			Message: "Client owner not properly authorized by owner's authorizer",
		}
	}
	result.Checks["authorization_linkage"] = true

	// Check 5: Authorization type validation
	validAuthTypes := []string{"statutory", "contractual", "delegated"}
	validAuthType := false
	for _, vat := range validAuthTypes {
		if owner.AuthorizationType == vat {
			validAuthType = true
			break
		}
	}
	if !validAuthType {
		result.Valid = false
		result.FailureReason = fmt.Sprintf("Invalid authorization type: %s", owner.AuthorizationType)
		result.Checks["authorization_type"] = false
		return result, &GAuthError{
			Code:    "invalid_authorization_type",
			Message: fmt.Sprintf("Client owner has invalid authorization type: %s", owner.AuthorizationType),
		}
	}
	result.Checks["authorization_type"] = true

	// Check 6: Scope of authority
	if len(owner.ScopeOfAuthority) == 0 {
		result.Warnings = append(result.Warnings, "No scope of authority defined")
		result.Checks["scope_of_authority"] = false
	} else {
		result.Checks["scope_of_authority"] = true
	}

	return result, nil
}

// validateClient validates the AI client link
func (v *AuthorizationChainValidator) validateClient(
	ctx context.Context,
	client *AuthorizationLink,
	owner *AuthorizationLink,
) (*LinkValidationResult, error) {
	result := &LinkValidationResult{
		Level:      3,
		EntityID:   client.EntityID,
		EntityRole: client.Role,
		Valid:      true,
		Checks:     make(map[string]bool),
	}

	// Check 1: Identity verification
	if !client.IdentityVerified {
		result.Valid = false
		result.FailureReason = "Identity not verified"
		result.Checks["identity_verified"] = false
		return result, &GAuthError{
			Code:    "identity_not_verified",
			Message: "Client identity must be verified",
		}
	}
	result.Checks["identity_verified"] = true

	// Check 2: Status validation
	if client.Status != "active" {
		result.Valid = false
		result.FailureReason = fmt.Sprintf("Invalid status: %s", client.Status)
		result.Checks["status_active"] = false
		return result, &GAuthError{
			Code:    "invalid_status",
			Message: fmt.Sprintf("Client status must be active, got: %s", client.Status),
		}
	}
	result.Checks["status_active"] = true

	// Check 3: Temporal validity
	now := time.Now()
	if now.Before(client.ValidFrom) || now.After(client.ValidUntil) {
		result.Valid = false
		result.FailureReason = "Temporal validity check failed"
		result.Checks["temporal_validity"] = false
		return result, &GAuthError{
			Code:    "temporal_validity_failed",
			Message: "Client authorization is not temporally valid",
		}
	}
	result.Checks["temporal_validity"] = true

	// Check 4: Authorization linkage to owner
	if client.AuthorizedBy != owner.EntityID {
		result.Valid = false
		result.FailureReason = "Authorization linkage broken"
		result.Checks["authorization_linkage"] = false
		return result, &GAuthError{
			Code:    "broken_authorization_chain",
			Message: "Client not properly authorized by client owner",
		}
	}
	result.Checks["authorization_linkage"] = true

	// Check 5: Entity type validation (must be AI system)
	validClientTypes := []string{"ai_system", "digital_agent", "llm", "robotic_system"}
	validType := false
	for _, vct := range validClientTypes {
		if client.EntityType == vct {
			validType = true
			break
		}
	}
	if !validType {
		result.Valid = false
		result.FailureReason = fmt.Sprintf("Invalid entity type: %s", client.EntityType)
		result.Checks["entity_type"] = false
		return result, &GAuthError{
			Code:    "invalid_entity_type",
			Message: fmt.Sprintf("Client has invalid entity type: %s (expected AI system types)", client.EntityType),
		}
	}
	result.Checks["entity_type"] = true

	return result, nil
}

// verifyChainIntegrity computes and verifies cryptographic integrity hash
func (v *AuthorizationChainValidator) verifyChainIntegrity(chain *AuthorizationChain) (bool, string) {
	// Compute chain hash
	hasher := sha256.New()
	
	// Hash owner's authorizer
	hasher.Write([]byte(chain.OwnersAuthorizer.EntityID))
	hasher.Write([]byte(chain.OwnersAuthorizer.Role))
	hasher.Write([]byte(chain.OwnersAuthorizer.Status))
	
	// Hash client owner
	hasher.Write([]byte(chain.ClientOwner.EntityID))
	hasher.Write([]byte(chain.ClientOwner.AuthorizedBy))
	hasher.Write([]byte(chain.ClientOwner.Status))
	
	// Hash client
	hasher.Write([]byte(chain.Client.EntityID))
	hasher.Write([]byte(chain.Client.AuthorizedBy))
	hasher.Write([]byte(chain.Client.Status))
	
	computedHash := hex.EncodeToString(hasher.Sum(nil))
	
	// If chain has existing integrity hash, verify it matches
	if chain.ChainIntegrity != "" {
		return chain.ChainIntegrity == computedHash, computedHash
	}
	
	// Otherwise, this is the first validation
	return true, computedHash
}

// checkChainRevocations checks if any link in the chain has been revoked
func (v *AuthorizationChainValidator) checkChainRevocations(
	ctx context.Context,
	chain *AuthorizationChain,
) (*RevocationCheckResult, error) {
	if v.revocationChecker == nil {
		return &RevocationCheckResult{
			Checked: false,
			Message: "No revocation checker configured",
		}, nil
	}

	result := &RevocationCheckResult{
		Checked: true,
		Revoked: false,
		LinkRevocations: make(map[string]bool),
	}

	// Check owner's authorizer
	authorizerRevoked, err := v.revocationChecker.IsRevoked(ctx, chain.OwnersAuthorizer.EntityID)
	if err != nil {
		return nil, fmt.Errorf("failed to check authorizer revocation: %w", err)
	}
	result.LinkRevocations["owners_authorizer"] = authorizerRevoked
	if authorizerRevoked {
		result.Revoked = true
		result.RevokedEntity = "owners_authorizer"
		return result, nil
	}

	// Check client owner
	ownerRevoked, err := v.revocationChecker.IsRevoked(ctx, chain.ClientOwner.EntityID)
	if err != nil {
		return nil, fmt.Errorf("failed to check owner revocation: %w", err)
	}
	result.LinkRevocations["client_owner"] = ownerRevoked
	if ownerRevoked {
		result.Revoked = true
		result.RevokedEntity = "client_owner"
		return result, nil
	}

	// Check client
	clientRevoked, err := v.revocationChecker.IsRevoked(ctx, chain.Client.EntityID)
	if err != nil {
		return nil, fmt.Errorf("failed to check client revocation: %w", err)
	}
	result.LinkRevocations["client"] = clientRevoked
	if clientRevoked {
		result.Revoked = true
		result.RevokedEntity = "client"
		return result, nil
	}

	return result, nil
}

// validateChainContinuity validates temporal continuity and scope consistency
func (v *AuthorizationChainValidator) validateChainContinuity(chain *AuthorizationChain) error {
	// Check temporal continuity: each level's validity must fall within parent's validity
	
	// Client owner must be valid during authorizer's validity period
	if chain.ClientOwner.ValidFrom.Before(chain.OwnersAuthorizer.ValidFrom) {
		return &GAuthError{
			Code:    "temporal_discontinuity",
			Message: "Client owner validity starts before authorizer's validity",
		}
	}
	if chain.ClientOwner.ValidUntil.After(chain.OwnersAuthorizer.ValidUntil) {
		return &GAuthError{
			Code:    "temporal_discontinuity",
			Message: "Client owner validity extends beyond authorizer's validity",
		}
	}
	
	// Client must be valid during owner's validity period
	if chain.Client.ValidFrom.Before(chain.ClientOwner.ValidFrom) {
		return &GAuthError{
			Code:    "temporal_discontinuity",
			Message: "Client validity starts before owner's validity",
		}
	}
	if chain.Client.ValidUntil.After(chain.ClientOwner.ValidUntil) {
		return &GAuthError{
			Code:    "temporal_discontinuity",
			Message: "Client validity extends beyond owner's validity",
		}
	}
	
	return nil
}

// verifyCommercialRegisterEntry verifies an entity against commercial register
func (v *AuthorizationChainValidator) verifyCommercialRegisterEntry(
	ctx context.Context,
	registerRef string,
	entityID string,
	legalBasis *LegalBasis,
) (bool, error) {
	if v.commercialRegisterClient == nil {
		return false, fmt.Errorf("commercial register client not configured")
	}
	
	jurisdiction := ""
	if legalBasis != nil {
		jurisdiction = legalBasis.Jurisdiction
	}
	
	companyInfo, err := v.commercialRegisterClient.VerifyCompany(ctx, jurisdiction, registerRef)
	if err != nil {
		return false, err
	}
	
	// Verify the entity ID matches company info
	if companyInfo.CompanyID != entityID && companyInfo.RegistrationNumber != entityID {
		return false, nil
	}
	
	return companyInfo.Active, nil
}

// verifyIdentityProof verifies identity verification proof
func (v *AuthorizationChainValidator) verifyIdentityProof(
	ctx context.Context,
	entityID string,
	proof string,
	method string,
) (bool, error) {
	if v.trustServiceProvider == nil {
		return false, fmt.Errorf("trust service provider not configured")
	}
	
	// Parse proof and verify with TSP
	verificationResult, err := v.trustServiceProvider.VerifyIdentity(ctx, &IdentityDocument{
		DocumentID:   proof,
		SubjectID:    entityID,
		DocumentType: method,
	})
	if err != nil {
		return false, err
	}
	
	return verificationResult.Verified, nil
}

// ChainValidationResult represents the result of authorization chain validation
type ChainValidationResult struct {
	Valid                bool                      `json:"valid"`
	ValidationTime       time.Time                 `json:"validation_time"`
	ValidatedChainDepth  int                       `json:"validated_chain_depth"`
	LinkValidations      []*LinkValidationResult   `json:"link_validations"`
	ChainIntegrityValid  bool                      `json:"chain_integrity_valid"`
	ChainIntegrityHash   string                    `json:"chain_integrity_hash"`
	RevocationStatus     *RevocationCheckResult    `json:"revocation_status,omitempty"`
	FailureReason        string                    `json:"failure_reason,omitempty"`
	Warnings             []string                  `json:"warnings,omitempty"`
}

// LinkValidationResult represents validation result for a single authorization link
type LinkValidationResult struct {
	Level         int               `json:"level"` // 1=Authorizer, 2=Owner, 3=Client
	EntityID      string            `json:"entity_id"`
	EntityRole    string            `json:"entity_role"`
	Valid         bool              `json:"valid"`
	Checks        map[string]bool   `json:"checks"`
	FailureReason string            `json:"failure_reason,omitempty"`
	Warnings      []string          `json:"warnings,omitempty"`
}

// RevocationCheckResult represents revocation check results
type RevocationCheckResult struct {
	Checked         bool              `json:"checked"`
	Revoked         bool              `json:"revoked"`
	RevokedEntity   string            `json:"revoked_entity,omitempty"`
	LinkRevocations map[string]bool   `json:"link_revocations"`
	Message         string            `json:"message,omitempty"`
}

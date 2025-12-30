// Package agentauth - Authorization Chain Validation per AAP-001
// Implements critical Gap #1 from QUALITY_MANAGER_RFC_COMPLIANCE_FINAL_ASSESSMENT.md
package agentauth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"
)

// Authorization chain validation constants
const (
	// MaxChainDepth is the maximum allowed depth for authorization chains
	// to prevent infinitely long delegation chains
	MaxChainDepth = 10

	// MaxDelegationHops is the maximum number of delegation steps allowed
	MaxDelegationHops = 5

	// ChainExpirationBuffer is the time buffer before expiration to warn
	ChainExpirationBuffer = 24 * time.Hour
)

// ValidationContext provides context for authorization chain validation
type ValidationContext struct {
	Context                  context.Context
	CommercialRegisterClient CommercialRegisterClient
	TrustServiceProvider     TrustServiceProvider
	RevocationChecker        RevocationChecker
	StrictMode               bool
}

// AuthorizationChainValidator validates complete authorization chains per AAP-001
type AuthorizationChainValidator struct {
	commercialRegisterClient CommercialRegisterClient
	trustServiceProvider     TrustServiceProvider
	revocationChecker        RevocationChecker
	strictMode               bool
	maxChainDepth            int
	maxDelegationHops        int
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
		maxChainDepth:            MaxChainDepth,
		maxDelegationHops:        MaxDelegationHops,
	}
}

// SetMaxChainDepth allows configuration of maximum chain depth
func (v *AuthorizationChainValidator) SetMaxChainDepth(depth int) {
	if depth > 0 && depth <= 100 {
		v.maxChainDepth = depth
	}
}

// SetMaxDelegationHops allows configuration of maximum delegation hops
func (v *AuthorizationChainValidator) SetMaxDelegationHops(hops int) {
	if hops > 0 && hops <= 20 {
		v.maxDelegationHops = hops
	}
}

// ValidateAuthorizationChain performs comprehensive AAP-001 authorization chain validation
// RFC Requirement (Section 3, Page 6):
// "The 'owner's authorizer' is the authorizer of the client owner or resource owner,
// respectively, and defines the power of attorney of the client owner or resource owner,
// e.g. its statutory authority."
func (v *AuthorizationChainValidator) ValidateAuthorizationChain(
	ctx context.Context,
	chain *AuthorizationChain,
) (*ChainValidationResult, error) {
	if chain == nil {
		return nil, &AgentAuthError{
			Code:    "missing_chain",
			Message: "Authorization chain is required",
		}
	}

	result := &ChainValidationResult{
		Valid:           true,
		ValidationTime:  time.Now(),
		LinkValidations: make([]*LinkValidationResult, 0),
		Warnings:        make([]string, 0),
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
	if revocationResult != nil && revocationResult.Revoked {
		result.Valid = false
		result.FailureReason = fmt.Sprintf("Authorization chain revoked: %s (Entity: %s)", revocationResult.Message, revocationResult.RevokedEntity)
		return result, nil
	}

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
		return &AgentAuthError{
			Code:    "missing_authorizer",
			Message: "Authorization chain must start with owner's authorizer",
		}
	}

	if chain.ClientOwner == nil {
		return &AgentAuthError{
			Code:    "missing_owner",
			Message: "Authorization chain must include client owner",
		}
	}

	if chain.Client == nil {
		return &AgentAuthError{
			Code:    "missing_client",
			Message: "Authorization chain must end with client",
		}
	}

	// Enhanced Validation 1: Check chain depth limit
	chainDepth := v.calculateChainDepth(chain)
	result.ValidatedChainDepth = chainDepth
	if chainDepth > v.maxChainDepth {
		return &AgentAuthError{
			Code:    "chain_too_deep",
			Message: fmt.Sprintf("Authorization chain depth %d exceeds maximum %d", chainDepth, v.maxChainDepth),
		}
	}

	// Enhanced Validation 2: Check for circular references
	if err := v.detectCircularReferences(chain); err != nil {
		return err
	}

	// Enhanced Validation 3: Validate delegation path
	if err := v.validateDelegationPath(chain, result); err != nil {
		return err
	}

	// Enhanced Validation 4: Check expiration across chain
	if err := v.validateChainExpiration(chain, result); err != nil {
		return err
	}

	// Validate chain linkage
	if chain.ClientOwner.AuthorizedBy != chain.OwnersAuthorizer.EntityID {
		return &AgentAuthError{
			Code:    "broken_chain",
			Message: "Client owner must be authorized by owner's authorizer",
		}
	}

	if chain.Client.AuthorizedBy != chain.ClientOwner.EntityID {
		return &AgentAuthError{
			Code:    "broken_chain",
			Message: "Client must be authorized by client owner",
		}
	}

	return nil
}

// calculateChainDepth calculates the depth of the authorization chain
func (v *AuthorizationChainValidator) calculateChainDepth(chain *AuthorizationChain) int {
	depth := 0

	if chain.OwnersAuthorizer != nil {
		depth++
	}
	if chain.ClientOwner != nil {
		depth++
	}
	if chain.Client != nil {
		depth++
	}

	// Use the ChainDepth field if available
	if chain.ChainDepth > depth {
		depth = chain.ChainDepth
	}

	return depth
}

// detectCircularReferences checks for circular references in the authorization chain
func (v *AuthorizationChainValidator) detectCircularReferences(chain *AuthorizationChain) error {
	seen := make(map[string]bool)

	// Check owner's authorizer
	if chain.OwnersAuthorizer != nil {
		entityID := chain.OwnersAuthorizer.EntityID
		if seen[entityID] {
			return &AgentAuthError{
				Code:    "circular_reference",
				Message: fmt.Sprintf("Circular reference detected: entity %s appears multiple times", entityID),
			}
		}
		seen[entityID] = true
	}

	// Check client owner
	if chain.ClientOwner != nil {
		entityID := chain.ClientOwner.EntityID
		if seen[entityID] {
			return &AgentAuthError{
				Code:    "circular_reference",
				Message: fmt.Sprintf("Circular reference detected: entity %s appears multiple times", entityID),
			}
		}
		seen[entityID] = true

		// Check if owner is authorizing itself
		if chain.OwnersAuthorizer != nil && entityID == chain.OwnersAuthorizer.EntityID {
			return &AgentAuthError{
				Code:    "self_authorization",
				Message: "Entity cannot authorize itself",
			}
		}
	}

	// Check client
	if chain.Client != nil {
		entityID := chain.Client.EntityID
		if seen[entityID] {
			return &AgentAuthError{
				Code:    "circular_reference",
				Message: fmt.Sprintf("Circular reference detected: entity %s appears multiple times", entityID),
			}
		}
		seen[entityID] = true
	}

	return nil
}

// validateDelegationPath validates the complete delegation path
func (v *AuthorizationChainValidator) validateDelegationPath(chain *AuthorizationChain, result *ChainValidationResult) error {
	// Validate the 3-level chain structure
	// Level 1: Owner's Authorizer → Level 2: Client Owner → Level 3: Client

	// Verify authorization types are appropriate for delegation
	if chain.ClientOwner != nil {
		// Client Owner must be authorized by Owner's Authorizer
		if chain.ClientOwner.AuthorizationType == "delegated" {
			// This is a delegated authorization - count as delegation hop
			if chain.ClientOwner.AuthorizedBy == "" {
				return &AgentAuthError{
					Code:    "missing_delegator",
					Message: "Delegated authorization must specify authorizer",
				}
			}
		}
	}

	if chain.Client != nil {
		// Client must be authorized by Client Owner
		if chain.Client.AuthorizationType == "delegated" {
			if chain.Client.AuthorizedBy == "" {
				return &AgentAuthError{
					Code:    "missing_delegator",
					Message: "Delegated client must specify authorizer",
				}
			}
		}

		// Verify client status is active
		if chain.Client.Status != string(PolicyStatusActive) {
			return &AgentAuthError{
				Code:    "inactive_client",
				Message: fmt.Sprintf("Client has invalid status: %s", chain.Client.Status),
			}
		}
	}

	return nil
}

// validateChainExpiration checks expiration across the entire chain
func (v *AuthorizationChainValidator) validateChainExpiration(chain *AuthorizationChain, result *ChainValidationResult) error {
	now := time.Now()
	warningThreshold := now.Add(ChainExpirationBuffer)

	// Check owner's authorizer expiration
	if chain.OwnersAuthorizer != nil && !chain.OwnersAuthorizer.ValidUntil.IsZero() {
		validUntil := chain.OwnersAuthorizer.ValidUntil

		if now.After(validUntil) {
			return &AgentAuthError{
				Code:    "authorizer_expired",
				Message: fmt.Sprintf("Owner's authorizer expired at %v", validUntil),
			}
		}

		if validUntil.Before(warningThreshold) {
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("Owner's authorizer expires soon: %v (within %v)", validUntil, ChainExpirationBuffer))
		}
	}

	// Check client owner expiration
	if chain.ClientOwner != nil && !chain.ClientOwner.ValidUntil.IsZero() {
		validUntil := chain.ClientOwner.ValidUntil

		if now.After(validUntil) {
			return &AgentAuthError{
				Code:    "owner_expired",
				Message: fmt.Sprintf("Client owner authorization expired at %v", validUntil),
			}
		}

		if validUntil.Before(warningThreshold) {
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("Client owner authorization expires soon: %v (within %v)", validUntil, ChainExpirationBuffer))
		}
	}

	// Check client expiration
	if chain.Client != nil && !chain.Client.ValidUntil.IsZero() {
		validUntil := chain.Client.ValidUntil

		if now.After(validUntil) {
			return &AgentAuthError{
				Code:    "client_expired",
				Message: fmt.Sprintf("Client authorization expired at %v", validUntil),
			}
		}

		if validUntil.Before(warningThreshold) {
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("Client authorization expires soon: %v (within %v)", validUntil, ChainExpirationBuffer))
		}
	}

	return nil
}

// validateOwnersAuthorizer validates the owner's authorizer with AAP-001 requirements
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
		return result, &AgentAuthError{
			Code:    "identity_not_verified",
			Message: "Owner's authorizer identity must be verified",
		}
	}
	result.Checks["identity_verified"] = true

	// Check 2: Status validation
	if authorizer.Status != string(PolicyStatusActive) {
		result.Valid = false
		result.FailureReason = fmt.Sprintf("Invalid status: %s", authorizer.Status)
		result.Checks["status_active"] = false
		return result, &AgentAuthError{
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
		return result, &AgentAuthError{
			Code:    "not_yet_valid",
			Message: "Owner's authorizer authorization not yet valid",
		}
	}
	if now.After(authorizer.ValidUntil) {
		result.Valid = false
		result.FailureReason = "Expired"
		result.Checks["temporal_validity"] = false
		return result, &AgentAuthError{
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
			return result, &AgentAuthError{
				Code:    "missing_statutory_authority",
				Message: "Owner's authorizer must have statutory authority per AAP-001",
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
				return result, &AgentAuthError{
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
			return result, &AgentAuthError{
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
				return result, &AgentAuthError{
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
			return result, &AgentAuthError{
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
		return result, &AgentAuthError{
			Code:    "identity_not_verified",
			Message: "Client owner identity must be verified",
		}
	}
	result.Checks["identity_verified"] = true

	// Check 2: Status validation
	if owner.Status != string(PolicyStatusActive) {
		result.Valid = false
		result.FailureReason = fmt.Sprintf("Invalid status: %s", owner.Status)
		result.Checks["status_active"] = false
		return result, &AgentAuthError{
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
		return result, &AgentAuthError{
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
		return result, &AgentAuthError{
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
		return result, &AgentAuthError{
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
		return result, &AgentAuthError{
			Code:    "identity_not_verified",
			Message: "Client identity must be verified",
		}
	}
	result.Checks["identity_verified"] = true

	// Check 2: Status validation
	if client.Status != string(PolicyStatusActive) {
		result.Valid = false
		result.FailureReason = fmt.Sprintf("Invalid status: %s", client.Status)
		result.Checks["status_active"] = false
		return result, &AgentAuthError{
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
		return result, &AgentAuthError{
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
		return result, &AgentAuthError{
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
		return result, &AgentAuthError{
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
		Checked:         true,
		Revoked:         false,
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

	// Check owner's authorizer delegation revocation (if applicable)
	if chain.OwnersAuthorizer.AuthorizationType == "delegated" && chain.OwnersAuthorizer.DelegationID != "" {
		delegationRevoked, err := v.revocationChecker.IsDelegationRevoked(ctx, chain.OwnersAuthorizer.DelegationID)
		if err != nil {
			return nil, fmt.Errorf("failed to check authorizer delegation revocation: %w", err)
		}
		result.LinkRevocations["owners_authorizer_delegation"] = delegationRevoked
		if delegationRevoked {
			result.Revoked = true
			result.RevokedEntity = "owners_authorizer_delegation"
			return result, nil
		}
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

	// Check client owner delegation revocation (if applicable)
	if chain.ClientOwner.AuthorizationType == "delegated" && chain.ClientOwner.DelegationID != "" {
		delegationRevoked, err := v.revocationChecker.IsDelegationRevoked(ctx, chain.ClientOwner.DelegationID)
		if err != nil {
			return nil, fmt.Errorf("failed to check client owner delegation revocation: %w", err)
		}
		result.LinkRevocations["client_owner_delegation"] = delegationRevoked
		if delegationRevoked {
			result.Revoked = true
			result.RevokedEntity = "client_owner_delegation"
			return result, nil
		}
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

	// Check client delegation revocation (if applicable)
	if chain.Client.AuthorizationType == "delegated" && chain.Client.DelegationID != "" {
		delegationRevoked, err := v.revocationChecker.IsDelegationRevoked(ctx, chain.Client.DelegationID)
		if err != nil {
			return nil, fmt.Errorf("failed to check client delegation revocation: %w", err)
		}
		result.LinkRevocations["client_delegation"] = delegationRevoked
		if delegationRevoked {
			result.Revoked = true
			result.RevokedEntity = "client_delegation"
			return result, nil
		}
	}

	return result, nil
}

// validateChainContinuity validates temporal continuity and scope consistency
func (v *AuthorizationChainValidator) validateChainContinuity(chain *AuthorizationChain) error {
	// Check temporal continuity: each level's validity must fall within parent's validity

	// Client owner must be valid during authorizer's validity period
	if chain.ClientOwner.ValidFrom.Before(chain.OwnersAuthorizer.ValidFrom) {
		return &AgentAuthError{
			Code:    "temporal_discontinuity",
			Message: "Client owner validity starts before authorizer's validity",
		}
	}
	if chain.ClientOwner.ValidUntil.After(chain.OwnersAuthorizer.ValidUntil) {
		return &AgentAuthError{
			Code:    "temporal_discontinuity",
			Message: "Client owner validity extends beyond authorizer's validity",
		}
	}

	// Client must be valid during owner's validity period
	if chain.Client.ValidFrom.Before(chain.ClientOwner.ValidFrom) {
		return &AgentAuthError{
			Code:    "temporal_discontinuity",
			Message: "Client validity starts before owner's validity",
		}
	}
	if chain.Client.ValidUntil.After(chain.ClientOwner.ValidUntil) {
		return &AgentAuthError{
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

	if !companyInfo.Active {
		return false, nil
	}

	// Check if entity is a managing director of the company
	director, err := v.commercialRegisterClient.VerifyManagingDirector(ctx, registerRef, entityID)
	if err != nil {
		// If director verification fails, check if entity is the company itself
		if companyInfo.CompanyID == entityID || companyInfo.RegistrationNumber == entityID {
			return companyInfo.Active, nil
		}
		return false, err
	}

	return director.Active, nil
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
	Valid               bool                    `json:"valid"`
	ValidationTime      time.Time               `json:"validation_time"`
	ValidatedChainDepth int                     `json:"validated_chain_depth"`
	LinkValidations     []*LinkValidationResult `json:"link_validations"`
	ChainIntegrityValid bool                    `json:"chain_integrity_valid"`
	ChainIntegrityHash  string                  `json:"chain_integrity_hash"`
	RevocationStatus    *RevocationCheckResult  `json:"revocation_status,omitempty"`
	FailureReason       string                  `json:"failure_reason,omitempty"`
	Warnings            []string                `json:"warnings,omitempty"`
}

// LinkValidationResult represents validation result for a single authorization link
type LinkValidationResult struct {
	Level         int             `json:"level"` // 1=Authorizer, 2=Owner, 3=Client
	EntityID      string          `json:"entity_id"`
	EntityRole    string          `json:"entity_role"`
	Valid         bool            `json:"valid"`
	Checks        map[string]bool `json:"checks"`
	FailureReason string          `json:"failure_reason,omitempty"`
	Warnings      []string        `json:"warnings,omitempty"`
}

// RevocationCheckResult represents revocation check results
type RevocationCheckResult struct {
	Checked         bool            `json:"checked"`
	Revoked         bool            `json:"revoked"`
	RevokedEntity   string          `json:"revoked_entity,omitempty"`
	LinkRevocations map[string]bool `json:"link_revocations"`
	Message         string          `json:"message,omitempty"`
}

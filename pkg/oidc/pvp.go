// Package oidc - OpenID Connect PowerVerificationPoint Implementation
// This file implements the PowerVerificationPoint interface for OIDC ID token verification
// Integrates with RFC-0111 Subscription Flow Steps I, III, and VI
package oidc

import (
	"context"
	"fmt"

	"github.com/mauriciomferz/Gauth_go/pkg/gauth"
)

// OIDCPowerVerificationPoint implements the PowerVerificationPoint interface for OIDC ID tokens
// It verifies identity using OpenID Connect ID tokens as proof
type OIDCPowerVerificationPoint struct {
	idTokenService *IDTokenService
	bridge         *IdentityBridge
	requiredACR    string // Minimum ACR value required for verification
}

// OIDCPVPConfig holds configuration for the OIDC PVP
type OIDCPVPConfig struct {
	IDTokenService *IDTokenService
	RequiredACR    string // Optional: minimum ACR (default: "substantial")
}

// NewOIDCPowerVerificationPoint creates a new OIDC-based PowerVerificationPoint
// Parameters:
//   - config: Configuration including ID token service and optional minimum ACR
//
// Returns:
//   - *OIDCPowerVerificationPoint: The configured PVP instance
//   - error: Error if configuration is invalid
func NewOIDCPowerVerificationPoint(config OIDCPVPConfig) (*OIDCPowerVerificationPoint, error) {
	if config.IDTokenService == nil {
		return nil, fmt.Errorf("id token service is required")
	}

	// Default to substantial if not specified
	requiredACR := config.RequiredACR
	if requiredACR == "" {
		requiredACR = "substantial"
	}

	bridge := NewIdentityBridge(config.IDTokenService)

	return &OIDCPowerVerificationPoint{
		idTokenService: config.IDTokenService,
		bridge:         bridge,
		requiredACR:    requiredACR,
	}, nil
}

// VerifyIdentityProof verifies identity using an OIDC ID token
// This implements the PowerVerificationPoint interface
// Parameters:
//   - ctx: Request context
//   - request: Identity proof request containing ID token in proof_data
//
// Returns:
//   - *gauth.IdentityProofResult: Verification result with trust level
//   - error: Error if verification fails
func (p *OIDCPowerVerificationPoint) VerifyIdentityProof(ctx context.Context, request *gauth.IdentityProofRequest) (*gauth.IdentityProofResult, error) {
	// Validate proof method
	if request.ProofMethod != ProofMethodOIDCIDToken && request.ProofMethod != ProofMethodOIDCExternal {
		return nil, fmt.Errorf("unsupported proof method: %s (expected %s or %s)",
			request.ProofMethod, ProofMethodOIDCIDToken, ProofMethodOIDCExternal)
	}

	// Extract ID token from proof data
	idToken, ok := request.ProofData["id_token"].(string)
	if !ok || idToken == "" {
		return &gauth.IdentityProofResult{
			Valid:         false,
			SubjectID:     request.SubjectID,
			FailureReason: "id_token not found or invalid in proof_data",
		}, nil
	}

	// Extract expected audience from proof data
	audience, ok := request.ProofData["audience"].(string)
	if !ok || audience == "" {
		return &gauth.IdentityProofResult{
			Valid:         false,
			SubjectID:     request.SubjectID,
			FailureReason: "audience not found or invalid in proof_data",
		}, nil
	}

	// Convert ID token to identity proof using the bridge
	result, err := p.bridge.ConvertIDTokenToIdentityProof(ctx, idToken, audience)
	if err != nil {
		return &gauth.IdentityProofResult{
			Valid:         false,
			SubjectID:     request.SubjectID,
			FailureReason: fmt.Sprintf("ID token validation failed: %v", err),
		}, nil
	}

	// Check if verification succeeded
	if !result.Valid {
		return result, nil
	}

	// Validate minimum ACR requirement if specified in request
	requiredACR := request.RequiredLevel
	if requiredACR == "" {
		requiredACR = p.requiredACR
	}

	// Map trust level to ACR for validation
	actualACR := p.bridge.trustMapper.MapTrustLevelToACR(result.TrustLevel)

	// Validate minimum trust level
	if !p.bridge.trustMapper.ValidateMinimumTrustLevel(actualACR, requiredACR) {
		return &gauth.IdentityProofResult{
			Valid:      false,
			SubjectID:  request.SubjectID,
			TrustLevel: result.TrustLevel,
			FailureReason: fmt.Sprintf("insufficient trust level: got %s (ACR: %s), required %s",
				result.TrustLevel, actualACR, requiredACR),
		}, nil
	}

	// Verify subject ID matches if provided
	if request.SubjectID != "" && result.SubjectID != request.SubjectID {
		return &gauth.IdentityProofResult{
			Valid:     false,
			SubjectID: request.SubjectID,
			FailureReason: fmt.Sprintf("subject ID mismatch: expected %s, got %s",
				request.SubjectID, result.SubjectID),
		}, nil
	}

	return result, nil
}

// GetSupportedProofMethods returns the proof methods supported by this PVP
// Returns:
//   - []string: List of supported proof methods
func (p *OIDCPowerVerificationPoint) GetSupportedProofMethods() []string {
	return []string{ProofMethodOIDCIDToken, ProofMethodOIDCExternal}
}

// GetRequiredACR returns the minimum ACR required by this PVP
// Returns:
//   - string: Minimum ACR value
func (p *OIDCPowerVerificationPoint) GetRequiredACR() string {
	return p.requiredACR
}

// SetRequiredACR updates the minimum ACR requirement
// Parameters:
//   - acr: New minimum ACR value
func (p *OIDCPowerVerificationPoint) SetRequiredACR(acr string) {
	p.requiredACR = acr
}

// ValidateProofData validates that the proof data contains required fields for OIDC verification
// Parameters:
//   - proofData: The proof data map to validate
//
// Returns:
//   - error: Validation error, or nil if valid
func (p *OIDCPowerVerificationPoint) ValidateProofData(proofData map[string]interface{}) error {
	if proofData == nil {
		return fmt.Errorf("proof_data is required")
	}

	idToken, ok := proofData["id_token"].(string)
	if !ok || idToken == "" {
		return fmt.Errorf("id_token is required in proof_data")
	}

	audience, ok := proofData["audience"].(string)
	if !ok || audience == "" {
		return fmt.Errorf("audience is required in proof_data")
	}

	return nil
}

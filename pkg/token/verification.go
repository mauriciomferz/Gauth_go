package token

import (
	"context"
	"fmt"
	"time"
)

// # Licensing
//
// This file is part of the GAuth project and is licensed under the Apache License 2.0.
// It incorporates code and concepts from:
//   - OAuth 2.0 and OpenID Connect (Apache 2.0 License)
//   - Model Context Protocol (MIT License)
// See the LICENSE file in the project root for details.

// VerificationSystem handles comprehensive token verification
type VerificationSystem interface {
	// VerifyPowerValidity checks if the token's power of attorney is valid
	VerifyPowerValidity(ctx context.Context, token *EnhancedToken) error

	// VerifyPrincipalStatus checks the legal status of the principal
	VerifyPrincipalStatus(ctx context.Context, token *EnhancedToken) error

	// ValidateAttestation verifies attestation authenticity
	ValidateAttestation(ctx context.Context, attestation *Attestation) error

	// CheckRevocationStatus verifies token hasn't been revoked
	CheckRevocationStatus(ctx context.Context, token *EnhancedToken) error

	// VerifyVersionHistory validates version chain integrity
	VerifyVersionHistory(ctx context.Context, token *EnhancedToken) error
}

// StandardVerificationSystem implements VerificationSystem
type StandardVerificationSystem struct {
	store       EnhancedStore
	regVerifier RegistryVerifier
}

// RegistryVerifier handles commercial registry verification
type RegistryVerifier interface {
	VerifyRegistration(ctx context.Context, info *RegistrationInfo) error
	ValidateLegalStatus(ctx context.Context, ownerInfo *OwnerInfo) error
}

// NewStandardVerificationSystem creates a new verification system
func NewStandardVerificationSystem(store EnhancedStore, regVerifier RegistryVerifier) *StandardVerificationSystem {
	return &StandardVerificationSystem{
		store:       store,
		regVerifier: regVerifier,
	}
}

// VerifyPowerValidity implements VerificationSystem
func (v *StandardVerificationSystem) VerifyPowerValidity(ctx context.Context, token *EnhancedToken) error {
	// Check token expiration
	if token.Token != nil && time.Now().After(token.Token.ExpiresAt) {
		return fmt.Errorf("token has expired")
	}

	// Verify owner authorization
	if err := v.verifyOwnerAuthorization(ctx, token.Owner); err != nil {
		return fmt.Errorf("owner authorization invalid: %w", err)
	}

	// Verify AI metadata
	if err := v.verifyAIMetadata(ctx, token.AI); err != nil {
		return fmt.Errorf("AI metadata invalid: %w", err)
	}

	// Verify attestations
	for _, att := range token.Attestations {
		if err := v.ValidateAttestation(ctx, &att); err != nil {
			return fmt.Errorf("attestation invalid: %w", err)
		}
	}

	return nil
}

// VerifyPrincipalStatus implements VerificationSystem
func (v *StandardVerificationSystem) VerifyPrincipalStatus(ctx context.Context, token *EnhancedToken) error {
	// Verify owner's legal status
	if err := v.regVerifier.ValidateLegalStatus(ctx, token.Owner); err != nil {
		return fmt.Errorf("principal status invalid: %w", err)
	}

	// Verify registration info if available
	if token.Owner.RegistrationInfo != nil {
		if err := v.regVerifier.VerifyRegistration(ctx, token.Owner.RegistrationInfo); err != nil {
			return fmt.Errorf("registration verification failed: %w", err)
		}
	}

	return nil
}

// ValidateAttestation implements VerificationSystem
func (v *StandardVerificationSystem) ValidateAttestation(ctx context.Context, attestation *Attestation) error {
	// Verify attestation hasn't expired
	if time.Since(attestation.AttestationDate) > 24*time.Hour*365 { // 1 year
		return fmt.Errorf("attestation has expired")
	}

	// Verify attester is authorized
	if err := v.verifyAttester(ctx, attestation.AttesterID); err != nil {
		return fmt.Errorf("attester verification failed: %w", err)
	}

	// Verify evidence
	if err := v.verifyEvidence(ctx, attestation.Evidence); err != nil {
		return fmt.Errorf("evidence verification failed: %w", err)
	}

	return nil
}

// CheckRevocationStatus implements VerificationSystem
func (v *StandardVerificationSystem) CheckRevocationStatus(_ context.Context, token *EnhancedToken) error {
	// Check if token has been revoked using canonical field
	if token.Token != nil && token.Token.RevocationStatus != nil && !token.Token.RevocationStatus.RevokedAt.IsZero() {
		return fmt.Errorf("token has been revoked")
	}
	return nil
}

// VerifyVersionHistory implements VerificationSystem
func (v *StandardVerificationSystem) VerifyVersionHistory(_ context.Context, token *EnhancedToken) error {
	if len(token.Versions) == 0 {
		return fmt.Errorf("token has no version history")
	}

	// Verify version chain integrity
	for i := 1; i < len(token.Versions); i++ {
		curr := token.Versions[i]
		prev := token.Versions[i-1]

		// Verify version sequence
		if curr.Version != prev.Version+1 {
			return fmt.Errorf("invalid version sequence")
		}

		// Verify timestamps
		if !curr.UpdatedAt.After(prev.UpdatedAt) {
			return fmt.Errorf("invalid version timestamp")
		}
	}

	return nil
}

// RegVerifier returns the registry verifier (for subscription.go compatibility)
func (v *StandardVerificationSystem) RegVerifier() RegistryVerifier {
	return v.regVerifier
}

// Helper methods

func (v *StandardVerificationSystem) verifyOwnerAuthorization(_ context.Context, owner *OwnerInfo) error {
	// Verify authorization document
	if owner.AuthorizationRef == "" {
		return fmt.Errorf("missing authorization reference")
	}

	// Additional authorization checks would go here
	return nil
}

func (v *StandardVerificationSystem) verifyAIMetadata(_ context.Context, ai *AIMetadata) error {
	if ai == nil {
		return fmt.Errorf("missing AI metadata")
	}

	// Verify AI capabilities
	if len(ai.Capabilities) == 0 {
		return fmt.Errorf("no AI capabilities specified")
	}

	// Verify delegation guidelines
	if len(ai.DelegationGuidelines) == 0 {
		return fmt.Errorf("no delegation guidelines specified")
	}

	return nil
}

func (v *StandardVerificationSystem) verifyAttester(_ context.Context, _ string) error {
	// Verify attester credentials would go here
	return nil
}

func (v *StandardVerificationSystem) verifyEvidence(_ context.Context, _ string) error {
	// Verify cryptographic evidence would go here
	return nil
}

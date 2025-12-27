package gauthplus

import (
	"context"
	"strings"
	"time"
)

// DefaultPrincipalVerifier implements PrincipalVerifier
type DefaultPrincipalVerifier struct {
	// In a real system, this would have a reference to a UserStore or Identity Service
}

// NewDefaultPrincipalVerifier creates a new default principal verifier
func NewDefaultPrincipalVerifier() *DefaultPrincipalVerifier {
	return &DefaultPrincipalVerifier{}
}

// VerifyPrincipal verifies a principal's status and determines their type
func (v *DefaultPrincipalVerifier) VerifyPrincipal(ctx context.Context, principalID string) (*PrincipalStatusResult, error) {
	// Simple implementation for now:
	// We distinguish humans from AI by ID prefix or naming convention if real store is missing.
	// In production, this would query the identity provider.

	res := &PrincipalStatusResult{
		PrincipalID: principalID,
		Valid:       true,
		Status:      "active",
		VerifiedAt:  time.Now(),
	}

	upperID := strings.ToUpper(principalID)
	if strings.Contains(upperID, "AI") || strings.Contains(upperID, "AGENT") || strings.Contains(upperID, "BOT") {
		res.EntityType = "ai_agent"
	} else {
		res.EntityType = "human"
	}

	return res, nil
}

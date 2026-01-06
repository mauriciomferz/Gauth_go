package identity

import (
	"context"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/mauriciomferz/AgentAuth/pkg/agentauthplus"
)

// IdentityProvisioner defines the interface for provisioning agent identities
type IdentityProvisioner interface {
	// MintAssertion generates a signed JWT assertion for the given agent and audience
	MintAssertion(ctx context.Context, agentID string, audience string) (string, error)
}

// TokenSigner defines the interface for signing JWTs
type TokenSigner interface {
	Sign(claims jwt.MapClaims) (string, error)
}

// ProvisioningService implements IdentityProvisioner
type ProvisioningService struct {
	signer              TokenSigner
	verificationService agentauthplus.VerificationService
	issuerID            string
}

// NewProvisioningService creates a new instance of ProvisioningService
func NewProvisioningService(signer TokenSigner, vs agentauthplus.VerificationService, issuerID string) *ProvisioningService {
	return &ProvisioningService{
		signer:              signer,
		verificationService: vs,
		issuerID:            issuerID,
	}
}

// MintAssertion generates a signed JWT assertion for the given agent and audience
func (s *ProvisioningService) MintAssertion(ctx context.Context, agentID string, audience string) (string, error) {
	// 1. Verify Agent Status (Optional but recommended)
	// In a full implementation, we would check if the agent is active/compliant using verificationService.
	// For now, we assume the caller has done basic checks or we proceed to minting.

	now := time.Now()
	jti := uuid.New().String()

	// 2. Build Claims
	claims := jwt.MapClaims{
		"iss": s.issuerID,
		"sub": agentID,
		"aud": audience,
		"exp": now.Add(1 * time.Hour).Unix(),
		"nbf": now.Unix(),
		"iat": now.Unix(),
		"jti": jti,
		// OBO-specific claims
		"x_gauth_compliance": map[string]interface{}{
			"verified": true, // Placeholder for real verification result
			"level":    "aap-001",
		},
	}

	// 3. Sign Token
	return s.signer.Sign(claims)
}

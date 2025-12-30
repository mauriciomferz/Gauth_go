// Package oidc - Identity Bridge
// Converts OIDC ID tokens to AgentAuth identity proof structures
package oidc

import (
	"context"
	"fmt"
	"time"

	"github.com/mauriciomferz/AgentAuth/pkg/gauth"
)

// IdentityBridge converts between OIDC and AgentAuth identity structures
// This enables using OIDC ID tokens as identity proofs in RFC-0111 flow
type IdentityBridge struct {
	idTokenService *IDTokenService
	trustMapper    *TrustLevelMapper
}

// NewIdentityBridge creates a new identity bridge
func NewIdentityBridge(idTokenService *IDTokenService) *IdentityBridge {
	return &IdentityBridge{
		idTokenService: idTokenService,
		trustMapper:    NewTrustLevelMapper(),
	}
}

// ConvertIDTokenToIdentityProof converts OIDC ID token to AgentAuth IdentityProofResult
// This is the core bridge function enabling OIDC in RFC-0111 Steps I, III, VI
func (b *IdentityBridge) ConvertIDTokenToIdentityProof(
	ctx context.Context,
	idToken string,
	expectedAudience string,
) (*gauth.IdentityProofResult, error) {
	// Validate ID token
	claims, err := b.idTokenService.ValidateIDToken(ctx, idToken, expectedAudience)
	if err != nil {
		return &gauth.IdentityProofResult{
			Valid:         false,
			FailureReason: fmt.Sprintf("ID token validation failed: %v", err),
		}, nil
	}

	// Extract identity information
	identity := claims.Subject
	if claims.Name != "" {
		identity = claims.Name
	}
	if claims.LegalEntityName != "" {
		identity = claims.LegalEntityName
	}

	// Map OIDC ACR to AgentAuth TrustLevel
	trustLevel := b.trustMapper.MapACRToTrustLevel(claims.ACR)

	// Build successful identity proof result
	return &gauth.IdentityProofResult{
		Valid:         true,
		SubjectID:     claims.Subject,
		Identity:      identity,
		VerifiedAt:    time.Now(),
		TrustLevel:    trustLevel,
		FailureReason: "",
	}, nil
}

// ConvertIdentityProofToIDToken converts AgentAuth identity proof to OIDC ID token
// This enables AgentAuth to issue OIDC-compliant ID tokens
func (b *IdentityBridge) ConvertIdentityProofToIDToken(
	ctx context.Context,
	proof *gauth.IdentityProofResult,
	audience []string,
	identityType string,
) (string, error) {
	if !proof.Valid {
		return "", fmt.Errorf("cannot issue ID token for invalid identity proof")
	}

	// Build additional claims from proof
	additionalClaims := map[string]interface{}{
		"name": proof.Identity,
	}

	// Create ID token
	return b.idTokenService.CreateIDTokenFromIdentity(
		ctx,
		proof.SubjectID,
		audience,
		identityType,
		proof.TrustLevel,
		additionalClaims,
	)
}

// TrustLevelMapper maps between OIDC ACR and AgentAuth TrustLevel
type TrustLevelMapper struct {
	acrToTrustLevel map[string]string
	trustLevelToACR map[string]string
}

// NewTrustLevelMapper creates a new trust level mapper with default mappings
func NewTrustLevelMapper() *TrustLevelMapper {
	mapper := &TrustLevelMapper{
		acrToTrustLevel: make(map[string]string),
		trustLevelToACR: make(map[string]string),
	}

	// Initialize default mappings from DefaultACRMappings
	for _, mapping := range DefaultACRMappings {
		mapper.acrToTrustLevel[mapping.ACR] = mapping.AgentAuthTrustLevel
	}

	// Reverse mappings (trust level → ACR)
	mapper.trustLevelToACR[trustLevelLow] = "1"
	mapper.trustLevelToACR[trustLevelSubstantial] = trustLevelSubstantial
	mapper.trustLevelToACR[trustLevelHigh] = trustLevelHigh

	return mapper
}

// MapACRToTrustLevel maps OIDC ACR value to AgentAuth trust level
func (m *TrustLevelMapper) MapACRToTrustLevel(acr string) string {
	if trustLevel, exists := m.acrToTrustLevel[acr]; exists {
		return trustLevel
	}

	// Default fallback: low trust
	return trustLevelLow
}

// MapTrustLevelToACR maps AgentAuth trust level to OIDC ACR value
func (m *TrustLevelMapper) MapTrustLevelToACR(trustLevel string) string {
	if acr, exists := m.trustLevelToACR[trustLevel]; exists {
		return acr
	}

	// Default fallback: ACR level 0
	return "0"
}

// AddCustomMapping adds a custom ACR → TrustLevel mapping
func (m *TrustLevelMapper) AddCustomMapping(acr string, trustLevel string) {
	m.acrToTrustLevel[acr] = trustLevel
}

// ValidateMinimumTrustLevel validates if ACR meets minimum trust requirement
func (m *TrustLevelMapper) ValidateMinimumTrustLevel(acr string, required string) bool {
	actualLevel := m.MapACRToTrustLevel(acr)
	return m.compareTrustLevels(actualLevel, required) >= 0
}

// compareTrustLevels compares two trust levels
// Returns: -1 if actual < required, 0 if equal, 1 if actual > required
func (m *TrustLevelMapper) compareTrustLevels(actual string, required string) int {
	levels := map[string]int{
		"low":         1,
		"substantial": 2,
		"high":        3,
	}

	actualValue, actualExists := levels[actual]
	requiredValue, requiredExists := levels[required]

	if !actualExists || !requiredExists {
		return -1
	}

	if actualValue < requiredValue {
		return -1
	} else if actualValue == requiredValue {
		return 0
	}
	return 1
}

// ExtractEntityTypeFromClaims extracts entity type from ID token claims
func ExtractEntityTypeFromClaims(claims *IDTokenClaims) string {
	if claims.EntityType != "" {
		return claims.EntityType
	}

	// Infer from claims
	if claims.LegalEntityName != "" {
		return "legal_entity"
	}

	return "natural_person"
}

// ExtractProofDataFromClaims extracts proof data map from ID token claims
func ExtractProofDataFromClaims(claims *IDTokenClaims) map[string]interface{} {
	proofData := make(map[string]interface{})

	if claims.Name != "" {
		proofData["name"] = claims.Name
	}
	if claims.Email != "" {
		proofData["email"] = claims.Email
		proofData["email_verified"] = claims.EmailVerified
	}
	if claims.EntityID != "" {
		proofData["entity_id"] = claims.EntityID
	}
	if claims.LegalEntityName != "" {
		proofData["legal_entity_name"] = claims.LegalEntityName
	}
	if claims.Jurisdiction != "" {
		proofData["jurisdiction"] = claims.Jurisdiction
	}
	if claims.TSPName != "" {
		proofData["tsp_name"] = claims.TSPName
	}
	if claims.TSPID != "" {
		proofData["tsp_id"] = claims.TSPID
	}
	if claims.ACR != "" {
		proofData["acr"] = claims.ACR
	}
	if len(claims.AMR) > 0 {
		proofData["amr"] = claims.AMR
	}

	return proofData
}

// BuildIdentityProofRequestFromIDToken creates IdentityProofRequest from ID token
// This enables using OIDC ID tokens in existing RFC-0111 flow
func BuildIdentityProofRequestFromIDToken(
	idToken string,
	idTokenService *IDTokenService,
	audience string,
) (*gauth.IdentityProofRequest, error) {
	// Parse ID token without full validation
	claims, err := idTokenService.ValidateIDToken(context.Background(), idToken, audience)
	if err != nil {
		return nil, fmt.Errorf("failed to parse ID token: %w", err)
	}

	// Extract entity type
	entityType := ExtractEntityTypeFromClaims(claims)

	// Extract proof data
	proofData := ExtractProofDataFromClaims(claims)

	// Map ACR to required level
	mapper := NewTrustLevelMapper()
	trustLevel := mapper.MapACRToTrustLevel(claims.ACR)

	return &gauth.IdentityProofRequest{
		SubjectID:     claims.Subject,
		IdentityType:  entityType,
		ProofMethod:   ProofMethodOIDCIDToken,
		ProofData:     proofData,
		RequiredLevel: trustLevel,
	}, nil
}

// ValidateIDTokenForIdentityProof validates ID token for use as identity proof
// Performs additional checks beyond standard OIDC validation
func ValidateIDTokenForIdentityProof(
	ctx context.Context,
	idToken string,
	idTokenService *IDTokenService,
	expectedAudience string,
	minTrustLevel string,
) error {
	// Standard OIDC validation
	claims, err := idTokenService.ValidateIDToken(ctx, idToken, expectedAudience)
	if err != nil {
		return fmt.Errorf("ID token validation failed: %w", err)
	}

	// Validate minimum trust level
	mapper := NewTrustLevelMapper()
	if !mapper.ValidateMinimumTrustLevel(claims.ACR, minTrustLevel) {
		return fmt.Errorf("insufficient trust level: ACR %s does not meet minimum %s",
			claims.ACR, minTrustLevel)
	}

	// Validate subject exists
	if claims.Subject == "" {
		return fmt.Errorf("subject (sub) claim is required")
	}

	return nil
}

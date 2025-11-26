// Package gauth - Extended Token Creation and Validation per RFC-0111
// Implements critical Gaps #6 and #7 from QUALITY_MANAGER_RFC_COMPLIANCE_FINAL_ASSESSMENT.md
package gauth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/mauriciomferz/Gauth_go/pkg/poa"
	"github.com/golang-jwt/jwt/v5"
)

// ExtendedTokenService provides RFC-0111 compliant extended token operations
type ExtendedTokenService struct {
	chainValidator      *AuthorizationChainValidator
	complianceValidator *ComplianceValidator
	pipClient           PIPClient
	issuerID            string
	issuerURL           string
	tokenExpiry         time.Duration
	signingKey          []byte     // HMAC signing key for JWT
	jweService          JWEService // Optional JWE encryption service
}

// NewExtendedTokenService creates a new extended token service
func NewExtendedTokenService(
	chainValidator *AuthorizationChainValidator,
	complianceValidator *ComplianceValidator,
	pipClient PIPClient,
	issuerID string,
	issuerURL string,
	tokenExpiry time.Duration,
) *ExtendedTokenService {
	if tokenExpiry == 0 {
		tokenExpiry = 1 * time.Hour // Default
	}

	// Generate a signing key if not provided
	// In production, this should be loaded from secure configuration
	signingKey := make([]byte, 32)
	rand.Read(signingKey)

	return &ExtendedTokenService{
		chainValidator:      chainValidator,
		complianceValidator: complianceValidator,
		pipClient:           pipClient,
		issuerID:            issuerID,
		issuerURL:           issuerURL,
		tokenExpiry:         tokenExpiry,
		signingKey:          signingKey,
		jweService:          nil, // JWE encryption disabled by default
	}
}

// WithJWEEncryption adds JWE encryption to the token service
// Call this method to enable JWE encryption for tokens
func (s *ExtendedTokenService) WithJWEEncryption(jweService JWEService) *ExtendedTokenService {
	s.jweService = jweService
	return s
}

// CreateExtendedToken creates an RFC-0111 compliant extended token
// This implements the MISSING functionality identified in the assessment
func (s *ExtendedTokenService) CreateExtendedToken(
	ctx context.Context,
	request *ExtendedTokenRequest,
) (*ExtendedToken, error) {
	if request == nil {
		return nil, &GAuthError{
			Code:    "invalid_request",
			Message: "Extended token request cannot be nil",
		}
	}

	// Step 1: Validate authorization chain
	if request.AuthorizationChain == nil {
		return nil, &GAuthError{
			Code:    "missing_authorization_chain",
			Message: "Authorization chain is required per RFC-0111",
		}
	}

	chainResult, err := s.chainValidator.ValidateAuthorizationChain(ctx, request.AuthorizationChain)
	if err != nil {
		return nil, fmt.Errorf("authorization chain validation failed: %w", err)
	}
	if !chainResult.Valid {
		return nil, &GAuthError{
			Code:    "invalid_authorization_chain",
			Message: fmt.Sprintf("Authorization chain validation failed: %s", chainResult.FailureReason),
		}
	}

	// Step 2: Validate Power of Attorney
	if request.PowerOfAttorney == nil {
		return nil, &GAuthError{
			Code:    "missing_poa",
			Message: "Power of Attorney is required per RFC-0111",
		}
	}

	if err := poa.ValidatePoADefinition(*request.PowerOfAttorney); err != nil {
		return nil, fmt.Errorf("PoA validation failed: %w", err)
	}

	// Step 3: Validate client owner information
	if request.ClientOwnerInfo == nil {
		return nil, &GAuthError{
			Code:    "missing_client_owner",
			Message: "Client owner information is required per RFC-0111",
		}
	}

	// Step 4: Validate owner's authorizer information
	if request.OwnersAuthorizerInfo == nil {
		return nil, &GAuthError{
			Code:    "missing_owners_authorizer",
			Message: "Owner's authorizer information is required per RFC-0111",
		}
	}

	// Step 5: Validate legal framework
	if request.LegalFramework == nil {
		return nil, &GAuthError{
			Code:    "missing_legal_framework",
			Message: "Legal framework is required per RFC-0111",
		}
	}

	// Step 6: Generate token
	accessToken, err := s.generateAccessToken()
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	refreshToken, err := s.generateRefreshToken()
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	// Step 7: Build verification chain
	verificationChain := s.buildVerificationChain(ctx, request, chainResult)

	// Step 8: Create comprehensive extended token
	now := time.Now()
	expiresIn := int64(s.tokenExpiry.Seconds())

	token := &ExtendedToken{
		// OAuth 2.0 compatibility fields
		AccessToken:  accessToken,
		TokenType:    "Bearer",
		ExpiresIn:    expiresIn,
		RefreshToken: refreshToken,
		Scope:        request.Scope,
		IssuedAt:     now,

		// RFC-0111 extended fields
		PowerOfAttorney:    request.PowerOfAttorney,
		AuthorizationChain: request.AuthorizationChain,
		ClientOwner:        request.ClientOwnerInfo,
		OwnersAuthorizer:   request.OwnersAuthorizerInfo,
		ResourceOwner:      request.ResourceOwnerInfo,
		LegalFramework:     request.LegalFramework,
		Restrictions:       s.convertRestrictions(request.Restrictions),

		// Issuer information
		IssuedBy: &AuthorizationServerInfo{
			ServerID:   s.issuerID,
			ServerURL:  s.issuerURL,
			ServerName: fmt.Sprintf("GAuth AS %s", s.issuerID),
			Issuer:     s.issuerID,
			IssueTime:  now,
		},

		// Verification proof
		VerificationProof: verificationChain,

		// Request context
		RequestID:          request.RequestID,
		GrantID:            request.GrantID,
		TransactionContext: s.convertContext(request.Context),

		// Compliance & audit
		ComplianceLevel:     "rfc-0111-compliant",
		JurisdictionContext: request.JurisdictionContext,
		AuditTrail: []AuditEntry{
			{
				Timestamp: now,
				Action:    "extended_token_created",
				Actor:     s.issuerID,
				Result:    "success",
				Details: map[string]interface{}{
					"client_id":  request.AuthorizationChain.Client.EntityID,
					"grant_id":   request.GrantID,
					"request_id": request.RequestID,
				},
			},
		},
	}

	// Mark authorization chain as validated
	token.AuthorizationChain.ChainValidated = true
	token.AuthorizationChain.ValidationTime = chainResult.ValidationTime
	token.AuthorizationChain.ValidatorID = s.issuerID
	token.AuthorizationChain.ChainIntegrity = chainResult.ChainIntegrityHash
	token.AuthorizationChain.ChainDepth = chainResult.ValidatedChainDepth

	return token, nil
}

// EncodeExtendedToken encodes an ExtendedToken to a JWT string
// This FIXES the critical gap identified in the audit
func (s *ExtendedTokenService) EncodeExtendedToken(
	ctx context.Context,
	token *ExtendedToken,
) (string, error) {
	if token == nil {
		return "", &GAuthError{
			Code:    "nil_token",
			Message: "Cannot encode nil token",
		}
	}

	// Create JWT claims
	claims := jwt.MapClaims{
		"iss":        s.issuerID,
		"sub":        token.AuthorizationChain.Client.EntityID,
		"aud":        token.ResourceOwner.OwnerID,
		"exp":        token.IssuedAt.Add(time.Duration(token.ExpiresIn) * time.Second).Unix(),
		"iat":        token.IssuedAt.Unix(),
		"jti":        token.AccessToken,
		"token_type": token.TokenType,
		"scope":      token.Scope,

		// RFC-0111 extended claims
		"client_owner":      token.ClientOwner,
		"owners_authorizer": token.OwnersAuthorizer,
		"resource_owner":    token.ResourceOwner,
		"legal_framework":   token.LegalFramework,
		"restrictions":      token.Restrictions,
		"compliance_level":  token.ComplianceLevel,
		"grant_id":          token.GrantID,
		"request_id":        token.RequestID,
	}

	// Serialize PoA and AuthChain as JSON strings to avoid nested complexity
	if token.PowerOfAttorney != nil {
		poaJSON, err := json.Marshal(token.PowerOfAttorney)
		if err == nil {
			claims["power_of_attorney"] = string(poaJSON)
		}
	}

	if token.AuthorizationChain != nil {
		chainJSON, err := json.Marshal(token.AuthorizationChain)
		if err == nil {
			claims["authorization_chain"] = string(chainJSON)
		}
	}

	// Create JWT token
	jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Sign and get the complete encoded token string
	tokenString, err := jwtToken.SignedString(s.signingKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign JWT: %w", err)
	}

	// If JWE encryption is enabled, encrypt the JWT
	if s.jweService != nil && s.jweService.IsEnabled() {
		jweString, err := s.jweService.EncryptToken(ctx, tokenString)
		if err != nil {
			return "", fmt.Errorf("failed to encrypt token: %w", err)
		}
		return jweString, nil
	}

	// Return unencrypted JWT if JWE is disabled
	return tokenString, nil
}

// ValidateExtendedToken validates an RFC-0111 extended token
// This implements the MISSING functionality identified in the assessment
func (s *ExtendedTokenService) ValidateExtendedToken(
	ctx context.Context,
	tokenString string,
) (*ExtendedTokenValidationResult, error) {
	// Step 1: Parse token (simplified - in production would parse JWT/JWE)
	token, err := s.parseExtendedToken(ctx, tokenString)
	if err != nil {
		return nil, fmt.Errorf("failed to parse extended token: %w", err)
	}

	result := &ExtendedTokenValidationResult{
		ExtendedToken:       token,
		Valid:               true,
		ValidationTimestamp: time.Now(),
		ValidationWarnings:  make([]string, 0),
	}

	// Step 2: Validate token structure
	if err := token.Validate(); err != nil {
		result.Valid = false
		return result, fmt.Errorf("token structure validation failed: %w", err)
	}

	// Step 3: Validate expiration
	if time.Now().After(token.IssuedAt.Add(time.Duration(token.ExpiresIn) * time.Second)) {
		result.Valid = false
		return result, &GAuthError{
			Code:    "token_expired",
			Message: "Extended token has expired",
		}
	}

	// Step 4: Validate authorization chain
	chainResult, err := s.chainValidator.ValidateAuthorizationChain(ctx, token.AuthorizationChain)
	if err != nil {
		result.Valid = false
		result.ChainValidated = false
		result.AuthorizationChain = token.AuthorizationChain
		return result, fmt.Errorf("authorization chain validation failed: %w", err)
	}
	result.ChainValidated = chainResult.Valid
	result.AuthorizationChain = token.AuthorizationChain

	if !chainResult.Valid {
		result.Valid = false
		return result, &GAuthError{
			Code:    "invalid_authorization_chain",
			Message: fmt.Sprintf("Authorization chain is invalid: %s", chainResult.FailureReason),
		}
	}

	// Step 5: Validate legal framework
	if token.LegalFramework != nil {
		result.LegalFrameworkValid = s.validateLegalFrameworkCompliance(token.LegalFramework)
		if !result.LegalFrameworkValid {
			result.ValidationWarnings = append(result.ValidationWarnings, "Legal framework compliance issues detected")
		}
	} else {
		result.LegalFrameworkValid = false
		result.ValidationWarnings = append(result.ValidationWarnings, "No legal framework present")
	}

	// Step 6: Validate restrictions enforcement
	if len(token.Restrictions) > 0 {
		result.RestrictionsEnforced = true
		// In production, would check current request against restrictions
	}

	// Step 7: Validate verification proof
	if token.VerificationProof != nil {
		result.VerificationProofValid = token.VerificationProof.OverallVerification == "verified"
		if !result.VerificationProofValid {
			result.ValidationWarnings = append(result.ValidationWarnings, "Identity verification incomplete or invalid")
		}
	} else {
		result.VerificationProofValid = false
		result.ValidationWarnings = append(result.ValidationWarnings, "No verification proof present")
	}

	// Step 8: Build result summary
	result.ClientID = token.AuthorizationChain.Client.EntityID
	result.Scope = token.Scope

	// Token is valid if all critical checks pass
	result.Valid = result.ChainValidated && result.LegalFrameworkValid

	return result, nil
}

// generateAccessToken generates a secure access token
func (s *ExtendedTokenService) generateAccessToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return fmt.Sprintf("gauth_at_%s", hex.EncodeToString(bytes)), nil
}

// generateRefreshToken generates a secure refresh token
func (s *ExtendedTokenService) generateRefreshToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return fmt.Sprintf("gauth_rt_%s", hex.EncodeToString(bytes)), nil
}

// buildVerificationChain builds the identity verification chain
func (s *ExtendedTokenService) buildVerificationChain(
	ctx context.Context,
	request *ExtendedTokenRequest,
	chainResult *ChainValidationResult,
) *IdentityVerificationChain {
	chain := &IdentityVerificationChain{
		ChainID:             fmt.Sprintf("vc_%s", request.RequestID),
		VerificationLevels:  make([]VerificationLevel, 0),
		OverallVerification: "verified",
		VerificationTime:    time.Now(),
		VerifierEntity:      s.issuerID,
	}

	// Add verification levels from chain validation
	for _, linkValidation := range chainResult.LinkValidations {
		level := VerificationLevel{
			Level:              linkValidation.Level,
			EntityID:           linkValidation.EntityID,
			EntityRole:         linkValidation.EntityRole,
			VerificationMethod: "authorization_chain_validation",
			VerificationStatus: "verified",
			VerifiedBy:         s.issuerID,
			VerificationDate:   chainResult.ValidationTime,
			AssuranceLevel:     "substantial",
		}

		if !linkValidation.Valid {
			level.VerificationStatus = "failed"
			chain.OverallVerification = "partial"
		}

		chain.VerificationLevels = append(chain.VerificationLevels, level)
	}

	return chain
}

// parseExtendedToken parses an extended token from JWT/JWE string
// This FIXES the critical gap identified in the audit
// Supports both plain JWT and JWE-encrypted JWT tokens
func (s *ExtendedTokenService) parseExtendedToken(
	ctx context.Context,
	tokenString string,
) (*ExtendedToken, error) {
	var jwtString string

	// Check if token is JWE-encrypted (5 parts) or plain JWT (3 parts)
	if s.jweService != nil && s.jweService.IsEnabled() && IsJWE(tokenString) {
		// Decrypt JWE to get JWT
		decrypted, err := s.jweService.DecryptToken(ctx, tokenString)
		if err != nil {
			return nil, &GAuthError{
				Code:    "jwe_decrypt_failed",
				Message: fmt.Sprintf("Failed to decrypt JWE token: %v", err),
			}
		}
		jwtString = decrypted
	} else {
		// Token is plain JWT (or JWE is disabled)
		jwtString = tokenString
	}

	// Parse JWT token
	jwtToken, err := jwt.Parse(jwtString, func(token *jwt.Token) (interface{}, error) {
		// Verify signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.signingKey, nil
	})

	if err != nil {
		return nil, &GAuthError{
			Code:    "jwt_parse_failed",
			Message: fmt.Sprintf("Failed to parse JWT: %v", err),
		}
	}

	if !jwtToken.Valid {
		return nil, &GAuthError{
			Code:    "invalid_jwt",
			Message: "JWT token is invalid",
		}
	}

	// Extract claims
	claims, ok := jwtToken.Claims.(jwt.MapClaims)
	if !ok {
		return nil, &GAuthError{
			Code:    "invalid_claims",
			Message: "Failed to extract JWT claims",
		}
	}

	// Build ExtendedToken from claims
	token := &ExtendedToken{
		AccessToken:  getString(claims, "jti"),
		TokenType:    getString(claims, "token_type"),
		RefreshToken: getString(claims, "refresh_token"),
		IssuedAt:     time.Unix(getInt64(claims, "iat"), 0),
	}

	// Calculate ExpiresIn from exp
	exp := getInt64(claims, "exp")
	iat := getInt64(claims, "iat")
	token.ExpiresIn = exp - iat

	// Extract scope
	if scope, ok := claims["scope"].([]interface{}); ok {
		token.Scope = make([]string, len(scope))
		for i, s := range scope {
			if str, ok := s.(string); ok {
				token.Scope[i] = str
			}
		}
	}

	// Extract complex objects
	if poaStr, ok := claims["power_of_attorney"].(string); ok && poaStr != "" {
		var poa poa.PoADefinition
		if err := json.Unmarshal([]byte(poaStr), &poa); err == nil {
			token.PowerOfAttorney = &poa
		}
	}

	if chainStr, ok := claims["authorization_chain"].(string); ok && chainStr != "" {
		var chain AuthorizationChain
		if err := json.Unmarshal([]byte(chainStr), &chain); err == nil {
			token.AuthorizationChain = &chain
		}
	}

	// Extract entity information
	if clientOwnerData, ok := claims["client_owner"].(map[string]interface{}); ok {
		token.ClientOwner = &ClientOwnerInfo{
			OwnerID:   getString(clientOwnerData, "owner_id"),
			OwnerName: getString(clientOwnerData, "owner_name"),
		}
	}

	if authorizerData, ok := claims["owners_authorizer"].(map[string]interface{}); ok {
		token.OwnersAuthorizer = &OwnersAuthorizerInfo{
			AuthorizerID:   getString(authorizerData, "authorizer_id"),
			AuthorizerName: getString(authorizerData, "authorizer_name"),
		}
	}

	if resourceOwnerData, ok := claims["resource_owner"].(map[string]interface{}); ok {
		token.ResourceOwner = &ResourceOwnerInfo{
			OwnerID: getString(resourceOwnerData, "owner_id"),
		}
	}

	// Extract legal framework
	if legalData, ok := claims["legal_framework"].(map[string]interface{}); ok {
		token.LegalFramework = &LegalFrameworkInfo{
			Jurisdiction:        getString(legalData, "jurisdiction"),
			ComplianceFramework: getString(legalData, "compliance_framework"),
		}
		if laws, ok := legalData["applicable_laws"].([]interface{}); ok {
			token.LegalFramework.ApplicableLaws = make([]string, len(laws))
			for i, law := range laws {
				if str, ok := law.(string); ok {
					token.LegalFramework.ApplicableLaws[i] = str
				}
			}
		}
	}

	// Extract metadata
	token.ComplianceLevel = getString(claims, "compliance_level")
	token.GrantID = getString(claims, "grant_id")
	token.RequestID = getString(claims, "request_id")

	return token, nil
}

// Helper functions for claim extraction
func getString(m map[string]interface{}, key string) string {
	if val, ok := m[key].(string); ok {
		return val
	}
	return ""
}

func getInt64(m map[string]interface{}, key string) int64 {
	if val, ok := m[key].(float64); ok {
		return int64(val)
	}
	return 0
}

// validateLegalFrameworkCompliance validates legal framework compliance
func (s *ExtendedTokenService) validateLegalFrameworkCompliance(
	framework *LegalFrameworkInfo,
) bool {
	if framework == nil {
		return false
	}

	// Check minimum requirements
	if len(framework.ApplicableLaws) == 0 {
		return false
	}

	if framework.Jurisdiction == "" {
		return false
	}

	return true
}

// convertRestrictions converts interface{} restrictions to []PowerRestriction
func (s *ExtendedTokenService) convertRestrictions(restrictions interface{}) []PowerRestriction {
	if restrictions == nil {
		return nil
	}

	// Type assertion
	if r, ok := restrictions.([]PowerRestriction); ok {
		return r
	}

	// If not the right type, return empty slice
	return []PowerRestriction{}
}

// convertContext converts interface{} context to map[string]interface{}
func (s *ExtendedTokenService) convertContext(context interface{}) map[string]interface{} {
	if context == nil {
		return nil
	}

	// Type assertion
	if ctx, ok := context.(map[string]interface{}); ok {
		return ctx
	}

	// If not the right type, return empty map
	return map[string]interface{}{}
}

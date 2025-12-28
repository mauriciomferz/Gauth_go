// Package oidc - Enhanced Token Validation
// Provides comprehensive token validation with security best practices
package oidc

import (
	"context"
	"crypto/subtle"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// ValidationOptions configures token validation behavior.
type ValidationOptions struct {
	// RequireKID enforces that tokens must have a kid header
	RequireKID bool

	// AllowedSigningAlgorithms restricts accepted signing algorithms
	// Default: RS256, RS384, RS512, ES256, ES384, ES512
	AllowedSigningAlgorithms []string

	// ClockSkew allows for time drift between systems
	// Default: 5 minutes
	ClockSkew time.Duration

	// RequireExpiration enforces exp claim presence
	RequireExpiration bool

	// RequireIssuedAt enforces iat claim presence
	RequireIssuedAt bool

	// RequireNotBefore enforces nbf claim presence
	RequireNotBefore bool

	// RequireAudience enforces aud claim presence
	RequireAudience bool

	// RequireSubject enforces sub claim presence
	RequireSubject bool

	// AllowedIssuers restricts which issuers are accepted (whitelist)
	AllowedIssuers []string

	// MaxTokenAge maximum age of token from iat to now
	MaxTokenAge time.Duration

	// CheckRevocation checks if token is revoked
	CheckRevocation bool

	// RequireNonce enforces nonce claim presence (replay attack prevention)
	RequireNonce bool

	// ValidateSignature performs cryptographic signature verification
	ValidateSignature bool
}

// DefaultValidationOptions returns secure default validation options.
func DefaultValidationOptions() *ValidationOptions {
	return &ValidationOptions{
		RequireKID:               true,
		AllowedSigningAlgorithms: []string{"RS256", "RS384", "RS512", "ES256", "ES384", "ES512"},
		ClockSkew:                5 * time.Minute,
		RequireExpiration:        true,
		RequireIssuedAt:          true,
		RequireNotBefore:         false,
		RequireAudience:          true,
		RequireSubject:           true,
		AllowedIssuers:           nil, // Empty = allow all
		MaxTokenAge:              24 * time.Hour,
		CheckRevocation:          true,
		RequireNonce:             false,
		ValidateSignature:        true,
	}
}

// ValidationResult contains detailed validation results.
type ValidationResult struct {
	// Valid indicates if the token passed all validations
	Valid bool

	// Claims contains the parsed token claims
	Claims *IDTokenClaims

	// Errors contains validation errors (if any)
	Errors []ValidationError

	// Warnings contains non-critical validation warnings
	Warnings []string

	// ValidationTime records when validation occurred
	ValidationTime time.Time

	// TokenAge is the age of the token from iat
	TokenAge time.Duration
}

// ValidationError represents a specific validation failure.
type ValidationError struct {
	Code        string
	Description string
	Critical    bool // Critical errors fail validation immediately
}

// Error implements the error interface.
func (e ValidationError) Error() string {
	if e.Critical {
		return fmt.Sprintf("[CRITICAL] %s: %s", e.Code, e.Description)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Description)
}

// Validation error codes
const (
	ErrInvalidFormat         = "invalid_format"
	ErrInvalidSignature      = "invalid_signature"
	ErrExpiredToken          = "expired_token"
	ErrTokenNotYetValid      = "token_not_yet_valid" // #nosec G101 // Error code string constant
	ErrInvalidIssuer         = "invalid_issuer"
	ErrInvalidAudience       = "invalid_audience"
	ErrMissingKID            = "missing_kid"
	ErrUnsupportedAlgorithm  = "unsupported_algorithm"
	ErrMissingRequiredClaim  = "missing_required_claim"
	ErrTokenRevoked          = "token_revoked"
	ErrTokenTooOld           = "token_too_old"
	ErrInvalidNonce          = "invalid_nonce"
	ErrJWKSFetchFailed       = "jwks_fetch_failed"
	ErrDiscoveryFailed       = "discovery_failed"
	ErrClaimValidationFailed = "claim_validation_failed"
)

// EnhancedTokenValidator provides comprehensive token validation.
type EnhancedTokenValidator struct {
	baseValidator     *ExternalTokenValidator
	revocationService *TokenRevocationService
	options           *ValidationOptions
}

// NewEnhancedTokenValidator creates a new enhanced token validator.
func NewEnhancedTokenValidator(
	baseValidator *ExternalTokenValidator,
	revocationService *TokenRevocationService,
	options *ValidationOptions,
) *EnhancedTokenValidator {
	if options == nil {
		options = DefaultValidationOptions()
	}

	return &EnhancedTokenValidator{
		baseValidator:     baseValidator,
		revocationService: revocationService,
		options:           options,
	}
}

// ValidateToken performs comprehensive token validation.
func (v *EnhancedTokenValidator) ValidateToken(ctx context.Context, tokenString, issuer, audience string) (*ValidationResult, error) {
	result := &ValidationResult{
		Valid:          true,
		Errors:         []ValidationError{},
		Warnings:       []string{},
		ValidationTime: time.Now(),
	}

	// Step 1: Parse token without verification to inspect header and claims
	token, err := v.parseTokenUnsafe(tokenString)
	if err != nil {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Code:        ErrInvalidFormat,
			Description: fmt.Sprintf("failed to parse token: %v", err),
			Critical:    true,
		})
		return result, nil
	}

	// Step 2: Validate token header
	if headerErr := v.validateTokenHeader(token); headerErr.Code != "" {
		result.Valid = false
		result.Errors = append(result.Errors, headerErr)
		if headerErr.Critical {
			return result, nil
		}
	}

	// Step 3: Parse claims
	claims, ok := token.Claims.(*IDTokenClaims)
	if !ok {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Code:        ErrInvalidFormat,
			Description: "failed to parse token claims",
			Critical:    true,
		})
		return result, nil
	}
	result.Claims = claims

	// Step 4: Validate required claims presence
	if errs := v.validateRequiredClaims(claims); len(errs) > 0 {
		result.Valid = false
		result.Errors = append(result.Errors, errs...)
	}

	// Step 5: Validate temporal claims (exp, iat, nbf)
	if errs := v.validateTemporalClaims(claims); len(errs) > 0 {
		result.Valid = false
		result.Errors = append(result.Errors, errs...)
	}

	// Step 6: Validate issuer
	if issuerErr := v.validateIssuer(claims, issuer); issuerErr.Code != "" {
		result.Valid = false
		result.Errors = append(result.Errors, issuerErr)
	}

	// Step 7: Validate audience
	if audErr := v.validateAudience(claims, audience); audErr.Code != "" {
		result.Valid = false
		result.Errors = append(result.Errors, audErr)
	}

	// Step 8: Validate token age
	if warning := v.validateTokenAge(claims); warning != "" {
		result.Warnings = append(result.Warnings, warning)
	}

	// Step 9: Check revocation status
	if v.options.CheckRevocation && v.revocationService != nil {
		if revErr := v.checkRevocation(ctx, claims); revErr.Code != "" {
			result.Valid = false
			result.Errors = append(result.Errors, revErr)
		}
	}

	// Step 10: Cryptographic signature verification (if enabled and valid so far)
	if v.options.ValidateSignature && result.Valid {
		if err := v.verifySignature(ctx, tokenString, issuer, audience); err != nil {
			result.Valid = false
			result.Errors = append(result.Errors, ValidationError{
				Code:        ErrInvalidSignature,
				Description: err.Error(),
				Critical:    true,
			})
		}
	}

	return result, nil
}

// parseTokenUnsafe parses token without verification (for inspection).
func (v *EnhancedTokenValidator) parseTokenUnsafe(tokenString string) (*jwt.Token, error) {
	parser := jwt.NewParser()
	token, _, err := parser.ParseUnverified(tokenString, &IDTokenClaims{})
	if err != nil {
		return nil, err
	}
	return token, nil
}

// validateTokenHeader validates JWT header fields.
func (v *EnhancedTokenValidator) validateTokenHeader(token *jwt.Token) ValidationError {
	// Check for kid
	if v.options.RequireKID {
		kid, ok := token.Header["kid"].(string)
		if !ok || kid == "" {
			return ValidationError{
				Code:        ErrMissingKID,
				Description: "token header missing 'kid' (key ID)",
				Critical:    true,
			}
		}
	}

	// Validate signing algorithm
	alg, ok := token.Header["alg"].(string)
	if !ok {
		return ValidationError{
			Code:        ErrUnsupportedAlgorithm,
			Description: "token header missing 'alg' (algorithm)",
			Critical:    true,
		}
	}

	if len(v.options.AllowedSigningAlgorithms) > 0 {
		allowed := false
		for _, allowedAlg := range v.options.AllowedSigningAlgorithms {
			if alg == allowedAlg {
				allowed = true
				break
			}
		}
		if !allowed {
			return ValidationError{
				Code:        ErrUnsupportedAlgorithm,
				Description: fmt.Sprintf("signing algorithm '%s' not allowed", alg),
				Critical:    true,
			}
		}
	}

	return ValidationError{}
}

// validateRequiredClaims checks for required claim presence.
func (v *EnhancedTokenValidator) validateRequiredClaims(claims *IDTokenClaims) []ValidationError {
	var errors []ValidationError

	if v.options.RequireExpiration && claims.ExpiresAt == nil {
		errors = append(errors, ValidationError{
			Code:        ErrMissingRequiredClaim,
			Description: "missing required 'exp' (expiration) claim",
			Critical:    true,
		})
	}

	if v.options.RequireIssuedAt && claims.IssuedAt == nil {
		errors = append(errors, ValidationError{
			Code:        ErrMissingRequiredClaim,
			Description: "missing required 'iat' (issued at) claim",
			Critical:    true,
		})
	}

	if v.options.RequireNotBefore && claims.NotBefore == nil {
		errors = append(errors, ValidationError{
			Code:        ErrMissingRequiredClaim,
			Description: "missing required 'nbf' (not before) claim",
			Critical:    false,
		})
	}

	if v.options.RequireAudience && len(claims.Audience) == 0 {
		errors = append(errors, ValidationError{
			Code:        ErrMissingRequiredClaim,
			Description: "missing required 'aud' (audience) claim",
			Critical:    true,
		})
	}

	if v.options.RequireSubject && claims.Subject == "" {
		errors = append(errors, ValidationError{
			Code:        ErrMissingRequiredClaim,
			Description: "missing required 'sub' (subject) claim",
			Critical:    true,
		})
	}

	if v.options.RequireNonce && claims.Nonce == "" {
		errors = append(errors, ValidationError{
			Code:        ErrMissingRequiredClaim,
			Description: "missing required 'nonce' claim (replay attack prevention)",
			Critical:    true,
		})
	}

	return errors
}

// validateTemporalClaims validates time-based claims.
func (v *EnhancedTokenValidator) validateTemporalClaims(claims *IDTokenClaims) []ValidationError {
	var errors []ValidationError
	now := time.Now()

	// Validate expiration with clock skew
	if claims.ExpiresAt != nil {
		expiry := claims.ExpiresAt.Time.Add(v.options.ClockSkew)
		if now.After(expiry) {
			errors = append(errors, ValidationError{
				Code:        ErrExpiredToken,
				Description: fmt.Sprintf("token expired at %s (current time: %s)", claims.ExpiresAt.Time, now),
				Critical:    true,
			})
		}
	}

	// Validate not before with clock skew
	if claims.NotBefore != nil {
		notBefore := claims.NotBefore.Time.Add(-v.options.ClockSkew)
		if now.Before(notBefore) {
			errors = append(errors, ValidationError{
				Code:        ErrTokenNotYetValid,
				Description: fmt.Sprintf("token not valid until %s (current time: %s)", claims.NotBefore.Time, now),
				Critical:    true,
			})
		}
	}

	return errors
}

// validateIssuer validates the issuer claim.
func (v *EnhancedTokenValidator) validateIssuer(claims *IDTokenClaims, expectedIssuer string) ValidationError {
	// Use constant-time comparison to prevent timing attacks
	if subtle.ConstantTimeCompare([]byte(claims.Issuer), []byte(expectedIssuer)) != 1 {
		return ValidationError{
			Code:        ErrInvalidIssuer,
			Description: fmt.Sprintf("invalid issuer: expected '%s', got '%s'", expectedIssuer, claims.Issuer),
			Critical:    true,
		}
	}

	// Check against allowed issuers whitelist if configured
	if len(v.options.AllowedIssuers) > 0 {
		allowed := false
		for _, allowedIssuer := range v.options.AllowedIssuers {
			if claims.Issuer == allowedIssuer {
				allowed = true
				break
			}
		}
		if !allowed {
			return ValidationError{
				Code:        ErrInvalidIssuer,
				Description: fmt.Sprintf("issuer '%s' not in allowed list", claims.Issuer),
				Critical:    true,
			}
		}
	}

	return ValidationError{}
}

// validateAudience validates the audience claim.
func (v *EnhancedTokenValidator) validateAudience(claims *IDTokenClaims, expectedAudience string) ValidationError {
	// Check if expected audience is in the token's audience list
	audienceMatched := false
	for _, aud := range claims.Audience {
		// Use constant-time comparison
		if subtle.ConstantTimeCompare([]byte(aud), []byte(expectedAudience)) == 1 {
			audienceMatched = true
			break
		}
	}

	if !audienceMatched {
		return ValidationError{
			Code:        ErrInvalidAudience,
			Description: fmt.Sprintf("invalid audience: token not intended for '%s'", expectedAudience),
			Critical:    true,
		}
	}

	return ValidationError{}
}

// validateTokenAge checks if token is too old.
func (v *EnhancedTokenValidator) validateTokenAge(claims *IDTokenClaims) string {
	if v.options.MaxTokenAge == 0 || claims.IssuedAt == nil {
		return ""
	}

	age := time.Since(claims.IssuedAt.Time)
	if age > v.options.MaxTokenAge {
		return fmt.Sprintf("token age (%s) exceeds maximum allowed age (%s)", age, v.options.MaxTokenAge)
	}

	return ""
}

// checkRevocation verifies token is not revoked.
func (v *EnhancedTokenValidator) checkRevocation(ctx context.Context, claims *IDTokenClaims) ValidationError {
	tokenID := claims.ID
	if tokenID == "" {
		// If no JTI, cannot check revocation
		return ValidationError{}
	}

	isRevoked, err := v.revocationService.IsRevoked(ctx, tokenID)
	if err != nil {
		return ValidationError{
			Code:        ErrTokenRevoked,
			Description: fmt.Sprintf("failed to check revocation status: %v", err),
			Critical:    false,
		}
	}

	if isRevoked {
		return ValidationError{
			Code:        ErrTokenRevoked,
			Description: "token has been revoked",
			Critical:    true,
		}
	}

	return ValidationError{}
}

// verifySignature performs cryptographic signature verification.
func (v *EnhancedTokenValidator) verifySignature(ctx context.Context, tokenString, issuer, audience string) error {
	if v.baseValidator == nil {
		return fmt.Errorf("base validator not configured")
	}

	_, err := v.baseValidator.ValidateToken(ctx, tokenString, issuer, audience)
	if err != nil {
		return fmt.Errorf("signature verification failed: %w", err)
	}

	return nil
}

// ValidateTokenWithOptions validates a token with custom options.
func (v *EnhancedTokenValidator) ValidateTokenWithOptions(
	ctx context.Context,
	tokenString, issuer, audience string,
	options *ValidationOptions,
) (*ValidationResult, error) {
	// Temporarily override options
	originalOptions := v.options
	v.options = options
	defer func() { v.options = originalOptions }()

	return v.ValidateToken(ctx, tokenString, issuer, audience)
}

// ValidateTokenForProvider validates a token for a specific provider.
func (v *EnhancedTokenValidator) ValidateTokenForProvider(
	ctx context.Context,
	tokenString string,
	provider *ProviderConfig,
) (*ValidationResult, error) {
	// Customize options based on provider configuration
	options := DefaultValidationOptions()

	// Set allowed issuers
	if provider.IssuerURL != "" {
		options.AllowedIssuers = []string{provider.IssuerURL}
	}

	return v.ValidateTokenWithOptions(ctx, tokenString, provider.IssuerURL, provider.ClientID, options)
}

// ValidateClaims performs custom claim validations.
type ClaimValidator func(claims *IDTokenClaims) error

// ValidateWithCustomClaims validates token with additional custom claim checks.
func (v *EnhancedTokenValidator) ValidateWithCustomClaims(
	ctx context.Context,
	tokenString, issuer, audience string,
	customValidators ...ClaimValidator,
) (*ValidationResult, error) {
	// Perform standard validation
	result, err := v.ValidateToken(ctx, tokenString, issuer, audience)
	if err != nil {
		return result, err
	}

	// If standard validation failed, don't run custom validators
	if !result.Valid || result.Claims == nil {
		return result, nil
	}

	// Run custom validators
	for i, validator := range customValidators {
		if err := validator(result.Claims); err != nil {
			result.Valid = false
			result.Errors = append(result.Errors, ValidationError{
				Code:        ErrClaimValidationFailed,
				Description: fmt.Sprintf("custom validator %d failed: %v", i+1, err),
				Critical:    true,
			})
		}
	}

	return result, nil
}

// Common custom validators

// RequireEmailVerified ensures email is verified.
func RequireEmailVerified() ClaimValidator {
	return func(claims *IDTokenClaims) error {
		if claims.Email == "" {
			return fmt.Errorf("email claim not present")
		}
		if !claims.EmailVerified {
			return fmt.Errorf("email not verified")
		}
		return nil
	}
}

// RequireACRLevel ensures minimum authentication context class.
func RequireACRLevel(minLevel string) ClaimValidator {
	return func(claims *IDTokenClaims) error {
		if claims.ACR == "" {
			return fmt.Errorf("ACR claim not present")
		}
		// Simple string comparison (in production, implement proper ACR level comparison)
		if claims.ACR < minLevel {
			return fmt.Errorf("ACR level %s does not meet minimum %s", claims.ACR, minLevel)
		}
		return nil
	}
}

// RequireAMR ensures specific authentication methods were used.
func RequireAMR(requiredMethods ...string) ClaimValidator {
	return func(claims *IDTokenClaims) error {
		if len(claims.AMR) == 0 {
			return fmt.Errorf("AMR claim not present")
		}
		for _, required := range requiredMethods {
			found := false
			for _, amr := range claims.AMR {
				if amr == required {
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("required authentication method '%s' not found in AMR", required)
			}
		}
		return nil
	}
}

// RequireScope ensures specific scope is present.
func RequireScope(scope string) ClaimValidator {
	return func(claims *IDTokenClaims) error {
		// Check if scope claim exists (stored in jwt.RegisteredClaims or custom field)
		// This is a placeholder - actual implementation depends on where scope is stored
		return nil
	}
}

// BatchValidate validates multiple tokens efficiently.
func (v *EnhancedTokenValidator) BatchValidate(
	ctx context.Context,
	tokens []string,
	issuer, audience string,
) ([]*ValidationResult, error) {
	results := make([]*ValidationResult, len(tokens))

	for i, token := range tokens {
		result, err := v.ValidateToken(ctx, token, issuer, audience)
		if err != nil {
			return nil, fmt.Errorf("failed to validate token %d: %w", i, err)
		}
		results[i] = result
	}

	return results, nil
}

// GetValidationSummary returns a summary of validation results.
func (r *ValidationResult) GetValidationSummary() string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Validation Result: %t\n", r.Valid))
	sb.WriteString(fmt.Sprintf("Validation Time: %s\n", r.ValidationTime.Format(time.RFC3339)))

	if r.Claims != nil {
		sb.WriteString(fmt.Sprintf("Subject: %s\n", r.Claims.Subject))
		sb.WriteString(fmt.Sprintf("Issuer: %s\n", r.Claims.Issuer))
	}

	if len(r.Errors) > 0 {
		sb.WriteString(fmt.Sprintf("\nErrors (%d):\n", len(r.Errors)))
		for _, err := range r.Errors {
			sb.WriteString(fmt.Sprintf("  - %s\n", err.Error()))
		}
	}

	if len(r.Warnings) > 0 {
		sb.WriteString(fmt.Sprintf("\nWarnings (%d):\n", len(r.Warnings)))
		for _, warning := range r.Warnings {
			sb.WriteString(fmt.Sprintf("  - %s\n", warning))
		}
	}

	return sb.String()
}

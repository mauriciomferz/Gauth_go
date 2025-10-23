package gauth

import (
	"crypto/ed25519"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/internal/observability"
)

// ValidateAdvancedToken validates a token with advanced claims support
func (g *Service) ValidateAdvancedToken(token string) (*AdvancedTokenValidationResult, error) {
	// First perform standard validation
	basicResult, err := g.ValidateToken(token)
	if err != nil {
		return nil, err
	}
	
	// Parse token to extract claims for advanced validation
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, ErrInvalidToken
	}
	
	// Decode payload
	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("failed to decode token payload: %w", err)
	}
	
	var claimsMap map[string]interface{}
	if err := json.Unmarshal(payloadBytes, &claimsMap); err != nil {
		return nil, fmt.Errorf("failed to unmarshal token claims: %w", err)
	}
	
	// Convert to AdvancedClaims
	advancedClaims := &AdvancedClaims{}
	if err := advancedClaims.FromMap(claimsMap); err != nil {
		return nil, fmt.Errorf("failed to convert to advanced claims: %w", err)
	}
	
	// Perform semantic validation
	if err := advancedClaims.ValidateSemantics(); err != nil {
		failMetric(g, observability.SigInvalid)
		return nil, fmt.Errorf("semantic validation failed: %w", err)
	}
	
	// Check restrictions if present
	if advancedClaims.ClaimsMetadata != nil && advancedClaims.ClaimsMetadata.Restrictions != nil {
		if !advancedClaims.ClaimsMetadata.Restrictions.IsInTimeWindow() {
			failMetric(g, observability.SigInvalid)
			return nil, fmt.Errorf("token usage outside allowed time window")
		}
	}
	
	return &AdvancedTokenValidationResult{
		TokenValidationResult: *basicResult,
		AdvancedClaims:       advancedClaims,
		ValidationMetadata: &ValidationMetadata{
			ValidatedAt:   time.Now(),
			ValidationID:  generateJTI(),
			Confidence:    getConfidenceScore(advancedClaims),
			Warnings:      []string{},
		},
	}, nil
}

// AdvancedTokenValidationResult extends the basic validation result
type AdvancedTokenValidationResult struct {
	TokenValidationResult
	AdvancedClaims     *AdvancedClaims     `json:"advanced_claims"`
	ValidationMetadata *ValidationMetadata `json:"validation_metadata"`
}

// ValidationMetadata provides additional validation context
type ValidationMetadata struct {
	ValidatedAt   time.Time `json:"validated_at"`
	ValidationID  string    `json:"validation_id"`
	Confidence    float64   `json:"confidence"`
	Warnings      []string  `json:"warnings,omitempty"`
}

// getConfidenceScore calculates a confidence score based on claims quality
func getConfidenceScore(claims *AdvancedClaims) float64 {
	score := 0.5 // Base score
	
	// Increase confidence for metadata presence
	if claims.ClaimsMetadata != nil {
		score += 0.2
		if claims.ClaimsMetadata.Confidence > 0 {
			score = (score + claims.ClaimsMetadata.Confidence) / 2
		}
	}
	
	// Increase confidence for proper token type
	if claims.TokenType != "" {
		score += 0.1
	}
	
	// Increase confidence for restrictions (shows deliberate configuration)
	if claims.ClaimsMetadata != nil && claims.ClaimsMetadata.Restrictions != nil {
		score += 0.1
	}
	
	// Increase confidence for longer-lived tokens (shows trust)
	if claims.ExpiresAt > 0 && claims.IssuedAt > 0 {
		duration := claims.ExpiresAt - claims.IssuedAt
		if duration > 3600 { // More than 1 hour
			score += 0.1
		}
	}
	
	// Cap at 1.0
	if score > 1.0 {
		score = 1.0
	}
	
	return score
}

// CreateAdvancedToken creates a token with advanced claims
func (g *Service) CreateAdvancedToken(req AdvancedTokenRequest) (*TokenResponse, error) {
	// Create advanced claims
	advancedClaims := &AdvancedClaims{
		Subject:   req.Subject,
		Issuer:    g.config.AuthServerURL,
		Audience:  req.Audience,
		ExpiresAt: time.Now().Add(req.TTL).Unix(),
		IssuedAt:  time.Now().Unix(),
		NotBefore: time.Now().Unix(),
		JWTID:     generateJTI(),
		Scope:     req.Scope,
		TokenType: req.TokenType,
		ClientID:  req.ClientID,
		ClaimsMetadata: req.ClaimsMetadata,
		Custom:    req.CustomClaims,
	}
	
	// Validate before issuing
	if err := advancedClaims.ValidateSemantics(); err != nil {
		return nil, fmt.Errorf("advanced claims validation failed: %w", err)
	}
	
	// Convert to map for token creation
	claimsMap := advancedClaims.ToMap()
	
	// Convert scope array to space-separated string for JWT compatibility
	if scopes, ok := claimsMap["scope"].([]string); ok && len(scopes) > 0 {
		claimsMap["scope"] = strings.Join(scopes, " ")
	}
	
	// Create token using existing JWT logic
	if g.keyMode == sigModeEdDSA {
		return g.createEdDSAToken(claimsMap, req.TTL)
	}
	
	return g.createHMACToken(claimsMap, req.TTL)
}

// AdvancedTokenRequest represents a request for an advanced token
type AdvancedTokenRequest struct {
	Subject        string                 `json:"subject"`
	Audience       []string               `json:"audience"`
	Scope          []string               `json:"scope"`
	TTL            time.Duration          `json:"ttl"`
	TokenType      string                 `json:"token_type"`
	ClientID       string                 `json:"client_id"`
	ClaimsMetadata *ClaimsMetadata        `json:"claims_metadata,omitempty"`
	CustomClaims   map[string]interface{} `json:"custom_claims,omitempty"`
}

// createEdDSAToken creates an EdDSA-signed token with advanced claims
func (g *Service) createEdDSAToken(claims map[string]interface{}, ttl time.Duration) (*TokenResponse, error) {
	if g.keyMgr == nil {
		return nil, errors.New("missing_key_manager")
	}
	
	kid := g.keyMgr.Active().ID
	head := map[string]any{
		"alg": "EdDSA",
		"typ": "JWT",
		"kid": kid,
	}
	
	hb, err := json.Marshal(head)
	if err != nil {
		return nil, fmt.Errorf("marshal header: %w", err)
	}
	
	pb, err := json.Marshal(claims)
	if err != nil {
		return nil, fmt.Errorf("marshal claims: %w", err)
	}
	
	hEnc := base64.RawURLEncoding.EncodeToString(hb)
	pEnc := base64.RawURLEncoding.EncodeToString(pb)
	unsigned := hEnc + "." + pEnc
	
	sig := ed25519.Sign(g.keyMgr.Active().Private, []byte(unsigned))
	sEnc := base64.RawURLEncoding.EncodeToString(sig)
	token := unsigned + "." + sEnc
	
	if g.metrics != nil {
		g.metrics.IncTokensIssued()
	}
	
	// Extract scope for response
	var scopes []string
	if scopeVal, ok := claims["scope"]; ok {
		if scopeSlice, ok := scopeVal.([]string); ok {
			scopes = scopeSlice
		} else if scopeStr, ok := scopeVal.(string); ok {
			scopes = strings.Split(scopeStr, " ")
		}
	}
	
	return &TokenResponse{
		Token:      token,
		Scope:      scopes,
		ValidUntil: time.Now().Add(ttl),
	}, nil
}

// createHMACToken creates an HMAC-signed token with advanced claims
func (g *Service) createHMACToken(claims map[string]interface{}, ttl time.Duration) (*TokenResponse, error) {
	header := base64.RawURLEncoding.EncodeToString([]byte(legacyHeaderHS256))
	
	pb, err := json.Marshal(claims)
	if err != nil {
		return nil, fmt.Errorf("marshal claims: %w", err)
	}
	
	payload := base64.RawURLEncoding.EncodeToString(pb)
	unsigned := header + "." + payload
	
	mac := hmac.New(sha256.New, g.signingKey)
	mac.Write([]byte(unsigned))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	token := unsigned + "." + sig
	
	if g.metrics != nil {
		g.metrics.IncTokensIssued()
	}
	
	// Extract scope for response
	var scopes []string
	if scopeVal, ok := claims["scope"]; ok {
		if scopeSlice, ok := scopeVal.([]string); ok {
			scopes = scopeSlice
		} else if scopeStr, ok := scopeVal.(string); ok {
			scopes = strings.Split(scopeStr, " ")
		}
	}
	
	return &TokenResponse{
		Token:      token,
		Scope:      scopes,
		ValidUntil: time.Now().Add(ttl),
	}, nil
}

// ValidatePASETOWithFooter validates a PASETO token with structured footer
func (g *Service) ValidatePASETOWithFooter(token string) (*PASETOValidationResult, error) {
	// This is a placeholder for PASETO validation with structured footer
	// In a real implementation, this would use a PASETO library
	
	// For demonstration, we'll parse what looks like a PASETO token
	// Format: v4.public.payload.footer
	parts := strings.Split(token, ".")
	if len(parts) < 3 {
		return nil, fmt.Errorf("invalid PASETO token format")
	}
	
	version := parts[0]
	purpose := parts[1]
	
	if version != "v4" || purpose != "public" {
		return nil, fmt.Errorf("unsupported PASETO version/purpose: %s.%s", version, purpose)
	}
	
	// Extract footer if present
	var footer *PASETOFooter
	if len(parts) >= 4 && parts[3] != "" {
		footerData, err := base64.RawURLEncoding.DecodeString(parts[3])
		if err == nil {
			footer = &PASETOFooter{}
			if err := json.Unmarshal(footerData, footer); err == nil {
				// Validate footer structure
				if footer.KeyID == "" || footer.Algorithm == "" {
					return nil, fmt.Errorf("invalid PASETO footer structure")
				}
			}
		}
	}
	
	return &PASETOValidationResult{
		Valid:   true,
		Version: version,
		Purpose: purpose,
		Footer:  footer,
		Claims:  map[string]interface{}{}, // Would extract from payload
	}, nil
}

// PASETOValidationResult represents PASETO validation result
type PASETOValidationResult struct {
	Valid   bool                   `json:"valid"`
	Version string                 `json:"version"`
	Purpose string                 `json:"purpose"`
	Footer  *PASETOFooter          `json:"footer,omitempty"`
	Claims  map[string]interface{} `json:"claims"`
}
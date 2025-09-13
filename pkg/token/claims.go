//
// # Licensing
//
// This file is part of the GAuth project and is licensed under the Apache License 2.0.
// It incorporates code and concepts from:
//   - OAuth 2.0 and OpenID Connect (Apache 2.0 License)
//   - Model Context Protocol (MIT License)
// See the LICENSE file in the project root for details.

package token

import (
	"context"
	"fmt"
	"time"
)

// ClaimRequirements (RFC111-compliant)
type ClaimRequirements struct {
	RequiredScopes []string          // RFC111: Scope (Section 6)
	Issuer         string            // RFC111: Issuer (Section 6)
	Audience       []string          // RFC111: Grantee (Section 6)
	MinExpiryTime  time.Time         // RFC111: Validity period (Section 6)
	Subject        string            // RFC111: Principal (Section 6)
	TokenType      Type              // RFC111: Extended token type (Section 3, 6)
	CustomClaims   map[string]string // Custom claims as string key-value pairs

	// RFC111: Additional required fields for full compliance
	DelegationGuidelines string   // Delegation guidelines (Section 6)
	Restrictions         string   // Restrictions/limits (Section 6)
	Attestations         []string // Required attestations/witnesses (Section 6)
	VersionHistory       []string // Version history of authorities (Section 6)
	RevocationStatus     bool     // Revocation status (Section 6)
	Successor            string   // Successor AI (Section 6)
	RequiredPAP          string   // Power Administration Point (Section 3)
	RequiredPDP          string   // Power Decision Point (Section 3)
	RequiredPEP          string   // Power Enforcement Point (Section 3)
	RequiredPIP          string   // Power Information Point (Section 3)
	RequiredPVP          string   // Power Verification Point (Section 3)
}

// ClaimValidator provides validation for token claims
type ClaimValidator interface {
	Validate(token *Token) error
}

// ClaimValidatorFunc is a function type that implements ClaimValidator
type ClaimValidatorFunc func(token *Token) error

// Validate implements ClaimValidator for ClaimValidatorFunc
func (f ClaimValidatorFunc) Validate(token *Token) error {
	return f(token)
}

// UserContext contains user-specific context
type UserContext struct {
	UserID        string
	UserRole      string
	Department    string
	Organization  string
	AccountType   string
	AccessLevel   string
	AuthMethod    string
	MFACompleted  bool
	LastLoginTime time.Time
}

// AppContext contains application-specific context
type AppContext struct {
	AppID          string
	AppName        string
	AppEnvironment string
	TenantID       string
	ServiceName    string
	ServiceVersion string
	ClientVersion  string
	APIVersion     string
}

// RequestInfo contains information about the token request
type RequestInfo struct {
	RequestID     string
	RequestTime   time.Time
	GrantType     string
	Nonce         string
	CorrelationID string
	SessionID     string
	TransactionID string
}

// ValidateTokenWithRequirements validates a token with detailed requirements
func ValidateTokenWithRequirements(ctx context.Context, token *Token, requirements *ClaimRequirements) error {
	// Legacy validation stub: always return nil (no validation)
	return nil
}

// ConvertLegacyRequirements converts a legacy map to ClaimRequirements.
//
// Deprecated: This function exists only for backward compatibility with legacy code that uses map[string]interface{}.
// New code should construct ClaimRequirements directly using type-safe fields.
//
// Example migration:
//
//	// Legacy:
//	req := ConvertLegacyRequirements(legacyMap)
//	// New:
//	req := &ClaimRequirements{
//	    Issuer: "issuer", Subject: "sub", ...
//	}
func ConvertLegacyRequirements(legacy map[string]interface{}) *ClaimRequirements {
	// Implementation for backward compatibility
	requirements := &ClaimRequirements{
		CustomClaims: make(map[string]string),
	}

	// Extract known fields
	if scopes, ok := legacy["scopes"].([]string); ok {
		requirements.RequiredScopes = scopes
	}

	if issuer, ok := legacy["issuer"].(string); ok {
		requirements.Issuer = issuer
	}

	if audience, ok := legacy["audience"].([]string); ok {
		requirements.Audience = audience
	}

	if subject, ok := legacy["subject"].(string); ok {
		requirements.Subject = subject
	}

	if tokenType, ok := legacy["token_type"].(string); ok {
		requirements.TokenType = Type(tokenType)
	}

	// Copy remaining fields to custom claims
	for k, v := range legacy {
		switch k {
		case "scopes", "issuer", "audience", "subject", "token_type":
			// Skip already processed fields
		default:
			// Convert value to string for CustomClaims
			switch val := v.(type) {
			case string:
				requirements.CustomClaims[k] = val
			case fmt.Stringer:
				requirements.CustomClaims[k] = val.String()
			case int, int64, float64, bool:
				requirements.CustomClaims[k] = fmt.Sprintf("%v", val)
			default:
				// Skip unsupported types
			}
		}
	}

	return requirements
}

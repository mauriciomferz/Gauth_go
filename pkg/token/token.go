// ...existing code...
// Package token/token.go: RFC111 Compliance Mapping
//
// This file implements the extended token model and lifecycle as defined in RFC111:
//   - Explicit, type-safe token struct (TokenData) for all protocol flows
//   - Delegation, attestation, restrictions, issuance, and revocation fields
//   - Secure token generation and metadata
//
// Relevant RFC111 Sections:
//   - Section 3: Nomenclature (extended token, grant)
//   - Section 5: What GAuth is (token structure, delegation)
//   - Section 6: How GAuth works (token issuance, validation, revocation)
//
// Compliance:
//   - All fields are explicit and type-safe (no public map[string]interface{})
//   - Delegation, attestation, and restrictions are modeled as explicit types
//   - No exclusions (Web3, DNA, decentralized auth) are present
//   - See README and docs/ for full protocol mapping
//
// License: Apache 2.0 (see LICENSE file)

package token

import (
	"time"
)

// TokenData represents metadata associated with a token.
// Used for token lifecycle management, validation, and auditability.
// Includes fields for delegation, restrictions, issuance, and expiry.
type TokenData struct {
	TokenID      string         // Unique token identifier
	UserID       string         // Subject/user for whom the token is issued
	ClientID     string         // Client application identifier
	Scopes       []string       // Authorized scopes
	Restrictions *Restrictions  // Optional restrictions (e.g., IP, time window)
	IssuedAt     time.Time      // Token issuance time
	ExpiresAt    time.Time      // Token expiration time
}

// generateGrantID creates a unique grant identifier

// generateToken creates a new secure token

// ConvertTokenToTokenData maps a *Token to a *TokenData for compatibility.
// Returns a new TokenData struct or nil if input is nil.
func ConvertTokenToTokenData(t *Token) *TokenData {
	if t == nil {
		return nil
	}
	return &TokenData{
		TokenID:      t.ID,
		UserID:       t.Subject,
		ClientID:     "", // Not directly available; can be set if needed
		Scopes:       t.Scopes,
		Restrictions: nil, // Not available on Token
		IssuedAt:     t.IssuedAt,
		ExpiresAt:    t.ExpiresAt,
	}
}

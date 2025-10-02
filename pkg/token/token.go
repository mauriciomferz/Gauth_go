// Package token/token.go: RFC111 Implementation Attempt
//
// This file attempts to implement token model patterns similar to RFC111:
//   - Type-safe token struct (TokenData) for protocol flows
//   - Delegation, attestation, restrictions, issuance, and revocation fields
//   - Token generation and metadata handling
//
// Relevant RFC111 Sections Referenced:
//   - Section 3: Nomenclature (extended token, grant)
//   - Section 5: What GAuth is (token structure, delegation)
//   - Section 6: How GAuth works (token issuance, validation, revocation)
//
// Implementation Notes:
//   - Fields are explicit and type-safe (no public map[string]interface{})
//   - Delegation, attestation, and restrictions are modeled as explicit types
//   - WARNING: Compliance not validated by legal experts
//   - See README and docs/ for implementation details
//
// License: Apache 2.0 (see LICENSE file)

package token

import (
	"crypto/rand"
	"encoding/base64"
	"time"
)

// Data represents metadata associated with a token
type Data struct {
	TokenID      string
	UserID       string
	ClientID     string
	Scopes       []string
	Restrictions *Restrictions // Use the type from enhanced_types.go
	IssuedAt     time.Time
	ExpiresAt    time.Time
}

// generateGrantID creates a unique grant identifier

// generateToken creates a new secure token
func generateToken() (string, error) {
	b := make([]byte, 48)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// convertTokenToTokenData maps a *Token to a *Data for compatibility
func ConvertTokenToTokenData(t *Token) *Data {
	if t == nil {
		return nil
	}
	return &Data{
		TokenID:      t.ID,
		UserID:       t.Subject,
		ClientID:     "", // Not directly available; can be set if needed
		Scopes:       t.Scopes,
		Restrictions: nil, // Not available on Token
		IssuedAt:     t.IssuedAt,
		ExpiresAt:    t.ExpiresAt,
	}
}

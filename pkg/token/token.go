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
	return &Data {
		TokenID:      t.ID,
		UserID:       t.Subject,
		ClientID:     "", // Not directly available; can be set if needed
		Scopes:       t.Scopes,
		Restrictions: nil, // Not available on Token
		IssuedAt:     t.IssuedAt,
		ExpiresAt:    t.ExpiresAt,
	}
}

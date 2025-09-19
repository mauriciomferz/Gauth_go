// Copyright (c) 2025 Gimel Foundation and the persons identified as the document authors.
// All rights reserved. This file is subject to the Gimel Foundation's Legal Provisions Relating to GiFo Documents.
// See http://GimelFoundation.com or https://github.com/Gimel-Foundation for details.
// Code Components extracted from GiFo-RfC 0111 must include this license text and are provided without warranty.
//
// GAuth Protocol Compliance: This file implements the GAuth protocol (GiFo-RfC 0111).
//
// Protocol Usage Declaration:
//   - GAuth protocol: IMPLEMENTED throughout this file (see [GAuth] comments below)
//   - OAuth 2.0:      NOT USED anywhere in this file
//   - PKCE:           NOT USED anywhere in this file
//   - OpenID:         NOT USED anywhere in this file
//
// [GAuth] = GAuth protocol logic (GiFo-RfC 0111)
// [Other] = Placeholder for OAuth2, OpenID, PKCE, or other protocols (none present in this file)
//
// [GAuth] Package gauth provides GAuth protocol types and requests.
package gauth

import (
	"context"
	"time"

	"github.com/mauriciomferz/Gauth_go/pkg/common"
	"github.com/mauriciomferz/Gauth_go/pkg/token"
)

// AuthorizationRequest represents a request to initiate authorization (delegation)
type AuthorizationRequest struct {
	ClientID string
	Scopes   []string
}

// TokenType represents the type of token issued.
//
// Standard token types include:
//   - access_token: Short-lived, MUST be used for resource access and MUST NOT be reused for delegation or attestation.
//   - refresh_token: Long-lived, MAY be used to obtain new access tokens, MUST be protected and not shared.
//
// Extended token types (per RFC 0111) MAY be defined for advanced delegation, attestation, or versioned power-of-attorney flows.
// Extended tokens MUST be explicitly typed and their semantics documented in code and comments.
//
// Examples of extended tokens:
//   - delegated_token: Used for explicit delegation, MUST include delegation and attestation fields.
//   - attested_token: Used for high-assurance flows, MUST include notary/witness attestation.
//
// All token types MUST be auditable and type-safe. See AuthorizationGrant and Attestation for required fields.
type TokenType string

const (
	// AccessToken represents a short-lived token for resource access.
	// This is the default token type for most authorization flows.
	AccessToken TokenType = "access_token"

	// RefreshToken represents a long-lived token for obtaining new access tokens.
	// This token type MUST NOT be used for direct resource access.
	RefreshToken TokenType = "refresh_token"

	// DelegatedToken represents an extended token type for explicit delegation flows.
	// This token type MUST include delegation and attestation fields in the grant.
	DelegatedToken TokenType = "delegated_token"

	// AttestedToken represents an extended token type for high-assurance, attested delegation.
	// This token type MUST include a valid Attestation in the grant.
	AttestedToken TokenType = "attested_token"
)

// AuthorizationGrant represents the granted authorization
// AuthorizationGrant represents the granted authorization.
//
// This type MUST be used to represent all delegation and power-of-attorney relationships.
// It SHOULD include delegation, revocation, attestation, and version history fields as per RFC 0111.
type AuthorizationGrant struct {
 GrantID      string
 ClientID     string
 Scope        []string
 Restrictions []Restriction
 ValidUntil   time.Time

 // Delegation: If this grant is delegated from another principal, this field MUST be set.
 DelegatedFrom string // Principal who delegated authority (optional)

 // Revocation: If this grant is revoked, this field MUST be set.
 Revoked      bool
 RevokedAt    *time.Time
 RevokedBy    string // Principal who revoked (optional)

 // Attestation: Optional notary/witness attestation for high-assurance delegation.
 Attestation  *Attestation

 // Version history: Track changes to the grant for auditability.
 Version      int
 VersionLog   []GrantVersion
}

// Attestation represents a notary or witness attestation for a grant.
type Attestation struct {
 Notary   string    // Notary or witness identifier
 Version  string    // Attestation version
 IssuedAt time.Time // When attestation was issued
}

// GrantVersion represents a historical version of an AuthorizationGrant.
type GrantVersion struct {
 Version     int
 ChangedAt   time.Time
 ChangedBy   string // Principal who made the change
 Description string // Description of the change
}

// TokenRequest represents a request for a token
type TokenRequest struct {
	GrantID      string
	Scope        []string
	Restrictions []Restriction
	Context      context.Context
}

// TokenResponse represents the response to a token request
type TokenResponse struct {
	Token        string
	ValidUntil   time.Time
	Scope        []string
	Restrictions []Restriction
}

// Config represents the configuration for GAuth

type Config struct {
	AuthServerURL     string                 // URL of the authorization server
	ClientID          string                 // Client identifier
	ClientSecret      string                 // Client secret
	Scopes            []string               // Default scopes
	RateLimit         common.RateLimitConfig // Rate limiting configuration
	AccessTokenExpiry time.Duration          // Token expiry duration
	TokenConfig       *token.Config          // Embedded token config (pointer)
}

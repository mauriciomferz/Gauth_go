// Copyright (c) 2025 Gimel Foundation and the persons identified as the document authors.
// All rights reserved. This file is subject to the Gimel Foundation's Legal Provisions Relating to GiFo Documents.
// See http://GimelFoundation.com or https://github.com/Gimel-Foundation for details.
// Code Components extracted from GiFo-RfC 0111 must include this license text and are provided without warranty.

// [GAuth] Response types for the GAuth protocol.
package gauth

import (
	"time"
)

// TokenResponse represents the response to a token request
type TokenResponse struct {
	Token        string
	ValidUntil   time.Time
	Scope        []string
	Restrictions []Restriction
}

// AuthorizationGrant represents the granted authorization
// This type MUST be used to represent all delegation and power-of-attorney relationships.
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

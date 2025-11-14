// Package gauth - External Service Integration Types
// Types used for external PVP and PIP client integrations
package gauth

import "time"

// IdentityVerificationRequest represents a request to verify identity through PVP
type IdentityVerificationRequest struct {
	SubjectID     string    `json:"subject_id"`
	RequiredLevel string    `json:"required_level"`
	ProofMethod   string    `json:"proof_method"`
	ProofData     []byte    `json:"proof_data,omitempty"`
	RequestID     string    `json:"request_id"`
	RequestedAt   time.Time `json:"requested_at"`
	Context       string    `json:"context,omitempty"`
}

// IdentityVerificationResult represents the result of identity verification
type IdentityVerificationResult struct {
	Verified           bool                   `json:"verified"`
	VerificationID     string                 `json:"verification_id"`
	SubjectID          string                 `json:"subject_id"`
	IdentityLevel      string                 `json:"identity_level"`
	VerifiedAt         time.Time              `json:"verified_at"`
	ExpiresAt          time.Time              `json:"expires_at"`
	VerificationMethod string                 `json:"verification_method"`
	IsFallback         bool                   `json:"is_fallback"`
	ConfidenceScore    float64                `json:"confidence_score,omitempty"`
	Metadata           map[string]interface{} `json:"metadata,omitempty"`
}

// PolicyRequest represents a request to retrieve policy information from PIP
type PolicyRequest struct {
	PolicyID       string                 `json:"policy_id"`
	RequesterID    string                 `json:"requester_id"`
	Context        string                 `json:"context,omitempty"`
	IncludeExpired bool                   `json:"include_expired"`
	RequestedAt    time.Time              `json:"requested_at"`
	Filters        map[string]interface{} `json:"filters,omitempty"`
}

// PowerOfAttorneyPolicy represents a power of attorney policy retrieved from PIP
type PowerOfAttorneyPolicy struct {
	PolicyID    string                 `json:"policy_id"`
	GrantorID   string                 `json:"grantor_id"`
	GranteeID   string                 `json:"grantee_id"`
	Scope       []string               `json:"scope"`
	Limitations []string               `json:"limitations,omitempty"`
	IssuedAt    time.Time              `json:"issued_at"`
	ExpiresAt   time.Time              `json:"expires_at"`
	Revoked     bool                   `json:"revoked"`
	RevokedAt   *time.Time             `json:"revoked_at,omitempty"`
	Status      string                 `json:"status"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

package token

import (
	"context"
	"time"
)

// OwnerInfo contains information about the token owner and authorizer
type OwnerInfo struct {
	// OwnerID is the unique identifier of the owner
	OwnerID string `json:"owner_id"`

	// OwnerType indicates if this is a client owner or resource owner
	OwnerType string `json:"owner_type"` // "client_owner" or "resource_owner"

	// AuthorizerID is the ID of the owner's authorizer
	AuthorizerID string `json:"authorizer_id"`

	// AuthorizationRef is a reference to the authorization document
	AuthorizationRef string `json:"authorization_ref"`

	// RegistrationInfo contains official registration details
	RegistrationInfo *RegistrationInfo `json:"registration_info,omitempty"`
}

// RegistrationInfo contains official registration details
type RegistrationInfo struct {
	// RegistryID is the ID in the official registry (e.g., commercial register)
	RegistryID string `json:"registry_id"`

	// RegistryType indicates the type of registry
	RegistryType string `json:"registry_type"`

	// JurisdictionCountry is the country of jurisdiction
	JurisdictionCountry string `json:"jurisdiction_country"`

	// RegistrationDate is when the entity was registered
	RegistrationDate time.Time `json:"registration_date"`
}

// AIMetadata contains AI-specific metadata
type AIMetadata struct {
	// AIType indicates the type of AI (digital agent, robot, etc)
	AIType string `json:"ai_type"`

	// Capabilities describes what this AI can do
	Capabilities []string `json:"capabilities"`

	// DelegationGuidelines specify principles for power transfer
	DelegationGuidelines []string `json:"delegation_guidelines"`

	// Restrictions define limits of transferred powers
	Restrictions *Restrictions `json:"restrictions"`

	// SuccessorID is the ID of a backup AI
	SuccessorID string `json:"successor_id,omitempty"`
}

// Restrictions defines limits on AI powers
type Restrictions struct {
	// ValueLimits for transactions
	ValueLimits *ValueLimits `json:"value_limits,omitempty"`

	// GeographicConstraints for actions
	GeographicConstraints []string `json:"geographic_constraints,omitempty"`

	// TimeConstraints for operations
	TimeConstraints *TimeConstraints `json:"time_constraints,omitempty"`

	// CustomLimits for specific restrictions
	CustomLimits map[string]float64 `json:"custom_limits,omitempty"`
}

// ValueLimits defines transaction value restrictions
type ValueLimits struct {
	// MaxTransactionValue is the maximum value for a single transaction
	MaxTransactionValue float64 `json:"max_transaction_value"`

	// DailyLimit is the maximum total value per day
	DailyLimit float64 `json:"daily_limit"`

	// Currency for the limits
	Currency string `json:"currency"`
}

// TimeConstraints defines time-based restrictions
type TimeConstraints struct {
	// AllowedTimeWindows specifies when operations are allowed
	AllowedTimeWindows []TimeWindow `json:"allowed_time_windows"`

	// TimeZone for the windows
	TimeZone string `json:"time_zone"`
}

// TimeWindow defines an allowed time period
type TimeWindow struct {
	// StartTime in 24h format (HH:MM)
	StartTime string `json:"start_time"`

	// EndTime in 24h format (HH:MM)
	EndTime string `json:"end_time"`

	// DaysOfWeek allowed (0 = Sunday, 6 = Saturday)
	DaysOfWeek []int `json:"days_of_week"`
}

// Attestation represents required verification
type Attestation struct {
	// Type of attestation (e.g., "notary", "witness")
	Type string `json:"type"`

	// AttesterID is who provided the attestation
	AttesterID string `json:"attester_id"`

	// AttestationDate is when it was provided
	AttestationDate time.Time `json:"attestation_date"`

	// Evidence is proof of attestation
	Evidence string `json:"evidence"`
}

// VersionInfo tracks authority changes
type VersionInfo struct {
	// Version number
	Version int `json:"version"`

	// UpdatedAt timestamp
	UpdatedAt time.Time `json:"updated_at"`

	// UpdatedBy identifier
	UpdatedBy string `json:"updated_by"`

	// ChangeType describes what changed
	ChangeType string `json:"change_type"`

	// ChangeSummary describes the changes
	ChangeSummary string `json:"change_summary"`
}

// EnhancedToken extends the base Token with GAuth-specific fields
type EnhancedToken struct {
	// Embed the base Token
	*Token

	// Owner information
	Owner *OwnerInfo `json:"owner"`

	// AI-specific metadata
	AI *AIMetadata `json:"ai"`

	// Required attestations
	Attestations []Attestation `json:"attestations"`

	// Version history
	Versions []VersionInfo `json:"versions"`
}

// EnhancedStore extends the base Store interface
type EnhancedStore interface {
	Store

	// Additional GAuth-specific methods
	ValidateAuthorization(ctx context.Context, token *EnhancedToken) error
	VerifyAttestation(ctx context.Context, attestation *Attestation) error
	TrackVersionHistory(ctx context.Context, token *EnhancedToken) error
}

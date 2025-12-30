package agentauthplus

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mauriciomferz/AgentAuth/pkg/registry"
)

// VerificationResult represents the result of a PoA verification
type VerificationResult struct {
	Valid              bool                `json:"valid"`
	PoAID              string              `json:"poa_id"`
	VerifiedAt         time.Time           `json:"verified_at"`
	Status             string              `json:"status"` // "active", "revoked", "expired", "pending"
	IssuerID           string              `json:"issuer_id"`
	GranteeID          string              `json:"grantee_id"`
	ValidFrom          time.Time           `json:"valid_from"`
	ValidUntil         time.Time           `json:"valid_until"`
	Scope              []string            `json:"scope"`
	Attestations       []Attestation       `json:"attestations"`
	Warnings           []string            `json:"warnings,omitempty"`
	VerificationMethod string              `json:"verification_method"`
	VerificationReport *VerificationReport `json:"verification_report,omitempty"`
}

// ScopeVerificationResult represents verification of an action against PoA scope
type ScopeVerificationResult struct {
	Authorized      bool               `json:"authorized"`
	PoAID           string             `json:"poa_id"`
	Action          Action             `json:"action"`
	MatchedScopes   []string           `json:"matched_scopes"`
	Restrictions    []RestrictionCheck `json:"restrictions"`
	GeographicCheck *GeographicCheck   `json:"geographic_check,omitempty"`
	ValueLimitCheck *ValueLimitCheck   `json:"value_limit_check,omitempty"`
	Reason          string             `json:"reason,omitempty"`
}

// PrincipalStatusResult represents verification of principal's legal capacity
type PrincipalStatusResult struct {
	Valid              bool         `json:"valid"`
	PrincipalID        string       `json:"principal_id"`
	LegalCapacity      string       `json:"legal_capacity"` // "full", "limited", "incapacitated"
	EntityType         string       `json:"entity_type"`    // "individual", "corporation", "partnership", etc.
	Status             string       `json:"status"`         // "active", "dissolved", "suspended"
	JurisdictionStatus string       `json:"jurisdiction_status"`
	VerifiedAt         time.Time    `json:"verified_at"`
	ExpiresAt          time.Time    `json:"expires_at"`
	Attestation        *Attestation `json:"attestation,omitempty"`
	Issues             []string     `json:"issues,omitempty"`
}

// PositionVerificationResult represents verification of representative's position
type PositionVerificationResult struct {
	Valid            bool         `json:"valid"`
	RepresentativeID string       `json:"representative_id"`
	OrganizationID   string       `json:"organization_id"`
	Position         string       `json:"position"`
	AuthorizedToAct  bool         `json:"authorized_to_act"`
	SigninagentAuthority bool         `json:"signing_authority"`
	EffectiveDate    time.Time    `json:"effective_date"`
	VerifiedAt       time.Time    `json:"verified_at"`
	Issues           []string     `json:"issues,omitempty"`
	Attestation      *Attestation `json:"attestation,omitempty"`
}

// RevocationStatusResult represents the revocation status of a PoA
type RevocationStatusResult struct {
	Revoked         bool       `json:"revoked"`
	PoAID           string     `json:"poa_id"`
	RevokedAt       *time.Time `json:"revoked_at,omitempty"`
	RevokedBy       string     `json:"revoked_by,omitempty"`
	Reason          string     `json:"reason,omitempty"`
	EffectiveDate   *time.Time `json:"effective_date,omitempty"`
	NotifiedParties []string   `json:"notified_parties,omitempty"`
	CheckedAt       time.Time  `json:"checked_at"`
}

// VerificationReport is a comprehensive report for relying parties
type VerificationReport struct {
	ReportID        string    `json:"report_id"`
	GeneratedAt     time.Time `json:"generated_at"`
	PoAID           string    `json:"poa_id"`
	RequestedAction Action    `json:"requested_action"`
	OverallValid    bool      `json:"overall_valid"`

	// Core verifications
	PoAVerification   *VerificationResult      `json:"poa_verification"`
	ScopeVerification *ScopeVerificationResult `json:"scope_verification"`
	PrincipalStatus   *PrincipalStatusResult   `json:"principal_status"`
	RevocationStatus  *RevocationStatusResult  `json:"revocation_status"`

	// Additional checks
	ChainOfAuthority    []AuthorityLink              `json:"chain_of_authority"`
	FiduciaryCompliance *FiduciaryComplianceCheck    `json:"fiduciary_compliance"`
	CapabilityCheck     *CapabilityVerificationCheck `json:"capability_check"`

	// Summary
	Attestations    []Attestation `json:"attestations,omitempty"`
	Warnings        []string      `json:"warnings,omitempty"`
	Recommendations []string      `json:"recommendations,omitempty"`
	ValidityPeriod  string        `json:"validity_period"`
	NextReviewDate  time.Time     `json:"next_review_date"`
}

// Action represents an action to be verified
type Action struct {
	Type      string                 `json:"type"` // "transaction", "decision", "signature"
	Resource  string                 `json:"resource"`
	Operation string                 `json:"operation"`
	Value     *float64               `json:"value,omitempty"`
	Currency  string                 `json:"currency,omitempty"`
	Location  *GeographicLocation    `json:"location,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// Attestation represents a legal or compliance attestation
type Attestation struct {
	ID            string                 `json:"id"`
	Type          string                 `json:"type"` // "notary", "witness", "electronic_signature", "compliance"
	AttestorID    string                 `json:"attestor_id"`
	AttestorName  string                 `json:"attestor_name"`
	AttestorRole  string                 `json:"attestor_role,omitempty"`
	Provider      string                 `json:"provider,omitempty"`
	Method        string                 `json:"method"` // "in_person", "video", "electronic"
	Timestamp     time.Time              `json:"timestamp"`
	VerifiedAt    time.Time              `json:"verified_at,omitempty"`
	Location      string                 `json:"location,omitempty"`
	CertificateID string                 `json:"certificate_id,omitempty"`
	SignatureHash string                 `json:"signature_hash,omitempty"`
	Status        string                 `json:"status"` // "pending", "verified", "failed"
	Verified      bool                   `json:"verified"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// RestrictionCheck represents a restriction verification
type RestrictionCheck struct {
	Type        string `json:"type"` // "value_limit", "geographic", "temporal", "conditional"
	Description string `json:"description"`
	Satisfied   bool   `json:"satisfied"`
	Details     string `json:"details,omitempty"`
}

// GeographicCheck represents geographic constraint verification
type GeographicCheck struct {
	Allowed           bool     `json:"allowed"`
	RequestedLocation string   `json:"requested_location"`
	AllowedCountries  []string `json:"allowed_countries"`
	ExcludedRegions   []string `json:"excluded_regions"`
	Reason            string   `json:"reason,omitempty"`
}

// GeographicLocation represents a location
type GeographicLocation struct {
	Country     string  `json:"country"`
	Region      string  `json:"region,omitempty"`
	City        string  `json:"city,omitempty"`
	Coordinates *LatLon `json:"coordinates,omitempty"`
}

// LatLon represents latitude/longitude coordinates
type LatLon struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

// ValueLimitCheck represents value limit verification
type ValueLimitCheck struct {
	Within              bool    `json:"within"`
	RequestedValue      float64 `json:"requested_value"`
	MaxValue            float64 `json:"max_value"`
	Currency            string  `json:"currency"`
	RequiresDualControl bool    `json:"requires_dual_control"`
}

// AuthorityLink represents a link in the authorization chain
type AuthorityLink struct {
	Level           int       `json:"level"`
	PoAID           string    `json:"poa_id"`
	IssuerID        string    `json:"issuer_id"`
	GranteeID       string    `json:"grantee_id"`
	GrantedAt       time.Time `json:"granted_at"`
	IsHuman         bool      `json:"is_human"`
	DelegationDepth int       `json:"delegation_depth"`
}

// FiduciaryComplianceCheck represents fiduciary duty compliance verification
type FiduciaryComplianceCheck struct {
	Compliant      bool                 `json:"compliant"`
	DutiesChecked  []string             `json:"duties_checked"`
	Violations     []FiduciaryViolation `json:"violations,omitempty"`
	LastAssessment time.Time            `json:"last_assessment"`
}

// FiduciaryViolation represents a fiduciary duty violation
type FiduciaryViolation struct {
	Duty        string    `json:"duty"`
	Description string    `json:"description"`
	Severity    string    `json:"severity"` // "minor", "major", "critical"
	DetectedAt  time.Time `json:"detected_at"`
}

// CapabilityVerificationCheck represents capability assessment verification
type CapabilityVerificationCheck struct {
	Sufficient      bool               `json:"sufficient"`
	RequiredLevel   string             `json:"required_level"`
	ActualLevel     string             `json:"actual_level"`
	RequiredDomains map[string]float64 `json:"required_domains"`
	ActualDomains   map[string]float64 `json:"actual_domains"`
	DeficientAreas  []string           `json:"deficient_areas,omitempty"`
	LastAssessment  time.Time          `json:"last_assessment"`
}

// VerificationService defines methods for comprehensive PoA verification
type VerificationService interface {
	// VerifyPowerOfAttorney verifies that a PoA is valid and active
	VerifyPowerOfAttorney(ctx context.Context, poaID string) (*VerificationResult, error)

	// VerifyScope verifies that an action falls within the PoA scope
	VerifyScope(ctx context.Context, poaID string, action Action) (*ScopeVerificationResult, error)

	// VerifyPrincipalStatus verifies the principal's legal capacity
	VerifyPrincipalStatus(ctx context.Context, principalID string) (*PrincipalStatusResult, error)

	// VerifyRepresentativePosition verifies the representative's position in an organization
	VerifyRepresentativePosition(ctx context.Context, repID string, orgID string) (*PositionVerificationResult, error)

	// CheckRevocationStatus checks if a PoA has been revoked
	CheckRevocationStatus(ctx context.Context, poaID string) (*RevocationStatusResult, error)

	// GenerateVerificationReport generates a comprehensive verification report for relying parties
	GenerateVerificationReport(ctx context.Context, poaID string, action Action) (*VerificationReport, error)

	// VerifyAuthorizationChain verifies the complete chain of authority back to a human principal
	VerifyAuthorizationChain(ctx context.Context, poaID string) ([]AuthorityLink, error)

	// VerifyAttestations verifies all attestations associated with a PoA
	VerifyAttestations(ctx context.Context, poaID string) ([]Attestation, error)
}

// VerificationServiceImpl implements the VerificationService interface
type VerificationServiceImpl struct {
	poaStore            PoAStore
	delegationService   DelegationService
	capabilityService   CapabilityAssessmentService
	fiduciaryService    FiduciaryDutyService
	principalVerifier   PrincipalVerifier
	attestationVerifier AttestationVerifier
	attestationSigner   AttestationSigner
	registerService     registry.CommercialRegisterService
}

// PoAStore interface for accessing PoA data
type PoAStore interface {
	GetPoA(ctx context.Context, poaID string) (*EnhancedPoA, error)
	GetPoAsByGrantee(ctx context.Context, granteeID string) ([]*EnhancedPoA, error)
	IsRevoked(ctx context.Context, poaID string) (bool, *RevocationInfo, error)
}

// EnhancedPoA represents the enhanced Power of Attorney structure
type EnhancedPoA struct {
	ID               string                  `json:"id"`
	IssuerID         string                  `json:"issuer_id"`
	GranteeID        string                  `json:"grantee_id"`
	SourcePOAID      *string                 `json:"source_poa_id,omitempty"`
	SuccessorID      *string                 `json:"successor_id,omitempty"`
	Scope            []string                `json:"scope"`
	StructuredScope  *StructuredScope        `json:"structured_scope,omitempty"`
	DelegationPolicy *DelegationPolicy       `json:"delegation_policy,omitempty"`
	Restrictions     *Restrictions           `json:"restrictions,omitempty"`
	FiduciaryDuties  *FiduciaryDuties        `json:"fiduciary_duties,omitempty"`
	CapabilityReqs   *CapabilityRequirements `json:"capability_requirements,omitempty"`
	ObligationType   ObligationType          `json:"obligation_type"`
	ValidFrom        time.Time               `json:"valid_from"`
	ValidUntil       time.Time               `json:"valid_until"`
	Status           string                  `json:"status"` // "active", "revoked", "expired", "pending"
	Attestations     []Attestation           `json:"attestations"`
	VersionNumber    int                     `json:"version_number"`
	VersionHistory   []PoAVersion            `json:"version_history"`
	CreatedAt        time.Time               `json:"created_at"`
	UpdatedAt        time.Time               `json:"updated_at"`
	RevokedAt        *time.Time              `json:"revoked_at,omitempty"`
	RevokedBy        *string                 `json:"revoked_by,omitempty"`
	RevocationReason *string                 `json:"revocation_reason,omitempty"`
}

// StructuredScope represents structured scope with detailed constraints
type StructuredScope struct {
	Transactions          []string               `json:"transactions"`
	Decisions             []string               `json:"decisions"`
	Actions               []string               `json:"actions"`
	GeographicConstraints *GeographicConstraints `json:"geographic_constraints,omitempty"`
	Conditions            *ScopeConditions       `json:"conditions,omitempty"`
}

// GeographicConstraints represents geographic limitations
type GeographicConstraints struct {
	AllowedCountries  []string `json:"allowed_countries,omitempty"`
	ExcludedRegions   []string `json:"excluded_regions,omitempty"`
	RequiresLocalAuth bool     `json:"requires_local_auth,omitempty"`
}

// ScopeConditions represents conditional limitations
type ScopeConditions struct {
	MaxTransactionValue      *float64 `json:"max_transaction_value,omitempty"`
	Currency                 string   `json:"currency,omitempty"`
	RequiresDualControlAbove *float64 `json:"requires_dual_control_above,omitempty"`
	TimeRestrictions         []string `json:"time_restrictions,omitempty"`
	ResourceRestrictions     []string `json:"resource_restrictions,omitempty"`
}

// Restrictions represents all restrictions on a PoA
type Restrictions struct {
	ValueLimits             *ValueLimits           `json:"value_limits,omitempty"`
	GeographicRestrictions  *GeographicConstraints `json:"geographic_restrictions,omitempty"`
	TemporalRestrictions    []TemporalRestriction  `json:"temporal_restrictions,omitempty"`
	ConditionalRestrictions []string               `json:"conditional_restrictions,omitempty"`
}

// ValueLimits represents financial value limitations
type ValueLimits struct {
	MaxSingleTransaction float64 `json:"max_single_transaction"`
	MaxDailyTotal        float64 `json:"max_daily_total,omitempty"`
	MaxMonthlyTotal      float64 `json:"max_monthly_total,omitempty"`
	Currency             string  `json:"currency"`
}

// TemporalRestriction represents time-based restrictions
type TemporalRestriction struct {
	Type        string    `json:"type"` // "hours", "days", "blackout_periods"
	Description string    `json:"description"`
	StartTime   time.Time `json:"start_time,omitempty"`
	EndTime     time.Time `json:"end_time,omitempty"`
}

// PoAVersion represents a version in the PoA history
type PoAVersion struct {
	VersionNumber int                    `json:"version_number"`
	ModifiedAt    time.Time              `json:"modified_at"`
	ModifiedBy    string                 `json:"modified_by"`
	Changes       []string               `json:"changes"`
	PreviousState map[string]interface{} `json:"previous_state,omitempty"`
}

// RevocationInfo contains revocation details
type RevocationInfo struct {
	RevokedAt       time.Time `json:"revoked_at"`
	RevokedBy       string    `json:"revoked_by"`
	Reason          string    `json:"reason"`
	NotifiedParties []string  `json:"notified_parties"`
}

// PrincipalVerifier interface for verifying principal status
type PrincipalVerifier interface {
	VerifyPrincipal(ctx context.Context, principalID string) (*PrincipalStatusResult, error)
}

// AttestationVerifier interface for verifying attestations
type AttestationVerifier interface {
	Verify(ctx context.Context, attestation Attestation) (bool, error)
}

// NewVerificationService creates a new verification service
func NewVerificationService(
	poaStore PoAStore,
	delegationService DelegationService,
	capabilityService CapabilityAssessmentService,
	fiduciaryService FiduciaryDutyService,
	principalVerifier PrincipalVerifier,
	attestationVerifier AttestationVerifier,
	attestationSigner AttestationSigner,
	registerService registry.CommercialRegisterService,
) VerificationService {
	return &VerificationServiceImpl{
		poaStore:            poaStore,
		delegationService:   delegationService,
		capabilityService:   capabilityService,
		fiduciaryService:    fiduciaryService,
		principalVerifier:   principalVerifier,
		attestationVerifier: attestationVerifier,
		attestationSigner:   attestationSigner,
		registerService:     registerService,
	}
}

// VerifyPowerOfAttorney verifies that a PoA is valid and active
func (v *VerificationServiceImpl) VerifyPowerOfAttorney(ctx context.Context, poaID string) (*VerificationResult, error) {
	// Get PoA from store
	poa, err := v.poaStore.GetPoA(ctx, poaID)
	if err != nil {
		return nil, fmt.Errorf("failed to get PoA: %w", err)
	}

	now := time.Now()
	warnings := []string{}

	// Check temporal validity
	valid := true
	status := poa.Status

	if now.Before(poa.ValidFrom) {
		valid = false
		status = "pending"
		warnings = append(warnings, "PoA is not yet valid")
	}

	if now.After(poa.ValidUntil) {
		valid = false
		status = "expired"
		warnings = append(warnings, "PoA has expired")
	}

	// Check revocation status
	revoked, revInfo, err := v.poaStore.IsRevoked(ctx, poaID)
	if err != nil {
		warnings = append(warnings, fmt.Sprintf("Could not verify revocation status: %v", err))
	} else if revoked {
		valid = false
		status = "revoked"
		warnings = append(warnings, fmt.Sprintf("PoA was revoked on %s", revInfo.RevokedAt.Format(time.RFC3339)))
	}

	// Verify attestations
	if len(poa.Attestations) > 0 {
		for _, att := range poa.Attestations {
			verified, err := v.attestationVerifier.Verify(ctx, att)
			if err != nil || !verified {
				warnings = append(warnings, fmt.Sprintf("Attestation %s could not be verified", att.Type))
			}
		}
	}

	return &VerificationResult{
		Valid:              valid,
		PoAID:              poaID,
		VerifiedAt:         now,
		Status:             status,
		IssuerID:           poa.IssuerID,
		GranteeID:          poa.GranteeID,
		ValidFrom:          poa.ValidFrom,
		ValidUntil:         poa.ValidUntil,
		Scope:              poa.Scope,
		Attestations:       poa.Attestations,
		Warnings:           warnings,
		VerificationMethod: "comprehensive",
	}, nil
}

// VerifyScope verifies that an action falls within the PoA scope
func (v *VerificationServiceImpl) VerifyScope(ctx context.Context, poaID string, action Action) (*ScopeVerificationResult, error) {
	poa, err := v.poaStore.GetPoA(ctx, poaID)
	if err != nil {
		return nil, fmt.Errorf("failed to get PoA: %w", err)
	}

	result := &ScopeVerificationResult{
		PoAID:         poaID,
		Action:        action,
		MatchedScopes: []string{},
		Restrictions:  []RestrictionCheck{},
	}

	// Check if action type is in scope
	authorized := false
	if poa.StructuredScope != nil {
		authorized = v.checkStructuredScope(poa.StructuredScope, action, result)
	} else {
		// Fallback to simple scope checking
		for _, scope := range poa.Scope {
			if scope == action.Type || scope == "*" {
				authorized = true
				result.MatchedScopes = append(result.MatchedScopes, scope)
			}
		}
	}

	// Check restrictions
	if authorized && poa.Restrictions != nil {
		authorized = v.checkRestrictions(poa.Restrictions, action, result)
	}

	result.Authorized = authorized
	if !authorized && result.Reason == "" {
		result.Reason = "Action does not fall within authorized scope"
	}

	return result, nil
}

// checkStructuredScope checks action against structured scope
func (v *VerificationServiceImpl) checkStructuredScope(scope *StructuredScope, action Action, result *ScopeVerificationResult) bool {
	authorized := false

	switch action.Type {
	case "transaction":
		for _, t := range scope.Transactions {
			if t == action.Operation || t == "*" {
				authorized = true
				result.MatchedScopes = append(result.MatchedScopes, fmt.Sprintf("transaction:%s", t))
			}
		}
	case "decision":
		for _, d := range scope.Decisions {
			if d == action.Operation || d == "*" {
				authorized = true
				result.MatchedScopes = append(result.MatchedScopes, fmt.Sprintf("decision:%s", d))
			}
		}
	case "action":
		for _, a := range scope.Actions {
			if a == action.Operation || a == "*" {
				authorized = true
				result.MatchedScopes = append(result.MatchedScopes, fmt.Sprintf("action:%s", a))
			}
		}
	}

	return authorized
}

// checkRestrictions checks action against all restrictions
func (v *VerificationServiceImpl) checkRestrictions(restrictions *Restrictions, action Action, result *ScopeVerificationResult) bool {
	allSatisfied := true

	// Check value limits
	if restrictions.ValueLimits != nil && action.Value != nil {
		valueLimitCheck := &ValueLimitCheck{
			RequestedValue: *action.Value,
			MaxValue:       restrictions.ValueLimits.MaxSingleTransaction,
			Currency:       restrictions.ValueLimits.Currency,
		}

		if *action.Value <= restrictions.ValueLimits.MaxSingleTransaction {
			valueLimitCheck.Within = true
			result.Restrictions = append(result.Restrictions, RestrictionCheck{
				Type:        "value_limit",
				Description: fmt.Sprintf("Transaction value within limit of %.2f %s", restrictions.ValueLimits.MaxSingleTransaction, restrictions.ValueLimits.Currency),
				Satisfied:   true,
			})
		} else {
			valueLimitCheck.Within = false
			allSatisfied = false
			result.Restrictions = append(result.Restrictions, RestrictionCheck{
				Type:        "value_limit",
				Description: fmt.Sprintf("Transaction value exceeds limit of %.2f %s", restrictions.ValueLimits.MaxSingleTransaction, restrictions.ValueLimits.Currency),
				Satisfied:   false,
				Details:     fmt.Sprintf("Requested: %.2f, Max: %.2f", *action.Value, restrictions.ValueLimits.MaxSingleTransaction),
			})
			result.Reason = "Transaction value exceeds authorized limit"
		}

		result.ValueLimitCheck = valueLimitCheck
	}

	// Check geographic restrictions
	if restrictions.GeographicRestrictions != nil && action.Location != nil {
		geoCheck := v.checkGeographicRestrictions(restrictions.GeographicRestrictions, action.Location)
		result.GeographicCheck = geoCheck

		if !geoCheck.Allowed {
			allSatisfied = false
			result.Restrictions = append(result.Restrictions, RestrictionCheck{
				Type:        "geographic",
				Description: "Geographic restriction violated",
				Satisfied:   false,
				Details:     geoCheck.Reason,
			})
			result.Reason = geoCheck.Reason
		} else {
			result.Restrictions = append(result.Restrictions, RestrictionCheck{
				Type:        "geographic",
				Description: "Geographic constraints satisfied",
				Satisfied:   true,
			})
		}
	}

	return allSatisfied
}

// checkGeographicRestrictions checks geographic constraints
func (v *VerificationServiceImpl) checkGeographicRestrictions(constraints *GeographicConstraints, location *GeographicLocation) *GeographicCheck {
	check := &GeographicCheck{
		RequestedLocation: location.Country,
		AllowedCountries:  constraints.AllowedCountries,
		ExcludedRegions:   constraints.ExcludedRegions,
	}

	// Check if country is in allowed list
	if len(constraints.AllowedCountries) > 0 {
		allowed := false
		for _, country := range constraints.AllowedCountries {
			if country == location.Country {
				allowed = true
				break
			}
		}
		if !allowed {
			check.Allowed = false
			check.Reason = fmt.Sprintf("Country %s is not in the allowed list", location.Country)
			return check
		}
	}

	// Check if region is in excluded list
	for _, region := range constraints.ExcludedRegions {
		if region == location.Region || region == location.Country {
			check.Allowed = false
			check.Reason = fmt.Sprintf("Region %s is excluded", region)
			return check
		}
	}

	check.Allowed = true
	return check
}

// CheckRevocationStatus checks if a PoA has been revoked
func (v *VerificationServiceImpl) CheckRevocationStatus(ctx context.Context, poaID string) (*RevocationStatusResult, error) {
	revoked, revInfo, err := v.poaStore.IsRevoked(ctx, poaID)
	if err != nil {
		return nil, fmt.Errorf("failed to check revocation status: %w", err)
	}

	result := &RevocationStatusResult{
		Revoked:   revoked,
		PoAID:     poaID,
		CheckedAt: time.Now(),
	}

	if revoked && revInfo != nil {
		result.RevokedAt = &revInfo.RevokedAt
		result.RevokedBy = revInfo.RevokedBy
		result.Reason = revInfo.Reason
		result.EffectiveDate = &revInfo.RevokedAt
		result.NotifiedParties = revInfo.NotifiedParties
	}

	return result, nil
}

// VerifyPrincipalStatus verifies the principal's legal capacity
func (v *VerificationServiceImpl) VerifyPrincipalStatus(ctx context.Context, principalID string) (*PrincipalStatusResult, error) {
	res, err := v.principalVerifier.VerifyPrincipal(ctx, principalID)
	if err != nil {
		return nil, err
	}

	// Sign for authoritative attestations (if signer is available)
	if v.attestationSigner != nil && res.Valid {
		att := Attestation{
			ID:         uuid.New().String(),
			Type:       "compliance",
			Provider:   v.attestationSigner.SignerID(),
			VerifiedAt: time.Now(),
			Status:     "verified",
			Verified:   true,
			Metadata: map[string]interface{}{
				"principal_id": res.PrincipalID,
				"entity_type":  res.EntityType,
				"status":       res.Status,
			},
		}

		proof, err := v.attestationSigner.Sign(ctx, att)
		if err == nil {
			att.Metadata["proof"] = proof
			res.Attestation = &att
		}
	}

	return res, nil
}

// VerifyRepresentativePosition verifies the representative's position
func (v *VerificationServiceImpl) VerifyRepresentativePosition(ctx context.Context, repID string, orgID string) (*PositionVerificationResult, error) {
	if v.registerService == nil {
		// Fallback to basic implementation if service not available
		return &PositionVerificationResult{
			Valid:            true,
			RepresentativeID: repID,
			OrganizationID:   orgID,
			Position:         "authorized_representative",
			AuthorizedToAct:  true,
			SigninagentAuthority: true,
			EffectiveDate:    time.Now().AddDate(-1, 0, 0),
			VerifiedAt:       time.Now(),
		}, nil
	}

	// Try to verify through commercial register
	// We need to resolve orgID to a registration number and jurisdiction
	// For this prototype, we'll assume orgID is structured as "REG-NUMBER:JURISDICTION"
	var regNum, jurisdiction string
	sepIdx := strings.LastIndex(orgID, ":")
	if sepIdx != -1 {
		regNum = orgID[:sepIdx]
		jurisdiction = orgID[sepIdx+1:]
	} else {
		// If not structured, try default jurisdiction or fail
		regNum = orgID
		jurisdiction = "DE" // Default to DE for demo
	}

	req := &registry.RepresentativeVerificationRequest{
		RepresentativeName: repID, // Assuming repID is the name for this lookup
		EntityRegistration: regNum,
		Jurisdiction:       jurisdiction,
	}

	res, err := v.registerService.VerifyAuthorizedRepresentative(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("commercial register verification failed: %w", err)
	}

	result := &PositionVerificationResult{
		Valid:            res.Verified,
		RepresentativeID: repID,
		OrganizationID:   orgID,
		Position:         res.Position,
		AuthorizedToAct:  res.Verified,
		SigninagentAuthority: res.SignatureAuthority == "sole" || res.SignatureAuthority == "joint",
		EffectiveDate:    res.AppointmentDate,
		VerifiedAt:       res.VerificationDate,
	}

	if !res.Verified {
		result.Issues = append(result.Issues, "Representative not found in commercial register or authority could not be confirmed")
	} else if v.attestationSigner != nil {
		// Sign the positive verification result
		att := Attestation{
			ID:         uuid.New().String(),
			Type:       "commercial_register",
			Provider:   v.attestationSigner.SignerID(),
			VerifiedAt: time.Now(),
			Status:     "verified",
			Verified:   true,
			Metadata: map[string]interface{}{
				"representative_id": repID,
				"organization_id":   orgID,
				"position":          res.Position,
				"effective_date":    res.AppointmentDate,
			},
		}

		proof, err := v.attestationSigner.Sign(ctx, att)
		if err == nil {
			att.Metadata["proof"] = proof
			result.Attestation = &att
		}
	}

	return result, nil
}

// GenerateVerificationReport generates a comprehensive verification report
func (v *VerificationServiceImpl) GenerateVerificationReport(ctx context.Context, poaID string, action Action) (*VerificationReport, error) {
	report := &VerificationReport{
		ReportID:        fmt.Sprintf("VR-%s-%d", poaID, time.Now().Unix()),
		GeneratedAt:     time.Now(),
		PoAID:           poaID,
		RequestedAction: action,
		Warnings:        []string{},
		Recommendations: []string{},
	}

	// Verify PoA
	poaVerif, err := v.VerifyPowerOfAttorney(ctx, poaID)
	if err != nil {
		return nil, fmt.Errorf("PoA verification failed: %w", err)
	}
	report.PoAVerification = poaVerif

	// Verify scope
	scopeVerif, err := v.VerifyScope(ctx, poaID, action)
	if err != nil {
		return nil, fmt.Errorf("scope verification failed: %w", err)
	}
	report.ScopeVerification = scopeVerif

	// Check revocation
	revStatus, err := v.CheckRevocationStatus(ctx, poaID)
	if err != nil {
		report.Warnings = append(report.Warnings, fmt.Sprintf("Could not verify revocation status: %v", err))
	} else {
		report.RevocationStatus = revStatus
	}

	// Verify principal status
	principalStatus, err := v.VerifyPrincipalStatus(ctx, poaVerif.IssuerID)
	if err != nil {
		report.Warnings = append(report.Warnings, fmt.Sprintf("Could not verify principal status: %v", err))
	} else {
		report.PrincipalStatus = principalStatus
		if principalStatus.Attestation != nil {
			report.Attestations = append(report.Attestations, *principalStatus.Attestation)
		}
	}

	// Verify authorization chain
	chain, err := v.VerifyAuthorizationChain(ctx, poaID)
	if err != nil {
		report.Warnings = append(report.Warnings, fmt.Sprintf("Could not verify authorization chain: %v", err))
	} else {
		report.ChainOfAuthority = chain
	}

	// Fiduciary Compliance check
	violations, err := v.fiduciaryService.GetViolations(ctx, poaID, "")
	if err != nil {
		report.Warnings = append(report.Warnings, fmt.Sprintf("Could not verify fiduciary compliance: %v", err))
	} else {
		report.FiduciaryCompliance = &FiduciaryComplianceCheck{
			Compliant:      len(violations) == 0,
			DutiesChecked:  []string{"care", "loyalty", "good_faith"}, // Example duties
			LastAssessment: time.Now(),
		}
		for _, v := range violations {
			report.FiduciaryCompliance.Violations = append(report.FiduciaryCompliance.Violations, FiduciaryViolation{
				Duty:        v.DutyType,
				Description: v.ViolationDescription,
				Severity:    v.Severity,
				DetectedAt:  v.DetectedAt,
			})
		}
	}

	// Capability Check
	capAssessment, err := v.capabilityService.GetLatestAssessment(ctx, poaVerif.GranteeID)
	if err != nil {
		report.Warnings = append(report.Warnings, fmt.Sprintf("Could not verify capability assessment: %v", err))
	} else {
		// Use CheckCapabilityMatch if we have the PoA for requirements
		poa, _ := v.poaStore.GetPoA(ctx, poaID)
		var sufficient bool
		var deficient []string
		if poa != nil && poa.CapabilityReqs != nil {
			sufficient, deficient, _ = v.capabilityService.CheckCapabilityMatch(ctx, poaVerif.GranteeID, poa.CapabilityReqs)
		} else {
			sufficient = true // Assume sufficient if no reqs
		}

		report.CapabilityCheck = &CapabilityVerificationCheck{
			Sufficient:      sufficient,
			RequiredLevel:   "N/A", // From PoA if needed
			ActualLevel:     capAssessment.OverallLevel,
			RequiredDomains: nil,
			ActualDomains:   capAssessment.DomainScores,
			DeficientAreas:  deficient,
			LastAssessment:  capAssessment.AssessmentDate,
		}
	}

	// Overall validity
	// 1. Core PoA must be valid and active
	// 2. Scope must be authorized
	// 3. PoA must not be revoked
	// 4. MUST have human accountability (Human-at-top requirement RR-014)
	// 5. MUST not exceed max delegation depth

	hasHumanAccountability := false
	maxDepthExceeded := false
	if len(chain) > 0 {
		hasHumanAccountability = chain[len(chain)-1].IsHuman

		// Check depth limits (RR-014)
		actualDepth := len(chain) - 1
		// Get root PoA policy for depth limit
		rootPoA, _ := v.poaStore.GetPoA(ctx, chain[len(chain)-1].PoAID)
		if rootPoA != nil && rootPoA.DelegationPolicy != nil && rootPoA.DelegationPolicy.MaxDepth > 0 {
			if actualDepth > rootPoA.DelegationPolicy.MaxDepth {
				maxDepthExceeded = true
				report.Warnings = append(report.Warnings, fmt.Sprintf("Delegation depth %d exceeds max allowed %d", actualDepth, rootPoA.DelegationPolicy.MaxDepth))
			}
		}
	}

	if !hasHumanAccountability {
		report.Warnings = append(report.Warnings, "Missing human accountability: Root of authority chain is not a human principal")
	}

	report.OverallValid = poaVerif.Valid && scopeVerif.Authorized && !revStatus.Revoked && hasHumanAccountability && !maxDepthExceeded

	// Set validity period
	report.ValidityPeriod = fmt.Sprintf("%s to %s",
		poaVerif.ValidFrom.Format("2006-01-02"),
		poaVerif.ValidUntil.Format("2006-01-02"))

	// Set next review date (30 days or expiry, whichever is sooner)
	nextReview := time.Now().AddDate(0, 0, 30)
	if nextReview.After(poaVerif.ValidUntil) {
		nextReview = poaVerif.ValidUntil
	}
	report.NextReviewDate = nextReview

	// Add recommendations
	if time.Until(poaVerif.ValidUntil) < 30*24*time.Hour {
		report.Recommendations = append(report.Recommendations, "PoA expires within 30 days - consider renewal")
	}
	if !hasHumanAccountability {
		report.Recommendations = append(report.Recommendations, "Establish a human-anchored Power of Attorney for this agent")
	}

	return report, nil
}

// VerifyAuthorizationChain verifies the complete chain of authority
// VerifyAuthorizationChain traces the authority from the actor back to a human principal
func (v *VerificationServiceImpl) VerifyAuthorizationChain(ctx context.Context, poaID string) ([]AuthorityLink, error) {
	chain := []AuthorityLink{}
	currentID := poaID
	depth := 0
	maxDepth := 10 // Safety limit

	for depth < maxDepth {
		// Get current link (PoA or Delegation)
		node, err := v.poaStore.GetPoA(ctx, currentID)
		if err != nil {
			return nil, fmt.Errorf("failed to get authority link %s: %w", currentID, err)
		}

		// Add current link
		chain = append(chain, AuthorityLink{
			Level:           depth,
			PoAID:           node.ID,
			IssuerID:        node.IssuerID,
			GranteeID:       node.GranteeID,
			GrantedAt:       node.CreatedAt,
			IsHuman:         v.isHumanEntity(node.IssuerID),
			DelegationDepth: depth,
		})

		// If the issuer is a human, the chain is complete and accounted for
		if v.isHumanEntity(node.IssuerID) {
			break
		}

		// Otherwise, try to find the power that authorized this issuer
		// If it's a delegation, we follow the source agent's authority for the same root PoA
		if node.SourcePOAID != nil {
			// Find the delegation where the target is the current issuer
			// and belongs to the same root PoA
			delegations, err := v.delegationService.GetDelegationChain(ctx, node.IssuerID)
			if err != nil || len(delegations) == 0 {
				// Try checking if the issuer has a direct PoA (PoA 1)
				rootPoA, err := v.poaStore.GetPoA(ctx, *node.SourcePOAID)
				if err == nil && rootPoA.GranteeID == node.IssuerID {
					currentID = rootPoA.ID
				} else {
					return nil, fmt.Errorf("interrupted authority chain: could not find source for %s", node.IssuerID)
				}
			} else {
				// Use the delegation that matches the source PoA
				found := false
				for _, d := range delegations {
					if d.SourcePOAID == *node.SourcePOAID {
						currentID = d.ID
						found = true
						break
					}
				}
				if !found {
					// Fallback to root PoA check
					rootPoA, err := v.poaStore.GetPoA(ctx, *node.SourcePOAID)
					if err == nil && rootPoA.GranteeID == node.IssuerID {
						currentID = rootPoA.ID
					} else {
						return nil, fmt.Errorf("interrupted authority chain: delegation for %s not found", node.IssuerID)
					}
				}
			}
		} else {
			// This is a direct PoA but issuer is NOT human (e.g. an AI system issuing powers?)
			// AgentAuth+ requires a human at the top.
			// Check if this AI was delegated power by someone else?
			// But if SourcePOAID is nil, it's a root. If root issuer isn't human, it's invalid in our policy.
			break
		}

		depth++
	}

	// Basic check: top link MUST be human
	if len(chain) > 0 {
		topLink := chain[len(chain)-1]
		if !topLink.IsHuman {
			// Note: We don't error here to allow the caller to decide based on warnings,
			// but we could mark the chain as invalid.
		}
	}

	return chain, nil
}

// VerifyAttestations verifies all attestations associated with a PoA
func (v *VerificationServiceImpl) VerifyAttestations(ctx context.Context, poaID string) ([]Attestation, error) {
	poa, err := v.poaStore.GetPoA(ctx, poaID)
	if err != nil {
		return nil, fmt.Errorf("failed to get PoA: %w", err)
	}

	verifiedAttestations := []Attestation{}

	if v.attestationVerifier == nil {
		// If no verifier, we can't confirm cryptographic proofs, but we return existing status
		return poa.Attestations, nil
	}

	for _, att := range poa.Attestations {
		verified, err := v.attestationVerifier.Verify(ctx, att)
		if err == nil {
			att.Verified = verified
			if verified {
				att.Status = "verified"
			}
		} else {
			// Log error if needed, but continue with others
			att.Verified = false
			att.Status = "verification_failed"
		}
		verifiedAttestations = append(verifiedAttestations, att)
	}

	return verifiedAttestations, nil
}

// isHumanEntity checks if an entity ID represents a human (simplified)
func (v *VerificationServiceImpl) isHumanEntity(entityID string) bool {
	if v.principalVerifier == nil {
		// Fallback to simple heuristic: humans don't have "AI" or "BOT" prefix
		return !strings.HasPrefix(strings.ToUpper(entityID), "AI") &&
			!strings.HasPrefix(strings.ToUpper(entityID), "BOT")
	}

	// Use PrincipalVerifier if available
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	res, err := v.principalVerifier.VerifyPrincipal(ctx, entityID)
	if err != nil {
		return false
	}

	return res.EntityType == "individual" || res.EntityType == "human"
}

// Package poa provides Power-of-Attorney functionality
// This is a compatibility alias for the rfc0111 package
package poa

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/errors"
)

// Simplified local POA status constants (legacy compatibility subset)
type POAStatus string

const (
	POAStatusActive  POAStatus = "active"
	POAStatusRevoked POAStatus = "revoked"
	POAStatusExpired POAStatus = "expired"
	POAStatusPending POAStatus = "pending"
)

type ClientOwnerInfo struct {
	Name                      string
	RegisteredPowerOfAttorney bool
	CommercialRegisterEntry   bool
}

type AuthorizedClient struct {
	Type              string
	Identity          string
	Version           string
	OperationalStatus string
}

// ...existing code...

// Example constants for demo compatibility

type AuthorizationScope struct {
	AuthorizationType AuthorizationType
	ApplicableSectors []IndustrySector
	ApplicableRegions []GeographicScope
	AuthorizedActions AuthorizedActions
}

type AuthorizationType struct {
	RepresentationType string
	Restrictions       []string
	SubProxyAuthority  bool
	SignatureType      string
}

type (
	IndustrySector  string
	GeographicScope struct {
		Type       string
		Identifier string
	}
)

type AuthorizedActions struct {
	Transactions       []Transaction
	Decisions          []Decision
	NonPhysicalActions []NonPhysicalAction
}

type (
	Transaction       string
	Decision          string
	NonPhysicalAction string
)

type Requirements struct {
	ValidityPeriod       ValidityPeriod
	FormalRequirements   FormalRequirements
	PowerLimits          PowerLimits
	RightsObligations    RightsObligations
	SpecialConditions    SpecialConditions
	DeathIncapacityRules DeathIncapacityRules
	SecurityCompliance   SecurityCompliance
	JurisdictionLaw      JurisdictionLaw
	ConflictResolution   ConflictResolution
}

type ValidityPeriod struct {
	StartTime             time.Time
	EndTime               time.Time
	AutoRenewalConditions []string
	TerminationConditions []string
}

type FormalRequirements struct {
	NotarialCertification  bool
	IDVerificationRequired bool
	DigitalSignatures      bool
}

type PowerLimits struct {
	PowerLevels        []string
	InteractionBounds  []string
	ToolLimitations    []string
	QuantumResistance  bool
	ExplicitExclusions []string
}

type RightsObligations struct {
	ReportingDuties   []string
	LiabilityRules    []string
	CompensationRules []string
}

type SpecialConditions struct {
	ConditionalEffectiveness []string
	ImmediateNotification    []string
}

type DeathIncapacityRules struct {
	ContinuationOnDeath    bool
	IncapacityInstructions string
}

type SecurityCompliance struct {
	CommunicationProtocols []string
	SecurityProperties     []string
	ComplianceInfo         []string
	UpdateMechanism        string
}

type JurisdictionLaw struct {
	Language            string
	GoverningLaw        string
	PlaceOfJurisdiction string
	AttachedDocuments   []string
}

type ConflictResolution struct {
	ArbitrationJurisdiction string
}

// --- RFC-0115 PoA Definition Compatibility Types ---

// PoADefinition aggregates all sections of a Power-of-Attorney definition.
type PoADefinition struct {
	Parties       Parties            `json:"parties"`
	Authorization AuthorizationScope `json:"authorization"`
	Requirements  Requirements       `json:"requirements"`
}

// Parties encapsulates principal, representative, and authorized client.
type Parties struct {
	Principal        Principal        `json:"principal"`
	Representative   *Representative  `json:"representative,omitempty"`
	AuthorizedClient AuthorizedClient `json:"authorized_client"`
}

// Principal represents the principal party (organization or individual)
type Principal struct {
	Type         string        `json:"type"`
	Identity     string        `json:"identity"`
	Organization *Organization `json:"organization,omitempty"`
}

// Organization contains registration info for the principal organization.
type Organization struct {
	Type                string `json:"type"`
	Name                string `json:"name"`
	RegisterEntry       string `json:"register_entry"`
	ManagingDirector    string `json:"managing_director"`
	RegisteredAuthority bool   `json:"registered_authority"`
}

// Representative contains client owner info linking to authorization.
type Representative struct {
	ClientOwner *ClientOwnerInfo `json:"client_owner,omitempty"`
}

// ValidatePoADefinition performs minimal structural validation for the RFC-0115 example.
func ValidatePoADefinition(def PoADefinition) error {
	if def.Parties.Principal.Identity == "" {
		return fmt.Errorf("principal identity required")
	}
	if def.Parties.AuthorizedClient.Identity == "" {
		return fmt.Errorf("authorized client identity required")
	}
	// Basic validity period sanity check
	if def.Requirements.ValidityPeriod.EndTime.Before(def.Requirements.ValidityPeriod.StartTime) {
		return fmt.Errorf("validity period end before start")
	}
	return nil
}

// Example constants for demo compatibility
const (
	PrincipalTypeOrganization = "Organization"
	OrgTypeNonProfit          = "NonProfit"
	ClientTypeLLM             = "LLM"
	RepresentationSole        = "Sole"
	SignatureSingle           = "Single"
	SectorInformationComm     = IndustrySector("InformationComm")
	SectorProfessional        = IndustrySector("Professional")
	SectorFinancialInsurance  = IndustrySector("FinancialInsurance")
	GeoTypeNational           = "National"
	GeoTypeRegional           = "Regional"
	TransactionLoan           = Transaction("Loan")
	TransactionPurchase       = Transaction("Purchase")
	DecisionFinancial         = Decision("Financial")
	DecisionStrategic         = Decision("Strategic")
	DecisionInfoSharing       = Decision("InfoSharing")
	ActionResearching         = NonPhysicalAction("Researching")
	ActionBrainstorming       = NonPhysicalAction("Brainstorming")
)

// Stub functions for RFC-0115 demo compatibility
// RFC0115Config models exclusion flags & limits referenced by RFC0111/0115 examples.
type RFC0115Config struct {
	ExcludeWeb3          bool
	ExcludeAIOperators   bool
	ExcludeDNAIdentities bool
	MaxValidityDays      int // upper bound for validity period
}

func CreateRFC0115CompliantConfig() interface{} {
	return RFC0115Config{ExcludeWeb3: true, ExcludeAIOperators: true, ExcludeDNAIdentities: true, MaxValidityDays: 365}
}

// ValidateRFC0115Compliance performs structural & semantic checks on PoA definition + config.
// Accepts either RFC0115Config, PoADefinition, or a composite struct {Config, Definition}.
func ValidateRFC0115Compliance(config interface{}) error {
	// Allow passing PoADefinition directly for backward examples.
	switch v := config.(type) {
	case RFC0115Config:
		if !v.ExcludeWeb3 || !v.ExcludeAIOperators || !v.ExcludeDNAIdentities {
			return fmt.Errorf("all exclusion flags must be true (web3, ai operators, dna identities)")
		}
		if v.MaxValidityDays <= 0 || v.MaxValidityDays > 730 {
			return fmt.Errorf("max validity days out of acceptable bounds: %d", v.MaxValidityDays)
		}
		return nil
	case PoADefinition:
		if err := ValidatePoADefinition(v); err != nil {
			return err
		}
		// Semantic checks: ensure at least one sector, action, and region.
		if len(v.Authorization.ApplicableSectors) == 0 {
			return fmt.Errorf("authorization must include at least one sector")
		}
		if len(v.Authorization.ApplicableRegions) == 0 {
			return fmt.Errorf("authorization must include at least one region")
		}
		if len(v.Authorization.AuthorizedActions.Transactions) == 0 && len(v.Authorization.AuthorizedActions.Decisions) == 0 && len(v.Authorization.AuthorizedActions.NonPhysicalActions) == 0 {
			return fmt.Errorf("authorization must include at least one action (transaction/decision/non-physical)")
		}
		// Validity duration sanity relative to 0 < EndTime-StartTime <= 2y
		dur := v.Requirements.ValidityPeriod.EndTime.Sub(v.Requirements.ValidityPeriod.StartTime)
		if dur <= 0 {
			return fmt.Errorf("validity period must be positive duration")
		}
		if dur > (time.Hour * 24 * 730) {
			return fmt.Errorf("validity period exceeds 2 years")
		}
		return nil
	default:
		// Attempt composite via reflection-like pattern
		// Accept map[string]any with keys "config" and/or "definition"
		if m, ok := config.(map[string]interface{}); ok {
			if cfgRaw, ok2 := m["config"]; ok2 {
				if err := ValidateRFC0115Compliance(cfgRaw); err != nil {
					return fmt.Errorf("config invalid: %w", err)
				}
			}
			if defRaw, ok2 := m["definition"]; ok2 {
				if err := ValidateRFC0115Compliance(defRaw); err != nil {
					return fmt.Errorf("definition invalid: %w", err)
				}
			}
			return nil
		}
		return fmt.Errorf("unsupported RFC0115 compliance object type %T", config)
	}
}

// ProofOfAuthorization represents a proof of authorization token
type ProofOfAuthorization struct {
	ID          string                 `json:"id"`
	Subject     string                 `json:"subject"`
	Resource    string                 `json:"resource"`
	Action      string                 `json:"action"`
	Issuer      string                 `json:"issuer"`
	IssuedAt    time.Time              `json:"issued_at"`
	ExpiresAt   time.Time              `json:"expires_at"`
	Scope       []string               `json:"scope"`
	Delegation  *Delegation            `json:"delegation,omitempty"`
	Attestation *Attestation           `json:"attestation,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// Delegation represents delegation information
type Delegation struct {
	DelegatedBy string    `json:"delegated_by"`
	DelegatedTo string    `json:"delegated_to"`
	DelegatedAt time.Time `json:"delegated_at"`
	ExpiresAt   time.Time `json:"expires_at"`
	Scope       []string  `json:"scope"`
	Constraints []string  `json:"constraints,omitempty"`
	Revocable   bool      `json:"revocable"`
}

// Attestation represents attestation information
type Attestation struct {
	AttestedBy    string                 `json:"attested_by"`
	AttestedAt    time.Time              `json:"attested_at"`
	Evidence      map[string]interface{} `json:"evidence"`
	Confidence    float64                `json:"confidence"`
	ValidityScore float64                `json:"validity_score"`
}

// Request represents a PoA request
type Request struct {
	Subject    string                 `json:"subject"`
	Resource   string                 `json:"resource"`
	Action     string                 `json:"action"`
	Scope      []string               `json:"scope,omitempty"`
	Delegation *DelegationRequest     `json:"delegation,omitempty"`
	Context    map[string]interface{} `json:"context,omitempty"`
}

// DelegationRequest represents a delegation request
type DelegationRequest struct {
	DelegatedBy string        `json:"delegated_by"`
	Scope       []string      `json:"scope"`
	Duration    time.Duration `json:"duration"`
	Constraints []string      `json:"constraints,omitempty"`
}

// Service defines the PoA service interface
type Service interface {
	Issue(ctx context.Context, req *Request) (*ProofOfAuthorization, error)
	Validate(ctx context.Context, poa *ProofOfAuthorization) error
	Revoke(ctx context.Context, poaID string) error
	List(ctx context.Context, subject string) ([]*ProofOfAuthorization, error)
}

// MemoryService implements the PoA service using in-memory storage
type MemoryService struct {
	proofs  map[string]*ProofOfAuthorization
	revoked map[string]bool
}

// NewMemoryService creates a new memory-based PoA service
func NewMemoryService() *MemoryService {
	return &MemoryService{
		proofs:  make(map[string]*ProofOfAuthorization),
		revoked: make(map[string]bool),
	}
}

// Issue issues a new proof of authorization
func (s *MemoryService) Issue(ctx context.Context, req *Request) (*ProofOfAuthorization, error) {
	if req.Subject == "" || req.Resource == "" || req.Action == "" {
		return nil, errors.New(errors.ErrCodeValidation, "subject, resource, and action are required")
	}

	poa := &ProofOfAuthorization{
		ID:        generateID(),
		Subject:   req.Subject,
		Resource:  req.Resource,
		Action:    req.Action,
		Issuer:    "gauth-poa-service",
		IssuedAt:  time.Now(),
		ExpiresAt: time.Now().Add(time.Hour), // Default 1 hour
		Scope:     req.Scope,
		Metadata:  req.Context,
	}

	// Handle delegation if requested
	if req.Delegation != nil {
		poa.Delegation = &Delegation{
			DelegatedBy: req.Delegation.DelegatedBy,
			DelegatedTo: req.Subject,
			DelegatedAt: time.Now(),
			ExpiresAt:   time.Now().Add(req.Delegation.Duration),
			Scope:       req.Delegation.Scope,
			Constraints: req.Delegation.Constraints,
			Revocable:   true,
		}
	}

	// Add basic attestation
	poa.Attestation = &Attestation{
		AttestedBy:    "gauth-attestation-service",
		AttestedAt:    time.Now(),
		Evidence:      make(map[string]interface{}),
		Confidence:    0.95,
		ValidityScore: 0.98,
	}

	s.proofs[poa.ID] = poa
	return poa, nil
}

// Validate validates a proof of authorization
func (s *MemoryService) Validate(ctx context.Context, poa *ProofOfAuthorization) error {
	if poa == nil {
		return errors.New(errors.ErrCodeValidation, "PoA is required")
	}

	// Check if revoked
	if s.revoked[poa.ID] {
		return errors.New(errors.ErrCodeUnauthorized, "PoA has been revoked")
	}

	// Check expiration
	if time.Now().After(poa.ExpiresAt) {
		return errors.New(errors.ErrCodeExpiredToken, "PoA has expired")
	}

	// Validate delegation if present
	if poa.Delegation != nil {
		if time.Now().After(poa.Delegation.ExpiresAt) {
			return errors.New(errors.ErrCodeExpiredToken, "delegation has expired")
		}
	}

	return nil
}

// Revoke revokes a proof of authorization
func (s *MemoryService) Revoke(ctx context.Context, poaID string) error {
	if _, exists := s.proofs[poaID]; !exists {
		return errors.New(errors.ErrCodeNotFound, "PoA not found")
	}

	s.revoked[poaID] = true
	return nil
}

// List lists all PoAs for a subject
func (s *MemoryService) List(ctx context.Context, subject string) ([]*ProofOfAuthorization, error) {
	var result []*ProofOfAuthorization

	for _, poa := range s.proofs {
		if poa.Subject == subject && !s.revoked[poa.ID] {
			result = append(result, poa)
		}
	}

	return result, nil
}

// generateID generates a random ID
func generateID() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "poa_fallback_id"
	}
	return fmt.Sprintf("poa_%s", hex.EncodeToString(bytes))
}

// CreateDelegationAttestation creates a delegation attestation
func CreateDelegationAttestation(delegatedBy, delegatedTo string, scope []string) *Delegation {
	return &Delegation{
		DelegatedBy: delegatedBy,
		DelegatedTo: delegatedTo,
		DelegatedAt: time.Now(),
		ExpiresAt:   time.Now().Add(24 * time.Hour),
		Scope:       scope,
		Revocable:   true,
	}
}

// CreateAttestation creates an attestation with evidence
func CreateAttestation(attestedBy string, evidence map[string]interface{}) *Attestation {
	return &Attestation{
		AttestedBy:    attestedBy,
		AttestedAt:    time.Now(),
		Evidence:      evidence,
		Confidence:    0.90,
		ValidityScore: 0.95,
	}
}

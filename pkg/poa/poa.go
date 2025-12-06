// Package poa provides Power-of-Attorney functionality
// This is a compatibility alias for the rfc0111 package
package poa

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	internalCrypto "github.com/mauriciomferz/Gauth_go/pkg/crypto"
	"github.com/mauriciomferz/Gauth_go/pkg/errors"
	"github.com/mauriciomferz/Gauth_go/pkg/poa/taxonomy"
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

// RFC-0115 Section A.3 Client Type Classification
type ClientType string

const (
	ClientTypeLLM           ClientType = "LLM"
	ClientTypeDigitalAgent  ClientType = "DigitalAgent"
	ClientTypeAgenticAI     ClientType = "AgenticAI"
	ClientTypeHumanoidRobot ClientType = "HumanoidRobot"
	ClientTypeRoboticSystem ClientType = "RoboticSystem"
	ClientTypeOther         ClientType = "Other"
)

type OperationalStatus string

const (
	OperationalStatusActive         OperationalStatus = "active"
	OperationalStatusSuspended      OperationalStatus = "suspended"
	OperationalStatusRevoked        OperationalStatus = "revoked"
	OperationalStatusMaintenance    OperationalStatus = "maintenance"
	OperationalStatusTesting        OperationalStatus = "testing"
	OperationalStatusDecommissioned OperationalStatus = "decommissioned"
)

type CapabilityLevel string

const (
	CapabilityL0 CapabilityLevel = "L0_no_automation"
	CapabilityL1 CapabilityLevel = "L1_assistance"
	CapabilityL2 CapabilityLevel = "L2_partial_automation"
	CapabilityL3 CapabilityLevel = "L3_conditional_automation"
	CapabilityL4 CapabilityLevel = "L4_high_automation"
	CapabilityL5 CapabilityLevel = "L5_full_automation"
)

type AuthorizedClient struct {
	Type              string // Legacy: kept for backward compatibility (use TypeEnum)
	TypeEnum          ClientType
	Identity          string
	Version           string
	OperationalStatus string // Legacy: kept for backward compatibility (use StatusEnum)
	StatusEnum        OperationalStatus
	CapabilityLevel   CapabilityLevel
	// Team composition for AgenticAI
	TeamComposition []string
	LeadAgent       string
	// Physical attributes for robots
	PhysicalAttributes *PhysicalAttributes
	// Model attributes for LLMs/digital agents
	ModelAttributes *ModelAttributes
	// Certifications
	Certifications []Certification
}

type PhysicalAttributes struct {
	Height               float64
	Weight               float64
	Mobility             string
	ManipulatorCount     int
	SensorSuite          []string
	MaxPayload           float64
	OperatingEnvironment []string
}

type ModelAttributes struct {
	Architecture     string
	ParameterCount   int64
	TrainingData     []string
	Modalities       []string
	ContextWindow    int
	ReasoningMethods []string
}

type Certification struct {
	Type              string
	IssuingAuthority  string
	CertificateNumber string
	ValidFrom         string
	ValidUntil        string
	Scope             string
}

// ValidateClientType validates RFC-0115 client type
func ValidateClientType(ct ClientType) error {
	switch ct {
	case ClientTypeLLM, ClientTypeDigitalAgent, ClientTypeAgenticAI,
		ClientTypeHumanoidRobot, ClientTypeRoboticSystem, ClientTypeOther:
		return nil
	default:
		return fmt.Errorf("invalid client type: %s", ct)
	}
}

// ValidateOperationalStatus validates RFC-0115 operational status
func ValidateOperationalStatus(status OperationalStatus) error {
	switch status {
	case OperationalStatusActive, OperationalStatusSuspended, OperationalStatusRevoked,
		OperationalStatusMaintenance, OperationalStatusTesting, OperationalStatusDecommissioned:
		return nil
	default:
		return fmt.Errorf("invalid operational status: %s", status)
	}
}

// ValidateCapabilityLevel validates autonomy capability level
func ValidateCapabilityLevel(level CapabilityLevel) error {
	if level == "" {
		return nil // Optional
	}
	switch level {
	case CapabilityL0, CapabilityL1, CapabilityL2, CapabilityL3, CapabilityL4, CapabilityL5:
		return nil
	default:
		return fmt.Errorf("invalid capability level: %s", level)
	}
}

// CanOperate checks if client can perform operations
func (ac *AuthorizedClient) CanOperate() bool {
	status := ac.StatusEnum
	if status == "" && ac.OperationalStatus != "" {
		status = OperationalStatus(ac.OperationalStatus)
	}
	return status == OperationalStatusActive || status == OperationalStatusTesting
}

// IsPhysicalSystem checks if client has physical embodiment
func (ac *AuthorizedClient) IsPhysicalSystem() bool {
	ct := ac.TypeEnum
	if ct == "" && ac.Type != "" {
		ct = ClientType(ac.Type)
	}
	return ct == ClientTypeHumanoidRobot || ct == ClientTypeRoboticSystem
}

// IsDigitalSystem checks if client is purely software
func (ac *AuthorizedClient) IsDigitalSystem() bool {
	ct := ac.TypeEnum
	if ct == "" && ac.Type != "" {
		ct = ClientType(ac.Type)
	}
	return ct == ClientTypeLLM || ct == ClientTypeDigitalAgent
}

// RequiresTeamCoordination checks if client involves multiple agents
func (ac *AuthorizedClient) RequiresTeamCoordination() bool {
	ct := ac.TypeEnum
	if ct == "" && ac.Type != "" {
		ct = ClientType(ac.Type)
	}
	return ct == ClientTypeAgenticAI
}

// GetRiskLevel returns risk assessment based on type and capability
func (ac *AuthorizedClient) GetRiskLevel() string {
	if ac.IsPhysicalSystem() {
		switch ac.CapabilityLevel {
		case CapabilityL4, CapabilityL5:
			return "high"
		case CapabilityL3:
			return "medium-high"
		case CapabilityL1, CapabilityL2:
			return string(taxonomy.RiskMedium)
		default:
			return "low"
		}
	}
	switch ac.CapabilityLevel {
	case CapabilityL5:
		return "medium-high"
	case CapabilityL4:
		return string(taxonomy.RiskMedium)
	case CapabilityL2, CapabilityL3:
		return "medium-low"
	default:
		return "low"
	}
}

// Validate performs complete validation of authorized client
func (ac *AuthorizedClient) Validate() error {
	// Sync legacy fields if new fields set
	if ac.TypeEnum != "" {
		ac.Type = string(ac.TypeEnum)
	} else if ac.Type != "" {
		ac.TypeEnum = ClientType(ac.Type)
	}
	if ac.StatusEnum != "" {
		ac.OperationalStatus = string(ac.StatusEnum)
	} else if ac.OperationalStatus != "" {
		ac.StatusEnum = OperationalStatus(ac.OperationalStatus)
	}

	// Validate type
	if err := ValidateClientType(ac.TypeEnum); err != nil {
		return fmt.Errorf("client type: %w", err)
	}

	// Validate identity and version
	if strings.TrimSpace(ac.Identity) == "" {
		return fmt.Errorf("client identity required")
	}
	if strings.TrimSpace(ac.Version) == "" {
		return fmt.Errorf("client version required")
	}

	// Validate status
	if err := ValidateOperationalStatus(ac.StatusEnum); err != nil {
		return fmt.Errorf("operational status: %w", err)
	}

	// Validate capability level
	if err := ValidateCapabilityLevel(ac.CapabilityLevel); err != nil {
		return fmt.Errorf("capability level: %w", err)
	}

	// Type-specific validation
	switch ac.TypeEnum {
	case ClientTypeAgenticAI:
		if len(ac.TeamComposition) == 0 {
			return fmt.Errorf("agentic AI must have team composition")
		}
		if ac.LeadAgent == "" {
			return fmt.Errorf("agentic AI must specify lead agent")
		}
		leadFound := false
		for _, member := range ac.TeamComposition {
			if member == ac.LeadAgent {
				leadFound = true
				break
			}
		}
		if !leadFound {
			return fmt.Errorf("lead agent must be in team composition")
		}
	case ClientTypeHumanoidRobot, ClientTypeRoboticSystem:
		if ac.PhysicalAttributes == nil {
			return fmt.Errorf("%s should include physical attributes", ac.TypeEnum)
		}
	case ClientTypeLLM, ClientTypeDigitalAgent:
		if ac.ModelAttributes == nil {
			return fmt.Errorf("%s should include model attributes", ac.TypeEnum)
		}
	}

	return nil
}

// ...existing code...

// Example constants for demo compatibility

type AuthorizationScope struct {
	AuthorizationType AuthorizationType
	ApplicableSectors []taxonomy.IndustrySector // RFC-0115 B.2 Industry Sector (uses full struct from sector_taxonomy.go)
	ApplicableRegions []GeographicScope
	AuthorizedActions AuthorizedActions
}

type AuthorizationType struct {
	RepresentationType string
	Restrictions       []string
	SubProxyAuthority  bool
	SignatureType      string
}

// GeographicScope represents authorized geographic regions per RFC-0115 Section B.3
type GeographicScope struct {
	// Type defines the geographic scope level
	Type GeographicType `json:"type"`

	// Identifier is ISO 3166-1 alpha-2 country code or ISO 3166-2 subdivision code
	Identifier string `json:"identifier"`

	// Name is human-readable geographic name
	Name string `json:"name,omitempty"`

	// IncludeSubdivisions indicates if subdivisions are included
	IncludeSubdivisions bool `json:"include_subdivisions,omitempty"`

	// ExcludedSubdivisions lists explicitly excluded subdivisions
	ExcludedSubdivisions []string `json:"excluded_subdivisions,omitempty"`
}

// GeographicType classifies geographic scope levels per RFC-0115 B.3
type GeographicType string

const (
	GeoTypeGlobal      GeographicType = "Global"
	GeoTypeRegional    GeographicType = "Regional"    // Multi-country region (e.g., "EU", "ASEAN")
	GeoTypeNational    GeographicType = "National"    // Country level (ISO 3166-1)
	GeoTypeSubnational GeographicType = "Subnational" // State/province (ISO 3166-2)
	GeoTypeMunicipal   GeographicType = "Municipal"   // City/local level
)

// ValidateGeographicType validates the geographic type
func ValidateGeographicType(gt GeographicType) error {
	validTypes := []GeographicType{
		GeoTypeGlobal, GeoTypeRegional, GeoTypeNational,
		GeoTypeSubnational, GeoTypeMunicipal,
	}
	for _, valid := range validTypes {
		if gt == valid {
			return nil
		}
	}
	return fmt.Errorf("invalid geographic type: %s", gt)
}

// Validate performs validation of geographic scope
func (gs *GeographicScope) Validate() error {
	if err := ValidateGeographicType(gs.Type); err != nil {
		return fmt.Errorf("type validation: %w", err)
	}

	if gs.Identifier == "" && gs.Type != GeoTypeGlobal {
		return fmt.Errorf("identifier required for non-global scope")
	}

	// Validate ISO 3166 format for national/subnational
	switch gs.Type {
	case GeoTypeNational:
		if len(gs.Identifier) != 2 {
			return fmt.Errorf("national scope requires ISO 3166-1 alpha-2 code (2 chars), got: %s", gs.Identifier)
		}
		// ISO 3166-1 codes are uppercase
		if gs.Identifier != strings.ToUpper(gs.Identifier) {
			return fmt.Errorf("ISO 3166-1 codes must be uppercase: %s", gs.Identifier)
		}
	case GeoTypeSubnational:
		// ISO 3166-2 format: CC-XXX (country code dash subdivision)
		if !strings.Contains(gs.Identifier, "-") {
			return fmt.Errorf("subnational scope requires ISO 3166-2 format (CC-XXX): %s", gs.Identifier)
		}
		parts := strings.Split(gs.Identifier, "-")
		if len(parts[0]) != 2 {
			return fmt.Errorf("ISO 3166-2 country prefix must be 2 chars: %s", gs.Identifier)
		}
	}

	return nil
}

// IsAuthorizedInRegion checks if a specific region is authorized
func IsAuthorizedInRegion(scopes []GeographicScope, checkRegion string) bool {
	for _, scope := range scopes {
		// Global scope authorizes everything
		if scope.Type == GeoTypeGlobal {
			return true
		}

		// Exact match
		if scope.Identifier == checkRegion {
			// Check if region is explicitly excluded
			for _, excluded := range scope.ExcludedSubdivisions {
				if excluded == checkRegion {
					return false
				}
			}
			return true
		}

		// Check subdivision inclusion
		if scope.IncludeSubdivisions && strings.HasPrefix(checkRegion, scope.Identifier+"-") {
			// Check if subdivision is explicitly excluded
			for _, excluded := range scope.ExcludedSubdivisions {
				if excluded == checkRegion {
					return false
				}
			}
			return true
		}
	}

	return false
}

type AuthorizedActions struct {
	Transactions       []taxonomy.TransactionType       // RFC-0115 B.4.1 Transaction types
	Decisions          []taxonomy.DecisionType          // RFC-0115 B.4.2 Decision types
	PhysicalActions    []taxonomy.ActionTypePhysical    // RFC-0115 B.4.3 Physical action types
	NonPhysicalActions []taxonomy.ActionTypeNonPhysical // RFC-0115 B.4.4 Non-physical action types
}

// Legacy type aliases for backward compatibility
type (
	Transaction           = taxonomy.TransactionType
	TransactionType       = taxonomy.TransactionType
	Decision              = taxonomy.DecisionType
	DecisionType          = taxonomy.DecisionType
	NonPhysicalAction     = taxonomy.ActionTypeNonPhysical
	ActionTypeNonPhysical = taxonomy.ActionTypeNonPhysical
	ActionTypePhysical    = taxonomy.ActionTypePhysical

	SectorScope = taxonomy.SectorScope
	// AuthorizedActionSet refers to the taxonomy struct with allow_all flags
	AuthorizedActionSet = taxonomy.AuthorizedActionSet
	IndustrySector      = taxonomy.IndustrySector
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

// PowerLimits is an alias for RFC-0115 Section C.2 comprehensive power limits
// Use PowerLimitSet from power_limits.go for new implementations
type PowerLimits struct {
	// Legacy fields for backward compatibility
	PowerLevels        []string `json:"power_levels,omitempty"`
	InteractionBounds  []string `json:"interaction_bounds,omitempty"`
	ToolLimitations    []string `json:"tool_limitations,omitempty"`
	QuantumResistance  bool     `json:"quantum_resistance,omitempty"`
	ExplicitExclusions []string `json:"explicit_exclusions,omitempty"`

	// RFC-0115 C.2 comprehensive power limits (preferred)
	Comprehensive *PowerLimitSet `json:"comprehensive,omitempty"`
}

// RightsObligations is an alias for RFC-0115 Section C.3 comprehensive rights/obligations
// Use RightsObligationSet from rights_obligations.go for new implementations
type RightsObligations struct {
	// Legacy fields for backward compatibility
	ReportingDuties   []string `json:"reporting_duties,omitempty"`
	LiabilityRules    []string `json:"liability_rules,omitempty"`
	CompensationRules []string `json:"compensation_rules,omitempty"`

	// RFC-0115 C.3 comprehensive rights and obligations (preferred)
	Comprehensive *RightsObligationSet `json:"comprehensive,omitempty"`
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

// Representative contains client owner info linking to authorization per RFC-0115 Section A.2
type Representative struct {
	// Legacy field for backward compatibility
	ClientOwner *ClientOwnerInfo `json:"client_owner,omitempty"`

	// RFC-0115 A.2 Representative details
	Identity            string               `json:"identity"`
	LegalRelationship   LegalRelationship    `json:"legal_relationship"`
	RegistrationInfo    *RegistrationInfo    `json:"registration_info,omitempty"`
	AuthorizationChain  []AuthorizationLink  `json:"authorization_chain,omitempty"`
	ContactInformation  *ContactInformation  `json:"contact_information,omitempty"`
	CertificationStatus *CertificationStatus `json:"certification_status,omitempty"`
}

// LegalRelationship defines the representative's relationship to the AI client
type LegalRelationship string

const (
	RelationshipOwner           LegalRelationship = "Owner"
	RelationshipOperator        LegalRelationship = "Operator"
	RelationshipLicensee        LegalRelationship = "Licensee"
	RelationshipContractor      LegalRelationship = "Contractor"
	RelationshipServiceProvider LegalRelationship = "ServiceProvider"
	RelationshipManufacturer    LegalRelationship = "Manufacturer"
	RelationshipDistributor     LegalRelationship = "Distributor"
	RelationshipAgent           LegalRelationship = "Agent"
	RelationshipOther           LegalRelationship = "Other"
)

// RegistrationInfo contains representative registration details
type RegistrationInfo struct {
	RegisteredName        string `json:"registered_name"`
	RegistrationNumber    string `json:"registration_number"`
	RegisteringAuthority  string `json:"registering_authority"`
	RegistrationDate      string `json:"registration_date"` // ISO 8601
	Jurisdiction          string `json:"jurisdiction"`
	BusinessType          string `json:"business_type"`
	TaxIdentifier         string `json:"tax_identifier,omitempty"`
	CommercialRegister    bool   `json:"commercial_register"`
	PowerOfAttorneyOnFile bool   `json:"power_of_attorney_on_file"`
}

// AuthorizationLink represents a link in the authorization chain
type AuthorizationLink struct {
	FromParty     string `json:"from_party"`
	ToParty       string `json:"to_party"`
	GrantedDate   string `json:"granted_date"`          // ISO 8601
	ExpiryDate    string `json:"expiry_date,omitempty"` // ISO 8601
	DocumentRef   string `json:"document_ref,omitempty"`
	Scope         string `json:"scope"`
	Revocable     bool   `json:"revocable"`
	SubDelegation bool   `json:"sub_delegation"` // Can further delegate
}

// ContactInformation contains representative contact details
type ContactInformation struct {
	PrimaryContact    string   `json:"primary_contact"`
	Email             string   `json:"email"`
	Phone             string   `json:"phone,omitempty"`
	Address           *Address `json:"address,omitempty"`
	EmergencyContact  string   `json:"emergency_contact,omitempty"`
	PreferredLanguage string   `json:"preferred_language,omitempty"`
}

// Address represents a physical or registered address
type Address struct {
	Street     string `json:"street"`
	City       string `json:"city"`
	State      string `json:"state,omitempty"`
	PostalCode string `json:"postal_code"`
	Country    string `json:"country"` // ISO 3166-1 alpha-2
}

// CertificationStatus tracks representative certifications
type CertificationStatus struct {
	Certified           bool     `json:"certified"`
	CertifyingBody      string   `json:"certifying_body,omitempty"`
	CertificateNumber   string   `json:"certificate_number,omitempty"`
	IssueDate           string   `json:"issue_date,omitempty"`  // ISO 8601
	ExpiryDate          string   `json:"expiry_date,omitempty"` // ISO 8601
	CertificationTypes  []string `json:"certification_types,omitempty"`
	ComplianceStandards []string `json:"compliance_standards,omitempty"`
}

// ValidateLegalRelationship validates the legal relationship type
func ValidateLegalRelationship(lr LegalRelationship) error {
	validRelationships := []LegalRelationship{
		RelationshipOwner, RelationshipOperator, RelationshipLicensee,
		RelationshipContractor, RelationshipServiceProvider, RelationshipManufacturer,
		RelationshipDistributor, RelationshipAgent, RelationshipOther,
	}
	for _, valid := range validRelationships {
		if lr == valid {
			return nil
		}
	}
	return fmt.Errorf("invalid legal relationship: %s", lr)
}

// Validate performs complete validation of representative information
func (r *Representative) Validate() error {
	if r.Identity == "" {
		return fmt.Errorf("representative identity required")
	}

	if err := ValidateLegalRelationship(r.LegalRelationship); err != nil {
		return fmt.Errorf("legal relationship: %w", err)
	}

	// Validate registration info if present
	if r.RegistrationInfo != nil {
		if r.RegistrationInfo.RegisteredName == "" {
			return fmt.Errorf("registered name required in registration info")
		}
		if r.RegistrationInfo.RegistrationNumber == "" {
			return fmt.Errorf("registration number required in registration info")
		}
		if r.RegistrationInfo.Jurisdiction == "" {
			return fmt.Errorf("jurisdiction required in registration info")
		}
	}

	// Validate authorization chain
	for i, link := range r.AuthorizationChain {
		if link.FromParty == "" || link.ToParty == "" {
			return fmt.Errorf("authorization chain link %d: from_party and to_party required", i)
		}
		if link.Scope == "" {
			return fmt.Errorf("authorization chain link %d: scope required", i)
		}
	}

	// Validate contact information if present
	if r.ContactInformation != nil {
		if r.ContactInformation.PrimaryContact == "" {
			return fmt.Errorf("primary contact required in contact information")
		}
		if r.ContactInformation.Email == "" {
			return fmt.Errorf("email required in contact information")
		}
	}

	return nil
}

// ValidateAuthorizationChain ensures authorization chain integrity
func ValidateAuthorizationChain(chain []AuthorizationLink) error {
	if len(chain) == 0 {
		return nil // Empty chain is valid (direct authorization)
	}

	// Check for chain continuity
	for i := 0; i < len(chain)-1; i++ {
		if chain[i].ToParty != chain[i+1].FromParty {
			return fmt.Errorf("authorization chain broken at link %d: %s -> %s != %s -> %s",
				i, chain[i].FromParty, chain[i].ToParty, chain[i+1].FromParty, chain[i+1].ToParty)
		}
	}

	// Check for unauthorized sub-delegation
	for i := range chain {
		if i > 0 && !chain[i-1].SubDelegation {
			return fmt.Errorf("authorization chain link %d: sub-delegation not allowed by previous link", i)
		}
	}

	return nil
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
	// ClientTypeLLM moved to proper ClientType enum above
	// TransactionLoan, TransactionPurchase etc. moved to action_types.go
	// DecisionFinancial, DecisionStrategic etc. moved to action_types.go
	// ActionResearching, ActionBrainstorming etc. moved to action_types.go
	// GeoTypeNational, GeoTypeRegional etc. moved to GeographicType enum above
	RepresentationSole = "Sole"
	SignatureSingle    = "Single"
)

// Helper functions to create IndustrySector instances for demo compatibility
var (
	DemoSectorInfoComm = taxonomy.IndustrySector{
		Code:        taxonomy.SectorInfoCommunication,
		Description: "Information and Communication",
		Authorized:  true,
	}
	DemoSectorProfessional = taxonomy.IndustrySector{
		Code:        taxonomy.SectorProfessionalScience,
		Description: "Professional, Scientific and Technical Activities",
		Authorized:  true,
	}
	DemoSectorFinancial = taxonomy.IndustrySector{
		Code:        taxonomy.SectorFinanceInsurance,
		Description: "Financial and Insurance Activities",
		Authorized:  true,
	}
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
	// Digest provides a canonical integrity hash over the core PoA fields (excluding metadata & attestation evidence maps which may vary).
	// Format: sha256:<hex>. Populated on Issue(). Recomputed on demand by CanonicalDigest().
	Digest string `json:"digest,omitempty"`
	// Multi-signature fields (optional). If threshold>0 then signatures must meet threshold for verification success.
	SignerKids []string `json:"signer_kids,omitempty"`
	Signatures []string `json:"signatures,omitempty"`
	SigMode    string   `json:"sig_mode,omitempty"`
	Threshold  int      `json:"threshold,omitempty"`

	// RFC-0115 Extended Token Format Fields
	// PoADefinitionID links to the RFC-0115 PoA definition used for this authorization
	PoADefinitionID string `json:"poa_definition_id,omitempty"`
	// SectorScopeRef references authorized industry sectors per RFC-0115 B.2
	SectorScopeRef *taxonomy.SectorScope `json:"sector_scope_ref,omitempty"`
	// AuthorizedActionsRef references authorized action types per RFC-0115 B.4
	AuthorizedActionsRef *taxonomy.AuthorizedActionSet `json:"authorized_actions_ref,omitempty"`
	// PowerLimitRefs references power limitations per RFC-0115 C.2
	PowerLimitRefs *PowerLimitSet `json:"power_limit_refs,omitempty"`
	// ObligationRefs references rights and obligations per RFC-0115 C.3
	ObligationRefs *RightsObligationSet `json:"obligation_refs,omitempty"`
	// ClientTypeInfo contains authorized client type information per RFC-0115 A.3
	ClientTypeInfo *AuthorizedClient `json:"client_type_info,omitempty"`
	// RepresentativeInfo contains representative details per RFC-0115 A.2
	RepresentativeInfo *Representative `json:"representative_info,omitempty"`
	// GeographicScopeRef references authorized geographic regions per RFC-0115 B.3
	GeographicScopeRef []GeographicScope `json:"geographic_scope_ref,omitempty"`
	// ComplianceVersion indicates RFC-0115 compliance version
	ComplianceVersion string `json:"compliance_version,omitempty"` // e.g., "RFC-0115-v1.0"
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

// MemoryService implements Service using in-memory storage
type MemoryService struct {
	proofs      map[string]*ProofOfAuthorization
	revoked     map[string]bool
	keyProvider internalCrypto.KeyProvider
}

// Option configures the MemoryService
type Option func(*MemoryService)

// WithKeyProvider injects a crypto key provider
func WithKeyProvider(kp internalCrypto.KeyProvider) Option {
	return func(s *MemoryService) {
		s.keyProvider = kp
	}
}

// NewMemoryService creates a new memory-based PoA service
func NewMemoryService(opts ...Option) *MemoryService {
	s := &MemoryService{
		proofs:  make(map[string]*ProofOfAuthorization),
		revoked: make(map[string]bool),
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
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

	// Compute canonical digest (after all core fields populated)
	poa.Digest = CanonicalDigest(poa)

	// Optional multi-signature issuance using KeyProvider (demo). Controlled via env:
	// GAUTH_POA_MULTISIG_KIDS=<kid1,kid2,...> GAUTH_POA_MULTISIG_THRESHOLD=<n>
	// If kids not set but provider available, uses active key only when GAUTH_POA_MULTISIG_SIGN=1.
	if os.Getenv("GAUTH_POA_MULTISIG_SIGN") == "1" {
		kidsRaw := os.Getenv("GAUTH_POA_MULTISIG_KIDS")
		var kids []string
		if kidsRaw != "" {
			for _, part := range strings.Split(kidsRaw, ",") {
				p := strings.TrimSpace(part)
				if p != "" {
					kids = append(kids, p)
				}
			}
		}
		// Fallback to active key if no explicit list
		if len(kids) == 0 && s.keyProvider != nil {
			if signer, err := s.keyProvider.ActiveSigner(); err == nil && signer != nil {
				kids = []string{signer.KeyID()}
			}
		}

		th := 0
		if rawTh := os.Getenv("GAUTH_POA_MULTISIG_THRESHOLD"); rawTh != "" {
			if v, err := strconv.Atoi(rawTh); err == nil && v >= 0 {
				th = v
			}
		}
		if th == 0 {
			th = len(kids)
		}
		poa.Threshold = th
		poa.SignerKids = append([]string(nil), kids...)
		poa.SigMode = "eddsa"
		msg := buildPoASigningPayload(poa)
		for _, kid := range kids {
			if s.keyProvider == nil {
				continue
			}

			// In a real implementation, we would need a way to get a Signer for a specific KeyID.
			// The current KeyProvider interface only exposes ActiveSigner().
			// For this demo refactor, we will only sign if the requested kid matches the active signer.
			// To fully support multi-sig with historical keys, KeyProvider would need Signer(keyID).
			// For now, we check against active signer.

			signer, err := s.keyProvider.ActiveSigner()
			if err != nil || signer == nil {
				continue
			}

			if signer.KeyID() == kid {
				sig, err := signer.Sign(msg)
				if err == nil {
					poa.Signatures = append(poa.Signatures, base64.RawStdEncoding.EncodeToString(sig))
				}
			}
		}
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

	// RFC-0115 Extended Validation
	if err := ValidateRFC0115Token(poa); err != nil {
		return errors.New(errors.ErrCodeValidation, fmt.Sprintf("RFC-0115 validation failed: %v", err))
	}

	return nil
}

// ValidateRFC0115Token performs RFC-0115 compliance validation for extended token format
func ValidateRFC0115Token(poa *ProofOfAuthorization) error {
	if poa == nil {
		return fmt.Errorf("PoA token is nil")
	}

	if err := validateComplianceVersion(poa.ComplianceVersion); err != nil {
		return err
	}

	if err := validateSectorScope(poa.SectorScopeRef); err != nil {
		return err
	}

	if err := validateTokenReferences(poa); err != nil {
		return err
	}

	if err := validatePoAParties(poa); err != nil {
		return err
	}

	return validatePoAGeographicScope(poa.GeographicScopeRef)
}

func validateComplianceVersion(version string) error {
	if version != "" && !strings.HasPrefix(version, "RFC-0115") {
		return fmt.Errorf("invalid compliance version: %s", version)
	}
	return nil
}

func validateSectorScope(scope *taxonomy.SectorScope) error {
	if scope != nil {
		if len(scope.Sectors) == 0 && !scope.AllSectors {
			return fmt.Errorf("sector scope must specify at least one sector or allow all sectors")
		}
		for i, sector := range scope.Sectors {
			if err := taxonomy.ValidateSectorCode(sector.Code); err != nil {
				return fmt.Errorf("sector scope sector %d: %w", i, err)
			}
		}
	}
	return nil
}

func validateTokenReferences(poa *ProofOfAuthorization) error {
	if poa.AuthorizedActionsRef != nil {
		if err := poa.AuthorizedActionsRef.Validate(); err != nil {
			return fmt.Errorf("authorized actions validation: %w", err)
		}
	}

	if poa.PowerLimitRefs != nil {
		if err := poa.PowerLimitRefs.Validate(); err != nil {
			return fmt.Errorf("power limits validation: %w", err)
		}
	}

	if poa.ObligationRefs != nil {
		if err := poa.ObligationRefs.Validate(); err != nil {
			return fmt.Errorf("obligations validation: %w", err)
		}
	}
	return nil
}

func validatePoAParties(poa *ProofOfAuthorization) error {
	if poa.ClientTypeInfo != nil {
		if err := poa.ClientTypeInfo.Validate(); err != nil {
			return fmt.Errorf("client type info validation: %w", err)
		}

		if poa.AuthorizedActionsRef != nil {
			if err := ActionCompatibilityCheck(poa.AuthorizedActionsRef, poa.ClientTypeInfo); err != nil {
				return fmt.Errorf("action/client compatibility: %w", err)
			}
		}
	}

	if poa.RepresentativeInfo != nil {
		if err := poa.RepresentativeInfo.Validate(); err != nil {
			return fmt.Errorf("representative info validation: %w", err)
		}

		if err := ValidateAuthorizationChain(poa.RepresentativeInfo.AuthorizationChain); err != nil {
			return fmt.Errorf("authorization chain validation: %w", err)
		}
	}
	return nil
}

func validatePoAGeographicScope(scopes []GeographicScope) error {
	if len(scopes) > 0 {
		for i, geo := range scopes {
			if geo.Type == "" || geo.Identifier == "" {
				return fmt.Errorf("geographic scope %d: type and identifier required", i)
			}
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

// buildPoASigningPayload constructs the domain-separated payload for PoA signatures.
func buildPoASigningPayload(p *ProofOfAuthorization) []byte {
	// Reuse canonical digest source (without signatures) to avoid circular changes.
	// Canonical subset identical to CanonicalDigest's internal struct.
	type canon struct {
		ID         string    `json:"id"`
		Subject    string    `json:"subject"`
		Resource   string    `json:"resource"`
		Action     string    `json:"action"`
		Issuer     string    `json:"issuer"`
		IssuedAt   time.Time `json:"issued_at"`
		ExpiresAt  time.Time `json:"expires_at"`
		Scope      []string  `json:"scope"`
		Delegation *struct {
			DelegatedBy string    `json:"delegated_by"`
			DelegatedTo string    `json:"delegated_to"`
			DelegatedAt time.Time `json:"delegated_at"`
			ExpiresAt   time.Time `json:"expires_at"`
			Scope       []string  `json:"scope"`
			Revocable   bool      `json:"revocable"`
		} `json:"delegation,omitempty"`
		Attestation *struct {
			AttestedBy    string    `json:"attested_by"`
			AttestedAt    time.Time `json:"attested_at"`
			Confidence    float64   `json:"confidence"`
			ValidityScore float64   `json:"validity_score"`
		} `json:"attestation,omitempty"`
	}
	c := canon{ID: p.ID, Subject: p.Subject, Resource: p.Resource, Action: p.Action, Issuer: p.Issuer, IssuedAt: p.IssuedAt, ExpiresAt: p.ExpiresAt, Scope: append([]string(nil), p.Scope...)}
	if p.Delegation != nil {
		c.Delegation = &struct {
			DelegatedBy string    `json:"delegated_by"`
			DelegatedTo string    `json:"delegated_to"`
			DelegatedAt time.Time `json:"delegated_at"`
			ExpiresAt   time.Time `json:"expires_at"`
			Scope       []string  `json:"scope"`
			Revocable   bool      `json:"revocable"`
		}{DelegatedBy: p.Delegation.DelegatedBy, DelegatedTo: p.Delegation.DelegatedTo, DelegatedAt: p.Delegation.DelegatedAt, ExpiresAt: p.Delegation.ExpiresAt, Scope: append([]string(nil), p.Delegation.Scope...), Revocable: p.Delegation.Revocable}
	}
	if p.Attestation != nil {
		c.Attestation = &struct {
			AttestedBy    string    `json:"attested_by"`
			AttestedAt    time.Time `json:"attested_at"`
			Confidence    float64   `json:"confidence"`
			ValidityScore float64   `json:"validity_score"`
		}{AttestedBy: p.Attestation.AttestedBy, AttestedAt: p.Attestation.AttestedAt, Confidence: p.Attestation.Confidence, ValidityScore: p.Attestation.ValidityScore}
	}
	raw, _ := json.Marshal(c)
	return append([]byte("GAUTH_POA:"), raw...)
}

// VerifyMultiSig validates all signatures present and evaluates threshold satisfaction.
// Returns (validSignatures, satisfied, requiredThreshold).
func VerifyMultiSig(p *ProofOfAuthorization, kp internalCrypto.KeyProvider) (int, bool, int) {
	if p == nil || len(p.Signatures) == 0 || len(p.SignerKids) == 0 || p.Threshold <= 0 {
		return 0, false, p.Threshold
	}
	msg := buildPoASigningPayload(p)
	valid := 0
	for i, sigB64 := range p.Signatures {
		if i >= len(p.SignerKids) {
			break
		}
		kid := p.SignerKids[i]

		if kp == nil {
			continue
		}

		sigBytes, err := base64.RawStdEncoding.DecodeString(sigB64)
		if err != nil {
			continue
		}

		if err := kp.VerifyWith(msg, sigBytes, kid); err == nil {
			valid++
		}
	}
	return valid, valid >= p.Threshold, p.Threshold
}

// CanonicalDigest computes a deterministic SHA256 hash over stable PoA fields.
// Excludes Metadata (arbitrary map), Attestation.Evidence (may be large/dynamic), and Delegation.Constraints
// to ensure digest stability across benign descriptive changes.
// Canonical serialization order is fixed by explicit struct used below.
func CanonicalDigest(p *ProofOfAuthorization) string {
	if p == nil {
		return ""
	}
	// Canonical view struct
	type canon struct {
		ID        string    `json:"id"`
		Subject   string    `json:"subject"`
		Resource  string    `json:"resource"`
		Action    string    `json:"action"`
		Issuer    string    `json:"issuer"`
		IssuedAt  time.Time `json:"issued_at"`
		ExpiresAt time.Time `json:"expires_at"`
		Scope     []string  `json:"scope"`
		// Delegation minimal canonical subset (identity & temporal scope only)
		Delegation *struct {
			DelegatedBy string    `json:"delegated_by"`
			DelegatedTo string    `json:"delegated_to"`
			DelegatedAt time.Time `json:"delegated_at"`
			ExpiresAt   time.Time `json:"expires_at"`
			Scope       []string  `json:"scope"`
			Revocable   bool      `json:"revocable"`
		} `json:"delegation,omitempty"`
		// Attestation canonical subset (exclude evidence map)
		Attestation *struct {
			AttestedBy    string    `json:"attested_by"`
			AttestedAt    time.Time `json:"attested_at"`
			Confidence    float64   `json:"confidence"`
			ValidityScore float64   `json:"validity_score"`
		} `json:"attestation,omitempty"`
	}
	c := canon{ID: p.ID, Subject: p.Subject, Resource: p.Resource, Action: p.Action, Issuer: p.Issuer, IssuedAt: p.IssuedAt, ExpiresAt: p.ExpiresAt, Scope: append([]string(nil), p.Scope...)}
	if p.Delegation != nil {
		c.Delegation = &struct {
			DelegatedBy string    `json:"delegated_by"`
			DelegatedTo string    `json:"delegated_to"`
			DelegatedAt time.Time `json:"delegated_at"`
			ExpiresAt   time.Time `json:"expires_at"`
			Scope       []string  `json:"scope"`
			Revocable   bool      `json:"revocable"`
		}{DelegatedBy: p.Delegation.DelegatedBy, DelegatedTo: p.Delegation.DelegatedTo, DelegatedAt: p.Delegation.DelegatedAt, ExpiresAt: p.Delegation.ExpiresAt, Scope: append([]string(nil), p.Delegation.Scope...), Revocable: p.Delegation.Revocable}
	}
	if p.Attestation != nil {
		c.Attestation = &struct {
			AttestedBy    string    `json:"attested_by"`
			AttestedAt    time.Time `json:"attested_at"`
			Confidence    float64   `json:"confidence"`
			ValidityScore float64   `json:"validity_score"`
		}{AttestedBy: p.Attestation.AttestedBy, AttestedAt: p.Attestation.AttestedAt, Confidence: p.Attestation.Confidence, ValidityScore: p.Attestation.ValidityScore}
	}
	raw, _ := json.Marshal(c)
	sum := sha256.Sum256(raw)
	return fmt.Sprintf("sha256:%x", sum[:])
}

// VerifyDigest recomputes the canonical digest and compares with embedded Digest field.
func VerifyDigest(p *ProofOfAuthorization) bool {
	if p == nil || p.Digest == "" {
		return false
	}
	return p.Digest == CanonicalDigest(p)
}

// Package rfc implements the combined GAuth 1.0 Authorization Framework and Power-of-Attorney
// Credential Definition as specified in GiFo-RFC-0111 and GiFo-RFC-0115 by Dr. Götz G. Wehberg
//
// Copyright (c) 2025 Gimel Foundation gGmbH i.G.
// Licensed under Apache 2.0
//
// Official Gimel Foundation Implementation
// Gimel Foundation gGmbH i.G., www.GimelFoundation.com
// Operated by Gimel Technologies GmbH
// MD: Bjørn Baunbæk, Dr. Götz G. Wehberg – Chairman of the Board: Daniel Hartert
// Hardtweg 31, D-53639 Königswinter, Siegburg HRB 18660, www.GimelID.com
//
// Combined Implementation:
// - GiFo-RFC-0111: The GAuth 1.0 Authorization Framework (ISBN: 978-3-00-084039-5)
//   Digital Supply Institute, Standards Track, Obsoletes: 1. August 2025
// - GiFo-RFC-0115: Power-of-Attorney Credential Definition (PoA-Definition)
//   Digital Supply Institute, Standards Track, Obsoletes: 15. September 2025
//
// This package provides comprehensive AI authorization capabilities integrating
// OAuth 2.0, OpenID Connect, and MCP protocols with enhanced Power-of-Attorney
// governance for autonomous AI systems including digital agents, agentic AI,
// and humanoid robots with mandatory exclusions enforcement.

package rfc

import (
	"fmt"
	"time"
)

// =============================================================================
// RFC-0111: GAuth 1.0 Authorization Framework Core Types
// =============================================================================

// RFC0111Config represents the complete GAuth 1.0 configuration combining
// OAuth 2.0, OpenID Connect, and MCP protocols with AI governance
type RFC0111Config struct {
	// RFC-0111 Core Components
	PPArchitecture  RFC0111PPArchitecture  `json:"pp_architecture"`
	Exclusions      RFC0111Exclusions      `json:"exclusions"`
	ExtendedTokens  RFC0111ExtendedTokens  `json:"extended_tokens"`
	GAuthRoles      RFC0111GAuthRoles      `json:"gauth_roles"`
	
	// Configuration Metadata
	Version         string            `json:"version"`
	Status          string            `json:"status"`
	CreatedAt       time.Time         `json:"created_at"`
	UpdatedAt       time.Time         `json:"updated_at"`
}

// RFC0111PPArchitecture represents the Power*Point architecture as defined in RFC-0111
type RFC0111PPArchitecture struct {
	PEP RFC0111PowerEnforcementPoint  `json:"pep"` // Power Enforcement Point
	PDP RFC0111PowerDecisionPoint     `json:"pdp"` // Power Decision Point  
	PIP RFC0111PowerInformationPoint  `json:"pip"` // Power Information Point
	PAP RFC0111PowerAdministrationPoint `json:"pap"` // Power Administration Point
	PVP RFC0111PowerVerificationPoint `json:"pvp"` // Power Verification Point
}

// RFC0111PowerEnforcementPoint represents the PEP in GAuth P*P architecture
type RFC0111PowerEnforcementPoint struct {
	SupplySide  RFC0111PEPSide `json:"supply_side"`  // Client-side enforcement
	DemandSide  RFC0111PEPSide `json:"demand_side"`  // Resource server-side enforcement
}

type RFC0111PEPSide struct {
	Entity      string   `json:"entity"`
	Enforcement []string `json:"enforcement"`
	Status      string   `json:"status"`
}

// RFC0111PowerDecisionPoint represents authorization decision-making instance
type RFC0111PowerDecisionPoint struct {
	PrimaryPDP    string `json:"primary_pdp"`    // Usually client owner
	SecondaryPDP  string `json:"secondary_pdp"`  // Resource owner if resource server is AI
	DecisionRules []string `json:"decision_rules"`
}

// RFC0111PowerInformationPoint represents data providers for approval decisions
type RFC0111PowerInformationPoint struct {
	AuthorizationServer string   `json:"authorization_server"`
	DataSources        []string `json:"data_sources"`
	InfoTypes          []string `json:"info_types"`
}

// RFC0111PowerAdministrationPoint represents administrative level for authorization policies
type RFC0111PowerAdministrationPoint struct {
	ClientOwnerAuthorizer   string   `json:"client_owner_authorizer"`
	ResourceOwnerAuthorizer string   `json:"resource_owner_authorizer"`
	PolicyManagement       []string `json:"policy_management"`
}

// RFC0111PowerVerificationPoint represents identity verification in GAuth processing
type RFC0111PowerVerificationPoint struct {
	TrustServiceProvider string   `json:"trust_service_provider"`
	VerificationMethods  []string `json:"verification_methods"`
	IdentityTypes        []string `json:"identity_types"`
}

// RFC0111Exclusions represents mandatory exclusions as per RFC-0111 Section 2
type RFC0111Exclusions struct {
	Web3Blockchain       RFC0111ExclusionRule `json:"web3_blockchain"`
	AIOperators          RFC0111ExclusionRule `json:"ai_operators"`
	DNABasedIdentities   RFC0111ExclusionRule `json:"dna_based_identities"`
	DecentralizedAuth    RFC0111ExclusionRule `json:"decentralized_auth"`
	EnforcementLevel     string               `json:"enforcement_level"`
}

type RFC0111ExclusionRule struct {
	Prohibited      bool     `json:"prohibited"`
	Description     string   `json:"description"`
	Alternatives    []string `json:"alternatives,omitempty"`
	LicenseRequired bool     `json:"license_required"`
}

// RFC0111ExtendedTokens represents GAuth extended tokens beyond OAuth access tokens
type RFC0111ExtendedTokens struct {
	TokenType        string                    `json:"token_type"`
	Scope            []string                  `json:"scope"`
	Duration         time.Duration             `json:"duration"`
	Authorization    RFC0111TokenAuthorization `json:"authorization"`
	Compliance       RFC0111TokenCompliance    `json:"compliance"`
}

type RFC0111TokenAuthorization struct {
	Transactions   []string `json:"transactions"`
	Decisions      []string `json:"decisions"`
	Actions        []string `json:"actions"`
	ResourceRights []string `json:"resource_rights"`
}

type RFC0111TokenCompliance struct {
	ComplianceTracking bool     `json:"compliance_tracking"`
	AuditTrail        []string `json:"audit_trail"`
	RevocationStatus  string   `json:"revocation_status"`
}

// RFC0111GAuthRoles represents the enhanced role definitions from RFC-0111
type RFC0111GAuthRoles struct {
	ResourceOwner       RFC0111ResourceOwner    `json:"resource_owner"`
	ResourceServer      RFC0111ResourceServer   `json:"resource_server"`
	Client              RFC0111Client           `json:"client"`
	AuthorizationServer RFC0111AuthServer       `json:"authorization_server"`
	ClientOwner         RFC0111ClientOwner      `json:"client_owner"`
	OwnerAuthorizer     RFC0111OwnerAuthorizer  `json:"owner_authorizer"`
}

// RFC0111ResourceOwner extends OAuth resource owner for AI governance
type RFC0111ResourceOwner struct {
	Identity            string   `json:"identity"`
	LegalCapacity       bool     `json:"legal_capacity"`
	TransactionAuthority []string `json:"transaction_authority"`
	DecisionAcceptance  []string `json:"decision_acceptance"`
	ActionImpact        []string `json:"action_impact"`
}

// RFC0111ResourceServer extends OAuth resource server for AI interactions
type RFC0111ResourceServer struct {
	Identity           string   `json:"identity"`
	AssetTypes         []string `json:"asset_types"`
	ProtectedResources []string `json:"protected_resources"`
	TokenValidation    string   `json:"token_validation"`
	AICapable          bool     `json:"ai_capable"`
}

// RFC0111Client represents AI clients (digital agents, agentic AI, humanoid robots)
type RFC0111Client struct {
	Type              RFC0111ClientType `json:"type"`
	Identity          string            `json:"identity"`
	AICapabilities    []string          `json:"ai_capabilities"`
	AutonomyLevel     string            `json:"autonomy_level"`
	RequestTypes      []string          `json:"request_types"`
	ComplianceMode    string            `json:"compliance_mode"`
}

type RFC0111ClientType string

const (
	RFC0111ClientTypeDigitalAgent  RFC0111ClientType = "digital_agent"
	RFC0111ClientTypeAgenticAI     RFC0111ClientType = "agentic_ai"      // team of agents
	RFC0111ClientTypeHumanoidRobot RFC0111ClientType = "humanoid_robot"
	RFC0111ClientTypeLLM           RFC0111ClientType = "llm"
	RFC0111ClientTypeOther         RFC0111ClientType = "other"
)

// RFC0111AuthServer represents the enhanced authorization server
type RFC0111AuthServer struct {
	Identity              string `json:"identity"`
	ExtendedTokenIssuing  bool   `json:"extended_token_issuing"`
	ComplianceTracking    bool   `json:"compliance_tracking"`
	PPArchitectureSupport bool   `json:"pp_architecture_support"`
	ExclusionsEnforced    bool   `json:"exclusions_enforced"`
}

// RFC0111ClientOwner represents the owner of the AI system
type RFC0111ClientOwner struct {
	Identity          string   `json:"identity"`
	AuthorizationLevel string  `json:"authorization_level"`
	AISystemOwnership []string `json:"ai_system_ownership"`
	DelegatedPowers   []string `json:"delegated_powers"`
}

// RFC0111OwnerAuthorizer represents the authorizer of client/resource owners
type RFC0111OwnerAuthorizer struct {
	Identity           string   `json:"identity"`
	StatutoryAuthority bool     `json:"statutory_authority"`
	AuthorizationScope []string `json:"authorization_scope"`
	VerificationMethod string   `json:"verification_method"`
}

// =============================================================================
// RFC-0115: Power-of-Attorney Credential Definition Types
// =============================================================================

// RFC0115PoADefinition represents the complete Power-of-Attorney Credential Definition
// as specified in GiFo-RFC-0115 Section 3
type RFC0115PoADefinition struct {
	// A. Parties
	Parties RFC0115Parties `json:"parties"`
	
	// B. Type and Scope of Authorization
	Authorization RFC0115AuthorizationScope `json:"authorization"`
	
	// C. Requirements
	Requirements RFC0115Requirements `json:"requirements"`
	
	// RFC-0111 Integration Context
	GAuthContext RFC0115GAuthIntegration `json:"gauth_context"`
}

// RFC0115GAuthIntegration represents RFC-0111 integration context
type RFC0115GAuthIntegration struct {
	PPArchitectureRole  string   `json:"pp_architecture_role"`
	ExclusionsCompliant bool     `json:"exclusions_compliant"`
	ExtendedTokenScope  []string `json:"extended_token_scope"`
	AIGovernanceLevel   string   `json:"ai_governance_level"`
}

// RFC0115Parties represents all parties involved in the PoA as per RFC-0115 Section 3.A
type RFC0115Parties struct {
	Principal         RFC0115Principal         `json:"principal"`
	Representative    *RFC0115Representative   `json:"representative,omitempty"` // Only if principal is organization
	AuthorizedClient  RFC0115AuthorizedClient  `json:"authorized_client"`
}

// RFC0115Principal represents the principal party (Individual or Organization)
type RFC0115Principal struct {
	Type         RFC0115PrincipalType `json:"type"`
	Identity     string               `json:"identity"`
	Individual   *RFC0115Individual   `json:"individual,omitempty"`
	Organization *RFC0115Organization `json:"organization,omitempty"`
}

type RFC0115PrincipalType string

const (
	RFC0115PrincipalTypeIndividual   RFC0115PrincipalType = "individual"
	RFC0115PrincipalTypeOrganization RFC0115PrincipalType = "organization"
)

// RFC0115Individual represents a natural person principal
type RFC0115Individual struct {
	Name        string `json:"name"`
	Citizenship string `json:"citizenship,omitempty"`
}

// RFC0115Organization represents an organizational principal as per RFC-0115
type RFC0115Organization struct {
	Type                RFC0115OrgType `json:"type"`
	Name                string         `json:"name"`
	RegisterEntry       string         `json:"register_entry,omitempty"`
	ManagingDirector    string         `json:"managing_director,omitempty"`
	RegisteredAuthority bool           `json:"registered_authority"`
}

type RFC0115OrgType string

const (
	RFC0115OrgTypeCommercial  RFC0115OrgType = "commercial_enterprise"    // AG, Ltd., partnership, etc.
	RFC0115OrgTypePublic      RFC0115OrgType = "public_authority"         // federal, state, municipal, etc.
	RFC0115OrgTypeNonProfit   RFC0115OrgType = "non_profit_organization"  // foundation, non-profit association, gGmbH, etc.
	RFC0115OrgTypeAssociation RFC0115OrgType = "other_association"        // non-profit or non-charitable
	RFC0115OrgTypeOther       RFC0115OrgType = "other"                   // church, cooperative, community of interest, etc.
)

// RFC0115Representative represents the representative/authorizer when principal is organization
type RFC0115Representative struct {
	ClientOwner          *RFC0115ClientOwnerInfo       `json:"client_owner,omitempty"`
	OwnerAuthorizer      *RFC0115OwnerAuthorizerInfo   `json:"owner_authorizer,omitempty"`
	OtherRepresentatives []RFC0115OtherRepresentative  `json:"other_representatives,omitempty"`
	Other                string                        `json:"other,omitempty"`
}

type RFC0115ClientOwnerInfo struct {
	Name                      string `json:"name"`
	RegisteredPowerOfAttorney bool   `json:"registered_power_of_attorney"`
	CommercialRegisterEntry   bool   `json:"commercial_register_entry"`
	Other                     string `json:"other,omitempty"`
}

type RFC0115OwnerAuthorizerInfo struct {
	Name                      string `json:"name"`
	RegisteredPowerOfAttorney bool   `json:"registered_power_of_attorney"`
	CommercialRegisterEntry   bool   `json:"commercial_register_entry"`
	Other                     string `json:"other,omitempty"`
}

type RFC0115OtherRepresentative struct {
	Name                      string `json:"name"`
	RegisteredPowerOfAttorney bool   `json:"registered_power_of_attorney"`
	CommercialRegisterEntry   bool   `json:"commercial_register_entry"`
	Other                     string `json:"other,omitempty"`
}

// RFC0115AuthorizedClient represents the AI client being authorized as per RFC-0115
type RFC0115AuthorizedClient struct {
	Type              RFC0115ClientType `json:"type"`
	Identity          string            `json:"identity"`
	Version           string            `json:"version"`
	OperationalStatus string            `json:"operational_status"` // e.g., active, revoked
	Other             string            `json:"other,omitempty"`
}

type RFC0115ClientType string

const (
	RFC0115ClientTypeLLM           RFC0115ClientType = "llm"
	RFC0115ClientTypeDigitalAgent  RFC0115ClientType = "digital_agent"
	RFC0115ClientTypeAgenticAI     RFC0115ClientType = "agentic_ai"      // team of agents
	RFC0115ClientTypeHumanoidRobot RFC0115ClientType = "humanoid_robot"
	RFC0115ClientTypeOther         RFC0115ClientType = "other"
)

// RFC0115AuthorizationScope represents Type and Scope of Authorization as per RFC-0115 Section 3.B
type RFC0115AuthorizationScope struct {
	AuthorizationType     RFC0115AuthorizationType     `json:"authorization_type"`
	ApplicableSectors     []RFC0115IndustrySector      `json:"applicable_sectors"`
	ApplicableRegions     []RFC0115GeographicScope     `json:"applicable_regions"`
	AuthorizedActions     RFC0115AuthorizedActions     `json:"authorized_actions"`
}

// RFC0115AuthorizationType represents the type of authorization as per RFC-0115
type RFC0115AuthorizationType struct {
	RepresentationType RFC0115RepresentationType `json:"representation_type"` // sole or joint
	Restrictions       []string                  `json:"restrictions,omitempty"`
	SubProxyAuthority  bool                      `json:"sub_proxy_authority"`
	SignatureType      RFC0115SignatureType     `json:"signature_type"`
	Other              string                    `json:"other,omitempty"`
}

type RFC0115RepresentationType string

const (
	RFC0115RepresentationSole  RFC0115RepresentationType = "sole"
	RFC0115RepresentationJoint RFC0115RepresentationType = "joint"
)

type RFC0115SignatureType string

const (
	RFC0115SignatureSingle     RFC0115SignatureType = "single"
	RFC0115SignatureJoint      RFC0115SignatureType = "joint"
	RFC0115SignatureCollective RFC0115SignatureType = "collective"
)

// RFC0115IndustrySector represents applicable sectors using global industry codes (ISIC/NACE)
type RFC0115IndustrySector string

const (
	RFC0115SectorAgriculture           RFC0115IndustrySector = "agriculture_forestry_fishing"
	RFC0115SectorMining                RFC0115IndustrySector = "mining_quarrying"
	RFC0115SectorManufacturing         RFC0115IndustrySector = "manufacturing"
	RFC0115SectorEnergySupply          RFC0115IndustrySector = "energy_supply"
	RFC0115SectorWaterSupply           RFC0115IndustrySector = "water_supply"
	RFC0115SectorWasteManagement       RFC0115IndustrySector = "waste_management"
	RFC0115SectorConstruction          RFC0115IndustrySector = "construction"
	RFC0115SectorTrade                 RFC0115IndustrySector = "trade"
	RFC0115SectorVehicleMaintenance    RFC0115IndustrySector = "vehicle_maintenance_repair"
	RFC0115SectorTransportStorage      RFC0115IndustrySector = "transport_storage"
	RFC0115SectorHospitality           RFC0115IndustrySector = "hospitality"
	RFC0115SectorInformationComm       RFC0115IndustrySector = "information_communication"
	RFC0115SectorFinancialInsurance    RFC0115IndustrySector = "financial_insurance_services"
	RFC0115SectorRealEstate            RFC0115IndustrySector = "real_estate"
	RFC0115SectorProfessional          RFC0115IndustrySector = "professional_scientific_technical"
	RFC0115SectorBusinessServices      RFC0115IndustrySector = "other_business_services"
	RFC0115SectorPublicAdmin           RFC0115IndustrySector = "public_administration_defence"
	RFC0115SectorEducation             RFC0115IndustrySector = "education"
	RFC0115SectorHealthSocial          RFC0115IndustrySector = "health_social_work"
	RFC0115SectorArtsEntertainment     RFC0115IndustrySector = "arts_entertainment_recreation"
	RFC0115SectorOtherServices         RFC0115IndustrySector = "other_services_sectors"
)

// RFC0115GeographicScope represents applicable regions as per RFC-0115
type RFC0115GeographicScope struct {
	Type       RFC0115GeoType `json:"type"`
	Identifier string         `json:"identifier"`
	Name       string         `json:"name"`
}

type RFC0115GeoType string

const (
	RFC0115GeoTypeGlobal       RFC0115GeoType = "global"
	RFC0115GeoTypeNational     RFC0115GeoType = "national"
	RFC0115GeoTypeRegional     RFC0115GeoType = "regional"
	RFC0115GeoTypeSubnational  RFC0115GeoType = "subnational"
)

// RFC0115AuthorizedActions represents specific actions the AI is authorized to perform
type RFC0115AuthorizedActions struct {
	DecisionMaking    []RFC0115DecisionType    `json:"decision_making"`
	TransactionTypes  []RFC0115TransactionType `json:"transaction_types"`
	CommunicationAuth []RFC0115CommType        `json:"communication_auth"`
	DocumentHandling  []RFC0115DocType         `json:"document_handling"`
	Other             []string                 `json:"other,omitempty"`
}

type RFC0115DecisionType string
type RFC0115TransactionType string
type RFC0115CommType string
type RFC0115DocType string

// RFC0115Requirements represents all requirements as per RFC-0115 Section 3.C
type RFC0115Requirements struct {
	ValidityPeriod       RFC0115ValidityPeriod       `json:"validity_period"`
	FormalRequirements   RFC0115FormalRequirements   `json:"formal_requirements"`
	PowerLimits          RFC0115PowerLimits          `json:"power_limits"`
	RightsObligations    RFC0115RightsObligations    `json:"rights_obligations"`
	SpecialConditions    RFC0115SpecialConditions    `json:"special_conditions"`
	DeathIncapacityRules RFC0115DeathIncapacityRules `json:"death_incapacity_rules"`
	SecurityCompliance   RFC0115SecurityCompliance   `json:"security_compliance"`
	JurisdictionLaw      RFC0115JurisdictionLaw      `json:"jurisdiction_law"`
	ConflictResolution   RFC0115ConflictResolution   `json:"conflict_resolution"`
}

// RFC0115ValidityPeriod represents the validity period of the PoA
type RFC0115ValidityPeriod struct {
	StartDate   *time.Time `json:"start_date,omitempty"`
	EndDate     *time.Time `json:"end_date,omitempty"`
	Duration    string     `json:"duration,omitempty"`
	Indefinite  bool       `json:"indefinite"`
	AutoRenewal bool       `json:"auto_renewal"`
	Other       string     `json:"other,omitempty"`
}

// RFC0115FormalRequirements represents formal requirements for the PoA
type RFC0115FormalRequirements struct {
	WrittenForm           bool     `json:"written_form"`
	NotarialCertification bool     `json:"notarial_certification"`
	WitnessRequirements   []string `json:"witness_requirements,omitempty"`
	OfficialRegistration  bool     `json:"official_registration"`
	Other                 string   `json:"other,omitempty"`
}

// RFC0115PowerLimits represents limits of powers as per RFC-0115
type RFC0115PowerLimits struct {
	PowerLevels         []string `json:"power_levels,omitempty"`
	InteractionBounds   []string `json:"interaction_boundaries,omitempty"`
	ToolLimitations     []string `json:"tool_limitations,omitempty"`
	OutcomeLimitations  []string `json:"outcome_limitations,omitempty"`
	ModelLimits         []string `json:"model_limits,omitempty"`
	BehaviouralLimits   []string `json:"behavioural_limits,omitempty"`
	QuantumResistance   bool     `json:"quantum_resistance"`
	ExplicitExclusions  []string `json:"explicit_exclusions,omitempty"`
	Other               string   `json:"other,omitempty"`
}

// RFC0115RightsObligations represents specific rights and obligations
type RFC0115RightsObligations struct {
	ReportingDuties   []string `json:"reporting_duties,omitempty"`
	LiabilityRules    []string `json:"liability_rules,omitempty"`
	CompensationRules []string `json:"compensation_rules,omitempty"`
	Other             string   `json:"other,omitempty"`
}

// RFC0115SpecialConditions represents special conditions
type RFC0115SpecialConditions struct {
	ConditionalEffectiveness []string `json:"conditional_effectiveness,omitempty"`
	ImmediateNotification   []string `json:"immediate_notification,omitempty"`
	Other                   string   `json:"other,omitempty"`
}

// RFC0115DeathIncapacityRules represents rules for death or incapacity
type RFC0115DeathIncapacityRules struct {
	ContinuationOnDeath    bool   `json:"continuation_on_death"`
	IncapacityInstructions string `json:"incapacity_instructions,omitempty"`
	Other                  string `json:"other,omitempty"`
}

// RFC0115SecurityCompliance represents security and compliance requirements
type RFC0115SecurityCompliance struct {
	CommunicationProtocols []string `json:"communication_protocols,omitempty"`
	SecurityProperties     []string `json:"security_properties,omitempty"`
	ComplianceInfo         []string `json:"compliance_info,omitempty"`
	UpdateMechanism        string   `json:"update_mechanism,omitempty"`
	Other                  string   `json:"other,omitempty"`
}

// RFC0115JurisdictionLaw represents place of jurisdiction and applicable law
type RFC0115JurisdictionLaw struct {
	Language            string   `json:"language"`
	GoverningLaw        string   `json:"governing_law"`
	PlaceOfJurisdiction string   `json:"place_of_jurisdiction"`
	AttachedDocuments   []string `json:"attached_documents,omitempty"`
	Other               string   `json:"other,omitempty"`
}

// RFC0115ConflictResolution represents conflict resolution arrangements
type RFC0115ConflictResolution struct {
	ArbitrationJurisdiction string `json:"arbitration_jurisdiction,omitempty"`
	Other                   string `json:"other,omitempty"`
}

// =============================================================================
// Combined RFC-0111 & RFC-0115 Integration Types
// =============================================================================

// CombinedRFCConfig represents the unified configuration for both RFC-0111 and RFC-0115
type CombinedRFCConfig struct {
	// RFC-0111 Configuration
	RFC0111 RFC0111Config `json:"rfc_0111"`
	
	// RFC-0115 Configuration
	RFC0115 *RFC0115PoADefinition `json:"rfc_0115,omitempty"`
	
	// Integration Metadata
	IntegrationLevel string            `json:"integration_level"`
	CombinedVersion  string            `json:"combined_version"`
	Compatibility    map[string]string `json:"compatibility"`
}

// =============================================================================
// Combined RFC-0111 & RFC-0115 Validation Functions
// =============================================================================

// ValidateCombinedRFCConfig validates the complete combined RFC configuration
func ValidateCombinedRFCConfig(config CombinedRFCConfig) error {
	// Validate RFC-0111 configuration
	if err := ValidateRFC0111Config(config.RFC0111); err != nil {
		return fmt.Errorf("RFC-0111 validation failed: %w", err)
	}
	
	// Validate RFC-0115 configuration if present
	if config.RFC0115 != nil {
		if err := ValidateRFC0115PoADefinition(*config.RFC0115); err != nil {
			return fmt.Errorf("RFC-0115 validation failed: %w", err)
		}
		
		// Validate integration compatibility
		if err := ValidateRFCIntegration(config.RFC0111, *config.RFC0115); err != nil {
			return fmt.Errorf("RFC integration validation failed: %w", err)
		}
	}
	
	return nil
}

// ValidateRFC0111Config validates RFC-0111 specific configuration
func ValidateRFC0111Config(config RFC0111Config) error {
	// Validate mandatory exclusions
	if err := ValidateRFC0111Exclusions(config.Exclusions); err != nil {
		return fmt.Errorf("RFC-0111 exclusions validation failed: %w", err)
	}
	
	// Validate PP Architecture
	if err := ValidateRFC0111PPArchitecture(config.PPArchitecture); err != nil {
		return fmt.Errorf("PP Architecture validation failed: %w", err)
	}
	
	// Validate Extended Tokens
	if err := ValidateRFC0111ExtendedTokens(config.ExtendedTokens); err != nil {
		return fmt.Errorf("Extended Tokens validation failed: %w", err)
	}
	
	return nil
}

// ValidateRFC0111Exclusions validates RFC-0111 mandatory exclusions
func ValidateRFC0111Exclusions(exclusions RFC0111Exclusions) error {
	// Web3/Blockchain exclusions
	if !exclusions.Web3Blockchain.Prohibited {
		return fmt.Errorf("RFC-0111 violation: Web3/blockchain technology must be prohibited for extended tokens")
	}
	
	// AI Operators exclusions
	if !exclusions.AIOperators.Prohibited {
		return fmt.Errorf("RFC-0111 violation: AI-controlled deployment lifecycle must be prohibited")
	}
	
	// DNA-based identities exclusions
	if !exclusions.DNABasedIdentities.Prohibited {
		return fmt.Errorf("RFC-0111 violation: DNA-based identities must be prohibited")
	}
	
	// Decentralized authorization exclusions
	if !exclusions.DecentralizedAuth.Prohibited {
		return fmt.Errorf("RFC-0111 violation: Decentralized AI authorization must be prohibited")
	}
	
	return nil
}

// ValidateRFC0111PPArchitecture validates the Power*Point architecture
func ValidateRFC0111PPArchitecture(pp RFC0111PPArchitecture) error {
	if pp.PEP.SupplySide.Entity == "" {
		return fmt.Errorf("PEP supply side entity must be specified")
	}
	if pp.PEP.DemandSide.Entity == "" {
		return fmt.Errorf("PEP demand side entity must be specified")
	}
	if pp.PDP.PrimaryPDP == "" {
		return fmt.Errorf("Primary PDP must be specified")
	}
	if pp.PIP.AuthorizationServer == "" {
		return fmt.Errorf("Authorization server in PIP must be specified")
	}
	return nil
}

// ValidateRFC0111ExtendedTokens validates extended tokens configuration
func ValidateRFC0111ExtendedTokens(tokens RFC0111ExtendedTokens) error {
	if tokens.TokenType == "" {
		return fmt.Errorf("Token type must be specified")
	}
	if len(tokens.Scope) == 0 {
		return fmt.Errorf("Token scope must be specified")
	}
	if tokens.Duration <= 0 {
		return fmt.Errorf("Token duration must be positive")
	}
	return nil
}

// ValidateRFC0115PoADefinition validates RFC-0115 PoA Definition structure
func ValidateRFC0115PoADefinition(poa RFC0115PoADefinition) error {
	// Validate parties
	if poa.Parties.Principal.Identity == "" {
		return fmt.Errorf("Principal identity is required")
	}
	if poa.Parties.AuthorizedClient.Identity == "" {
		return fmt.Errorf("Authorized client identity is required")
	}
	
	// Validate authorization scope
	if len(poa.Authorization.ApplicableSectors) == 0 {
		return fmt.Errorf("At least one applicable sector is required")
	}
	if len(poa.Authorization.ApplicableRegions) == 0 {
		return fmt.Errorf("At least one applicable region is required")
	}
	
	// Validate GAuth integration
	if !poa.GAuthContext.ExclusionsCompliant {
		return fmt.Errorf("PoA Definition must be RFC-0111 exclusions compliant")
	}
	
	return nil
}

// ValidateRFCIntegration validates the integration between RFC-0111 and RFC-0115
func ValidateRFCIntegration(rfc0111 RFC0111Config, rfc0115 RFC0115PoADefinition) error {
	// Ensure RFC-0115 client type matches RFC-0111 client capabilities
	if rfc0115.Parties.AuthorizedClient.Type == "" {
		return fmt.Errorf("RFC-0115 client type must be specified for integration")
	}
	
	// Ensure exclusions compliance across both specifications
	if !rfc0115.GAuthContext.ExclusionsCompliant {
		return fmt.Errorf("RFC-0115 must be compliant with RFC-0111 exclusions")
	}
	
	// Validate PP Architecture role consistency
	if rfc0115.GAuthContext.PPArchitectureRole == "" {
		return fmt.Errorf("RFC-0115 must specify PP Architecture role for integration")
	}
	
	// Validate token scope consistency
	if len(rfc0115.GAuthContext.ExtendedTokenScope) == 0 {
		return fmt.Errorf("RFC-0115 must specify extended token scope for integration")
	}
	
	return nil
}

// =============================================================================
// Factory Functions for Combined Implementation
// =============================================================================

// CreateCombinedRFCConfig creates a new combined RFC-0111 and RFC-0115 configuration
func CreateCombinedRFCConfig() CombinedRFCConfig {
	return CombinedRFCConfig{
		RFC0111:          CreateRFC0111Config(),
		RFC0115:          CreateRFC0115PoADefinition(),
		IntegrationLevel: "full",
		CombinedVersion:  "1.0",
		Compatibility: map[string]string{
			"rfc_0111": "1.0",
			"rfc_0115": "1.0",
			"oauth":    "2.0",
			"oidc":     "1.0",
			"mcp":      "latest",
		},
	}
}

// CreateRFC0111Config creates a new RFC-0111 compliant GAuth configuration
func CreateRFC0111Config() RFC0111Config {
	return RFC0111Config{
		PPArchitecture: CreateRFC0111PPArchitecture(),
		Exclusions:     CreateRFC0111Exclusions(),
		ExtendedTokens: CreateRFC0111ExtendedTokens(),
		GAuthRoles:     CreateRFC0111GAuthRoles(),
		Version:        "1.0",
		Status:         "active",
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
}

// CreateRFC0111Exclusions creates RFC-0111 compliant exclusions
func CreateRFC0111Exclusions() RFC0111Exclusions {
	return RFC0111Exclusions{
		Web3Blockchain: RFC0111ExclusionRule{
			Prohibited:      true,
			Description:     "Web3/blockchain technology prohibited for extended tokens",
			LicenseRequired: true,
		},
		AIOperators: RFC0111ExclusionRule{
			Prohibited:      true,
			Description:     "AI-controlled deployment lifecycle prohibited",
			LicenseRequired: true,
		},
		DNABasedIdentities: RFC0111ExclusionRule{
			Prohibited:      true,
			Description:     "DNA-based identities prohibited for biometrics",
			LicenseRequired: true,
		},
		DecentralizedAuth: RFC0111ExclusionRule{
			Prohibited:      true,
			Description:     "Decentralized AI authorization prohibited",
			LicenseRequired: true,
		},
		EnforcementLevel: "mandatory",
	}
}

// CreateRFC0111PPArchitecture creates a default PP Architecture
func CreateRFC0111PPArchitecture() RFC0111PPArchitecture {
	return RFC0111PPArchitecture{
		PEP: RFC0111PowerEnforcementPoint{
			SupplySide: RFC0111PEPSide{
				Entity:      "client",
				Enforcement: []string{"compliance_validation", "authorization_check"},
				Status:      "active",
			},
			DemandSide: RFC0111PEPSide{
				Entity:      "resource_server",
				Enforcement: []string{"token_validation", "authorization_verification"},
				Status:      "active",
			},
		},
		PDP: RFC0111PowerDecisionPoint{
			PrimaryPDP:    "client_owner",
			SecondaryPDP:  "resource_owner",
			DecisionRules: []string{"authorization_policies", "exclusions_enforcement"},
		},
		PIP: RFC0111PowerInformationPoint{
			AuthorizationServer: "gauth_server",
			DataSources:         []string{"commercial_register", "identity_provider"},
			InfoTypes:           []string{"authorization_data", "compliance_data"},
		},
		PAP: RFC0111PowerAdministrationPoint{
			ClientOwnerAuthorizer:   "owner_authorizer",
			ResourceOwnerAuthorizer: "resource_authorizer",
			PolicyManagement:        []string{"policy_creation", "policy_updates"},
		},
		PVP: RFC0111PowerVerificationPoint{
			TrustServiceProvider: "trust_service",
			VerificationMethods:  []string{"identity_verification", "authority_verification"},
			IdentityTypes:        []string{"natural_person", "legal_entity"},
		},
	}
}

// CreateRFC0111ExtendedTokens creates extended tokens configuration
func CreateRFC0111ExtendedTokens() RFC0111ExtendedTokens {
	return RFC0111ExtendedTokens{
		TokenType: "extended_token",
		Scope:     []string{"transactions", "decisions", "actions"},
		Duration:  24 * time.Hour,
		Authorization: RFC0111TokenAuthorization{
			Transactions:   []string{"financial", "legal", "operational"},
			Decisions:      []string{"strategic", "operational", "tactical"},
			Actions:        []string{"execute", "communicate", "report"},
			ResourceRights: []string{"read", "write", "modify"},
		},
		Compliance: RFC0111TokenCompliance{
			ComplianceTracking: true,
			AuditTrail:         []string{"issuance", "usage", "validation"},
			RevocationStatus:   "active",
		},
	}
}

// CreateRFC0111GAuthRoles creates default GAuth roles
func CreateRFC0111GAuthRoles() RFC0111GAuthRoles {
	return RFC0111GAuthRoles{
		ResourceOwner: RFC0111ResourceOwner{
			Identity:             "resource_owner_id",
			LegalCapacity:        true,
			TransactionAuthority: []string{"approve", "reject", "delegate"},
			DecisionAcceptance:   []string{"strategic", "operational"},
			ActionImpact:         []string{"financial", "legal", "operational"},
		},
		ResourceServer: RFC0111ResourceServer{
			Identity:           "resource_server_id",
			AssetTypes:         []string{"data", "services", "infrastructure"},
			ProtectedResources: []string{"api_endpoints", "data_stores"},
			TokenValidation:    "oauth2_bearer",
			AICapable:          false,
		},
		Client: RFC0111Client{
			Type:           RFC0111ClientTypeDigitalAgent,
			Identity:       "ai_client_id",
			AICapabilities: []string{"reasoning", "decision_making", "communication"},
			AutonomyLevel:  "supervised",
			RequestTypes:   []string{"transactions", "decisions", "actions"},
			ComplianceMode: "strict",
		},
		AuthorizationServer: RFC0111AuthServer{
			Identity:              "auth_server_id",
			ExtendedTokenIssuing:  true,
			ComplianceTracking:    true,
			PPArchitectureSupport: true,
			ExclusionsEnforced:    true,
		},
		ClientOwner: RFC0111ClientOwner{
			Identity:          "client_owner_id",
			AuthorizationLevel: "full",
			AISystemOwnership: []string{"digital_agent_1", "digital_agent_2"},
			DelegatedPowers:   []string{"transaction_approval", "decision_making"},
		},
		OwnerAuthorizer: RFC0111OwnerAuthorizer{
			Identity:           "owner_authorizer_id",
			StatutoryAuthority: true,
			AuthorizationScope: []string{"client_authorization", "resource_authorization"},
			VerificationMethod: "commercial_register",
		},
	}
}

// CreateRFC0115PoADefinition creates a new RFC-0115 compliant PoA Definition with RFC-0111 integration
func CreateRFC0115PoADefinition() *RFC0115PoADefinition {
	return &RFC0115PoADefinition{
		Parties: RFC0115Parties{
			Principal: RFC0115Principal{
				Type:     RFC0115PrincipalTypeOrganization,
				Identity: "principal_org_id",
				Organization: &RFC0115Organization{
					Type:                RFC0115OrgTypeCommercial,
					Name:                "Principal Organization",
					RegisterEntry:       "HRB 12345",
					ManagingDirector:    "Dr. Example Director",
					RegisteredAuthority: true,
				},
			},
			AuthorizedClient: RFC0115AuthorizedClient{
				Type:              RFC0115ClientTypeDigitalAgent,
				Identity:          "ai_client_id",
				Version:           "1.0",
				OperationalStatus: "active",
			},
		},
		Authorization: RFC0115AuthorizationScope{
			AuthorizationType: RFC0115AuthorizationType{
				RepresentationType: RFC0115RepresentationSole,
				SignatureType:      RFC0115SignatureSingle,
				SubProxyAuthority:  false,
			},
			ApplicableSectors: []RFC0115IndustrySector{RFC0115SectorInformationComm},
			ApplicableRegions: []RFC0115GeographicScope{
				{Type: RFC0115GeoTypeNational, Identifier: "DE", Name: "Germany"},
			},
		},
		Requirements: RFC0115Requirements{
			ValidityPeriod: RFC0115ValidityPeriod{
				Duration:    "1 year",
				AutoRenewal: false,
			},
			PowerLimits: RFC0115PowerLimits{
				QuantumResistance:  true,
				ExplicitExclusions: []string{"web3", "ai_operators", "dna_identities", "decentralized_auth"},
			},
			SecurityCompliance: RFC0115SecurityCompliance{
				CommunicationProtocols: []string{"TLS 1.3", "OAuth 2.0", "OpenID Connect", "MCP"},
			},
			JurisdictionLaw: RFC0115JurisdictionLaw{
				Language:            "German",
				GoverningLaw:        "German Law",
				PlaceOfJurisdiction: "Germany",
			},
		},
		GAuthContext: RFC0115GAuthIntegration{
			PPArchitectureRole:  "client",
			ExclusionsCompliant: true,
			ExtendedTokenScope:  []string{"transactions", "decisions", "actions"},
			AIGovernanceLevel:   "comprehensive",
		},
	}
}
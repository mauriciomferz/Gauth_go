// Package poa implements the combined GAuth 1.0 Authorization Framework and Power-of-Attorney
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
// and humanoid robots.

package poa

import (
	"time"
)

// PoADefinition represents the complete Power-of-Attorney Credential Definition
// as specified in GiFo-RFC-0115 Section 3
type PoADefinition struct {
	// A. Parties
	Parties Parties `json:"parties"`
	
	// B. Type and Scope of Authorization
	Authorization AuthorizationScope `json:"authorization"`
	
	// C. Requirements
	Requirements Requirements `json:"requirements"`
}

// Parties represents all parties involved in the PoA as per RFC-0115 Section 3.A
type Parties struct {
	Principal         Principal         `json:"principal"`
	Representative    *Representative   `json:"representative,omitempty"` // Only if principal is organization
	AuthorizedClient  AuthorizedClient  `json:"authorized_client"`
}

// Principal represents the principal party (Individual or Organization)
type Principal struct {
	Type         PrincipalType `json:"type"`
	Identity     string        `json:"identity"`
	Individual   *Individual   `json:"individual,omitempty"`
	Organization *Organization `json:"organization,omitempty"`
}

type PrincipalType string

const (
	PrincipalTypeIndividual   PrincipalType = "individual"
	PrincipalTypeOrganization PrincipalType = "organization"
)

// Individual represents a natural person principal
type Individual struct {
	Name        string `json:"name"`
	Citizenship string `json:"citizenship,omitempty"`
	// Additional individual-specific fields
}

// Organization represents an organizational principal as per RFC-0115
type Organization struct {
	Type                OrgType `json:"type"`
	Name                string  `json:"name"`
	RegisterEntry       string  `json:"register_entry,omitempty"`
	ManagingDirector    string  `json:"managing_director,omitempty"`
	RegisteredAuthority bool    `json:"registered_authority"`
}

type OrgType string

const (
	OrgTypeCommercial  OrgType = "commercial_enterprise"    // AG, Ltd., partnership, etc.
	OrgTypePublic      OrgType = "public_authority"         // federal, state, municipal, etc.
	OrgTypeNonProfit   OrgType = "non_profit_organization"  // foundation, non-profit association, gGmbH, etc.
	OrgTypeAssociation OrgType = "other_association"        // non-profit or non-charitable
	OrgTypeOther       OrgType = "other"                   // church, cooperative, community of interest, etc.
)

// Representative represents the representative/authorizer when principal is organization
type Representative struct {
	ClientOwner       *ClientOwnerInfo       `json:"client_owner,omitempty"`
	OwnerAuthorizer   *OwnerAuthorizerInfo   `json:"owner_authorizer,omitempty"`
	OtherRepresentatives []OtherRepresentative `json:"other_representatives,omitempty"`
	Other             string                 `json:"other,omitempty"`
}

type ClientOwnerInfo struct {
	Name                string `json:"name"`
	RegisteredPowerOfAttorney bool   `json:"registered_power_of_attorney"`
	CommercialRegisterEntry   bool   `json:"commercial_register_entry"`
	Other                     string `json:"other,omitempty"`
}

type OwnerAuthorizerInfo struct {
	Name                string `json:"name"`
	RegisteredPowerOfAttorney bool   `json:"registered_power_of_attorney"`
	CommercialRegisterEntry   bool   `json:"commercial_register_entry"`
	Other                     string `json:"other,omitempty"`
}

type OtherRepresentative struct {
	Name                string `json:"name"`
	RegisteredPowerOfAttorney bool   `json:"registered_power_of_attorney"`
	CommercialRegisterEntry   bool   `json:"commercial_register_entry"`
	Other                     string `json:"other,omitempty"`
}

// AuthorizedClient represents the AI client being authorized as per RFC-0115
type AuthorizedClient struct {
	Type              ClientType `json:"type"`
	Identity          string     `json:"identity"`
	Version           string     `json:"version"`
	OperationalStatus string     `json:"operational_status"` // e.g., active, revoked
	Other             string     `json:"other,omitempty"`
}

type ClientType string

const (
	ClientTypeLLM           ClientType = "llm"
	ClientTypeDigitalAgent  ClientType = "digital_agent"
	ClientTypeAgenticAI     ClientType = "agentic_ai"      // team of agents
	ClientTypeHumanoidRobot ClientType = "humanoid_robot"
	ClientTypeOther         ClientType = "other"
)

// AuthorizationScope represents Type and Scope of Authorization as per RFC-0115 Section 3.B
type AuthorizationScope struct {
	AuthorizationType     AuthorizationType     `json:"authorization_type"`
	ApplicableSectors     []IndustrySector      `json:"applicable_sectors"`
	ApplicableRegions     []GeographicScope     `json:"applicable_regions"`
	AuthorizedActions     AuthorizedActions     `json:"authorized_actions"`
}

// AuthorizationType represents the type of authorization as per RFC-0115
type AuthorizationType struct {
	RepresentationType RepresentationType `json:"representation_type"` // sole or joint
	Restrictions       []string          `json:"restrictions,omitempty"`
	SubProxyAuthority  bool              `json:"sub_proxy_authority"`
	SignatureType      SignatureType     `json:"signature_type"`
	Other              string            `json:"other,omitempty"`
}

type RepresentationType string

const (
	RepresentationSole  RepresentationType = "sole"
	RepresentationJoint RepresentationType = "joint"
)

type SignatureType string

const (
	SignatureSingle     SignatureType = "single"
	SignatureJoint      SignatureType = "joint"
	SignatureCollective SignatureType = "collective"
)

// IndustrySector represents applicable sectors using global industry codes (ISIC/NACE)
type IndustrySector string

const (
	SectorAgriculture           IndustrySector = "agriculture_forestry_fishing"
	SectorMining                IndustrySector = "mining_quarrying"
	SectorManufacturing         IndustrySector = "manufacturing"
	SectorEnergySupply          IndustrySector = "energy_supply"
	SectorWaterSupply           IndustrySector = "water_supply"
	SectorWasteManagement       IndustrySector = "waste_management"
	SectorConstruction          IndustrySector = "construction"
	SectorTrade                 IndustrySector = "trade"
	SectorVehicleMaintenance    IndustrySector = "vehicle_maintenance_repair"
	SectorTransportStorage      IndustrySector = "transport_storage"
	SectorHospitality           IndustrySector = "hospitality"
	SectorInformationComm       IndustrySector = "information_communication"
	SectorFinancialInsurance    IndustrySector = "financial_insurance_services"
	SectorRealEstate            IndustrySector = "real_estate"
	SectorProfessional          IndustrySector = "professional_scientific_technical"
	SectorBusinessServices      IndustrySector = "other_business_services"
	SectorPublicAdmin           IndustrySector = "public_administration_defence"
	SectorEducation             IndustrySector = "education"
	SectorHealthSocial          IndustrySector = "health_social_work"
	SectorArtsEntertainment     IndustrySector = "arts_entertainment_recreation"
	SectorOtherServices         IndustrySector = "other_services_sectors"
)

// GeographicScope represents applicable regions as per RFC-0115
type GeographicScope struct {
	Type       GeoType `json:"type"`
	Identifier string  `json:"identifier"`
}

type GeoType string

const (
	GeoTypeGlobal         GeoType = "global"
	GeoTypeNational       GeoType = "national"        // specify country
	GeoTypeInternational  GeoType = "international"   // specify countries/regions
	GeoTypeRegional       GeoType = "regional"        // e.g. DACH, Benelux, NAFTA
	GeoTypeSubnational    GeoType = "subnational"     // states, provinces, municipalities
	GeoTypeSpecificLocation GeoType = "specific_location" // specific locations or branches
	GeoTypeOther          GeoType = "other"
)

// AuthorizedActions represents types of transactions/decisions/actions as per RFC-0115
type AuthorizedActions struct {
	Transactions       []Transaction       `json:"transactions,omitempty"`
	Decisions          []Decision          `json:"decisions,omitempty"`
	NonPhysicalActions []NonPhysicalAction `json:"non_physical_actions,omitempty"`
	PhysicalActions    []PhysicalAction    `json:"physical_actions,omitempty"`
}

type Transaction string

const (
	TransactionLoan     Transaction = "loan_transactions"
	TransactionPurchase Transaction = "purchase_transactions"
	TransactionSale     Transaction = "sale_transactions"
	TransactionLeasing  Transaction = "leasing_rental_transactions"
	TransactionOther    Transaction = "other_transactions"
)

type Decision string

const (
	DecisionPersonnel     Decision = "personnel_decisions"
	DecisionFinancial     Decision = "financial_commitments"
	DecisionBuySell       Decision = "buy_sell_transactions"
	DecisionConceptual    Decision = "conceptual_determinations"
	DecisionDesign        Decision = "design_decisions"
	DecisionInfoSharing   Decision = "information_sharing"
	DecisionStrategic     Decision = "strategic_decisions"
	DecisionLegal         Decision = "legal_proceedings"
	DecisionAssetMgmt     Decision = "asset_management_decisions"
	DecisionOther         Decision = "other_decisions"
)

type NonPhysicalAction string

const (
	ActionSharing      NonPhysicalAction = "sharing_presenting"
	ActionBrainstorming NonPhysicalAction = "brainstorming_discussing"
	ActionResearching  NonPhysicalAction = "researching_rag"
	ActionOtherNonPhys NonPhysicalAction = "other_non_physical_actions"
)

type PhysicalAction string

const (
	ActionShipments     PhysicalAction = "shipments"
	ActionProduction    PhysicalAction = "production"
	ActionRecycling     PhysicalAction = "recycling"
	ActionStorage       PhysicalAction = "storage"
	ActionCustomization PhysicalAction = "customization"
	ActionPackage       PhysicalAction = "package"
	ActionClean         PhysicalAction = "clean"
	ActionOtherPhys     PhysicalAction = "other_actions"
)

// Requirements represents all requirements as per RFC-0115 Section 3.C
type Requirements struct {
	ValidityPeriod    ValidityPeriod    `json:"validity_period"`
	FormalRequirements FormalRequirements `json:"formal_requirements"`
	PowerLimits       PowerLimits       `json:"power_limits"`
	RightsObligations RightsObligations `json:"rights_obligations"`
	SpecialConditions SpecialConditions `json:"special_conditions"`
	DeathIncapacityRules DeathIncapacityRules `json:"death_incapacity_rules"`
	SecurityCompliance SecurityCompliance `json:"security_compliance"`
	JurisdictionLaw   JurisdictionLaw   `json:"jurisdiction_law"`
	ConflictResolution ConflictResolution `json:"conflict_resolution"`
}

// ValidityPeriod represents validity period requirements
type ValidityPeriod struct {
	StartTime              time.Time `json:"start_time"`
	EndTime                time.Time `json:"end_time"`
	AutoRenewalConditions  []string  `json:"auto_renewal_conditions,omitempty"`
	TerminationConditions  []string  `json:"termination_conditions,omitempty"`
	Other                  string    `json:"other,omitempty"`
}

// FormalRequirements represents formal requirements
type FormalRequirements struct {
	NotarialCertification bool   `json:"notarial_certification"`
	IDVerificationRequired bool   `json:"id_verification_required"`
	DigitalSignatures     bool   `json:"digital_signatures"`
	Other                 string `json:"other,omitempty"`
}

// PowerLimits represents limits of powers as per RFC-0115
type PowerLimits struct {
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

// RightsObligations represents specific rights and obligations
type RightsObligations struct {
	ReportingDuties    []string `json:"reporting_duties,omitempty"`
	LiabilityRules     []string `json:"liability_rules,omitempty"`
	CompensationRules  []string `json:"compensation_rules,omitempty"`
	Other              string   `json:"other,omitempty"`
}

// SpecialConditions represents special conditions
type SpecialConditions struct {
	ConditionalEffectiveness []string `json:"conditional_effectiveness,omitempty"`
	ImmediateNotification   []string `json:"immediate_notification,omitempty"`
	Other                   string   `json:"other,omitempty"`
}

// DeathIncapacityRules represents rules for death or incapacity
type DeathIncapacityRules struct {
	ContinuationOnDeath     bool   `json:"continuation_on_death"`
	IncapacityInstructions  string `json:"incapacity_instructions,omitempty"`
	Other                   string `json:"other,omitempty"`
}

// SecurityCompliance represents security and compliance requirements
type SecurityCompliance struct {
	CommunicationProtocols []string `json:"communication_protocols,omitempty"`
	SecurityProperties     []string `json:"security_properties,omitempty"`
	ComplianceInfo         []string `json:"compliance_info,omitempty"`
	UpdateMechanism        string   `json:"update_mechanism,omitempty"`
	Other                  string   `json:"other,omitempty"`
}

// JurisdictionLaw represents place of jurisdiction and applicable law
type JurisdictionLaw struct {
	Language            string   `json:"language"`
	GoverningLaw        string   `json:"governing_law"`
	PlaceOfJurisdiction string   `json:"place_of_jurisdiction"`
	AttachedDocuments   []string `json:"attached_documents,omitempty"`
	Other               string   `json:"other,omitempty"`
}

// ConflictResolution represents conflict resolution arrangements
type ConflictResolution struct {
	ArbitrationJurisdiction string `json:"arbitration_jurisdiction,omitempty"`
	Other                   string `json:"other,omitempty"`
}
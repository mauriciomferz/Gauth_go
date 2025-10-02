// RFC 0111 & 0115 Implementation for GAuth 1.0
// Official Gimel Foundation Implementation
// Compliant with GiFo-RFC-0111 GAuth 1.0 Authorization Framework
// Compliant with GiFo-RFC-0115 Power-of-Attorney Credential Definition
//
// Copyright (c) 2025 Gimel Foundation gGmbH i.G.
// Licensed under Apache 2.0 (building blocks: OAuth, OpenID Connect, MCP)
// Built on the excellent professional JWT foundation
//
// Demo Implementation Author: Mauricio Fernandez
// GitHub: https://github.com/mauriciomferz

package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
)

// Constants to avoid goconst violations
const (
	responseTypeToken = "token"
	statusActive      = "active"
)

// RFC 111: GAuth 1.0 Authorization Framework Types
// Implements the P*P (Power*Point) Architecture as defined in GiFo-RFC-0111

// GAuthRequest represents the standard GAuth authorization request
// Follows OAuth 2.0 pattern with power-of-attorney extensions
type GAuthRequest struct {
	// OAuth 2.0 Base Fields
	ClientID     string   `json:"client_id"`
	ResponseType string   `json:"response_type"`
	Scope        []string `json:"scope"`
	RedirectURI  string   `json:"redirect_uri"`
	State        string   `json:"state"`
	
	// GAuth Extended Token Fields (RFC 111)
	PowerType     string `json:"power_type"`
	PrincipalID   string `json:"principal_id"`
	AIAgentID     string `json:"ai_agent_id"`
	Jurisdiction  string `json:"jurisdiction"`
	LegalBasis    string `json:"legal_basis"`
	
	// RFC 115: PoA Definition Structure
	PoADefinition PoADefinition `json:"poa_definition"`
}

// RFC 115: Complete Power-of-Attorney Credential Definition
// As specified in GiFo-RFC-0115 PoA-Definition
type PoADefinition struct {
	// A. Parties (RFC 115 Section 3.A)
	Principal    Principal    `json:"principal"`
	Authorizer   Authorizer   `json:"authorizer,omitempty"`
	Client       ClientAI     `json:"client"`
	
	// B. Type and Scope of Authorization (RFC 115 Section 3.B)
	AuthorizationType AuthorizationType `json:"authorization_type"`
	ScopeDefinition   ScopeDefinition   `json:"scope_definition"`
	
	// C. Requirements (RFC 115 Section 3.C)
	Requirements Requirements `json:"requirements"`
}

// Principal represents the entity granting authority (RFC 115 Section 3.A)
type Principal struct {
	Type         PrincipalType `json:"type"`         // Individual or Organization
	Identity     string        `json:"identity"`     // Principal identifier
	Organization *Organization `json:"organization,omitempty"` // If type is Organization
}

type PrincipalType string

const (
	PrincipalTypeIndividual   PrincipalType = "individual"
	PrincipalTypeOrganization PrincipalType = "organization"
)

// Organization defines organizational principal details (RFC 115)
type Organization struct {
	Type                  OrganizationType `json:"type"`
	Name                  string           `json:"name"`
	RegisterEntry         string           `json:"register_entry,omitempty"`
	ManagingDirector      string           `json:"managing_director,omitempty"`
	RegisteredAuthority   bool             `json:"registered_authority"`
}

type OrganizationType string

const (
	OrgTypeCommercial   OrganizationType = "commercial_enterprise"  // AG, Ltd., partnership, etc.
	OrgTypePublic       OrganizationType = "public_authority"       // federal, state, municipal
	OrgTypeNonProfit    OrganizationType = "non_profit_organization" // foundation, gGmbH
	OrgTypeAssociation  OrganizationType = "association"            // non-profit or non-charitable
	OrgTypeOther        OrganizationType = "other"                  // church, cooperative, etc.
)

// Authorizer represents the representative/authorizer (RFC 115 Section 3.A)
type Authorizer struct {
	ClientOwner      *AuthorizedRepresentative `json:"client_owner,omitempty"`
	OwnersAuthorizer *AuthorizedRepresentative `json:"owners_authorizer,omitempty"`
	OtherReps        []AuthorizedRepresentative `json:"other_representatives,omitempty"`
}

type AuthorizedRepresentative struct {
	Name                string `json:"name"`
	RegisteredAuthority bool   `json:"registered_authority"`
	RegisterEntry       string `json:"register_entry,omitempty"`
	AuthorityType       string `json:"authority_type"` // Prokura, managing director, etc.
}

// ClientAI represents the authorized AI client (RFC 115 Section 3.A)
type ClientAI struct {
	Type              ClientType `json:"type"`
	Identity          string     `json:"identity"`
	Version           string     `json:"version"`
	OperationalStatus string     `json:"operational_status"` // active, revoked
}

type ClientType string

const (
	ClientTypeLLM         ClientType = "llm"
	ClientTypeAgent       ClientType = "digital_agent"
	ClientTypeAgenticAI   ClientType = "agentic_ai"        // team of agents
	ClientTypeRobot       ClientType = "humanoid_robot"
	ClientTypeOther       ClientType = "other"
)

// AuthorizationType defines the type and scope of authorization (RFC 115 Section 3.B)
type AuthorizationType struct {
	RepresentationType    RepresentationType `json:"representation_type"`    // sole or joint
	RestrictionsExclusions []string          `json:"restrictions_exclusions,omitempty"`
	SubProxyAuthority     bool              `json:"sub_proxy_authority"`    // authority to delegate
	SignatureType         SignatureType     `json:"signature_type"`         // single, joint, collective
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

// ScopeDefinition defines applicable sectors, regions, and transaction types (RFC 115 Section 3.B)
type ScopeDefinition struct {
	ApplicableSectors  []IndustrySector   `json:"applicable_sectors"`
	ApplicableRegions  []GeographicScope  `json:"applicable_regions"`
	AuthorizedActions  AuthorizedActions  `json:"authorized_actions"`
}

// IndustrySector based on ISIC/NACE codes (RFC 115)
type IndustrySector string

const (
	SectorAgriculture     IndustrySector = "agriculture_forestry_fishing"
	SectorMining          IndustrySector = "mining_quarrying"
	SectorManufacturing   IndustrySector = "manufacturing"
	SectorEnergy          IndustrySector = "energy_supply"
	SectorWater           IndustrySector = "water_supply"
	SectorWaste           IndustrySector = "waste_management"
	SectorConstruction    IndustrySector = "construction"
	SectorTrade           IndustrySector = "trade"
	SectorTransport       IndustrySector = "transport_storage"
	SectorHospitality     IndustrySector = "hospitality"
	SectorICT             IndustrySector = "information_communication"
	SectorFinancial       IndustrySector = "financial_insurance"
	SectorRealEstate      IndustrySector = "real_estate"
	SectorProfessional    IndustrySector = "professional_scientific"
	SectorBusiness        IndustrySector = "business_services"
	SectorPublicAdmin     IndustrySector = "public_administration"
	SectorEducation       IndustrySector = "education"
	SectorHealth          IndustrySector = "health_social_work"
	SectorArts            IndustrySector = "arts_entertainment"
	SectorOtherServices   IndustrySector = "other_services"
)

// GeographicScope defines regional applicability (RFC 115)
type GeographicScope struct {
	Type        GeographicType `json:"type"`
	Identifier  string         `json:"identifier"`
	Description string         `json:"description,omitempty"`
}

type GeographicType string

const (
	GeoTypeGlobal      GeographicType = "global"
	GeoTypeNational    GeographicType = "national"        // specify country
	GeoTypeRegional    GeographicType = "regional"        // DACH, Benelux, NAFTA
	GeoTypeSubnational GeographicType = "subnational"     // states, provinces
	GeoTypeSpecific    GeographicType = "specific_location" // branches
)

// AuthorizedActions defines transactions, decisions, and actions (RFC 115 Section 3.B)
type AuthorizedActions struct {
	Transactions      []TransactionType `json:"transactions"`
	Decisions         []DecisionType    `json:"decisions"`
	NonPhysicalActions []NonPhysicalAction `json:"non_physical_actions"`
	PhysicalActions   []PhysicalAction  `json:"physical_actions"`
}

// Transaction types (RFC 115)
type TransactionType string

const (
	TransactionLoan     TransactionType = "loan_transactions"
	TransactionPurchase TransactionType = "purchase_transactions"
	TransactionSale     TransactionType = "sale_transactions"
	TransactionLease    TransactionType = "leasing_rental"
)

// Decision types (RFC 115)
type DecisionType string

const (
	DecisionPersonnel    DecisionType = "personnel_decisions"
	DecisionFinancial    DecisionType = "financial_commitments"
	DecisionBuySell      DecisionType = "buy_sell_transactions"
	DecisionConceptual   DecisionType = "conceptual_determinations"
	DecisionDesign       DecisionType = "design_decisions"
	DecisionInformation  DecisionType = "information_sharing"
	DecisionStrategic    DecisionType = "strategic_decisions"
	DecisionLegal        DecisionType = "legal_proceedings"
	DecisionAsset        DecisionType = "asset_management"
)

// Non-physical action types (RFC 115)
type NonPhysicalAction string

const (
	ActionSharing      NonPhysicalAction = "sharing_presenting"
	ActionBrainstorm   NonPhysicalAction = "brainstorming"
	ActionResearch     NonPhysicalAction = "researching_rag"
)

// Physical action types (RFC 115)
type PhysicalAction string

const (
	ActionShipment     PhysicalAction = "shipments"
	ActionProduction   PhysicalAction = "production"
	ActionRecycling    PhysicalAction = "recycling"
	ActionStorage      PhysicalAction = "storage"
	ActionCustomization PhysicalAction = "customization"
	ActionPackage      PhysicalAction = "package"
	ActionClean        PhysicalAction = "clean"
)

// Requirements defines formal requirements and limits (RFC 115 Section 3.C)
type Requirements struct {
	ValidityPeriod    ValidityPeriod    `json:"validity_period"`
	FormalRequirements FormalRequirements `json:"formal_requirements"`
	PowerLimits       PowerLimits       `json:"power_limits"`
	SpecificRights    SpecificRights    `json:"specific_rights"`
	SpecialConditions SpecialConditions `json:"special_conditions"`
	DeathIncapacity   DeathIncapacity   `json:"death_incapacity"`
	SecurityCompliance SecurityCompliance `json:"security_compliance"`
	JurisdictionLaw   JurisdictionLaw   `json:"jurisdiction_law"`
	ConflictResolution ConflictResolution `json:"conflict_resolution"`
}

// FormalRequirements (RFC 115 Section 3.C)
type FormalRequirements struct {
	NotarialCertification bool `json:"notarial_certification"`
	IDVerificationRequired bool `json:"id_verification_required"`
	DigitalSignatureAccepted bool `json:"digital_signature_accepted"`
	WrittenFormRequired   bool `json:"written_form_required"`
}

// PowerLimits defines limitations on delegated powers (RFC 115 Section 3.C)
type PowerLimits struct {
	PowerLevels        []PowerLevel     `json:"power_levels"`
	InteractionBoundaries []string      `json:"interaction_boundaries"`
	ToolLimitations    []string         `json:"tool_limitations"`
	OutcomeLimitations []string         `json:"outcome_limitations"`
	ModelLimits        []ModelLimit     `json:"model_limits"`
	BehavioralLimits   []string         `json:"behavioral_limits"`
	QuantumResistance  bool             `json:"quantum_resistance"`
	ExplicitExclusions []string         `json:"explicit_exclusions"`
}

type PowerLevel struct {
	Type        string  `json:"type"`        // amount, transaction_type
	Limit       float64 `json:"limit"`
	Currency    string  `json:"currency,omitempty"`
	Description string  `json:"description"`
}

type ModelLimit struct {
	ParameterCount   int64    `json:"parameter_count,omitempty"`
	ReasoningMethods []string `json:"reasoning_methods,omitempty"`
	TrainingMethods  []string `json:"training_methods,omitempty"`
	Description      string   `json:"description"`
}

// SpecificRights defines rights and obligations (RFC 115 Section 3.C)
type SpecificRights struct {
	ReportingDuties       []string `json:"reporting_duties"`
	LiabilityRules        []string `json:"liability_rules"`
	CompensationRules     []string `json:"compensation_rules"`
	ExpenseReimbursement  bool     `json:"expense_reimbursement"`
}

// SpecialConditions (RFC 115 Section 3.C)
type SpecialConditions struct {
	ConditionalEffectiveness []string `json:"conditional_effectiveness"`
	ImmediateNotification   []string `json:"immediate_notification"`
	OtherConditions         []string `json:"other_conditions"`
}

// DeathIncapacity rules (RFC 115 Section 3.C)
type DeathIncapacity struct {
	ContinuationOnDeath    bool     `json:"continuation_on_death"`
	IncapacityInstructions []string `json:"incapacity_instructions"`
	OtherRules            []string `json:"other_rules"`
}

// SecurityCompliance (RFC 115 Section 3.C)
type SecurityCompliance struct {
	CommunicationProtocols []string `json:"communication_protocols"`
	SecurityProperties     []string `json:"security_properties"`
	ComplianceInfo        []string `json:"compliance_info"` // GDPR, eIDAS 2.0
	UpdateMechanism       string   `json:"update_mechanism"`
}

// JurisdictionLaw (RFC 115 Section 3.C)
type JurisdictionLaw struct {
	Language           string   `json:"language"`
	GoverningLaw       string   `json:"governing_law"`
	PlaceOfJurisdiction string   `json:"place_of_jurisdiction"`
	AttachedDocuments  []string `json:"attached_documents"`
}

// ConflictResolution (RFC 115 Section 3.C)
type ConflictResolution struct {
	ArbitrationAgreed   bool     `json:"arbitration_agreed"`
	CourtJurisdiction   string   `json:"court_jurisdiction,omitempty"`
	OtherMechanisms    []string `json:"other_mechanisms"`
}

// Response Types for RFC 111 & 115

// GAuthResponse represents RFC 111 compliant authorization response
type GAuthResponse struct {
	AuthorizationCode string   `json:"code"`
	State            string   `json:"state"`
	ExtendedToken    string   `json:"extended_token,omitempty"` // RFC 111 extended token
	LegalCompliance  bool     `json:"legal_compliance"`
	AuditRecordID    string   `json:"audit_record_id"`
	ExpiresIn        int      `json:"expires_in"`
	Scope           []string `json:"scope"`
	PoAValidation   PoAValidationResult `json:"poa_validation"`
}

// PoAValidationResult contains validation results for PoA Definition
type PoAValidationResult struct {
	Valid             bool     `json:"valid"`
	ValidationErrors  []string `json:"validation_errors,omitempty"`
	ComplianceLevel   string   `json:"compliance_level"`
	AttestationStatus string   `json:"attestation_status"`
}

// GAuthToken represents RFC 111 extended token with comprehensive metadata
type GAuthToken struct {
	AccessToken      string            `json:"access_token"`
	TokenType        string            `json:"token_type"`
	ExpiresIn        int               `json:"expires_in"`
	Scope           []string          `json:"scope"`
	ExtendedMetadata ExtendedMetadata  `json:"extended_metadata"`
}

// ExtendedMetadata contains the comprehensive power-of-attorney metadata
type ExtendedMetadata struct {
	PowerType        string         `json:"power_type"`
	PrincipalID      string         `json:"principal_id"`
	AIAgentID        string         `json:"ai_agent_id"`
	PoADefinition    PoADefinition  `json:"poa_definition"`
	AttestationLevel string         `json:"attestation_level"`
	ComplianceProof  []string       `json:"compliance_proof"`
}

// Backward Compatibility Types (for existing demos)
type PowerOfAttorneyRequest = GAuthRequest



// LegalFramework defines the legal validation context
type LegalFramework struct {
	Jurisdiction         string `json:"jurisdiction"`
	EntityType          string `json:"entity_type"`
	CapacityVerification bool   `json:"capacity_verification"`
	RegulationFramework  string `json:"regulation_framework,omitempty"`
	ComplianceLevel     string `json:"compliance_level,omitempty"`
}

// PowerRestrictions defines limitations on delegated powers
type PowerRestrictions struct {
	AmountLimit      float64           `json:"amount_limit,omitempty"`
	GeoRestrictions  []string         `json:"geo_restrictions,omitempty"`
	TimeRestrictions TimeRestrictions `json:"time_restrictions,omitempty"`
	ScopeRestrictions []string        `json:"scope_restrictions,omitempty"`
}

// TimeRestrictions defines temporal limitations
type TimeRestrictions struct {
	BusinessHoursOnly bool   `json:"business_hours_only,omitempty"`
	WeekdaysOnly     bool   `json:"weekdays_only,omitempty"`
	StartTime        string `json:"start_time,omitempty"`
	EndTime          string `json:"end_time,omitempty"`
	Timezone         string `json:"timezone,omitempty"`
}

// RFC 115: Advanced Delegation Framework Types

// DelegationRequest represents an RFC 115 compliant delegation request
type DelegationRequest struct {
	// Core Delegation Fields
	PrincipalID string `json:"principal_id"`
	DelegateID  string `json:"delegate_id"`
	PowerType   string `json:"power_type"`
	Scope       []string `json:"scope"`
	
	// Advanced RFC 115 Features
	Restrictions            PowerRestrictions       `json:"restrictions"`
	AttestationRequirement  AttestationRequirement  `json:"attestation_requirement"`
	ValidityPeriod         ValidityPeriod          `json:"validity_period"`
	Jurisdiction           string                  `json:"jurisdiction"`
	LegalBasis             string                  `json:"legal_basis"`
}

// AttestationRequirement defines multi-level attestation requirements
type AttestationRequirement struct {
	Type           string   `json:"type"`
	Level          string   `json:"level"`
	MultiSignature bool     `json:"multi_signature"`
	Attesters      []string `json:"attesters"`
	RequiredCount  int      `json:"required_count,omitempty"`
}

// ValidityPeriod defines precise temporal and geographic constraints
type ValidityPeriod struct {
	StartTime       time.Time      `json:"start_time"`
	EndTime         time.Time      `json:"end_time"`
	TimeWindows     []TimeWindow   `json:"time_windows,omitempty"`
	GeoConstraints  []string       `json:"geo_constraints,omitempty"`
	SuspensionRules []string       `json:"suspension_rules,omitempty"`
}

// TimeWindow defines specific operational time periods
type TimeWindow struct {
	Start    string `json:"start"`
	End      string `json:"end"`
	Timezone string `json:"timezone"`
	Days     []string `json:"days,omitempty"`
}

// Professional RFC Implementation Service

// RFCCompliantService implements RFC 111 & 115 on top of professional JWT
type RFCCompliantService struct {
	jwtService    *ProperJWTService
	legalValidator *LegalFrameworkValidator
	delegationManager *DelegationManager
	attestationService *AttestationService
}

// NewRFCCompliantService creates a new RFC 111/115 compliant service
func NewRFCCompliantService(issuer, audience string) (*RFCCompliantService, error) {
	jwtService, err := NewProperJWTService(issuer, audience)
	if err != nil {
		return nil, fmt.Errorf("failed to create JWT service: %w", err)
	}
	
	return &RFCCompliantService{
		jwtService:         jwtService,
		legalValidator:     NewLegalFrameworkValidator(),
		delegationManager:  NewDelegationManager(),
		attestationService: NewAttestationService(),
	}, nil
}

// RFC 111 Implementation - GAuth 1.0 Authorization Framework

// AuthorizeGAuth implements RFC 111 authorization flow with complete PoA Definition
func (s *RFCCompliantService) AuthorizeGAuth(ctx context.Context, req GAuthRequest) (*GAuthResponse, error) {
	// Step 1: Validate PoA Definition (RFC 115 requirement)
	poaValidation, err := s.validatePoADefinition(ctx, req.PoADefinition)
	if err != nil {
		return nil, fmt.Errorf("PoA definition validation failed: %w", err)
	}
	
	// Step 2: Validate principal capacity (RFC 111 requirement)
	if err := s.validatePrincipalCapacity(ctx, req.PrincipalID, req.PowerType); err != nil {
		return nil, fmt.Errorf("principal capacity validation failed: %w", err)
	}
	
	// Step 3: Validate AI client capabilities (RFC 111 requirement)
	if err := s.validateAIClientCapabilities(ctx, req.PoADefinition.Client, req.PoADefinition.ScopeDefinition.AuthorizedActions); err != nil {
		return nil, fmt.Errorf("AI client capability validation failed: %w", err)
	}
	
	// Step 4: Validate legal framework and jurisdiction (RFC 111 & 115)
	if err := s.validateLegalCompliance(ctx, req.PoADefinition.Requirements.JurisdictionLaw, req.Jurisdiction); err != nil {
		return nil, fmt.Errorf("legal compliance validation failed: %w", err)
	}
	
	// Step 5: Generate authorization code with extended metadata
	authCode, err := s.generateGAuthCode(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("authorization code generation failed: %w", err)
	}
	
	// Step 6: Create comprehensive audit record (RFC 111 requirement)
	auditRecord := s.createGAuthAuditRecord(ctx, req, authCode)
	
	// Step 7: Generate extended token if requested
	var extendedToken string
	if req.ResponseType == responseTypeToken {
		extendedToken, err = s.generateExtendedToken(ctx, req, authCode)
		if err != nil {
			return nil, fmt.Errorf("extended token generation failed: %w", err)
		}
	}
	
	return &GAuthResponse{
		AuthorizationCode: authCode,
		State:            req.State,
		ExtendedToken:    extendedToken,
		LegalCompliance:  poaValidation.Valid,
		AuditRecordID:    auditRecord.ID,
		ExpiresIn:        300, // 5 minutes as per RFC 111
		Scope:           req.Scope,
		PoAValidation:   *poaValidation,
	}, nil
}

// CreatePowerOfAttorneyToken implements RFC 111 token exchange
func (s *RFCCompliantService) CreatePowerOfAttorneyToken(ctx context.Context, authCode string) (*PowerOfAttorneyToken, error) {
	// Validate authorization code
	authData, err := s.validateAuthorizationCode(ctx, authCode)
	if err != nil {
		return nil, fmt.Errorf("invalid authorization code: %w", err)
	}
	
	// Enhanced metadata will be embedded in JWT claims during token creation
	// The professional JWT service handles the actual token creation
	
	// Use professional JWT service to create token
	tokenString, err := s.jwtService.CreateToken(authData.PrincipalID, authData.Scope, time.Hour)
	if err != nil {
		return nil, fmt.Errorf("token creation failed: %w", err)
	}
	
	return &PowerOfAttorneyToken{
		AccessToken:  tokenString,
		TokenType:    "bearer",
		ExpiresIn:    3600,
		Scope:        authData.Scope,
		PowerType:    authData.PowerType,
		Restrictions: authData.Restrictions,
	}, nil
}

// RFC 115 Implementation

// CreateAdvancedDelegation implements RFC 115 delegation flow
func (s *RFCCompliantService) CreateAdvancedDelegation(ctx context.Context, req DelegationRequest) (*DelegationResponse, error) {
	// Step 1: Validate delegation request (RFC 115 requirements)
	if err := s.validateDelegationRequest(ctx, req); err != nil {
		return nil, fmt.Errorf("delegation request validation failed: %w", err)
	}
	
	// Step 2: Process attestation requirements (RFC 115 feature)
	attestations, err := s.attestationService.ProcessAttestationRequirement(ctx, req.AttestationRequirement)
	if err != nil {
		return nil, fmt.Errorf("attestation processing failed: %w", err)
	}
	
	// Step 3: Create delegation record with validity period
	delegation := &Delegation{
		ID:              generateSecureDelegationID(),
		PrincipalID:     req.PrincipalID,
		DelegateID:      req.DelegateID,
		PowerType:       req.PowerType,
		Scope:          req.Scope,
		Restrictions:   req.Restrictions,
		ValidityPeriod: req.ValidityPeriod,
		Attestations:   attestations,
		CreatedAt:      time.Now(),
		Status:         "active",
	}
	
	// Step 4: Store delegation (RFC 115 requirement)
	if err := s.delegationManager.StoreDelegation(ctx, delegation); err != nil {
		return nil, fmt.Errorf("delegation storage failed: %w", err)
	}
	
	// Step 5: Generate delegation token with cryptographic proof
	delegationToken, err := s.generateDelegationToken(ctx, delegation)
	if err != nil {
		return nil, fmt.Errorf("delegation token generation failed: %w", err)
	}
	
	return &DelegationResponse{
		DelegationID:    delegation.ID,
		DelegationToken: delegationToken,
		Status:          "active",
		ValidUntil:      req.ValidityPeriod.EndTime,
		Attestations:    attestations,
		ComplianceStatus: "rfc115_compliant",
	}, nil
}

// Supporting Types and Responses

// PowerOfAttorneyResponse represents RFC 111 authorization response
type PowerOfAttorneyResponse struct {
	AuthorizationCode string   `json:"code"`
	State            string   `json:"state"`
	LegalCompliance  bool     `json:"legal_compliance"`
	AuditRecordID    string   `json:"audit_record_id"`
	ExpiresIn        int      `json:"expires_in"`
	Scope           []string `json:"scope"`
}

// PowerOfAttorneyClaims extends CustomClaims with power-of-attorney metadata
type PowerOfAttorneyClaims struct {
	CustomClaims
	PowerType        string            `json:"power_type"`
	PrincipalID      string            `json:"principal_id"`
	AIAgentID        string            `json:"ai_agent_id"`
	LegalFramework   LegalFramework    `json:"legal_framework"`
	Restrictions     PowerRestrictions `json:"restrictions"`
	AttestationLevel string            `json:"attestation_level"`
}

// PowerOfAttorneyToken represents RFC 111 access token
type PowerOfAttorneyToken struct {
	AccessToken  string            `json:"access_token"`
	TokenType    string            `json:"token_type"`
	ExpiresIn    int               `json:"expires_in"`
	Scope        []string          `json:"scope"`
	PowerType    string            `json:"power_type"`
	Restrictions PowerRestrictions `json:"restrictions"`
}

// Delegation represents an RFC 115 delegation record
type Delegation struct {
	ID              string                 `json:"id"`
	PrincipalID     string                 `json:"principal_id"`
	DelegateID      string                 `json:"delegate_id"`
	PowerType       string                 `json:"power_type"`
	Scope          []string               `json:"scope"`
	Restrictions   PowerRestrictions      `json:"restrictions"`
	ValidityPeriod ValidityPeriod         `json:"validity_period"`
	Attestations   []Attestation          `json:"attestations"`
	CreatedAt      time.Time              `json:"created_at"`
	Status         string                 `json:"status"`
}

// DelegationResponse represents RFC 115 delegation response
type DelegationResponse struct {
	DelegationID     string        `json:"delegation_id"`
	DelegationToken  string        `json:"delegation_token"`
	Status           string        `json:"status"`
	ValidUntil       time.Time     `json:"valid_until"`
	Attestations     []Attestation `json:"attestations"`
	ComplianceStatus string        `json:"compliance_status"`
}

// Attestation represents a digital attestation record
type Attestation struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	Attester  string    `json:"attester"`
	Timestamp time.Time `json:"timestamp"`
	Signature string    `json:"signature"`
	Status    string    `json:"status"`
}

// Utility functions

// generateSecureDelegationID creates a cryptographically secure delegation ID
func generateSecureDelegationID() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback for mock implementation
		return fmt.Sprintf("del_fallback_%d", time.Now().UnixNano())
	}
	return "del_" + hex.EncodeToString(bytes)
}

// generateAuthorizationCode creates a secure authorization code
func (s *RFCCompliantService) generateAuthorizationCode(ctx context.Context, req PowerOfAttorneyRequest) (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return "auth_" + hex.EncodeToString(bytes), nil
}

// Supporting service implementations (RFC-compliant)

// LegalFrameworkValidator validates legal frameworks per RFC specifications
type LegalFrameworkValidator struct {
	supportedJurisdictions map[string]bool
	supportedEntityTypes   map[string]bool
	complianceRules       map[string][]string
}

func NewLegalFrameworkValidator() *LegalFrameworkValidator {
	return &LegalFrameworkValidator{
		supportedJurisdictions: map[string]bool{
			"US": true, "EU": true, "CA": true, "UK": true, "AU": true,
		},
		supportedEntityTypes: map[string]bool{
			"corporation": true, "llc": true, "partnership": true, 
			"individual": true, "trust": true, "government": true,
		},
		complianceRules: map[string][]string{
			"US": {"SOX", "GDPR", "CCPA", "FINRA"},
			"EU": {"GDPR", "MiFID", "PSD2"},
			"CA": {"PIPEDA", "OSC"},
		},
	}
}

func (v *LegalFrameworkValidator) ValidateFramework(ctx context.Context, framework LegalFramework) error {
	// RFC 111 Requirement: Validate jurisdiction support
	if !v.supportedJurisdictions[framework.Jurisdiction] {
		return fmt.Errorf("unsupported jurisdiction: %s", framework.Jurisdiction)
	}
	
	// RFC 111 Requirement: Validate entity type
	if !v.supportedEntityTypes[framework.EntityType] {
		return fmt.Errorf("unsupported entity type: %s", framework.EntityType)
	}
	
	// RFC 111 Requirement: Capacity verification must be explicit
	if !framework.CapacityVerification {
		return fmt.Errorf("capacity verification is required for power-of-attorney")
	}
	
	// RFC 111 Requirement: Check compliance framework requirements
	if framework.RegulationFramework != "" {
		rules, exists := v.complianceRules[framework.Jurisdiction]
		if !exists {
			return fmt.Errorf("no compliance rules defined for jurisdiction: %s", framework.Jurisdiction)
		}
		
		found := false
		for _, rule := range rules {
			if rule == framework.RegulationFramework {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("unsupported regulation framework %s for jurisdiction %s", 
				framework.RegulationFramework, framework.Jurisdiction)
		}
	}
	
	return nil
}

// DelegationManager manages delegation lifecycle per RFC 115
type DelegationManager struct {
	delegations     map[string]*Delegation
	delegationChains map[string][]string // Track delegation chains
	maxChainDepth   int
}

func NewDelegationManager() *DelegationManager {
	return &DelegationManager{
		delegations:     make(map[string]*Delegation),
		delegationChains: make(map[string][]string),
		maxChainDepth:   5, // RFC 115 maximum delegation depth
	}
}

func (m *DelegationManager) StoreDelegation(ctx context.Context, delegation *Delegation) error {
	// RFC 115 Requirement: Validate delegation doesn't create cycles
	if err := m.validateDelegationChain(delegation); err != nil {
		return fmt.Errorf("delegation chain validation failed: %w", err)
	}
	
	// RFC 115 Requirement: Validate time bounds
	now := time.Now()
	if delegation.ValidityPeriod.StartTime.After(now.Add(24 * time.Hour)) {
		return fmt.Errorf("delegation start time cannot be more than 24 hours in future")
	}
	if delegation.ValidityPeriod.EndTime.Before(now) {
		return fmt.Errorf("delegation end time cannot be in the past")
	}
	if delegation.ValidityPeriod.EndTime.Sub(delegation.ValidityPeriod.StartTime) > 365*24*time.Hour {
		return fmt.Errorf("delegation period cannot exceed 1 year")
	}
	
	// RFC 115 Requirement: Validate attestation requirements met
	if len(delegation.Attestations) == 0 {
		return fmt.Errorf("at least one attestation is required for delegation")
	}
	
	// Store delegation
	m.delegations[delegation.ID] = delegation
	
	// Update delegation chain tracking
	chain := m.delegationChains[delegation.PrincipalID]
	chain = append(chain, delegation.ID)
	m.delegationChains[delegation.PrincipalID] = chain
	
	return nil
}

func (m *DelegationManager) validateDelegationChain(delegation *Delegation) error {
	// Check if creating a cycle
	if delegation.PrincipalID == delegation.DelegateID {
		return fmt.Errorf("self-delegation not allowed")
	}
	
	// Check chain depth
	chain := m.delegationChains[delegation.PrincipalID]
	if len(chain) >= m.maxChainDepth {
		return fmt.Errorf("delegation chain depth exceeds maximum of %d", m.maxChainDepth)
	}
	
	// Check for cycles in delegation chain
	visited := make(map[string]bool)
	current := delegation.DelegateID
	
	for {
		if visited[current] {
			return fmt.Errorf("delegation would create a cycle")
		}
		visited[current] = true
		
		// Find if current is delegating to someone else
		found := false
		for _, del := range m.delegations {
			if del.PrincipalID == current && del.Status == statusActive {
				current = del.DelegateID
				found = true
				break
			}
		}
		
		if !found {
			break
		}
		
		if len(visited) > m.maxChainDepth {
			return fmt.Errorf("delegation chain too deep")
		}
	}
	
	return nil
}

func (m *DelegationManager) GetDelegation(ctx context.Context, delegationID string) (*Delegation, error) {
	delegation, exists := m.delegations[delegationID]
	if !exists {
		return nil, fmt.Errorf("delegation not found: %s", delegationID)
	}
	
	// Check if delegation is still valid
	now := time.Now()
	if now.Before(delegation.ValidityPeriod.StartTime) || now.After(delegation.ValidityPeriod.EndTime) {
		return nil, fmt.Errorf("delegation is not currently valid")
	}
	
	return delegation, nil
}

func (m *DelegationManager) RevokeDelegation(ctx context.Context, delegationID string) error {
	delegation, exists := m.delegations[delegationID]
	if !exists {
		return fmt.Errorf("delegation not found: %s", delegationID)
	}
	
	delegation.Status = "revoked"
	return nil
}

// AttestationService handles multi-level attestations per RFC 115
type AttestationService struct {
	attestationStore map[string]Attestation
	validAttesterTypes map[string]bool
	crypto *ProperCryptoService // Use professional crypto for signatures
}

func NewAttestationService() *AttestationService {
	crypto, _ := NewProperCryptoService() // Use existing professional crypto
	return &AttestationService{
		attestationStore: make(map[string]Attestation),
		validAttesterTypes: map[string]bool{
			"notary_public": true,
			"legal_counsel": true,
			"board_member": true,
			"witness": true,
			"digital_signature": true,
			"biometric": true,
		},
		crypto: crypto,
	}
}

func (s *AttestationService) ProcessAttestationRequirement(ctx context.Context, req AttestationRequirement) ([]Attestation, error) {
	var attestations []Attestation
	
	// RFC 115 Requirement: Validate attestation type
	if !s.validAttesterTypes[req.Type] {
		return nil, fmt.Errorf("invalid attestation type: %s", req.Type)
	}
	
	// RFC 115 Requirement: Validate attestation level
	validLevels := map[string]bool{"basic": true, "enhanced": true, "maximum": true}
	if !validLevels[req.Level] {
		return nil, fmt.Errorf("invalid attestation level: %s", req.Level)
	}
	
	// RFC 115 Requirement: Process each required attester
	requiredCount := req.RequiredCount
	if requiredCount == 0 {
		requiredCount = len(req.Attesters)
	}
	
	if len(req.Attesters) < requiredCount {
		return nil, fmt.Errorf("insufficient attesters: need %d, provided %d", requiredCount, len(req.Attesters))
	}
	
	// Create attestations for each attester
	for i, attester := range req.Attesters {
		if i >= requiredCount {
			break // Only process required count
		}
		
		attestation := s.createAttestation(req.Type, attester, req.Level)
		attestations = append(attestations, attestation)
		s.attestationStore[attestation.ID] = attestation
	}
	
	// RFC 115 Requirement: Validate multi-signature if required
	if req.MultiSignature && len(attestations) < 2 {
		return nil, fmt.Errorf("multi-signature required but insufficient attestations")
	}
	
	return attestations, nil
}

func (s *AttestationService) createAttestation(attType, attester, level string) Attestation {
	// Generate secure attestation ID
	bytes := make([]byte, 16)
	var attestationID string
	if _, err := rand.Read(bytes); err != nil {
		// Fallback for mock implementation
		attestationID = fmt.Sprintf("att_fallback_%d", time.Now().UnixNano())
	} else {
		attestationID = "att_" + hex.EncodeToString(bytes)
	}
	
	// Create cryptographic signature using professional crypto service
	data := fmt.Sprintf("%s:%s:%s:%d", attType, attester, level, time.Now().Unix())
	signature, _ := s.crypto.SignData([]byte(data))
	
	return Attestation{
		ID:        attestationID,
		Type:      attType,
		Attester:  attester,
		Timestamp: time.Now(),
		Signature: hex.EncodeToString(signature),
		Status:    "verified",
	}
}

func (s *AttestationService) ValidateAttestation(ctx context.Context, attestationID string) (*Attestation, error) {
	attestation, exists := s.attestationStore[attestationID]
	if !exists {
		return nil, fmt.Errorf("attestation not found: %s", attestationID)
	}
	
	// RFC 115 Requirement: Check attestation age (max 30 days for most types)
	maxAge := 30 * 24 * time.Hour
	if attestation.Type == "notary_public" {
		maxAge = 90 * 24 * time.Hour // Notary attestations valid longer
	}
	
	if time.Since(attestation.Timestamp) > maxAge {
		return nil, fmt.Errorf("attestation expired: %s", attestationID)
	}
	
	return &attestation, nil
}

// ProperCryptoService stub for demonstration - in real implementation would use existing proper_crypto.go
type ProperCryptoService struct{}

func NewProperCryptoService() (*ProperCryptoService, error) {
	return &ProperCryptoService{}, nil
}

func (p *ProperCryptoService) SignData(data []byte) ([]byte, error) {
	// In real implementation, this would use proper_crypto.go functions
	hash := make([]byte, 32)
	if _, err := rand.Read(hash); err != nil {
		// Fallback for mock implementation
		for i := range hash {
			hash[i] = byte(i)
		}
	}
	return hash, nil
}

// Validation helper methods (RFC-compliant implementations)

func (s *RFCCompliantService) validatePrincipalCapacity(ctx context.Context, principalID, powerType string) error {
	// RFC 111 Requirement: Validate principal has authority to grant power-of-attorney
	
	// Basic principal validation
	if principalID == "" {
		return fmt.Errorf("principal ID cannot be empty")
	}
	
	// Power type validation
	validPowerTypes := map[string]bool{
		"financial_transactions": true,
		"financial_advisory_powers": true,
		"legal_decisions": true,
		"operational_management": true,
		"data_management": true,
		"system_administration": true,
		"asset_management": true,
		"risk_analysis": true,
	}
	
	if !validPowerTypes[powerType] {
		return fmt.Errorf("invalid power type: %s", powerType)
	}
	
	// RFC 111 Requirement: Certain power types require enhanced verification
	enhancedVerificationRequired := map[string]bool{
		"financial_transactions": true,
		"legal_decisions": true,
	}
	
	if enhancedVerificationRequired[powerType] {
		// In a real implementation, this would check against identity databases,
		// corporate records, etc. For demonstration, we validate format
		if len(principalID) < 8 {
			return fmt.Errorf("principal ID too short for enhanced verification required by power type %s", powerType)
		}
	}
	
	return nil
}

func (s *RFCCompliantService) validateAIAgentCapabilities(ctx context.Context, agentID string, powers []string) error {
	// RFC 111 Requirement: Validate AI agent capabilities match requested powers
	
	if agentID == "" {
		return fmt.Errorf("AI agent ID cannot be empty")
	}
	
	// Define AI agent capability matrix
	aiCapabilities := map[string][]string{
		"ai_trading_assistant_v2": {"sign_contracts", "manage_investments", "authorize_payments"},
		"ai_legal_assistant": {"review_documents", "draft_contracts", "legal_research"},
		"ai_data_processor": {"data_analysis", "report_generation", "data_export"},
		"ai_operations_manager": {"resource_allocation", "schedule_management", "status_reporting"},
	}
	
	capabilities, exists := aiCapabilities[agentID]
	if !exists {
		return fmt.Errorf("unknown AI agent: %s", agentID)
	}
	
	// RFC 111 Requirement: All requested powers must be within AI capabilities
	for _, requestedPower := range powers {
		found := false
		for _, capability := range capabilities {
			if capability == requestedPower {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("AI agent %s does not have capability for power: %s", agentID, requestedPower)
		}
	}
	
	return nil
}

func (s *RFCCompliantService) validateAuthorizationCode(ctx context.Context, code string) (*AuthorizationData, error) {
	// RFC 111 Requirement: Validate authorization code format and freshness
	
	if !strings.HasPrefix(code, "auth_") {
		return nil, fmt.Errorf("invalid authorization code format")
	}
	
	if len(code) < 20 {
		return nil, fmt.Errorf("authorization code too short")
	}
	
	// In a real implementation, this would be stored and validated against a database
	// For demonstration, we'll create mock authorization data
	authData := &AuthorizationData{
		Scope:       []string{"ai_power_of_attorney", "financial_transactions"},
		PrincipalID: "corp_ceo_123",
		AIAgentID:   "ai_trading_assistant_v2",
		PowerType:   "financial_transactions",
		LegalFramework: LegalFramework{
			Jurisdiction:         "US",
			EntityType:          "corporation",
			CapacityVerification: true,
		},
		Restrictions: PowerRestrictions{
			AmountLimit: 50000.0,
			GeoRestrictions: []string{"US", "EU"},
		},
	}
	
	return authData, nil
}

func (s *RFCCompliantService) validateDelegationRequest(ctx context.Context, req DelegationRequest) error {
	// RFC 115 Requirement: Comprehensive delegation request validation
	
	// Basic field validation
	if req.PrincipalID == "" {
		return fmt.Errorf("principal ID is required")
	}
	if req.DelegateID == "" {
		return fmt.Errorf("delegate ID is required")
	}
	if req.PowerType == "" {
		return fmt.Errorf("power type is required")
	}
	if len(req.Scope) == 0 {
		return fmt.Errorf("scope cannot be empty")
	}
	
	// RFC 115 Requirement: Validate delegation period
	if req.ValidityPeriod.EndTime.Before(req.ValidityPeriod.StartTime) {
		return fmt.Errorf("end time must be after start time")
	}
	
	duration := req.ValidityPeriod.EndTime.Sub(req.ValidityPeriod.StartTime)
	if duration > 365*24*time.Hour {
		return fmt.Errorf("delegation period cannot exceed 1 year")
	}
	
	// RFC 115 Requirement: Validate geographic constraints
	validRegions := map[string]bool{
		"US": true, "EU": true, "CA": true, "UK": true, "AU": true,
		"US_eastern": true, "EU_central": true,
	}
	
	for _, region := range req.ValidityPeriod.GeoConstraints {
		if !validRegions[region] {
			return fmt.Errorf("invalid geographic constraint: %s", region)
		}
	}
	
	// RFC 115 Requirement: Validate time windows
	for _, window := range req.ValidityPeriod.TimeWindows {
		if window.Start == "" || window.End == "" {
			return fmt.Errorf("time window start and end are required")
		}
		// Additional time format validation would go here
	}
	
	return nil
}

func (s *RFCCompliantService) createPowerOfAttorneyAuditRecord(ctx context.Context, req PowerOfAttorneyRequest, authCode string) *AuditRecord {
	// RFC 111 Requirement: Create comprehensive audit record
	
	// Generate secure audit ID
	bytes := make([]byte, 16)
	var auditID string
	if _, err := rand.Read(bytes); err != nil {
		// Fallback for mock implementation
		auditID = fmt.Sprintf("audit_fallback_%d", time.Now().UnixNano())
	} else {
		auditID = "audit_" + hex.EncodeToString(bytes)
	}
	
	// In a real implementation, this would create a comprehensive audit record
	// with all request details, timestamps, etc.
	return &AuditRecord{
		ID:                auditID,
		Timestamp:         time.Now(),
		PrincipalID:       req.PrincipalID,
		AIAgentID:         req.AIAgentID,
		PowerType:         req.PowerType,
		AuthorizationCode: authCode,
		RequestedPowers:   s.extractRequestedPowers(req.PoADefinition),
		Restrictions:      s.extractRestrictions(req.PoADefinition),
		LegalFramework:    s.extractLegalFramework(req.PoADefinition),
	}
}

func (s *RFCCompliantService) generateDelegationToken(ctx context.Context, delegation *Delegation) (string, error) {
	// Use professional JWT service to create delegation token
	return s.jwtService.CreateToken(delegation.PrincipalID, delegation.Scope, time.Until(delegation.ValidityPeriod.EndTime))
}

// Supporting types for RFC implementation
type AuthorizationData struct {
	RegisteredClaims interface{}
	Scope           []string
	PrincipalID     string
	AIAgentID       string
	PowerType       string
	LegalFramework  LegalFramework
	Restrictions    PowerRestrictions
}

type AuditRecord struct {
	ID                string                `json:"id"`
	Timestamp         time.Time            `json:"timestamp"`
	PrincipalID       string               `json:"principal_id"`
	AIAgentID         string               `json:"ai_agent_id"`
	PowerType         string               `json:"power_type"`
	AuthorizationCode string               `json:"authorization_code"`
	RequestedPowers   []string             `json:"requested_powers"`
	Restrictions      PowerRestrictions    `json:"restrictions"`
	LegalFramework    LegalFramework       `json:"legal_framework"`
}

// RFC 115 PoA Definition Validation Methods

// validatePoADefinition performs comprehensive RFC 115 validation
func (s *RFCCompliantService) validatePoADefinition(ctx context.Context, poa PoADefinition) (*PoAValidationResult, error) {
	result := &PoAValidationResult{
		Valid:             true,
		ValidationErrors:  []string{},
		ComplianceLevel:   "rfc115_compliant",  
		AttestationStatus: "pending",
	}
	
	// Validate Principal (RFC 115 Section 3.A)
	if err := s.validatePrincipal(ctx, poa.Principal); err != nil {
		result.Valid = false
		result.ValidationErrors = append(result.ValidationErrors, fmt.Sprintf("principal validation: %v", err))
	}
	
	// Validate Client AI (RFC 115 Section 3.A)
	if err := s.validateClientAI(ctx, poa.Client); err != nil {
		result.Valid = false
		result.ValidationErrors = append(result.ValidationErrors, fmt.Sprintf("client validation: %v", err))
	}
	
	// Validate Authorization Type (RFC 115 Section 3.B)
	if err := s.validateAuthorizationType(ctx, poa.AuthorizationType); err != nil {
		result.Valid = false
		result.ValidationErrors = append(result.ValidationErrors, fmt.Sprintf("authorization type: %v", err))
	}
	
	// Validate Scope Definition (RFC 115 Section 3.B)
	if err := s.validateScopeDefinition(ctx, poa.ScopeDefinition); err != nil {
		result.Valid = false
		result.ValidationErrors = append(result.ValidationErrors, fmt.Sprintf("scope definition: %v", err))
	}
	
	// Validate Requirements (RFC 115 Section 3.C)
	if err := s.validateRequirements(ctx, poa.Requirements); err != nil {
		result.Valid = false
		result.ValidationErrors = append(result.ValidationErrors, fmt.Sprintf("requirements: %v", err))
	}
	
	if !result.Valid {
		return result, fmt.Errorf("PoA definition validation failed: %d errors", len(result.ValidationErrors))
	}
	
	result.AttestationStatus = "validated"
	return result, nil
}

// validatePrincipal validates principal entity (RFC 115 Section 3.A)
func (s *RFCCompliantService) validatePrincipal(ctx context.Context, principal Principal) error {
	if principal.Identity == "" {
		return fmt.Errorf("principal identity is required")
	}
	
	switch principal.Type {
	case PrincipalTypeIndividual:
		// Individual validation
		return nil
	case PrincipalTypeOrganization:
		if principal.Organization == nil {
			return fmt.Errorf("organization details required for organizational principal")
		}
		return s.validateOrganization(ctx, *principal.Organization)
	default:
		return fmt.Errorf("unsupported principal type: %s", principal.Type)
	}
}

// validateOrganization validates organizational details
func (s *RFCCompliantService) validateOrganization(ctx context.Context, org Organization) error {
	if org.Name == "" {
		return fmt.Errorf("organization name is required")
	}
	
	validOrgTypes := map[OrganizationType]bool{
		OrgTypeCommercial: true, OrgTypePublic: true, OrgTypeNonProfit: true,
		OrgTypeAssociation: true, OrgTypeOther: true,
	}
	
	if !validOrgTypes[org.Type] {
		return fmt.Errorf("invalid organization type: %s", org.Type)
	}
	
	return nil
}

// validateClientAI validates AI client details (RFC 115 Section 3.A)
func (s *RFCCompliantService) validateClientAI(ctx context.Context, client ClientAI) error {
	if client.Identity == "" {
		return fmt.Errorf("client identity is required")
	}
	
	if client.Version == "" {
		return fmt.Errorf("client version is required")
	}
	
	validClientTypes := map[ClientType]bool{
		ClientTypeLLM: true, ClientTypeAgent: true, ClientTypeAgenticAI: true,
		ClientTypeRobot: true, ClientTypeOther: true,
	}
	
	if !validClientTypes[client.Type] {
		return fmt.Errorf("invalid client type: %s", client.Type)
	}
	
	if client.OperationalStatus != "active" {
		return fmt.Errorf("client must be in active operational status, got: %s", client.OperationalStatus)
	}
	
	return nil
}

// validateAuthorizationType validates authorization type (RFC 115 Section 3.B)
func (s *RFCCompliantService) validateAuthorizationType(ctx context.Context, authType AuthorizationType) error {
	validRepTypes := map[RepresentationType]bool{
		RepresentationSole: true, RepresentationJoint: true,
	}
	
	if !validRepTypes[authType.RepresentationType] {
		return fmt.Errorf("invalid representation type: %s", authType.RepresentationType)
	}
	
	validSigTypes := map[SignatureType]bool{
		SignatureSingle: true, SignatureJoint: true, SignatureCollective: true,
	}
	
	if !validSigTypes[authType.SignatureType] {
		return fmt.Errorf("invalid signature type: %s", authType.SignatureType)
	}
	
	return nil
}

// validateScopeDefinition validates scope definition (RFC 115 Section 3.B)
func (s *RFCCompliantService) validateScopeDefinition(ctx context.Context, scope ScopeDefinition) error {
	if len(scope.ApplicableSectors) == 0 {
		return fmt.Errorf("at least one applicable sector is required")
	}
	
	if len(scope.ApplicableRegions) == 0 {
		return fmt.Errorf("at least one applicable region is required")
	}
	
	// Validate that authorized actions are specified
	actions := scope.AuthorizedActions
	if len(actions.Transactions) == 0 && len(actions.Decisions) == 0 && 
	   len(actions.NonPhysicalActions) == 0 && len(actions.PhysicalActions) == 0 {
		return fmt.Errorf("at least one type of authorized action is required")
	}
	
	return nil
}

// validateRequirements validates requirements (RFC 115 Section 3.C)
func (s *RFCCompliantService) validateRequirements(ctx context.Context, req Requirements) error {
	// Validate validity period
	if req.ValidityPeriod.EndTime.Before(req.ValidityPeriod.StartTime) {
		return fmt.Errorf("validity period end time must be after start time")
	}
	
	// Validate maximum delegation period (1 year as per RFC)
	maxDuration := 365 * 24 * time.Hour
	if req.ValidityPeriod.EndTime.Sub(req.ValidityPeriod.StartTime) > maxDuration {
		return fmt.Errorf("delegation period cannot exceed 1 year")
	}
	
	// Validate jurisdiction law
	if req.JurisdictionLaw.GoverningLaw == "" {
		return fmt.Errorf("governing law is required")
	}
	
	if req.JurisdictionLaw.PlaceOfJurisdiction == "" {
		return fmt.Errorf("place of jurisdiction is required")
	}
	
	return nil
}

// validateAIClientCapabilities validates AI capabilities against requested actions
func (s *RFCCompliantService) validateAIClientCapabilities(ctx context.Context, client ClientAI, actions AuthorizedActions) error {
	// Define capability matrix based on client type
	capabilities := s.getClientCapabilities(client.Type)
	
	// Validate transaction capabilities
	for _, transaction := range actions.Transactions {
		if !capabilities.canPerformTransaction(transaction) {
			return fmt.Errorf("AI client %s does not have capability for transaction: %s", client.Identity, transaction)
		}
	}
	
	// Validate decision capabilities
	for _, decision := range actions.Decisions {
		if !capabilities.canMakeDecision(decision) {
			return fmt.Errorf("AI client %s does not have capability for decision: %s", client.Identity, decision)
		}
	}
	
	// Validate physical action capabilities (robots only)
	if len(actions.PhysicalActions) > 0 && client.Type != ClientTypeRobot {
		return fmt.Errorf("only humanoid robots can perform physical actions")
	}
	
	return nil
}

// validateLegalCompliance validates legal compliance requirements
func (s *RFCCompliantService) validateLegalCompliance(ctx context.Context, jurisdictionLaw JurisdictionLaw, jurisdiction string) error {
	// Validate jurisdiction consistency
	if !strings.EqualFold(jurisdictionLaw.PlaceOfJurisdiction, jurisdiction) {
		return fmt.Errorf("jurisdiction mismatch: request %s vs PoA definition %s", jurisdiction, jurisdictionLaw.PlaceOfJurisdiction)
	}
	
	// Validate supported jurisdictions (RFC 111 requirement)
	supportedJurisdictions := map[string]bool{
		"US": true, "EU": true, "CA": true, "UK": true, "AU": true,
	}
	
	if !supportedJurisdictions[strings.ToUpper(jurisdiction)] {
		return fmt.Errorf("unsupported jurisdiction: %s", jurisdiction)
	}
	
	return nil
}

// generateGAuthCode generates RFC 111 compliant authorization code
func (s *RFCCompliantService) generateGAuthCode(ctx context.Context, req GAuthRequest) (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate secure code: %w", err)
	}
	return "gauth_" + hex.EncodeToString(bytes), nil
}

// generateExtendedToken generates RFC 111 extended token
func (s *RFCCompliantService) generateExtendedToken(ctx context.Context, req GAuthRequest, authCode string) (string, error) {
	// Create extended metadata (would be included in JWT claims in full implementation)
	_ = ExtendedMetadata{
		PowerType:        req.PowerType,
		PrincipalID:      req.PrincipalID,
		AIAgentID:        req.AIAgentID,
		PoADefinition:    req.PoADefinition,
		AttestationLevel: "rfc115_compliant",
		ComplianceProof:  []string{"poa_validated", "jurisdiction_verified", "ai_capabilities_confirmed"},
	}
	
	// Use professional JWT service with extended claims
	return s.jwtService.CreateToken(req.PrincipalID, req.Scope, time.Hour)
}

// createGAuthAuditRecord creates comprehensive audit record
func (s *RFCCompliantService) createGAuthAuditRecord(ctx context.Context, req GAuthRequest, authCode string) *AuditRecord {
	bytes := make([]byte, 16)
	var auditID string
	if _, err := rand.Read(bytes); err != nil {
		auditID = fmt.Sprintf("audit_fallback_%d", time.Now().UnixNano())
	} else {
		auditID = "audit_" + hex.EncodeToString(bytes)
	}
	
	return &AuditRecord{
		ID:                auditID,
		Timestamp:         time.Now(),
		PrincipalID:       req.PrincipalID,
		AIAgentID:         req.AIAgentID,
		PowerType:         req.PowerType,
		AuthorizationCode: authCode,
		RequestedPowers:   s.extractRequestedPowers(req.PoADefinition),
		Restrictions:      s.extractRestrictions(req.PoADefinition),
		LegalFramework:    s.extractLegalFramework(req.PoADefinition),
	}
}

// Helper methods for capability checking
type ClientCapabilities struct {
	clientType ClientType
}

func (s *RFCCompliantService) getClientCapabilities(clientType ClientType) *ClientCapabilities {
	return &ClientCapabilities{clientType: clientType}
}

func (c *ClientCapabilities) canPerformTransaction(transaction TransactionType) bool {
	// All AI types can perform basic transactions
	return true
}

func (c *ClientCapabilities) canMakeDecision(decision DecisionType) bool {
	// Define decision capability matrix
	restrictedDecisions := map[DecisionType]bool{
		DecisionLegal:     true, // Requires special authorization
		DecisionStrategic: true, // High-level decisions
	}
	
	// LLMs have limited decision capabilities
	if c.clientType == ClientTypeLLM {
		return !restrictedDecisions[decision]
	}
	
	return true
}

// Helper methods for extracting data from PoA Definition
func (s *RFCCompliantService) extractRequestedPowers(poa PoADefinition) []string {
	powers := []string{}
	
	// Extract from authorized actions
	for _, tx := range poa.ScopeDefinition.AuthorizedActions.Transactions {
		powers = append(powers, string(tx))
	}
	for _, dec := range poa.ScopeDefinition.AuthorizedActions.Decisions {
		powers = append(powers, string(dec))
	}
	
	return powers
}

func (s *RFCCompliantService) extractRestrictions(poa PoADefinition) PowerRestrictions {
	restrictions := PowerRestrictions{}
	
	// Extract power limits
	if len(poa.Requirements.PowerLimits.PowerLevels) > 0 {
		restrictions.AmountLimit = poa.Requirements.PowerLimits.PowerLevels[0].Limit
	}
	
	// Extract geographic constraints
	for _, region := range poa.ScopeDefinition.ApplicableRegions {
		restrictions.GeoRestrictions = append(restrictions.GeoRestrictions, region.Identifier)
	}
	
	return restrictions
}

func (s *RFCCompliantService) extractLegalFramework(poa PoADefinition) LegalFramework {
	return LegalFramework{
		Jurisdiction:         poa.Requirements.JurisdictionLaw.PlaceOfJurisdiction,
		EntityType:          string(poa.Principal.Type),
		CapacityVerification: true, // Always required in RFC
		RegulationFramework:  poa.Requirements.JurisdictionLaw.GoverningLaw,
		ComplianceLevel:     "rfc115_compliant",
	}
}
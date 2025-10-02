// Package rfc0111 implements the official GAuth 1.0 Authorization Framework
// as specified in GiFo-RFC-0111 by Dr. Götz G. Wehberg
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
// GiFo-Request for Comments: 0111
// Digital Supply Institute
// Category: Standards Track
// ISBN: 978-3-00-084039-5
// Status: Gimel Foundation Standards Track Document

package rfc0111

import (
	"context"
	"fmt"
	"time"
)

// GAuth 1.0 Authorization Framework - Official RFC-0111 Implementation
// This package provides the complete, type-safe implementation of the
// GAuth authorization framework as specified in the official RFC-0111 document.

// RFC0111ResourceOwner represents an entity capable of granting access to a protected resource,
// entering a legally binding transaction and accepting a decision or action.
// As per GiFo-RFC-0111 Section 3 (Nomenclature)
type RFC0111ResourceOwner struct {
	ID           string                 `json:"id"`
	Type         RFC0111ResourceOwnerType `json:"type"`
	Identity     RFC0111VerifiedIdentity `json:"identity"`
	Authorization RFC0111ResourceOwnerAuth `json:"authorization"`
	Powers       []RFC0111GrantedPower   `json:"powers"`
	Restrictions []RFC0111PowerRestriction `json:"restrictions,omitempty"`
	Metadata     map[string]interface{}  `json:"metadata,omitempty"`
}

type RFC0111ResourceOwnerType string

const (
	RFC0111ResourceOwnerTypePerson       RFC0111ResourceOwnerType = "person"       // End-user/individual
	RFC0111ResourceOwnerTypeOrganization RFC0111ResourceOwnerType = "organization" // Legal entity
)

type RFC0111ResourceOwnerAuth struct {
	StatutoryAuthority   bool                    `json:"statutory_authority"`
	RegisteredAuthority  bool                    `json:"registered_authority"`
	Authorizer          *RFC0111OwnerAuthorizer  `json:"authorizer,omitempty"`
	NotarizationLevel   RFC0111NotarizationLevel `json:"notarization_level"`
	Metadata            map[string]interface{}   `json:"metadata,omitempty"`
}

type RFC0111NotarizationLevel string

const (
	RFC0111NotarizationNone     RFC0111NotarizationLevel = "none"
	RFC0111NotarizationBasic    RFC0111NotarizationLevel = "basic"
	RFC0111NotarizationStandard RFC0111NotarizationLevel = "standard"
	RFC0111NotarizationFull     RFC0111NotarizationLevel = "full"
)

// RFC0111ResourceServer represents the server hosting protected resources
// capable of accepting and responding to protected resource requests using extended tokens.
// As per GiFo-RFC-0111 Section 3 (Nomenclature)
type RFC0111ResourceServer struct {
	ID           string                    `json:"id"`
	URL          string                    `json:"url"`
	Capabilities []RFC0111ServerCapability `json:"capabilities"`
	Owner        RFC0111ResourceOwner      `json:"owner"`
	Metadata     map[string]interface{}    `json:"metadata,omitempty"`
}

type RFC0111ServerCapability string

const (
	RFC0111ServerCapabilityTransaction RFC0111ServerCapability = "transaction"
	RFC0111ServerCapabilityValidation  RFC0111ServerCapability = "validation"
	RFC0111ServerCapabilityAudit      RFC0111ServerCapability = "audit"
	RFC0111ServerCapabilityDelegation RFC0111ServerCapability = "delegation"
)

// RFC0111Client represents an AI application making protected resource requests
// including digital agents, agentic AI, or humanoid robots
// As per GiFo-RFC-0111 Section 3 (Nomenclature)
type RFC0111Client struct {
	ID          string                   `json:"id"`
	Type        RFC0111ClientType        `json:"type"`
	Identity    RFC0111ClientIdentity    `json:"identity"`
	Owner       RFC0111ClientOwner       `json:"owner"`
	Capabilities []RFC0111ClientCapability `json:"capabilities"`
	Status      RFC0111ClientStatus      `json:"status"`
	Version     string                   `json:"version"`
	Metadata    map[string]interface{}   `json:"metadata,omitempty"`
}

type RFC0111ClientType string

const (
	RFC0111ClientTypeDigitalAgent  RFC0111ClientType = "digital_agent"  // Digital agent
	RFC0111ClientTypeAgenticAI     RFC0111ClientType = "agentic_ai"     // Team of agents
	RFC0111ClientTypeHumanoidRobot RFC0111ClientType = "humanoid_robot" // Physical manifestation
	RFC0111ClientTypeAI            RFC0111ClientType = "ai"             // General AI client
)

// RFC0111ClientOwner defines the owner of the AI system that authorizes the AI system
// to enter transactions, act and take decisions
// As per GiFo-RFC-0111 Section 3 (Nomenclature)
type RFC0111ClientOwner struct {
	ID           string                    `json:"id"`
	Identity     RFC0111VerifiedIdentity   `json:"identity"`
	Authorizer   *RFC0111OwnerAuthorizer   `json:"authorizer,omitempty"` // Power of attorney of client owner
	Powers       []RFC0111GrantedPower     `json:"powers"`
	Restrictions []RFC0111PowerRestriction `json:"restrictions,omitempty"`
	Metadata     map[string]interface{}    `json:"metadata,omitempty"`
}

// RFC0111OwnerAuthorizer is the authorizer of the client owner or resource owner
// defining the power of attorney (e.g., statutory authority)
// As per GiFo-RFC-0111 Section 3 (Nomenclature)
type RFC0111OwnerAuthorizer struct {
	ID                    string                   `json:"id"`
	Identity              RFC0111VerifiedIdentity  `json:"identity"`
	AuthorityType         RFC0111AuthorityType     `json:"authority_type"`
	RegisteredAuthority   bool                     `json:"registered_authority"`   // e.g. commercial register
	NotarizationRequired  bool                     `json:"notarization_required"`
	StatutoryAuthority    bool                     `json:"statutory_authority"`
	Metadata              map[string]interface{}   `json:"metadata,omitempty"`
}

type RFC0111AuthorityType string

const (
	RFC0111AuthorityTypeStatutory    RFC0111AuthorityType = "statutory"    // Legal/statutory authority
	RFC0111AuthorityTypeRegistered   RFC0111AuthorityType = "registered"   // Commercial register entry
	RFC0111AuthorityTypeNotarized    RFC0111AuthorityType = "notarized"    // Notarized power of attorney
	RFC0111AuthorityTypeDelegated    RFC0111AuthorityType = "delegated"    // Delegated authority
)

// RFC0111ExtendedToken represents the credential used to serve a specific request
// representing specific scopes and durations of authorization
// As per GiFo-RFC-0111 Section 3 (Nomenclature)
type RFC0111ExtendedToken struct {
	TokenID      string                       `json:"token_id"`
	ClientID     string                       `json:"client_id"`
	ResourceID   string                       `json:"resource_id"`
	Scope        RFC0111AuthorizationScope    `json:"scope"`
	Powers       []RFC0111GrantedPower        `json:"powers"`
	Restrictions []RFC0111PowerRestriction    `json:"restrictions,omitempty"`
	ValidFrom    time.Time                    `json:"valid_from"`
	ValidUntil   time.Time                    `json:"valid_until"`
	Revoked      bool                         `json:"revoked"`
	Metadata     map[string]interface{}       `json:"metadata,omitempty"`
}

// RFC0111Request represents a credentializing application to enter a transaction,
// accept a decision, or execute an action with approval of resource owner
// As per GiFo-RFC-0111 Section 3 (Nomenclature)
type RFC0111Request struct {
	RequestID    string                    `json:"request_id"`
	ClientID     string                    `json:"client_id"`
	ResourceID   string                    `json:"resource_id"`
	RequestType  RFC0111RequestType        `json:"request_type"`
	Action       RFC0111RequestedAction    `json:"action"`
	Scope        RFC0111AuthorizationScope `json:"scope"`
	Justification string                   `json:"justification,omitempty"`
	Urgency      RFC0111RequestUrgency     `json:"urgency"`
	Metadata     map[string]interface{}    `json:"metadata,omitempty"`
}

type RFC0111RequestType string

const (
	RFC0111RequestTypeTransaction RFC0111RequestType = "transaction" // Enter a transaction
	RFC0111RequestTypeDecision    RFC0111RequestType = "decision"    // Accept/make a decision  
	RFC0111RequestTypeAction      RFC0111RequestType = "action"      // Execute an action
)

type RFC0111RequestUrgency string

const (
	RFC0111RequestUrgencyLow      RFC0111RequestUrgency = "low"
	RFC0111RequestUrgencyNormal   RFC0111RequestUrgency = "normal"  
	RFC0111RequestUrgencyHigh     RFC0111RequestUrgency = "high"
	RFC0111RequestUrgencyCritical RFC0111RequestUrgency = "critical"
)

// Power*Point (P*P) Architecture Implementation
// As per GiFo-RFC-0111 Section 3 (Nomenclature)

// RFC0111PowerEnforcementPoint (PEP) - enforces authorization decisions
type RFC0111PowerEnforcementPoint struct {
	ID           string                 `json:"id"`
	Type         RFC0111PEPType         `json:"type"`
	ClientID     string                 `json:"client_id,omitempty"`
	ResourceID   string                 `json:"resource_id,omitempty"`
	Capabilities []string               `json:"capabilities"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

type RFC0111PEPType string

const (
	RFC0111PEPTypeSupplySide RFC0111PEPType = "supply_side" // Client-side enforcement
	RFC0111PEPTypeDemandSide RFC0111PEPType = "demand_side" // Resource owner/server enforcement
)

// RFC0111PowerDecisionPoint (PDP) - makes authorization decisions
type RFC0111PowerDecisionPoint struct {
	ID           string                          `json:"id"`
	Owner        RFC0111ClientOwner              `json:"owner"`        // Typically client owner
	Policies     []RFC0111AuthorizationPolicy    `json:"policies"`
	DecisionLog  []RFC0111PowerDecision          `json:"decision_log"`
	Metadata     map[string]interface{}          `json:"metadata,omitempty"`
}

// RFC0111PowerInformationPoint (PIP) - provides data for authorization decisions  
type RFC0111PowerInformationPoint struct {
	ID           string                      `json:"id"`
	DataSources  []RFC0111InformationSource  `json:"data_sources"`
	Capabilities []string                    `json:"capabilities"`
	Metadata     map[string]interface{}      `json:"metadata,omitempty"`
}

// RFC0111PowerAdministrationPoint (PAP) - manages authorization policies
type RFC0111PowerAdministrationPoint struct {
	ID           string                        `json:"id"`
	Administrator RFC0111OwnerAuthorizer       `json:"administrator"` // Typically owner's authorizer
	Policies     []RFC0111AuthorizationPolicy  `json:"policies"`
	Metadata     map[string]interface{}        `json:"metadata,omitempty"`
}

// RFC0111PowerVerificationPoint (PVP) - verifies identities
type RFC0111PowerVerificationPoint struct {
	ID                string                         `json:"id"`
	TrustServiceProvider string                      `json:"trust_service_provider"`
	VerificationMethods []RFC0111VerificationMethod  `json:"verification_methods"`
	Metadata           map[string]interface{}        `json:"metadata,omitempty"`
}

// Core Authorization Framework Types

type RFC0111VerifiedIdentity struct {
	Subject          string                       `json:"subject"`
	IdentityProvider string                       `json:"identity_provider"`
	VerificationLevel RFC0111VerificationLevel    `json:"verification_level"`
	Attributes       map[string]interface{}       `json:"attributes,omitempty"`
	VerifiedAt       time.Time                    `json:"verified_at"`
}

type RFC0111VerificationLevel string

const (
	RFC0111VerificationLevelBasic    RFC0111VerificationLevel = "basic"     // Basic identity verification
	RFC0111VerificationLevelStandard RFC0111VerificationLevel = "standard"  // Standard verification
	RFC0111VerificationLevelHigh     RFC0111VerificationLevel = "high"      // High assurance
	RFC0111VerificationLevelMaximum  RFC0111VerificationLevel = "maximum"   // Maximum assurance (LOA 4)
	RFC0111VerificationLevelGimelID  RFC0111VerificationLevel = "gimel_id"  // GimelID level 5 (ACR_LOA_5)
)

type RFC0111GrantedPower struct {
	PowerID      string                       `json:"power_id"`
	PowerType    RFC0111PowerType             `json:"power_type"`
	Scope        RFC0111AuthorizationScope    `json:"scope"`
	Restrictions []RFC0111PowerRestriction    `json:"restrictions,omitempty"`
	DelegationRules []RFC0111DelegationRule   `json:"delegation_rules,omitempty"`
	ValidFrom    time.Time                    `json:"valid_from"`
	ValidUntil   time.Time                    `json:"valid_until"`
	Revocable    bool                         `json:"revocable"`
	Metadata     map[string]interface{}       `json:"metadata,omitempty"`
}

type RFC0111PowerType string

const (
	RFC0111PowerTypeTransaction RFC0111PowerType = "transaction" // Power to enter transactions
	RFC0111PowerTypeDecision    RFC0111PowerType = "decision"    // Power to make decisions
	RFC0111PowerTypeAction      RFC0111PowerType = "action"      // Power to perform actions
	RFC0111PowerTypeSign        RFC0111PowerType = "sign"        // Signing authority
	RFC0111PowerTypeInstruct    RFC0111PowerType = "instruct"    // Authority to issue instructions
)

type RFC0111AuthorizationScope struct {
	Resources    []string                       `json:"resources,omitempty"`
	Actions      []string                       `json:"actions,omitempty"`
	Transactions []string                       `json:"transactions,omitempty"`
	Geographic   []RFC0111GeographicScope       `json:"geographic,omitempty"`
	Temporal     *RFC0111TemporalScope          `json:"temporal,omitempty"`
	Monetary     *RFC0111MonetaryScope          `json:"monetary,omitempty"`
	Custom       map[string]interface{}         `json:"custom,omitempty"`
}

type RFC0111GeographicScope struct {
	Type       string   `json:"type"`        // country, region, global
	Identifier string   `json:"identifier"`  // ISO country code, region name
	Exclusions []string `json:"exclusions,omitempty"`
}

type RFC0111TemporalScope struct {
	ValidFrom      time.Time                    `json:"valid_from"`
	ValidUntil     time.Time                    `json:"valid_until"`
	TimeZone       string                       `json:"timezone,omitempty"`
	BusinessHours  *RFC0111BusinessHours        `json:"business_hours,omitempty"`
	Exclusions     []RFC0111TimeExclusion       `json:"exclusions,omitempty"`
}

type RFC0111BusinessHours struct {
	Days  []string `json:"days"`   // monday, tuesday, etc.
	Start string   `json:"start"`  // 09:00
	End   string   `json:"end"`    // 17:00
}

type RFC0111TimeExclusion struct {
	Type        string    `json:"type"`         // holiday, maintenance, etc.
	Description string    `json:"description"`
	Start       time.Time `json:"start"`
	End         time.Time `json:"end"`
}

type RFC0111MonetaryScope struct {
	Currency    string  `json:"currency"`     // ISO currency code
	MaxAmount   float64 `json:"max_amount"`   // Maximum transaction amount
	DailyLimit  float64 `json:"daily_limit,omitempty"`
	MonthlyLimit float64 `json:"monthly_limit,omitempty"`
}

type RFC0111PowerRestriction struct {
	RestrictionID   string                    `json:"restriction_id"`
	Type            RFC0111RestrictionType    `json:"type"`
	Description     string                    `json:"description"`
	Scope           RFC0111AuthorizationScope `json:"scope,omitempty"`
	Conditions      []string                  `json:"conditions,omitempty"`
	RequiredApproval bool                     `json:"required_approval,omitempty"`
	Metadata        map[string]interface{}    `json:"metadata,omitempty"`
}

type RFC0111RestrictionType string

const (
	RFC0111RestrictionTypeValueLimit     RFC0111RestrictionType = "value_limit"     // Monetary limits
	RFC0111RestrictionTypeGeographic     RFC0111RestrictionType = "geographic"      // Geographic constraints  
	RFC0111RestrictionTypeTemporal       RFC0111RestrictionType = "temporal"        // Time-based restrictions
	RFC0111RestrictionTypeApproval       RFC0111RestrictionType = "approval"        // Requires additional approval
	RFC0111RestrictionTypeExclusion      RFC0111RestrictionType = "exclusion"       // Explicit exclusions
	RFC0111RestrictionTypeNotarization   RFC0111RestrictionType = "notarization"    // Requires notarization
)

// Additional supporting types...

type RFC0111ClientIdentity struct {
	AgentID          string                         `json:"agent_id"`
	SystemVersion    string                         `json:"system_version"`
	Capabilities     []string                       `json:"capabilities"`
	TrustLevel       RFC0111TrustLevel              `json:"trust_level"`
	CertificationLevel RFC0111CertificationLevel    `json:"certification_level"`
	Metadata         map[string]interface{}         `json:"metadata,omitempty"`
}

type RFC0111TrustLevel string

const (
	RFC0111TrustLevelUntrusted RFC0111TrustLevel = "untrusted"
	RFC0111TrustLevelBasic     RFC0111TrustLevel = "basic"
	RFC0111TrustLevelStandard  RFC0111TrustLevel = "standard"
	RFC0111TrustLevelHigh      RFC0111TrustLevel = "high"
	RFC0111TrustLevelCertified RFC0111TrustLevel = "certified"
)

type RFC0111CertificationLevel string

const (
	RFC0111CertificationNone       RFC0111CertificationLevel = "none"
	RFC0111CertificationBasic      RFC0111CertificationLevel = "basic"
	RFC0111CertificationStandard   RFC0111CertificationLevel = "standard"
	RFC0111CertificationAdvanced   RFC0111CertificationLevel = "advanced"
	RFC0111CertificationGimelPlus  RFC0111CertificationLevel = "gimel_plus" // GAuth+ exclusive
)

type RFC0111ClientStatus string

const (
	RFC0111ClientStatusActive     RFC0111ClientStatus = "active"
	RFC0111ClientStatusSuspended  RFC0111ClientStatus = "suspended"
	RFC0111ClientStatusRevoked    RFC0111ClientStatus = "revoked"
	RFC0111ClientStatusPending    RFC0111ClientStatus = "pending"
)

type RFC0111ClientCapability string

const (
	RFC0111CapabilityTransaction RFC0111ClientCapability = "transaction"
	RFC0111CapabilityDecision    RFC0111ClientCapability = "decision"  
	RFC0111CapabilityAction      RFC0111ClientCapability = "action"
	RFC0111CapabilitySigning     RFC0111ClientCapability = "signing"
	RFC0111CapabilityDelegation  RFC0111ClientCapability = "delegation"
)

type RFC0111RequestedAction struct {
	Type         string                 `json:"type"`
	Description  string                 `json:"description"`
	Parameters   map[string]interface{} `json:"parameters,omitempty"`
	ExpectedOutcome string              `json:"expected_outcome,omitempty"`
}

type RFC0111AuthorizationPolicy struct {
	PolicyID    string                  `json:"policy_id"`
	Name        string                  `json:"name"`
	Version     string                  `json:"version"`
	Rules       []RFC0111PolicyRule     `json:"rules"`
	Priority    int                     `json:"priority"`
	Enabled     bool                    `json:"enabled"`
	ValidFrom   time.Time               `json:"valid_from"`
	ValidUntil  time.Time               `json:"valid_until"`
	Metadata    map[string]interface{}  `json:"metadata,omitempty"`
}

type RFC0111PolicyRule struct {
	RuleID      string                  `json:"rule_id"`
	Condition   string                  `json:"condition"`   // Boolean expression
	Effect      RFC0111PolicyEffect     `json:"effect"`      // Allow, Deny, Require
	Actions     []string                `json:"actions,omitempty"`
	Metadata    map[string]interface{}  `json:"metadata,omitempty"`
}

type RFC0111PolicyEffect string

const (
	RFC0111PolicyEffectAllow   RFC0111PolicyEffect = "allow"
	RFC0111PolicyEffectDeny    RFC0111PolicyEffect = "deny"
	RFC0111PolicyEffectRequire RFC0111PolicyEffect = "require" // Require additional conditions
)

type RFC0111PowerDecision struct {
	DecisionID   string                     `json:"decision_id"`
	RequestID    string                     `json:"request_id"`
	Decision     RFC0111DecisionResult      `json:"decision"`
	Reasoning    string                     `json:"reasoning"`
	DecidedBy    string                     `json:"decided_by"`
	DecidedAt    time.Time                  `json:"decided_at"`
	Conditions   []RFC0111GrantCondition    `json:"conditions,omitempty"`
	Metadata     map[string]interface{}     `json:"metadata,omitempty"`
}

type RFC0111DecisionResult string

const (
	RFC0111DecisionAllow    RFC0111DecisionResult = "allow"
	RFC0111DecisionDeny     RFC0111DecisionResult = "deny"
	RFC0111DecisionConditional RFC0111DecisionResult = "conditional"
)

type RFC0111GrantCondition struct {
	ConditionID   string                    `json:"condition_id"`
	Type          RFC0111ConditionType      `json:"type"`
	Description   string                    `json:"description"`
	Required      bool                      `json:"required"`
	Parameters    map[string]interface{}    `json:"parameters,omitempty"`
}

type RFC0111ConditionType string

const (
	RFC0111ConditionTypeApproval      RFC0111ConditionType = "approval"      // Requires human approval
	RFC0111ConditionTypeNotification  RFC0111ConditionType = "notification"  // Requires notification
	RFC0111ConditionTypeAudit         RFC0111ConditionType = "audit"         // Requires audit logging
	RFC0111ConditionTypeVerification  RFC0111ConditionType = "verification"  // Requires additional verification
)

type RFC0111InformationSource struct {
	SourceID     string                   `json:"source_id"`
	Type         RFC0111SourceType        `json:"type"`
	URL          string                   `json:"url,omitempty"`
	Capabilities []string                 `json:"capabilities"`
	TrustLevel   RFC0111TrustLevel        `json:"trust_level"`
	Metadata     map[string]interface{}   `json:"metadata,omitempty"`
}

type RFC0111SourceType string

const (
	RFC0111SourceTypeIdentityProvider  RFC0111SourceType = "identity_provider"
	RFC0111SourceTypeCommercialRegister RFC0111SourceType = "commercial_register"
	RFC0111SourceTypeNotaryService     RFC0111SourceType = "notary_service"
	RFC0111SourceTypeTrustService      RFC0111SourceType = "trust_service"
	RFC0111SourceTypeAuditLog          RFC0111SourceType = "audit_log"
)

type RFC0111VerificationMethod struct {
	MethodID     string                        `json:"method_id"`
	Type         RFC0111VerificationType       `json:"type"`
	Description  string                        `json:"description"`
	TrustLevel   RFC0111TrustLevel             `json:"trust_level"`
	Required     bool                          `json:"required"`
	Metadata     map[string]interface{}        `json:"metadata,omitempty"`
}

type RFC0111VerificationType string

const (
	RFC0111VerificationTypePassword    RFC0111VerificationType = "password"
	RFC0111VerificationTypeBiometric   RFC0111VerificationType = "biometric"
	RFC0111VerificationTypeCertificate RFC0111VerificationType = "certificate"
	RFC0111VerificationTypeNotary      RFC0111VerificationType = "notary"
	RFC0111VerificationTypeGimelID     RFC0111VerificationType = "gimel_id" // DNA-based (GAuth+ exclusive)
)

type RFC0111DelegationRule struct {
	RuleID       string                       `json:"rule_id"`
	Type         RFC0111DelegationType        `json:"type"`
	MaxDepth     int                          `json:"max_depth"`     // Maximum delegation depth
	AllowedTypes []RFC0111ClientType          `json:"allowed_types"` // Which client types can be delegated to
	Restrictions []RFC0111PowerRestriction    `json:"restrictions,omitempty"`
	RequiresApproval bool                     `json:"requires_approval"`
	Metadata     map[string]interface{}       `json:"metadata,omitempty"`
}

type RFC0111DelegationType string

const (
	RFC0111DelegationTypeNone     RFC0111DelegationType = "none"     // No delegation allowed
	RFC0111DelegationTypeLimited  RFC0111DelegationType = "limited"  // Limited delegation
	RFC0111DelegationTypeFull     RFC0111DelegationType = "full"     // Full delegation allowed
	RFC0111DelegationTypeCascade  RFC0111DelegationType = "cascade"  // Cascade delegation (GAuth+ exclusive)
)

// Main GAuth RFC-0111 Service Interface
// Implements the complete GiFo-RFC-0111 specification

type RFC0111Service interface {
	// Core Authorization Flow (RFC-0111 Section 6)
	RequestAuthorization(ctx context.Context, req *RFC0111Request) (*RFC0111AuthorizationGrant, error)
	ValidateGrant(ctx context.Context, grant *RFC0111AuthorizationGrant) error
	IssueExtendedToken(ctx context.Context, grant *RFC0111AuthorizationGrant) (*RFC0111ExtendedToken, error)
	ValidateExtendedToken(ctx context.Context, token *RFC0111ExtendedToken) error
	
	// Power Management
	GrantPower(ctx context.Context, owner *RFC0111ResourceOwner, client *RFC0111Client, power *RFC0111GrantedPower) error
	RevokePower(ctx context.Context, powerID string) error
	ValidatePowerCompliance(ctx context.Context, clientID string, action *RFC0111RequestedAction) error
	
	// P*P Architecture Components
	GetPowerDecisionPoint(ctx context.Context, clientID string) (*RFC0111PowerDecisionPoint, error)
	GetPowerInformationPoint(ctx context.Context) (*RFC0111PowerInformationPoint, error)
	GetPowerVerificationPoint(ctx context.Context) (*RFC0111PowerVerificationPoint, error)
	
	// Identity and Verification
	VerifyIdentity(ctx context.Context, identity *RFC0111VerifiedIdentity) error
	RegisterClient(ctx context.Context, client *RFC0111Client, owner *RFC0111ClientOwner) error
	RegisterResourceOwner(ctx context.Context, owner *RFC0111ResourceOwner, authorizer *RFC0111OwnerAuthorizer) error
	
	// Compliance and Audit
	LogDecision(ctx context.Context, decision *RFC0111PowerDecision) error
	TrackCompliance(ctx context.Context, clientID string, action *RFC0111RequestedAction) error
	GetAuditTrail(ctx context.Context, clientID string, from, to time.Time) ([]RFC0111PowerDecision, error)
}

// Configuration for RFC-0111 GAuth Service
type RFC0111Config struct {
	// Core service configuration
	AuthorizationServerURL string
	TrustServiceProvider   string
	
	// P*P Architecture configuration
	PowerDecisionPoint      *RFC0111PowerDecisionPoint
	PowerInformationPoint   *RFC0111PowerInformationPoint
	PowerVerificationPoint  *RFC0111PowerVerificationPoint
	
	// Security configuration
	RequireNotarization     bool
	MaxDelegationDepth      int
	DefaultTokenValidity    time.Duration
	
	// Compliance configuration
	AuditingEnabled         bool
	ComplianceTrackingEnabled bool
	
	// RFC-0111 Exclusions (Section 2) - MUST be true for open source
	ExcludeWeb3             bool // Must be true for open source
	ExcludeAIOperators      bool // Must be true for open source  
	ExcludeDNAIdentities    bool // Must be true for open source
	
	// Storage and external services
	TokenStore     RFC0111TokenStore
	AuditLogger    RFC0111AuditLogger  
	EventPublisher RFC0111EventPublisher
}

// Supporting interfaces for RFC-0111
type RFC0111TokenStore interface {
	StoreToken(ctx context.Context, token *RFC0111ExtendedToken) error
	GetToken(ctx context.Context, tokenID string) (*RFC0111ExtendedToken, error)
	RevokeToken(ctx context.Context, tokenID string) error
	ListTokens(ctx context.Context, clientID string) ([]*RFC0111ExtendedToken, error)
}

type RFC0111AuditLogger interface {
	LogDecision(ctx context.Context, decision *RFC0111PowerDecision) error
	LogAction(ctx context.Context, clientID, action, outcome string) error
	GetAuditTrail(ctx context.Context, clientID string, from, to time.Time) ([]RFC0111AuditEntry, error)
}

type RFC0111EventPublisher interface {
	PublishAuthorizationEvent(ctx context.Context, event *RFC0111AuthorizationEvent) error
	PublishComplianceEvent(ctx context.Context, event *RFC0111ComplianceEvent) error
}

type RFC0111AuditEntry struct {
	EntryID   string                 `json:"entry_id"`
	Timestamp time.Time              `json:"timestamp"`
	ClientID  string                 `json:"client_id"`
	Action    string                 `json:"action"`
	Outcome   string                 `json:"outcome"`
	Details   map[string]interface{} `json:"details,omitempty"`
}

type RFC0111AuthorizationEvent struct {
	EventID   string                 `json:"event_id"`
	Type      string                 `json:"type"`
	ClientID  string                 `json:"client_id"`
	Details   map[string]interface{} `json:"details"`
	Timestamp time.Time              `json:"timestamp"`
}

type RFC0111ComplianceEvent struct {
	EventID     string                 `json:"event_id"`
	ClientID    string                 `json:"client_id"`
	Violation   string                 `json:"violation,omitempty"`
	Action      string                 `json:"action"`
	Compliant   bool                   `json:"compliant"`
	Details     map[string]interface{} `json:"details"`
	Timestamp   time.Time              `json:"timestamp"`
}

// RFC0111AuthorizationGrant represents the resource owner's authorization
// used by the client to obtain an extended token
// As per GiFo-RFC-0111 Section 3 (Nomenclature)
type RFC0111AuthorizationGrant struct {
	GrantID      string                    `json:"grant_id"`
	ClientID     string                    `json:"client_id"`
	ResourceOwnerID string                 `json:"resource_owner_id"`
	GrantType    RFC0111GrantType          `json:"grant_type"`
	Scope        RFC0111AuthorizationScope `json:"scope"`
	Powers       []RFC0111GrantedPower     `json:"powers"`
	Conditions   []RFC0111GrantCondition   `json:"conditions,omitempty"`
	ValidFrom    time.Time                 `json:"valid_from"`
	ValidUntil   time.Time                 `json:"valid_until"`
	Metadata     map[string]interface{}    `json:"metadata,omitempty"`
}

type RFC0111GrantType string

const (
	RFC0111GrantTypeAuthorizationCode RFC0111GrantType = "authorization_code"
	RFC0111GrantTypeImplicit         RFC0111GrantType = "implicit"
	RFC0111GrantTypeClientCredentials RFC0111GrantType = "client_credentials"
	RFC0111GrantTypeResourceOwner     RFC0111GrantType = "resource_owner"
)

// GAuth RFC-0111 Errors
type RFC0111GAuthError struct {
	Code    string                 `json:"code"`
	Message string                 `json:"message"`
	Details map[string]interface{} `json:"details,omitempty"`
}

func (e *RFC0111GAuthError) Error() string {
	return fmt.Sprintf("[RFC-0111:%s] %s", e.Code, e.Message)
}

// Error codes as per GiFo-RFC-0111
const (
	RFC0111ErrorCodeUnauthorized         = "unauthorized"
	RFC0111ErrorCodeInvalidGrant         = "invalid_grant"
	RFC0111ErrorCodeInvalidToken         = "invalid_token"
	RFC0111ErrorCodeInsufficientPowers   = "insufficient_powers"
	RFC0111ErrorCodePowerRevoked         = "power_revoked"
	RFC0111ErrorCodeComplianceViolation  = "compliance_violation"
	RFC0111ErrorCodeIdentityVerificationFailed = "identity_verification_failed"
	RFC0111ErrorCodeExclusionViolation   = "exclusion_violation" // RFC-0111 Section 2
)

// RFC-0111 Abstract Protocol Flow Implementation
// As per GiFo-RFC-0111 Section 6 (How GAuth works)

// CreateRFC0111Service creates a new RFC-0111 compliant GAuth service
func CreateRFC0111Service(config *RFC0111Config) (RFC0111Service, error) {
	// Validate RFC-0111 compliance
	if !config.ExcludeWeb3 || !config.ExcludeAIOperators || !config.ExcludeDNAIdentities {
		return nil, &RFC0111GAuthError{
			Code:    RFC0111ErrorCodeExclusionViolation,
			Message: "RFC-0111 Section 2: Web3, AI operators, and DNA identities must be excluded in open source implementation",
		}
	}
	
	// Implementation would be provided by the concrete service
	return nil, fmt.Errorf("concrete implementation required")
}

// Validation functions for RFC-0111 compliance

// ValidateRFC0111Compliance validates that the implementation follows RFC-0111 requirements
func ValidateRFC0111Compliance(config *RFC0111Config) error {
	// Check mandatory exclusions (RFC-0111 Section 2)
	if !config.ExcludeWeb3 {
		return &RFC0111GAuthError{
			Code:    RFC0111ErrorCodeExclusionViolation,
			Message: "RFC-0111: Web3/blockchain technology must be excluded from open source implementation",
		}
	}
	
	if !config.ExcludeAIOperators {
		return &RFC0111GAuthError{
			Code:    RFC0111ErrorCodeExclusionViolation,
			Message: "RFC-0111: AI operators controlling entire process must be excluded from open source implementation",
		}
	}
	
	if !config.ExcludeDNAIdentities {
		return &RFC0111GAuthError{
			Code:    RFC0111ErrorCodeExclusionViolation,
			Message: "RFC-0111: DNA-based identities must be excluded from open source implementation",
		}
	}
	
	return nil
}
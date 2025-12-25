// Package gauth - Extended Token Implementation per RFC-0111 Section 3
package gauth

import (
	"encoding/json"
	"time"

	"github.com/mauriciomferz/Gauth_go/pkg/poa"
)

// ExtendedToken represents the RFC-0111 comprehensive authorization credential
// RFC Requirement (Section 3, Page 6):
// "Extended tokens represent specific scopes and durations of authorization,
// granted by the resource owner, and enforced by the resource server and authorization server.
// As a digital representation in terms of set of data or any other form of representation
// an extended token summarizes the authorization for a specific request, potentially including
// access rights but beyond and more comprehensive."
type ExtendedToken struct {
	// OAuth 2.0 Compatibility Fields
	AccessToken  string    `json:"access_token"`
	TokenType    string    `json:"token_type"`
	ExpiresIn    int64     `json:"expires_in"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	Scope        []string  `json:"scope"`
	IssuedAt     time.Time `json:"issued_at"`

	// RFC-0111 Extended Token Fields (Comprehensive Authorization)
	PowerOfAttorney    *poa.PoADefinition         `json:"power_of_attorney"`
	AuthorizationChain *AuthorizationChain        `json:"authorization_chain"`
	ClientOwner        *ClientOwnerInfo           `json:"client_owner"`
	OwnersAuthorizer   *OwnersAuthorizerInfo      `json:"owners_authorizer"`
	ResourceOwner      *ResourceOwnerInfo         `json:"resource_owner,omitempty"`
	LegalFramework     *LegalFrameworkInfo        `json:"legal_framework"`
	Restrictions       []PowerRestriction         `json:"restrictions,omitempty"`
	IssuedBy           *AuthorizationServerInfo   `json:"issued_by"`
	VerificationProof  *IdentityVerificationChain `json:"verification_proof"`

	// Request Context
	RequestID          string                 `json:"request_id"`
	GrantID            string                 `json:"grant_id"`
	TransactionContext map[string]interface{} `json:"transaction_context,omitempty"`

	// Subscription Tracking (RFC-0111 Steps I-VIII)
	SubscriptionID string `json:"subscription_id,omitempty"` // Links token to its originating subscription

	// Compliance & Audit
	ComplianceLevel     string               `json:"compliance_level"`
	AuditTrail          []AuditEntry         `json:"audit_trail,omitempty"`
	JurisdictionContext *JurisdictionContext `json:"jurisdiction_context"`
	// RFC 9396 Rich Authorization Requests
	AuthorizationDetails []AuthorizationDetail `json:"authorization_details,omitempty"`
}

// AuthorizationChain represents the complete authorization hierarchy
// RFC Requirement: Must trace from Owner's Authorizer → Client Owner → Client
type AuthorizationChain struct {
	// Chain levels (ordered from root to leaf)
	OwnersAuthorizer *AuthorizationLink `json:"owners_authorizer"` // Level 1: Board/Managing Director
	ClientOwner      *AuthorizationLink `json:"client_owner"`      // Level 2: AI System Owner
	Client           *AuthorizationLink `json:"client"`            // Level 3: AI Client/Agent

	// Chain validation
	ChainValidated bool      `json:"chain_validated"`
	ValidationTime time.Time `json:"validation_time"`
	ValidatorID    string    `json:"validator_id"`

	// Chain metadata
	ChainDepth     int    `json:"chain_depth"`
	ChainIntegrity string `json:"chain_integrity"` // cryptographic hash of chain
}

// AuthorizationLink represents a single link in the authorization chain
type AuthorizationLink struct {
	EntityID     string `json:"entity_id"`
	DelegationID string `json:"delegation_id,omitempty"`
	EntityType   string `json:"entity_type"` // "natural_person", "organization", "ai_system"
	EntityName   string `json:"entity_name"`
	Role         string `json:"role"` // "authorizer", "owner", "client"

	// Authorization details
	AuthorizedBy          string    `json:"authorized_by,omitempty"` // Entity ID of parent authorizer
	AuthorizationDate     time.Time `json:"authorization_date"`
	AuthorizationType     string    `json:"authorization_type"` // "statutory", "contractual", "delegated"
	AuthorizationDocument string    `json:"authorization_document,omitempty"`

	// Legal basis
	LegalBasis            *LegalBasis `json:"legal_basis"`
	StatutoryAuthority    string      `json:"statutory_authority,omitempty"`
	CommercialRegisterRef string      `json:"commercial_register_ref,omitempty"`

	// Identity verification
	IdentityVerified   bool   `json:"identity_verified"`
	VerificationMethod string `json:"verification_method,omitempty"`
	VerificationProof  string `json:"verification_proof,omitempty"`

	// Scope of authority
	ScopeOfAuthority []string `json:"scope_of_authority"`
	Limitations      []string `json:"limitations,omitempty"`

	// Validity
	ValidFrom  time.Time `json:"valid_from"`
	ValidUntil time.Time `json:"valid_until"`
	Revocable  bool      `json:"revocable"`
	Status     string    `json:"status"` // "active", "suspended", "revoked"
}

// LegalBasis represents the legal foundation for authorization
type LegalBasis struct {
	BasisType          string   `json:"basis_type"`                 // "company_law", "power_of_attorney", "statutory", "contractual"
	Jurisdiction       string   `json:"jurisdiction"`               // ISO 3166-1 alpha-2
	LegalReferences    []string `json:"legal_references,omitempty"` // Law articles, contract clauses
	RegistrationNumber string   `json:"registration_number,omitempty"`
	IssuingAuthority   string   `json:"issuing_authority,omitempty"`
}

// ClientOwnerInfo represents the AI system owner
type ClientOwnerInfo struct {
	OwnerID   string `json:"owner_id"`
	OwnerName string `json:"owner_name"`
	OwnerType string `json:"owner_type"` // "individual", "organization"

	// Organization details (if applicable)
	OrganizationLegalName    string `json:"organization_legal_name,omitempty"`
	OrganizationRegistration string `json:"organization_registration,omitempty"`
	JurisdictionOfIncorp     string `json:"jurisdiction_of_incorporation,omitempty"`

	// Authorization status
	RegisteredPowerOfAttorney bool   `json:"registered_power_of_attorney"`
	CommercialRegisterEntry   bool   `json:"commercial_register_entry"`
	CommercialRegisterID      string `json:"commercial_register_id,omitempty"`

	// Verification
	IdentityVerified   bool      `json:"identity_verified"`
	VerificationDate   time.Time `json:"verification_date,omitempty"`
	VerificationMethod string    `json:"verification_method,omitempty"`
}

// OwnersAuthorizerInfo represents the entity that authorizes the client owner
// RFC-0111 Section 3: "The 'owner's authorizer' is the authorizer of the client owner
// or resource owner, respectively, and defines the power of attorney of the client owner
// or resource owner, e.g. its statutory authority."
type OwnersAuthorizerInfo struct {
	AuthorizerID   string `json:"authorizer_id"`
	AuthorizerName string `json:"authorizer_name"`
	AuthorizerType string `json:"authorizer_type"` // "managing_director", "board_member", "legal_representative"

	// Statutory authority
	StatutoryAuthority string   `json:"statutory_authority"`
	AuthorityType      string   `json:"authority_type"` // "individual", "joint", "collective"
	AuthorityScope     []string `json:"authority_scope"`

	// Commercial register proof
	CommercialRegisterEntry bool   `json:"commercial_register_entry"`
	CommercialRegisterID    string `json:"commercial_register_id,omitempty"`
	RegisterJurisdiction    string `json:"register_jurisdiction,omitempty"`
	RegisterVerificationRef string `json:"register_verification_ref,omitempty"`

	// Power of attorney documentation
	PowerOfAttorneyType     string    `json:"power_of_attorney_type,omitempty"` // "general", "special", "prokura"
	PowerOfAttorneyDocument string    `json:"power_of_attorney_document,omitempty"`
	PowerOfAttorneyDate     time.Time `json:"power_of_attorney_date,omitempty"`

	// Identity verification
	IdentityVerified   bool      `json:"identity_verified"`
	VerificationMethod string    `json:"verification_method,omitempty"`
	VerificationDate   time.Time `json:"verification_date,omitempty"`
	VerificationProof  string    `json:"verification_proof,omitempty"`

	// Relationship to client owner
	RelationshipToOwner string `json:"relationship_to_owner"`
	AuthorizationBasis  string `json:"authorization_basis"`
}

// ResourceOwnerInfo represents the resource owner (when applicable)
type ResourceOwnerInfo struct {
	OwnerID          string    `json:"owner_id"`
	OwnerName        string    `json:"owner_name"`
	OwnerType        string    `json:"owner_type"`
	Jurisdiction     string    `json:"jurisdiction,omitempty"`
	IdentityVerified bool      `json:"identity_verified"`
	VerificationDate time.Time `json:"verification_date,omitempty"`
}

// LegalFrameworkInfo represents the legal compliance context
type LegalFrameworkInfo struct {
	ApplicableLaws       []string        `json:"applicable_laws"`
	Jurisdiction         string          `json:"jurisdiction"`
	ComplianceFramework  string          `json:"compliance_framework,omitempty"`
	RegulatoryReferences []string        `json:"regulatory_references,omitempty"`
	FiduciaryDuties      []FiduciaryDuty `json:"fiduciary_duties,omitempty"`
	LegalOpinion         string          `json:"legal_opinion,omitempty"`
}

// FiduciaryDuty represents a fiduciary obligation
type FiduciaryDuty struct {
	DutyType        string   `json:"duty_type"` // "loyalty", "care", "good_faith", "disclosure"
	Description     string   `json:"description"`
	Scope           []string `json:"scope"`
	ValidationRules []string `json:"validation_rules,omitempty"`
}

// PowerRestriction represents a constraint on the granted powers
type PowerRestriction struct {
	RestrictionType  string      `json:"restriction_type"` // "value_limit", "scope_limit", "time_limit", "geographic_limit"
	Description      string      `json:"description"`
	Value            interface{} `json:"value,omitempty"`
	Scope            []string    `json:"scope,omitempty"`
	EnforcementLevel string      `json:"enforcement_level"` // "mandatory", "advisory"
}

// AuthorizationServerInfo represents the issuing authorization server
type AuthorizationServerInfo struct {
	ServerID         string    `json:"server_id"`
	ServerURL        string    `json:"server_url"`
	ServerName       string    `json:"server_name,omitempty"`
	Issuer           string    `json:"issuer"` // OAuth issuer
	PublicKeyID      string    `json:"public_key_id,omitempty"`
	CertificateChain []string  `json:"certificate_chain,omitempty"`
	IssueTime        time.Time `json:"issue_time"`
}

// IdentityVerificationChain represents the PVP verification chain
// RFC-0111 Section 3, Page 8: "Power Verification Point (PVP) – verification of the
// identities that perform a specific role along the GAuth processing."
type IdentityVerificationChain struct {
	ChainID              string                    `json:"chain_id"`
	VerificationLevels   []VerificationLevel       `json:"verification_levels"`
	OverallVerification  string                    `json:"overall_verification"` // "verified", "partial", "unverified"
	VerificationTime     time.Time                 `json:"verification_time"`
	VerifierEntity       string                    `json:"verifier_entity"`
	TrustServiceProvider *TrustServiceProviderInfo `json:"trust_service_provider,omitempty"`
	CryptographicProof   string                    `json:"cryptographic_proof,omitempty"`
}

// VerificationLevel represents identity verification at each authorization level
type VerificationLevel struct {
	Level              int       `json:"level"` // 1=Authorizer, 2=Owner, 3=Client
	EntityID           string    `json:"entity_id"`
	EntityRole         string    `json:"entity_role"`
	VerificationMethod string    `json:"verification_method"` // "eIDAS", "government_id", "commercial_register", "certificate"
	VerificationStatus string    `json:"verification_status"` // "verified", "pending", "failed"
	VerifiedBy         string    `json:"verified_by,omitempty"`
	VerificationDate   time.Time `json:"verification_date"`
	VerificationProof  string    `json:"verification_proof,omitempty"`
	AssuranceLevel     string    `json:"assurance_level,omitempty"` // "substantial", "high", "low"
}

// TrustServiceProviderInfo represents the trust service provider
type TrustServiceProviderInfo struct {
	ProviderID       string   `json:"provider_id"`
	ProviderName     string   `json:"provider_name"`
	ProviderType     string   `json:"provider_type"` // "qualified", "non-qualified"
	Jurisdiction     string   `json:"jurisdiction"`
	AccreditationRef string   `json:"accreditation_ref,omitempty"`
	ServiceTypes     []string `json:"service_types"` // "identity_verification", "signature", "timestamp"
}

// AuditEntry represents an audit log entry
type AuditEntry struct {
	Timestamp time.Time              `json:"timestamp"`
	Action    string                 `json:"action"`
	Actor     string                 `json:"actor"`
	Result    string                 `json:"result"`
	Details   map[string]interface{} `json:"details,omitempty"`
}

// JurisdictionContext represents the legal jurisdiction context
type JurisdictionContext struct {
	PrimaryJurisdiction    string   `json:"primary_jurisdiction"`
	SecondaryJurisdictions []string `json:"secondary_jurisdictions,omitempty"`
	ApplicableLaws         []string `json:"applicable_laws"`
	ConflictOfLawsRule     string   `json:"conflict_of_laws_rule,omitempty"`
}

// ExtendedTokenResponse extends TokenResponse with RFC-0111 comprehensive data
type ExtendedTokenResponse struct {
	*ExtendedToken
}

// ValidateExtendedToken performs comprehensive validation of extended token
func (et *ExtendedToken) Validate() error {
	if et.AccessToken == "" {
		return ErrInvalidToken
	}

	// Validate authorization chain
	if et.AuthorizationChain == nil {
		return &GAuthError{Code: "missing_authorization_chain", Message: "Extended token must include authorization chain"}
	}

	if err := et.AuthorizationChain.Validate(); err != nil {
		return err
	}

	// Validate client owner
	if et.ClientOwner == nil {
		return &GAuthError{Code: "missing_client_owner", Message: "Extended token must include client owner"}
	}

	// Validate owner's authorizer
	if et.OwnersAuthorizer == nil {
		return &GAuthError{Code: "missing_owners_authorizer", Message: "Extended token must include owner's authorizer"}
	}

	// Validate verification chain
	if et.VerificationProof == nil {
		return &GAuthError{Code: "missing_verification_proof", Message: "Extended token must include identity verification proof"}
	}

	// Validate legal framework
	if et.LegalFramework == nil {
		return &GAuthError{Code: "missing_legal_framework", Message: "Extended token must include legal framework"}
	}

	// Validate issuer
	if et.IssuedBy == nil {
		return &GAuthError{Code: "missing_issuer", Message: "Extended token must include authorization server info"}
	}

	return nil
}

// Validate performs validation of authorization chain
func (ac *AuthorizationChain) Validate() error {
	if ac.OwnersAuthorizer == nil {
		return &GAuthError{Code: "invalid_chain", Message: "Authorization chain must start with owner's authorizer"}
	}

	if ac.ClientOwner == nil {
		return &GAuthError{Code: "invalid_chain", Message: "Authorization chain must include client owner"}
	}

	if ac.Client == nil {
		return &GAuthError{Code: "invalid_chain", Message: "Authorization chain must end with client"}
	}

	// Validate chain linkage
	if ac.ClientOwner.AuthorizedBy != ac.OwnersAuthorizer.EntityID {
		return &GAuthError{Code: "broken_chain", Message: "Client owner must be authorized by owner's authorizer"}
	}

	if ac.Client.AuthorizedBy != ac.ClientOwner.EntityID {
		return &GAuthError{Code: "broken_chain", Message: "Client must be authorized by client owner"}
	}

	// Validate all links are active
	if err := ac.OwnersAuthorizer.Validate(); err != nil {
		return err
	}
	if err := ac.ClientOwner.Validate(); err != nil {
		return err
	}
	if err := ac.Client.Validate(); err != nil {
		return err
	}

	return nil
}

// Validate performs validation of authorization link
func (al *AuthorizationLink) Validate() error {
	if al.EntityID == "" {
		return &GAuthError{Code: "invalid_link", Message: "Authorization link must have entity ID"}
	}

	if al.Status != "active" {
		return &GAuthError{Code: "inactive_link", Message: "Authorization link must be active"}
	}

	if time.Now().After(al.ValidUntil) {
		return &GAuthError{Code: "expired_link", Message: "Authorization link has expired"}
	}

	if !al.IdentityVerified {
		return &GAuthError{Code: "unverified_identity", Message: "Authorization link identity must be verified"}
	}

	return nil
}

// ToJSON serializes extended token to JSON
func (et *ExtendedToken) ToJSON() ([]byte, error) {
	return json.Marshal(et)
}

// FromJSON deserializes extended token from JSON
func (et *ExtendedToken) FromJSON(data []byte) error {
	return json.Unmarshal(data, et)
}

// GetAuthorizationChainDepth returns the depth of the authorization chain
func (et *ExtendedToken) GetAuthorizationChainDepth() int {
	if et.AuthorizationChain == nil {
		return 0
	}
	return et.AuthorizationChain.ChainDepth
}

// IsFullyVerified checks if all identities in chain are verified
func (et *ExtendedToken) IsFullyVerified() bool {
	if et.VerificationProof == nil {
		return false
	}
	return et.VerificationProof.OverallVerification == "verified"
}

// HasCommercialRegisterProof checks if owner's authorizer has commercial register proof
func (et *ExtendedToken) HasCommercialRegisterProof() bool {
	if et.OwnersAuthorizer == nil {
		return false
	}
	return et.OwnersAuthorizer.CommercialRegisterEntry && et.OwnersAuthorizer.CommercialRegisterID != ""
}

// HasMCPScope checks if token has specific MCP scope
// MCP scopes follow format: mcp:resource:read, mcp:tool:call, mcp:prompt:get, etc.
// Supports wildcard scopes like mcp:* or mcp:resource:*
func (et *ExtendedToken) HasMCPScope(requiredScope string) bool {
	for _, scope := range et.Scope {
		if scope == requiredScope {
			return true
		}
		// Check wildcard scopes
		if len(scope) > 0 && scope[len(scope)-1] == '*' {
			prefix := scope[:len(scope)-1]
			if len(requiredScope) >= len(prefix) && requiredScope[:len(prefix)] == prefix {
				return true
			}
		}
	}
	return false
}

// GetMCPScopes returns all MCP-related scopes from token
// Returns empty slice if no MCP scopes present
func (et *ExtendedToken) GetMCPScopes() []string {
	mcpScopes := make([]string, 0)
	for _, scope := range et.Scope {
		if len(scope) >= 4 && scope[:4] == "mcp:" {
			mcpScopes = append(mcpScopes, scope)
		}
	}
	return mcpScopes
}

// AddMCPScope adds an MCP scope to the token if not already present
// Returns true if scope was added, false if already exists
func (et *ExtendedToken) AddMCPScope(scope string) bool {
	// Validate MCP scope format
	if len(scope) < 4 || scope[:4] != "mcp:" {
		return false
	}

	// Check if already exists
	for _, existing := range et.Scope {
		if existing == scope {
			return false
		}
	}

	et.Scope = append(et.Scope, scope)
	return true
}

// ExtendedTokenRequest represents an RFC-0111 compliant token request
type ExtendedTokenRequest struct {
	// Basic OAuth 2.0 fields
	GrantID      string      `json:"grant_id"`
	Scope        []string    `json:"scope"`
	Restrictions interface{} `json:"restrictions,omitempty"`
	Context      interface{} `json:"context,omitempty"`

	// RFC-0111 Extended fields (complete)
	PowerOfAttorney      *poa.PoADefinition    `json:"power_of_attorney,omitempty"`
	AuthorizationChain   *AuthorizationChain   `json:"authorization_chain,omitempty"`
	ClientOwnerInfo      *ClientOwnerInfo      `json:"client_owner,omitempty"`
	OwnersAuthorizerInfo *OwnersAuthorizerInfo `json:"owners_authorizer,omitempty"`
	ResourceOwnerInfo    *ResourceOwnerInfo    `json:"resource_owner,omitempty"`
	LegalFramework       *LegalFrameworkInfo   `json:"legal_framework,omitempty"`
	RequestedActions     []string              `json:"requested_actions,omitempty"`
	RequestID            string                `json:"request_id,omitempty"`
	JurisdictionCode     string                `json:"jurisdiction,omitempty"`
	JurisdictionContext  *JurisdictionContext  `json:"jurisdiction_context,omitempty"`

	// RFC 9396
	AuthorizationDetails []AuthorizationDetail `json:"authorization_details,omitempty"`
}

// ExtendedTokenValidationResult represents RFC-0111 compliant validation result
type ExtendedTokenValidationResult struct {
	// Basic validation fields (backward compatibility)
	ClientID string   `json:"client_id"`
	Scope    []string `json:"scope"`
	Valid    bool     `json:"valid"`

	// RFC-0111 Extended validation results
	ExtendedToken          *ExtendedToken      `json:"extended_token,omitempty"`
	AuthorizationChain     *AuthorizationChain `json:"authorization_chain,omitempty"`
	ChainValidated         bool                `json:"chain_validated"`
	LegalFrameworkValid    bool                `json:"legal_framework_valid"`
	RestrictionsEnforced   bool                `json:"restrictions_enforced"`
	VerificationProofValid bool                `json:"verification_proof_valid"`
	ValidationTimestamp    time.Time           `json:"validation_timestamp"`
	ValidationWarnings     []string            `json:"validation_warnings,omitempty"`
}

// ToLegacyValidationResult converts to legacy TokenValidationResult for backward compatibility
func (evr *ExtendedTokenValidationResult) ToLegacyValidationResult() *TokenValidationResult {
	return &TokenValidationResult{
		ClientID: evr.ClientID,
		Scope:    evr.Scope,
		Valid:    evr.Valid,
	}
}

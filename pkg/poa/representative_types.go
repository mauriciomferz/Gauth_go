// Package poa provides enhanced representative type system per AAP001 and AAP002
// This file extends the base Representative types in poa.go with enhanced authorization chains
package poa

import (
	"fmt"
	"time"
)

// Note: This file uses types from poa.go:
// - LegalRelationship (enum)
// - RegistrationInfo (struct)
// - ContactInformation (struct)
// - CertificationStatus (struct)
// - ValidateLegalRelationship (function)

// RepresentativeType distinguishes the type of representative in authorization chain
// per AAP001 §4.1 Authorization Chain Requirements
type RepresentativeType string

const (
	// RepTypeOwnersAuthorizer - Statutory authorizer with legal authority over client owner
	// Examples: Managing Director, Prokura holder, Board member
	// Authorization Basis: Commercial register, corporate statutes, legal appointment
	RepTypeOwnersAuthorizer RepresentativeType = "OwnersAuthorizer"

	// RepTypeClientOwner - Owner/operator of the AI client system
	// Examples: Company owning the AI system, individual operator
	// Authorization Basis: Ownership, operational control, licensing
	RepTypeClientOwner RepresentativeType = "ClientOwner"

	// RepTypeDelegate - Delegated representative with limited authority
	// Examples: Employee with delegated authority, contractor
	// Authorization Basis: Power of attorney, employment contract, delegation agreement
	RepTypeDelegate RepresentativeType = "Delegate"

	// RepTypeServiceProvider - Third-party service provider
	// Examples: Cloud provider, AI service provider, infrastructure operator
	// Authorization Basis: Service agreement, terms of service
	RepTypeServiceProvider RepresentativeType = "ServiceProvider"
)

// ValidateRepresentativeType validates the representative type
func ValidateRepresentativeType(rt RepresentativeType) error {
	switch rt {
	case RepTypeOwnersAuthorizer, RepTypeClientOwner, RepTypeDelegate, RepTypeServiceProvider:
		return nil
	default:
		return fmt.Errorf("invalid representative type: %s", rt)
	}
}

// AuthorizationProof represents proof of authorization in the chain
// This provides verifiable evidence of authorization at each level
type AuthorizationProof struct {
	// ProofType indicates the type of proof provided
	ProofType AuthorizationProofType `json:"proof_type"`

	// DocumentReference points to the authorizing document
	// Examples: "Commercial Register Entry HRB 12345", "Proof of Authorization #2024-001"
	DocumentReference string `json:"document_reference"`

	// IssuingAuthority identifies who issued the proof
	// Examples: "Central Registry", "Notary Public", "Company Board"
	IssuingAuthority string `json:"issuing_authority"`

	// IssueDate when the authorization was granted (ISO 8601)
	IssueDate string `json:"issue_date"`

	// ExpiryDate when the authorization expires (ISO 8601, optional)
	ExpiryDate string `json:"expiry_date,omitempty"`

	// VerificationURI allows third-party verification
	// Examples: "https://handelsregister.de/verify?id=HRB12345"
	VerificationURI string `json:"verification_uri,omitempty"`

	// DigitalSignature cryptographic signature over the proof
	DigitalSignature string `json:"digital_signature,omitempty"`

	// CertificateChain for signature verification (PEM-encoded)
	CertificateChain []string `json:"certificate_chain,omitempty"`

	// Metadata for additional proof details
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// AuthorizationProofType classifies types of authorization proof
type AuthorizationProofType string

const (
	// ProofTypeCommercialRegister - Entry in commercial/trade register
	ProofTypeCommercialRegister AuthorizationProofType = "CommercialRegister"

	// ProofTypePowerOfAttorney - Notarized power of attorney document
	ProofTypePowerOfAttorney AuthorizationProofType = "PowerOfAttorney"

	// ProofTypeCorporateResolution - Board resolution or shareholders' decision
	ProofTypeCorporateResolution AuthorizationProofType = "CorporateResolution"

	// ProofTypeEmploymentContract - Employment agreement with authorization clause
	ProofTypeEmploymentContract AuthorizationProofType = "EmploymentContract"

	// ProofTypeServiceAgreement - Service or licensing agreement
	ProofTypeServiceAgreement AuthorizationProofType = "ServiceAgreement"

	// ProofTypeDelegationAgreement - Formal delegation of authority
	ProofTypeDelegationAgreement AuthorizationProofType = "DelegationAgreement"

	// ProofTypeCertificate - Professional or legal certificate
	ProofTypeCertificate AuthorizationProofType = "Certificate"

	// ProofTypeStatutoryAppointment - Legal appointment per statute
	ProofTypeStatutoryAppointment AuthorizationProofType = "StatutoryAppointment"
)

// ValidateAuthorizationProofType validates the proof type
func ValidateAuthorizationProofType(pt AuthorizationProofType) error {
	validTypes := []AuthorizationProofType{
		ProofTypeCommercialRegister, ProofTypePowerOfAttorney, ProofTypeCorporateResolution,
		ProofTypeEmploymentContract, ProofTypeServiceAgreement, ProofTypeDelegationAgreement,
		ProofTypeCertificate, ProofTypeStatutoryAppointment,
	}
	for _, valid := range validTypes {
		if pt == valid {
			return nil
		}
	}
	return fmt.Errorf("invalid authorization proof type: %s", pt)
}

// Validate performs validation of authorization proof
func (ap *AuthorizationProof) Validate() error {
	if err := ValidateAuthorizationProofType(ap.ProofType); err != nil {
		return fmt.Errorf("proof type: %w", err)
	}

	if ap.DocumentReference == "" {
		return fmt.Errorf("document reference required")
	}

	if ap.IssuingAuthority == "" {
		return fmt.Errorf("issuing authority required")
	}

	if ap.IssueDate == "" {
		return fmt.Errorf("issue date required")
	}

	// Validate date format (basic ISO 8601 check)
	if _, err := time.Parse(time.RFC3339, ap.IssueDate); err != nil {
		return fmt.Errorf("invalid issue date format (must be ISO 8601): %w", err)
	}

	if ap.ExpiryDate != "" {
		if _, err := time.Parse(time.RFC3339, ap.ExpiryDate); err != nil {
			return fmt.Errorf("invalid expiry date format (must be ISO 8601): %w", err)
		}
	}

	return nil
}

// IsExpired checks if the authorization proof has expired
func (ap *AuthorizationProof) IsExpired() bool {
	if ap.ExpiryDate == "" {
		return false // No expiry
	}

	expiryTime, err := time.Parse(time.RFC3339, ap.ExpiryDate)
	if err != nil {
		return true // Invalid date treated as expired
	}

	return time.Now().After(expiryTime)
}

// EnhancedRepresentative extends the basic Representative with type distinction
// and comprehensive authorization proof chain per AAP001 §4.1
type EnhancedRepresentative struct {
	// Type distinguishes the representative's role in authorization chain
	Type RepresentativeType `json:"type"`

	// Identity uniquely identifies the representative
	Identity string `json:"identity"`

	// Name is the human-readable name
	Name string `json:"name,omitempty"`

	// LegalRelationship to the AI client (from base Representative)
	LegalRelationship LegalRelationship `json:"legal_relationship"`

	// AuthorizationProof provides verifiable proof of authority
	AuthorizationProof *AuthorizationProof `json:"authorization_proof"`

	// RegistrationInfo contains commercial register details
	RegistrationInfo *RegistrationInfo `json:"registration_info,omitempty"`

	// ContactInformation for the representative
	ContactInformation *ContactInformation `json:"contact_information,omitempty"`

	// CertificationStatus tracks certifications
	CertificationStatus *CertificationStatus `json:"certification_status,omitempty"`

	// ChainPosition indicates position in authorization chain (0 = top/authorizer)
	ChainPosition int `json:"chain_position"`

	// AuthorizesToParty indicates which party this representative authorizes
	// For OwnersAuthorizer: authorizes ClientOwner
	// For ClientOwner: authorizes Client (AI system)
	AuthorizesToParty string `json:"authorizes_to_party,omitempty"`

	// Metadata for additional representative details
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// Validate performs comprehensive validation of enhanced representative
func (er *EnhancedRepresentative) Validate() error {
	if err := ValidateRepresentativeType(er.Type); err != nil {
		return fmt.Errorf("type: %w", err)
	}

	if er.Identity == "" {
		return fmt.Errorf("identity required")
	}

	if err := ValidateLegalRelationship(er.LegalRelationship); err != nil {
		return fmt.Errorf("legal relationship: %w", err)
	}

	if err := er.validateAuthorizationProof(); err != nil {
		return err
	}

	if err := er.validateTypeSpecificRules(); err != nil {
		return err
	}

	return er.validateOptionalDetails()
}

func (er *EnhancedRepresentative) validateAuthorizationProof() error {
	if er.AuthorizationProof == nil {
		return fmt.Errorf("authorization proof required")
	}

	if err := er.AuthorizationProof.Validate(); err != nil {
		return fmt.Errorf("authorization proof validation: %w", err)
	}

	if er.AuthorizationProof.IsExpired() {
		return fmt.Errorf("authorization proof has expired")
	}
	return nil
}

func (er *EnhancedRepresentative) validateTypeSpecificRules() error {
	switch er.Type {
	case RepTypeOwnersAuthorizer:
		// Owner's Authorizer must have commercial register proof or statutory appointment
		if er.AuthorizationProof.ProofType != ProofTypeCommercialRegister &&
			er.AuthorizationProof.ProofType != ProofTypeStatutoryAppointment &&
			er.AuthorizationProof.ProofType != ProofTypeCorporateResolution {
			return fmt.Errorf("owner's authorizer must have commercial register, statutory appointment, or corporate resolution proof")
		}
		if er.RegistrationInfo == nil {
			return fmt.Errorf("owner's authorizer must have registration info")
		}
		if er.ChainPosition != 0 {
			return fmt.Errorf("owner's authorizer must be at chain position 0")
		}

	case RepTypeClientOwner:
		// Client Owner must have registration info
		if er.RegistrationInfo == nil {
			return fmt.Errorf("client owner must have registration info")
		}
		if er.ChainPosition < 1 {
			return fmt.Errorf("client owner must be at chain position >= 1")
		}

	case RepTypeDelegate:
		// Delegate must have delegation agreement or power of attorney
		if er.AuthorizationProof.ProofType != ProofTypeDelegationAgreement &&
			er.AuthorizationProof.ProofType != ProofTypePowerOfAttorney &&
			er.AuthorizationProof.ProofType != ProofTypeEmploymentContract {
			return fmt.Errorf("delegate must have delegation agreement, power of attorney, or employment contract")
		}
	}
	return nil
}

func (er *EnhancedRepresentative) validateOptionalDetails() error {
	// Validate registration info if present
	if er.RegistrationInfo != nil {
		if er.RegistrationInfo.RegisteredName == "" {
			return fmt.Errorf("registration info: registered name required")
		}
		if er.RegistrationInfo.RegistrationNumber == "" {
			return fmt.Errorf("registration info: registration number required")
		}
		if er.RegistrationInfo.Jurisdiction == "" {
			return fmt.Errorf("registration info: jurisdiction required")
		}
	}

	// Validate contact info if present
	if er.ContactInformation != nil {
		if er.ContactInformation.PrimaryContact == "" {
			return fmt.Errorf("contact information: primary contact required")
		}
		if er.ContactInformation.Email == "" {
			return fmt.Errorf("contact information: email required")
		}
	}

	return nil
}

// IsAuthorizer checks if this representative is an Owner's Authorizer
func (er *EnhancedRepresentative) IsAuthorizer() bool {
	return er.Type == RepTypeOwnersAuthorizer
}

// IsOwner checks if this representative is a Client Owner
func (er *EnhancedRepresentative) IsOwner() bool {
	return er.Type == RepTypeClientOwner
}

// HasCommercialRegisterProof checks if authorization is backed by commercial register
func (er *EnhancedRepresentative) HasCommercialRegisterProof() bool {
	return er.AuthorizationProof != nil &&
		er.AuthorizationProof.ProofType == ProofTypeCommercialRegister
}

// GetAuthorizationLevel returns the authorization strength level
func (er *EnhancedRepresentative) GetAuthorizationLevel() string {
	if er.AuthorizationProof == nil {
		return "none"
	}

	// Strongest proofs
	if er.AuthorizationProof.ProofType == ProofTypeCommercialRegister ||
		er.AuthorizationProof.ProofType == ProofTypeStatutoryAppointment {
		return "high"
	}

	// Strong proofs
	if er.AuthorizationProof.ProofType == ProofTypePowerOfAttorney ||
		er.AuthorizationProof.ProofType == ProofTypeCorporateResolution {
		return "medium-high"
	}

	// Standard proofs
	if er.AuthorizationProof.ProofType == ProofTypeDelegationAgreement ||
		er.AuthorizationProof.ProofType == ProofTypeEmploymentContract {
		return "medium"
	}

	// Other proofs
	return "medium-low"
}

// AuthorizationChainLink represents a single link in the full authorization chain
// This extends the basic AuthorizationLink with enhanced proof and type information
type AuthorizationChainLink struct {
	// Position in the chain (0 = top/authorizer)
	Position int `json:"position"`

	// Representative at this chain position
	Representative EnhancedRepresentative `json:"representative"`

	// FromParty who grants authorization
	FromParty string `json:"from_party"`

	// ToParty who receives authorization
	ToParty string `json:"to_party"`

	// GrantedDate when authorization was granted (ISO 8601)
	GrantedDate string `json:"granted_date"`

	// ExpiryDate when authorization expires (ISO 8601, optional)
	ExpiryDate string `json:"expiry_date,omitempty"`

	// Scope of authority granted
	Scope string `json:"scope"`

	// Revocable indicates if authorization can be revoked
	Revocable bool `json:"revocable"`

	// SubDelegation allows further delegation
	SubDelegation bool `json:"sub_delegation"`

	// LegalBasis for the authorization
	LegalBasis string `json:"legal_basis"`
}

// Validate performs validation of authorization chain link
func (acl *AuthorizationChainLink) Validate() error {
	if err := acl.Representative.Validate(); err != nil {
		return fmt.Errorf("representative: %w", err)
	}

	if acl.FromParty == "" {
		return fmt.Errorf("from_party required")
	}

	if acl.ToParty == "" {
		return fmt.Errorf("to_party required")
	}

	if acl.Scope == "" {
		return fmt.Errorf("scope required")
	}

	if acl.GrantedDate == "" {
		return fmt.Errorf("granted_date required")
	}

	// Validate date format
	if _, err := time.Parse(time.RFC3339, acl.GrantedDate); err != nil {
		return fmt.Errorf("invalid granted date format: %w", err)
	}

	if acl.ExpiryDate != "" {
		if _, err := time.Parse(time.RFC3339, acl.ExpiryDate); err != nil {
			return fmt.Errorf("invalid expiry date format: %w", err)
		}
	}

	if acl.LegalBasis == "" {
		return fmt.Errorf("legal_basis required")
	}

	return nil
}

// IsExpired checks if this chain link has expired
func (acl *AuthorizationChainLink) IsExpired() bool {
	if acl.ExpiryDate == "" {
		return false
	}

	expiryTime, err := time.Parse(time.RFC3339, acl.ExpiryDate)
	if err != nil {
		return true
	}

	return time.Now().After(expiryTime)
}

// ValidateEnhancedAuthorizationChain validates a full authorization chain with enhanced representatives
func ValidateEnhancedAuthorizationChain(chain []AuthorizationChainLink) error {
	if len(chain) == 0 {
		return fmt.Errorf("authorization chain cannot be empty")
	}

	// Validate each link
	for i, link := range chain {
		if err := link.Validate(); err != nil {
			return fmt.Errorf("chain link %d: %w", i, err)
		}

		if link.IsExpired() {
			return fmt.Errorf("chain link %d has expired", i)
		}

		// Verify position matches index
		if link.Position != i {
			return fmt.Errorf("chain link %d: position mismatch (expected %d, got %d)", i, i, link.Position)
		}
	}

	// Verify chain continuity
	for i := 0; i < len(chain)-1; i++ {
		if chain[i].ToParty != chain[i+1].FromParty {
			return fmt.Errorf("chain broken at link %d: %s -> %s != %s -> %s",
				i, chain[i].FromParty, chain[i].ToParty, chain[i+1].FromParty, chain[i+1].ToParty)
		}
	}

	// Verify sub-delegation permissions
	for i := 1; i < len(chain); i++ {
		if !chain[i-1].SubDelegation {
			return fmt.Errorf("chain link %d: sub-delegation not allowed by previous link", i)
		}
	}

	// Verify type hierarchy: OwnersAuthorizer (position 0) -> ClientOwner (position 1+) -> Client
	if chain[0].Representative.Type != RepTypeOwnersAuthorizer {
		return fmt.Errorf("chain must start with OwnersAuthorizer (got %s)", chain[0].Representative.Type)
	}

	// Verify chain positions match representative chain positions
	for i, link := range chain {
		if link.Representative.ChainPosition != i {
			return fmt.Errorf("chain link %d: representative chain position mismatch", i)
		}
	}

	return nil
}

// BuildAuthorizationChainSummary creates a human-readable summary of the chain
func BuildAuthorizationChainSummary(chain []AuthorizationChainLink) string {
	if len(chain) == 0 {
		return "No authorization chain"
	}

	summary := fmt.Sprintf("Authorization Chain (%d links):\n", len(chain))
	for i, link := range chain {
		summary += fmt.Sprintf("  [%d] %s (%s) -> %s\n",
			i,
			link.Representative.Name,
			link.Representative.Type,
			link.ToParty)
		summary += fmt.Sprintf("      Proof: %s (%s)\n",
			link.Representative.AuthorizationProof.ProofType,
			link.Representative.AuthorizationProof.DocumentReference)
		summary += fmt.Sprintf("      Scope: %s\n", link.Scope)
	}

	return summary
}

// Package gauth - External Integration Interfaces per RFC-0111
// Implements critical Gaps #4 and #5 from QUALITY_MANAGER_RFC_COMPLIANCE_FINAL_ASSESSMENT.md
package gauth

import (
	"context"
	"time"
)

// CommercialRegisterClient defines the interface for commercial register integration
// RFC-0111 Requirement: Validation of owner's authorizer statutory authority
// through commercial register verification
type CommercialRegisterClient interface {
	// VerifyCompany verifies a company's existence and active status
	VerifyCompany(ctx context.Context, jurisdiction string, companyID string) (*CompanyInfo, error)

	// VerifyManagingDirector verifies a person's managing director authority
	VerifyManagingDirector(ctx context.Context, companyID string, personID string) (*DirectorInfo, error)

	// VerifyPowerOfAttorney verifies a registered power of attorney
	VerifyPowerOfAttorney(ctx context.Context, companyID string, poaID string) (*PoARegistration, error)

	// GetSignatoryRights retrieves signatory rights for a person in a company
	GetSignatoryRights(ctx context.Context, companyID string, personID string) (*SignatoryRights, error)

	// GetCompanyStructure retrieves company legal structure information
	GetCompanyStructure(ctx context.Context, companyID string) (*CompanyStructure, error)
}

// TrustServiceProvider defines the interface for trust service provider integration
// RFC-0111 Section 3 (PVP): "verification of the identities that perform a specific
// role along the GAuth processing. E.g., a trust service provider that also runs
// the authorization server."
type TrustServiceProvider interface {
	// VerifyIdentity verifies an identity document or credential
	VerifyIdentity(ctx context.Context, identity *IdentityDocument) (*VerificationResult, error)

	// VerifySignature verifies a digital signature
	VerifySignature(ctx context.Context, data []byte, signature []byte, certID string) error

	// GetCertificateChain retrieves a certificate chain for validation
	GetCertificateChain(ctx context.Context, certID string) ([]*X509Certificate, error)

	// VerifyTimestamp verifies a trusted timestamp
	VerifyTimestamp(ctx context.Context, timestamp *Timestamp) (*TimestampValidation, error)

	// GetQualificationStatus retrieves the qualification status of the TSP
	GetQualificationStatus(ctx context.Context) (*TSPQualificationStatus, error)
}

// RevocationChecker defines the interface for checking authorization revocations
type RevocationChecker interface {
	// IsRevoked checks if an entity's authorization has been revoked
	IsRevoked(ctx context.Context, entityID string) (bool, error)

	// IsDelegationRevoked checks if a specific delegation ID has been revoked
	IsDelegationRevoked(ctx context.Context, delegationID string) (bool, error)

	// GetRevocationInfo retrieves detailed revocation information
	GetRevocationInfo(ctx context.Context, entityID string) (*RevocationInfo, error)

	// CheckCertificateRevocation checks certificate revocation status (OCSP/CRL)
	CheckCertificateRevocation(ctx context.Context, certID string) (*CertificateRevocationStatus, error)
}

// CompanyInfo represents commercial register company information
type CompanyInfo struct {
	CompanyID             string           `json:"company_id"`
	RegistrationNumber    string           `json:"registration_number"`
	LegalName             string           `json:"legal_name"`
	LegalForm             string           `json:"legal_form"` // e.g., "GmbH", "AG", "Ltd", "Inc"
	Jurisdiction          string           `json:"jurisdiction"`
	RegisterType          string           `json:"register_type"` // e.g., "Handelsregister", "Companies House"
	RegistrationDate      time.Time        `json:"registration_date"`
	Active                bool             `json:"active"`
	Status                string           `json:"status"` // "active", "liquidation", "dissolved"
	RegisteredAddress     *Address         `json:"registered_address"`
	ManagingDirectors     []*DirectorInfo  `json:"managing_directors,omitempty"`
	AuthorizedSignatories []*SignatoryInfo `json:"authorized_signatories,omitempty"`
	ShareCapital          *ShareCapital    `json:"share_capital,omitempty"`
	VerificationDate      time.Time        `json:"verification_date"`
	VerificationSource    string           `json:"verification_source"`
}

// DirectorInfo represents managing director information from commercial register
type DirectorInfo struct {
	PersonID              string    `json:"person_id"`
	FirstName             string    `json:"first_name"`
	LastName              string    `json:"last_name"`
	Title                 string    `json:"title,omitempty"`
	Role                  string    `json:"role"` // "managing_director", "board_member", "ceo"
	AppointmentDate       time.Time `json:"appointment_date"`
	AppointmentEnd        time.Time `json:"appointment_end,omitempty"`
	Active                bool      `json:"active"`
	SignatoryRights       string    `json:"signatory_rights"` // "individual", "joint", "collective"
	PowerOfRepresentation string    `json:"power_of_representation,omitempty"`
	Limitations           []string  `json:"limitations,omitempty"`
	VerificationDate      time.Time `json:"verification_date"`
}

// PoARegistration represents a registered power of attorney from commercial register
type PoARegistration struct {
	PoAID            string    `json:"poa_id"`
	PoAType          string    `json:"poa_type"` // "general", "special", "prokura"
	GrantorID        string    `json:"grantor_id"`
	GrantorName      string    `json:"grantor_name"`
	AttorneyID       string    `json:"attorney_id"`
	AttorneyName     string    `json:"attorney_name"`
	RegistrationDate time.Time `json:"registration_date"`
	EffectiveDate    time.Time `json:"effective_date"`
	ExpirationDate   time.Time `json:"expiration_date,omitempty"`
	Scope            []string  `json:"scope"`
	Limitations      []string  `json:"limitations,omitempty"`
	Revoked          bool      `json:"revoked"`
	RevocationDate   time.Time `json:"revocation_date,omitempty"`
	Active           bool      `json:"active"`
	VerificationDate time.Time `json:"verification_date"`
}

// SignatoryRights represents signatory authority information
type SignatoryRights struct {
	PersonID         string       `json:"person_id"`
	PersonName       string       `json:"person_name"`
	CompanyID        string       `json:"company_id"`
	RightsType       string       `json:"rights_type"` // "individual", "joint", "collective"
	Scope            []string     `json:"scope"`
	Limitations      []string     `json:"limitations,omitempty"`
	ValueLimits      *ValueLimits `json:"value_limits,omitempty"`
	GeographicScope  []string     `json:"geographic_scope,omitempty"`
	ValidFrom        time.Time    `json:"valid_from"`
	ValidUntil       time.Time    `json:"valid_until,omitempty"`
	Active           bool         `json:"active"`
	Source           string       `json:"source"` // "statutory", "power_of_attorney", "delegation"
	VerificationDate time.Time    `json:"verification_date"`
}

// CompanyStructure represents company legal structure information
type CompanyStructure struct {
	CompanyID                string               `json:"company_id"`
	LegalForm                string               `json:"legal_form"`
	GovernanceModel          string               `json:"governance_model"`     // "monistic", "dualistic"
	ManagementStructure      string               `json:"management_structure"` // "single_director", "board", "management_board"
	Shareholders             []*ShareholderInfo   `json:"shareholders,omitempty"`
	ControllingEntities      []*ControllingEntity `json:"controlling_entities,omitempty"`
	SubsidiaryOf             string               `json:"subsidiary_of,omitempty"`
	UltimateBeneficialOwners []*UBOInfo           `json:"ultimate_beneficial_owners,omitempty"`
	VerificationDate         time.Time            `json:"verification_date"`
}

// Address represents a physical address
type Address struct {
	Street     string `json:"street"`
	Number     string `json:"number"`
	City       string `json:"city"`
	PostalCode string `json:"postal_code"`
	Region     string `json:"region,omitempty"`
	Country    string `json:"country"` // ISO 3166-1 alpha-2
}

// ShareCapital represents company share capital information
type ShareCapital struct {
	Currency   string  `json:"currency"`
	Authorized float64 `json:"authorized"`
	Issued     float64 `json:"issued"`
	PaidUp     float64 `json:"paid_up"`
}

// SignatoryInfo represents an authorized signatory
type SignatoryInfo struct {
	PersonID   string    `json:"person_id"`
	Name       string    `json:"name"`
	Role       string    `json:"role"`
	ValidFrom  time.Time `json:"valid_from"`
	ValidUntil time.Time `json:"valid_until,omitempty"`
	Active     bool      `json:"active"`
}

// ShareholderInfo represents shareholder information
type ShareholderInfo struct {
	ShareholderID   string  `json:"shareholder_id"`
	Name            string  `json:"name"`
	EntityType      string  `json:"entity_type"` // "natural_person", "legal_entity"
	SharePercentage float64 `json:"share_percentage"`
	VotingRights    float64 `json:"voting_rights"`
}

// ControllingEntity represents a controlling entity
type ControllingEntity struct {
	EntityID      string  `json:"entity_id"`
	Name          string  `json:"name"`
	ControlType   string  `json:"control_type"`   // "majority_shareholder", "parent_company", "group_entity"
	ControlDegree float64 `json:"control_degree"` // percentage
}

// UBOInfo represents ultimate beneficial owner information
type UBOInfo struct {
	PersonID      string  `json:"person_id"`
	Name          string  `json:"name"`
	Nationality   string  `json:"nationality"`
	BenefitType   string  `json:"benefit_type"`   // "ownership", "control", "both"
	BenefitDegree float64 `json:"benefit_degree"` // percentage
}

// ValueLimits represents value-based limitations
type ValueLimits struct {
	Currency          string  `json:"currency"`
	SingleTransaction float64 `json:"single_transaction,omitempty"`
	DailyLimit        float64 `json:"daily_limit,omitempty"`
	MonthlyLimit      float64 `json:"monthly_limit,omitempty"`
	AnnualLimit       float64 `json:"annual_limit,omitempty"`
}

// IdentityDocument represents an identity document for verification
type IdentityDocument struct {
	DocumentID       string            `json:"document_id"`
	DocumentType     string            `json:"document_type"` // "passport", "id_card", "eidas_certificate"
	DocumentNumber   string            `json:"document_number,omitempty"`
	IssuingCountry   string            `json:"issuing_country,omitempty"`
	IssuingAuthority string            `json:"issuing_authority,omitempty"`
	IssueDate        time.Time         `json:"issue_date,omitempty"`
	ExpirationDate   time.Time         `json:"expiration_date,omitempty"`
	SubjectID        string            `json:"subject_id"`
	SubjectName      string            `json:"subject_name,omitempty"`
	SubjectData      map[string]string `json:"subject_data,omitempty"`
	VerificationData []byte            `json:"verification_data,omitempty"`
}

// VerificationResult represents identity verification result
type VerificationResult struct {
	Verified           bool                   `json:"verified"`
	AssuranceLevel     string                 `json:"assurance_level"` // "low", "substantial", "high" (eIDAS levels)
	VerificationMethod string                 `json:"verification_method"`
	VerifierID         string                 `json:"verifier_id"`
	VerificationDate   time.Time              `json:"verification_date"`
	ValidUntil         time.Time              `json:"valid_until,omitempty"`
	VerificationProof  string                 `json:"verification_proof,omitempty"`
	Warnings           []string               `json:"warnings,omitempty"`
	Metadata           map[string]interface{} `json:"metadata,omitempty"`
}

// X509Certificate represents an X.509 certificate
type X509Certificate struct {
	CertificateID   string    `json:"certificate_id"`
	Subject         string    `json:"subject"`
	Issuer          string    `json:"issuer"`
	SerialNumber    string    `json:"serial_number"`
	NotBefore       time.Time `json:"not_before"`
	NotAfter        time.Time `json:"not_after"`
	PublicKey       []byte    `json:"public_key"`
	Signature       []byte    `json:"signature"`
	CertificateData []byte    `json:"certificate_data"`
}

// Timestamp represents a trusted timestamp
type Timestamp struct {
	TimestampID        string    `json:"timestamp_id"`
	Timestamp          time.Time `json:"timestamp"`
	HashedData         []byte    `json:"hashed_data"`
	TimestampToken     []byte    `json:"timestamp_token"`
	TSAIdentifier      string    `json:"tsa_identifier"`
	SignatureAlgorithm string    `json:"signature_algorithm"`
}

// TimestampValidation represents timestamp validation result
type TimestampValidation struct {
	Valid            bool      `json:"valid"`
	Timestamp        time.Time `json:"timestamp"`
	TSAVerified      bool      `json:"tsa_verified"`
	SignatureValid   bool      `json:"signature_valid"`
	CertificateValid bool      `json:"certificate_valid"`
	Message          string    `json:"message,omitempty"`
}

// TSPQualificationStatus represents TSP qualification status
type TSPQualificationStatus struct {
	ProviderID          string    `json:"provider_id"`
	ProviderName        string    `json:"provider_name"`
	Qualified           bool      `json:"qualified"`          // eIDAS qualified
	QualificationType   string    `json:"qualification_type"` // "eIDAS", "national", "international"
	AccreditationBody   string    `json:"accreditation_body"`
	AccreditationDate   time.Time `json:"accreditation_date"`
	AccreditationExpiry time.Time `json:"accreditation_expiry,omitempty"`
	ServiceTypes        []string  `json:"service_types"`
	Jurisdiction        string    `json:"jurisdiction"`
	Status              string    `json:"status"` // "active", "suspended", "withdrawn"
	VerificationDate    time.Time `json:"verification_date"`
}

// RevocationInfo represents detailed revocation information
type RevocationInfo struct {
	EntityID         string    `json:"entity_id"`
	Revoked          bool      `json:"revoked"`
	RevocationDate   time.Time `json:"revocation_date,omitempty"`
	RevocationReason string    `json:"revocation_reason,omitempty"`
	RevokedBy        string    `json:"revoked_by,omitempty"`
	EffectiveDate    time.Time `json:"effective_date,omitempty"`
	VerificationDate time.Time `json:"verification_date"`
}

// CertificateRevocationStatus represents certificate revocation status
type CertificateRevocationStatus struct {
	CertificateID    string    `json:"certificate_id"`
	Revoked          bool      `json:"revoked"`
	RevocationDate   time.Time `json:"revocation_date,omitempty"`
	RevocationReason string    `json:"revocation_reason,omitempty"`
	CheckMethod      string    `json:"check_method"` // "OCSP", "CRL"
	CheckDate        time.Time `json:"check_date"`
	NextUpdate       time.Time `json:"next_update,omitempty"`
}

// Package agentauth provides formal requirements enforcement for AAP-001/AAP-002 compliance
package agentauth

import (
	"context"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"strings"
	"time"

	"github.com/mauriciomferz/AgentAuth/pkg/poa"
)

// FormalRequirementsValidator validates formal legal requirements for PoA
// AAP-002 Section C.1 - Formal Requirements
type FormalRequirementsValidator struct {
	// External service clients
	notaryVerifier NotarialCertificateVerifier
	idVerifier     IdentityDocumentVerifier
	sigVerifier    DigitalSignatureVerifier

	// Configuration
	strictMode            bool
	requireNotaryForValue float64 // Threshold requiring notarial certification
	acceptedIDTypes       []string
	acceptedJurisdictions []string
}

// NotarialCertificateVerifier verifies notarial certifications
type NotarialCertificateVerifier interface {
	VerifyNotarialCertificate(ctx context.Context, cert *NotarialCertificate) (*NotarialVerificationResult, error)
	VerifyNotaryLicense(ctx context.Context, notaryID, jurisdiction string) (*NotaryLicenseInfo, error)
	CheckNotarySealAuthenticity(ctx context.Context, sealData []byte, notaryID string) (bool, error)
}

// IdentityDocumentVerifier verifies identity documents
// Uses IdentityDocument type from external_integrations.go
type IdentityDocumentVerifier interface {
	VerifyIdentityDocument(ctx context.Context, doc *IdentityDocument) (*IDVerificationResult, error)
	VerifyGovernmentID(ctx context.Context, idNumber, idType, issuingCountry string) (*GovernmentIDInfo, error)
	CheckIDExpiration(ctx context.Context, idNumber, idType string) (bool, time.Time, error)
	VerifyBiometricMatch(ctx context.Context, biometric []byte, idNumber string) (bool, float64, error)
}

// DigitalSignatureVerifier verifies digital signatures
type DigitalSignatureVerifier interface {
	VerifyDigitalSignature(ctx context.Context, data []byte, signature []byte, cert *x509.Certificate) error
	VerifyQualifiedSignature(ctx context.Context, data []byte, signature []byte, qcert *QualifiedCertificate) error
	CheckSignatureTimestamp(ctx context.Context, signature []byte) (*SignatureTimestamp, error)
	VerifySignatureChain(ctx context.Context, signatures []DigitalSignature) (*SignatureChainResult, error)
}

// NotarialCertificate represents a notarial certification
type NotarialCertificate struct {
	CertificateID        string
	NotaryID             string
	NotaryName           string
	NotaryLicense        string
	Jurisdiction         string
	IssuingAuthority string
	CertificationDate    time.Time
	ExpirationDate       time.Time
	DocumentHash         string
	NotarySeal           []byte
	NotarySignature      []byte
	ApostilleAttached    bool
	ApostilleNumber      string
	CertificationType    string // "acknowledgment", "jurat", "oath", "affirmation"
	WitnessStatements    []WitnessStatement
	Metadata             map[string]interface{}
}

// WitnessStatement represents a witness attestation
type WitnessStatement struct {
	WitnessName    string
	WitnessID      string
	WitnessAddress string
	Statement      string
	SignatureDate  time.Time
	Signature      []byte
}

// Note: Using IdentityDocument from external_integrations.go

// QualifiedCertificate represents an eIDAS qualified certificate
type QualifiedCertificate struct {
	Certificate       *x509.Certificate
	QualificationInfo QualificationInfo
	TSPInfo           TSPInfo
	ValidationURL     string
	RevocationURL     string
}

// QualificationInfo contains eIDAS qualification information
type QualificationInfo struct {
	QualificationType string // "QES" (Qualified Electronic Signature), "QSeal", "QTS"
	AssuranceLevel    string // "Substantial", "High"
	LegalEffect       string
	Policies          []string
}

// TSPInfo contains Trust Service Provider information
type TSPInfo struct {
	Name          string
	TradeName     string
	Country       string
	TrustMark     string
	Status        string
	SupervisionBy string
}

// DigitalSignature represents a digital signature
type DigitalSignature struct {
	SignatureValue []byte
	SignatureAlg   string
	Certificate    *x509.Certificate
	Timestamp      time.Time
	SignerInfo     string
	Metadata       map[string]interface{}
}

// NotarialVerificationResult contains notarial verification results
type NotarialVerificationResult struct {
	Valid                bool
	CertificateAuthentic bool
	NotaryLicenseValid   bool
	SealAuthentic        bool
	ExpirationValid      bool
	JurisdictionValid    bool
	ApostilleValid       bool
	Issues               []string
	Warnings             []string
	VerificationDate     time.Time
	VerifierInfo         string
}

// IDVerificationResult contains identity verification results
type IDVerificationResult struct {
	Valid                bool
	DocumentAuthentic    bool
	NotExpired           bool
	BiometricMatch       bool
	BiometricScore       float64
	SecurityFeatureOK    bool
	IssuinagentAuthValid bool
	ChipDataValid        bool
	Issues               []string
	Warnings             []string
	VerificationDate     time.Time
	VerificationMethod   string
}

// GovernmentIDInfo contains government ID information
type GovernmentIDInfo struct {
	IDNumber       string
	IDType         string
	HolderName     string
	IssuingCountry string
	Status         string // "valid", "expired", "revoked", "reported_lost"
	IssueDate      time.Time
	ExpiryDate     time.Time
	Metadata       map[string]interface{}
}

// SignatureTimestamp contains signature timestamp information
type SignatureTimestamp struct {
	Timestamp      time.Time
	TSAName        string
	TSACountry     string
	Accuracy       time.Duration
	Verified       bool
	TimestampToken []byte
}

// SignatureChainResult contains signature chain verification results
type SignatureChainResult struct {
	Valid           bool
	SignaturesCount int
	AllVerified     bool
	ChainIntegrity  bool
	TimestampValid  bool
	Issues          []string
	VerifiedSigners []string
}

// NotaryLicenseInfo contains notary license information
type NotaryLicenseInfo struct {
	NotaryID             string
	LicenseNumber        string
	Status               string // "active", "suspended", "revoked", "expired"
	Jurisdiction         string
	IssueDate            time.Time
	ExpiryDate           time.Time
	IssuingAuthority string
	LicenseType          string
	Restrictions         []string
}

// FormalRequirementsResult contains validation results
type FormalRequirementsResult struct {
	Valid                       bool
	NotarialCertificationValid  bool
	IDVerificationValid         bool
	DigitalSignaturesValid      bool
	WrittenFormCompliant        bool
	JurisdictionCompliant       bool
	Issues                      []string
	Warnings                    []string
	ValidationDate              time.Time
	ApplicableLegalRequirements []string
}

// NewFormalRequirementsValidator creates a new formal requirements validator
func NewFormalRequirementsValidator(
	notaryVerifier NotarialCertificateVerifier,
	idVerifier IdentityDocumentVerifier,
	sigVerifier DigitalSignatureVerifier,
	strictMode bool,
) *FormalRequirementsValidator {
	return &FormalRequirementsValidator{
		notaryVerifier:        notaryVerifier,
		idVerifier:            idVerifier,
		sigVerifier:           sigVerifier,
		strictMode:            strictMode,
		requireNotaryForValue: 100000.0, // Default threshold
		acceptedIDTypes: []string{
			"passport", "national_id", "drivers_license",
			"residence_permit", "diplomatic_id",
		},
		acceptedJurisdictions: []string{
			"DE", "AT", "CH", "FR", "IT", "ES", "NL", "BE",
			"UK", "IE", "US", "CA", "AU", "NZ", "SG",
		},
	}
}

// ValidateFormalRequirements validates formal requirements for a PoA
// AAP-002 Section C.1 - Formal Requirements validation
func (v *FormalRequirementsValidator) ValidateFormalRequirements(
	ctx context.Context,
	poaDef *poa.PoADefinition,
	notaryCert *NotarialCertificate,
	identityDocs []*IdentityDocument,
	digitalSigs []DigitalSignature,
) (*FormalRequirementsResult, error) {
	result := &FormalRequirementsResult{
		Valid:                       true,
		ValidationDate:              time.Now(),
		Issues:                      []string{},
		Warnings:                    []string{},
		ApplicableLegalRequirements: []string{},
	}

	// Step 1: Check if formal requirements are specified
	if poaDef.Requirements.FormalRequirements == (poa.FormalRequirements{}) {
		result.Warnings = append(result.Warnings, "No formal requirements specified in PoA")
		return result, nil
	}

	formalReqs := poaDef.Requirements.FormalRequirements

	// Step 2: Validate notarial certification if required
	if formalReqs.NotarialCertification {
		result.ApplicableLegalRequirements = append(result.ApplicableLegalRequirements,
			"AAP-002 Section C.1: Notarial certification required")

		if notaryCert == nil {
			result.Valid = false
			result.NotarialCertificationValid = false
			result.Issues = append(result.Issues, "Notarial certification required but not provided")
		} else {
			notaryResult, err := v.ValidateNotarialCertification(ctx, notaryCert)
			if err != nil {
				return nil, fmt.Errorf("notarial verification failed: %w", err)
			}
			result.NotarialCertificationValid = notaryResult.Valid
			if !notaryResult.Valid {
				result.Valid = false
				result.Issues = append(result.Issues, notaryResult.Issues...)
			}
			result.Warnings = append(result.Warnings, notaryResult.Warnings...)
		}
	} else {
		result.NotarialCertificationValid = true // Not required
	}

	// Step 3: Validate ID verification if required
	if formalReqs.IDVerificationRequired {
		result.ApplicableLegalRequirements = append(result.ApplicableLegalRequirements,
			"AAP-002 Section C.1: Identity verification required")

		if len(identityDocs) == 0 {
			result.Valid = false
			result.IDVerificationValid = false
			result.Issues = append(result.Issues, "ID verification required but no identity documents provided")
		} else {
			idResult, err := v.ValidateIdentityDocuments(ctx, identityDocs)
			if err != nil {
				return nil, fmt.Errorf("ID verification failed: %w", err)
			}
			result.IDVerificationValid = idResult.Valid
			if !idResult.Valid {
				result.Valid = false
				result.Issues = append(result.Issues, idResult.Issues...)
			}
			result.Warnings = append(result.Warnings, idResult.Warnings...)
		}
	} else {
		result.IDVerificationValid = true // Not required
	}

	// Step 4: Validate digital signatures if required
	if formalReqs.DigitalSignatures {
		result.ApplicableLegalRequirements = append(result.ApplicableLegalRequirements,
			"AAP-002 Section C.1: Digital signatures required")

		if len(digitalSigs) == 0 {
			result.Valid = false
			result.DigitalSignaturesValid = false
			result.Issues = append(result.Issues, "Digital signatures required but none provided")
		} else {
			sigResult, err := v.ValidateDigitalSignatures(ctx, digitalSigs, poaDef)
			if err != nil {
				return nil, fmt.Errorf("digital signature verification failed: %w", err)
			}
			result.DigitalSignaturesValid = sigResult.Valid
			if !sigResult.Valid {
				result.Valid = false
				result.Issues = append(result.Issues, sigResult.Issues...)
			}
			// SignatureChainResult doesn't have Warnings field
		}
	} else {
		result.DigitalSignaturesValid = true // Not required
	}

	// Step 5: Validate written form compliance
	writtenFormResult := v.validateWrittenFormRequirements(poaDef)
	result.WrittenFormCompliant = writtenFormResult.Valid
	if !writtenFormResult.Valid {
		if v.strictMode {
			result.Valid = false
		}
		result.Issues = append(result.Issues, writtenFormResult.Issues...)
	}
	result.Warnings = append(result.Warnings, writtenFormResult.Warnings...)

	// Step 6: Validate jurisdiction-specific requirements
	jurisdictionResult := v.validateJurisdictionRequirements(poaDef)
	result.JurisdictionCompliant = jurisdictionResult.Valid
	if !jurisdictionResult.Valid {
		result.Warnings = append(result.Warnings, jurisdictionResult.Issues...)
	}

	return result, nil
}

// ValidateNotarialCertification validates a notarial certification
func (v *FormalRequirementsValidator) ValidateNotarialCertification(
	ctx context.Context,
	cert *NotarialCertificate,
) (*NotarialVerificationResult, error) {
	result := &NotarialVerificationResult{
		Valid:            true,
		VerificationDate: time.Now(),
		Issues:           []string{},
		Warnings:         []string{},
	}

	// Step 1: Verify basic certificate structure
	if cert.NotaryID == "" || cert.NotaryLicense == "" {
		result.Valid = false
		result.Issues = append(result.Issues, "Missing notary identification information")
		return result, nil
	}

	// Step 2: Verify notary license
	if v.notaryVerifier != nil {
		licenseInfo, err := v.notaryVerifier.VerifyNotaryLicense(ctx, cert.NotaryID, cert.Jurisdiction)
		if err != nil {
			return nil, fmt.Errorf("failed to verify notary license: %w", err)
		}

		if licenseInfo.Status != "active" {
			result.Valid = false
			result.NotaryLicenseValid = false
			result.Issues = append(result.Issues, fmt.Sprintf("Notary license status: %s", licenseInfo.Status))
		} else {
			result.NotaryLicenseValid = true
		}

		// Check license expiration
		if time.Now().After(licenseInfo.ExpiryDate) {
			result.Valid = false
			result.ExpirationValid = false
			result.Issues = append(result.Issues, "Notary license expired")
		} else {
			result.ExpirationValid = true
		}
	} else {
		result.Warnings = append(result.Warnings, "Notary license verification skipped (verifier not configured)")
	}

	// Step 3: Verify certificate expiration
	if time.Now().After(cert.ExpirationDate) {
		result.Valid = false
		result.ExpirationValid = false
		result.Issues = append(result.Issues, "Notarial certificate expired")
	} else if time.Now().After(cert.ExpirationDate.Add(-30 * 24 * time.Hour)) {
		result.Warnings = append(result.Warnings, "Notarial certificate expires within 30 days")
	}

	// Step 4: Verify notary seal authenticity
	if v.notaryVerifier != nil && len(cert.NotarySeal) > 0 {
		sealValid, err := v.notaryVerifier.CheckNotarySealAuthenticity(ctx, cert.NotarySeal, cert.NotaryID)
		if err != nil {
			return nil, fmt.Errorf("failed to verify notary seal: %w", err)
		}
		result.SealAuthentic = sealValid
		if !sealValid {
			result.Valid = false
			result.Issues = append(result.Issues, "Notary seal authentication failed")
		}
	}

	// Step 5: Verify jurisdiction
	jurisdictionValid := false
	for _, accepted := range v.acceptedJurisdictions {
		if cert.Jurisdiction == accepted {
			jurisdictionValid = true
			break
		}
	}
	result.JurisdictionValid = jurisdictionValid
	if !jurisdictionValid && v.strictMode {
		result.Valid = false
		result.Issues = append(result.Issues, fmt.Sprintf("Jurisdiction not accepted: %s", cert.Jurisdiction))
	}

	// Step 6: Verify apostille if required for international use
	if cert.ApostilleAttached {
		result.ApostilleValid = cert.ApostilleNumber != ""
		if !result.ApostilleValid {
			result.Warnings = append(result.Warnings, "Apostille attached but number missing")
		}
	}

	// Step 7: Verify certification type
	validTypes := []string{"acknowledgment", "jurat", "oath", "affirmation", "certification"}
	typeValid := false
	for _, vt := range validTypes {
		if cert.CertificationType == vt {
			typeValid = true
			break
		}
	}
	if !typeValid {
		result.Warnings = append(result.Warnings, fmt.Sprintf("Unknown certification type: %s", cert.CertificationType))
	}

	result.CertificateAuthentic = result.Valid

	return result, nil
}

// ValidateIdentityDocuments validates identity documents
func (v *FormalRequirementsValidator) ValidateIdentityDocuments(
	ctx context.Context,
	docs []*IdentityDocument,
) (*IDVerificationResult, error) {
	result := &IDVerificationResult{
		Valid:              true,
		VerificationDate:   time.Now(),
		VerificationMethod: "composite",
		Issues:             []string{},
		Warnings:           []string{},
	}

	if len(docs) == 0 {
		result.Valid = false
		result.Issues = append(result.Issues, "No identity documents provided")
		return result, nil
	}

	// Validate each document
	allValid := true
	for i, doc := range docs {
		docResult, err := v.validateSingleIdentityDocument(ctx, doc)
		if err != nil {
			return nil, fmt.Errorf("failed to verify document %d: %w", i, err)
		}

		if !docResult.Valid {
			allValid = false
			result.Issues = append(result.Issues, fmt.Sprintf("Document %d (%s): %s",
				i, doc.DocumentType, strings.Join(docResult.Issues, "; ")))
		}
		result.Warnings = append(result.Warnings, docResult.Warnings...)

		// Aggregate results
		result.DocumentAuthentic = result.DocumentAuthentic || docResult.DocumentAuthentic
		result.NotExpired = result.NotExpired || docResult.NotExpired
		result.BiometricMatch = result.BiometricMatch || docResult.BiometricMatch
		if docResult.BiometricScore > result.BiometricScore {
			result.BiometricScore = docResult.BiometricScore
		}
		result.SecurityFeatureOK = result.SecurityFeatureOK || docResult.SecurityFeatureOK
		result.IssuinagentAuthValid = result.IssuinagentAuthValid || docResult.IssuinagentAuthValid
		result.ChipDataValid = result.ChipDataValid || docResult.ChipDataValid
	}

	result.Valid = allValid

	// At least one valid document required
	if !result.DocumentAuthentic || !result.NotExpired {
		result.Valid = false
		if !result.DocumentAuthentic {
			result.Issues = append(result.Issues, "No authentic documents provided")
		}
		if !result.NotExpired {
			result.Issues = append(result.Issues, "All documents expired")
		}
	}

	return result, nil
}

// validateSingleIdentityDocument validates a single identity document
func (v *FormalRequirementsValidator) validateSingleIdentityDocument(
	ctx context.Context,
	doc *IdentityDocument,
) (*IDVerificationResult, error) {
	result := &IDVerificationResult{
		Valid:              true,
		VerificationDate:   time.Now(),
		VerificationMethod: "single_document",
		Issues:             []string{},
		Warnings:           []string{},
	}

	// Step 1: Check document type
	typeValid := false
	for _, accepted := range v.acceptedIDTypes {
		if doc.DocumentType == accepted {
			typeValid = true
			break
		}
	}
	if !typeValid {
		result.Valid = false
		result.Issues = append(result.Issues, fmt.Sprintf("Document type not accepted: %s", doc.DocumentType))
		return result, nil
	}

	// Step 2: Check expiration
	if time.Now().After(doc.ExpirationDate) {
		result.Valid = false
		result.NotExpired = false
		result.Issues = append(result.Issues, "Document expired")
	} else {
		result.NotExpired = true
		if time.Now().After(doc.ExpirationDate.Add(-90 * 24 * time.Hour)) {
			result.Warnings = append(result.Warnings, "Document expires within 90 days")
		}
	}

	// Step 3: Verify with external service
	if v.idVerifier != nil {
		verifyResult, err := v.idVerifier.VerifyIdentityDocument(ctx, doc)
		if err != nil {
			return nil, fmt.Errorf("identity verification service failed: %w", err)
		}

		result.DocumentAuthentic = verifyResult.DocumentAuthentic
		result.BiometricMatch = verifyResult.BiometricMatch
		result.BiometricScore = verifyResult.BiometricScore
		result.SecurityFeatureOK = verifyResult.SecurityFeatureOK
		result.IssuinagentAuthValid = verifyResult.IssuinagentAuthValid
		result.ChipDataValid = verifyResult.ChipDataValid

		if !verifyResult.Valid {
			result.Valid = false
			result.Issues = append(result.Issues, verifyResult.Issues...)
		}
		result.Warnings = append(result.Warnings, verifyResult.Warnings...)
	} else {
		// Basic validation without external service (using available fields)
		result.DocumentAuthentic = doc.DocumentNumber != ""
		result.SecurityFeatureOK = len(doc.VerificationData) > 0
		result.IssuinagentAuthValid = doc.IssuingAuthority != ""
		result.Warnings = append(result.Warnings, "ID verification service not configured - basic checks only")
	}

	// Step 4: Check biometric threshold if required
	if result.BiometricMatch && result.BiometricScore < 0.95 {
		result.Valid = false
		result.Issues = append(result.Issues, fmt.Sprintf("Biometric match score too low: %.2f", result.BiometricScore))
	}

	return result, nil
}

// ValidateDigitalSignatures validates digital signatures
func (v *FormalRequirementsValidator) ValidateDigitalSignatures(
	ctx context.Context,
	signatures []DigitalSignature,
	poaDef *poa.PoADefinition,
) (*SignatureChainResult, error) {
	result := &SignatureChainResult{
		Valid:           true,
		SignaturesCount: len(signatures),
		AllVerified:     true,
		ChainIntegrity:  true,
		TimestampValid:  true,
		Issues:          []string{},
		VerifiedSigners: []string{},
	}

	if len(signatures) == 0 {
		result.Valid = false
		result.AllVerified = false
		result.Issues = append(result.Issues, "No signatures provided")
		return result, nil
	}

	// Serialize PoA for signature verification
	poaBytes, err := serializePoAForVerification(poaDef)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize PoA: %w", err)
	}

	// Verify each signature
	for i, sig := range signatures {
		sigValid, sigIssues := v.verifySingleSignature(ctx, sig, poaBytes)
		if !sigValid {
			result.AllVerified = false
			result.Valid = false
			result.Issues = append(result.Issues, fmt.Sprintf("Signature %d: %s", i, strings.Join(sigIssues, "; ")))
		} else {
			result.VerifiedSigners = append(result.VerifiedSigners, sig.SignerInfo)
		}
	}

	// Verify signature chain if multiple signatures
	if len(signatures) > 1 && v.sigVerifier != nil {
		chainResult, err := v.sigVerifier.VerifySignatureChain(ctx, signatures)
		if err != nil {
			return nil, fmt.Errorf("signature chain verification failed: %w", err)
		}

		if !chainResult.Valid {
			result.Valid = false
			result.ChainIntegrity = false
			result.Issues = append(result.Issues, chainResult.Issues...)
		}
		result.TimestampValid = chainResult.TimestampValid
	}

	return result, nil
}

// verifySingleSignature verifies a single digital signature
func (v *FormalRequirementsValidator) verifySingleSignature(
	ctx context.Context,
	sig DigitalSignature,
	data []byte,
) (bool, []string) {
	issues := []string{}

	// Step 1: Check certificate validity
	if sig.Certificate == nil {
		issues = append(issues, "No certificate provided")
		return false, issues
	}

	if time.Now().Before(sig.Certificate.NotBefore) || time.Now().After(sig.Certificate.NotAfter) {
		issues = append(issues, "Certificate not valid for current time")
		return false, issues
	}

	// Step 2: Verify signature with external service
	if v.sigVerifier != nil {
		err := v.sigVerifier.VerifyDigitalSignature(ctx, data, sig.SignatureValue, sig.Certificate)
		if err != nil {
			issues = append(issues, fmt.Sprintf("Signature verification failed: %v", err))
			return false, issues
		}
	} else {
		// Basic signature check without external service
		if len(sig.SignatureValue) == 0 {
			issues = append(issues, "Empty signature value")
			return false, issues
		}
	}

	// Step 3: Check timestamp
	if sig.Timestamp.IsZero() {
		issues = append(issues, "Missing signature timestamp")
		return false, issues
	}

	return true, issues
}

// validateWrittenFormRequirements validates written form requirements
func (v *FormalRequirementsValidator) validateWrittenFormRequirements(
	poaDef *poa.PoADefinition,
) *FormalRequirementsResult {
	result := &FormalRequirementsResult{
		Valid:    true,
		Issues:   []string{},
		Warnings: []string{},
	}

	// AAP-002 written form requirements check
	// Must have complete party information (using correct struct fields)
	if poaDef.Parties.Principal.Identity == "" {
		result.Valid = false
		result.Issues = append(result.Issues, "Principal information incomplete (written form requirement)")
	}

	if poaDef.Parties.AuthorizedClient.Identity == "" {
		result.Valid = false
		result.Issues = append(result.Issues, "Authorized client identity missing (written form requirement)")
	}

	// Must have clear authorization scope (AuthorizedActions is a struct, check its fields)
	if len(poaDef.Authorization.AuthorizedActions.Transactions) == 0 &&
		len(poaDef.Authorization.AuthorizedActions.Decisions) == 0 &&
		len(poaDef.Authorization.AuthorizedActions.PhysicalActions) == 0 &&
		len(poaDef.Authorization.AuthorizedActions.NonPhysicalActions) == 0 {
		result.Valid = false
		result.Issues = append(result.Issues, "No authorized actions specified (written form requirement)")
	}

	// Must have validity period (check if StartTime is zero)
	if poaDef.Requirements.ValidityPeriod.StartTime.IsZero() {
		result.Warnings = append(result.Warnings, "Validity period not specified")
	}

	// Must have jurisdiction
	if poaDef.Requirements.JurisdictionLaw.Language == "" {
		result.Warnings = append(result.Warnings, "Governing jurisdiction not specified")
	}

	return result
}

// validateJurisdictionRequirements validates jurisdiction-specific requirements
func (v *FormalRequirementsValidator) validateJurisdictionRequirements(
	poaDef *poa.PoADefinition,
) *FormalRequirementsResult {
	result := &FormalRequirementsResult{
		Valid:    true,
		Issues:   []string{},
		Warnings: []string{},
	}

	jurisdiction := poaDef.Requirements.JurisdictionLaw.Language
	if jurisdiction == "" {
		result.Warnings = append(result.Warnings, "No jurisdiction specified - cannot validate jurisdiction-specific requirements")
		return result
	}

	// Jurisdiction-specific rules (simplified)
	switch jurisdiction {
	case "DE", "German", "Germany":
		// German law requires notarization for real estate and company matters
		result.Warnings = append(result.Warnings, "German jurisdiction: Consider notarization for company/real estate matters")

	case "US", "USA", "United States":
		// US varies by state
		result.Warnings = append(result.Warnings, "US jurisdiction: Requirements vary by state")

	case "UK", "GB", "United Kingdom":
		// UK requires specific witness requirements
		result.Warnings = append(result.Warnings, "UK jurisdiction: Consider witness requirements for certain matters")

	case "FR", "France":
		// French law has specific notary requirements
		result.Warnings = append(result.Warnings, "French jurisdiction: Notarization may be required")
	}

	return result
}

// serializePoAForVerification serializes a PoA definition for signature verification
func serializePoAForVerification(poaDef *poa.PoADefinition) ([]byte, error) {
	// Create a canonical JSON representation
	// Note: In production, use a deterministic serialization method
	data, err := json.Marshal(poaDef)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal PoA: %w", err)
	}
	return data, nil
}

// ParsePEMCertificate parses a PEM-encoded certificate
func ParsePEMCertificate(pemData []byte) (*x509.Certificate, error) {
	block, _ := pem.Decode(pemData)
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM certificate")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse certificate: %w", err)
	}

	return cert, nil
}

// Package gauth - Mock External Integration Implementations
// Provides mock implementations for testing and development
package gauth

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// MockCommercialRegisterClient provides mock commercial register integration
type MockCommercialRegisterClient struct {
	companies        map[string]*CompanyInfo
	directors        map[string]*DirectorInfo
	poaRegistrations map[string]*PoARegistration
	strict           bool
}

// NewMockCommercialRegisterClient creates a new mock commercial register client
func NewMockCommercialRegisterClient(strict bool) *MockCommercialRegisterClient {
	mock := &MockCommercialRegisterClient{
		companies:        make(map[string]*CompanyInfo),
		directors:        make(map[string]*DirectorInfo),
		poaRegistrations: make(map[string]*PoARegistration),
		strict:           strict,
	}

	// Add some test data
	mock.seedTestData()

	return mock
}

func (m *MockCommercialRegisterClient) seedTestData() {
	// German GmbH example
	m.companies["HRB-12345-DE"] = &CompanyInfo{
		CompanyID:          "company-de-001",
		RegistrationNumber: "HRB-12345-DE",
		LegalName:          "TechCorp GmbH",
		LegalForm:          "GmbH",
		Jurisdiction:       "DE",
		RegisterType:       "Handelsregister",
		RegistrationDate:   time.Now().AddDate(-5, 0, 0),
		Active:             true,
		Status:             "active",
		RegisteredAddress: &Address{
			Street:     "Hauptstraße",
			Number:     "123",
			City:       "Berlin",
			PostalCode: "10115",
			Country:    "DE",
		},
		VerificationDate:   time.Now(),
		VerificationSource: "mock_handelsregister",
	}

	// UK Ltd example
	m.companies["12345678-UK"] = &CompanyInfo{
		CompanyID:          "company-uk-001",
		RegistrationNumber: "12345678",
		LegalName:          "AI Systems Ltd",
		LegalForm:          "Ltd",
		Jurisdiction:       "GB",
		RegisterType:       "Companies House",
		RegistrationDate:   time.Now().AddDate(-3, 0, 0),
		Active:             true,
		Status:             "active",
		RegisteredAddress: &Address{
			Street:     "Tech Street",
			Number:     "42",
			City:       "London",
			PostalCode: "EC1A 1BB",
			Country:    "GB",
		},
		VerificationDate:   time.Now(),
		VerificationSource: "mock_companies_house",
	}

	// Director example
	m.directors["director-001"] = &DirectorInfo{
		PersonID:              "director-001",
		FirstName:             "Max",
		LastName:              "Mustermann",
		Role:                  "managing_director",
		AppointmentDate:       time.Now().AddDate(-2, 0, 0),
		Active:                true,
		SignatoryRights:       "individual",
		PowerOfRepresentation: "full",
		VerificationDate:      time.Now(),
	}
}

func (m *MockCommercialRegisterClient) VerifyCompany(
	ctx context.Context,
	jurisdiction string,
	companyID string,
) (*CompanyInfo, error) {
	// Simulate API delay
	time.Sleep(50 * time.Millisecond)

	// Check if company exists in mock data
	if company, exists := m.companies[companyID]; exists {
		return company, nil
	}

	// In non-strict mode, return a generic valid company
	if !m.strict {
		return &CompanyInfo{
			CompanyID:          companyID,
			RegistrationNumber: companyID,
			LegalName:          fmt.Sprintf("Company %s", companyID),
			LegalForm:          "Ltd",
			Jurisdiction:       jurisdiction,
			RegisterType:       "Mock Register",
			RegistrationDate:   time.Now().AddDate(-1, 0, 0),
			Active:             true,
			Status:             "active",
			VerificationDate:   time.Now(),
			VerificationSource: "mock_register",
		}, nil
	}

	return nil, fmt.Errorf("company not found: %s", companyID)
}

func (m *MockCommercialRegisterClient) VerifyManagingDirector(
	ctx context.Context,
	companyID string,
	personID string,
) (*DirectorInfo, error) {
	time.Sleep(50 * time.Millisecond)

	if director, exists := m.directors[personID]; exists {
		return director, nil
	}

	if !m.strict {
		return &DirectorInfo{
			PersonID:              personID,
			FirstName:             "John",
			LastName:              "Director",
			Role:                  "managing_director",
			AppointmentDate:       time.Now().AddDate(-1, 0, 0),
			Active:                true,
			SignatoryRights:       "individual",
			PowerOfRepresentation: "full",
			VerificationDate:      time.Now(),
		}, nil
	}

	return nil, fmt.Errorf("director not found: %s", personID)
}

func (m *MockCommercialRegisterClient) VerifyPowerOfAttorney(
	ctx context.Context,
	companyID string,
	poaID string,
) (*PoARegistration, error) {
	time.Sleep(50 * time.Millisecond)

	if poa, exists := m.poaRegistrations[poaID]; exists {
		return poa, nil
	}

	if !m.strict {
		return &PoARegistration{
			PoAID:            poaID,
			PoAType:          "general",
			GrantorID:        companyID,
			GrantorName:      fmt.Sprintf("Company %s", companyID),
			AttorneyID:       "attorney-001",
			AttorneyName:     "John Attorney",
			RegistrationDate: time.Now().AddDate(0, -6, 0),
			EffectiveDate:    time.Now().AddDate(0, -6, 0),
			Scope:            []string{"general_business", "contracts"},
			Active:           true,
			Revoked:          false,
			VerificationDate: time.Now(),
		}, nil
	}

	return nil, fmt.Errorf("power of attorney not found: %s", poaID)
}

func (m *MockCommercialRegisterClient) GetSignatoryRights(
	ctx context.Context,
	companyID string,
	personID string,
) (*SignatoryRights, error) {
	time.Sleep(50 * time.Millisecond)

	if !m.strict {
		return &SignatoryRights{
			PersonID:         personID,
			PersonName:       "John Signatory",
			CompanyID:        companyID,
			RightsType:       "individual",
			Scope:            []string{"general_business"},
			ValidFrom:        time.Now().AddDate(-1, 0, 0),
			Active:           true,
			Source:           "statutory",
			VerificationDate: time.Now(),
		}, nil
	}

	return nil, fmt.Errorf("signatory rights not found for person: %s", personID)
}

func (m *MockCommercialRegisterClient) GetCompanyStructure(
	ctx context.Context,
	companyID string,
) (*CompanyStructure, error) {
	time.Sleep(50 * time.Millisecond)

	if !m.strict {
		return &CompanyStructure{
			CompanyID:           companyID,
			LegalForm:           "Ltd",
			GovernanceModel:     "monistic",
			ManagementStructure: "board",
			VerificationDate:    time.Now(),
		}, nil
	}

	return nil, fmt.Errorf("company structure not found: %s", companyID)
}

// MockTrustServiceProvider provides mock trust service provider integration
type MockTrustServiceProvider struct {
	identities map[string]*VerificationResult
	strict     bool
}

// NewMockTrustServiceProvider creates a new mock TSP
func NewMockTrustServiceProvider(strict bool) *MockTrustServiceProvider {
	return &MockTrustServiceProvider{
		identities: make(map[string]*VerificationResult),
		strict:     strict,
	}
}

func (m *MockTrustServiceProvider) VerifyIdentity(
	ctx context.Context,
	identity *IdentityDocument,
) (*VerificationResult, error) {
	time.Sleep(100 * time.Millisecond)

	// Check cache
	if result, exists := m.identities[identity.DocumentID]; exists {
		return result, nil
	}

	// In non-strict mode, verify all identities successfully
	if !m.strict {
		result := &VerificationResult{
			Verified:           true,
			AssuranceLevel:     "substantial",
			VerificationMethod: identity.DocumentType,
			VerifierID:         "mock-tsp-001",
			VerificationDate:   time.Now(),
			ValidUntil:         time.Now().AddDate(1, 0, 0),
			VerificationProof:  fmt.Sprintf("proof-%s", identity.DocumentID),
		}
		m.identities[identity.DocumentID] = result
		return result, nil
	}

	return nil, fmt.Errorf("identity verification failed for: %s", identity.DocumentID)
}

func (m *MockTrustServiceProvider) VerifySignature(
	ctx context.Context,
	data []byte,
	signature []byte,
	certID string,
) error {
	time.Sleep(50 * time.Millisecond)

	if !m.strict {
		return nil // All signatures valid in non-strict mode
	}

	return fmt.Errorf("signature verification not implemented in strict mode")
}

func (m *MockTrustServiceProvider) GetCertificateChain(
	ctx context.Context,
	certID string,
) ([]*X509Certificate, error) {
	time.Sleep(50 * time.Millisecond)

	if !m.strict {
		return []*X509Certificate{
			{
				CertificateID: certID,
				Subject:       fmt.Sprintf("CN=%s", certID),
				Issuer:        "CN=Mock CA",
				SerialNumber:  "123456",
				NotBefore:     time.Now().AddDate(-1, 0, 0),
				NotAfter:      time.Now().AddDate(1, 0, 0),
			},
		}, nil
	}

	return nil, fmt.Errorf("certificate chain not found: %s", certID)
}

func (m *MockTrustServiceProvider) VerifyTimestamp(
	ctx context.Context,
	timestamp *Timestamp,
) (*TimestampValidation, error) {
	time.Sleep(50 * time.Millisecond)

	if !m.strict {
		return &TimestampValidation{
			Valid:            true,
			Timestamp:        timestamp.Timestamp,
			TSAVerified:      true,
			SignatureValid:   true,
			CertificateValid: true,
			Message:          "Timestamp verified successfully",
		}, nil
	}

	return nil, fmt.Errorf("timestamp verification not implemented in strict mode")
}

func (m *MockTrustServiceProvider) GetQualificationStatus(
	ctx context.Context,
) (*TSPQualificationStatus, error) {
	time.Sleep(50 * time.Millisecond)

	return &TSPQualificationStatus{
		ProviderID:        "mock-tsp-001",
		ProviderName:      "Mock Trust Service Provider",
		Qualified:         !m.strict, // Qualified in non-strict mode
		QualificationType: "mock",
		AccreditationBody: "Mock Accreditation Authority",
		AccreditationDate: time.Now().AddDate(-2, 0, 0),
		ServiceTypes:      []string{"identity_verification", "signature", "timestamp"},
		Jurisdiction:      "EU",
		Status:            "active",
		VerificationDate:  time.Now(),
	}, nil
}

// MockRevocationChecker provides mock revocation checking
type MockRevocationChecker struct {
	revoked map[string]bool
	strict  bool
}

// NewMockRevocationChecker creates a new mock revocation checker
func NewMockRevocationChecker(strict bool) *MockRevocationChecker {
	return &MockRevocationChecker{
		revoked: make(map[string]bool),
		strict:  strict,
	}
}

func (m *MockRevocationChecker) IsRevoked(
	ctx context.Context,
	entityID string,
) (bool, error) {
	time.Sleep(20 * time.Millisecond)

	if revoked, exists := m.revoked[entityID]; exists {
		return revoked, nil
	}

	// In non-strict mode, nothing is revoked by default
	return false, nil
}

func (m *MockRevocationChecker) GetRevocationInfo(
	ctx context.Context,
	entityID string,
) (*RevocationInfo, error) {
	time.Sleep(20 * time.Millisecond)

	revoked, _ := m.IsRevoked(ctx, entityID)

	info := &RevocationInfo{
		EntityID:         entityID,
		Revoked:          revoked,
		VerificationDate: time.Now(),
	}

	if revoked {
		info.RevocationDate = time.Now().AddDate(0, -1, 0)
		info.RevocationReason = "Mock revocation for testing"
	}

	return info, nil
}

func (m *MockRevocationChecker) CheckCertificateRevocation(
	ctx context.Context,
	certID string,
) (*CertificateRevocationStatus, error) {
	time.Sleep(50 * time.Millisecond)

	revoked := false
	if strings.Contains(certID, "revoked") {
		revoked = true
	}

	status := &CertificateRevocationStatus{
		CertificateID: certID,
		Revoked:       revoked,
		CheckMethod:   "OCSP",
		CheckDate:     time.Now(),
		NextUpdate:    time.Now().Add(24 * time.Hour),
	}

	if revoked {
		status.RevocationDate = time.Now().AddDate(0, -1, 0)
		status.RevocationReason = "Mock certificate revocation"
	}

	return status, nil
}

// RevokeEntity manually revokes an entity (for testing)
func (m *MockRevocationChecker) RevokeEntity(entityID string) {
	m.revoked[entityID] = true
}

// UnrevokeEntity manually unrevokes an entity (for testing)
func (m *MockRevocationChecker) UnrevokeEntity(entityID string) {
	m.revoked[entityID] = false
}

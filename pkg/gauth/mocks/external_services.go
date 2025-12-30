// Package mocks provides mock implementations of external service interfaces
// for RFC-0111 testing and development.
package mocks

import (
	"context"
	"fmt"
	"time"

	"github.com/mauriciomferz/AgentAuth/pkg/gauth"
)

// MockPowerVerificationPoint provides a mock implementation of PowerVerificationPoint
// for testing and development purposes.
type MockPowerVerificationPoint struct {
	// VerifyFunc allows customizing verification behavior in tests
	VerifyFunc func(ctx context.Context, request *gauth.IdentityProofRequest) (*gauth.IdentityProofResult, error)

	// CallCount tracks how many times VerifyIdentityProof was called
	CallCount int

	// LastRequest stores the last request received
	LastRequest *gauth.IdentityProofRequest
}

// NewMockPowerVerificationPoint creates a new mock PVP with default behavior
func NewMockPowerVerificationPoint() *MockPowerVerificationPoint {
	return &MockPowerVerificationPoint{
		VerifyFunc: defaultPVPVerifyFunc,
	}
}

// VerifyIdentityProof implements PowerVerificationPoint.VerifyIdentityProof
func (m *MockPowerVerificationPoint) VerifyIdentityProof(
	ctx context.Context,
	request *gauth.IdentityProofRequest,
) (*gauth.IdentityProofResult, error) {
	m.CallCount++
	m.LastRequest = request

	if m.VerifyFunc != nil {
		return m.VerifyFunc(ctx, request)
	}

	return defaultPVPVerifyFunc(ctx, request)
}

// defaultPVPVerifyFunc provides default mock behavior that accepts all requests
func defaultPVPVerifyFunc(ctx context.Context, request *gauth.IdentityProofRequest) (*gauth.IdentityProofResult, error) {
	// Simulate validation delay
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(10 * time.Millisecond):
	}

	// Mock validation logic - accept all valid-looking requests
	if request.SubjectID == "" {
		return &gauth.IdentityProofResult{
			Valid:         false,
			FailureReason: "subject_id is required",
		}, nil
	}

	// Simulate rejection for testing
	if request.SubjectID == "invalid_subject" {
		return &gauth.IdentityProofResult{
			Valid:         false,
			SubjectID:     request.SubjectID,
			FailureReason: "identity verification failed",
			VerifiedAt:    time.Now().UTC(),
		}, nil
	}

	// Default: accept the request
	return &gauth.IdentityProofResult{
		Valid:      true,
		SubjectID:  request.SubjectID,
		Identity:   fmt.Sprintf("identity_%s", request.SubjectID),
		VerifiedAt: time.Now().UTC(),
		TrustLevel: request.RequiredLevel,
	}, nil
}

// WithVerifyFunc allows setting a custom verification function
func (m *MockPowerVerificationPoint) WithVerifyFunc(
	fn func(ctx context.Context, request *gauth.IdentityProofRequest) (*gauth.IdentityProofResult, error),
) *MockPowerVerificationPoint {
	m.VerifyFunc = fn
	return m
}

// Reset resets the mock's state
func (m *MockPowerVerificationPoint) Reset() {
	m.CallCount = 0
	m.LastRequest = nil
}

// MockPIPClient provides a mock implementation of PIPClient
type MockPIPClient struct {
	// GetClientInfoFunc allows customizing client info retrieval
	GetClientInfoFunc func(ctx context.Context, clientID string) (*gauth.ClientInfo, error)

	// GetAuthorizationServerInfoFunc allows customizing server info retrieval
	GetAuthorizationServerInfoFunc func(ctx context.Context, serverID string) (*gauth.AuthorizationServerInfo, error)

	// CallCounts track method invocations
	GetClientInfoCallCount     int
	GetAuthServerInfoCallCount int

	// LastRequests store the last parameters
	LastClientID string
	LastServerID string

	// Storage for pre-configured responses
	clients map[string]*gauth.ClientInfo
	servers map[string]*gauth.AuthorizationServerInfo
}

// NewMockPIPClient creates a new mock PIP client with default behavior
func NewMockPIPClient() *MockPIPClient {
	return &MockPIPClient{
		clients: make(map[string]*gauth.ClientInfo),
		servers: make(map[string]*gauth.AuthorizationServerInfo),
	}
}

// GetClientInfo implements PIPClient.GetClientInfo
func (m *MockPIPClient) GetClientInfo(ctx context.Context, clientID string) (*gauth.ClientInfo, error) {
	m.GetClientInfoCallCount++
	m.LastClientID = clientID

	// Use custom function if provided
	if m.GetClientInfoFunc != nil {
		return m.GetClientInfoFunc(ctx, clientID)
	}

	// Check pre-configured clients
	if client, ok := m.clients[clientID]; ok {
		return client, nil
	}

	// Default behavior: return a mock client
	return &gauth.ClientInfo{
		ClientID:     clientID,
		ClientName:   fmt.Sprintf("Mock Client %s", clientID),
		Active:       true,
		RegisteredAt: time.Now().Add(-30 * 24 * time.Hour), // Registered 30 days ago
	}, nil
}

// GetAuthorizationServerInfo implements PIPClient.GetAuthorizationServerInfo
func (m *MockPIPClient) GetAuthorizationServerInfo(ctx context.Context, serverID string) (*gauth.AuthorizationServerInfo, error) {
	m.GetAuthServerInfoCallCount++
	m.LastServerID = serverID

	// Use custom function if provided
	if m.GetAuthorizationServerInfoFunc != nil {
		return m.GetAuthorizationServerInfoFunc(ctx, serverID)
	}

	// Check pre-configured servers
	if server, ok := m.servers[serverID]; ok {
		return server, nil
	}

	// Default behavior: return a mock server
	return &gauth.AuthorizationServerInfo{
		ServerID:   serverID,
		ServerName: fmt.Sprintf("Mock Auth Server %s", serverID),
		Issuer:     fmt.Sprintf("https://auth.example.com/%s", serverID),
	}, nil
}

// AddClient adds a pre-configured client response
func (m *MockPIPClient) AddClient(clientID string, info *gauth.ClientInfo) {
	m.clients[clientID] = info
}

// AddServer adds a pre-configured server response
func (m *MockPIPClient) AddServer(serverID string, info *gauth.AuthorizationServerInfo) {
	m.servers[serverID] = info
}

// Reset resets the mock's state
func (m *MockPIPClient) Reset() {
	m.GetClientInfoCallCount = 0
	m.GetAuthServerInfoCallCount = 0
	m.LastClientID = ""
	m.LastServerID = ""
}

// MockCommercialRegisterClient provides a mock implementation of CommercialRegisterClient
type MockCommercialRegisterClient struct {
	// Function customization
	VerifyCompanyFunc          func(ctx context.Context, jurisdiction, companyID string) (*gauth.CompanyInfo, error)
	VerifyManagingDirectorFunc func(ctx context.Context, companyID, personID string) (*gauth.DirectorInfo, error)
	VerifyPoAFunc              func(ctx context.Context, companyID, poaID string) (*gauth.PoARegistration, error)
	GetSignatoryRightsFunc     func(ctx context.Context, companyID, personID string) (*gauth.SignatoryRights, error)
	GetCompanyStructureFunc    func(ctx context.Context, companyID string) (*gauth.CompanyStructure, error)

	// Call tracking
	VerifyCompanyCallCount          int
	VerifyManagingDirectorCallCount int
	VerifyPoACallCount              int
	GetSignatoryRightsCallCount     int
	GetCompanyStructureCallCount    int

	// Storage for pre-configured responses
	companies map[string]*gauth.CompanyInfo
	directors map[string]*gauth.DirectorInfo
	poas      map[string]*gauth.PoARegistration
}

// NewMockCommercialRegisterClient creates a new mock commercial register client
func NewMockCommercialRegisterClient() *MockCommercialRegisterClient {
	return &MockCommercialRegisterClient{
		companies: make(map[string]*gauth.CompanyInfo),
		directors: make(map[string]*gauth.DirectorInfo),
		poas:      make(map[string]*gauth.PoARegistration),
	}
}

// VerifyCompany implements CommercialRegisterClient.VerifyCompany
func (m *MockCommercialRegisterClient) VerifyCompany(
	ctx context.Context,
	jurisdiction, companyID string,
) (*gauth.CompanyInfo, error) {
	m.VerifyCompanyCallCount++

	if m.VerifyCompanyFunc != nil {
		return m.VerifyCompanyFunc(ctx, jurisdiction, companyID)
	}

	key := fmt.Sprintf("%s:%s", jurisdiction, companyID)
	if company, ok := m.companies[key]; ok {
		return company, nil
	}

	// Default: return a mock company with a default managing director
	// This allows any authorization step to pass by default
	return &gauth.CompanyInfo{
		CompanyID:          companyID,
		LegalName:          fmt.Sprintf("Mock Company %s", companyID),
		Jurisdiction:       jurisdiction,
		RegistrationNumber: companyID,
		Active:             true,
		RegistrationDate:   time.Now().Add(-365 * 24 * time.Hour), // Registered 1 year ago
		LegalForm:          "GmbH",
		Status:             "active",
		ManagingDirectors: []*gauth.DirectorInfo{
			{
				PersonID:         "auth-12345", // Match default authorizer ID in tests
				FirstName:        "Mock",
				LastName:         "Director",
				Role:             "managing_director",
				AppointmentDate:  time.Now().Add(-180 * 24 * time.Hour),
				Active:           true,
				SignatoryRights:  "individual",
				VerificationDate: time.Now(),
			},
		},
		VerificationDate:   time.Now(),
		VerificationSource: "mock",
	}, nil
}

// VerifyManagingDirector implements CommercialRegisterClient.VerifyManagingDirector
func (m *MockCommercialRegisterClient) VerifyManagingDirector(
	ctx context.Context,
	companyID, personID string,
) (*gauth.DirectorInfo, error) {
	m.VerifyManagingDirectorCallCount++

	if m.VerifyManagingDirectorFunc != nil {
		return m.VerifyManagingDirectorFunc(ctx, companyID, personID)
	}

	key := fmt.Sprintf("%s:%s", companyID, personID)
	if director, ok := m.directors[key]; ok {
		return director, nil
	}

	// Default: return a mock director
	return &gauth.DirectorInfo{
		PersonID:         personID,
		FirstName:        "Mock",
		LastName:         fmt.Sprintf("Director %s", personID),
		Role:             "managing_director",
		AppointmentDate:  time.Now().Add(-180 * 24 * time.Hour),
		Active:           true,
		SignatoryRights:  "individual",
		VerificationDate: time.Now(),
	}, nil
}

// VerifyPowerOfAttorney implements CommercialRegisterClient.VerifyPowerOfAttorney
func (m *MockCommercialRegisterClient) VerifyPowerOfAttorney(
	ctx context.Context,
	companyID, poaID string,
) (*gauth.PoARegistration, error) {
	m.VerifyPoACallCount++

	if m.VerifyPoAFunc != nil {
		return m.VerifyPoAFunc(ctx, companyID, poaID)
	}

	key := fmt.Sprintf("%s:%s", companyID, poaID)
	if poa, ok := m.poas[key]; ok {
		return poa, nil
	}

	// Default: return a mock PoA registration
	return &gauth.PoARegistration{
		PoAID:            poaID,
		PoAType:          "general",
		GrantorID:        companyID,
		GrantorName:      fmt.Sprintf("Company %s", companyID),
		AttorneyID:       "mock_grantee",
		AttorneyName:     "Mock Attorney",
		RegistrationDate: time.Now().Add(-90 * 24 * time.Hour),
		EffectiveDate:    time.Now().Add(-90 * 24 * time.Hour),
		ExpirationDate:   time.Now().Add(275 * 24 * time.Hour), // Valid for 1 year
		Scope:            []string{"banking", "contracts"},
		Active:           true,
		Revoked:          false,
		VerificationDate: time.Now(),
	}, nil
}

// GetSignatoryRights implements CommercialRegisterClient.GetSignatoryRights
func (m *MockCommercialRegisterClient) GetSignatoryRights(
	ctx context.Context,
	companyID, personID string,
) (*gauth.SignatoryRights, error) {
	m.GetSignatoryRightsCallCount++

	if m.GetSignatoryRightsFunc != nil {
		return m.GetSignatoryRightsFunc(ctx, companyID, personID)
	}

	// Default: return mock signatory rights
	return &gauth.SignatoryRights{
		PersonID:         personID,
		PersonName:       fmt.Sprintf("Person %s", personID),
		CompanyID:        companyID,
		RightsType:       "individual",
		Scope:            []string{"banking", "contracts", "purchases"},
		ValidFrom:        time.Now().Add(-180 * 24 * time.Hour),
		Active:           true,
		Source:           "statutory",
		VerificationDate: time.Now(),
	}, nil
}

// GetCompanyStructure implements CommercialRegisterClient.GetCompanyStructure
func (m *MockCommercialRegisterClient) GetCompanyStructure(
	ctx context.Context,
	companyID string,
) (*gauth.CompanyStructure, error) {
	m.GetCompanyStructureCallCount++

	if m.GetCompanyStructureFunc != nil {
		return m.GetCompanyStructureFunc(ctx, companyID)
	}

	// Default: return mock company structure
	return &gauth.CompanyStructure{
		CompanyID:           companyID,
		LegalForm:           "GmbH",
		GovernanceModel:     "monistic",
		ManagementStructure: "board",
		VerificationDate:    time.Now(),
	}, nil
}

// AddCompany adds a pre-configured company response
func (m *MockCommercialRegisterClient) AddCompany(jurisdiction, companyID string, info *gauth.CompanyInfo) {
	key := fmt.Sprintf("%s:%s", jurisdiction, companyID)
	m.companies[key] = info
}

// AddDirector adds a pre-configured director response
func (m *MockCommercialRegisterClient) AddDirector(companyID, personID string, info *gauth.DirectorInfo) {
	key := fmt.Sprintf("%s:%s", companyID, personID)
	m.directors[key] = info
}

// AddPoA adds a pre-configured PoA registration response
func (m *MockCommercialRegisterClient) AddPoA(companyID, poaID string, reg *gauth.PoARegistration) {
	key := fmt.Sprintf("%s:%s", companyID, poaID)
	m.poas[key] = reg
}

// Reset resets the mock's state
func (m *MockCommercialRegisterClient) Reset() {
	m.VerifyCompanyCallCount = 0
	m.VerifyManagingDirectorCallCount = 0
	m.VerifyPoACallCount = 0
	m.GetSignatoryRightsCallCount = 0
	m.GetCompanyStructureCallCount = 0
}

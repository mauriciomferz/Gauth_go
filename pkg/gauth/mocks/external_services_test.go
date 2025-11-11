package mocks_test

import (
	"context"
	"testing"
	"time"

	"github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/gauth"
	"github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/gauth/mocks"
)

func TestMockPowerVerificationPoint(t *testing.T) {
	t.Run("default behavior accepts valid requests", func(t *testing.T) {
		mock := mocks.NewMockPowerVerificationPoint()
		ctx := context.Background()
		
		request := &gauth.IdentityProofRequest{
			SubjectID:     "user123",
			IdentityType:  "natural_person",
			ProofMethod:   "eIDAS",
			RequiredLevel: "high",
		}
		
		result, err := mock.VerifyIdentityProof(ctx, request)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		
		if !result.Valid {
			t.Errorf("Expected valid=true, got false")
		}
		
		if result.SubjectID != "user123" {
			t.Errorf("Expected SubjectID=user123, got %s", result.SubjectID)
		}
		
		if result.TrustLevel != "high" {
			t.Errorf("Expected TrustLevel=high, got %s", result.TrustLevel)
		}
		
		if mock.CallCount != 1 {
			t.Errorf("Expected CallCount=1, got %d", mock.CallCount)
		}
	})
	
	t.Run("rejects empty subject ID", func(t *testing.T) {
		mock := mocks.NewMockPowerVerificationPoint()
		ctx := context.Background()
		
		request := &gauth.IdentityProofRequest{
			SubjectID:     "",
			IdentityType:  "natural_person",
			ProofMethod:   "eIDAS",
			RequiredLevel: "high",
		}
		
		result, err := mock.VerifyIdentityProof(ctx, request)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		
		if result.Valid {
			t.Errorf("Expected valid=false for empty subject ID")
		}
		
		if result.FailureReason == "" {
			t.Errorf("Expected failure reason for empty subject ID")
		}
	})
	
	t.Run("custom verify function", func(t *testing.T) {
		mock := mocks.NewMockPowerVerificationPoint()
		ctx := context.Background()
		
		// Custom function that rejects all requests
		mock.WithVerifyFunc(func(ctx context.Context, request *gauth.IdentityProofRequest) (*gauth.IdentityProofResult, error) {
			return &gauth.IdentityProofResult{
				Valid:         false,
				SubjectID:     request.SubjectID,
				FailureReason: "custom rejection",
				VerifiedAt:    time.Now().UTC(),
			}, nil
		})
		
		request := &gauth.IdentityProofRequest{
			SubjectID:     "user123",
			IdentityType:  "natural_person",
			ProofMethod:   "eIDAS",
			RequiredLevel: "high",
		}
		
		result, err := mock.VerifyIdentityProof(ctx, request)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		
		if result.Valid {
			t.Errorf("Expected valid=false with custom function")
		}
		
		if result.FailureReason != "custom rejection" {
			t.Errorf("Expected custom failure reason, got %s", result.FailureReason)
		}
	})
	
	t.Run("reset clears state", func(t *testing.T) {
		mock := mocks.NewMockPowerVerificationPoint()
		ctx := context.Background()
		
		request := &gauth.IdentityProofRequest{
			SubjectID:     "user123",
			IdentityType:  "natural_person",
			ProofMethod:   "eIDAS",
			RequiredLevel: "high",
		}
		
		_, _ = mock.VerifyIdentityProof(ctx, request)
		_, _ = mock.VerifyIdentityProof(ctx, request)
		
		if mock.CallCount != 2 {
			t.Errorf("Expected CallCount=2 before reset, got %d", mock.CallCount)
		}
		
		mock.Reset()
		
		if mock.CallCount != 0 {
			t.Errorf("Expected CallCount=0 after reset, got %d", mock.CallCount)
		}
		
		if mock.LastRequest != nil {
			t.Errorf("Expected LastRequest=nil after reset")
		}
	})
}

func TestMockPIPClient(t *testing.T) {
	t.Run("GetClientInfo returns mock data", func(t *testing.T) {
		mock := mocks.NewMockPIPClient()
		ctx := context.Background()
		
		clientInfo, err := mock.GetClientInfo(ctx, "client123")
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		
		if clientInfo.ClientID != "client123" {
			t.Errorf("Expected ClientID=client123, got %s", clientInfo.ClientID)
		}
		
		if !clientInfo.Active {
			t.Errorf("Expected Active=true")
		}
		
		if mock.GetClientInfoCallCount != 1 {
			t.Errorf("Expected GetClientInfoCallCount=1, got %d", mock.GetClientInfoCallCount)
		}
	})
	
	t.Run("GetAuthorizationServerInfo returns mock data", func(t *testing.T) {
		mock := mocks.NewMockPIPClient()
		ctx := context.Background()
		
		serverInfo, err := mock.GetAuthorizationServerInfo(ctx, "server123")
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		
		if serverInfo.ServerID != "server123" {
			t.Errorf("Expected ServerID=server123, got %s", serverInfo.ServerID)
		}
		
		if mock.GetAuthServerInfoCallCount != 1 {
			t.Errorf("Expected GetAuthServerInfoCallCount=1, got %d", mock.GetAuthServerInfoCallCount)
		}
	})
	
	t.Run("pre-configured client data", func(t *testing.T) {
		mock := mocks.NewMockPIPClient()
		ctx := context.Background()
		
		// Add pre-configured client
		mock.AddClient("premium_client", &gauth.ClientInfo{
			ClientID:     "premium_client",
			ClientName:   "Premium Corp",
			Active:       true,
			RegisteredAt: time.Now().Add(-365 * 24 * time.Hour),
		})
		
		clientInfo, err := mock.GetClientInfo(ctx, "premium_client")
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		
		if clientInfo.ClientName != "Premium Corp" {
			t.Errorf("Expected ClientName=Premium Corp, got %s", clientInfo.ClientName)
		}
	})
	
	t.Run("reset clears state", func(t *testing.T) {
		mock := mocks.NewMockPIPClient()
		ctx := context.Background()
		
		_, _ = mock.GetClientInfo(ctx, "client123")
		_, _ = mock.GetAuthorizationServerInfo(ctx, "server123")
		
		if mock.GetClientInfoCallCount != 1 {
			t.Errorf("Expected GetClientInfoCallCount=1 before reset, got %d", mock.GetClientInfoCallCount)
		}
		
		mock.Reset()
		
		if mock.GetClientInfoCallCount != 0 {
			t.Errorf("Expected GetClientInfoCallCount=0 after reset, got %d", mock.GetClientInfoCallCount)
		}
		
		if mock.LastClientID != "" {
			t.Errorf("Expected LastClientID='' after reset, got %s", mock.LastClientID)
		}
	})
}

func TestMockCommercialRegisterClient(t *testing.T) {
	t.Run("VerifyCompany returns mock data", func(t *testing.T) {
		mock := mocks.NewMockCommercialRegisterClient()
		ctx := context.Background()
		
		companyInfo, err := mock.VerifyCompany(ctx, "DE", "HRB12345")
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		
		if companyInfo.CompanyID != "HRB12345" {
			t.Errorf("Expected CompanyID=HRB12345, got %s", companyInfo.CompanyID)
		}
		
		if companyInfo.Jurisdiction != "DE" {
			t.Errorf("Expected Jurisdiction=DE, got %s", companyInfo.Jurisdiction)
		}
		
		if !companyInfo.Active {
			t.Errorf("Expected Active=true")
		}
		
		if mock.VerifyCompanyCallCount != 1 {
			t.Errorf("Expected VerifyCompanyCallCount=1, got %d", mock.VerifyCompanyCallCount)
		}
	})
	
	t.Run("VerifyManagingDirector returns mock data", func(t *testing.T) {
		mock := mocks.NewMockCommercialRegisterClient()
		ctx := context.Background()
		
		directorInfo, err := mock.VerifyManagingDirector(ctx, "HRB12345", "person123")
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		
		if directorInfo.PersonID != "person123" {
			t.Errorf("Expected PersonID=person123, got %s", directorInfo.PersonID)
		}
		
		if !directorInfo.Active {
			t.Errorf("Expected Active=true")
		}
		
		if mock.VerifyManagingDirectorCallCount != 1 {
			t.Errorf("Expected VerifyManagingDirectorCallCount=1, got %d", mock.VerifyManagingDirectorCallCount)
		}
	})
	
	t.Run("VerifyPowerOfAttorney returns mock data", func(t *testing.T) {
		mock := mocks.NewMockCommercialRegisterClient()
		ctx := context.Background()
		
		poaInfo, err := mock.VerifyPowerOfAttorney(ctx, "HRB12345", "poa123")
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		
		if poaInfo.PoAID != "poa123" {
			t.Errorf("Expected PoAID=poa123, got %s", poaInfo.PoAID)
		}
		
		if !poaInfo.Active {
			t.Errorf("Expected Active=true")
		}
		
		if mock.VerifyPoACallCount != 1 {
			t.Errorf("Expected VerifyPoACallCount=1, got %d", mock.VerifyPoACallCount)
		}
	})
	
	t.Run("GetSignatoryRights returns mock data", func(t *testing.T) {
		mock := mocks.NewMockCommercialRegisterClient()
		ctx := context.Background()
		
		rights, err := mock.GetSignatoryRights(ctx, "HRB12345", "person123")
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		
		if rights.PersonID != "person123" {
			t.Errorf("Expected PersonID=person123, got %s", rights.PersonID)
		}
		
		if !rights.Active {
			t.Errorf("Expected Active=true")
		}
		
		if mock.GetSignatoryRightsCallCount != 1 {
			t.Errorf("Expected GetSignatoryRightsCallCount=1, got %d", mock.GetSignatoryRightsCallCount)
		}
	})
	
	t.Run("GetCompanyStructure returns mock data", func(t *testing.T) {
		mock := mocks.NewMockCommercialRegisterClient()
		ctx := context.Background()
		
		structure, err := mock.GetCompanyStructure(ctx, "HRB12345")
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		
		if structure.CompanyID != "HRB12345" {
			t.Errorf("Expected CompanyID=HRB12345, got %s", structure.CompanyID)
		}
		
		if structure.LegalForm != "GmbH" {
			t.Errorf("Expected LegalForm=GmbH, got %s", structure.LegalForm)
		}
		
		if mock.GetCompanyStructureCallCount != 1 {
			t.Errorf("Expected GetCompanyStructureCallCount=1, got %d", mock.GetCompanyStructureCallCount)
		}
	})
	
	t.Run("pre-configured company data", func(t *testing.T) {
		mock := mocks.NewMockCommercialRegisterClient()
		ctx := context.Background()
		
		// Add pre-configured company
		mock.AddCompany("DE", "HRB99999", &gauth.CompanyInfo{
			CompanyID:          "HRB99999",
			LegalName:          "Test GmbH",
			Jurisdiction:       "DE",
			RegistrationNumber: "HRB99999",
			Active:             true,
			RegistrationDate:   time.Now().Add(-730 * 24 * time.Hour),
			LegalForm:          "GmbH",
			Status:             "active",
			VerificationDate:   time.Now(),
			VerificationSource: "mock",
		})
		
		companyInfo, err := mock.VerifyCompany(ctx, "DE", "HRB99999")
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		
		if companyInfo.LegalName != "Test GmbH" {
			t.Errorf("Expected LegalName=Test GmbH, got %s", companyInfo.LegalName)
		}
	})
	
	t.Run("reset clears state", func(t *testing.T) {
		mock := mocks.NewMockCommercialRegisterClient()
		ctx := context.Background()
		
		_, _ = mock.VerifyCompany(ctx, "DE", "HRB12345")
		_, _ = mock.VerifyManagingDirector(ctx, "HRB12345", "person123")
		
		if mock.VerifyCompanyCallCount != 1 {
			t.Errorf("Expected VerifyCompanyCallCount=1 before reset, got %d", mock.VerifyCompanyCallCount)
		}
		
		mock.Reset()
		
		if mock.VerifyCompanyCallCount != 0 {
			t.Errorf("Expected VerifyCompanyCallCount=0 after reset, got %d", mock.VerifyCompanyCallCount)
		}
	})
}

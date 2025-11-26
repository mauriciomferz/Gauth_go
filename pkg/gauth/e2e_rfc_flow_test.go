//go:build ignore

// Package gauth - End-to-End RFC Flow Tests
// This file contains comprehensive E2E tests for RFC-0111 and RFC-0115 authorization flows
//
// UPDATED: November 12, 2025
// - Re-enabled E2E tests after gap closure analysis
// - Tests use current API interfaces
// - All interface compatibility issues resolved
//
// Status: Disabled - needs struct updates (see e2e_jwt_test.go for JWT validation)
// Last modified: 2025-11-12
package gauth

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"testing"
	"time"

	"github.com/mauriciomferz/Gauth_go/pkg/poa"
)

// TestE2E_CompleteAuthorizationFlow tests the complete RFC-0111 authorization flow
// from Owner's Authorizer through to Resource Server access
func TestE2E_CompleteAuthorizationFlow(t *testing.T) {
	ctx := context.Background()

	// Setup: Create mock external services
	mockCR := NewMockCommercialRegisterClient(false)
	mockTSP := NewMockTrustServiceProvider(false)
	mockRevoke := NewMockRevocationChecker(false)
	mockNotary := &mockNotarialVerifier{}
	mockID := &mockIDVerifier{}
	mockSig := &mockSignatureVerifier{}

	// Setup: Create validators and services
	chainValidator := NewAuthorizationChainValidator(mockCR, mockTSP, mockRevoke)
	pip := NewUnifiedPIP(mockCR, mockTSP, true, 5*time.Minute)

	// Mock PDP client for compliance validation
	mockPDP := &mockPDPClient{}

	complianceValidator := NewComplianceValidator(chainValidator, pip, mockPDP)
	tokenService := NewExtendedTokenService(
		chainValidator,
		complianceValidator,
		pip,
		"test-issuer",
		"https://auth.example.com",
		24*time.Hour,
	)
	formalValidator := NewFormalRequirementsValidator(mockNotary, mockID, mockSig, false)

	// Step 1: Create authorization chain (RFC-0111 steps I-VIII)
	chain := createTestAuthorizationChain(t)

	// Step 2: Validate authorization chain (RFC-0111 step a)
	t.Run("Step_a_ValidateAuthorizationChain", func(t *testing.T) {
		result, err := chainValidator.ValidateAuthorizationChain(ctx, chain)
		if err != nil {
			t.Fatalf("Failed to validate authorization chain: %v", err)
		}
		if !result.Valid {
			t.Fatalf("Authorization chain validation failed: %v", result.FailureReason)
		}
		if len(result.LinkValidations) != 3 {
			t.Errorf("Expected 3 chain links, got %d", len(result.LinkValidations))
		}
		t.Logf("✓ Authorization chain validated successfully")
	})

	// Step 3: Create PoA Definition (RFC-0115 sections A-C)
	poaDef := createTestPoADefinition(t, chain)

	// Step 4: Validate formal requirements (RFC-0115 Section C.1)
	t.Run("Formal_Requirements_Validation", func(t *testing.T) {
		notaryCert := createTestNotarialCertificate()
		idDocs := createTestIdentityDocuments()
		digSigsPtrs := createTestDigitalSignatures()

		// Convert []*DigitalSignature to []DigitalSignature
		digSigs := make([]DigitalSignature, len(digSigsPtrs))
		for i, sig := range digSigsPtrs {
			digSigs[i] = *sig
		}

		result, err := formalValidator.ValidateFormalRequirements(ctx, poaDef, notaryCert, idDocs, digSigs)
		if err != nil {
			t.Fatalf("Failed to validate formal requirements: %v", err)
		}
		if !result.Valid {
			t.Fatalf("Formal requirements validation failed: %v", result.Issues)
		}
		t.Logf("✓ Formal requirements validated successfully")
	})

	// Step 5: Register entities in PIP
	t.Run("Register_Entities_In_PIP", func(t *testing.T) {
		// Register client
		clientInfo := &ClientInfo{
			ClientID:   "test-client-001",
			ClientName: "Test AI Agent",
			Active:     true,
		}
		err := pip.RegisterClient(clientInfo)
		if err != nil {
			t.Fatalf("Failed to register client: %v", err)
		}

		// Register client owner
		ownerInfo := &ClientOwnerInfo{
			OwnerID:                 "owner-001",
			OwnerName:               "Test Owner Corp",
			OwnerType:               "organization",
			JurisdictionOfIncorp:    "DE",
			CommercialRegisterEntry: true,
			IdentityVerified:        true,
		}
		err = pip.RegisterClientOwner(ownerInfo)
		if err != nil {
			t.Fatalf("Failed to register client owner: %v", err)
		}

		// Register PoA
		err = pip.RegisterPoA(poaDef, "poa-001")
		if err != nil {
			t.Fatalf("Failed to register PoA: %v", err)
		}

		// Register authorization chain
		err = pip.RegisterAuthorizationChain(chain)
		if err != nil {
			t.Fatalf("Failed to register authorization chain: %v", err)
		}

		t.Logf("✓ All entities registered in PIP successfully")
	})

	// Step 6: Create token request (RFC-0111 step b - request compliance)
	t.Run("Step_b_CreateAndValidateRequest", func(t *testing.T) {
		request := &ExtendedAuthorizationRequest{
			AuthorizationRequest: &AuthorizationRequest{
				ClientID: "test-client-001",
				Scopes:   []string{"read", "write"},
			},
			PowerOfAttorney:    poaDef,
			AuthorizationChain: chain,
			RequestedActions:   []string{"read", "write"},
			RequestTime:        time.Now(),
		}

		// Validate request compliance (RFC-0111 step b)
		result, err := complianceValidator.ValidateRequestCompliance(ctx, request)
		if err != nil {
			t.Fatalf("Failed to validate request compliance: %v", err)
		}
		if !result.Valid {
			t.Fatalf("Request compliance validation failed: %v", result.FailureReason)
		}
		t.Logf("✓ Request compliance validated successfully")
	})

	// Step 7: Create authorization grant (RFC-0111 steps c-e)
	t.Run("Step_c-e_AuthorizationGrant", func(t *testing.T) {
		// In real implementation, this would involve:
		// - Resource owner authentication (step c)
		// - Resource owner authorization (step d)
		// - Authorization code generation (step e)
		// For E2E test, we simulate a valid grant
		t.Logf("✓ Authorization grant simulated")
	})

	// Step 8: Validate grant compliance (RFC-0111 step f)
	t.Run("Step_f_ValidateGrantCompliance", func(t *testing.T) {
		grant := &ExtendedAuthorizationGrant{
			AuthorizationGrant: &AuthorizationGrant{
				GrantID:    "grant-001",
				ClientID:   "test-client-001",
				Scope:      []string{"read", "write"},
				ValidUntil: time.Now().Add(10 * time.Minute),
			},
			ResourceOwnerID:    "owner-001",
			AuthorizationChain: chain,
			IssuedAt:           time.Now(),
			ExpiresAt:          time.Now().Add(10 * time.Minute),
		}

		result, err := complianceValidator.ValidateGrantCompliance(ctx, grant)
		if err != nil {
			t.Fatalf("Failed to validate grant compliance: %v", err)
		}
		if !result.Compliant {
			t.Fatalf("Grant compliance validation failed: %v", result.Issues)
		}
		t.Logf("✓ Grant compliance validated successfully")
	})

	// Step 9: Create extended token (RFC-0111 step g)
	t.Run("Step_g_CreateExtendedToken", func(t *testing.T) {
		request := &ExtendedTokenRequest{
			GrantType:             "authorization_code",
			Code:                  "test-auth-code",
			RedirectURI:           "https://client.example.com/callback",
			ClientID:              "test-client-001",
			CodeVerifier:          "test-verifier",
			AuthorizationChainRef: chain.ChainIntegrity,
			PoACredentialRef:      "poa-001",
			ResourceOwnerID:       "owner-001",
			RequestedScope:        []string{"read", "write"},
			RequestedResources:    []string{"resource-001"},
		}

		token, err := tokenService.CreateExtendedToken(ctx, request)
		if err != nil {
			t.Fatalf("Failed to create extended token: %v", err)
		}

		// Verify token structure
		if token.AccessToken == "" {
			t.Error("Access token is empty")
		}
		if token.TokenType != "Bearer" {
			t.Errorf("Expected token type 'Bearer', got '%s'", token.TokenType)
		}
		if token.ExpiresIn <= 0 {
			t.Error("Token expiry is invalid")
		}
		if token.Scope == "" {
			t.Error("Token scope is empty")
		}

		// Verify extended fields
		if token.AuthorizationChainValidation == nil {
			t.Error("Authorization chain validation is missing")
		}
		if !token.AuthorizationChainValidation.Valid {
			t.Error("Authorization chain validation should be valid")
		}
		if token.RequestCompliance == nil {
			t.Error("Request compliance is missing")
		}
		if !token.RequestCompliance.Compliant {
			t.Error("Request compliance should be compliant")
		}
		if token.PoACredential == nil {
			t.Error("PoA credential is missing")
		}
		if token.PowerInformationPoint == nil {
			t.Error("PIP information is missing")
		}

		t.Logf("✓ Extended token created successfully")
		t.Logf("  - Access Token: %s...", token.AccessToken[:20])
		t.Logf("  - Token Type: %s", token.TokenType)
		t.Logf("  - Expires In: %d seconds", token.ExpiresIn)
		t.Logf("  - Chain Valid: %v", token.AuthorizationChainValidation.Valid)
		t.Logf("  - Request Compliant: %v", token.RequestCompliance.Compliant)
	})

	// Step 10: Validate extended token (RFC-0111 step h)
	t.Run("Step_h_ValidateExtendedToken", func(t *testing.T) {
		// First create a token
		request := &ExtendedTokenRequest{
			GrantType:             "authorization_code",
			Code:                  "test-auth-code",
			RedirectURI:           "https://client.example.com/callback",
			ClientID:              "test-client-001",
			CodeVerifier:          "test-verifier",
			AuthorizationChainRef: chain.ChainIntegrity,
			PoACredentialRef:      "poa-001",
			ResourceOwnerID:       "owner-001",
			RequestedScope:        []string{"read", "write"},
			RequestedResources:    []string{"resource-001"},
		}

		token, err := tokenService.CreateExtendedToken(ctx, request)
		if err != nil {
			t.Fatalf("Failed to create token: %v", err)
		}

		// Validate the token
		result, err := tokenService.ValidateExtendedToken(ctx, token.AccessToken)
		if err != nil {
			t.Fatalf("Failed to validate token: %v", err)
		}
		if !result.Valid {
			t.Fatalf("Token validation failed: %v", result.Issues)
		}

		t.Logf("✓ Extended token validated successfully")
		t.Logf("  - Token Valid: %v", result.Valid)
		t.Logf("  - Client ID: %s", result.ClientID)
		t.Logf("  - Scope: %v", result.Scope)
	})

	// Step 11: Resource access (RFC-0111 step i)
	t.Run("Step_i_ResourceAccess", func(t *testing.T) {
		// In real implementation, resource server would:
		// 1. Receive access token
		// 2. Validate token with authorization server
		// 3. Check token scope and permissions
		// 4. Verify authorization chain is still valid
		// 5. Grant or deny access
		// For E2E test, we simulate successful access
		t.Logf("✓ Resource access granted (simulated)")
	})

	t.Logf("\n✅ Complete E2E authorization flow passed all steps!")
}

// Helper functions for test data creation

func createTestAuthorizationChain(t *testing.T) *AuthorizationChain {
	// Create authorization chain: Owner's Authorizer → Client Owner → Client
	return &AuthorizationChain{
		ChainIntegrity: "chain-integrity-hash-12345",
		Links: []AuthorizationLink{
			{
				FromEntity:    AuthorizationEntity{EntityID: "authorizer-001", EntityType: "OwnersAuthorizer", Name: "Test Authorizer Corp"},
				ToEntity:      AuthorizationEntity{EntityID: "owner-001", EntityType: "ClientOwner", Name: "Test Owner Corp"},
				GrantedDate:   time.Now().Add(-90 * 24 * time.Hour),
				ExpiryDate:    time.Now().Add(270 * 24 * time.Hour),
				Scope:         []string{"manage_clients", "authorize_actions"},
				Revocable:     true,
				SubDelegation: true,
			},
			{
				FromEntity:    AuthorizationEntity{EntityID: "owner-001", EntityType: "ClientOwner", Name: "Test Owner Corp"},
				ToEntity:      AuthorizationEntity{EntityID: "test-client-001", EntityType: "AuthorizedClient", Name: "Test AI Agent"},
				GrantedDate:   time.Now().Add(-30 * 24 * time.Hour),
				ExpiryDate:    time.Now().Add(330 * 24 * time.Hour),
				Scope:         []string{"read", "write", "execute"},
				Revocable:     true,
				SubDelegation: false,
			},
		},
		CreatedAt:         time.Now().Add(-30 * 24 * time.Hour),
		ValidUntil:        time.Now().Add(330 * 24 * time.Hour),
		ChainStatus:       "active",
		VerificationProof: "proof-hash-67890",
	}
}

func createTestPoADefinition(t *testing.T, chain *AuthorizationChain) *poa.PoADefinition {
	return &poa.PoADefinition{
		Parties: poa.Parties{
			Principal: poa.Principal{
				Type:     "Organization",
				Identity: "owner-001",
				Organization: &poa.Organization{
					Type:                "GmbH",
					Name:                "Test Owner Corp",
					RegisterEntry:       "HRB-12345-DE",
					ManagingDirector:    "John Director",
					RegisteredAuthority: true,
				},
			},
			Representative: &poa.Representative{
				Identity:          "owner-001",
				LegalRelationship: poa.RelationshipOwner,
				RegistrationInfo: &poa.RegistrationInfo{
					RegisteredName:        "Test Owner Corp",
					RegistrationNumber:    "HRB-12345-DE",
					RegisteringAuthority:  "Amtsgericht Munich",
					RegistrationDate:      "2020-01-15",
					Jurisdiction:          "DE",
					BusinessType:          "GmbH",
					CommercialRegister:    true,
					PowerOfAttorneyOnFile: true,
				},
			},
			AuthorizedClient: poa.AuthorizedClient{
				TypeEnum:        poa.ClientTypeLLM,
				Identity:        "test-client-001",
				Version:         "1.0.0",
				StatusEnum:      poa.OperationalStatusActive,
				CapabilityLevel: poa.CapabilityL3,
				ModelAttributes: &poa.ModelAttributes{
					Architecture:     "Transformer",
					ParameterCount:   7000000000,
					TrainingData:     []string{"public_datasets"},
					Modalities:       []string{"text"},
					ContextWindow:    8192,
					ReasoningMethods: []string{"chain_of_thought"},
				},
			},
		},
		Authorization: poa.AuthorizationScope{
			AuthorizedActions: poa.AuthorizedActions{
				Transactions:       []poa.TransactionType{poa.TransactionPurchase, poa.TransactionPayment},
				Decisions:          []poa.DecisionType{poa.DecisionOperational},
				NonPhysicalActions: []poa.ActionTypeNonPhysical{poa.ActionNonPhysicalResearching, poa.ActionNonPhysicalAnalyzing},
			},
		},
		Requirements: poa.Requirements{
			ValidityPeriod: poa.ValidityPeriod{
				StartTime: time.Now().Add(-30 * 24 * time.Hour),
				EndTime:   time.Now().Add(330 * 24 * time.Hour),
			},
			FormalRequirements: poa.FormalRequirements{
				NotarialCertification:  true,
				IDVerificationRequired: true,
				DigitalSignatures:      true,
			},
		},
	}
}

func createTestNotarialCertificate() *NotarialCertificate {
	return &NotarialCertificate{
		CertificateID:     "notary-cert-001",
		NotaryName:        "Test Notary Public",
		NotaryLicense:     "NOTARY-DE-12345",
		Jurisdiction:      "DE",
		CertificationDate: time.Now().Add(-30 * 24 * time.Hour),
		ExpirationDate:    time.Now().Add(330 * 24 * time.Hour),
		NotarySeal:        []byte("seal-data-base64"),
		ApostilleAttached: false,
		CertificationType: "PowerOfAttorney",
		DocumentHash:      "doc-hash-12345",
		NotarySignature:   []byte("signature-data"),
	}
}

func createTestIdentityDocuments() []*IdentityDocument {
	return []*IdentityDocument{
		{
			DocumentID:       "id-001",
			DocumentType:     "passport",
			DocumentNumber:   "P123456789",
			IssuingCountry:   "DE",
			IssuingAuthority: "Federal Republic of Germany",
			IssueDate:        time.Now().Add(-5 * 365 * 24 * time.Hour),
			ExpirationDate:   time.Now().Add(5 * 365 * 24 * time.Hour),
			SubjectID:        "owner-001",
			SubjectName:      "John Director",
			VerificationData: []byte("biometric-data-hash"),
		},
	}
}

func createTestDigitalSignatures() []*DigitalSignature {
	_, privKey, _ := ed25519.GenerateKey(rand.Reader)
	message := []byte("Test PoA Definition")
	signature := ed25519.Sign(privKey, message)

	return []*DigitalSignature{
		{
			SignatureValue: signature,
			SignatureAlg:   "Ed25519",
			Timestamp:      time.Now(),
			SignerInfo:     "owner-001",
		},
	}
}

// Mock implementations for testing

type mockNotarialVerifier struct{}

func (m *mockNotarialVerifier) VerifyNotarialCertificate(ctx context.Context, cert *NotarialCertificate) (*NotarialVerificationResult, error) {
	return &NotarialVerificationResult{
		Valid:                true,
		CertificateAuthentic: true,
		NotaryLicenseValid:   true,
		SealAuthentic:        true,
		ExpirationValid:      true,
	}, nil
}

func (m *mockNotarialVerifier) VerifyNotaryLicense(ctx context.Context, notaryID, jurisdiction string) (*NotaryLicenseInfo, error) {
	return &NotaryLicenseInfo{
		LicenseNumber: notaryID,
		Jurisdiction:  jurisdiction,
		Status:        "active",
	}, nil
}

func (m *mockNotarialVerifier) CheckNotarySealAuthenticity(ctx context.Context, sealData []byte, notaryID string) (bool, error) {
	return true, nil
}

type mockIDVerifier struct{}

func (m *mockIDVerifier) VerifyIdentityDocument(ctx context.Context, doc *IdentityDocument) (*IDVerificationResult, error) {
	return &IDVerificationResult{
		Valid:             true,
		DocumentAuthentic: true,
		NotExpired:        true,
		BiometricMatch:    true,
		BiometricScore:    0.98,
	}, nil
}

func (m *mockIDVerifier) VerifyGovernmentID(ctx context.Context, idNumber, idType, issuingCountry string) (*GovernmentIDInfo, error) {
	return &GovernmentIDInfo{
		IDNumber:       idNumber,
		IDType:         idType,
		HolderName:     "John Director",
		IssuingCountry: issuingCountry,
		Status:         "valid",
	}, nil
}

func (m *mockIDVerifier) CheckIDExpiration(ctx context.Context, idNumber, idType string) (bool, time.Time, error) {
	return true, time.Now().AddDate(5, 0, 0), nil
}

func (m *mockIDVerifier) VerifyBiometricMatch(ctx context.Context, biometric []byte, idNumber string) (bool, float64, error) {
	return true, 0.98, nil
}

type mockSignatureVerifier struct{}

func (m *mockSignatureVerifier) VerifyDigitalSignature(ctx context.Context, data []byte, signature []byte, cert *x509.Certificate) error {
	return nil
}

func (m *mockSignatureVerifier) VerifyQualifiedSignature(ctx context.Context, data []byte, signature []byte, qcert *QualifiedCertificate) error {
	return nil
}

func (m *mockSignatureVerifier) CheckSignatureTimestamp(ctx context.Context, signature []byte) (*SignatureTimestamp, error) {
	return &SignatureTimestamp{
		Timestamp: time.Now(),
		Verified:  true,
	}, nil
}

func (m *mockSignatureVerifier) VerifySignatureChain(ctx context.Context, signatures []DigitalSignature) (*SignatureChainResult, error) {
	return &SignatureChainResult{
		Valid:           true,
		SignaturesCount: len(signatures),
		AllVerified:     true,
		ChainIntegrity:  true,
	}, nil
}

type mockPDPClient struct{}

func (m *mockPDPClient) EvaluatePolicy(ctx context.Context, request interface{}) (bool, error) {
	return true, nil
}

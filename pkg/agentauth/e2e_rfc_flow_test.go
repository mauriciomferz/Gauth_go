//go:build e2e

// Package agentauth - End-to-End RFC Flow Tests
// This file contains comprehensive E2E tests for AAP001 and AAP002 authorization flows
//
// UPDATED: November 12, 2025
// - Re-enabled E2E tests after gap closure analysis
// - Tests use current API interfaces
// - All interface compatibility issues resolved
//
// Status: Disabled - needs struct updates (see e2e_jwt_test.go for JWT validation)
// Last modified: 2025-11-12
package agentauth

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"math/big"
	"testing"
	"time"

	"github.com/mauriciomferz/AgentAuth/pkg/poa"
	"github.com/mauriciomferz/AgentAuth/pkg/poa/taxonomy"
)

// TestE2E_CompleteAuthorizationFlow tests the complete AAP001 authorization flow
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

	// Step 1: Create authorization chain (AAP001 steps I-VIII)
	chain := createTestAuthorizationChain(t)

	// Step 2: Validate authorization chain (AAP001 step a)
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

	// Step 3: Create PoA Definition (AAP002 sections A-C)
	poaDef := createTestPoADefinition(t, chain)

	// Step 4: Validate formal requirements (AAP002 Section C.1)
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

		// Register Authorization Server (Issuer)
		issuerInfo := &AuthorizationServerInfo{
			ServerID:   "test-issuer",
			ServerName: "Test Auth Server",
			Issuer:     "test-issuer",
			ServerURL:  "https://auth.example.com",
			IssueTime:  time.Now(),
		}
		err = pip.RegisterAuthorizationServer(issuerInfo)
		if err != nil {
			t.Fatalf("Failed to register authorization server: %v", err)
		}

		t.Logf("✓ All entities registered in PIP successfully")
	})

	// Step 6: Create token request (AAP001 step b - request compliance)
	t.Run("Step_b_CreateAndValidateRequest", func(t *testing.T) {
		request := &ExtendedAuthorizationRequest{
			AuthorizationRequest: &AuthorizationRequest{
				ClientID: "test-client-001",
				Scopes:   []string{"read", "write"},
			},
			LegalFramework: &LegalFrameworkInfo{
				ApplicableLaws: []string{"AgentAuth-Law-2025"},
				Jurisdiction:   "DE",
			},
			PowerOfAttorney:    poaDef,
			AuthorizationChain: chain,
			RequestedActions:   []string{"read", "write"},
			RequestTime:        time.Now(),
		}

		// Validate request compliance (AAP001 step b)
		result, err := complianceValidator.ValidateRequestCompliance(ctx, request)
		if err != nil {
			t.Fatalf("Failed to validate request compliance: %v", err)
		}
		if !result.Valid {
			t.Fatalf("Request compliance validation failed: %v", result.FailureReason)
		}
		t.Logf("✓ Request compliance validated successfully")
	})

	// Step 7: Create authorization grant (AAP001 steps c-e)
	t.Run("Step_c-e_AuthorizationGrant", func(t *testing.T) {
		// In real implementation, this would involve:
		// - Resource owner authentication (step c)
		// - Resource owner authorization (step d)
		// - Authorization code generation (step e)
		// For E2E test, we simulate a valid grant
		t.Logf("✓ Authorization grant simulated")
	})

	// Step 8: Validate grant compliance (AAP001 step f)
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
			IssuerID:           "test-issuer",
			LegalFramework: &LegalFrameworkInfo{
				ApplicableLaws: []string{"AgentAuth-Law-2025"},
				Jurisdiction:   "DE",
			},
		}

		result, err := complianceValidator.ValidateGrantCompliance(ctx, grant)
		if err != nil {
			t.Fatalf("Failed to validate grant compliance: %v", err)
		}
		if !result.Valid {
			t.Fatalf("Grant compliance validation failed: %v", result.FailureReason)
		}
		t.Logf("✓ Grant compliance validated successfully")
	})

	// Step 9: Create extended token (AAP001 step g)
	t.Run("Step_g_CreateExtendedToken", func(t *testing.T) {
		request := &ExtendedTokenRequest{
			GrantID:            "grant-001",
			Scope:              []string{"read", "write"},
			AuthorizationChain: chain,
			// PoACredentialRef replaced by actual definition in this struct version
			PowerOfAttorney: poaDef,
			ClientOwnerInfo: &ClientOwnerInfo{
				OwnerID:   "owner-001",
				OwnerName: "Test Owner Corp",
			},
			OwnersAuthorizerInfo: &OwnersAuthorizerInfo{
				AuthorizerID:            "authorizer-001",
				AuthorizerName:          "Test Authorizer Corp",
				CommercialRegisterEntry: true,
				IdentityVerified:        true,
			},
			ResourceOwnerInfo: &ResourceOwnerInfo{
				OwnerID: "owner-001",
			},
			LegalFramework: &LegalFrameworkInfo{
				ApplicableLaws: []string{"AgentAuth-Law-2025"},
				Jurisdiction:   "DE",
			},
			RequestID: "req-001",
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
		if len(token.Scope) == 0 {
			t.Error("Token scope is empty")
		}

		// Verify extended fields
		if token.AuthorizationChain == nil {
			t.Error("Authorization chain is missing")
		} else if !token.AuthorizationChain.ChainValidated {
			t.Error("Authorization chain should be validated")
		}

		if token.ComplianceLevel != "AAP001-compliant" {
			t.Errorf("Expected compliance level "AAP001-compliant', got '%s'", token.ComplianceLevel)
		}

		if token.PowerOfAttorney == nil {
			t.Error("PoA credential is missing")
		}

		t.Logf("✓ Extended token created successfully")
		t.Logf("  - Access Token: %s...", token.AccessToken[:20])
		t.Logf("  - Token Type: %s", token.TokenType)
		t.Logf("  - Expires In: %d seconds", token.ExpiresIn)
		t.Logf("  - Chain Valid: %v", token.AuthorizationChain.ChainValidated)
		t.Logf("  - Compliance Level: %s", token.ComplianceLevel)
	})

	// Step 10: Validate extended token (AAP001 step h)
	t.Run("Step_h_ValidateExtendedToken", func(t *testing.T) {
		// First create a token
		request := &ExtendedTokenRequest{
			GrantID:            "grant-002",
			Scope:              []string{"read", "write"},
			AuthorizationChain: chain,
			PowerOfAttorney:    poaDef,
			ClientOwnerInfo: &ClientOwnerInfo{
				OwnerID:   "owner-001",
				OwnerName: "Test Owner Corp",
			},
			OwnersAuthorizerInfo: &OwnersAuthorizerInfo{
				AuthorizerID:            "authorizer-001",
				AuthorizerName:          "Test Authorizer Corp",
				CommercialRegisterEntry: true,
				IdentityVerified:        true,
			},
			ResourceOwnerInfo: &ResourceOwnerInfo{
				OwnerID: "owner-001",
			},
			LegalFramework: &LegalFrameworkInfo{
				ApplicableLaws: []string{"AgentAuth-Law-2025"},
				Jurisdiction:   "DE",
			},
			RequestID: "req-002",
		}

		token, err := tokenService.CreateExtendedToken(ctx, request)
		if err != nil {
			t.Fatalf("Failed to create token: %v", err)
		}

		// Validate the token
		// Use EncodeExtendedToken to get the string, then Validate
		tokenStr, err := tokenService.EncodeExtendedToken(ctx, token)
		if err != nil {
			t.Fatalf("Failed to encode token: %v", err)
		}

		result, err := tokenService.ValidateExtendedToken(ctx, tokenStr)
		if err != nil {
			t.Fatalf("Failed to validate token: %v", err)
		}
		if !result.Valid {
			t.Fatalf("Token validation failed: %v", result.ValidationWarnings)
		}

		t.Logf("✓ Extended token validated successfully")
		t.Logf("  - Token Valid: %v", result.Valid)
		t.Logf("  - Client ID: %s", result.ClientID)
		t.Logf("  - Scope: %v", result.Scope)
	})

	// Step 11: Resource access (AAP001 step i)
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
	// Create Link 1: Owner's Authorizer (Root)
	authorizer := &AuthorizationLink{
		EntityID:           "authorizer-001",
		EntityType:         "organization",
		EntityName:         "Test Authorizer Corp",
		Role:               "authorizer",
		AuthorizationDate:  time.Now().Add(-90 * 24 * time.Hour),
		AuthorizationType:  "statutory",
		StatutoryAuthority: "Statutory Law Section 1", // AAP001 Requirement
		LegalBasis: &LegalBasis{
			BasisType:    "statutory",
			Jurisdiction: "DE",
		},
		IdentityVerified: true,
		ScopeOfAuthority: []string{"manage_clients", "authorize_actions"},
		ValidFrom:        time.Now().Add(-90 * 24 * time.Hour),
		ValidUntil:       time.Now().Add(500 * 24 * time.Hour),
		Revocable:        true,
		Status:           "active",
	}

	// Create Link 2: Client Owner (Authorized by Authorizer)
	owner := &AuthorizationLink{
		EntityID:          "owner-001",
		EntityType:        "natural_person", // or organization
		EntityName:        "Test Owner Corp",
		Role:              "owner",
		AuthorizedBy:      "authorizer-001",
		AuthorizationDate: time.Now().Add(-90 * 24 * time.Hour),
		AuthorizationType: "delegated",
		LegalBasis: &LegalBasis{
			BasisType:    "power_of_attorney",
			Jurisdiction: "DE",
		},
		IdentityVerified: true,
		ScopeOfAuthority: []string{"manage_ai_assets"},
		ValidFrom:        time.Now().Add(-90 * 24 * time.Hour),
		ValidUntil:       time.Now().Add(400 * 24 * time.Hour),
		Revocable:        true,
		Status:           "active",
	}

	// Create Link 3: Client (Authorized by Owner)
	client := &AuthorizationLink{
		EntityID:          "test-client-001",
		EntityType:        "ai_system",
		EntityName:        "Test AI Agent",
		Role:              "client",
		AuthorizedBy:      "owner-001",
		AuthorizationDate: time.Now().Add(-30 * 24 * time.Hour),
		AuthorizationType: "technical_assignment",
		LegalBasis: &LegalBasis{
			BasisType: "technical_configuration",
		},
		IdentityVerified: true,
		ScopeOfAuthority: []string{"read", "write", "execute"},
		ValidFrom:        time.Now().Add(-30 * 24 * time.Hour),
		ValidUntil:       time.Now().Add(365 * 24 * time.Hour),
		Revocable:        true,
		Status:           "active",
	}

	return &AuthorizationChain{
		OwnersAuthorizer: authorizer,
		ClientOwner:      owner,
		Client:           client,
		ChainValidated:   true,
		ValidationTime:   time.Now(),
		ChainDepth:       3,
		ChainIntegrity:   "chain-integrity-hash-12345",
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
				Transactions:       []taxonomy.TransactionType{taxonomy.TransactionPurchase, taxonomy.TransactionPayment},
				Decisions:          []taxonomy.DecisionType{taxonomy.DecisionOperational},
				NonPhysicalActions: []taxonomy.ActionTypeNonPhysical{taxonomy.ActionNonPhysicalResearching, taxonomy.ActionNonPhysicalAnalyzing},
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
		NotaryID:          "notary-123",
		NotaryName:        "Test Notary Public",
		NotaryLicense:     "LIC-12345",
		Jurisdiction:      "DE",
		IssuingAuthority:  "Chamber of Notaries",
		CertificationDate: time.Now(),
		ExpirationDate:    time.Now().Add(5 * 365 * 24 * time.Hour),
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

	// Create a dummy certificate
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		NotBefore:    time.Now().Add(-1 * time.Hour),
		NotAfter:     time.Now().Add(1 * time.Hour),
	}
	// We don't need a valid x509 cert for the mock verifier unless it parses it,
	// but the struct field must be non-nil.
	// However, formal_requirements_validation.go checks IsZero() or nil fields.
	// Let's use the template as is (since it assumes it's parsed).

	return []*DigitalSignature{
		{
			SignatureValue: signature,
			SignatureAlg:   "Ed25519",
			Timestamp:      time.Now(),
			SignerInfo:     "owner-001",
			Certificate:    template, // Mocked certificate
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
		ExpiryDate:    time.Now().Add(365 * 24 * time.Hour),
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

// Package gauth - Integration Tests
// This file contains integration tests for the implemented RFC-0111/RFC-0115 components
package gauth

import (
	"context"
	"testing"
	"time"

	"github.com/mauriciomferz/AgentAuth/pkg/poa"
	"github.com/mauriciomferz/AgentAuth/pkg/poa/taxonomy"
)

// TestAuthorizationChainValidation tests the authorization chain validator
func TestAuthorizationChainValidation(t *testing.T) {
	ctx := context.Background()

	// Setup mock services
	mockCR := NewMockCommercialRegisterClient(false)
	mockTSP := NewMockTrustServiceProvider(false)
	mockRevoke := NewMockRevocationChecker(false)

	// Create validator
	validator := NewAuthorizationChainValidator(mockCR, mockTSP, mockRevoke)

	// Create test chain with actual struct format
	now := time.Now()
	chain := &AuthorizationChain{
		OwnersAuthorizer: &AuthorizationLink{
			EntityID:              "HRB-12345-DE",
			EntityType:            "natural_person",
			EntityName:            "Test Authorizer",
			Role:                  "authorizer",
			AuthorizationDate:     now.Add(-90 * 24 * time.Hour),
			AuthorizationType:     "statutory",
			StatutoryAuthority:    "Managing Director",
			CommercialRegisterRef: "HRB-12345-DE",
			LegalBasis: &LegalBasis{
				BasisType:          "company_law",
				Jurisdiction:       "DE",
				RegistrationNumber: "HRB-12345-DE",
			},
			IdentityVerified:   true,
			VerificationMethod: "eIDAS",
			ScopeOfAuthority:   []string{"manage_clients"},
			ValidFrom:          now.Add(-90 * 24 * time.Hour),
			ValidUntil:         now.Add(270 * 24 * time.Hour),
			Revocable:          true,
			Status:             "active",
		},
		ClientOwner: &AuthorizationLink{
			EntityID:          "owner-001",
			EntityType:        "organization",
			EntityName:        "Test Owner Corp",
			Role:              "owner",
			AuthorizedBy:      "HRB-12345-DE",
			AuthorizationDate: now.Add(-60 * 24 * time.Hour),
			AuthorizationType: "delegated",
			LegalBasis: &LegalBasis{
				BasisType:    "power_of_attorney",
				Jurisdiction: "DE",
			},
			IdentityVerified:   true,
			VerificationMethod: "commercial_register",
			ScopeOfAuthority:   []string{"operate_ai_systems"},
			ValidFrom:          now.Add(-60 * 24 * time.Hour),
			ValidUntil:         now.Add(200 * 24 * time.Hour),
			Revocable:          true,
			Status:             "active",
		},
		Client: &AuthorizationLink{
			EntityID:          "client-001",
			EntityType:        "ai_system",
			EntityName:        "Test AI Client",
			Role:              "client",
			AuthorizedBy:      "owner-001",
			AuthorizationDate: now.Add(-30 * 24 * time.Hour),
			AuthorizationType: "delegated",
			LegalBasis: &LegalBasis{
				BasisType:    "power_of_attorney",
				Jurisdiction: "DE",
			},
			IdentityVerified:   true,
			VerificationMethod: "digital_certificate",
			ScopeOfAuthority:   []string{"read", "write", "analyze"},
			ValidFrom:          now.Add(-30 * 24 * time.Hour),
			ValidUntil:         now.Add(180 * 24 * time.Hour),
			Revocable:          true,
			Status:             "active",
		},
		ChainValidated: false,
		ChainDepth:     3,
		ChainIntegrity: "test-chain-hash-abc123",
	}

	// Validate chain
	result, err := validator.ValidateAuthorizationChain(ctx, chain)
	if err != nil {
		t.Fatalf("Failed to validate chain: %v", err)
	}

	if !result.Valid {
		t.Errorf("Chain validation should succeed, got valid=%v, reason=%v", result.Valid, result.FailureReason)
	}

	t.Logf("✓ Authorization chain validation passed")
	t.Logf("  - Valid: %v", result.Valid)
	t.Logf("  - Chain Depth: %d", result.ValidatedChainDepth)
	t.Logf("  - Link Validations: %d", len(result.LinkValidations))
}

// TestUnifiedPIP tests the unified PIP implementation
func TestUnifiedPIP(t *testing.T) {
	ctx := context.Background()

	// Setup mock services
	mockCR := NewMockCommercialRegisterClient(false)
	mockTSP := NewMockTrustServiceProvider(false)

	// Create PIP
	pip := NewUnifiedPIP(mockCR, mockTSP, true, 5*time.Minute)

	// Test client registration
	t.Run("RegisterClient", func(t *testing.T) {
		client := &ClientInfo{
			ClientID: "client-001",
		}
		err := pip.RegisterClient(client)
		if err != nil {
			t.Fatalf("Failed to register client: %v", err)
		}
		t.Logf("✓ Client registered successfully")
	})

	// Test client owner registration
	t.Run("RegisterClientOwner", func(t *testing.T) {
		owner := &ClientOwnerInfo{
			OwnerID: "owner-001",
		}
		err := pip.RegisterClientOwner(owner)
		if err != nil {
			t.Fatalf("Failed to register owner: %v", err)
		}
		t.Logf("✓ Client owner registered successfully")
	})

	// Test PoA registration
	t.Run("RegisterPoA", func(t *testing.T) {
		poaDef := &poa.PoADefinition{
			Parties: poa.Parties{
				Principal: poa.Principal{
					Type:     "Organization",
					Identity: "owner-001",
				},
				AuthorizedClient: poa.AuthorizedClient{
					TypeEnum: poa.ClientTypeLLM,
					Identity: "client-001",
				},
			},
			Authorization: poa.AuthorizationScope{
				AuthorizedActions: poa.AuthorizedActions{
					NonPhysicalActions: []taxonomy.ActionTypeNonPhysical{
						taxonomy.ActionNonPhysicalResearching,
					},
				},
			},
		}

		err := pip.RegisterPoA(poaDef, "poa-001")
		if err != nil {
			t.Fatalf("Failed to register PoA: %v", err)
		}
		t.Logf("✓ PoA registered successfully")
	})

	// Test attribute management
	t.Run("AttributeManagement", func(t *testing.T) {
		// Set attribute
		err := pip.SetAttribute(ctx, "client-001", "test_attr", "test_value")
		if err != nil {
			t.Fatalf("Failed to set attribute: %v", err)
		}

		// Get attribute
		value, err := pip.GetAttribute(ctx, "client-001", "test_attr")
		if err != nil {
			t.Fatalf("Failed to get attribute: %v", err)
		}

		if value != "test_value" {
			t.Errorf("Expected 'test_value', got '%v'", value)
		}

		t.Logf("✓ Attribute management working correctly")
	})

	// Test PIP status
	t.Run("GetStatus", func(t *testing.T) {
		status, err := pip.GetStatus(ctx)
		if err != nil {
			t.Fatalf("Failed to get status: %v", err)
		}

		if !status.Operational {
			t.Error("PIP should be operational")
		}

		t.Logf("✓ PIP status retrieved successfully")
		t.Logf("  - Operational: %v", status.Operational)
		t.Logf("  - Cache Enabled: %v", status.CacheEnabled)
		t.Logf("  - Total Queries: %d", status.Stats.TotalQueries)
	})
}

// TestActionTaxonomy tests the action taxonomy implementation
func TestActionTaxonomy(t *testing.T) {
	// Test action set validation
	t.Run("ValidateActionSet", func(t *testing.T) {
		actionSet := &poa.AuthorizedActionSet{
			Transactions: []taxonomy.TransactionType{
				taxonomy.TransactionPurchase,
				taxonomy.TransactionPayment,
			},
			Decisions: []taxonomy.DecisionType{
				taxonomy.DecisionOperational,
			},
			NonPhysicalActions: []taxonomy.ActionTypeNonPhysical{
				taxonomy.ActionNonPhysicalResearching,
				taxonomy.ActionNonPhysicalAnalyzing,
			},
		}

		err := actionSet.Validate()
		if err != nil {
			t.Fatalf("Action set validation failed: %v", err)
		}

		t.Logf("✓ Action set validated successfully")
	})

	// Test comprehensive taxonomy report
	t.Run("GenerateTaxonomyReport", func(t *testing.T) {
		actionSet := &poa.AuthorizedActionSet{
			Transactions: []taxonomy.TransactionType{
				taxonomy.TransactionPurchase,
			},
			Decisions: []taxonomy.DecisionType{
				taxonomy.DecisionOperational,
			},
			NonPhysicalActions: []taxonomy.ActionTypeNonPhysical{
				taxonomy.ActionNonPhysicalResearching,
			},
		}

		report, err := taxonomy.GenerateComprehensiveTaxonomyReport(actionSet)
		if err != nil {
			t.Fatalf("Failed to generate report: %v", err)
		}

		if report.TotalActions != 3 {
			t.Errorf("Expected 3 total actions, got %d", report.TotalActions)
		}

		t.Logf("✓ Taxonomy report generated successfully")
		t.Logf("  - Total Actions: %d", report.TotalActions)
		t.Logf("  - Overall Risk: %s", report.OverallRisk)
		t.Logf("  - Requires Approval: %d", report.RequiresApproval)
		t.Logf("  - Summary: %s", report.Summary)
	})

	// Test transaction metadata
	t.Run("GetTransactionMetadata", func(t *testing.T) {
		meta, err := taxonomy.GetTransactionMetadata(taxonomy.TransactionPurchase)
		if err != nil {
			t.Fatalf("Failed to get metadata: %v", err)
		}

		if meta.Type != taxonomy.TransactionPurchase {
			t.Errorf("Expected TransactionPurchase, got %v", meta.Type)
		}

		if meta.Category != taxonomy.CategoryFinancial {
			t.Errorf("Expected CategoryFinancial, got %v", meta.Category)
		}

		t.Logf("✓ Transaction metadata retrieved successfully")
		t.Logf("  - Type: %s", meta.Type)
		t.Logf("  - Category: %s", meta.Category)
		t.Logf("  - Risk: %s", meta.Risk)
		t.Logf("  - Description: %s", meta.Description)
	})

	// Test action compatibility check
	t.Run("ActionCompatibilityCheck", func(t *testing.T) {
		actionSet := &poa.AuthorizedActionSet{
			NonPhysicalActions: []taxonomy.ActionTypeNonPhysical{
				taxonomy.ActionNonPhysicalResearching,
			},
		}

		client := &poa.AuthorizedClient{
			TypeEnum:        poa.ClientTypeLLM,
			StatusEnum:      poa.OperationalStatusActive,
			CapabilityLevel: poa.CapabilityL3,
		}

		err := poa.ActionCompatibilityCheck(actionSet, client)
		if err != nil {
			t.Fatalf("Compatibility check failed: %v", err)
		}

		t.Logf("✓ Action compatibility check passed")
	})
}

// TestExtendedTokenService tests the extended token service
func TestExtendedTokenService(t *testing.T) {
	// Setup dependencies
	mockCR := NewMockCommercialRegisterClient(false)
	mockTSP := NewMockTrustServiceProvider(false)
	mockRevoke := NewMockRevocationChecker(false)

	chainValidator := NewAuthorizationChainValidator(mockCR, mockTSP, mockRevoke)
	pip := NewUnifiedPIP(mockCR, mockTSP, true, 5*time.Minute)

	// Create a simple mock PDP
	mockPDP := &simpleMockPDP{}

	complianceValidator := NewComplianceValidator(chainValidator, pip, mockPDP)

	tokenService := NewExtendedTokenService(
		chainValidator,
		complianceValidator,
		pip,
		"test-issuer",
		"https://auth.example.com",
		24*time.Hour,
	)

	t.Run("ServiceCreation", func(t *testing.T) {
		if tokenService == nil {
			t.Fatal("Token service should not be nil")
		}
		t.Logf("✓ Extended token service created successfully")
	})

	// Additional token tests would require more complete request/response structures
	// which need struct alignment work
}

// simpleMockPDP is a minimal PDP implementation for testing
type simpleMockPDP struct{}

func (m *simpleMockPDP) EvaluatePolicy(ctx context.Context, request interface{}) (bool, error) {
	return true, nil
}

// TestMockImplementations tests that all mock implementations work correctly
func TestMockImplementations(t *testing.T) {
	ctx := context.Background()

	t.Run("MockCommercialRegister", func(t *testing.T) {
		mock := NewMockCommercialRegisterClient(false)

		company, err := mock.VerifyCompany(ctx, "DE", "HRB-12345")
		if err != nil {
			t.Fatalf("Failed to verify company: %v", err)
		}
		if company == nil {
			t.Fatal("Company should not be nil")
		}
		t.Logf("✓ Mock commercial register working")
	})

	t.Run("MockTrustServiceProvider", func(t *testing.T) {
		mock := NewMockTrustServiceProvider(false)

		doc := &IdentityDocument{
			DocumentType: "passport",
			DocumentID:   "test-123",
		}
		result, err := mock.VerifyIdentity(ctx, doc)
		if err != nil {
			t.Fatalf("Failed to verify identity: %v", err)
		}
		if result == nil {
			t.Fatal("Result should not be nil")
		}
		t.Logf("✓ Mock trust service provider working")
	})

	t.Run("MockRevocationChecker", func(t *testing.T) {
		mock := NewMockRevocationChecker(false)

		revoked, err := mock.IsRevoked(ctx, "test-id")
		if err != nil {
			t.Fatalf("Failed to check revocation: %v", err)
		}
		if revoked {
			t.Error("Should not be revoked in non-strict mode")
		}
		t.Logf("✓ Mock revocation checker working")
	})
}

// TestIntegration_CompleteFlow tests a simplified end-to-end flow
func TestIntegration_CompleteFlow(t *testing.T) {
	ctx := context.Background()

	// Setup all services
	mockCR := NewMockCommercialRegisterClient(false)
	mockTSP := NewMockTrustServiceProvider(false)
	mockRevoke := NewMockRevocationChecker(false)

	chainValidator := NewAuthorizationChainValidator(mockCR, mockTSP, mockRevoke)
	pip := NewUnifiedPIP(mockCR, mockTSP, true, 5*time.Minute)

	t.Run("Step1_CreateAndValidateChain", func(t *testing.T) {
		now := time.Now()
		chain := &AuthorizationChain{
			OwnersAuthorizer: &AuthorizationLink{
				EntityID:              "HRB-12345-DE",
				EntityType:            "natural_person",
				EntityName:            "Authorizer",
				Role:                  "authorizer",
				AuthorizationDate:     now.Add(-90 * 24 * time.Hour),
				AuthorizationType:     "statutory",
				StatutoryAuthority:    "Managing Director",
				CommercialRegisterRef: "HRB-12345-DE",
				LegalBasis: &LegalBasis{
					BasisType:          "company_law",
					Jurisdiction:       "DE",
					RegistrationNumber: "HRB-12345-DE",
				},
				IdentityVerified:   true,
				VerificationMethod: "eIDAS",
				ScopeOfAuthority:   []string{"manage"},
				ValidFrom:          now.Add(-90 * 24 * time.Hour),
				ValidUntil:         now.Add(270 * 24 * time.Hour),
				Revocable:          true,
				Status:             "active",
			},
			ClientOwner: &AuthorizationLink{
				EntityID:          "owner-001",
				EntityType:        "organization",
				EntityName:        "Owner Corp",
				Role:              "owner",
				AuthorizedBy:      "HRB-12345-DE",
				AuthorizationDate: now.Add(-60 * 24 * time.Hour),
				AuthorizationType: "delegated",
				LegalBasis: &LegalBasis{
					BasisType:    "power_of_attorney",
					Jurisdiction: "DE",
				},
				IdentityVerified:   true,
				VerificationMethod: "commercial_register",
				ScopeOfAuthority:   []string{"operate_ai"},
				ValidFrom:          now.Add(-60 * 24 * time.Hour),
				ValidUntil:         now.Add(200 * 24 * time.Hour),
				Revocable:          true,
				Status:             "active",
			},
			Client: &AuthorizationLink{
				EntityID:          "client-001",
				EntityType:        "ai_system",
				EntityName:        "AI Client",
				Role:              "client",
				AuthorizedBy:      "owner-001",
				AuthorizationDate: now.Add(-30 * 24 * time.Hour),
				AuthorizationType: "delegated",
				LegalBasis: &LegalBasis{
					BasisType:    "power_of_attorney",
					Jurisdiction: "DE",
				},
				IdentityVerified:   true,
				VerificationMethod: "digital_certificate",
				ScopeOfAuthority:   []string{"read", "write"},
				ValidFrom:          now.Add(-30 * 24 * time.Hour),
				ValidUntil:         now.Add(180 * 24 * time.Hour),
				Revocable:          true,
				Status:             "active",
			},
			ChainDepth:     3,
			ChainIntegrity: "chain-hash-complete",
		}

		result, err := chainValidator.ValidateAuthorizationChain(ctx, chain)
		if err != nil {
			t.Fatalf("Chain validation failed: %v", err)
		}
		if !result.Valid {
			t.Fatalf("Chain should be valid, reason: %v", result.FailureReason)
		}

		t.Logf("✓ Step 1: Authorization chain validated")
	})

	t.Run("Step2_RegisterEntitiesInPIP", func(t *testing.T) {
		client := &ClientInfo{ClientID: "client-001"}
		owner := &ClientOwnerInfo{OwnerID: "owner-001"}

		if err := pip.RegisterClient(client); err != nil {
			t.Fatalf("Failed to register client: %v", err)
		}
		if err := pip.RegisterClientOwner(owner); err != nil {
			t.Fatalf("Failed to register owner: %v", err)
		}

		t.Logf("✓ Step 2: Entities registered in PIP")
	})

	t.Run("Step3_VerifyPIPOperational", func(t *testing.T) {
		status, err := pip.GetStatus(ctx)
		if err != nil {
			t.Fatalf("Failed to get PIP status: %v", err)
		}
		if !status.Operational {
			t.Fatal("PIP should be operational")
		}

		t.Logf("✓ Step 3: PIP operational and functioning")
	})

	t.Logf("\n✅ Complete integration flow test passed!")
}

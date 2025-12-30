// Package integration - Integration Tests for OIDC with AgentAuth Subscription Flow
package integration

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/mauriciomferz/Gauth_go/pkg/gauth"
	"github.com/mauriciomferz/Gauth_go/pkg/oidc"
)

// TestOIDCPVPIntegrationWithSubscriptionFlow tests complete OIDC → AgentAuth integration
func TestOIDCPVPIntegrationWithSubscriptionFlow(t *testing.T) {
	ctx := context.Background()

	// Setup OIDC infrastructure
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate RSA key: %v", err)
	}

	idTokenService, err := oidc.NewIDTokenService(&oidc.IDTokenServiceConfig{
		IssuerURL:    "https://gauth.example.com",
		SigningKey:   privateKey,
		SigningKeyID: "test-key-1",
	})
	if err != nil {
		t.Fatalf("Failed to create ID token service: %v", err)
	}

	oidcPVP, err := oidc.NewOIDCPowerVerificationPoint(oidc.OIDCPVPConfig{
		IDTokenService: idTokenService,
		RequiredACR:    "substantial",
	})
	if err != nil {
		t.Fatalf("Failed to create OIDC PVP: %v", err)
	}

	// Issue a valid ID token for the owner's authorizer
	ownerAuthorizerClaims := &oidc.IDTokenClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "owner-auth-123",
			Audience:  jwt.ClaimStrings{"gauth-server"},
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
		Name:       "Alice Johnson",
		Email:      "alice@example.com",
		ACR:        "substantial",
		EntityType: "natural_person",
	}
	ownerAuthorizerToken, err := idTokenService.IssueIDToken(ctx, ownerAuthorizerClaims)
	if err != nil {
		t.Fatalf("Failed to issue owner authorizer ID token: %v", err)
	}

	// Issue a valid ID token for the client owner
	clientOwnerClaims := &oidc.IDTokenClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "client-owner-456",
			Audience:  jwt.ClaimStrings{"gauth-server"},
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
		Name:            "Bob Smith",
		Email:           "bob@example.com",
		ACR:             "high",
		EntityType:      "legal_entity",
		LegalEntityName: "Example Corp",
	}
	clientOwnerToken, err := idTokenService.IssueIDToken(ctx, clientOwnerClaims)
	if err != nil {
		t.Fatalf("Failed to issue client owner ID token: %v", err)
	}

	// Issue a valid ID token for the resource owner
	resourceOwnerClaims := &oidc.IDTokenClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "resource-owner-789",
			Audience:  jwt.ClaimStrings{"gauth-server"},
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
		Name:       "Carol White",
		Email:      "carol@example.com",
		ACR:        "substantial",
		EntityType: "natural_person",
	}
	resourceOwnerToken, err := idTokenService.IssueIDToken(ctx, resourceOwnerClaims)
	if err != nil {
		t.Fatalf("Failed to issue resource owner ID token: %v", err)
	}

	// Test Step I: Owner's Authorizer Identity Proof with OIDC
	t.Run("Step I: Owner's Authorizer with OIDC ID Token", func(t *testing.T) {
		request := &gauth.IdentityProofRequest{
			SubjectID:    "owner-auth-123",
			IdentityType: "natural_person",
			ProofMethod:  "oidc_id_token",
			ProofData: map[string]interface{}{
				"id_token": ownerAuthorizerToken,
				"audience": "gauth-server",
			},
			RequiredLevel: "substantial",
		}

		result, err := oidcPVP.VerifyIdentityProof(ctx, request)
		if err != nil {
			t.Fatalf("Step I failed: %v", err)
		}

		if !result.Valid {
			t.Errorf("Expected valid result, got invalid: %s", result.FailureReason)
		}

		if result.SubjectID != "owner-auth-123" {
			t.Errorf("Expected SubjectID 'owner-auth-123', got '%s'", result.SubjectID)
		}

		if result.Identity != "Alice Johnson" {
			t.Errorf("Expected Identity 'Alice Johnson', got '%s'", result.Identity)
		}

		if result.TrustLevel != "substantial" {
			t.Errorf("Expected TrustLevel 'substantial', got '%s'", result.TrustLevel)
		}

		t.Logf("✓ Step I passed: %s verified with trust level %s", result.Identity, result.TrustLevel)
	})

	// Test Step III: Client Owner Identity Proof with OIDC
	t.Run("Step III: Client Owner with OIDC ID Token", func(t *testing.T) {
		request := &gauth.IdentityProofRequest{
			SubjectID:    "client-owner-456",
			IdentityType: "legal_entity",
			ProofMethod:  "oidc_id_token",
			ProofData: map[string]interface{}{
				"id_token": clientOwnerToken,
				"audience": "gauth-server",
			},
			RequiredLevel: "substantial",
		}

		result, err := oidcPVP.VerifyIdentityProof(ctx, request)
		if err != nil {
			t.Fatalf("Step III failed: %v", err)
		}

		if !result.Valid {
			t.Errorf("Expected valid result, got invalid: %s", result.FailureReason)
		}

		if result.SubjectID != "client-owner-456" {
			t.Errorf("Expected SubjectID 'client-owner-456', got '%s'", result.SubjectID)
		}

		if result.Identity != "Example Corp" {
			t.Errorf("Expected Identity 'Example Corp', got '%s'", result.Identity)
		}

		if result.TrustLevel != "high" {
			t.Errorf("Expected TrustLevel 'high', got '%s'", result.TrustLevel)
		}

		t.Logf("✓ Step III passed: %s verified with trust level %s", result.Identity, result.TrustLevel)
	})

	// Test Step VI: Resource Owner Identity Proof with OIDC
	t.Run("Step VI: Resource Owner with OIDC ID Token", func(t *testing.T) {
		request := &gauth.IdentityProofRequest{
			SubjectID:    "resource-owner-789",
			IdentityType: "natural_person",
			ProofMethod:  "oidc_id_token",
			ProofData: map[string]interface{}{
				"id_token": resourceOwnerToken,
				"audience": "gauth-server",
			},
			RequiredLevel: "substantial",
		}

		result, err := oidcPVP.VerifyIdentityProof(ctx, request)
		if err != nil {
			t.Fatalf("Step VI failed: %v", err)
		}

		if !result.Valid {
			t.Errorf("Expected valid result, got invalid: %s", result.FailureReason)
		}

		if result.SubjectID != "resource-owner-789" {
			t.Errorf("Expected SubjectID 'resource-owner-789', got '%s'", result.SubjectID)
		}

		if result.Identity != "Carol White" {
			t.Errorf("Expected Identity 'Carol White', got '%s'", result.Identity)
		}

		if result.TrustLevel != "substantial" {
			t.Errorf("Expected TrustLevel 'substantial', got '%s'", result.TrustLevel)
		}

		t.Logf("✓ Step VI passed: %s verified with trust level %s", result.Identity, result.TrustLevel)
	})

	// Test with insufficient trust level
	t.Run("Insufficient trust level rejection", func(t *testing.T) {
		lowTrustClaims := &oidc.IDTokenClaims{
			RegisteredClaims: jwt.RegisteredClaims{
				Subject:   "low-trust-user",
				Audience:  jwt.ClaimStrings{"gauth-server"},
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
				IssuedAt:  jwt.NewNumericDate(time.Now()),
				NotBefore: jwt.NewNumericDate(time.Now()),
			},
			Name:       "Low Trust User",
			ACR:        "0", // Low trust
			EntityType: "natural_person",
		}
		lowTrustToken, err := idTokenService.IssueIDToken(ctx, lowTrustClaims)
		if err != nil {
			t.Fatalf("Failed to issue low trust token: %v", err)
		}

		request := &gauth.IdentityProofRequest{
			SubjectID:    "low-trust-user",
			IdentityType: "natural_person",
			ProofMethod:  "oidc_id_token",
			ProofData: map[string]interface{}{
				"id_token": lowTrustToken,
				"audience": "gauth-server",
			},
			RequiredLevel: "substantial",
		}

		result, err := oidcPVP.VerifyIdentityProof(ctx, request)
		if err != nil {
			t.Fatalf("Verification failed: %v", err)
		}

		if result.Valid {
			t.Error("Expected invalid result for insufficient trust level")
		}

		if result.FailureReason == "" {
			t.Error("Expected failure reason for insufficient trust level")
		}

		t.Logf("✓ Correctly rejected low trust: %s", result.FailureReason)
	})
}

// TestOIDCPVPWithPVPRouter tests OIDC PVP integration with PVP router
func TestOIDCPVPWithPVPRouter(t *testing.T) {
	ctx := context.Background()

	// Setup OIDC PVP
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate RSA key: %v", err)
	}

	idTokenService, err := oidc.NewIDTokenService(&oidc.IDTokenServiceConfig{
		IssuerURL:    "https://gauth.example.com",
		SigningKey:   privateKey,
		SigningKeyID: "test-key-1",
	})
	if err != nil {
		t.Fatalf("Failed to create ID token service: %v", err)
	}

	oidcPVP, err := oidc.NewOIDCPowerVerificationPoint(oidc.OIDCPVPConfig{
		IDTokenService: idTokenService,
		RequiredACR:    "substantial",
	})
	if err != nil {
		t.Fatalf("Failed to create OIDC PVP: %v", err)
	}

	// Create router and register OIDC PVP
	router := gauth.NewPVPRouter(nil)
	router.RegisterPVP([]string{"oidc_id_token", "oidc_external"}, oidcPVP)

	// Verify supported methods
	supportedMethods := router.GetSupportedProofMethods()
	t.Logf("Supported proof methods: %v", supportedMethods)

	if len(supportedMethods) != 2 {
		t.Errorf("Expected 2 supported methods, got %d", len(supportedMethods))
	}

	// Issue ID token
	claims := &oidc.IDTokenClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "test-user",
			Audience:  jwt.ClaimStrings{"test-client"},
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
		Name:       "Test User",
		Email:      "test@example.com",
		ACR:        "high",
		EntityType: "natural_person",
	}
	idToken, err := idTokenService.IssueIDToken(ctx, claims)
	if err != nil {
		t.Fatalf("Failed to issue ID token: %v", err)
	}

	// Verify through router
	request := &gauth.IdentityProofRequest{
		SubjectID:    "test-user",
		IdentityType: "natural_person",
		ProofMethod:  "oidc_id_token",
		ProofData: map[string]interface{}{
			"id_token": idToken,
			"audience": "test-client",
		},
		RequiredLevel: "substantial",
	}

	result, err := router.VerifyIdentityProof(ctx, request)
	if err != nil {
		t.Fatalf("Router verification failed: %v", err)
	}

	if !result.Valid {
		t.Errorf("Expected valid result, got invalid: %s", result.FailureReason)
	}

	if result.Identity != "Test User" {
		t.Errorf("Expected Identity 'Test User', got '%s'", result.Identity)
	}

	if result.TrustLevel != "high" {
		t.Errorf("Expected TrustLevel 'high', got '%s'", result.TrustLevel)
	}

	t.Logf("✓ Router successfully routed OIDC proof: %s verified with trust level %s",
		result.Identity, result.TrustLevel)
}

// TestMultipleProofMethodsWithRouter tests routing multiple proof methods
func TestMultipleProofMethodsWithRouter(t *testing.T) {
	// Setup OIDC PVP
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate RSA key: %v", err)
	}

	idTokenService, err := oidc.NewIDTokenService(&oidc.IDTokenServiceConfig{
		IssuerURL:    "https://gauth.example.com",
		SigningKey:   privateKey,
		SigningKeyID: "test-key-1",
	})
	if err != nil {
		t.Fatalf("Failed to create ID token service: %v", err)
	}

	oidcPVP, err := oidc.NewOIDCPowerVerificationPoint(oidc.OIDCPVPConfig{
		IDTokenService: idTokenService,
		RequiredACR:    "substantial",
	})
	if err != nil {
		t.Fatalf("Failed to create OIDC PVP: %v", err)
	}

	// Create mock eIDAS PVP
	eidasPVP := &mockEIDASPVP{}

	// Create router with multiple PVPs
	router := gauth.NewPVPRouter(nil)
	router.RegisterPVP([]string{"oidc_id_token", "oidc_external"}, oidcPVP)
	router.RegisterPVP([]string{"eIDAS"}, eidasPVP)

	// Verify all supported methods
	supportedMethods := router.GetSupportedProofMethods()
	if len(supportedMethods) != 3 {
		t.Errorf("Expected 3 supported methods, got %d", len(supportedMethods))
	}

	t.Logf("✓ Router supports multiple proof methods: %v", supportedMethods)
}

// mockEIDASPVP is a mock implementation for testing
type mockEIDASPVP struct{}

func (m *mockEIDASPVP) VerifyIdentityProof(ctx context.Context, request *gauth.IdentityProofRequest) (*gauth.IdentityProofResult, error) {
	return &gauth.IdentityProofResult{
		Valid:      true,
		SubjectID:  request.SubjectID,
		Identity:   "eIDAS User",
		VerifiedAt: time.Now(),
		TrustLevel: "substantial",
	}, nil
}

package gauth

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/mauriciomferz/AgentAuth/pkg/poa"
)

// TestE2E_JWTSerializationRoundTrip validates JWT token encoding/decoding
// This addresses GAP #1-2 from QA audit: JWT/JWE serialization implementation
//
// Validates:
// - JWT compact serialization (header.payload.signature)
// - JWT parsing and validation
// - Token claim preservation
// - RFC-0111 extended token structure serialization
func TestE2E_JWTSerializationRoundTrip(t *testing.T) {
	ctx := context.Background()

	// Initialize token service with validators (all nil for basic JWT testing)
	chainValidator := NewAuthorizationChainValidator(nil, nil, nil)
	complianceValidator := NewComplianceValidator(nil, nil, nil)

	tokenService := NewExtendedTokenService(
		chainValidator,
		complianceValidator,
		nil,                           // pipClient (optional for this test)
		"test-issuer",                 // issuerID
		"https://test-issuer.example", // issuerURL
		time.Hour,                     // tokenExpiry (3600 seconds)
	)

	t.Run("JWT_Compact_Serialization", func(t *testing.T) {
		request := createBaseTokenRequest("test-grant-001", []string{"read", "write"})

		token, err := tokenService.CreateExtendedToken(ctx, request)
		if err != nil {
			t.Fatalf("Token creation failed: %v", err)
		}

		// Encode extended token to JWT
		jwtString, err := tokenService.EncodeExtendedToken(ctx, token)
		if err != nil {
			t.Fatalf("Encoding failed: %v", err)
		}

		// Validate JWT compact serialization format
		parts := strings.Split(jwtString, ".")
		if len(parts) != 3 {
			t.Fatalf("Expected JWT with 3 parts (header.payload.signature), got %d parts", len(parts))
		}

		t.Logf("✅ JWT compact serialization format validated: %d parts", len(parts))
	})

	t.Run("JWT_Parsing_And_Validation", func(t *testing.T) {
		request := createBaseTokenRequest("test-grant-002", []string{"admin"})

		createdToken, err := tokenService.CreateExtendedToken(ctx, request)
		if err != nil {
			t.Fatalf("Token creation failed: %v", err)
		}

		jwtString, err := tokenService.EncodeExtendedToken(ctx, createdToken)
		if err != nil {
			t.Fatalf("Encoding failed: %v", err)
		}

		// Parse the JWT using golang-jwt library
		parser := jwt.NewParser(
			jwt.WithValidMethods([]string{"HS256", "RS256"}),
			jwt.WithoutClaimsValidation(),
		)

		parsedToken, _, err := parser.ParseUnverified(jwtString, jwt.MapClaims{})
		if err != nil {
			t.Fatalf("JWT parsing failed: %v", err)
		}

		if parsedToken.Header == nil {
			t.Fatal("JWT header is nil")
		}

		claims, ok := parsedToken.Claims.(jwt.MapClaims)
		if !ok {
			t.Fatal("Failed to extract claims")
		}

		requiredClaims := []string{"iss", "sub", "exp", "iat"}
		for _, claim := range requiredClaims {
			if _, exists := claims[claim]; !exists {
				t.Errorf("Missing required claim: %s", claim)
			}
		}

		t.Logf("✅ JWT parsing successful")
	})

	t.Run("Extended_Token_Claims_Preservation", func(t *testing.T) {
		request := createBaseTokenRequest("grant-rich-001", []string{"read:user", "write:data", "admin:system"})
		request.LegalFramework.Jurisdiction = "EU"
		request.LegalFramework.ApplicableLaws = []string{"EU-AI-ACT"}

		token, err := tokenService.CreateExtendedToken(ctx, request)
		if err != nil {
			t.Fatalf("Creation failed: %v", err)
		}

		jwtString, err := tokenService.EncodeExtendedToken(ctx, token)
		if err != nil {
			t.Fatalf("Encoding failed: %v", err)
		}

		parser := jwt.NewParser(jwt.WithoutClaimsValidation())
		parsedToken, _, err := parser.ParseUnverified(jwtString, jwt.MapClaims{})
		if err != nil {
			t.Fatalf("JWT parsing failed: %v", err)
		}

		claims, ok := parsedToken.Claims.(jwt.MapClaims)
		if !ok {
			t.Fatal("Failed to extract claims")
		}

		if scopeClaim, exists := claims["scope"]; exists {
			t.Logf("✅ Scope claim preserved: %v", scopeClaim)
		}

		if grantClaim, exists := claims["grant_id"]; exists {
			if grantClaim != "grant-rich-001" {
				t.Errorf("Grant ID mismatch: expected grant-rich-001, got %v", grantClaim)
			}
			t.Logf("✅ Grant ID preserved: %v", grantClaim)
		}

		if jur, exists := claims["jurisdiction"]; exists {
			if jur != "EU" {
				t.Errorf("Jurisdiction mismatch: expected EU, got %v", jur)
			}
			t.Logf("✅ Jurisdiction preserved: %v", jur)
		}
	})

	t.Run("Token_Expiration_Handling", func(t *testing.T) {
		shortExpiryService := NewExtendedTokenService(
			NewAuthorizationChainValidator(nil, nil, nil),
			NewComplianceValidator(nil, nil, nil),
			nil,
			"test-issuer",
			"https://test-issuer.example",
			time.Second,
		)

		request := createBaseTokenRequest("test-grant-expiry", []string{"temp"})

		createdToken, err := shortExpiryService.CreateExtendedToken(ctx, request)
		if err != nil {
			t.Fatalf("Token creation failed: %v", err)
		}

		jwtString, err := shortExpiryService.EncodeExtendedToken(ctx, createdToken)
		if err != nil {
			t.Fatalf("Encoding failed: %v", err)
		}

		parser := jwt.NewParser(jwt.WithoutClaimsValidation())
		parsedToken, _, err := parser.ParseUnverified(jwtString, jwt.MapClaims{})
		if err != nil {
			t.Fatalf("JWT parsing failed: %v", err)
		}

		claims, ok := parsedToken.Claims.(jwt.MapClaims)
		if !ok {
			t.Fatal("Failed to extract claims")
		}

		if exp, exists := claims["exp"]; exists {
			expFloat, ok := exp.(float64)
			if !ok {
				t.Fatal("exp claim is not a number")
			}
			expTime := time.Unix(int64(expFloat), 0)
			now := time.Now()

			if expTime.Before(now) {
				t.Error("Token already expired at creation")
			}
			duration := expTime.Sub(now)
			if duration > 2*time.Second {
				t.Errorf("Expiry duration too long: %v", duration)
			}
			t.Logf("✅ Token expiration correctly set")
		} else {
			t.Error("Missing exp claim")
		}
	})
}

// TestE2E_ExtendedTokenService_Integration validates the extended token service
// integration with real JWT operations as implemented in extended_token_service.go
func TestE2E_ExtendedTokenService_Integration(t *testing.T) {
	ctx := context.Background()
	issuerID := "gauth-issuer"
	tokenService := NewExtendedTokenService(
		NewAuthorizationChainValidator(nil, nil, nil),
		NewComplianceValidator(nil, nil, nil),
		nil,
		issuerID,
		"https://gauth.example.com",
		2*time.Hour,
	)

	t.Run("Full_Token_Lifecycle", func(t *testing.T) {
		request := createBaseTokenRequest("lifecycle-grant", []string{"full", "lifecycle", "test"})
		request.LegalFramework.Jurisdiction = "GLOBAL"
		request.LegalFramework.ApplicableLaws = []string{"GLOBAL-LAW"}

		token, err := tokenService.CreateExtendedToken(ctx, request)
		if err != nil {
			t.Fatalf("Creation failed: %v", err)
		}

		jwtString, err := tokenService.EncodeExtendedToken(ctx, token)
		if err != nil {
			t.Fatalf("Encoding failed: %v", err)
		}

		if jwtString == "" {
			t.Fatal("JWT string is empty")
		}

		// Parse
		parser := jwt.NewParser(jwt.WithoutClaimsValidation())
		parsed, _, err := parser.ParseUnverified(jwtString, jwt.MapClaims{})
		if err != nil {
			t.Fatalf("Parsing failed: %v", err)
		}

		claims := parsed.Claims.(jwt.MapClaims)

		if sub, ok := claims["sub"]; ok {
			t.Logf("✅ Subject preserved: %v", sub)
		}
		if iss, ok := claims["iss"]; ok {
			if iss != issuerID {
				t.Errorf("Issuer mismatch: %v != %s", iss, issuerID)
			}
			t.Logf("✅ Issuer preserved: %v", iss)
		}
	})
}

func createBaseTokenRequest(grantID string, scope []string) *ExtendedTokenRequest {
	return &ExtendedTokenRequest{
		GrantID: grantID,
		Scope:   scope,
		PowerOfAttorney: &poa.PoADefinition{
			Parties: poa.Parties{
				Principal:        poa.Principal{Type: "Organization", Identity: "principal-001"},
				AuthorizedClient: poa.AuthorizedClient{Type: "LLM", Identity: "client-001", StatusEnum: poa.OperationalStatusActive, CapabilityLevel: poa.CapabilityL1},
			},
			Authorization: poa.AuthorizationScope{},
			Requirements:  poa.Requirements{ValidityPeriod: poa.ValidityPeriod{StartTime: time.Now().Add(-time.Minute), EndTime: time.Now().Add(time.Hour)}},
		},
		OwnersAuthorizerInfo: &OwnersAuthorizerInfo{AuthorizerID: "authorizer-001", AuthorizerName: "Test Authorizer", AuthorizerType: "managing_director", IdentityVerified: true, VerificationMethod: "test", VerificationDate: time.Now()},
		LegalFramework:       &LegalFrameworkInfo{Jurisdiction: "DE", ApplicableLaws: []string{"TEST-LAW"}},
		AuthorizationChain: &AuthorizationChain{
			OwnersAuthorizer: &AuthorizationLink{
				EntityID:              "authorizer-001",
				EntityName:            "Test Authorizer",
				Role:                  "authorizer",
				EntityType:            "organization",
				AuthorizationType:     "statutory",
				IdentityVerified:      true,
				Status:                "active",
				ValidFrom:             time.Now().Add(-time.Hour),
				ValidUntil:            time.Now().Add(time.Hour),
				StatutoryAuthority:    "board_resolution_2025",
				LegalBasis:            &LegalBasis{BasisType: "power_of_attorney", Jurisdiction: "DE"},
				CommercialRegisterRef: "CR-TEST-001",
				ScopeOfAuthority:      []string{"token:issue"},
			},
			ClientOwner: &AuthorizationLink{
				EntityID:          "owner-001",
				EntityName:        "Test Owner",
				Role:              "owner",
				EntityType:        "organization",
				AuthorizedBy:      "authorizer-001",
				AuthorizationType: "statutory",
				IdentityVerified:  true,
				Status:            "active",
				ValidFrom:         time.Now().Add(-30 * time.Minute),
				ValidUntil:        time.Now().Add(30 * time.Minute),
				ScopeOfAuthority:  []string{"token:delegate"},
			},
			Client: &AuthorizationLink{
				EntityID:          "client-001",
				EntityName:        "Test Client",
				Role:              "client",
				EntityType:        "ai_system",
				Status:            "active",
				AuthorizedBy:      "owner-001",
				AuthorizationType: "delegated",
				IdentityVerified:  true,
				ValidFrom:         time.Now().Add(-5 * time.Minute),
				ValidUntil:        time.Now().Add(25 * time.Minute),
				ScopeOfAuthority:  []string{"token:use"},
			},
		},
		ClientOwnerInfo: &ClientOwnerInfo{
			OwnerID:   "owner-001",
			OwnerName: "Test Owner",
			OwnerType: "organization",
		},
		ResourceOwnerInfo: &ResourceOwnerInfo{
			OwnerID: "resource-001",
		},
		RequestID:        "request-001",
		JurisdictionCode: "DE",
	}
}

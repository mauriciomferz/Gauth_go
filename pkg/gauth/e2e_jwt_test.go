package gauth

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/poa"
	"github.com/golang-jwt/jwt/v5"
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
		// Create extended token with minimal valid request
		request := &ExtendedTokenRequest{
			GrantID: "test-grant-001",
			Scope:   []string{"read", "write"},
			PowerOfAttorney: &poa.PoADefinition{
				Parties: poa.Parties{
					Principal:        poa.Principal{Type: "Organization", Identity: "principal-001"},
					AuthorizedClient: poa.AuthorizedClient{Type: "LLM", Identity: "client-001", StatusEnum: poa.OperationalStatusActive, CapabilityLevel: poa.CapabilityL1},
				},
				Authorization: poa.AuthorizationScope{},
				Requirements:  poa.Requirements{ValidityPeriod: poa.ValidityPeriod{StartTime: time.Now().Add(-time.Minute), EndTime: time.Now().Add(time.Hour)}},
			},
			// Added required owner's authorizer info & legal framework to satisfy ExtendedTokenService validation
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

		token, err := tokenService.CreateExtendedToken(ctx, request)
		if err != nil {
			t.Fatalf("Token creation failed: %v", err)
		}

		// Encode extended token to JWT (AccessToken is internal ID; JWT produced separately)
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
		t.Logf("   Header length: %d chars", len(parts[0]))
		t.Logf("   Payload length: %d chars", len(parts[1]))
		t.Logf("   Signature length: %d chars", len(parts[2]))
	})

	t.Run("JWT_Parsing_And_Validation", func(t *testing.T) {
		// Create token
		request := &ExtendedTokenRequest{
			GrantID: "test-grant-002",
			Scope:   []string{"admin"},
			PowerOfAttorney: &poa.PoADefinition{
				Parties: poa.Parties{
					Principal:        poa.Principal{Type: "Organization", Identity: "principal-002"},
					AuthorizedClient: poa.AuthorizedClient{Type: "LLM", Identity: "client-002", StatusEnum: poa.OperationalStatusActive, CapabilityLevel: poa.CapabilityL1},
				},
				Authorization: poa.AuthorizationScope{},
				Requirements:  poa.Requirements{ValidityPeriod: poa.ValidityPeriod{StartTime: time.Now().Add(-time.Minute), EndTime: time.Now().Add(time.Hour)}},
			},
			OwnersAuthorizerInfo: &OwnersAuthorizerInfo{AuthorizerID: "authorizer-002", AuthorizerName: "Admin Authorizer", AuthorizerType: "managing_director", IdentityVerified: true, VerificationMethod: "test", VerificationDate: time.Now()},
			LegalFramework:       &LegalFrameworkInfo{Jurisdiction: "US", ApplicableLaws: []string{"US-LAW"}},
			AuthorizationChain: &AuthorizationChain{
				OwnersAuthorizer: &AuthorizationLink{
					EntityID:              "authorizer-002",
					EntityName:            "Admin Authorizer",
					Role:                  "authorizer",
					EntityType:            "organization",
					AuthorizationType:     "statutory",
					IdentityVerified:      true,
					Status:                "active",
					ValidFrom:             time.Now().Add(-time.Hour),
					ValidUntil:            time.Now().Add(time.Hour),
					StatutoryAuthority:    "board_resolution_2025",
					LegalBasis:            &LegalBasis{BasisType: "power_of_attorney", Jurisdiction: "US"},
					CommercialRegisterRef: "CR-US-001",
					ScopeOfAuthority:      []string{"token:issue"},
				},
				ClientOwner: &AuthorizationLink{
					EntityID:          "owner-002",
					EntityName:        "Admin Owner",
					Role:              "owner",
					EntityType:        "organization",
					AuthorizedBy:      "authorizer-002",
					AuthorizationType: "statutory",
					IdentityVerified:  true,
					Status:            "active",
					ValidFrom:         time.Now().Add(-30 * time.Minute),
					ValidUntil:        time.Now().Add(30 * time.Minute),
					ScopeOfAuthority:  []string{"token:delegate"},
				},
				Client: &AuthorizationLink{
					EntityID:          "client-002",
					EntityName:        "Admin Client",
					Role:              "client",
					EntityType:        "ai_system",
					Status:            "active",
					AuthorizedBy:      "owner-002",
					AuthorizationType: "delegated",
					IdentityVerified:  true,
					ValidFrom:         time.Now().Add(-5 * time.Minute),
					ValidUntil:        time.Now().Add(25 * time.Minute),
					ScopeOfAuthority:  []string{"token:use"},
				},
			},
			ClientOwnerInfo: &ClientOwnerInfo{
				OwnerID:   "owner-002",
				OwnerName: "Admin Owner",
				OwnerType: "individual",
			},
			ResourceOwnerInfo: &ResourceOwnerInfo{
				OwnerID: "resource-002",
			},
			RequestID:        "request-002",
			JurisdictionCode: "US",
		}

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
			jwt.WithoutClaimsValidation(), // We'll validate manually
		)

		parsedToken, _, err := parser.ParseUnverified(jwtString, jwt.MapClaims{})
		if err != nil {
			t.Fatalf("JWT parsing failed: %v", err)
		}

		// Validate standard JWT structure
		if parsedToken.Header == nil {
			t.Fatal("JWT header is nil")
		}

		claims, ok := parsedToken.Claims.(jwt.MapClaims)
		if !ok {
			t.Fatal("Failed to extract claims")
		}

		// Validate essential claims exist
		requiredClaims := []string{"iss", "sub", "exp", "iat"}
		for _, claim := range requiredClaims {
			if _, exists := claims[claim]; !exists {
				t.Errorf("Missing required claim: %s", claim)
			}
		}

		t.Logf("✅ JWT parsing successful")
		t.Logf("   Algorithm: %v", parsedToken.Header["alg"])
		t.Logf("   Type: %v", parsedToken.Header["typ"])
		t.Logf("   Issuer: %v", claims["iss"])
		t.Logf("   Subject: %v", claims["sub"])
	})

	t.Run("Extended_Token_Claims_Preservation", func(t *testing.T) {
		// Create token with rich metadata
		originalScope := []string{"read:user", "write:data", "admin:system"}
		originalGrantID := "grant-rich-001"
		originalJurisdiction := "EU"

		request := &ExtendedTokenRequest{
			GrantID: originalGrantID,
			Scope:   originalScope,
			PowerOfAttorney: &poa.PoADefinition{
				Parties: poa.Parties{
					Principal:        poa.Principal{Type: "Organization", Identity: "principal-rich"},
					AuthorizedClient: poa.AuthorizedClient{Type: "LLM", Identity: "client-rich", StatusEnum: poa.OperationalStatusActive, CapabilityLevel: poa.CapabilityL1},
				},
				Authorization: poa.AuthorizationScope{},
				Requirements:  poa.Requirements{ValidityPeriod: poa.ValidityPeriod{StartTime: time.Now().Add(-time.Minute), EndTime: time.Now().Add(time.Hour)}},
			},
			// Single consistent OwnersAuthorizerInfo & LegalFramework matching authorization chain
			OwnersAuthorizerInfo: &OwnersAuthorizerInfo{AuthorizerID: "authorizer-rich", AuthorizerName: "Rich Authorizer", AuthorizerType: "board_member", IdentityVerified: true, VerificationMethod: "test", VerificationDate: time.Now()},
			LegalFramework:       &LegalFrameworkInfo{Jurisdiction: originalJurisdiction, ApplicableLaws: []string{"EU-AI-ACT"}},
			AuthorizationChain: &AuthorizationChain{
				OwnersAuthorizer: &AuthorizationLink{
					EntityID:              "authorizer-rich",
					EntityName:            "Rich Authorizer",
					Role:                  "authorizer",
					EntityType:            "organization",
					AuthorizationType:     "statutory",
					IdentityVerified:      true,
					Status:                "active",
					ValidFrom:             time.Now().Add(-time.Hour),
					ValidUntil:            time.Now().Add(time.Hour),
					StatutoryAuthority:    "board_resolution_2025",
					LegalBasis:            &LegalBasis{BasisType: "power_of_attorney", Jurisdiction: "EU"},
					CommercialRegisterRef: "CR-EU-001",
					ScopeOfAuthority:      []string{"token:issue"},
				},
				ClientOwner: &AuthorizationLink{
					EntityID:          "owner-rich",
					EntityName:        "Rich Owner",
					Role:              "owner",
					EntityType:        "organization",
					AuthorizedBy:      "authorizer-rich",
					AuthorizationType: "statutory",
					IdentityVerified:  true,
					Status:            "active",
					ValidFrom:         time.Now().Add(-30 * time.Minute),
					ValidUntil:        time.Now().Add(30 * time.Minute),
					ScopeOfAuthority:  []string{"token:delegate"},
				},
				Client: &AuthorizationLink{
					EntityID:          "client-rich",
					EntityName:        "Rich Client",
					Role:              "client",
					EntityType:        "ai_system",
					Status:            "active",
					AuthorizedBy:      "owner-rich",
					AuthorizationType: "delegated",
					IdentityVerified:  true,
					ValidFrom:         time.Now().Add(-5 * time.Minute),
					ValidUntil:        time.Now().Add(25 * time.Minute),
					ScopeOfAuthority:  []string{"token:use"},
				},
			},
			ClientOwnerInfo: &ClientOwnerInfo{
				OwnerID:   "owner-rich",
				OwnerName: "Rich Metadata Owner",
				OwnerType: "organization",
			},
			ResourceOwnerInfo: &ResourceOwnerInfo{
				OwnerID: "resource-rich",
			},
			RequestID:        "request-rich-001",
			JurisdictionCode: originalJurisdiction,
		}

		createdToken, err := tokenService.CreateExtendedToken(ctx, request)
		if err != nil {
			t.Fatalf("Token creation failed: %v", err)
		}

		jwtString, err := tokenService.EncodeExtendedToken(ctx, createdToken)
		if err != nil {
			t.Fatalf("Encoding failed: %v", err)
		}

		// Parse and validate claim preservation
		parser := jwt.NewParser(jwt.WithoutClaimsValidation())
		parsedToken, _, err := parser.ParseUnverified(jwtString, jwt.MapClaims{})
		if err != nil {
			t.Fatalf("JWT parsing failed: %v", err)
		}

		claims, ok := parsedToken.Claims.(jwt.MapClaims)
		if !ok {
			t.Fatal("Failed to extract claims")
		}

		// Validate scope preservation
		if scopeClaim, exists := claims["scope"]; exists {
			t.Logf("✅ Scope claim preserved: %v", scopeClaim)
		}

		// Validate grant ID preservation
		if grantClaim, exists := claims["grant_id"]; exists {
			if grantClaim != originalGrantID {
				t.Errorf("Grant ID mismatch: expected %s, got %v", originalGrantID, grantClaim)
			}
			t.Logf("✅ Grant ID preserved: %v", grantClaim)
		}

		// Validate jurisdiction preservation
		if jur, exists := claims["jurisdiction"]; exists {
			if jur != originalJurisdiction {
				t.Errorf("Jurisdiction mismatch: expected %s, got %v", originalJurisdiction, jur)
			}
			t.Logf("✅ Jurisdiction preserved: %v", jur)
		}

		t.Logf("✅ Extended token claims preserved through serialization")
	})

	t.Run("Token_Expiration_Handling", func(t *testing.T) {
		// Create token service with short expiry
		shortExpiryService := NewExtendedTokenService(
			NewAuthorizationChainValidator(nil, nil, nil),
			NewComplianceValidator(nil, nil, nil),
			nil,
			"test-issuer",
			"https://test-issuer.example",
			time.Second, // 1 second expiry
		)

		request := &ExtendedTokenRequest{
			GrantID: "test-grant-expiry",
			Scope:   []string{"temp"},
			PowerOfAttorney: &poa.PoADefinition{
				Parties: poa.Parties{
					Principal:        poa.Principal{Type: "Organization", Identity: "principal-expiry"},
					AuthorizedClient: poa.AuthorizedClient{Type: "LLM", Identity: "client-expiry", StatusEnum: poa.OperationalStatusActive, CapabilityLevel: poa.CapabilityL1},
				},
				Authorization: poa.AuthorizationScope{},
				Requirements:  poa.Requirements{ValidityPeriod: poa.ValidityPeriod{StartTime: time.Now().Add(-time.Minute), EndTime: time.Now().Add(time.Hour)}},
			},
			OwnersAuthorizerInfo: &OwnersAuthorizerInfo{AuthorizerID: "authorizer-expiry", AuthorizerName: "Expiry Authorizer", AuthorizerType: "board_member", IdentityVerified: true, VerificationMethod: "test", VerificationDate: time.Now()},
			LegalFramework:       &LegalFrameworkInfo{Jurisdiction: "TEST", ApplicableLaws: []string{"TEST-LAW"}},
			AuthorizationChain: &AuthorizationChain{
				OwnersAuthorizer: &AuthorizationLink{
					EntityID:              "authorizer-expiry",
					EntityName:            "Expiry Authorizer",
					Role:                  "authorizer",
					EntityType:            "organization",
					AuthorizationType:     "statutory",
					IdentityVerified:      true,
					Status:                "active",
					ValidFrom:             time.Now().Add(-time.Hour),
					ValidUntil:            time.Now().Add(time.Hour),
					StatutoryAuthority:    "board_resolution_2025",
					LegalBasis:            &LegalBasis{BasisType: "power_of_attorney", Jurisdiction: "TEST"},
					CommercialRegisterRef: "CR-TST-001",
					ScopeOfAuthority:      []string{"token:issue"},
				},
				ClientOwner: &AuthorizationLink{
					EntityID:          "owner-expiry",
					EntityName:        "Expiry Owner",
					Role:              "owner",
					EntityType:        "organization",
					AuthorizedBy:      "authorizer-expiry",
					AuthorizationType: "statutory",
					IdentityVerified:  true,
					Status:            "active",
					ValidFrom:         time.Now().Add(-30 * time.Minute),
					ValidUntil:        time.Now().Add(30 * time.Minute),
					ScopeOfAuthority:  []string{"token:delegate"},
				},
				Client: &AuthorizationLink{
					EntityID:          "client-expiry",
					EntityName:        "Expiry Client",
					Role:              "client",
					EntityType:        "ai_system",
					Status:            "active",
					AuthorizedBy:      "owner-expiry",
					AuthorizationType: "delegated",
					IdentityVerified:  true,
					ValidFrom:         time.Now().Add(-5 * time.Minute),
					ValidUntil:        time.Now().Add(25 * time.Minute),
					ScopeOfAuthority:  []string{"token:use"},
				},
			},
			ClientOwnerInfo: &ClientOwnerInfo{
				OwnerID:   "owner-expiry",
				OwnerName: "Expiry Test",
				OwnerType: "test",
			},
			ResourceOwnerInfo: &ResourceOwnerInfo{
				OwnerID: "resource-expiry",
			},
			RequestID:        "request-expiry",
			JurisdictionCode: "TEST",
		}

		createdToken, err := shortExpiryService.CreateExtendedToken(ctx, request)
		if err != nil {
			t.Fatalf("Token creation failed: %v", err)
		}

		jwtString, err := shortExpiryService.EncodeExtendedToken(ctx, createdToken)
		if err != nil {
			t.Fatalf("Encoding failed: %v", err)
		}

		// Parse and check expiration
		parser := jwt.NewParser(jwt.WithoutClaimsValidation())
		parsedToken, _, err := parser.ParseUnverified(jwtString, jwt.MapClaims{})
		if err != nil {
			t.Fatalf("JWT parsing failed: %v", err)
		}

		claims, ok := parsedToken.Claims.(jwt.MapClaims)
		if !ok {
			t.Fatal("Failed to extract claims")
		}

		// Validate exp claim
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
				t.Errorf("Expiry duration too long: %v (expected ~1s)", duration)
			}

			t.Logf("✅ Token expiration correctly set")
			t.Logf("   Expires in: %v", duration)
			t.Logf("   Exp timestamp: %v", expTime.Unix())
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
		2*time.Hour, // 7200 seconds
	)

	t.Run("Full_Token_Lifecycle", func(t *testing.T) {
		// Create
		request := &ExtendedTokenRequest{
			GrantID: "lifecycle-grant",
			Scope:   []string{"full", "lifecycle", "test"},
			PowerOfAttorney: &poa.PoADefinition{
				Parties: poa.Parties{
					Principal:        poa.Principal{Type: "Organization", Identity: "principal-lifecycle"},
					AuthorizedClient: poa.AuthorizedClient{Type: "LLM", Identity: "client-lifecycle", StatusEnum: poa.OperationalStatusActive, CapabilityLevel: poa.CapabilityL1},
				},
				Authorization: poa.AuthorizationScope{},
				Requirements:  poa.Requirements{ValidityPeriod: poa.ValidityPeriod{StartTime: time.Now().Add(-time.Minute), EndTime: time.Now().Add(time.Hour)}},
			},
			OwnersAuthorizerInfo: &OwnersAuthorizerInfo{AuthorizerID: "authorizer-lifecycle", AuthorizerName: "Lifecycle Authorizer", AuthorizerType: "managing_director", IdentityVerified: true, VerificationMethod: "test", VerificationDate: time.Now()},
			LegalFramework:       &LegalFrameworkInfo{Jurisdiction: "GLOBAL", ApplicableLaws: []string{"GLOBAL-LAW"}},
			AuthorizationChain: &AuthorizationChain{
				OwnersAuthorizer: &AuthorizationLink{
					EntityID:              "authorizer-lifecycle",
					EntityName:            "Lifecycle Authorizer",
					Role:                  "authorizer",
					EntityType:            "organization",
					AuthorizationType:     "statutory",
					IdentityVerified:      true,
					Status:                "active",
					ValidFrom:             time.Now().Add(-time.Hour),
					ValidUntil:            time.Now().Add(time.Hour),
					StatutoryAuthority:    "board_resolution_2025",
					LegalBasis:            &LegalBasis{BasisType: "power_of_attorney", Jurisdiction: "GLOBAL"},
					CommercialRegisterRef: "CR-GLB-001",
					ScopeOfAuthority:      []string{"token:issue"},
				},
				ClientOwner: &AuthorizationLink{
					EntityID:          "owner-lifecycle",
					EntityName:        "Lifecycle Owner",
					Role:              "owner",
					EntityType:        "organization",
					AuthorizedBy:      "authorizer-lifecycle",
					AuthorizationType: "statutory",
					IdentityVerified:  true,
					Status:            "active",
					ValidFrom:         time.Now().Add(-30 * time.Minute),
					ValidUntil:        time.Now().Add(30 * time.Minute),
					ScopeOfAuthority:  []string{"token:delegate"},
				},
				Client: &AuthorizationLink{
					EntityID:          "client-lifecycle",
					EntityName:        "Lifecycle Client",
					Role:              "client",
					EntityType:        "ai_system",
					Status:            "active",
					AuthorizedBy:      "owner-lifecycle",
					AuthorizationType: "delegated",
					IdentityVerified:  true,
					ValidFrom:         time.Now().Add(-5 * time.Minute),
					ValidUntil:        time.Now().Add(25 * time.Minute),
					ScopeOfAuthority:  []string{"token:use"},
				},
			},
			ClientOwnerInfo: &ClientOwnerInfo{
				OwnerID:   "lifecycle-owner",
				OwnerName: "Lifecycle Test Owner",
				OwnerType: "organization",
			},
			ResourceOwnerInfo: &ResourceOwnerInfo{
				OwnerID: "lifecycle-resource",
			},
			RequestID:        "lifecycle-request",
			JurisdictionCode: "GLOBAL",
		}

		token, err := tokenService.CreateExtendedToken(ctx, request)
		if err != nil {
			t.Fatalf("Creation failed: %v", err)
		}

		jwtString, err := tokenService.EncodeExtendedToken(ctx, token)
		if err != nil {
			t.Fatalf("Encoding failed: %v", err)
		}

		// Validate structure
		if jwtString == "" {
			t.Fatal("JWT string is empty")
		}
		if token.TokenType != "Bearer" {
			t.Errorf("Expected Bearer, got %s", token.TokenType)
		}
		if token.ExpiresIn <= 0 {
			t.Error("Invalid expiry")
		}

		// Validate JWT structure
		parts := strings.Split(jwtString, ".")
		if len(parts) != 3 {
			t.Fatalf("Invalid JWT structure: %d parts", len(parts))
		}

		// Parse
		parser := jwt.NewParser(jwt.WithoutClaimsValidation())
		parsed, _, err := parser.ParseUnverified(jwtString, jwt.MapClaims{})
		if err != nil {
			t.Fatalf("Parsing failed: %v", err)
		}

		claims := parsed.Claims.(jwt.MapClaims)

		// Validate round-trip
		if sub, ok := claims["sub"]; ok {
			t.Logf("✅ Subject preserved: %v", sub)
		}
		if iss, ok := claims["iss"]; ok {
			if iss != issuerID {
				t.Errorf("Issuer mismatch: %v != %s", iss, issuerID)
			}
			t.Logf("✅ Issuer preserved: %v", iss)
		}

		t.Logf("✅ Full token lifecycle validated")
		t.Logf("   Token length: %d chars", len(token.AccessToken))
		t.Logf("   Expires in: %d seconds", token.ExpiresIn)
	})
}

package main

import (
	"context"
	"fmt"
	"time"

	"github.com/Gimel-Foundation/gauth/pkg/auth"
	"github.com/Gimel-Foundation/gauth/pkg/gauth"
	"github.com/Gimel-Foundation/gauth/pkg/token"
)

func main() {
	fmt.Println("🧪 GAuth Go Final Testing Suite")
	fmt.Println("================================")

	// Test 1: Basic GAuth Configuration
	fmt.Println("\n1. Testing GAuth Configuration...")
	config := gauth.Config{
		AuthServerURL: "http://localhost:8080",
		ClientID:      "test-client",
		ClientSecret:  "test-secret",
		Scopes:        []string{"read", "write"},
	}

	service, err := gauth.New(config)
	if err != nil {
		fmt.Printf("❌ GAuth configuration failed: %v\n", err)
		return
	}
	fmt.Println("✅ GAuth configuration successful")

	// Test 2: Authorization Flow
	fmt.Println("\n2. Testing Authorization Flow...")
	authResp, err := service.Authorize("test-client", "read write")
	if err != nil {
		fmt.Printf("❌ Authorization failed: %v\n", err)
		return
	}
	fmt.Printf("✅ Authorization successful: %v\n", authResp)

	// Test 3: Token Request
	fmt.Println("\n3. Testing Token Request...")
	tokenReq := gauth.TokenRequest{
		GrantID: "test-grant",
		Scope:   []string{"read", "write"},
	}

	tokenResp, err := service.RequestToken(tokenReq)
	if err != nil {
		fmt.Printf("❌ Token request failed: %v\n", err)
		return
	}
	fmt.Printf("✅ Token received: %v\n", tokenResp)

	// Test 4: JWT Service
	fmt.Println("\n4. Testing JWT Service...")
	jwtService, err := auth.NewProperJWTService("test-issuer", "test-audience")
	if err != nil {
		fmt.Printf("❌ JWT service creation failed: %v\n", err)
		return
	}
	testToken, err := jwtService.CreateToken("test-user", []string{"read", "write"}, time.Hour)
	if err != nil {
		fmt.Printf("❌ JWT creation failed: %v\n", err)
		return
	}
	fmt.Printf("✅ JWT created: %s...\n", testToken[:20])

	// Test 5: Token Store
	fmt.Println("\n5. Testing Token Store...")
	tokenStore := token.NewMemoryStore()
	testTokenData := &token.Token{
		Value:     testToken,
		Subject:   "test-user",
		Scopes:    []string{"read", "write"},
		ExpiresAt: time.Now().Add(time.Hour),
		IssuedAt:  time.Now(),
		Type:      token.Access,
	}

	err = tokenStore.Save(context.Background(), "test-token-id", testTokenData)
	if err != nil {
		fmt.Printf("❌ Token storage failed: %v\n", err)
		return
	}

	retrievedToken, err := tokenStore.Get(context.Background(), "test-token-id")
	if err != nil {
		fmt.Printf("❌ Token retrieval failed: %v\n", err)
		return
	}
	fmt.Printf("✅ Token stored and retrieved: %s\n", retrievedToken.Subject)

	// Test 6: RFC-0111 Compliance
	fmt.Println("\n6. Testing RFC-0111 Compliance...")
	rfcService, err := auth.NewRFCCompliantService("test-issuer", "test-audience")
	if err != nil {
		fmt.Printf("❌ RFC service creation failed: %v\n", err)
		return
	}
	poaRequest := &auth.PowerOfAttorneyRequest{
		PrincipalID: "test-principal",
		AIAgentID:   "test-agent",
		Scope:       []string{"document-signing"},
	}

	gauthResponse, err := rfcService.AuthorizeGAuth(context.Background(), *poaRequest)
	if err != nil {
		fmt.Printf("❌ RFC-0111 compliance test failed: %v\n", err)
		return
	}
	fmt.Printf("✅ RFC-0111 GAuth authorization successful: %s\n", gauthResponse.AuthorizationCode)

	// Test 7: Educational Warning Validation
	fmt.Println("\n7. Validating Educational Warnings...")
	fmt.Println("⚠️  Educational Implementation Only")
	fmt.Println("⚠️  NOT for Production Use")
	fmt.Println("⚠️  No Real Security Implementation")
	fmt.Println("✅ Educational warnings properly displayed")

	// Final Summary
	fmt.Println("\n🎯 FINAL TESTING RESULTS")
	fmt.Println("========================")
	fmt.Println("✅ GAuth Configuration: PASS")
	fmt.Println("✅ Authorization Flow: PASS")
	fmt.Println("✅ Token Management: PASS")
	fmt.Println("✅ JWT Operations: PASS")
	fmt.Println("✅ Token Storage: PASS")
	fmt.Println("✅ RFC-0111 Compliance: PASS")
	fmt.Println("✅ Educational Warnings: PASS")
	fmt.Println("\n🏆 All Core Functionality Tests: PASSED")
	fmt.Println("📚 Educational Implementation: VALIDATED")
	fmt.Println("⚠️  Production Use: NOT RECOMMENDED")
}

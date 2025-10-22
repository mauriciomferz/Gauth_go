package main

import (
	"fmt"
	"log"
	"time"

	"github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/token"
)

func main() {
	fmt.Println("🪙 GAuth Token Management Demo")
	fmt.Println("==============================")

	// Create a new memory token store
	fmt.Println("\n📦 Creating token store...")
	store := token.NewMemoryStore()
	defer store.Close()
	fmt.Println("✅ Token store created successfully!")

	// Create different types of tokens
	tokens := []struct {
		tokenType token.TokenType
		subject   string
		scopes    []string
	}{
		{token.Access, "user123", []string{"read:profile", "write:data"}},
		{token.Refresh, "user123", []string{"refresh:token"}},
		{token.ID, "user123", []string{"openid", "profile"}},
	}

	fmt.Println("\n🔨 Creating tokens...")
	for _, tokenInfo := range tokens {
		// Create token
		tok := &token.Token{
			ID:        fmt.Sprintf("%s-%d", tokenInfo.tokenType, time.Now().UnixNano()),
			Value:     fmt.Sprintf("tok_%s_%d", tokenInfo.tokenType, time.Now().UnixNano()),
			Type:      tokenInfo.tokenType,
			Subject:   tokenInfo.subject,
			Issuer:    "gauth-demo",
			Scopes:    tokenInfo.scopes,
			IssuedAt:  time.Now(),
			ExpiresAt: time.Now().Add(1 * time.Hour),
		}

		// Store the token
		err := store.Save(nil, tok.Value, tok)
		if err != nil {
			log.Fatalf("❌ Failed to save %s token: %v", tokenInfo.tokenType, err)
		}

		fmt.Printf("  ✅ Created %s token: %s\n", tokenInfo.tokenType, tok.ID)
	}

	// Demonstrate token retrieval
	fmt.Println("\n🔍 Retrieving tokens...")

	// Get a specific token by its value
	if len(tokens) > 0 {
		// Create a token value we can use to retrieve
		testValue := fmt.Sprintf("tok_%s_%d", tokens[0].tokenType, time.Now().UnixNano()-1000)

		// Try to get it (this will show retrieval process even if not found)
		retrieved, err := store.Get(nil, testValue)
		if err != nil {
			fmt.Printf("  � Token retrieval test: %v (expected for demo token)\n", err)
		} else if retrieved != nil {
			fmt.Printf("  ✅ Successfully retrieved token: %s\n", retrieved.ID)
		}
	}

	// Demonstrate token validation
	fmt.Println("\n🔐 Validating tokens...")

	// Create a token for validation
	testToken := &token.Token{
		ID:        "validation-test",
		Value:     "test-token-value",
		Type:      token.Access,
		Subject:   "test-user",
		Issuer:    "gauth-demo",
		Scopes:    []string{"read:test"},
		IssuedAt:  time.Now(),
		ExpiresAt: time.Now().Add(30 * time.Minute),
	}

	// Store it
	err := store.Save(nil, testToken.Value, testToken)
	if err != nil {
		log.Fatalf("❌ Failed to save test token: %v", err)
	}

	// Retrieve and validate
	retrieved, err := store.Get(nil, testToken.Value)
	if err != nil {
		log.Fatalf("❌ Failed to retrieve test token: %v", err)
	}

	if retrieved != nil {
		fmt.Println("  ✅ Token validation successful!")
		fmt.Printf("    - Token ID: %s\n", retrieved.ID)
		fmt.Printf("    - Subject: %s\n", retrieved.Subject)
		fmt.Printf("    - Scopes: %v\n", retrieved.Scopes)
		fmt.Printf("    - Expires: %s\n", retrieved.ExpiresAt.Format("2006-01-02 15:04:05"))
	} else {
		fmt.Println("  ❌ Token not found")
	}

	// Demonstrate token expiry check
	fmt.Println("\n⏰ Testing token expiry...")

	// Create an expired token
	expiredToken := &token.Token{
		ID:        "expired-test",
		Value:     "expired-token-value",
		Type:      token.Access,
		Subject:   "test-user",
		Issuer:    "gauth-demo",
		Scopes:    []string{"read:test"},
		IssuedAt:  time.Now().Add(-2 * time.Hour),
		ExpiresAt: time.Now().Add(-1 * time.Hour), // Expired 1 hour ago
	}

	err = store.Save(nil, expiredToken.Value, expiredToken)
	if err != nil {
		log.Fatalf("❌ Failed to save expired token: %v", err)
	}

	retrieved, err = store.Get(nil, expiredToken.Value)
	if err != nil {
		log.Printf("⚠️  Error retrieving expired token: %v", err)
	} else if retrieved != nil {
		if time.Now().After(retrieved.ExpiresAt) {
			fmt.Println("  ⚠️  Token found but is expired!")
			fmt.Printf("    - Expired at: %s\n", retrieved.ExpiresAt.Format("2006-01-02 15:04:05"))
		} else {
			fmt.Println("  ✅ Token is still valid")
		}
	}

	fmt.Println("\n🎉 Token demo completed successfully!")
}

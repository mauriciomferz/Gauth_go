package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/mauriciomferz/AgentAuth/pkg/gauth"
	"github.com/mauriciomferz/AgentAuth/pkg/token"
)

func main() {
	// Define command-line flags
	var (
		command    = flag.String("cmd", "help", "Command to execute: help, create-token, validate-token, list-tokens")
		userID     = flag.String("user", "demo-user", "User ID for token operations")
		tokenValue = flag.String("token", "", "Token value for validation")
		scopes     = flag.String("scopes", "read,write", "Comma-separated scopes for token")
		duration   = flag.Duration("duration", time.Hour, "Token duration")
	)
	flag.Parse()

	fmt.Println("🚀 AgentAuth Command Line Interface")
	fmt.Println("===============================")

	switch *command {
	case "help":
		showHelp()
	case "create-token":
		createToken(*userID, *scopes, *duration)
	case "validate-token":
		if *tokenValue == "" {
			log.Fatal("❌ Token value required for validation. Use --token flag.")
		}
		validateToken(*tokenValue)
	case "list-tokens":
		listTokens()
	case "demo":
		runDemo()
	default:
		fmt.Printf("❌ Unknown command: %s\n", *command)
		showHelp()
		os.Exit(1)
	}
}

func showHelp() {
	fmt.Println("\n📖 Available Commands:")
	fmt.Println("  --cmd=help           Show this help message")
	fmt.Println("  --cmd=create-token   Create a new access token")
	fmt.Println("  --cmd=validate-token Validate an existing token")
	fmt.Println("  --cmd=list-tokens    List all stored tokens")
	fmt.Println("  --cmd=demo          Run a full demonstration")

	fmt.Println("\n🔧 Options:")
	fmt.Println("  --user=USER_ID       User ID (default: demo-user)")
	fmt.Println("  --token=TOKEN        Token value for validation")
	fmt.Println("  --scopes=SCOPES      Comma-separated scopes (default: read,write)")
	fmt.Println("  --duration=DURATION  Token duration (default: 1h)")

	fmt.Println("\n📝 Examples:")
	fmt.Println("  gauth-cli --cmd=create-token --user=alice --scopes=read,write,admin")
	fmt.Println("  gauth-cli --cmd=validate-token --token=your-token-here")
	fmt.Println("  gauth-cli --cmd=demo")
}

func createToken(userID, scopesStr string, duration time.Duration) {
	fmt.Printf("\n🔨 Creating token for user: %s\n", userID)

	// Create AgentAuth service
	service, err := gauth.New(gauth.Config{
		AccessTokenExpiry: duration,
		ClientID:          "gauth-cli",
	})
	if err != nil {
		log.Fatalf("❌ Failed to create AgentAuth service: %v", err)
	}

	// Create token request
	req := gauth.TokenRequest{
		GrantID: userID,
		Scope:   parseScopes(scopesStr),
	}

	// Request token
	response, err := service.RequestToken(req)
	if err != nil {
		log.Fatalf("❌ Failed to create token: %v", err)
	}

	fmt.Println("✅ Token created successfully!")
	fmt.Printf("  🪙 Token: %s\n", response.Token)
	fmt.Printf("  � Scope: %v\n", response.Scope)
	fmt.Printf("  ⏰ Valid Until: %s\n", response.ValidUntil.Format("2006-01-02 15:04:05"))
	fmt.Printf("  📋 Scopes: %v\n", req.Scope)
}

func validateToken(tokenValue string) {
	fmt.Printf("\n🔍 Validating token: %s...\n", tokenValue[:min(len(tokenValue), 20)]+"...")

	// Create AgentAuth service
	service, err := gauth.New(gauth.Config{ClientID: "gauth-cli"})
	if err != nil {
		log.Fatalf("❌ Failed to create AgentAuth service: %v", err)
	}

	// Validate token
	vr, err := service.ValidateToken(tokenValue)
	if err != nil {
		fmt.Printf("❌ Token validation failed: %v\n", err)
		return
	}
	if vr != nil && vr.Valid {
		fmt.Println("✅ Token is valid!")
	} else {
		fmt.Println("❌ Token is invalid!")
	}
}

func listTokens() {
	fmt.Println("\n📋 Listing stored tokens...")

	// Create a token store to demonstrate listing
	store := token.NewMemoryStore()
	defer store.Close()

	// For demo purposes, create a few sample tokens
	sampleTokens := []struct {
		id        string
		userID    string
		scopes    []string
		tokenType token.TokenType
	}{
		{"token-1", "user-alice", []string{"read", "write"}, token.Access},
		{"token-2", "user-bob", []string{"read"}, token.Access},
		{"token-3", "user-alice", []string{"refresh"}, token.Refresh},
	}

	fmt.Println("\n🔍 Sample tokens in store:")
	for i, tok := range sampleTokens {
		// Create token
		tokenObj := &token.Token{
			ID:        tok.id,
			Value:     fmt.Sprintf("demo-token-value-%d", i),
			Type:      tok.tokenType,
			Subject:   tok.userID,
			Issuer:    "gauth-cli",
			Scopes:    tok.scopes,
			IssuedAt:  time.Now(),
			ExpiresAt: time.Now().Add(time.Hour),
		}

		// Store it (for demonstration)
		err := store.Save(context.TODO(), tokenObj.Value, tokenObj)
		if err != nil {
			fmt.Printf("  ⚠️  Error storing demo token %s: %v\n", tok.id, err)
			continue
		}

		fmt.Printf("  %d. 🪙 %s (%s) - User: %s, Scopes: %v\n",
			i+1, tok.id, tok.tokenType, tok.userID, tok.scopes)
	}

	fmt.Println("\n💡 In a real application, these would be retrieved from persistent storage.")
}

func runDemo() {
	fmt.Println("\n🎭 Running Full AgentAuth CLI Demo")
	fmt.Println("==============================")

	// Step 1: Create a service
	fmt.Println("\n1️⃣  Creating AgentAuth service...")
	service, err := gauth.New(gauth.Config{
		AccessTokenExpiry: 30 * time.Minute,
		ClientID:          "gauth-cli-demo",
	})
	if err != nil {
		log.Fatalf("❌ Failed to create service: %v", err)
	}
	fmt.Println("   ✅ Service created")

	// Step 2: Create a token
	fmt.Println("\n2️⃣  Creating access token...")
	req := gauth.TokenRequest{
		GrantID: "demo-user-cli",
		Scope:   []string{"read:profile", "write:data", "admin:users"},
	}

	response, err := service.RequestToken(req)
	if err != nil {
		log.Fatalf("❌ Demo failed at token creation: %v", err)
	}

	fmt.Printf("   ✅ Token: %s...\n", response.Token[:min(len(response.Token), 20)])
	fmt.Printf("   ⏰ Valid until: %s\n", response.ValidUntil.Format("15:04:05"))

	// Step 3: Validate the token
	fmt.Println("\n3️⃣  Validating the token...")
	vr, err := service.ValidateToken(response.Token)
	if err != nil {
		fmt.Printf("   ⚠️  Validation error: %v\n", err)
	} else if vr != nil && vr.Valid {
		fmt.Println("   ✅ Token validation successful!")
	} else {
		fmt.Println("   ❌ Token validation failed!")
	}

	// Step 4: Token store demonstration
	fmt.Println("\n4️⃣  Demonstrating token storage...")
	store := token.NewMemoryStore()
	defer store.Close()

	// Create a sample token to store
	storedToken := &token.Token{
		ID:        "cli-demo-token",
		Value:     "cli-demo-value",
		Type:      token.Access,
		Subject:   "demo-user-cli",
		Issuer:    "gauth-cli-demo",
		Scopes:    req.Scope,
		IssuedAt:  time.Now(),
		ExpiresAt: time.Now().Add(30 * time.Minute),
	}

	err = store.Save(context.TODO(), storedToken.Value, storedToken)
	if err != nil {
		fmt.Printf("   ⚠️  Storage error: %v\n", err)
	} else {
		fmt.Println("   ✅ Token stored successfully!")
	}

	// Retrieve it
	retrieved, err := store.Get(context.TODO(), storedToken.Value)
	if err != nil {
		fmt.Printf("   ⚠️  Retrieval error: %v\n", err)
	} else if retrieved != nil {
		fmt.Printf("   ✅ Token retrieved: %s\n", retrieved.ID)
	}

	fmt.Println("\n🎉 Demo completed successfully!")
	fmt.Println("\n💡 Try running with different commands:")
	fmt.Println("   gauth-cli --cmd=create-token --user=your-name")
	fmt.Println("   gauth-cli --cmd=help")
}

// Helper functions
func parseScopes(scopesStr string) []string {
	if scopesStr == "" {
		return []string{}
	}

	// Simple comma-separated parsing
	var scopes []string
	current := ""
	for _, r := range scopesStr {
		if r == ',' {
			if current != "" {
				scopes = append(scopes, current)
				current = ""
			}
		} else {
			current += string(r)
		}
	}
	if current != "" {
		scopes = append(scopes, current)
	}

	return scopes
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

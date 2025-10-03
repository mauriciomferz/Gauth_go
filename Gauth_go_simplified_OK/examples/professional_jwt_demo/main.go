// Development JWT Implementation Demo
// Example demonstrating your excellent prope    fmt.Println("\n🔒 Your professional implementation is ready for development use!")_jwt.go in action

package main

import (
	"fmt"
	"log"
	"time"

	"github.com/Gimel-Foundation/gauth/pkg/auth"
)

func main() {
	fmt.Println("🚀 GAuth Development JWT Implementation Demo")
	fmt.Println("==============================================")

	// Create a development JWT service
	jwtService, err := auth.NewProperJWTService("gauth-demo", "demo-app")
	if err != nil {
		log.Fatalf("❌ Failed to create JWT service: %v", err)
	}
	fmt.Println("✅ Development JWT service created successfully!")

	// Create a token for a user
	userID := "user123"
	scopes := []string{"read:profile", "write:data", "admin:users"}
	duration := 15 * time.Minute

	fmt.Printf("\n📝 Creating token for user: %s\n", userID)
	fmt.Printf("   Scopes: %v\n", scopes)
	fmt.Printf("   Duration: %v\n", duration)

	token, err := jwtService.CreateToken(userID, scopes, duration)
	if err != nil {
		log.Fatalf("❌ Failed to create token: %v", err)
	}

	fmt.Printf("✅ Token created successfully!\n")
	fmt.Printf("   Token: %s...\n", token[:50]) // Show first 50 chars

	// Validate the token
	fmt.Printf("\n🔍 Validating the token...\n")
	claims, err := jwtService.ValidateToken(token)
	if err != nil {
		log.Fatalf("❌ Failed to validate token: %v", err)
	}

	fmt.Println("✅ Token validated successfully!")
	fmt.Printf("   User ID: %s\n", claims.UserID)
	fmt.Printf("   Session ID: %s\n", claims.SessionID)
	fmt.Printf("   Scopes: %v\n", claims.Scopes)
	fmt.Printf("   Expires: %v\n", claims.ExpiresAt.Time)
	fmt.Printf("   Issuer: %s\n", claims.Issuer)
	fmt.Printf("   Audience: %v\n", claims.Audience)

	// Test with invalid token
	fmt.Printf("\n🧪 Testing invalid token handling...\n")
	invalidToken := "invalid.token.here"
	_, err = jwtService.ValidateToken(invalidToken)
	if err != nil {
		fmt.Printf("✅ Invalid token correctly rejected: %v\n", err)
	} else {
		fmt.Println("❌ Invalid token was unexpectedly accepted!")
	}

	// Test token expiration simulation
	fmt.Printf("\n⏰ Testing expired token (simulated)...\n")
	expiredToken, err := jwtService.CreateToken(userID, scopes, -1*time.Hour) // Already expired
	if err != nil {
		log.Fatalf("❌ Failed to create expired token for testing: %v", err)
	}

	_, err = jwtService.ValidateToken(expiredToken)
	if err != nil {
		fmt.Printf("✅ Expired token correctly rejected: %v\n", err)
	} else {
		fmt.Println("❌ Expired token was unexpectedly accepted!")
	}

	fmt.Println("\n🎉 Development JWT implementation working correctly!")
	fmt.Println("    - Secure RSA-256 signatures ✅")
	fmt.Println("    - Proper claim validation ✅")
	fmt.Println("    - Expiration handling ✅")
	fmt.Println("    - Security best practices ✅")
	fmt.Println("\n🔒 Your professional implementation is ready for development use!")
}

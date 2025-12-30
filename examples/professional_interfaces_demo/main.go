// Professional Interface Usage Example
// Shows how to use the new professional interfaces
package main

import (
	"fmt"
	"log"
	"time"

	"github.com/mauriciomferz/AgentAuth/pkg/auth"
)

func main() {
	fmt.Println("🔧 Professional Interface Integration Demo")
	fmt.Println("==========================================")

	// Create professional configuration
	config := auth.ProfessionalConfig{
		Issuer:            "agentauth-mesh",
		Audience:          "service-mesh",
		TokenExpiry:       30 * time.Minute,
		ServiceID:         "api-service-1",
		MeshID:            "demo-mesh", // Beta demonstration mesh ID
		UseSecureDefaults: true,
	}

	// Create professional authentication service
	authService, err := auth.NewProfessionalAuthService(config)
	if err != nil {
		log.Fatalf("❌ Failed to create professional auth service: %v", err)
	}
	fmt.Println("✅ Professional authentication service created!")

	// Create a service token
	userID := "service-user-123"
	scopes := []string{"service:read", "service:write", "mesh:communicate"}

	fmt.Printf("\n📝 Creating service token...\n")
	fmt.Printf("   Service ID: %s\n", config.ServiceID)
	fmt.Printf("   Mesh ID: %s\n", config.MeshID)
	fmt.Printf("   Scopes: %v\n", scopes)

	token, err := authService.CreateToken(userID, scopes, config.TokenExpiry)
	if err != nil {
		log.Fatalf("❌ Failed to create service token: %v", err)
	}
	if len(token) > 50 {
		fmt.Printf("✅ Service token created: %s...\n", token[:50])
	} else {
		fmt.Printf("✅ Service token created: %s\n", token)
	}

	// Validate the token using professional interface
	fmt.Printf("\n🔍 Validating token using professional interface...\n")
	claims, err := authService.ValidateToken(token)
	if err != nil {
		log.Fatalf("❌ Failed to validate token: %v", err)
	}

	fmt.Println("✅ Token validation successful!")
	fmt.Printf("   User ID: %s\n", claims.UserID)
	fmt.Printf("   Scopes: %v\n", claims.Scopes)
	fmt.Printf("   Expires: %v\n", claims.ExpiresAt)

	fmt.Println("\n🎯 Professional Interface Benefits:")
	fmt.Println("   - Clean, type-safe interfaces ✅")
	fmt.Println("   - Development JWT implementation ✅")
	fmt.Println("   - Service mesh integration ready ✅")
	fmt.Println("   - Backward compatibility maintained ✅")
	fmt.Println("\n🚀 Ready for mesh package integration!")
}

// token_example.go
// Title: Minimal Token Lifecycle
// Description: Demonstrates creating and validating a simple beta demonstration token with mock claims.
package main

import (
	"fmt"
	"time"

	auth "github.com/mauriciomferz/Gauth_go/pkg/auth"
)

func main() {
	svc, err := auth.NewProfessionalAuthService(auth.ProfessionalConfig{
		Issuer:      "demo-issuer",
		Audience:    "demo-audience",
		TokenExpiry: time.Hour,
	})
	if err != nil {
		fmt.Println("Failed to create auth service:", err)
		return
	}
	// Create a token
	token, err := svc.CreateToken("user-123", []string{"read", "write"}, time.Hour)
	if err != nil {
		fmt.Println("Token creation failed:", err)
		return
	}
	fmt.Println("Token created:", token)
	// Validate the token
	claims, err := svc.ValidateToken(token)
	if err != nil {
		fmt.Println("Token validation failed:", err)
		return
	}
	fmt.Printf("Token valid for user: %s, scopes: %v, expires: %s\n", claims.UserID, claims.Scopes, claims.ExpiresAt.Format(time.RFC3339))
}

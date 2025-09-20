package main

import (
	"fmt"
	"log"
	"time"

	"github.com/mauriciomferz/Gauth_go/pkg/gauth"
)

func main() {
	fmt.Println("GAuth RFC111 Demo Application")
	fmt.Println("==============================")

	config := &gauth.Config{
		AuthServerURL:     "https://auth.example.com",
		ClientID:          "demo-client",
		ClientSecret:      "demo-secret",
		Scopes:            []string{"transaction:execute", "read", "write"},
		AccessTokenExpiry: time.Hour,
	}

	gauthService, err := gauth.New(config, nil)
	if err != nil {
		log.Fatalf("Failed to initialize GAuth: %v", err)
	}
	fmt.Println("1. GAuth service initialized")

	authReq := gauth.AuthorizationRequest{
		ClientID: "demo-client",
		Scopes:   []string{"transaction:execute"},
	}

	grant, err := gauthService.InitiateAuthorization(authReq)
	if err != nil {
		log.Fatalf("Authorization failed: %v", err)
	}
	fmt.Printf("✓ Authorization grant received: %s\n", grant.GrantID)
}

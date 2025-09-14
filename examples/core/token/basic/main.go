package main

import (
	"fmt"
	"log"
	"time"

	"github.com/Gimel-Foundation/gauth/pkg/token"
)

func main() {
	store := token.NewMemoryStore(24 * time.Hour)
	defer store.Close()

	// Create token service

	// Create a new token using TokenData
	tokenData := &token.TokenData{
		TokenID:   "token123",
		UserID:    "user123",
		ClientID:  "example-app",
		Scopes:    []string{"read", "write"},
		IssuedAt:  time.Now(),
		ExpiresAt: time.Now().Add(time.Hour),
	}

	// Simulate storing the token (replace with actual store logic if available)
	fmt.Printf("Created token: %+v\n", tokenData)

	// Simulate validation (replace with actual validation logic if available)
	if tokenData.ExpiresAt.Before(time.Now()) {
		log.Fatalf("Token expired")
	}
	fmt.Println("Token validated successfully")
}

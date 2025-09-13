package tokenmanagement
// Package demo provides examples of GAuth usage
package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/Gimel-Foundation/gauth/pkg/gauth"
	"github.com/Gimel-Foundation/gauth/pkg/events"
)

func main() {
	// Create a custom event handler for logging
	eventHandler := events.HandlerFunc(func(event events.Event) {
		fmt.Printf("[%s] %s: %s - %s\n", 
			event.Type, 
			event.Action, 
			event.Status, 
			event.Message,
		)
	})

	// Create an event bus with our handler
	bus := events.NewEventBus()
	bus.Subscribe(eventHandler)

	// Create a new GAuth instance with events
	auth := gauth.New(gauth.Config{
		TokenSecret: "demo-secret-key-do-not-use-in-production",
		EventBus:    bus,
	})

	// Add time-based restriction (only allow during business hours)
	businessHours := gauth.CreateTimeRangeRestriction(
		time.Date(2023, 1, 1, 9, 0, 0, 0, time.Local),  // 9 AM
		time.Date(2023, 1, 1, 17, 0, 0, 0, time.Local), // 5 PM
	)
	auth.AddRestriction(businessHours)

	// Add rate limiting restriction (100 requests per minute)
	rateLimit := gauth.CreateRateLimitRestriction(100, time.Minute)
	auth.AddRestriction(rateLimit)

	// Set up HTTP handlers
	http.HandleFunc("/token/create", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Create a token for user "demo-user"
		token, err := auth.CreateToken("demo-user", nil)
		if err != nil {
			http.Error(w, "Failed to create token: "+err.Error(), http.StatusInternalServerError)
			return
		}

		fmt.Fprintf(w, "Token: %s\nExpires: %s\n", 
			token.Token, 
			token.ExpiresAt.Format(time.RFC3339),
		)
	})

	http.HandleFunc("/token/validate", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		tokenStr := r.FormValue("token")
		if tokenStr == "" {
			http.Error(w, "Token required", http.StatusBadRequest)
			return
		}

		// Validate the token
		valid, err := auth.ValidateToken(tokenStr)
		if err != nil {
			http.Error(w, "Error validating token: "+err.Error(), http.StatusInternalServerError)
			return
		}

		if !valid {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		fmt.Fprintf(w, "Token is valid\n")
	})

	http.HandleFunc("/token/revoke", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		tokenStr := r.FormValue("token")
		if tokenStr == "" {
			http.Error(w, "Token required", http.StatusBadRequest)
			return
		}

		// Revoke the token
		err := auth.RevokeToken(tokenStr)
		if err != nil {
			http.Error(w, "Error revoking token: "+err.Error(), http.StatusInternalServerError)
			return
		}

		fmt.Fprintf(w, "Token revoked successfully\n")
	})

	http.HandleFunc("/protected", func(w http.ResponseWriter, r *http.Request) {
		// Extract token from Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" || len(authHeader) < 8 || authHeader[:7] != "Bearer " {
			http.Error(w, "Authorization header required", http.StatusUnauthorized)
			return
		}
		tokenStr := authHeader[7:]

		// Validate the token
		valid, err := auth.ValidateToken(tokenStr)
		if err != nil {
			http.Error(w, "Error validating token: "+err.Error(), http.StatusInternalServerError)
			return
		}

		if !valid {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		// If token is valid, show protected content
		fmt.Fprintf(w, "Protected resource accessed successfully\n")
	})

	// Start the server
	log.Println("Token management demo server starting on :8080")
	log.Println("Use the following endpoints:")
	log.Println("  POST /token/create - Create a new token")
	log.Println("  POST /token/validate?token=<token> - Validate a token")
	log.Println("  POST /token/revoke?token=<token> - Revoke a token")
	log.Println("  GET /protected (with Authorization: Bearer <token>) - Access protected resource")

	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
package typedstructuresdemo
// Package demo provides examples of GAuth usage with typed structures
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
	// Create a custom event handler for logging with typed metadata
	eventHandler := events.HandlerFunc(func(event events.Event) {
		fmt.Printf("[%s] %s: %s - %s\n", 
			event.Type, 
			event.Action, 
			event.Status, 
			event.Message,
		)
		
		// Access typed metadata if available
		if event.Metadata != nil {
			if ipAddr, ok := event.Metadata.GetString("ip_address"); ok {
				fmt.Printf("  IP: %s\n", ipAddr)
			}
			
			if requestID, ok := event.Metadata.GetString("request_id"); ok {
				fmt.Printf("  Request ID: %s\n", requestID)
			}
			
			if attempts, ok := event.Metadata.GetInt("attempts"); ok {
				fmt.Printf("  Attempts: %d\n", attempts)
			}
		}
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
		
		// Create typed metadata for the token creation event
		metadata := events.NewMetadata()
		metadata.SetString("ip_address", r.RemoteAddr)
		metadata.SetString("request_id", fmt.Sprintf("req-%d", time.Now().UnixNano()))
		metadata.SetString("user_agent", r.UserAgent())
		
		// Create a token for user "demo-user" with metadata
		token, err := auth.CreateTokenWithMetadata("demo-user", nil, metadata)
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
		
		// Create typed metadata for validation attempt
		metadata := events.NewMetadata()
		metadata.SetString("ip_address", r.RemoteAddr)
		metadata.SetInt("attempts", 1) // Could track actual attempts in a real app
		
		// Validate the token with metadata
		valid, err := auth.ValidateTokenWithMetadata(tokenStr, metadata)
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
		
		// Create typed metadata for revocation
		metadata := events.NewMetadata()
		metadata.SetString("ip_address", r.RemoteAddr)
		metadata.SetString("reason", r.FormValue("reason"))
		metadata.SetTime("revocation_time", time.Now())

		// Revoke the token with metadata
		err := auth.RevokeTokenWithMetadata(tokenStr, metadata)
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
		
		// Create typed metadata for access attempt
		metadata := events.NewMetadata()
		metadata.SetString("ip_address", r.RemoteAddr)
		metadata.SetString("resource", r.URL.Path)
		metadata.SetString("method", r.Method)

		// Validate the token with metadata
		valid, err := auth.ValidateTokenWithMetadata(tokenStr, metadata)
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
	log.Println("Typed structures demo server starting on :8080")
	log.Println("Use the following endpoints:")
	log.Println("  POST /token/create - Create a new token with typed metadata")
	log.Println("  POST /token/validate?token=<token> - Validate a token with typed metadata")
	log.Println("  POST /token/revoke?token=<token>&reason=<reason> - Revoke a token with typed metadata")
	log.Println("  GET /protected (with Authorization: Bearer <token>) - Access protected resource")

	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
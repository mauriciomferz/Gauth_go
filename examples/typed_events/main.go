package main

import (
	"context"
	"fmt"
	"time"

	"github.com/Gimel-Foundation/gauth/pkg/events"
)

// Define strongly typed metadata structures
type UserMetadata struct {
	UserID    string `json:"user_id"`
	Username  string `json:"username"`
	Email     string `json:"email"`
	FirstName string `json:"first_name,omitempty"`
	LastName  string `json:"last_name,omitempty"`
	Role      string `json:"role"`
}

type AuthenticationMetadata struct {
	Method        string    `json:"method"` // password, oauth, mfa, etc.
	SourceIP      string    `json:"source_ip"`
	UserAgent     string    `json:"user_agent"`
	Timestamp     time.Time `json:"timestamp"`
	Successful    bool      `json:"successful"`
	FailureReason string    `json:"failure_reason,omitempty"`
}

type TokenMetadata struct {
	TokenID     string    `json:"token_id"`
	Type        string    `json:"type"` // access, refresh, etc.
	Scopes      []string  `json:"scopes"`
	IssuedAt    time.Time `json:"issued_at"`
	ExpiresAt   time.Time `json:"expires_at"`
	ClientID    string    `json:"client_id,omitempty"`
	DeviceID    string    `json:"device_id,omitempty"`
	Fingerprint string    `json:"fingerprint,omitempty"`
}

// Define typed event data structure
type AuthenticationEvent struct {
	User   UserMetadata           `json:"user"`
	Auth   AuthenticationMetadata `json:"auth"`
	Token  TokenMetadata          `json:"token,omitempty"`
	Custom map[string]interface{} `json:"custom,omitempty"` // Only for truly dynamic data
}

func main() {
	ctx := context.Background()

	// Create an event publisher with typed events
	publisher := events.NewPublisher()

	// Subscribe to authentication events
	subscriber := events.NewSubscriber()
	subscriber.Subscribe("authentication.success", handleAuthSuccess)
	subscriber.Subscribe("authentication.failure", handleAuthFailure)

	// Register subscriber with publisher
	publisher.Register(subscriber)

	// Create a successful authentication event with typed metadata
	successEvent := &AuthenticationEvent{
		User: UserMetadata{
			UserID:   "user123",
			Username: "johndoe",
			Email:    "john@example.com",
			Role:     "admin",
		},
		Auth: AuthenticationMetadata{
			Method:     "password",
			SourceIP:   "192.168.1.1",
			UserAgent:  "Mozilla/5.0 (Windows NT 10.0; Win64; x64) Chrome/91.0.4472.124",
			Timestamp:  time.Now(),
			Successful: true,
		},
		Token: TokenMetadata{
			TokenID:   "tk_123456",
			Type:      "access",
			Scopes:    []string{"read", "write", "admin"},
			IssuedAt:  time.Now(),
			ExpiresAt: time.Now().Add(time.Hour),
			ClientID:  "client_web",
		},
	}

	// Create a failed authentication event
	failedEvent := &AuthenticationEvent{
		User: UserMetadata{
			Username: "janedoe",
			Email:    "jane@example.com",
		},
		Auth: AuthenticationMetadata{
			Method:        "password",
			SourceIP:      "10.0.0.1",
			UserAgent:     "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) Safari/605.1.15",
			Timestamp:     time.Now(),
			Successful:    false,
			FailureReason: "invalid_credentials",
		},
	}

	// Publish events
	publisher.Publish(ctx, "authentication.success", successEvent)
	publisher.Publish(ctx, "authentication.failure", failedEvent)

	// Wait for events to be processed
	time.Sleep(100 * time.Millisecond)
}

func handleAuthSuccess(ctx context.Context, eventName string, eventData interface{}) error {
	// Type assertion to get strongly typed event data
	event, ok := eventData.(*AuthenticationEvent)
	if !ok {
		return fmt.Errorf("expected AuthenticationEvent, got %T", eventData)
	}

	// Access typed fields directly
	fmt.Printf("Successful authentication for user %s (ID: %s) using %s method\n",
		event.User.Username, event.User.UserID, event.Auth.Method)
	fmt.Printf("Token %s issued with scopes: %v\n", event.Token.TokenID, event.Token.Scopes)

	return nil
}

func handleAuthFailure(ctx context.Context, eventName string, eventData interface{}) error {
	// Type assertion to get strongly typed event data
	event, ok := eventData.(*AuthenticationEvent)
	if !ok {
		return fmt.Errorf("expected AuthenticationEvent, got %T", eventData)
	}

	// Access typed fields directly
	fmt.Printf("Failed authentication for user %s: %s\n",
		event.User.Username, event.Auth.FailureReason)
	fmt.Printf("Attempt from IP %s at %s\n",
		event.Auth.SourceIP, event.Auth.Timestamp.Format(time.RFC3339))

	return nil
}

package main

import (
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
	// (ctx removed, not needed)

	// Create an event publisher
	publisher := events.NewEventPublisher()

	// Create and subscribe handlers for authentication events
	successHandler := &authSuccessHandler{}
	failureHandler := &authFailureHandler{}
	publisher.Subscribe(successHandler)
	publisher.Subscribe(failureHandler)

	// (successEvent and failedEvent removed, not needed)

	// Publish events (handlers will process all events, filter inside handler)
	publisher.Publish(events.Event{
		Type:     events.EventTypeAuth,
		Action:   "authentication.success",
		Status:   "success",
		Message:  "Authentication succeeded",
		Metadata: events.NewMetadata(), // Optionally encode successEvent as needed
	})
	publisher.Publish(events.Event{
		Type:     events.EventTypeAuth,
		Action:   "authentication.failure",
		Status:   "failure",
		Message:  "Authentication failed",
		Metadata: events.NewMetadata(), // Optionally encode failedEvent as needed
	})

	// Wait for events to be processed
	time.Sleep(100 * time.Millisecond)
}

// Handler for successful authentication events
type authSuccessHandler struct{}

func (h *authSuccessHandler) Handle(event events.Event) {
	if event.Action == "authentication.success" {
		fmt.Println("[SUCCESS]", event.Message)
	}
}

// Handler for failed authentication events
type authFailureHandler struct{}

func (h *authFailureHandler) Handle(event events.Event) {
	if event.Action == "authentication.failure" {
		fmt.Println("[FAILURE]", event.Message)
	}
}

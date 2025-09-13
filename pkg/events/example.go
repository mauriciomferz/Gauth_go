// Package events provides a unified event system for GAuth
// This file demonstrates how to use the event system with strongly typed events

package events

import (
	"fmt"
	"time"
)

// ExampleFluentAPI demonstrates how to use the event system with a fluent builder pattern
func ExampleFluentAPI() {
	// Create a new typed authentication event with fluent builder pattern
	loginEvent := CreateEvent().
		WithType(EventTypeAuth).
		WithActionEnum(ActionLogin).
		WithStatusEnum(StatusSuccess).
		WithSubject("user-123").
		WithMessage("User logged in successfully").
		WithStringMetadata("ip_address", "192.168.1.1").
		WithStringMetadata("user_agent", "Mozilla/5.0...")

		// Create a new typed token event with fluent builder pattern
	tokenEvent := CreateEvent().
		WithType(EventTypeToken).
		WithActionEnum(ActionTokenIssued).
		WithStatusEnum(StatusSuccess).
		WithSubject("user-123").
		WithResource("token-456").
		WithMessage("Access token issued").
		WithStringMetadata("token_type", "access_token").
		WithIntMetadata("expires_in", 3600)

		// Create a delegation event (RFC111 specific) with fluent builder pattern
	delegationEvent := CreateEvent().
		WithType(EventTypeAuthz).
		WithActionEnum(ActionDelegationExercised).
		WithStatusEnum(StatusSuccess).
		WithSubject("delegate-123").
		WithResource("account-456").
		WithMessage("Delegate exercised power of attorney").
		WithStringMetadata("delegator", "principal-789").
		WithStringMetadata("power_type", "financial_transactions").
		WithStringMetadata("transaction_id", "tx-123").
		WithIntMetadata("amount", 500)

	// Use helper functions for common event types
	loginEvent2 := CreateAuthEvent(ActionLogin, StatusSuccess).
		WithSubject("user-456").
		WithMessage("User logged in via OAuth2")

	tokenEvent2 := CreateTokenEvent(ActionTokenIssued, StatusSuccess).
		WithSubject("user-456").
		WithResource("token-789")

	// Create a handler to process events
	handler := &ExampleHandler{}

	// Handle the events
	handler.Handle(loginEvent)
	handler.Handle(tokenEvent)
	handler.Handle(delegationEvent)
	handler.Handle(loginEvent2)
	handler.Handle(tokenEvent2)

	// Publish events to all registered handlers
	Publish(loginEvent)
}

// Example of using the DefaultPublisher
func ExamplePublisher() {
	// Create a sample handler
	handler := &ExampleHandler{}

	// Subscribe the handler to receive events
	Subscribe(handler)

	// Create and publish an event
	loginEvent := CreateAuthEvent(ActionLogin, StatusSuccess).
		WithSubject("user-789").
		WithMessage("User login via publisher")

	// This will be sent to all subscribed handlers
	Publish(loginEvent)
}

// ExampleHandler is a sample event handler
type ExampleHandler struct{}

// Handle processes events
func (h *ExampleHandler) Handle(event Event) {
	// Process based on event type
	switch event.Type {
	case EventTypeAuth:
		h.handleAuthEvent(event)
	case EventTypeAuthz:
		h.handleAuthzEvent(event)
	case EventTypeToken:
		h.handleTokenEvent(event)
	default:
		fmt.Printf("[%s] %s: %s\n",
			event.Type,
			event.Action,
			event.Message)
	}
}

// handleAuthEvent processes authentication events
func (h *ExampleHandler) handleAuthEvent(event Event) {
	switch event.Action {
	case string(ActionLogin):
		fmt.Printf("User %s logged in at %s\n",
			event.Subject,
			event.Timestamp.Format(time.RFC3339))
	case string(ActionLogout):
		fmt.Printf("User %s logged out at %s\n",
			event.Subject,
			event.Timestamp.Format(time.RFC3339))
	default:
		fmt.Printf("Auth event: %s for user %s\n",
			event.Action,
			event.Subject)
	}
}

// handleAuthzEvent processes authorization events
func (h *ExampleHandler) handleAuthzEvent(event Event) {
	switch event.Action {
	case string(ActionDelegationExercised):
		delegator, _ := event.Metadata.GetString("delegator")
		powerType, _ := event.Metadata.GetString("power_type")

		fmt.Printf("Delegation: %s exercised power '%s' on behalf of %s for resource %s\n",
			event.Subject,
			powerType,
			delegator,
			event.Resource)
	default:
		fmt.Printf("Authorization event: %s for subject %s and resource %s\n",
			event.Action,
			event.Subject,
			event.Resource)
	}
}

// handleTokenEvent processes token events
func (h *ExampleHandler) handleTokenEvent(event Event) {
	switch event.Action {
	case string(ActionTokenIssued):
		tokenType, _ := event.Metadata.GetString("token_type")
		expiresIn, _ := event.Metadata.GetInt("expires_in")

		fmt.Printf("Token issued: %s token for user %s, expires in %d seconds\n",
			tokenType,
			event.Subject,
			expiresIn)
	case string(ActionTokenRevoked):
		fmt.Printf("Token revoked: token %s for user %s\n",
			event.Resource,
			event.Subject)
	default:
		fmt.Printf("Token event: %s for token %s\n",
			event.Action,
			event.Resource)
	}
}

// Typed Event System Example
// Demonstrates creation, dispatch, and handling of events with strongly-typed metadata using AgentAuth.
// Shows authentication, audit, and error event patterns with custom handlers.

package main

import (
	"fmt"
	"time"

	"github.com/mauriciomferz/AgentAuth/pkg/events"
)

// CustomEventHandler demonstrates how to implement an event handler for typed events
type CustomEventHandler struct {
	name string
}

// Handle processes an event and prints metadata details
func (h *CustomEventHandler) Handle(event events.Event) {
	fmt.Printf("[%s] Received event: %s/%s - %s\n",
		h.name, event.Type, event.Action, event.Message)

	// Access typed metadata
	if event.Metadata != nil {
		if userID, ok := event.Metadata.GetString("user_id"); ok {
			fmt.Printf("[%s] User ID: %s\n", h.name, userID)
		}

		if attempts, ok := event.Metadata.GetInt("login_attempts"); ok {
			fmt.Printf("[%s] Login attempts: %d\n", h.name, attempts)
		}

		if timestamp, ok := event.Metadata.GetTime("last_login"); ok {
			fmt.Printf("[%s] Last login: %s\n", h.name, timestamp.Format(time.RFC3339))
		}
	}
}

func main() {
	fmt.Println("=== Event System Example ===")

	// Create event handlers for demonstration
	securityHandler := &CustomEventHandler{name: "SecurityMonitor"}
	auditHandler := &CustomEventHandler{name: "AuditLog"}

	// Create a simple event dispatcher
	dispatcher := events.NewSimpleDispatcher()

	// Register handlers for specific event types
	dispatcher.RegisterHandler(events.EventTypeAuth, securityHandler)
	dispatcher.RegisterHandler(events.EventTypeAudit, auditHandler)
	// Subscribe to all events (for monitoring)
	dispatcher.RegisterHandler(events.EventType("*"), &CustomEventHandler{name: "Monitor"})

	// Example 1: Create and dispatch a simple authentication event
	fmt.Println("\n=== Example 1: Simple Authentication Event ===")

	authEvent := events.Event{
		ID:        "evt-001",
		Type:      events.EventTypeAuth,
		Action:    "login",
		Status:    "success",
		Timestamp: time.Now(),
		Subject:   "user123",
		Resource:  "portal",
		Message:   "User successfully authenticated",
	}

	// Dispatch the event to registered handlers
	dispatcher.Dispatch(authEvent)

	// Example 2: Event with typed metadata
	fmt.Println("\n=== Example 2: Event with Typed Metadata ===")

	// Create metadata with different value types
	metadata := events.NewMetadata()
	metadata.SetString("user_id", "user123")
	metadata.SetInt("login_attempts", 1)
	metadata.SetTime("last_login", time.Now().Add(-24*time.Hour))
	metadata.SetBool("is_admin", false)
	metadata.SetFloat("score", 0.95)
	metadata.SetStringSlice("roles", []string{"user", "editor"})

	// Create event with metadata
	authEventWithMeta := events.Event{
		ID:        "evt-002",
		Type:      events.EventTypeAuth,
		Action:    "login",
		Status:    "success",
		Timestamp: time.Now(),
		Subject:   "user123",
		Resource:  "portal",
		Message:   "User authenticated with additional context",
		Metadata:  metadata,
	}

	// Dispatch the event to registered handlers
	dispatcher.Dispatch(authEventWithMeta)

	// Example 3: Audit event
	fmt.Println("\n=== Example 3: Audit Event ===")

	auditMeta := events.NewMetadata()
	auditMeta.SetString("ip_address", "192.168.1.100")
	auditMeta.SetString("user_agent", "Mozilla/5.0")
	auditMeta.SetString("session_id", "sess-abc-123")

	auditEvent := events.Event{
		ID:        "evt-003",
		Type:      events.EventTypeAudit,
		Action:    "resource_access",
		Status:    "success",
		Timestamp: time.Now(),
		Subject:   "user123",
		Resource:  "document/123",
		Message:   "User accessed confidential document",
		Metadata:  auditMeta,
	}

	// Dispatch the event to registered handlers
	dispatcher.Dispatch(auditEvent)

	// Example 4: Failed authentication event
	fmt.Println("\n=== Example 4: Failed Authentication Event ===")

	failureMeta := events.NewMetadata()
	failureMeta.SetString("error_code", "invalid_credentials")
	failureMeta.SetInt("failure_count", 3)

	failedAuthEvent := events.Event{
		ID:        "evt-004",
		Type:      events.EventTypeAuth,
		Action:    "login",
		Status:    "failure",
		Timestamp: time.Now(),
		Subject:   "user456",
		Resource:  "api",
		Message:   "Authentication failed: invalid credentials",
		Metadata:  failureMeta,
		Error:     "invalid username or password",
	}

	// Dispatch the event to registered handlers
	dispatcher.Dispatch(failedAuthEvent)
}

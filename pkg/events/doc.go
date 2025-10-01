/*
Package events provides a typed event system for GAuth with strong type safety.

# Quick Start

	import "github.com/Gimel-Foundation/gauth/pkg/events"

	publisher := events.NewPublisher()
	evt := events.NewEvent(events.EventTypeToken, "token.issued").WithSubject("user123")
	publisher.PublishEvent(evt)
	// ...

See runnable examples in examples/events/basic/main.go.

# RFC111 Mapping

This package implements the event-driven compliance, audit, and transparency requirements of GiFo-RfC 0111 (September 2025):
  - Typed events for all protocol steps (authentication, authorization, token, audit, system)
  - Centralized, verifiable event bus for protocol flow and compliance
  - No support for excluded features (Web3, DNA-based identity, AI-controlled event logic)

# Overview

The events package offers a complete event system with the following features:
 1. Strongly-typed events with structured metadata
 2. Event dispatching and handling
 3. Type-based event filtering and subscription
 4. Thread-safe operation
 5. Typed metadata with strict type checking
 6. Pluggable handlers for flexible processing

# See Also
  - package auth: for authentication integration
  - package authz: for authorization integration
  - package audit: for audit trail integration
  - package token: for token lifecycle events

For advanced usage and integration, see the examples/ directory and the project README.

Event Structure:

Each event contains:

	ID:        Unique identifier for the event
	Type:      The event type (strongly typed enum)
	Action:    The specific action (e.g., "login", "logout")
	Status:    Status of the event (e.g., "success", "failure")
	Timestamp: When the event occurred
	Subject:   The subject of the event (typically a user ID)
	Resource:  The resource being accessed
	Message:   A human-readable message
	Metadata:  Structured additional data
	Error:     Error information if the event represents a failure

Typed Metadata:

Instead of using map[string]interface{}, the package uses a strongly-typed
Metadata structure:

	metadata := events.NewMetadata()
	metadata.SetString("user_id", "user123")
	metadata.SetInt("login_attempts", 3)
	metadata.SetTime("last_login", time.Now())

	// Type-safe retrieval
	if userID, ok := metadata.GetString("user_id"); ok {
		// Use userID with confidence in its type
	}

3. Event Handlers:
  - Logging handlers
  - Metrics collection
  - Alert generation
  - Audit trail creation

Basic Usage:

	// Create an event publisher
	publisher := events.NewPublisher()

Event Handlers:

Implement the EventHandler interface to process events:

	type CustomHandler struct{}

	func (h *CustomHandler) Handle(event events.Event) {
		// Process the event
	}

Event Dispatching:

Use the Dispatcher to send events to registered handlers:

	dispatcher := events.NewSimpleDispatcher()

	// Register handlers for specific event types
	dispatcher.RegisterHandler(events.EventTypeAuth, securityHandler)
	dispatcher.RegisterHandler(events.EventTypeAudit, auditHandler)

	// Register handler for all events
	dispatcher.RegisterHandler("*", monitorHandler)

	// Dispatch an event
	dispatcher.Dispatch(authEvent)

Event Types:

1. Authentication Events:
  - Login attempts
  - Logout events
  - MFA operations
  - Session management

2. Authorization Events:
  - Access attempts
  - Policy decisions
  - Permission changes
  - Role assignments

3. Token Events:
  - Token creation
  - Token validation
  - Token revocation
  - Key rotation

4. Audit Events:
  - Configuration changes
  - Policy updates
  - System access
  - Data access

Event Handling:

Events can be handled in multiple ways:

1. Synchronous Processing:

	handler.HandleEvent(event)

2. Asynchronous Publishing:

	publisher.PublishEvent(event)

3. Filtered Handling:

	handler.HandleEventWithFilter(event, filter)

Error Handling:

The package provides specific error types and handling:

	type EventError struct {
		Event Event
		Op    string
		Err   error
	}

Thread Safety:

All types in this package are designed to be thread-safe
and can be used concurrently.

Monitoring Integration:

Events can be integrated with monitoring systems:

1. Metrics Collection:
  - Event counts by type
  - Success/failure rates
  - Timing information
  - Handler performance

2. Alert Generation:
  - Security violations
  - System issues
  - Performance problems
  - Resource constraints

Storage Integration:

Events can be persisted using various backends:

1. Database Storage:
  - SQL databases
  - Document stores
  - Time series databases

2. Message Queues:
  - Kafka
  - RabbitMQ
  - Redis Streams

Best Practices:

1. Event Design:
  - Use strongly-typed events
  - Include relevant context
  - Follow consistent patterns
  - Consider privacy implications

2. Handler Implementation:
  - Handle errors appropriately
  - Process events asynchronously
  - Implement timeout handling
  - Consider backpressure

3. Monitoring:
  - Track event volumes
  - Monitor handler performance
  - Set up alerting
  - Analyze patterns

4. Storage:
  - Consider retention policies
  - Implement backup strategies
  - Handle storage failures
  - Monitor storage usage

See Also:
- Package auth for authentication integration
- Package authz for authorization integration
- Package audit for audit trail integration
*/
package events

// Package events provides strongly-typed event handling functionality.
//
// # Overview
//
// The events package implements a type-safe event system with:
//   - Enumerated event types
//   - Strongly-typed event details
//   - Clear event categorization
//   - Type-safe event handling
//
// # Key Components
//
// 1. Event Types
//
//	const (
//	    AuthSuccess Type = iota
//	    AuthFailure
//	    TokenIssued
//	    TokenRevoked
//	    RateLimitExceeded
//	    CircuitBreakerOpen
//	)
//
// Provides clear categorization of system events.
//
// 2. Event Details
//
//	type Event struct {
//	    Type      Type
//	    Timestamp time.Time
//	    Subject   string
//	    Details   EventDetails
//	}
//
// Represents events with their metadata and typed details.
//
// # Usage Examples
//
// Creating and handling events:
//
//	event := &events.Event{
//	    Type:      events.AuthSuccess,
//	    Timestamp: time.Now(),
//	    Subject:   "user-123",
//	    Details: &events.AuthEventDetails{
//	        ClientID:  "client-123",
//	        GrantType: "password",
//	        Scopes:    []string{"read"},
//	    },
//	}
//
//	switch event.Type {
//	case events.AuthSuccess:
//	    details := event.Details.(*events.AuthEventDetails)
//	    // Handle authentication success
//	case events.RateLimitExceeded:
//	    details := event.Details.(*events.RateLimitEventDetails)
//	    // Handle rate limit exceeded
//	}
//
// # Best Practices
//
// 1. Event Creation:
//   - Use appropriate event types
//   - Include relevant details
//   - Set accurate timestamps
//
// 2. Event Handling:
//   - Type-assert safely
//   - Handle all event types
//   - Process events promptly
//
// 3. Event Details:
//   - Use strongly-typed details
//   - Include necessary context
//   - Avoid sensitive information
//
// # Extension Points
//
// 1. Custom Events:
//   - Add new event types
//   - Create custom detail types
//   - Implement event handling
//
// 2. Event Processing:
//   - Add event listeners
//   - Implement filtering
//   - Handle event routing
//
// See the examples directory for usage patterns.
package events

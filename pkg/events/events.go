// Package events provides a unified event system for GAuth authentication framework.
// It supports typed event handling for authentication, authorization, token management,
// user activity tracking, and system events.
package events

import (
	"time"

	"github.com/google/uuid"
)

// Define legacy types from the old system
// These are kept for backward compatibility
type legacyEventType int

const (
	// Service Events
	ServiceStarted legacyEventType = iota
	ServiceStopped
	ServiceDegraded
	ServiceRestored

	// Circuit Breaker Events
	CircuitOpened
	CircuitClosed
	CircuitHalfOpen

	// Rate Limiting Events
	RateLimitExceeded
	RateLimitReset

	// Request Events
	RequestStarted
	RequestCompleted
	RequestFailed
	RequestTimeout

	// Resource Events
	ResourceExhausted
	ResourceReleased
)

// EventType represents the type of event in the new system
type EventType string

// Common event types
const (
	EventTypeAuth         EventType = "auth"
	EventTypeAuthz        EventType = "authz"
	EventTypeToken        EventType = "token"
	EventTypeUserActivity EventType = "user_activity"
	EventTypeAudit        EventType = "audit"
	EventTypeSystem       EventType = "system"
	EventTypeResource     EventType = "resource"
	EventTypeCircuit      EventType = "circuit"
	EventTypeRateLimit    EventType = "rate_limit"
)

// Event represents a system event with strongly typed fields
type Event struct {
	ID        string    `json:"id"`
	Type      EventType `json:"type"`
	Action    string    `json:"action"`
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
	Subject   string    `json:"subject,omitempty"`
	Resource  string    `json:"resource,omitempty"`
	Message   string    `json:"message,omitempty"`
	Metadata  *Metadata `json:"metadata,omitempty"`
	Error     string    `json:"error,omitempty"`
}

// NewEvent creates a basic event with required fields
func NewEvent() Event {
	return Event{
		ID:        uuid.New().String(),
		Timestamp: time.Now(),
		Metadata:  NewMetadata(),
	}
}

// EventHandler defines the interface for event handlers
type EventHandler interface {
	Handle(Event)
}

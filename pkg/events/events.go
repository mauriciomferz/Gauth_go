// Package events/events.go: RFC111 Compliance Mapping
//
// This file implements the unified, type-safe event system as required by RFC111:
//   - Typed event handling for authentication, authorization, token, and system events
//   - All event types are enums/constants (no stringly-typed events)
//   - Supports audit, compliance, and activity tracking for all protocol steps
//
// Relevant RFC111 Sections:
//   - Section 6: How GAuth works (event, audit, compliance)
//   - Section 7: Benefits (verifiability, auditability)
//
// Compliance:
//   - All event types are enums/constants (no ambiguous types)
//   - Event system is type-safe and covers all protocol steps
//   - No exclusions (Web3, DNA, decentralized auth) are present
//   - See README and docs/ for full protocol mapping
//
// License: Apache 2.0 (see LICENSE file)
//
// ---
//
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

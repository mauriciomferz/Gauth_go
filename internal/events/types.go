package events

import "time"

// Type represents an event type
type Type int

const (
	// AuthSuccess indicates successful authentication
	AuthSuccess Type = iota
	// AuthFailure indicates failed authentication
	AuthFailure
	// TokenIssued indicates token issuance
	TokenIssued
	// TokenRevoked indicates token revocation
	TokenRevoked
	// RateLimitExceeded indicates rate limit violation
	RateLimitExceeded
	// CircuitBreakerOpen indicates circuit breaker activation
	CircuitBreakerOpen
)

// String returns the string representation of an event type
func (t Type) String() string {
	switch t {
	case AuthSuccess:
		return "AuthSuccess"
	case AuthFailure:
		return "AuthFailure"
	case TokenIssued:
		return "TokenIssued"
	case TokenRevoked:
		return "TokenRevoked"
	case RateLimitExceeded:
		return "RateLimitExceeded"
	case CircuitBreakerOpen:
		return "CircuitBreakerOpen"
	default:
		return "Unknown"
	}
}

// Event represents a system event
type Event struct {
	// Type is the event type
	Type Type

	// Timestamp is when the event occurred
	Timestamp time.Time

	// Subject is the entity the event relates to
	Subject string

	// Details contains event-specific information
	Details EventDetails
}

// EventDetails is a marker interface for event-specific details
type EventDetails interface {
	isEventDetails()
}

// AuthEventDetails contains authentication event details
type AuthEventDetails struct {
	ClientID    string
	GrantType   string
	Scopes      []string
	ErrorCode   string
	ErrorDetail string
}

func (AuthEventDetails) isEventDetails() {}

// TokenEventDetails contains token-related event details
type TokenEventDetails struct {
	TokenType  string
	ExpiresIn  int64
	Scopes     []string
	ReasonCode string
}

func (TokenEventDetails) isEventDetails() {}

// RateLimitEventDetails contains rate limit event details
type RateLimitEventDetails struct {
	ClientID        string
	Limit           int
	CurrentUsage    int
	WindowSize      int
	RemainingWindow int
}

func (RateLimitEventDetails) isEventDetails() {}

// CircuitBreakerEventDetails contains circuit breaker event details
type CircuitBreakerEventDetails struct {
	Service         string
	ErrorRate       float64
	FailureCount    int
	TimeoutDuration time.Duration
}

func (CircuitBreakerEventDetails) isEventDetails() {}

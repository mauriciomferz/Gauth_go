package events

import (
	"time"
)

// EventType represents different types of events
type EventType string

const (
	CircuitOpened     EventType = "circuit_opened"
	CircuitClosed     EventType = "circuit_closed"
	RateLimitExceeded EventType = "rate_limit_exceeded"
	RequestCompleted  EventType = "request_completed"
	RequestFailed     EventType = "request_failed"
)

// Event represents a system event
type Event struct {
	ID        string                 `json:"id"`
	Type      EventType              `json:"type"`
	ServiceID string                 `json:"service_id"`
	Source    string                 `json:"source"`
	Data      map[string]interface{} `json:"data"`
	Timestamp time.Time              `json:"timestamp"`
	Duration  time.Duration          `json:"duration,omitempty"`
	Error     string                 `json:"error,omitempty"`
}

// EventHandler interface for handling events
type EventHandler interface {
	Handle(event Event)
}

// Events package for cascade systems
func EventsPackageInit() string {
	return "events package initialized"
}

package events

import "time"

// EventType represents different types of service events
type EventType int

const (
	// ServiceStarted indicates a service has started
	ServiceStarted EventType = iota
	// ServiceStopped indicates a service has stopped
	ServiceStopped
	// RequestReceived indicates a new request was received
	RequestReceived
	// RequestCompleted indicates a request was completed successfully
	RequestCompleted
	// RequestFailed indicates a request failed
	RequestFailed
	// CircuitOpened indicates a circuit breaker was opened
	CircuitOpened
	// CircuitClosed indicates a circuit breaker was closed
	CircuitClosed
	// CircuitHalfOpen indicates a circuit breaker entered half-open state
	CircuitHalfOpen
	// RateLimitExceeded indicates a rate limit was exceeded
	RateLimitExceeded
	// BulkheadRejected indicates a request was rejected by the bulkhead
	BulkheadRejected
)

// Event represents a service event with its context
type Event struct {
	Type      EventType
	ServiceID string
	Timestamp time.Time
	Duration  time.Duration
	Error     error
	Details   map[string]interface{} // Only for truly dynamic metadata
}

// String returns a human-readable representation of the event type
func (et EventType) String() string {
	switch et {
	case ServiceStarted:
		return "ServiceStarted"
	case ServiceStopped:
		return "ServiceStopped"
	case RequestReceived:
		return "RequestReceived"
	case RequestCompleted:
		return "RequestCompleted"
	case RequestFailed:
		return "RequestFailed"
	case CircuitOpened:
		return "CircuitOpened"
	case CircuitClosed:
		return "CircuitClosed"
	case CircuitHalfOpen:
		return "CircuitHalfOpen"
	case RateLimitExceeded:
		return "RateLimitExceeded"
	case BulkheadRejected:
		return "BulkheadRejected"
	default:
		return "UnknownEvent"
	}
}

// EventHandler defines the interface for handling events
type EventHandler interface {
	Handle(Event)
}

// EventPublisher provides an interface for publishing events
type EventPublisher interface {
	PublishEvent(Event)
	Subscribe(EventHandler)
	Unsubscribe(EventHandler)
}

// SimpleEventBus provides a basic implementation of EventPublisher
type SimpleEventBus struct {
	handlers []EventHandler
}

// NewSimpleEventBus creates a new SimpleEventBus
func NewSimpleEventBus() *SimpleEventBus {
	return &SimpleEventBus{
		handlers: make([]EventHandler, 0),
	}
}

// PublishEvent publishes an event to all subscribers
func (b *SimpleEventBus) PublishEvent(e Event) {
	for _, handler := range b.handlers {
		handler.Handle(e)
	}
}

// Subscribe adds a new event handler
func (b *SimpleEventBus) Subscribe(handler EventHandler) {
	b.handlers = append(b.handlers, handler)
}

// Unsubscribe removes an event handler
func (b *SimpleEventBus) Unsubscribe(handler EventHandler) {
	for i, h := range b.handlers {
		if &h == &handler {
			b.handlers = append(b.handlers[:i], b.handlers[i+1:]...)
			return
		}
	}
}

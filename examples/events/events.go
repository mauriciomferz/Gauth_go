package events

import "time"

type EventType int

const (
	ServiceStarted EventType = iota
	ServiceStopped
	RequestReceived
	RequestCompleted
	RequestFailed
	CircuitOpened
	CircuitClosed
	CircuitHalfOpen
	RateLimitExceeded
	BulkheadRejected
)

type Event struct {
	Type      EventType
	ServiceID string
	Timestamp time.Time
	Duration  time.Duration
	Error     error
	Details   map[string]interface{}
}

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

type EventHandler func(Event)

type EventPublisher interface {
	PublishEvent(Event)
	Subscribe(EventHandler)
	Unsubscribe(EventHandler)
}

type SimpleEventBus struct {
	handlers []EventHandler
}

func NewEventBus() *SimpleEventBus {
	return &SimpleEventBus{
		handlers: make([]EventHandler, 0),
	}
}

func (b *SimpleEventBus) PublishEvent(e Event) {
	for _, handler := range b.handlers {
		handler(e)
	}
}

func (b *SimpleEventBus) Subscribe(handler EventHandler) {
	b.handlers = append(b.handlers, handler)
}

func (b *SimpleEventBus) Unsubscribe(handler EventHandler) {
	for i, h := range b.handlers {
		if &h == &handler {
			b.handlers = append(b.handlers[:i], b.handlers[i+1:]...)
			return
		}
	}
}

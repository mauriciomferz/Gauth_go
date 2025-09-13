package events

import "sync"

// EventBus manages event publishing and subscriptions
type EventBus struct {
	handlers []EventHandler
	mu       sync.RWMutex
}

// NewEventBus creates a new event bus
func NewEventBus() *EventBus {
	return &EventBus{
		handlers: make([]EventHandler, 0),
	}
}

// Subscribe adds a new event handler
func (bus *EventBus) Subscribe(handler EventHandler) {
	bus.mu.Lock()
	defer bus.mu.Unlock()
	bus.handlers = append(bus.handlers, handler)
}

// Publish sends an event to all subscribers
func (bus *EventBus) Publish(event Event) {
	bus.mu.RLock()
	handlers := make([]EventHandler, len(bus.handlers))
	copy(handlers, bus.handlers)
	bus.mu.RUnlock()

	for _, handler := range handlers {
		handler.Handle(event)
	}
}

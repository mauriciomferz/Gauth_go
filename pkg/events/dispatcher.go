// Package events provides a typed event system
package events

import (
	"sync"
)

// Dispatcher manages event distribution
type Dispatcher interface {
	// Dispatch sends an event to all registered handlers
	Dispatch(event Event)

	// RegisterHandler registers a handler for specific event types
	// Use "*" as type to receive all events
	RegisterHandler(eventType EventType, handler EventHandler)

	// UnregisterHandler removes a handler
	UnregisterHandler(eventType EventType, handler EventHandler)
}

// SimpleDispatcher is a basic implementation of the Dispatcher interface
type SimpleDispatcher struct {
	handlers map[string][]EventHandler
	mu       sync.RWMutex
}

// NewSimpleDispatcher creates a new event dispatcher
func NewSimpleDispatcher() *SimpleDispatcher {
	return &SimpleDispatcher{
		handlers: make(map[string][]EventHandler),
	}
}

// Dispatch sends an event to all registered handlers
func (d *SimpleDispatcher) Dispatch(event Event) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	// Find specific handlers for this event type
	if handlers, ok := d.handlers[string(event.Type)]; ok {
		for _, handler := range handlers {
			handler.Handle(event)
		}
	}

	// Also send to wildcard handlers that receive all events
	if wildcardHandlers, ok := d.handlers["*"]; ok {
		for _, handler := range wildcardHandlers {
			handler.Handle(event)
		}
	}
}

// RegisterHandler registers a handler for a specific event type
func (d *SimpleDispatcher) RegisterHandler(eventType EventType, handler EventHandler) {
	d.mu.Lock()
	defer d.mu.Unlock()

	typeKey := string(eventType)

	// Initialize the slice if it doesn't exist
	if _, ok := d.handlers[typeKey]; !ok {
		d.handlers[typeKey] = []EventHandler{}
	}

	// Add the handler
	d.handlers[typeKey] = append(d.handlers[typeKey], handler)
}

// UnregisterHandler removes a handler
func (d *SimpleDispatcher) UnregisterHandler(eventType EventType, handlerToRemove EventHandler) {
	d.mu.Lock()
	defer d.mu.Unlock()

	typeKey := string(eventType)

	if handlers, ok := d.handlers[typeKey]; ok {
		// Create a new slice without the handler to remove
		newHandlers := make([]EventHandler, 0, len(handlers))
		for _, h := range handlers {
			// Only keep handlers that are not the one we're removing
			// Note: This is a simple equality check that works for function pointers
			// but may not work for all handler implementations
			if h != handlerToRemove {
				newHandlers = append(newHandlers, h)
			}
		}

		if len(newHandlers) > 0 {
			d.handlers[typeKey] = newHandlers
		} else {
			delete(d.handlers, typeKey)
		}
	}
}

// Package events provides a unified event system for GAuth
// This file implements the publisher functionality for events

package events

// EventPublisher manages event subscriptions and publishing
type EventPublisher struct {
	handlers []EventHandler
}

// NewEventPublisher creates a new event publisher
func NewEventPublisher() *EventPublisher {
	return &EventPublisher{
		handlers: make([]EventHandler, 0),
	}
}

// Subscribe adds a new event handler
func (p *EventPublisher) Subscribe(handler EventHandler) {
	p.handlers = append(p.handlers, handler)
}

// Publish sends an event to all subscribed handlers
func (p *EventPublisher) Publish(event Event) {
	for _, handler := range p.handlers {
		handler.Handle(event)
	}
}

// DefaultPublisher is a singleton event publisher
var DefaultPublisher = NewEventPublisher()

// Subscribe adds a handler to the default publisher
func Subscribe(handler EventHandler) {
	DefaultPublisher.Subscribe(handler)
}

// Publish sends an event to the default publisher
func Publish(event Event) {
	DefaultPublisher.Publish(event)
}

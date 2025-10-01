package handlers

import (
	"github.com/Gimel-Foundation/gauth/pkg/events"
)

// MultiHandler combines multiple handlers
type MultiHandler struct {
	handlers []events.EventHandler
}

// NewMultiHandler creates a new multi-handler
func NewMultiHandler() *MultiHandler {
	return &MultiHandler{
		handlers: make([]events.EventHandler, 0),
	}
}

// AddHandler adds a handler
func (h *MultiHandler) AddHandler(handler events.EventHandler) {
	h.handlers = append(h.handlers, handler)
}

// Handle implements the EventHandler interface
func (h *MultiHandler) Handle(event events.Event) {
	// Forward the event to all handlers
	for _, handler := range h.handlers {
		handler.Handle(event)
	}
}

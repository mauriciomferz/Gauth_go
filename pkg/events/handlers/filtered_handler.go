package handlers

import (
	"github.com/Gimel-Foundation/gauth/pkg/events"
)

// FilteredHandler filters events before passing them to the underlying handler
type FilteredHandler struct {
	handler events.EventHandler
	filter  func(events.Event) bool
}

// NewFilteredHandler creates a new filtered handler
func NewFilteredHandler(handler events.EventHandler, filter func(events.Event) bool) *FilteredHandler {
	return &FilteredHandler{
		handler: handler,
		filter:  filter,
	}
}

// Handle implements the EventHandler interface
func (h *FilteredHandler) Handle(event events.Event) {
	// Only handle the event if it passes the filter
	if h.filter(event) {
		h.handler.Handle(event)
	}
}

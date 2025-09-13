package handlers

import (
	"github.com/Gimel-Foundation/gauth/pkg/events"
)

// AuditHandler stores events for audit purposes
type AuditHandler struct {
	// Maximum number of events to store
	MaxEvents int

	// Events storage
	events []events.Event

	// Current index
	currentIndex int

	// Whether the buffer has wrapped around
	wrapped bool

	// Optional persistence function
	persistFunc func([]events.Event)
}

// NewAuditHandler creates a new audit handler
func NewAuditHandler(maxEvents int) *AuditHandler {
	return &AuditHandler{
		MaxEvents:    maxEvents,
		events:       make([]events.Event, maxEvents),
		currentIndex: 0,
		wrapped:      false,
	}
}

// SetPersistFunc sets the persistence function
func (h *AuditHandler) SetPersistFunc(persistFunc func([]events.Event)) {
	h.persistFunc = persistFunc
}

// Handle implements the EventHandler interface
func (h *AuditHandler) Handle(event events.Event) {
	// Store the event in the circular buffer
	h.events[h.currentIndex] = event

	// Increment the index
	h.currentIndex = (h.currentIndex + 1) % h.MaxEvents

	// Check if we've wrapped around
	if h.currentIndex == 0 {
		h.wrapped = true
	}

	// Persist events if a persistence function is set
	if h.persistFunc != nil {
		h.persistFunc(h.GetEvents())
	}
}

// GetEvents returns all stored events
func (h *AuditHandler) GetEvents() []events.Event {
	if !h.wrapped {
		// We haven't wrapped around yet, just return the used portion
		return h.events[:h.currentIndex]
	}

	// We've wrapped around, stitch together the two parts
	result := make([]events.Event, h.MaxEvents)
	copy(result, h.events[h.currentIndex:])
	copy(result[h.MaxEvents-h.currentIndex:], h.events[:h.currentIndex])
	return result
}

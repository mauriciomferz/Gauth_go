package handlers

import (
	"time"

	"github.com/Gimel-Foundation/gauth/pkg/events"
)

// MetricsHandler collects metrics from events
type MetricsHandler struct {
	// Counters for different event types
	AuthCounter         int64
	AuthzCounter        int64
	TokenCounter        int64
	UserActivityCounter int64

	// Counters by status
	SuccessCounter int64
	FailureCounter int64

	// Last event timestamp by type
	LastAuthEvent  time.Time
	LastAuthzEvent time.Time
	LastTokenEvent time.Time

	// Custom metrics collectors
	collectors []events.MetricsCollector
}

// NewMetricsHandler creates a new metrics handler
func NewMetricsHandler() *MetricsHandler {
	return &MetricsHandler{
		collectors: make([]events.MetricsCollector, 0),
	}
}

// AddCollector adds a metrics collector
func (h *MetricsHandler) AddCollector(collector events.MetricsCollector) {
	h.collectors = append(h.collectors, collector)
}

// Handle implements the EventHandler interface
func (h *MetricsHandler) Handle(event events.Event) {
	// Update counters by type
	switch event.Type {
	case events.EventTypeAuth:
		h.AuthCounter++
		h.LastAuthEvent = event.Timestamp

		// Notify collectors
		for _, c := range h.collectors {
			c.RecordAuthEvent(event)
		}

	case events.EventTypeAuthz:
		h.AuthzCounter++
		h.LastAuthzEvent = event.Timestamp

		// Notify collectors
		for _, c := range h.collectors {
			c.RecordAuthzEvent(event)
		}

	case events.EventTypeToken:
		h.TokenCounter++
		h.LastTokenEvent = event.Timestamp

		// Notify collectors
		for _, c := range h.collectors {
			c.RecordTokenEvent(event)
		}

	case events.EventTypeUserActivity:
		h.UserActivityCounter++
	}

	// Update counters by status
	switch event.Status {
	case string(events.StatusSuccess):
		h.SuccessCounter++
	case string(events.StatusFailure), string(events.StatusError):
		h.FailureCounter++
	}
}

// GetAuthCounter returns the number of auth events
func (h *MetricsHandler) GetAuthCounter() int64 {
	return h.AuthCounter
}

// GetAuthzCounter returns the number of authz events
func (h *MetricsHandler) GetAuthzCounter() int64 {
	return h.AuthzCounter
}

// GetTokenCounter returns the number of token events
func (h *MetricsHandler) GetTokenCounter() int64 {
	return h.TokenCounter
}

// GetSuccessCounter returns the number of successful events
func (h *MetricsHandler) GetSuccessCounter() int64 {
	return h.SuccessCounter
}

// GetFailureCounter returns the number of failed events
func (h *MetricsHandler) GetFailureCounter() int64 {
	return h.FailureCounter
}

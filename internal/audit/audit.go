// Package audit provides audit logging functionality for GAuth
package audit

import (
	"sync"
	"time"
)

// Logger provides thread-safe audit logging capabilities
type Logger struct {
	mu      sync.RWMutex
	events  []Event
	maxSize int
}

// NewLogger creates a new audit logger with specified max event history
func NewLogger(maxSize int) *Logger {
	if maxSize <= 0 {
		maxSize = 1000 // Default size if invalid
	}
	return &Logger{
		events:  make([]Event, 0, maxSize),
		maxSize: maxSize,
	}
}

// Log records a new audit event
func (l *Logger) Log(event Event) {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Ensure timestamp is set
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	// Add event to the front for most recent first
	l.events = append([]Event{event}, l.events...)

	// Trim if exceeding max size
	if len(l.events) > l.maxSize {
		l.events = l.events[:l.maxSize]
	}
}

// GetRecent returns the n most recent events
func (l *Logger) GetRecent(n int) []Event {
	l.mu.RLock()
	defer l.mu.RUnlock()

	if n <= 0 || n > len(l.events) {
		n = len(l.events)
	}

	result := make([]Event, n)
	copy(result, l.events[:n])
	return result
}

// Query returns events matching the given criteria
func (l *Logger) Query(filter func(Event) bool) []Event {
	l.mu.RLock()
	defer l.mu.RUnlock()

	var results []Event
	for _, event := range l.events {
		if filter(event) {
			results = append(results, event)
		}
	}
	return results
}

// Clear removes all events from the logger
func (l *Logger) Clear() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.events = make([]Event, 0, l.maxSize)
}

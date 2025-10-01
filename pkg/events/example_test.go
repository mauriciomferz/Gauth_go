// Package events provides a unified event system for GAuth.
package events

import (
	"fmt"
	"time"
)

// Example demonstrates how to use the event system
func Example() {
	// Create an event
	event := Event{
		ID:        "evt-123",
		Type:      "auth.login",
		Timestamp: time.Now(),
		Subject:   "user-123",
		Message:   "User login successful",
	}

	// Print the event details
	fmt.Printf("Event: %s for %s\n", event.Type, event.Subject)
	// Output: Event: auth.login for user-123
}

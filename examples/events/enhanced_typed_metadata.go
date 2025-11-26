package main

import (
	"fmt"
	"time"

	"github.com/mauriciomferz/Gauth_go/pkg/events"
)

// This example demonstrates using the enhanced typed metadata system
// to create, manage and access strongly typed event data.

// UserInfo is a structure containing user information
type UserInfo struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	IsActive  bool      `json:"is_active"`
	Score     float64   `json:"score"`
}

// func main() {
//   // Example main for enhanced typed metadata events
//   // This is commented out to avoid duplicate main redeclaration errors.
// }
//   metadata.SetFloat("confidence_score", 0.95)
//   metadata.SetTime("login_time", time.Now())
//
//   // Add read-only metadata that cannot be modified
//   metadata.SetReadOnly("account_id", events.NewStringValue("acc_67890"))
//
//   // Build and dispatch the event
//   event := eventBuilder.WithMetadata(metadata).Build()
//   bus.Dispatch(event)
//
//   // --- Advanced usage with structured metadata ---
//
//   // Create user info
//   user := UserInfo{
//       ID:        "usr_12345",
//       Name:      "John Doe",
//       CreatedAt: time.Now().Add(-24 * time.Hour),
//       IsActive:  true,
//       Score:     0.85,
//   }
//
//   // Create a more complex event
//   advancedEvent := events.NewEventBuilder().
//       WithType("user").
//       WithAction("profile_update").
//       WithMessage("User profile updated").
//       WithMetadata(userInfoToMetadata(user)).
//       Build()
//
//   bus.Dispatch(advancedEvent)
//
//   // Demonstrate retrieving a complex structure
//   retrieveUserInfo(advancedEvent.Metadata)
// }

// MetadataLoggingHandler demonstrates how to work with typed metadata
type MetadataLoggingHandler struct {
	name string
}

// Handle processes events and logs metadata
func (h *MetadataLoggingHandler) Handle(event events.Event) {
	fmt.Printf("\n[%s] Event: %s/%s - %s\n",
		h.name, event.Type, event.Action, event.Message)
	fmt.Printf("Metadata: (%d fields)\n", event.Metadata.Size())

	// Iterate through all metadata keys
	for _, key := range event.Metadata.Keys() {
		// Get the raw value
		value, exists := event.Metadata.Get(key)
		if !exists {
			continue
		}

		// Print the key and value based on its type
		fmt.Printf("  - %s: ", key)
		switch v := value.Value.(type) {
		case string:
			fmt.Printf("String(%s)\n", v)
		case int:
			fmt.Printf("Int(%d)\n", v)
		case float64:
			fmt.Printf("Float(%.2f)\n", v)
		case bool:
			fmt.Printf("Bool(%t)\n", v)
		case time.Time:
			fmt.Printf("Time(%s)\n", v.Format(time.RFC3339))
		default:
			fmt.Printf("Unknown(%v)\n", v)
		}
	}
}

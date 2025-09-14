package main

import (
	"fmt"
	"time"

	"github.com/Gimel-Foundation/gauth/pkg/events"
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

		switch value.Type {
		case "string":
			val := value.ToString()
			fmt.Printf("String(%s)\n", val)
		case "int":
			val, _ := value.ToInt()
			fmt.Printf("Int(%d)\n", val)
		case "float":
			val, _ := value.ToFloat()
			fmt.Printf("Float(%.2f)\n", val)
		case "bool":
			val, _ := value.ToBool()
			fmt.Printf("Bool(%t)\n", val)
		case "time":
			val, _ := value.ToTime()
			fmt.Printf("Time(%s)\n", val.Format(time.RFC3339))
		default:
			fmt.Printf("Unknown(%v)\n", value.Value)
		}

		// Note if the value is read-only
		if value.ReadOnly {
			fmt.Printf("    (read-only)\n")
		}
	}
}

// userInfoToMetadata converts a UserInfo struct to typed event metadata
func userInfoToMetadata(user UserInfo) *events.Metadata {
	metadata := events.NewMetadata()
	metadata.SetString("user.id", user.ID)
	metadata.SetString("user.name", user.Name)
	metadata.SetTime("user.created_at", user.CreatedAt)
	metadata.SetBool("user.is_active", user.IsActive)
	metadata.SetFloat("user.score", user.Score)
	return metadata
}

// retrieveUserInfo demonstrates how to reconstruct a structured object
// from typed metadata
func retrieveUserInfo(metadata *events.Metadata) {
	// Only proceed if we have metadata
	if metadata == nil {
		fmt.Println("No metadata available")
		return
	}

	// Check if this appears to be user metadata
	if !metadata.Has("user.id") {
		fmt.Println("Not user metadata")
		return
	}

	// Extract all fields with proper type conversions
	id, _ := metadata.GetString("user.id")
	name, _ := metadata.GetString("user.name")
	createdAt, _ := metadata.GetTime("user.created_at")
	isActive, _ := metadata.GetBool("user.is_active")
	score, _ := metadata.GetFloat("user.score")

	// Reconstruct the user object
	user := UserInfo{
		ID:        id,
		Name:      name,
		CreatedAt: createdAt,
		IsActive:  isActive,
		Score:     score,
	}

	// Print the reconstructed user
	fmt.Println("\nReconstructed User Info:")
	fmt.Printf("  ID:        %s\n", user.ID)
	fmt.Printf("  Name:      %s\n", user.Name)
	fmt.Printf("  Created:   %s\n", user.CreatedAt.Format(time.RFC3339))
	fmt.Printf("  Is Active: %t\n", user.IsActive)
	fmt.Printf("  Score:     %.2f\n", user.Score)
}

package events

import (
	"fmt"
	"time"
)

func ExampleTypedMetadataAccessor() {
	// Create a new event with type-safe metadata
	event := NewEvent().
		WithType(EventTypeAuth).
		WithActionEnum(ActionLogin).
		WithStatusEnum(StatusSuccess).
		WithSubject("user123").
		WithResource("api/v1/data").
		WithStringMetadata("client_ip", "192.168.1.1").
		WithIntMetadata("attempt", 1).
		WithBoolMetadata("remember_me", true)

	// Add custom typed metadata
	customValue := NewStringValue("oauth2")
	event = event.WithTypedMetadata("auth_method", customValue)

	// Access typed metadata in a type-safe way
	if event.Metadata != nil {
		if ip, exists := event.Metadata.GetString("client_ip"); exists {
			fmt.Printf("Connection from IP: %s\n", ip)
		}

		if attempt, exists := event.Metadata.GetInt("attempt"); exists {
			fmt.Printf("Login attempt #%d\n", attempt)
		}

		if rememberMe, exists := event.Metadata.GetBool("remember_me"); exists {
			if rememberMe {
				fmt.Println("Extended session requested")
			}
		}
	}

	// Create event with different metadata types
	timestamp := time.Now().Add(-time.Hour) // 1 hour ago

	loginEvent := NewEvent().
		WithType(EventTypeAuth).
		WithActionEnum(ActionLogin).
		WithStatusEnum(StatusSuccess).
		WithSubject("admin").
		WithStringMetadata("client_ip", "10.0.0.1").
		WithStringMetadata("auth_method", "mfa").
		WithTimeMetadata("last_login", timestamp).
		WithIntMetadata("session_duration", 3600)

	// Extract information from the event
	var sessionDuration time.Duration
	if duration, exists := loginEvent.Metadata.GetInt("session_duration"); exists {
		sessionDuration = time.Duration(duration) * time.Second
		fmt.Printf("Session duration: %s\n", sessionDuration)
	}

	if timeVal, exists := loginEvent.Metadata.Get("last_login"); exists {
		if loginTime, err := timeVal.ToTime(); err == nil {
			fmt.Printf("Last login: %s\n", loginTime.Format(time.RFC3339))
		}
	}
}

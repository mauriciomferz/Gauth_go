package events

import (
	"fmt"
	"time"

	"github.com/mauriciomferz/Gauth_go/pkg/events"
)

// CustomEventHandler demonstrates how to implement an event handler
// CustomEventHandler demonstrates how to implement an event handler
type CustomEventHandler struct {
	name string
}

// Handle processes an event
func (h *CustomEventHandler) Handle(event events.Event) {
	fmt.Printf("[%s] Received event: %s/%s - %s\n",
		h.name, event.Type, event.Action, event.Message)

	// Access typed metadata
	if event.Metadata != nil {
		if userID, ok := event.Metadata.GetString("user_id"); ok {
			fmt.Printf("[%s] User ID: %s\n", h.name, userID)
		}

		if attempts, ok := event.Metadata.GetInt("login_attempts"); ok {
			fmt.Printf("[%s] Login attempts: %d\n", h.name, attempts)
		}

		if timestamp, err := event.Metadata.GetTime("last_login"); err == nil {
			fmt.Printf("[%s] Last login: %s\n", h.name, timestamp.Format(time.RFC3339))
		}
	}
}

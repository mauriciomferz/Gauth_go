package main

import (
	"fmt"
	"time"

	"github.com/Gimel-Foundation/gauth/pkg/events"
)

// UserProfile demonstrates a structured data type that can be
// stored and retrieved from event metadata
type UserProfile struct {
	ID        string    `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
	LastLogin time.Time `json:"last_login"`
	IsActive  bool      `json:"is_active"`
	Roles     []string  `json:"roles"`
	Score     float64   `json:"score"`
}

// StoreInMetadata stores the UserProfile in event metadata
// using typed fields with proper prefixing
func (u UserProfile) StoreInMetadata(metadata *events.Metadata) {
	if metadata == nil {
		return
	}

	// Store all fields with a common prefix for namespacing
	prefix := "user."
	metadata.SetString(prefix+"id", u.ID)
	metadata.SetString(prefix+"username", u.Username)
	metadata.SetString(prefix+"email", u.Email)
	metadata.SetTime(prefix+"created_at", u.CreatedAt)
	metadata.SetTime(prefix+"last_login", u.LastLogin)
	metadata.SetBool(prefix+"is_active", u.IsActive)
	metadata.SetFloat(prefix+"score", u.Score)

	// Store array as individual elements
	for i, role := range u.Roles {
		metadata.SetString(fmt.Sprintf("%sroles.%d", prefix, i), role)
	}
	metadata.SetInt(prefix+"roles.count", len(u.Roles))
}

// LoadFromMetadata reconstructs a UserProfile from event metadata
// Returns error if required fields are missing
func LoadUserProfileFromMetadata(metadata *events.Metadata) (UserProfile, error) {
	if metadata == nil {
		return UserProfile{}, fmt.Errorf("metadata is nil")
	}

	// Use prefix for all fields
	prefix := "user."
	var user UserProfile
	var err error

	// Required fields
	user.ID, err = metadata.GetString(prefix + "id")
	if err != nil {
		return UserProfile{}, fmt.Errorf("missing required user.id field: %v", err)
	}

	user.Username, err = metadata.GetString(prefix + "username")
	if err != nil {
		return UserProfile{}, fmt.Errorf("missing required user.username field: %v", err)
	}

	// Optional fields - use zero values if not present
	user.Email, _ = metadata.GetString(prefix + "email")
	user.CreatedAt, _ = metadata.GetTime(prefix + "created_at")
	user.LastLogin, _ = metadata.GetTime(prefix + "last_login")
	user.IsActive, _ = metadata.GetBool(prefix + "is_active")
	user.Score, _ = metadata.GetFloat(prefix + "score")

	// Load array of roles
	roleCount, err := metadata.GetInt(prefix + "roles.count")
	if err == nil && roleCount > 0 {
		user.Roles = make([]string, 0, roleCount)
		for i := 0; i < roleCount; i++ {
			if role, err := metadata.GetString(fmt.Sprintf("%sroles.%d", prefix, i)); err == nil {
				user.Roles = append(user.Roles, role)
			}
		}
	}

	return user, nil
}

// AuditHandler demonstrates a handler that processes events with typed metadata
type AuditHandler struct{}

// Handle processes an event
func (h *AuditHandler) Handle(event events.Event) {
	fmt.Printf("\n--- Audit Log Entry ---\n")
	fmt.Printf("Event: %s/%s at %s\n", event.Type, event.Action, event.Timestamp.Format(time.RFC3339))

	// Check for user profile data in the event
	if event.Metadata != nil && event.Metadata.Has("user.id") {
		user, err := LoadUserProfileFromMetadata(event.Metadata)
		if err == nil {
			fmt.Printf("User: %s (%s)\n", user.Username, user.ID)
			if len(user.Roles) > 0 {
				fmt.Printf("Roles: %v\n", user.Roles)
			}
			if !user.LastLogin.IsZero() {
				fmt.Printf("Previous login: %s\n", user.LastLogin.Format(time.RFC3339))
			}
		}
	}

	// Extract other relevant metadata
	if event.Metadata != nil {
		if ip, err := event.Metadata.GetString("connection.ip"); err == nil {
			fmt.Printf("IP Address: %s\n", ip)
		}

		if device, err := event.Metadata.GetString("connection.device"); err == nil {
			fmt.Printf("Device: %s\n", device)
		}

		if success, err := event.Metadata.GetBool("auth.successful"); err == nil {
			fmt.Printf("Auth Success: %t\n", success)

			if !success {
				if reason, err := event.Metadata.GetString("auth.failure_reason"); err == nil {
					fmt.Printf("Failure Reason: %s\n", reason)
				}
			}
		}
	}

	fmt.Printf("------------------------\n")
}

func main() {
	// Create event bus and register handlers
	bus := events.NewEventBus()
	bus.RegisterHandler(&AuditHandler{})

	// Create a user profile
	user := UserProfile{
		ID:        "usr_123456",
		Username:  "alice_smith",
		Email:     "alice@example.com",
		CreatedAt: time.Now().Add(-30 * 24 * time.Hour), // 30 days ago
		LastLogin: time.Now().Add(-2 * 24 * time.Hour),  // 2 days ago
		IsActive:  true,
		Roles:     []string{"user", "admin", "developer"},
		Score:     92.5,
	}

	// Create successful login event with rich metadata
	loginEvent := events.NewEvent().
		WithType("authentication").
		WithAction("login").
		WithMessage("User login successful").
		WithTimestamp(time.Now())

	// Create metadata and add user profile
	metadata := events.NewMetadata()
	user.StoreInMetadata(metadata)

	// Add additional connection metadata
	metadata.SetString("connection.ip", "192.168.1.100")
	metadata.SetString("connection.device", "iPhone 13 Pro")
	metadata.SetString("connection.client", "Mobile App v2.1.4")
	metadata.SetBool("auth.successful", true)
	metadata.SetInt("auth.attempt", 1)

	// Attach metadata to event and dispatch
	loginEvent.Metadata = metadata
	bus.Dispatch(loginEvent)

	// Now create a failed login event
	failedEvent := events.NewEvent().
		WithType("authentication").
		WithAction("login").
		WithMessage("User login failed").
		WithTimestamp(time.Now())

	// Create metadata for failed login
	failedMeta := events.NewMetadata()
	failedMeta.SetString("user.id", "usr_123456")
	failedMeta.SetString("user.username", "alice_smith")
	failedMeta.SetString("connection.ip", "203.0.113.42") // Different IP
	failedMeta.SetString("connection.device", "Unknown")
	failedMeta.SetBool("auth.successful", false)
	failedMeta.SetString("auth.failure_reason", "Invalid password")
	failedMeta.SetInt("auth.attempt", 3)

	// Dispatch the failed login event
	failedEvent.Metadata = failedMeta
	bus.Dispatch(failedEvent)

	// Demonstrate retrieving a user profile from metadata
	retrievedUser, err := LoadUserProfileFromMetadata(metadata)
	if err != nil {
		fmt.Printf("Error retrieving user profile: %v\n", err)
	} else {
		fmt.Printf("\nRetrieved User Profile:\n")
		fmt.Printf("ID:        %s\n", retrievedUser.ID)
		fmt.Printf("Username:  %s\n", retrievedUser.Username)
		fmt.Printf("Email:     %s\n", retrievedUser.Email)
		fmt.Printf("Created:   %s\n", retrievedUser.CreatedAt.Format(time.RFC3339))
		fmt.Printf("Last Login:%s\n", retrievedUser.LastLogin.Format(time.RFC3339))
		fmt.Printf("Active:    %t\n", retrievedUser.IsActive)
		fmt.Printf("Roles:     %v\n", retrievedUser.Roles)
		fmt.Printf("Score:     %.1f\n", retrievedUser.Score)
	}
}

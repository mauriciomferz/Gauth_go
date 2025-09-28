package events

import (
	"testing"
	"time"
)

// Test constants
const (
	testIPAddress = "192.168.1.1"
)

func TestEventCreation(t *testing.T) {
	// Test basic event creation
	evt := CreateEvent()
	if evt.ID == "" {
		t.Error("Event ID should be generated automatically")
	}
	if evt.Timestamp.IsZero() {
		t.Error("Event timestamp should be set automatically")
	}
	if evt.Metadata == nil {
		t.Error("Event metadata should be initialized")
	}

	// Test fluent API
	loginEvent := CreateEvent().
		WithType(EventTypeAuth).
		WithActionEnum(ActionLogin).
		WithStatusEnum(StatusSuccess).
		WithSubject("user-123").
		WithMessage("User logged in successfully")
	loginEvent.Metadata.SetString("ip_address", "192.168.1.1")

	if loginEvent.Type != EventTypeAuth {
		t.Errorf("Expected event type %s, got %s", EventTypeAuth, loginEvent.Type)
	}
	if loginEvent.Action != string(ActionLogin) {
		t.Errorf("Expected action %s, got %s", ActionLogin, loginEvent.Action)
	}
	if loginEvent.Status != string(StatusSuccess) {
		t.Errorf("Expected status %s, got %s", StatusSuccess, loginEvent.Status)
	}
	if loginEvent.Subject != "user-123" {
		t.Errorf("Expected subject %s, got %s", "user-123", loginEvent.Subject)
	}
	if loginEvent.Message != "User logged in successfully" {
		t.Errorf("Expected message %s, got %s", "User logged in successfully", loginEvent.Message)
	}
	ipAddress, ok := loginEvent.Metadata.GetString("ip_address")
	if !ok || ipAddress != testIPAddress {
		t.Errorf("Expected metadata ip_address=%s, got %v", testIPAddress, ipAddress)
	}
}

func TestEventFactoryFunctions(t *testing.T) {
	// Test auth event factory
	authEvent := CreateAuthEvent(ActionLogin, StatusSuccess)
	if authEvent.Type != EventTypeAuth {
		t.Errorf("Expected event type %s, got %s", EventTypeAuth, authEvent.Type)
	}
	if authEvent.Action != string(ActionLogin) {
		t.Errorf("Expected action %s, got %s", ActionLogin, authEvent.Action)
	}
	if authEvent.Status != string(StatusSuccess) {
		t.Errorf("Expected status %s, got %s", StatusSuccess, authEvent.Status)
	}

	// Test token event factory
	tokenEvent := CreateTokenEvent(ActionTokenIssued, StatusSuccess)
	if tokenEvent.Type != EventTypeToken {
		t.Errorf("Expected event type %s, got %s", EventTypeToken, tokenEvent.Type)
	}
	if tokenEvent.Action != string(ActionTokenIssued) {
		t.Errorf("Expected action %s, got %s", ActionTokenIssued, tokenEvent.Action)
	}

	// Test backward compatibility with old factory names
	legacyEvent := NewAuthEvent(ActionLogin, StatusSuccess)
	if legacyEvent.Type != EventTypeAuth {
		t.Errorf("Legacy factory expected event type %s, got %s", EventTypeAuth, legacyEvent.Type)
	}
}

// MockHandler is a simple event handler for testing
type MockHandler struct {
	LastEvent  Event
	EventCount int
}

func (m *MockHandler) Handle(event Event) {
	m.LastEvent = event
	m.EventCount++
}

func TestEventPublisher(t *testing.T) {
	// Create a publisher
	publisher := NewEventPublisher()

	// Create handlers
	handler1 := &MockHandler{}
	handler2 := &MockHandler{}

	// Subscribe handlers
	publisher.Subscribe(handler1)
	publisher.Subscribe(handler2)

	// Create and publish an event
	event := CreateAuthEvent(ActionLogin, StatusSuccess)
	publisher.Publish(event)

	// Verify both handlers received the event
	if handler1.EventCount != 1 {
		t.Errorf("Expected handler1 to receive 1 event, got %d", handler1.EventCount)
	}
	if handler2.EventCount != 1 {
		t.Errorf("Expected handler2 to receive 1 event, got %d", handler2.EventCount)
	}

	// Verify event contents match
	if handler1.LastEvent.ID != event.ID {
		t.Errorf("Handler received different event than published")
	}
}

func TestDefaultPublisher(t *testing.T) {
	// Reset the default publisher's handlers for testing
	DefaultPublisher = NewEventPublisher()

	// Create a handler
	handler := &MockHandler{}

	// Subscribe handler to default publisher
	Subscribe(handler)

	// Create and publish an event
	event := CreateAuthEvent(ActionLogin, StatusSuccess)
	Publish(event)

	// Verify handler received the event
	if handler.EventCount != 1 {
		t.Errorf("Expected handler to receive 1 event, got %d", handler.EventCount)
	}

	// Verify event contents match
	if handler.LastEvent.ID != event.ID {
		t.Errorf("Handler received different event than published")
	}
}

func BenchmarkEventCreation(b *testing.B) {
	b.Run("Direct", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = Event{
				ID:        "test-id",
				Type:      EventTypeAuth,
				Action:    string(ActionLogin),
				Status:    string(StatusSuccess),
				Timestamp: time.Now(),
				Metadata:  NewMetadata(),
			}
		}
	})

	b.Run("Factory", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = CreateAuthEvent(ActionLogin, StatusSuccess)
		}
	})

	b.Run("Fluent", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = CreateEvent().
				WithType(EventTypeAuth).
				WithActionEnum(ActionLogin).
				WithStatusEnum(StatusSuccess)
		}
	})
}

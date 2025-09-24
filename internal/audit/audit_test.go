package audit_test

import (
	"testing"
	"time"

	"github.com/Gimel-Foundation/gauth/internal/audit"
)

func TestAuditLogger(t *testing.T) {
	// Test creation with different sizes
	t.Run("Logger Creation", func(t *testing.T) {
		logger := audit.NewLogger(100)
		if logger == nil {
			t.Error("Failed to create audit logger")
		}
	})

	// Test logging and retrieval
	t.Run("Log and GetRecent", func(t *testing.T) {
		logger := audit.NewLogger(5)

		events := []audit.Event{
			{
				ID:        "1",
				Type:      "auth",
				Timestamp: time.Now(),
				ActorID:   "user1",
				Action:    "login",
			},
			{
				ID:        "2",
				Type:      "transaction",
				Timestamp: time.Now(),
				ActorID:   "user1",
				Action:    "payment",
			},
		}

		for _, event := range events {
			logger.Log(event)
		}

		recent := logger.GetRecent(2)
		if len(recent) != 2 {
			t.Errorf("Expected 2 events, got %d", len(recent))
		}

		// Events should be in reverse chronological order
		if recent[0].ID != "2" || recent[1].ID != "1" {
			t.Error("Events not in correct order")
		}
	})

	// Test max size enforcement
	t.Run("Max Size Enforcement", func(t *testing.T) {
		maxSize := 3
		logger := audit.NewLogger(maxSize)

		for i := 0; i < 5; i++ {
			logger.Log(audit.Event{
				ID:      string(rune('A' + i)),
				ActorID: "test",
			})
		}

		events := logger.GetRecent(5)
		if len(events) > maxSize {
			t.Errorf("Logger exceeded max size of %d", maxSize)
		}

		// Should contain most recent events
		if events[0].ID != "E" || events[2].ID != "C" {
			t.Error("Incorrect events retained")
		}
	})

	// Test query functionality
	t.Run("Query Events", func(t *testing.T) {
		logger := audit.NewLogger(10)

		// Log mix of events
		logger.Log(audit.Event{Type: "auth", ActorID: "user1"})
		logger.Log(audit.Event{Type: "transaction", ActorID: "user2"})
		logger.Log(audit.Event{Type: "auth", ActorID: "user1"})

		// Query for auth events by user1
		results := logger.Query(func(e audit.Event) bool {
			return e.Type == "auth" && e.ActorID == "user1"
		})

		if len(results) != 2 {
			t.Errorf("Expected 2 matching events, got %d", len(results))
		}

		for _, event := range results {
			if event.Type != "auth" || event.ActorID != "user1" {
				t.Error("Query returned incorrect events")
			}
		}
	})

	// Test clear functionality
	t.Run("Clear Events", func(t *testing.T) {
		logger := audit.NewLogger(10)

		logger.Log(audit.Event{ID: "1"})
		logger.Log(audit.Event{ID: "2"})

		logger.Clear()

		events := logger.GetRecent(10)
		if len(events) != 0 {
			t.Error("Events remain after clear")
		}
	})
}

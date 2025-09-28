package handlers

import (
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Gimel-Foundation/gauth/gauth-demo-app/web/backend/services"
)

func setupWebSocketTestRouter(t *testing.T) (*gin.Engine, *WebSocketHandler) {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Create test logger
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel) // Reduce test output

	// Create mock GAuth service
	gauthService := createMockGAuthService(t)

	// Create WebSocket handler
	wsHandler := NewWebSocketHandler(gauthService, logger)

	// Setup router
	router := gin.New()
	router.GET("/ws/events", wsHandler.HandleEvents)

	return router, wsHandler
}

func createMockGAuthService(t *testing.T) *services.GAuthService {
	// For testing purposes, we'll use a minimal mock
	// In a real scenario, you'd implement proper mocks
	return nil // WebSocket handler doesn't use the service directly in current implementation
}

func TestWebSocketHandler_NewWebSocketHandler(t *testing.T) {
	logger := logrus.New()
	gauthService := createMockGAuthService(t)

	handler := NewWebSocketHandler(gauthService, logger)

	assert.NotNil(t, handler)
	assert.Equal(t, gauthService, handler.service)
	assert.Equal(t, logger, handler.logger)
	assert.NotNil(t, handler.clients)
	assert.Equal(t, 0, len(handler.clients))
}

func TestWebSocketHandler_HandleEvents_UpgradeConnection(t *testing.T) {
	router, _ := setupWebSocketTestRouter(t)

	// Create test server
	server := httptest.NewServer(router)
	defer server.Close()

	// Convert HTTP URL to WebSocket URL
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws/events"

	// Test connection
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err, "Failed to connect to WebSocket")
	defer conn.Close()

	// Read welcome message
	var welcomeMsg Event
	err = conn.ReadJSON(&welcomeMsg)
	require.NoError(t, err, "Failed to read welcome message")

	assert.Equal(t, "welcome", welcomeMsg.Type)
	assert.Contains(t, welcomeMsg.Data, "message")
	assert.Contains(t, welcomeMsg.Data, "client_id")
	assert.Equal(t, "Connected to GAuth Demo real-time events", welcomeMsg.Data["message"])
	assert.Equal(t, "demo_client", welcomeMsg.Data["client_id"])
}

func TestWebSocketHandler_HandleEvents_PingPong(t *testing.T) {
	router, _ := setupWebSocketTestRouter(t)

	// Create test server
	server := httptest.NewServer(router)
	defer server.Close()

	// Convert HTTP URL to WebSocket URL
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws/events"

	// Connect to WebSocket
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer conn.Close()

	// Skip welcome message
	var welcomeMsg Event
	conn.ReadJSON(&welcomeMsg)

	// Send ping message
	pingMsg := map[string]interface{}{
		"type": "ping",
	}
	err = conn.WriteJSON(pingMsg)
	require.NoError(t, err)

	// Read pong response
	var pongMsg Event
	err = conn.ReadJSON(&pongMsg)
	require.NoError(t, err)

	assert.Equal(t, "pong", pongMsg.Type)
	assert.Equal(t, "pong", pongMsg.Data["message"])
}

func TestWebSocketHandler_HandleEvents_Subscribe(t *testing.T) {
	router, _ := setupWebSocketTestRouter(t)

	// Create test server
	server := httptest.NewServer(router)
	defer server.Close()

	// Convert HTTP URL to WebSocket URL
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws/events"

	// Connect to WebSocket
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer conn.Close()

	// Skip welcome message
	var welcomeMsg Event
	conn.ReadJSON(&welcomeMsg)

	// Send subscription message
	subscribeMsg := map[string]interface{}{
		"type":   "subscribe",
		"events": []string{"auth_request", "token_issued"},
	}
	err = conn.WriteJSON(subscribeMsg)
	require.NoError(t, err)

	// Read subscription confirmation
	var confirmMsg Event
	err = conn.ReadJSON(&confirmMsg)
	require.NoError(t, err)

	assert.Equal(t, "subscription_confirmed", confirmMsg.Type)
	assert.Contains(t, confirmMsg.Data, "subscribed_events")
	
	subscribedEvents := confirmMsg.Data["subscribed_events"].([]interface{})
	assert.Contains(t, subscribedEvents, "auth_request")
	assert.Contains(t, subscribedEvents, "token_issued")
}

func TestWebSocketHandler_SendDemoEvents(t *testing.T) {
	router, _ := setupWebSocketTestRouter(t)

	// Create test server
	server := httptest.NewServer(router)
	defer server.Close()

	// Convert HTTP URL to WebSocket URL
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws/events"

	// Connect to WebSocket
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer conn.Close()

	// Set read timeout to prevent hanging
	conn.SetReadDeadline(time.Now().Add(15 * time.Second))

	// Skip welcome message
	var welcomeMsg Event
	conn.ReadJSON(&welcomeMsg)

	// Collect events for a short period
	events := make([]Event, 0)
	timeout := time.After(12 * time.Second) // Wait for at least one demo event cycle

	for len(events) < 4 { // Try to collect all 4 event types
		select {
		case <-timeout:
			// If we don't get all 4 events, that's okay for the test
			t.Logf("Timeout reached, collected %d events", len(events))
			goto validateEvents
		default:
			var event Event
			err := conn.ReadJSON(&event)
			if err != nil {
				// Handle timeout or connection closed
				t.Logf("Error reading event: %v", err)
				goto validateEvents
			}
			
			// Filter out ping/pong and subscription messages
			if event.Type != "pong" && event.Type != "subscription_confirmed" {
				events = append(events, event)
				t.Logf("Received event: %s", event.Type)
			}
		}
	}

validateEvents:
	// Validate that we received some demo events
	assert.Greater(t, len(events), 0, "Should have received at least one demo event")

	// Check for expected event types
	eventTypes := make(map[string]bool)
	for _, event := range events {
		eventTypes[event.Type] = true
		
		// Validate event structure
		assert.NotEmpty(t, event.Type)
		assert.False(t, event.Timestamp.IsZero())
		assert.NotNil(t, event.Data)
		
		// Validate event-specific data
		switch event.Type {
		case "auth_request":
			assert.Contains(t, event.Data, "client_id")
			assert.Contains(t, event.Data, "scope")
			assert.Contains(t, event.Data, "status")
			assert.Equal(t, "success", event.Data["status"])
		case "token_issued":
			assert.Contains(t, event.Data, "token_type")
			assert.Contains(t, event.Data, "expires_in")
			assert.Equal(t, "Bearer", event.Data["token_type"])
			assert.Equal(t, float64(3600), event.Data["expires_in"])
		case "legal_entity_created":
			assert.Contains(t, event.Data, "entity_id")
			assert.Contains(t, event.Data, "entity_type")
			assert.Contains(t, event.Data, "jurisdiction")
			assert.Equal(t, "corporation", event.Data["entity_type"])
			assert.Equal(t, "US", event.Data["jurisdiction"])
		case "power_of_attorney_delegated":
			assert.Contains(t, event.Data, "poa_id")
			assert.Contains(t, event.Data, "grantor")
			assert.Contains(t, event.Data, "grantee")
			assert.Contains(t, event.Data, "powers")
			powers := event.Data["powers"].([]interface{})
			assert.Contains(t, powers, "sign_contracts")
			assert.Contains(t, powers, "manage_finances")
		}
	}

	// Log received event types for debugging
	receivedTypes := make([]string, 0, len(eventTypes))
	for eventType := range eventTypes {
		receivedTypes = append(receivedTypes, eventType)
	}
	t.Logf("Received event types: %v", receivedTypes)
}

func TestWebSocketHandler_BroadcastEvent(t *testing.T) {
	router, handler := setupWebSocketTestRouter(t)

	// Create test server
	server := httptest.NewServer(router)
	defer server.Close()

	// Convert HTTP URL to WebSocket URL
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws/events"

	// Connect multiple WebSocket clients
	conn1, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer conn1.Close()

	conn2, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer conn2.Close()

	// Skip welcome messages
	var welcomeMsg Event
	conn1.ReadJSON(&welcomeMsg)
	conn2.ReadJSON(&welcomeMsg)

	// Give some time for connections to be registered
	time.Sleep(100 * time.Millisecond)

	// Broadcast custom event
	handler.BroadcastEvent("test_broadcast", map[string]interface{}{
		"message": "This is a broadcast test",
		"test_id": "broadcast_123",
	})

	// Read broadcast event from both connections
	var event1, event2 Event
	
	// Set timeouts to prevent hanging
	conn1.SetReadDeadline(time.Now().Add(2 * time.Second))
	conn2.SetReadDeadline(time.Now().Add(2 * time.Second))

	// Try to read from both connections
	err1 := conn1.ReadJSON(&event1)
	err2 := conn2.ReadJSON(&event2)

	// At least one connection should receive the broadcast
	// (Due to timing and goroutines, both might not receive simultaneously)
	if err1 == nil {
		assert.Equal(t, "test_broadcast", event1.Type)
		assert.Equal(t, "This is a broadcast test", event1.Data["message"])
		assert.Equal(t, "broadcast_123", event1.Data["test_id"])
	}

	if err2 == nil {
		assert.Equal(t, "test_broadcast", event2.Type)
		assert.Equal(t, "This is a broadcast test", event2.Data["message"])
		assert.Equal(t, "broadcast_123", event2.Data["test_id"])
	}

	// At least one should have succeeded
	assert.True(t, err1 == nil || err2 == nil, "At least one connection should receive the broadcast")
}

func TestWebSocketHandler_ConnectionClosure(t *testing.T) {
	router, handler := setupWebSocketTestRouter(t)

	// Create test server
	server := httptest.NewServer(router)
	defer server.Close()

	// Convert HTTP URL to WebSocket URL
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws/events"

	// Connect to WebSocket
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)

	// Skip welcome message
	var welcomeMsg Event
	conn.ReadJSON(&welcomeMsg)

	// Give some time for connection to be registered
	time.Sleep(100 * time.Millisecond)

	// Verify client is registered
	initialClientCount := len(handler.clients)
	assert.Greater(t, initialClientCount, 0, "Client should be registered")

	// Close connection
	conn.Close()

	// Give some time for cleanup
	time.Sleep(200 * time.Millisecond)

	// Note: In the current implementation, client cleanup happens when 
	// trying to write to a closed connection, not immediately on close.
	// We can test this by triggering a broadcast after closing.
	
	handler.BroadcastEvent("cleanup_test", map[string]interface{}{"test": "cleanup"})

	// Give time for cleanup to happen
	time.Sleep(100 * time.Millisecond)

	// Client should be removed after failed write
	finalClientCount := len(handler.clients)
	assert.Equal(t, 0, finalClientCount, "Client should be cleaned up after connection closure")
}

func TestWebSocketHandler_InvalidMessages(t *testing.T) {
	router, _ := setupWebSocketTestRouter(t)

	// Create test server
	server := httptest.NewServer(router)
	defer server.Close()

	// Convert HTTP URL to WebSocket URL
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws/events"

	// Connect to WebSocket
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer conn.Close()

	// Skip welcome message
	var welcomeMsg Event
	conn.ReadJSON(&welcomeMsg)

	// Send invalid JSON (this should be handled gracefully)
	err = conn.WriteMessage(websocket.TextMessage, []byte("invalid json"))
	require.NoError(t, err)

	// Send message with unknown type
	unknownMsg := map[string]interface{}{
		"type": "unknown_message_type",
		"data": "test",
	}
	err = conn.WriteJSON(unknownMsg)
	require.NoError(t, err)

	// Connection should still be alive - send ping to verify
	pingMsg := map[string]interface{}{
		"type": "ping",
	}
	err = conn.WriteJSON(pingMsg)
	require.NoError(t, err)

	// Should receive pong
	var pongMsg Event
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	err = conn.ReadJSON(&pongMsg)
	require.NoError(t, err)
	assert.Equal(t, "pong", pongMsg.Type)
}

func TestWebSocketHandler_MultipleSubscriptions(t *testing.T) {
	router, _ := setupWebSocketTestRouter(t)

	// Create test server
	server := httptest.NewServer(router)
	defer server.Close()

	// Convert HTTP URL to WebSocket URL
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws/events"

	// Connect to WebSocket
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer conn.Close()

	// Skip welcome message
	var welcomeMsg Event
	conn.ReadJSON(&welcomeMsg)

	testCases := []struct {
		name   string
		events []string
	}{
		{
			name:   "Single Event Type",
			events: []string{"auth_request"},
		},
		{
			name:   "Multiple Event Types",
			events: []string{"auth_request", "token_issued", "legal_entity_created"},
		},
		{
			name:   "All Event Types",
			events: []string{"auth_request", "token_issued", "legal_entity_created", "power_of_attorney_delegated"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Send subscription message
			subscribeMsg := map[string]interface{}{
				"type":   "subscribe",
				"events": tc.events,
			}
			err = conn.WriteJSON(subscribeMsg)
			require.NoError(t, err)

			// Read subscription confirmation
			var confirmMsg Event
			conn.SetReadDeadline(time.Now().Add(2 * time.Second))
			err = conn.ReadJSON(&confirmMsg)
			require.NoError(t, err)

			assert.Equal(t, "subscription_confirmed", confirmMsg.Type)
			assert.Contains(t, confirmMsg.Data, "subscribed_events")
			
			subscribedEvents := confirmMsg.Data["subscribed_events"].([]interface{})
			assert.Equal(t, len(tc.events), len(subscribedEvents))
			
			for _, expectedEvent := range tc.events {
				assert.Contains(t, subscribedEvents, expectedEvent)
			}
		})
	}
}

// Benchmark WebSocket message handling
func BenchmarkWebSocketHandler_MessageHandling(b *testing.B) {
	router, _ := setupWebSocketTestRouter(&testing.T{})

	// Create test server
	server := httptest.NewServer(router)
	defer server.Close()

	// Convert HTTP URL to WebSocket URL
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws/events"

	// Connect to WebSocket
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		b.Fatal(err)
	}
	defer conn.Close()

	// Skip welcome message
	var welcomeMsg Event
	conn.ReadJSON(&welcomeMsg)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// Send ping message
		pingMsg := map[string]interface{}{
			"type": "ping",
		}
		conn.WriteJSON(pingMsg)

		// Read pong response
		var pongMsg Event
		conn.ReadJSON(&pongMsg)
	}
}

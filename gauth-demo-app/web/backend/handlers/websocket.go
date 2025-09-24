package handlers

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"

	"github.com/Gimel-Foundation/gauth/gauth-demo-app/web/backend/services"
)

// WebSocketHandler handles WebSocket connections for real-time updates
type WebSocketHandler struct {
	service     *services.GAuthService
	logger      *logrus.Logger
	upgrader    websocket.Upgrader
	clients     map[*websocket.Conn]bool
	clientsMutex sync.RWMutex
	broadcast   chan Event
}

// Event represents a real-time event
type Event struct {
	Type      string                 `json:"type"`
	Timestamp time.Time              `json:"timestamp"`
	Data      map[string]interface{} `json:"data"`
}

// NewWebSocketHandler creates a new WebSocket handler
func NewWebSocketHandler(service *services.GAuthService, logger *logrus.Logger) *WebSocketHandler {
	h := &WebSocketHandler{
		service: service,
		logger:  logger,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow connections from any origin for demo
			},
		},
		clients:   make(map[*websocket.Conn]bool),
		broadcast: make(chan Event, 256),
	}

	// Start the broadcast handler
	go h.handleBroadcast()

	return h
}

// HandleEvents handles WebSocket connections
func (h *WebSocketHandler) HandleEvents(c *gin.Context) {
	conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		h.logger.WithError(err).Error("Failed to upgrade WebSocket connection")
		return
	}
	defer conn.Close()

	// Register client
	h.clientsMutex.Lock()
	h.clients[conn] = true
	h.clientsMutex.Unlock()

	h.logger.Info("WebSocket client connected")

	// Send welcome message
	welcomeEvent := Event{
		Type:      "welcome",
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"message":   "Connected to GAuth Demo real-time events",
			"client_id": "demo_client",
		},
	}

	if err := conn.WriteJSON(welcomeEvent); err != nil {
		h.logger.WithError(err).Error("Failed to send welcome message")
		h.removeClient(conn)
		return
	}

	// Start sending periodic demo events
	go h.sendDemoEvents(conn)

	// Handle incoming messages
	for {
		var message map[string]interface{}
		err := conn.ReadJSON(&message)
		if err != nil {
			h.logger.WithError(err).Debug("WebSocket connection closed")
			h.removeClient(conn)
			break
		}

		// Handle different message types
		if msgType, ok := message["type"].(string); ok {
			switch msgType {
			case "ping":
				pongEvent := Event{
					Type:      "pong",
					Timestamp: time.Now(),
					Data: map[string]interface{}{
						"message": "pong",
					},
				}
				if err := conn.WriteJSON(pongEvent); err != nil {
					h.logger.WithError(err).Error("Failed to send pong")
					h.removeClient(conn)
					return
				}
			case "subscribe":
				// Handle subscription to specific event types
				h.logger.Info("Client subscribed to events")
			default:
				h.logger.WithField("type", msgType).Debug("Received unknown message type")
			}
		}
	}
}

// handleBroadcast handles broadcasting events to all connected clients
func (h *WebSocketHandler) handleBroadcast() {
	for event := range h.broadcast {
		h.clientsMutex.RLock()
		for conn := range h.clients {
			if err := conn.WriteJSON(event); err != nil {
				h.logger.WithError(err).Error("Failed to broadcast event")
				conn.Close()
				delete(h.clients, conn)
			}
		}
		h.clientsMutex.RUnlock()
	}
}

// removeClient removes a client from the clients map
func (h *WebSocketHandler) removeClient(conn *websocket.Conn) {
	h.clientsMutex.Lock()
	defer h.clientsMutex.Unlock()
	if _, ok := h.clients[conn]; ok {
		delete(h.clients, conn)
		h.logger.Info("WebSocket client disconnected")
	}
}

// sendDemoEvents sends periodic demo events to keep the connection active
func (h *WebSocketHandler) sendDemoEvents(conn *websocket.Conn) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	events := []Event{
		{
			Type:      "rfc111_authorization",
			Timestamp: time.Now(),
			Data: map[string]interface{}{
				"ai_agent_id":     "corporate_ai_assistant_v3",
				"business_owner":  "cfo_jane_smith",
				"power_type":      "corporate_financial_authority",
				"jurisdiction":    "US",
				"status":          "authorized",
			},
		},
		{
			Type:      "power_delegation_request",
			Timestamp: time.Now(),
			Data: map[string]interface{}{
				"business_owner": "cfo_jane_smith",
				"power_type":     "financial_transactions",
				"amount_limit":   500000,
				"status":         "pending_approval",
			},
		},
		{
			Type:      "compliance_assessment",
			Timestamp: time.Now(),
			Data: map[string]interface{}{
				"compliance_level": "high",
				"jurisdiction":     "US",
				"assessment_type":  "legal_capacity_verification",
				"status":           "passed",
			},
		},
	}

	eventIndex := 0
	for {
		select {
		case <-ticker.C:
			event := events[eventIndex%len(events)]
			event.Timestamp = time.Now() // Update timestamp

			if err := conn.WriteJSON(event); err != nil {
				h.logger.WithError(err).Error("Failed to send demo event")
				return
			}
			eventIndex++

		case <-time.After(60 * time.Second):
			// Send a heartbeat ping
			pingEvent := Event{
				Type:      "heartbeat",
				Timestamp: time.Now(),
				Data: map[string]interface{}{
					"message": "heartbeat",
				},
			}
			if err := conn.WriteJSON(pingEvent); err != nil {
				h.logger.WithError(err).Error("Failed to send heartbeat")
				return
			}
		}
	}
}

// BroadcastEvent broadcasts an event to all connected clients
func (h *WebSocketHandler) BroadcastEvent(eventType string, data map[string]interface{}) {
	event := Event{
		Type:      eventType,
		Timestamp: time.Now(),
		Data:      data,
	}

	select {
	case h.broadcast <- event:
	default:
		h.logger.Warn("Broadcast channel full, dropping event")
	}
}

// BroadcastMetrics broadcasts system metrics
func (h *WebSocketHandler) BroadcastMetrics(metrics map[string]interface{}) {
	event := Event{
		Type:      "metrics_update",
		Timestamp: time.Now(),
		Data:      metrics,
	}

	select {
	case h.broadcast <- event:
	default:
		h.logger.Warn("Broadcast channel full, dropping metrics")
	}
}

// GetConnectedClients returns the number of connected clients
func (h *WebSocketHandler) GetConnectedClients() int {
	h.clientsMutex.RLock()
	defer h.clientsMutex.RUnlock()
	return len(h.clients)
}
package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"

	"github.com/Gimel-Foundation/gauth/gauth-demo-app/web/backend/services"
)

// WebSocketHandler handles WebSocket connections for real-time updates
type WebSocketHandler struct {
	service  *services.GAuthService
	logger   *logrus.Logger
	upgrader websocket.Upgrader
	clients  map[*websocket.Conn]bool
}

// NewWebSocketHandler creates a new WebSocket handler
func NewWebSocketHandler(service *services.GAuthService, logger *logrus.Logger) *WebSocketHandler {
	return &WebSocketHandler{
		service: service,
		logger:  logger,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				// For demo purposes, allow all origins
				return true
			},
		},
		clients: make(map[*websocket.Conn]bool),
	}
}

// Event represents a real-time event
type Event struct {
	Type      string                 `json:"type"`
	Timestamp time.Time              `json:"timestamp"`
	Data      map[string]interface{} `json:"data"`
}

// HandleEvents handles WebSocket connections for real-time events
func (h *WebSocketHandler) HandleEvents(c *gin.Context) {
	conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		h.logger.WithError(err).Error("Failed to upgrade to WebSocket")
		return
	}
	defer conn.Close()

	// Register client
	h.clients[conn] = true
	h.logger.Info("WebSocket client connected")

	// Send welcome message
	welcomeEvent := Event{
		Type:      "welcome",
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"message": "Connected to GAuth Demo real-time events",
			"client_id": "demo_client",
		},
	}
	
	if err := conn.WriteJSON(welcomeEvent); err != nil {
		h.logger.WithError(err).Error("Failed to send welcome message")
		delete(h.clients, conn)
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
			delete(h.clients, conn)
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
				conn.WriteJSON(pongEvent)
			case "subscribe":
				// Handle subscription to specific event types
				if eventTypes, ok := message["events"].([]interface{}); ok {
					h.logger.WithField("events", eventTypes).Info("Client subscribed to events")
					subscribeEvent := Event{
						Type:      "subscription_confirmed",
						Timestamp: time.Now(),
						Data: map[string]interface{}{
							"subscribed_events": eventTypes,
						},
					}
					conn.WriteJSON(subscribeEvent)
				}
			}
		}
	}
}

// sendDemoEvents sends periodic demo events to the WebSocket client
func (h *WebSocketHandler) sendDemoEvents(conn *websocket.Conn) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	eventCounter := 0
	
	for {
		select {
		case <-ticker.C:
			eventCounter++
			
			// Send different types of demo events
			var event Event
			
			switch eventCounter % 4 {
			case 0:
				event = Event{
					Type:      "auth_request",
					Timestamp: time.Now(),
					Data: map[string]interface{}{
						"client_id":    "demo_client_" + string(rune(eventCounter)),
						"scope":        "read write",
						"redirect_uri": "http://localhost:3000/callback",
						"status":       "success",
					},
				}
			case 1:
				event = Event{
					Type:      "token_issued",
					Timestamp: time.Now(),
					Data: map[string]interface{}{
						"token_type":   "Bearer",
						"expires_in":   3600,
						"scope":        "read write",
						"client_id":    "demo_client",
					},
				}
			case 2:
				event = Event{
					Type:      "legal_entity_created",
					Timestamp: time.Now(),
					Data: map[string]interface{}{
						"entity_id":    "entity_" + string(rune(eventCounter)),
						"entity_name":  "Demo Corporation " + string(rune(eventCounter)),
						"entity_type":  "corporation",
						"jurisdiction": "US",
					},
				}
			case 3:
				event = Event{
					Type:      "power_of_attorney_delegated",
					Timestamp: time.Now(),
					Data: map[string]interface{}{
						"poa_id":    "poa_" + string(rune(eventCounter)),
						"grantor":   "demo_grantor",
						"grantee":   "demo_grantee_" + string(rune(eventCounter)),
						"powers":    []string{"sign_contracts", "manage_finances"},
					},
				}
			}

			if err := conn.WriteJSON(event); err != nil {
				h.logger.WithError(err).Error("Failed to send demo event")
				return
			}
			
			h.logger.WithFields(logrus.Fields{
				"event_type": event.Type,
				"counter":    eventCounter,
			}).Debug("Sent demo event")
		}
	}
}

// BroadcastEvent broadcasts an event to all connected WebSocket clients
func (h *WebSocketHandler) BroadcastEvent(event Event) {
	message, err := json.Marshal(event)
	if err != nil {
		h.logger.WithError(err).Error("Failed to marshal event")
		return
	}

	for client := range h.clients {
		err := client.WriteMessage(websocket.TextMessage, message)
		if err != nil {
			h.logger.WithError(err).Error("Failed to send event to client")
			client.Close()
			delete(h.clients, client)
		}
	}
}
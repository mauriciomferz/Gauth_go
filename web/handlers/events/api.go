package events

// Events handlers - extracted from server_clean.go
// Provides in-memory pub/sub event system with SSE streaming.

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// Event represents a system event.
type Event struct {
	ID   string    `json:"id"`
	At   time.Time `json:"at"`
	Type string    `json:"type"`
	Data any       `json:"data"`
}

// Hub manages in-memory event storage and pub/sub.
type Hub struct {
	mu     sync.RWMutex
	cap    int
	events []*Event
	subs   map[chan *Event]struct{}
}

// NewHub creates a new event hub with specified capacity.
func NewHub(capacity int) *Hub {
	if capacity <= 0 {
		capacity = 200
	}
	return &Hub{cap: capacity, events: make([]*Event, 0, capacity), subs: make(map[chan *Event]struct{})}
}

// Emit adds an event and broadcasts to all subscribers.
func (h *Hub) Emit(e *Event) {
	h.mu.Lock()
	h.events = append(h.events, e)
	if len(h.events) > h.cap {
		h.events = h.events[len(h.events)-h.cap:]
	}
	for ch := range h.subs {
		select {
		case ch <- e:
		default:
		}
	}
	h.mu.Unlock()
}

// List returns the most recent events up to limit.
func (h *Hub) List(limit int) []*Event {
	h.mu.RLock()
	defer h.mu.RUnlock()
	n := len(h.events)
	if limit <= 0 || limit > n {
		limit = n
	}
	out := make([]*Event, limit)
	copy(out, h.events[n-limit:])
	return out
}

// Subscribe creates a new subscription channel.
func (h *Hub) Subscribe() chan *Event {
	ch := make(chan *Event, 32)
	h.mu.Lock()
	h.subs[ch] = struct{}{}
	h.mu.Unlock()
	return ch
}

// Unsubscribe removes a subscription channel.
func (h *Hub) Unsubscribe(ch chan *Event) {
	h.mu.Lock()
	delete(h.subs, ch)
	h.mu.Unlock()
	close(ch)
}

// RandomNonceFunc generates random nonces for event IDs.
type RandomNonceFunc func(n int) string

// HubProvider defines the interface for event hub operations.
type HubProvider interface {
	Emit(e *Event)
	List(limit int) []*Event
	Subscribe() chan *Event
	Unsubscribe(ch chan *Event)
}

// API provides HTTP handlers for events.
type API struct {
	hub         HubProvider
	randomNonce RandomNonceFunc
}

// NewAPI creates a new events API handler.
func NewAPI(hub HubProvider, randomNonce RandomNonceFunc) *API {
	return &API{hub: hub, randomNonce: randomNonce}
}

// RegisterRoutes registers event endpoints on the router.
func (a *API) RegisterRoutes(r *gin.Engine) {
	r.POST("/api/v1/events/emit", a.Emit)
	r.GET("/api/v1/events/stream", a.Stream)
}

// Emit creates a new event.
func (a *API) Emit(c *gin.Context) {
	var req struct {
		Type string `json:"type"`
		Data any    `json:"data"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Type == "" {
		c.JSON(400, gin.H{"success": false, "message": "invalid payload"})
		return
	}
	e := &Event{ID: a.randomNonce(6), At: time.Now(), Type: req.Type, Data: req.Data}
	a.hub.Emit(e)
	c.JSON(201, gin.H{"success": true, "event": e})
}

// Stream provides SSE event streaming.
func (a *API) Stream(c *gin.Context) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")
	ch := a.hub.Subscribe()
	defer a.hub.Unsubscribe(ch)
	// Send snapshot
	snapshot := a.hub.List(20)
	for _, e := range snapshot {
		if b, err := json.Marshal(e); err == nil {
			_, _ = fmt.Fprintf(c.Writer, "event: event\ndata: %s\n\n", b)
		}
	}
	_, _ = fmt.Fprint(c.Writer, "event: open\ndata: {\"ok\":true}\n\n")
	c.Writer.Flush()
	heartbeat := time.NewTicker(10 * time.Second)
	defer heartbeat.Stop()
	for {
		select {
		case <-c.Request.Context().Done():
			return
		case e := <-ch:
			if b, err := json.Marshal(e); err == nil {
				_, _ = fmt.Fprintf(c.Writer, "event: event\ndata: %s\n\n", b)
				c.Writer.Flush()
			}
		case <-heartbeat.C:
			_, _ = fmt.Fprint(c.Writer, ": ping\n\n")
			c.Writer.Flush()
		}
	}
}

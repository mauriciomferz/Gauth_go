// Package events provides an event system for the AgentAuth framework
package events

import (
	"sync"
	"time"
)

// Value represents a typed metadata value
type Value struct {
	Type  ValueType
	Value interface{}
}

// EventType represents the type of event
type EventType string

const (
	EventTypeToken  EventType = "token"
	EventTypeAuth   EventType = "auth"
	EventTypeSystem EventType = "system"
	EventTypeAudit  EventType = "audit"
)

// Metadata provides structured metadata with helper methods
// ValueType and Value struct for typed metadata
type ValueType string

const (
	ValueTypeString  ValueType = "string"
	ValueTypeInt     ValueType = "int"
	ValueTypeFloat   ValueType = "float"
	ValueTypeBool    ValueType = "bool"
	ValueTypeTime    ValueType = "time"
	ValueTypeSlice   ValueType = "slice"
	ValueTypeUnknown ValueType = "unknown"
)

// (keep only the first set of Value methods)

type Metadata struct {
	data map[string]*Value
}

// NewMetadata creates a new metadata instance
func NewMetadata() *Metadata {
	return &Metadata{
		data: make(map[string]*Value),
	}
}

// SetString sets a string value
func (m *Metadata) SetString(key, value string) {
	m.data[key] = &Value{Type: ValueTypeString, Value: value}
}

// SetInt sets an integer value
func (m *Metadata) SetInt(key string, value int) {
	m.data[key] = &Value{Type: ValueTypeInt, Value: value}
}

// SetTime sets a time value
func (m *Metadata) SetTime(key string, value time.Time) {
	m.data[key] = &Value{Type: ValueTypeTime, Value: value}
}

// SetBool sets a boolean value
func (m *Metadata) SetBool(key string, value bool) {
	m.data[key] = &Value{Type: ValueTypeBool, Value: value}
}

// SetFloat sets a float value
func (m *Metadata) SetFloat(key string, value float64) {
	m.data[key] = &Value{Type: ValueTypeFloat, Value: value}
}

// Size returns the number of metadata fields
func (m *Metadata) Size() int {
	return len(m.data)
}

// Keys returns all metadata keys
func (m *Metadata) Keys() []string {
	keys := make([]string, 0, len(m.data))
	for k := range m.data {
		keys = append(keys, k)
	}
	return keys
}

// Get returns the Value and existence
func (m *Metadata) Get(key string) (*Value, bool) {
	val, ok := m.data[key]
	return val, ok
}

// GetString returns a string value if present
func (m *Metadata) GetString(key string) (string, bool) {
	if v, ok := m.Get(key); ok && v.Type == ValueTypeString {
		if s, ok2 := v.Value.(string); ok2 {
			return s, true
		}
	}
	return "", false
}

// GetInt returns an int value if present
func (m *Metadata) GetInt(key string) (int, bool) {
	if v, ok := m.Get(key); ok && v.Type == ValueTypeInt {
		if i, ok2 := v.Value.(int); ok2 {
			return i, true
		}
	}
	return 0, false
}

// GetFloat returns a float64 value if present
func (m *Metadata) GetFloat(key string) (float64, bool) {
	if v, ok := m.Get(key); ok && v.Type == ValueTypeFloat {
		if f, ok2 := v.Value.(float64); ok2 {
			return f, true
		}
	}
	return 0, false
}

// GetBool returns a bool value if present
func (m *Metadata) GetBool(key string) (bool, bool) {
	if v, ok := m.Get(key); ok && v.Type == ValueTypeBool {
		if b, ok2 := v.Value.(bool); ok2 {
			return b, true
		}
	}
	return false, false
}

// GetTime returns a time value if present
func (m *Metadata) GetTime(key string) (time.Time, bool) {
	if v, ok := m.Get(key); ok && v.Type == ValueTypeTime {
		if t, ok2 := v.Value.(time.Time); ok2 {
			return t, true
		}
	}
	return time.Time{}, false
}

// Has checks if a key exists
func (m *Metadata) Has(key string) bool {
	_, ok := m.data[key]
	return ok
}

// (keep only the first correct set of Metadata methods)

// EventPublisher manages event publishing and subscription
type EventPublisher struct {
	handlers []EventHandler
}

// NewEventPublisher creates a new event publisher
func NewEventPublisher() *EventPublisher {
	return &EventPublisher{
		handlers: make([]EventHandler, 0),
	}
}

// Subscribe adds an event handler
func (p *EventPublisher) Subscribe(handler EventHandler) {
	p.handlers = append(p.handlers, handler)
}

// Publish sends an event to all subscribed handlers
func (p *EventPublisher) Publish(event Event) {
	for _, handler := range p.handlers {
		go handler.Handle(event) // Handle asynchronously
	}
}

// EventDispatcher manages event routing

// Event represents a system event
type Event struct {
	ID        string    `json:"id"`
	Type      EventType `json:"type"`
	Action    string    `json:"action"`
	Status    string    `json:"status"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
	Subject   string    `json:"subject"`
	Resource  string    `json:"resource"`
	Metadata  *Metadata `json:"metadata"`
	Error     string    `json:"error,omitempty"`
}

// SetStringSlice sets a slice of strings value
func (m *Metadata) SetStringSlice(key string, value []string) {
	m.data[key] = &Value{Type: ValueTypeSlice, Value: value}
}

// SimpleDispatcher for event handling by type
type SimpleDispatcher struct {
	handlers map[EventType][]EventHandler
}

func NewSimpleDispatcher() *SimpleDispatcher {
	return &SimpleDispatcher{
		handlers: make(map[EventType][]EventHandler),
	}
}

func (d *SimpleDispatcher) RegisterHandler(eventType EventType, handler EventHandler) {
	d.handlers[eventType] = append(d.handlers[eventType], handler)
}

func (d *SimpleDispatcher) Dispatch(event Event) {
	// Dispatch to specific type
	if hs, ok := d.handlers[event.Type]; ok {
		for _, h := range hs {
			h.Handle(event)
		}
	}
	// Dispatch to wildcard
	if hs, ok := d.handlers[EventType("*")]; ok {
		for _, h := range hs {
			h.Handle(event)
		}
	}
}

// EventHandler defines the interface for event handlers
type EventHandler interface {
	Handle(event Event)
}

// EventBus manages event distribution
type EventBus struct {
	mu       sync.RWMutex
	handlers []EventHandler
}

// NewEventBus creates a new event bus
func NewEventBus() *EventBus {
	return &EventBus{
		handlers: make([]EventHandler, 0),
	}
}

// Subscribe adds an event handler to the bus
func (eb *EventBus) Subscribe(handler EventHandler) {
	eb.mu.Lock()
	defer eb.mu.Unlock()
	eb.handlers = append(eb.handlers, handler)
}

// Publish sends an event to all registered handlers
func (eb *EventBus) Publish(event Event) {
	eb.mu.RLock()
	defer eb.mu.RUnlock()

	// Set timestamp if not already set
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	for _, handler := range eb.handlers {
		handler.Handle(event)
	}
}

// (removed duplicate NewEventPublisher definition)

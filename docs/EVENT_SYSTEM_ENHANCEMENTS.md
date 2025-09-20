
# Event System Enhancements

---

## RFC 0111 Compliance & Legal Notice

GAuth implements the GiFo-RfC 0111 (GAuth) standard for AI power-of-attorney, delegation, and auditability. All protocol roles, flows, and exclusions are respected. See https://gimelfoundation.com for the full RFC.

**Exclusions:** GAuth MUST NOT include Web3, DNA-based identity, or decentralized auth logic. See RFC 0111 Section 2.

**Licensing:** Code is subject to the Gimel Foundation's Legal Provisions Relating to GiFo Documents. See LICENSE, Apache 2.0, and referenced licenses for OAuth, OpenID Connect, and MCP.

**P*P Roles:** GAuth implements Power*Point roles (PEP, PDP, PIP, PAP, PVP) as defined in RFC 0111. See the Architecture Guide for details.

---

## Overview

We've significantly improved the GAuth event system to support strongly typed events while maintaining backward compatibility. The new system provides a fluent API for event creation and handling, with better type safety and cleaner code organization.

## Key Changes

### 1. Unified Event Structure

The event system now uses a single, consistent `Event` struct with the following fields:

```go
type Event struct {
	ID        string                 `json:"id"`
	Type      EventType              `json:"type"`
	Action    string                 `json:"action"`
	Status    string                 `json:"status"`
	Timestamp time.Time              `json:"timestamp"`
	Subject   string                 `json:"subject,omitempty"`
	Resource  string                 `json:"resource,omitempty"`
	Message   string                 `json:"message,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	Error     string                 `json:"error,omitempty"`
}
```

### 2. Strongly Typed Actions and Statuses

Events now use strongly typed actions and statuses:

```go
// EventAction represents specific actions within an event type
type EventAction string

// Auth event actions
const (
	ActionLogin         EventAction = "login"
	ActionLogout        EventAction = "logout"
	// ...many more predefined actions
)

// EventStatus represents the status of an event
type EventStatus string

// Event statuses
const (
	StatusSuccess EventStatus = "success"
	StatusFailure EventStatus = "failure"
	// ...other statuses
)
```

### 3. Fluent Builder API

Events can be created using a fluent builder API:

```go
// Create an authentication event with fluent builder pattern
loginEvent := CreateEvent().
    WithType(EventTypeAuth).
    WithActionEnum(ActionLogin).
    WithStatusEnum(StatusSuccess).
    WithSubject("user-123").
    WithMessage("User login successful").
    WithMetadata("ip_address", "192.168.1.1")
```

### 4. Factory Functions

Convenience factory functions for common event types:

```go
// Create an authentication event
loginEvent := CreateAuthEvent(ActionLogin, StatusSuccess)
loginEvent.WithSubject("user-123")

// Create a token event
tokenEvent := CreateTokenEvent(ActionTokenIssued, StatusSuccess)
```

### 5. Event Publisher Pattern

A central publisher mechanism for distributing events to multiple handlers:

```go
// Subscribe a handler to receive events
events.Subscribe(myHandler)

// Publish an event to all handlers
events.Publish(loginEvent)
```

### 6. Type-Safe Event Handling

Event handlers with strongly typed interfaces:

```go
// LoggingHandler logs events to the standard logger
type LoggingHandler struct {
	IncludeTimestamp bool
	IncludeMetadata  bool
	LogLevel         string
}

// Handle implements the EventHandler interface
func (h *LoggingHandler) Handle(event events.Event) {
	// Event handling logic
}
```

## Backward Compatibility

To maintain backward compatibility, we've:

1. Preserved legacy event creation functions (`NewAuthEvent`, etc.)
2. Maintained compatible event structure with JSON tags
3. Created handlers that work with both old and new event formats

## Next Steps

1. Update all event generation sites to use the new typed events
2. Add additional event handlers for various backends (database, messaging, etc.)
3. Create specialized event types for more complex use cases
4. Add validation for events to ensure required fields are populated
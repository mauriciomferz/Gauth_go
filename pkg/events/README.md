# Event System Package

The `events` package provides a type-safe event system for GAuth with structured metadata.

## Overview

The events package replaces the traditional approach of using `map[string]interface{}` with strongly-typed structures, providing better:

- Type safety
- Code readability
- IDE autocompletion
- Error handling

## Key Components

### 1. Event Structure

The core `Event` type includes:

```go
type Event struct {
    ID        string
    Type      EventType
    Action    string
    Status    string
    Timestamp time.Time
    Subject   string
    Resource  string
    Message   string
    Metadata  *Metadata
    Error     string
}
```

### 2. Event Types

Events are categorized by strongly-typed `EventType`:

```go
type EventType string

const (
    EventTypeAuth         EventType = "auth"
    EventTypeAuthz        EventType = "authz"
    EventTypeToken        EventType = "token"
    EventTypeUserActivity EventType = "user_activity"
    EventTypeAudit        EventType = "audit"
    EventTypeSystem       EventType = "system"
)
```

### 3. Typed Metadata

The `Metadata` type provides type-safe access to additional event data:

```go
// Creating metadata
metadata := events.NewMetadata()
metadata.SetString("user_id", "user123")
metadata.SetInt("login_attempts", 3)
metadata.SetTime("last_login", time.Now())
metadata.SetBool("is_admin", false)

// Retrieving typed values
if userID, ok := metadata.GetString("user_id"); ok {
    // Use userID as a string
}

if attempts, ok := metadata.GetInt("login_attempts"); ok {
    // Use attempts as an int
}
```

### 4. Event Handlers & Dispatcher

Register handlers to process specific event types:

```go
// Create a dispatcher
dispatcher := events.NewSimpleDispatcher()

// Register handlers for specific event types
dispatcher.RegisterHandler(events.EventTypeAuth, securityHandler)
dispatcher.RegisterHandler(events.EventTypeAudit, auditHandler)

// Register a handler for all events
dispatcher.RegisterHandler("*", monitoringHandler)

// Dispatch events
dispatcher.Dispatch(myEvent)
```

## Usage Patterns

### Creating Events

```go
event := events.Event{
    ID:        GenerateUUID(),
    Type:      events.EventTypeAuth,
    Action:    "login",
    Status:    "success",
    Timestamp: time.Now(),
    Subject:   userID,
    Resource:  "api",
    Message:   "User successfully authenticated",
    Metadata:  metadata,
}
```

### Using Handlers

Implement the `EventHandler` interface:

```go
type AuditLogHandler struct {
    logger Logger
}

func (h *AuditLogHandler) Handle(event events.Event) {
    h.logger.Log(
        "event_id", event.ID,
        "type", event.Type,
        "action", event.Action,
        "status", event.Status,
        "subject", event.Subject,
        "timestamp", event.Timestamp,
    )
}
```

## Best Practices

1. **Use Type Safety**: Avoid using generic getters/setters when possible. Use the typed methods.
2. **Structured Events**: Define clear event types and actions for consistency.
3. **Useful Metadata**: Include relevant context but avoid excessive data.
4. **Efficient Handlers**: Keep handlers focused and efficient.
5. **Error Handling**: Always handle errors from metadata access.

## Examples

See the cmd/demo application for a demonstration of the event system.
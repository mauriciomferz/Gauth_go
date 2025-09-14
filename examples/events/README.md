# Typed Event System Example

This example demonstrates how to use the GAuth typed event system. The event system provides a way to create, dispatch, and handle events with strongly-typed metadata.

## Features Demonstrated

1. Creating and dispatching events
2. Handling events with custom handlers
3. Working with typed metadata
4. Event filtering by type
5. Error handling in events

## Event Types

The example shows different types of events:

- Authentication events
- Authorization events
- Audit events
- System events

## Typed Metadata

The event system uses a strongly-typed metadata system instead of `map[string]interface{}`, providing:

- Type safety
- Self-documenting code
- Proper error handling
- Better IDE support

## Running the Example

```bash
go run main.go
```

## Expected Output

The example will show different event handlers receiving and processing events with typed metadata.

## Key Components

### Event Structure

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

### Metadata Usage

```go
// Creating metadata
metadata := events.NewMetadata()
metadata.SetString("user_id", "user123")
metadata.SetInt("login_attempts", 1)
metadata.SetTime("last_login", time.Now())
metadata.SetBool("is_admin", false)

// Reading metadata
if userID, ok := metadata.GetString("user_id"); ok {
    fmt.Printf("User ID: %s\n", userID)
}
```

### Event Handlers

```go
// Define a handler
type CustomHandler struct{}

// Implement the EventHandler interface
func (h *CustomHandler) Handle(event events.Event) {
    fmt.Printf("Received event: %s - %s\n", event.Type, event.Message)
}
```

## Migration Note

This example uses the latest GAuth event system with strongly-typed metadata and event handlers. If you are migrating from code that used untyped `map[string]interface{}` for event metadata or legacy handler patterns, see the Migration Guide in `docs/CODE_IMPROVEMENTS.md` for details on updating to the new type-safe event system.

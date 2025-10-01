# GAuth Events System

The GAuth events system provides a type-safe mechanism for generating, handling, and auditing events across the authentication framework. It uses a fluent API design for creating and managing events.

## Event Structure

Each event consists of the following fields:

- `ID`: A unique identifier for the event (UUID)
- `Type`: The type of event (auth, authz, token, etc.)
- `Action`: The specific action (login, logout, token_issued, etc.)
- `Status`: The outcome status (success, failure, error, etc.)
- `Timestamp`: When the event occurred
- `Subject`: The principal associated with the event (user, service, etc.)
- `Resource`: The resource affected by the event (token, session, etc.)
- `Message`: A descriptive message about the event
- `Metadata`: Strongly typed additional context (strings, numbers, booleans, timestamps)
- `Error`: Error message if applicable

## Creating Events

There are two ways to create events:

### 1. Using Factory Functions

```go
// Create an authentication event
loginEvent := events.CreateAuthEvent(events.ActionLogin, events.StatusSuccess)
loginEvent.WithSubject("user-123").WithMessage("User login successful")

// Create a token event
tokenEvent := events.CreateTokenEvent(events.ActionTokenIssued, events.StatusSuccess)
tokenEvent.WithSubject("user-123").WithResource("token-456")
```

### 2. Using Fluent Builder API

```go
// Create an authentication event with fluent builder pattern
loginEvent := events.CreateEvent().
    WithType(events.EventTypeAuth).
    WithActionEnum(events.ActionLogin).
    WithStatusEnum(events.StatusSuccess).
    WithSubject("user-123").
    WithMessage("User login successful").
    WithStringMetadata("ip_address", "192.168.1.1").
    WithIntMetadata("login_count", 5).
    WithBoolMetadata("remember_me", true)
```

## Event Actions

The system provides predefined event actions for common scenarios:

### Authentication Actions
- `ActionLogin`: User login
- `ActionLogout`: User logout
- `ActionLoginFailed`: Failed login attempt
- `ActionPasswordChanged`: Password change
- `ActionPasswordReset`: Password reset
- ...

### Authorization Actions
- `ActionAuthorizationGranted`: Authorization was granted
- `ActionAuthorizationDenied`: Authorization was denied
- `ActionRoleAssigned`: Role was assigned to a user
- ...

### Token Actions
- `ActionTokenIssued`: Token was issued
- `ActionTokenRefreshed`: Token was refreshed
- `ActionTokenRevoked`: Token was revoked
- ...

### Delegation Actions (RFC111)
- `ActionDelegationCreated`: Delegation was created
- `ActionDelegationExercised`: Delegate exercised power of attorney
- ...

## Event Handlers

Events can be processed by implementing the `EventHandler` interface:

```go
type EventHandler interface {
    Handle(Event)
}
```

### Built-in Handlers

- `LoggingHandler`: Logs events with configurable formats
- `MetricsHandler`: Collects metrics from events
- `AuditHandler`: Records audit trails for events

### Publishing Events

Events can be published to all registered handlers:

```go
// Subscribe a handler to receive events
events.Subscribe(myHandler)

// Publish an event to all handlers
events.Publish(loginEvent)
```

## Best Practices

1. **Use Typed Events**: Always use the factory functions or builder methods to ensure type safety
2. **Use Typed Metadata**: Use the typed metadata methods (WithStringMetadata, WithIntMetadata, etc.) for type safety
3. **Include Context**: Add subject, resource, and metadata to provide adequate context
4. **Consistent Status**: Use the predefined status constants for consistent status reporting
5. **Meaningful Messages**: Provide clear, concise messages that explain the event
6. **Error Handling**: When an event represents an error, include the error details

See [TYPED_METADATA.md](TYPED_METADATA.md) for more details on the typed metadata system.

## Example Usage

```go
// Create an event for a user login
loginEvent := events.CreateAuthEvent(events.ActionLogin, events.StatusSuccess).
    WithSubject("user-123").
    WithMessage("User logged in successfully").
    WithStringMetadata("ip_address", "192.168.1.1").
    WithStringMetadata("user_agent", "Mozilla/5.0...").
    WithTimeMetadata("login_time", time.Now())

// Create a logging handler
handler := handlers.NewLoggingHandler()

// Handle the event
handler.Handle(loginEvent)

// Or publish to all registered handlers
events.Publish(loginEvent)
```
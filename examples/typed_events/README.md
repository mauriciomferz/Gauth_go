# Typed Events Example

This example demonstrates how to use GAuth's event system with strongly typed event metadata instead of using `map[string]interface{}`.

## Features

- Defines strongly typed metadata structures for users, authentication, and tokens
- Shows how to publish typed events
- Demonstrates subscribing to and handling typed events
- Provides type-safe access to event data in handlers

## Benefits over `map[string]interface{}`

1. **Type Safety**: Compile-time checking of event data structure
2. **Better IDE Support**: Auto-completion and documentation for event properties
3. **Self-documenting Code**: Clear definition of what data each event contains
4. **Performance**: Reduced need for type assertions and map lookups
5. **Maintainability**: Easier to update and refactor event structures

## Usage

To run the example:

```bash
go run main.go
```

## Implementation Details

This example defines three main types of metadata:

1. `UserMetadata`: Contains information about the user involved in the event
2. `AuthenticationMetadata`: Details about the authentication attempt
3. `TokenMetadata`: Information about any tokens that were generated

These are combined in the `AuthenticationEvent` struct which provides a structured representation of authentication events.

## Integration with Existing Code

To migrate from `map[string]interface{}` to typed structures:

1. Define structs for your common event data patterns
2. Update event publishers to use the typed structures
3. Update event handlers to use type assertions to the specific event types
4. Gradually replace `map[string]interface{}` with typed structures throughout your codebase
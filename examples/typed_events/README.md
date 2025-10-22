# Typed Events Example: Code Review & Beta Notes

> Last Updated: 2025-10-17
> Status: Active

## Overview
This example demonstrates GAuth's event system with strongly typed event metadata. It covers publishing, subscribing, and handling events with type-safe structures for users, authentication, and tokens.

## How to Run
```bash
cd examples/typed_events
go run main.go
```

## Code Review Notes
- Code is idiomatic Go, modular, and demonstrates type-safe event patterns.
- All event metadata is strongly typed for safety and maintainability.
- Inline comments added for Beta readability.

## Educational Comments
- See `main.go` for event publishing and handler logic.
- All major event patterns are demonstrated and tested.

---

For more, see the [Architecture Guide](../../docs/ARCHITECTURE.md).
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

---
Need context? See: README.md | docs/ARCHITECTURE.md | docs/GETTING_STARTED.md

---
title: "Typed Structures Demo"
category: example
status: active
lastUpdated: 2025-11-12
owners: architecture-team
refreshCadence: on-change
---
# Typed Structures Demo

> Last Updated: 2025-10-17
> Status: Active

This example demonstrates AgentAuth's type-safe structures for improved code safety and readability.

## Overview

This demo showcases:

1. **Typed Metadata for Events** - Using strongly-typed metadata instead of untyped maps
2. **Typed Properties for Restrictions** - Using proper types for restriction configuration
3. **TimeRange Utility** - Using the type-safe TimeRange for time-based restrictions
4. **Event Handling with Types** - Processing typed event data

## Running the Demo

```bash
go run main.go
```

This starts a server on `localhost:8080` with several endpoints demonstrating typed structures.

## Key Features

### 1. Typed Event Metadata

Instead of using `map[string]interface{}` for event metadata, the example uses AgentAuth's strongly-typed `Metadata`:

```go
metadata := events.NewMetadata()
metadata.SetString("ip_address", r.RemoteAddr)
metadata.SetInt("attempts", 1)
metadata.SetTime("timestamp", time.Now())
```

This provides compile-time type checking and better code readability.

### 2. Strongly-Typed Restrictions

The demo uses helper functions to create typed restrictions:

```go
businessHours := gauth.CreateTimeRangeRestriction(
    time.Date(2023, 1, 1, 9, 0, 0, 0, time.Local),  // 9 AM
    time.Date(2023, 1, 1, 17, 0, 0, 0, time.Local), // 5 PM
)

rateLimit := gauth.CreateRateLimitRestriction(100, time.Minute)
```

### 3. Type-Safe Event Handling

The event handler demonstrates how to safely access typed metadata:

```go
eventHandler := events.HandlerFunc(func(event events.Event) {
    // Access typed metadata if available
    if ipAddr, ok := event.Metadata.GetString("ip_address"); ok {
        fmt.Printf("  IP: %s\n", ipAddr)
    }
    
    if attempts, ok := event.Metadata.GetInt("attempts"); ok {
        fmt.Printf("  Attempts: %d\n", attempts)
    }
})
```

## API Endpoints

### Create a Token

```bash
curl -X POST http://localhost:8080/token/create
```

Creates a token with typed metadata for IP address, request ID, and user agent.

### Validate a Token

```bash
curl -X POST http://localhost:8080/token/validate -d "token=YOUR_TOKEN_HERE"
```

Validates a token and includes typed metadata for tracking validation attempts.

### Revoke a Token

```bash
curl -X POST http://localhost:8080/token/revoke -d "token=YOUR_TOKEN_HERE&reason=user_logout"
```

Revokes a token with typed metadata including reason and timestamp.

### Access Protected Resource

```bash
curl -H "Authorization: Bearer YOUR_TOKEN_HERE" http://localhost:8080/protected
```

Accesses a protected resource, using typed metadata to track access details.

## Migration Note

This example uses the latest AgentAuth APIs for token management:
- `RequestToken` for token creation
- `ValidateToken` for token validation
- `InvalidateToken` for token revocation

If you are migrating from an older version, see the Migration Guide in `docs/CODE_IMPROVEMENTS.md` for details on updating from legacy token and event APIs to the new type-safe patterns.

## Benefits Demonstrated

1. **Type Safety** - Catch errors at compile time rather than runtime
2. **Code Readability** - Clear indication of data types and expected values
3. **IDE Support** - Better code completion and documentation
4. **Reduced Runtime Errors** - Less chance of type conversion issues
5. **Self-Documenting Code** - Types indicate the expected format and structure
[6. **Idiomatic Go Practices** - The code demonstrates clear separation of concerns, use of interfaces, and proper error handling.]

---

## Beta Notes & Code Review Summary

This example is designed for Beta demonstration clarity and maintainability. Key code review findings:

- **Type Safety:** All event metadata and restriction properties use strong types, reducing runtime errors and improving code readability.
- **Separation of Concerns:** HTTP handlers, event logging, and token management are clearly separated, making the code easy to follow and extend.
- **Idiomatic Go:** The code uses Go best practices, including context usage, error handling, and modular design.
- **Event-Driven Architecture:** Typed events are published and handled via a central event bus, demonstrating scalable event processing.
- **Documentation:** Inline comments and this README provide guidance for each endpoint and code segment.

**Best Practices Demonstrated:**
- Prefer typed structures over untyped maps for metadata and configuration.
- Use helper functions for restriction creation to ensure type safety.
- Document endpoints and code logic for maintainability.
- Handle errors explicitly and return informative responses.

For further learning, review the inline comments in `main.go` and experiment with the endpoints using curl or Postman.

---
Need context? See: README.md | docs/ARCHITECTURE.md | docs/GETTING_STARTED.md

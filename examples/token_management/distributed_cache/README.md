---
title: "Distributed Token Cache Example"
category: example
status: active
lastUpdated: 2025-11-12
owners: architecture-team
refreshCadence: on-change
---
# Distributed Cache Token Management Example

> Last Updated: 2025-10-17
> Status: Active

This example demonstrates token management using a Redis-backed distributed cache in AgentAuth.

## Key Concepts
- **RedisStore**: Implements token storage and retrieval using Redis.
- **Distributed Cache**: Enables scalable, persistent token management across services.

## How to Run
1. Start a Redis server (default: localhost:6379).
2. Run the example:
```bash
go run main.go
```

## Code Review Summary
- Code is idiomatic Go and modular.
- Demonstrates Redis connection, token storage, and retrieval.
- Inline comments added for Beta readability.

## Beta Notes
- See `main.go` for Redis store setup, token marshaling, and distributed cache operations.
- All major distributed cache patterns are demonstrated.

---

For more, see the [Token Package Documentation](../../../pkg/token/doc.go).

---
title: "Tracing Integration Examples"
category: example
status: active
lastUpdated: 2025-11-12
owners: architecture-team
refreshCadence: on-change
---
# Tracing Example: Code Review & Beta Notes

> Last Updated: 2025-10-17
> Status: Active

## Overview
This example demonstrates AgentAuth tracing and observability integration using OpenTelemetry. It covers authentication, authorization, and token lifecycle events with type-safe metadata.

## How to Run
```bash
cd examples/tracing
go run main.go
```

## Code Review Notes
- Code is idiomatic Go, modular, and demonstrates tracing best practices.
- All tracing spans and events are clearly separated and easy to follow.
- Inline comments added for Beta readability.

## Educational Comments
- See `main.go` for tracing integration and span usage.
- All major authentication and authorization events are traced and observable.

---

For more, see the [Architecture Guide](../../docs/ARCHITECTURE.md).
# Tracing Example

This example demonstrates how to use AgentAuth's tracing and observability features with the latest type-safe APIs.

## Features Demonstrated

- Integration of tracing with authentication and authorization flows
- Use of type-safe event metadata
- Observing request and token lifecycle events

## Running the Example

```bash
go run main.go
```

This starts a demo server on `localhost:8080` with tracing enabled.

## Key Concepts

- **Type-Safe Metadata**: All event and trace metadata uses the new strongly-typed structures for safety and clarity.
- **Default Allow Policy**: The example uses a default allow policy for demonstration purposes.
- **Observability**: Tracing spans and events are emitted for key authentication and authorization actions.

## Migration Note

This example uses the latest AgentAuth APIs for tracing and event handling. If you are migrating from older code, see the Migration Guide in `docs/CODE_IMPROVEMENTS.md` for details on updating to the new type-safe patterns.

---
Need context? See: README.md | docs/ARCHITECTURE.md | docs/GETTING_STARTED.md

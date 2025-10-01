# Tracing Example

This example demonstrates how to use GAuth's tracing and observability features with the latest type-safe APIs.

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

This example uses the latest GAuth APIs for tracing and event handling. If you are migrating from older code, see the Migration Guide in `docs/CODE_IMPROVEMENTS.md` for details on updating to the new type-safe patterns.

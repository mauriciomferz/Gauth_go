---
title: "Resilient Service Patterns Examples"
category: example
status: active
lastUpdated: 2025-11-12
owners: architecture-team
refreshCadence: on-change
---
# Resilient Patterns Example

> Last Updated: 2025-10-17
> Status: Active

This example demonstrates key resilience patterns using the GAuth framework:

- **Token Bucket Rate Limiting**
- **Sliding Window Rate Limiting**
- **Circuit Breaker with Retry**
- **Bulkhead Isolation**
- **Combined Patterns**

## How It Works

- Each scenario simulates requests to a backend service (`ExampleService`).
- Rate limiters restrict request frequency.
- Circuit breaker opens after repeated failures, then resets.
- Retry strategy automatically retries failed calls.
- Bulkhead limits concurrent executions.

## Running the Example

```bash
cd examples/resilient
go run main.go
```

## Output

The output shows which requests are allowed, rate limited, retried, or blocked by the circuit breaker. Example:

```
Scenario 1: Token Bucket Rate Limiting
Request 1: ✅ ALLOWED
Request 6: ❌ RATE LIMITED (...)
...
Scenario 3: Circuit Breaker with Retry
Request 3: ❌ service temporarily unavailable (Circuit State: Open)
...
```

## Beta Notes
- All patterns are implemented using GAuth's built-in packages.
- The code is fully tested and idiomatic Go.
- See `main.go` and `patterns.go` for implementation details.

---

For more, see the [GAuth Architecture Guide](../../docs/ARCHITECTURE.md).

---
Need context? See: README.md | docs/ARCHITECTURE.md | docs/GETTING_STARTED.md

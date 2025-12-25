---
title: Readme
category: guide
status: draft
lastUpdated: 2025-12-25
owners: [system]
---

# Comprehensive Resilience Patterns Example

> Last Updated: 2025-10-17
> Status: Active

This example demonstrates a full suite of resilience patterns using the GAuth framework:

- Circuit Breaker
- Retry
- Rate Limiting
- Bulkhead Isolation

## Key Concepts
- **SimulatedService**: Mimics an external service with intermittent failures.
- **Resilience Patterns**: Configured via the `resilience.NewPatterns` API.
- **HTTP Handler**: Exposes a `/resilient` endpoint protected by all patterns.

## How to Run
```bash
cd examples/resilience/comprehensive
go run main.go
```
Then visit [http://localhost:8080/resilient](http://localhost:8080/resilient) in your browser or use curl:
```bash
curl http://localhost:8080/resilient
```

## Code Review Notes
 - Comments added for beta clarity.
## Beta Comments

---

For more, see the [Architecture Guide](../../../docs/ARCHITECTURE.md).

---
Need context? See: README.md | docs/ARCHITECTURE.md | docs/GETTING_STARTED.md

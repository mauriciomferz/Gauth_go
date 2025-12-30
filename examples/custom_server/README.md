---
title: "Custom Resource Server Example"
category: example
status: active
lastUpdated: 2025-11-12
owners: architecture-team
refreshCadence: on-change
---
# Custom Resource Server Example

> Last Updated: 2025-10-17
> Status: Active

This example demonstrates how to extend the AgentAuth ResourceServer with custom transaction processing and metrics.

## Key Concepts
- **CustomResourceServer**: Extends base ResourceServer for custom logic and metrics.
- **Custom Transaction Processing**: Validates and processes transactions with custom rules.
- **Metrics Tracking**: Tracks total transactions and amounts processed.

## How to Run
```bash
cd examples/custom_server
go run main.go
```

## Code Review Summary
 - Beta comments clarify major blocks and usage patterns.

- Extend to add more business rules or metrics as needed.

---

For more, see the [AgentAuth Package Documentation](../../pkg/agentauth/doc.go).

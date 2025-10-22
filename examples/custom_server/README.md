# Custom Resource Server Example

> Last Updated: 2025-10-17
> Status: Active

This example demonstrates how to extend the GAuth ResourceServer with custom transaction processing and metrics.

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

For more, see the [GAuth Package Documentation](../../pkg/gauth/doc.go).

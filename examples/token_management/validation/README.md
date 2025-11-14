---
title: "Token Validation Customization Example"
category: example
status: active
lastUpdated: 2025-11-12
owners: architecture-team
refreshCadence: on-change
---
# Token Validation Example

> Last Updated: 2025-10-17
> Status: Active

This example demonstrates custom and chain-based token validation using GAuth's token package.

## Key Concepts
- **CustomValidator**: Implements issuer-based validation logic.
- **ValidationChain**: Composes multiple validation rules (issuer, clock skew, etc.).
- **Token Store**: In-memory store for tokens.
- **Blacklist**: Demonstrates how to blacklist tokens.
- **Querier**: Shows how to query tokens (API subject to change).

## How to Run
```bash
go run main.go
```

## Code Review Summary
 - Beta comments clarify major blocks and usage patterns.

- See `main.go` for example usage and extension points.

---

For more, see the [Token Package Documentation](../../../pkg/token/doc.go).

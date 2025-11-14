---
title: "PASETO Token Support Example"
category: example
status: active
lastUpdated: 2025-11-12
owners: architecture-team
refreshCadence: on-change
---
# PASETO Token Management Example

> Last Updated: 2025-10-17
> Status: Active

This example demonstrates token management using PASETO (Platform-Agnostic Security Tokens) in GAuth.

## Key Concepts
- **PasetoManager**: Handles PASETO token signing and verification with Ed25519 keys.
- **PasetoClaims**: Encodes token fields for PASETO payloads.

## How to Run
```bash
go run main.go
```

## Code Review Summary
- Code is idiomatic Go and modular.
- Demonstrates PASETO token signing and verification.
- Inline comments added for Beta readability.

## Beta Notes
- See `main.go` for PasetoManager implementation and usage.
- All major PASETO token management patterns are demonstrated.

---

For more, see the [Token Package Documentation](../../../pkg/token/doc.go).

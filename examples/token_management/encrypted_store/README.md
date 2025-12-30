---
title: "Encrypted Token Store Example"
category: example
status: active
lastUpdated: 2025-11-12
owners: architecture-team
refreshCadence: on-change
---
# Encrypted Token Store Example

> Last Updated: 2025-10-17
> Status: Active

This example demonstrates how to wrap a AgentAuth token store with AES-GCM encryption for secure token value storage.

## Key Concepts
- **EncryptedStore**: Wraps a token store, encrypting token values using AES-GCM.
- **Encryption/Decryption**: Token values are encrypted before storage and decrypted on retrieval.

## How to Run
```bash
go run main.go
```

## Code Review Summary
- Code is idiomatic Go and modular.
- Demonstrates secure token value encryption and decryption.
- Inline comments added for Beta readability.

## Beta Notes
- See `main.go` for EncryptedStore implementation and usage.
- Metadata encryption is not implemented for strongly-typed Metadata struct.

---

For more, see the [Token Package Documentation](../../../pkg/token/doc.go).

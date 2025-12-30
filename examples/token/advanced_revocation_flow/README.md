# Advanced Revocation Flow Example

> Last Updated: 2025-10-17
> Status: Active

This example demonstrates an AAP-001-style advanced revocation scenario using the AgentAuth token API. It covers multi-attestation, delegation, selective revocation, and compliance checks.

## Key Concepts
- **Multi-Attestation**: Token issuance with multiple attesters.
- **Delegation**: Tokens delegated to multiple agents.
- **Selective Revocation**: Revoke a single delegate's token, others remain valid.
- **Compliance Check**: Validate token status after revocation.

## How to Run
```bash
cd examples/token/advanced_revocation_flow
go run main.go
```

## Code Review Notes
- Code is idiomatic Go, modular, and demonstrates advanced revocation and delegation patterns.
- All steps are clearly separated and easy to follow.
- Inline comments added for Beta readability.

## Educational Comments
- See `main.go` for the full revocation and delegation flow.
- All major AAP-001 patterns are demonstrated and tested.

---

For more, see the [Architecture Guide](../../../docs/ARCHITECTURE.md).

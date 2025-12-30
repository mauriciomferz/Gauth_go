# AAP-001 Protocol Flow Example

> Last Updated: 2025-10-17
> Status: Active

This example demonstrates a full AAP-001-compliant token protocol flow using the AgentAuth token API. It covers owner proof, grant, attestation, revocation, and compliance checks.

## Key Concepts
- **Owner Proof**: Subject provides proof of control.
- **Grant**: Authorization server issues token.
- **Attestation**: Third party attests to token validity.
- **Revocation & Compliance**: Revoke token and validate status.

## How to Run
```bash
cd examples/token/AAP-001_protocol_flow
go run main.go
```

## Code Review Notes
- Code is idiomatic Go, modular, and demonstrates AAP-001 protocol patterns.
- All steps are clearly separated and easy to follow.
- Inline comments added for Beta readability.

## Educational Comments
- See `main.go` for the full protocol flow.
- All major AAP-001 patterns are demonstrated and tested.

---

For more, see the [Architecture Guide](../../../docs/ARCHITECTURE.md).

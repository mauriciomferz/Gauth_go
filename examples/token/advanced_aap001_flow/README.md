# Advanced AAP-001 Protocol Flow Example

> Last Updated: 2025-10-17
> Status: Active

This example demonstrates an advanced AAP-001 protocol flow using the AgentAuth token API. It covers owner proof, grant, multi-attestation, chained delegation, revocation, and compliance checks.

## Key Concepts
- **Owner Proof**: Initial verification of token owner.
- **Grant**: Token issuance for owner.
- **Multi-Attestation**: Multiple attesters for token validity.
- **Chained Delegation**: Token delegated to another subject.
- **Revocation & Compliance**: Revoke delegated token and validate status.

## How to Run
```bash
cd examples/token/advanced_AAP-001_flow
go run main.go
```

## Code Review Notes
- Code is idiomatic Go, modular, and demonstrates advanced AAP-001 protocol patterns.
- All steps are clearly separated and easy to follow.
- Inline comments added for Beta readability.

## Educational Comments
- See `main.go` for the full protocol flow.
- All major AAP-001 patterns are demonstrated and tested.

---

For more, see the [Architecture Guide](../../../docs/ARCHITECTURE.md).

---
Need context? See: README.md | docs/ARCHITECTURE.md | docs/GETTING_STARTED.md

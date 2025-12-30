# Distributed Token Management Example

> Last Updated: 2025-10-17
> Status: Active

This example demonstrates distributed token management using AgentAuth, simulating a cluster of nodes with event-driven token creation and revocation.

## Key Concepts
- **Cluster Nodes**: Each node manages its own token store and listens for token events.
- **Event Channel**: Shared channel for token events (creation, revocation, rotation).
- **Distributed Revocation**: Revocation events are propagated to all nodes.

## How to Run
```bash
go run main.go
```

## Code Review Summary
- Code is idiomatic Go and modular.
- Demonstrates distributed event handling and token lifecycle management.
- Inline comments added for Beta readability.

## Beta Notes
- See `main.go` for cluster node setup, event handling, and distributed revocation.
- All major distributed token management patterns are demonstrated.

---

For more, see the [Token Package Documentation](../../../pkg/token/doc.go).

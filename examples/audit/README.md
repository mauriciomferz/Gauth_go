# Audit Logging Example

> Last Updated: 2025-10-17
> Status: Active

This example demonstrates audit logging patterns using the GAuth framework. It covers authentication events, token management, event chains, and log searching.

## Key Concepts
- **AuditLogger**: Central logger for audit events.
- **Audit Entries**: Log authentication, token, and resource events.
- **Event Chains**: Link related events by session or transaction.
- **Search & Cleanup**: (Stubbed) Search and cleanup logic for audit logs.

## How to Run
```bash
cd examples/audit
go run main.go
```

## Code Review Notes
 - Comments added for beta clarity.
## Beta Comments

---

For more, see the [Architecture Guide](../../docs/ARCHITECTURE.md).
